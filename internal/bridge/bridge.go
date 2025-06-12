package bridge

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/config"
	"github.com/odata-mcp/go/internal/constants"
	"github.com/odata-mcp/go/internal/mcp"
	"github.com/odata-mcp/go/internal/models"
)

// ODataMCPBridge connects OData services to MCP
type ODataMCPBridge struct {
	config     *config.Config
	client     *client.ODataClient
	server     *mcp.Server
	metadata   *models.ODataMetadata
	tools      map[string]*models.ToolInfo
	mu         sync.RWMutex
	running    bool
	stopChan   chan struct{}
}

// NewODataMCPBridge creates a new bridge instance
func NewODataMCPBridge(cfg *config.Config) (*ODataMCPBridge, error) {
	// Create OData client
	odataClient := client.NewODataClient(cfg.ServiceURL, cfg.Verbose)

	// Configure authentication
	if cfg.HasBasicAuth() {
		odataClient.SetBasicAuth(cfg.Username, cfg.Password)
	} else if cfg.HasCookieAuth() {
		odataClient.SetCookies(cfg.Cookies)
	}

	// Create MCP server
	mcpServer := mcp.NewServer(constants.MCPServerName, constants.MCPServerVersion)

	bridge := &ODataMCPBridge{
		config:   cfg,
		client:   odataClient,
		server:   mcpServer,
		tools:    make(map[string]*models.ToolInfo),
		stopChan: make(chan struct{}),
	}

	// Initialize metadata and tools
	if err := bridge.initialize(); err != nil {
		return nil, fmt.Errorf("failed to initialize bridge: %w", err)
	}

	return bridge, nil
}

// initialize loads metadata and generates tools
func (b *ODataMCPBridge) initialize() error {
	ctx := context.Background()

	// Fetch metadata
	metadata, err := b.client.GetMetadata(ctx)
	if err != nil {
		return fmt.Errorf("failed to fetch metadata: %w", err)
	}

	b.metadata = metadata

	// Generate tools
	if err := b.generateTools(); err != nil {
		return fmt.Errorf("failed to generate tools: %w", err)
	}

	return nil
}

// generateTools creates MCP tools based on metadata
func (b *ODataMCPBridge) generateTools() error {
	// Generate service info tool
	b.generateServiceInfoTool()

	// Generate entity set tools
	for name, entitySet := range b.metadata.EntitySets {
		if b.shouldIncludeEntity(name) {
			b.generateEntitySetTools(name, entitySet)
		}
	}

	// Generate function import tools
	for name, function := range b.metadata.FunctionImports {
		if b.shouldIncludeFunction(name) {
			b.generateFunctionTool(name, function)
		}
	}

	return nil
}

// shouldIncludeEntity checks if an entity should be included based on filters
func (b *ODataMCPBridge) shouldIncludeEntity(entityName string) bool {
	if len(b.config.AllowedEntities) == 0 {
		return true
	}

	for _, pattern := range b.config.AllowedEntities {
		if b.matchesPattern(entityName, pattern) {
			return true
		}
	}

	return false
}

// shouldIncludeFunction checks if a function should be included based on filters
func (b *ODataMCPBridge) shouldIncludeFunction(functionName string) bool {
	if len(b.config.AllowedFunctions) == 0 {
		return true
	}

	for _, pattern := range b.config.AllowedFunctions {
		if b.matchesPattern(functionName, pattern) {
			return true
		}
	}

	return false
}

// matchesPattern checks if a name matches a pattern (supports wildcards)
func (b *ODataMCPBridge) matchesPattern(name, pattern string) bool {
	if pattern == name {
		return true
	}

	// Simple wildcard support
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(name, prefix)
	}

	if strings.HasPrefix(pattern, "*") {
		suffix := strings.TrimPrefix(pattern, "*")
		return strings.HasSuffix(name, suffix)
	}

	return false
}

// generateServiceInfoTool creates a tool to get service information
func (b *ODataMCPBridge) generateServiceInfoTool() {
	toolName := b.formatToolName("odata_service_info", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get information about the OData service including metadata, entity sets, and capabilities",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"include_metadata": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed metadata information",
					"default":     false,
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleServiceInfo(ctx, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpInfo,
	}
}

// generateEntitySetTools creates tools for an entity set
func (b *ODataMCPBridge) generateEntitySetTools(entitySetName string, entitySet *models.EntitySet) {
	// Get entity type
	entityType, exists := b.metadata.EntityTypes[entitySet.EntityType]
	if !exists {
		if b.config.Verbose {
			fmt.Printf("[VERBOSE] Entity type not found for entity set %s: %s\n", entitySetName, entitySet.EntityType)
		}
		return
	}

	// Generate filter/list tool
	b.generateFilterTool(entitySetName, entitySet, entityType)

	// Generate count tool  
	b.generateCountTool(entitySetName, entitySet, entityType)

	// Generate search tool if supported
	if entitySet.Searchable {
		b.generateSearchTool(entitySetName, entitySet, entityType)
	}

	// Generate get tool
	b.generateGetTool(entitySetName, entitySet, entityType)

	// Generate create tool if allowed
	if entitySet.Creatable {
		b.generateCreateTool(entitySetName, entitySet, entityType)
	}

	// Generate update tool if allowed
	if entitySet.Updatable {
		b.generateUpdateTool(entitySetName, entitySet, entityType)
	}

	// Generate delete tool if allowed
	if entitySet.Deletable {
		b.generateDeleteTool(entitySetName, entitySet, entityType)
	}
}

// generateFilterTool creates a filter/list tool for an entity set
func (b *ODataMCPBridge) generateFilterTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpFilter, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("List/filter %s entities with OData query options", entitySetName)

	// Build input schema
	properties := map[string]interface{}{
		"$filter": map[string]interface{}{
			"type":        "string",
			"description": "OData filter expression",
		},
		"$select": map[string]interface{}{
			"type":        "string", 
			"description": "Comma-separated list of properties to select",
		},
		"$expand": map[string]interface{}{
			"type":        "string",
			"description": "Navigation properties to expand",
		},
		"$orderby": map[string]interface{}{
			"type":        "string",
			"description": "Properties to order by",
		},
		"$top": map[string]interface{}{
			"type":        "integer",
			"description": "Maximum number of entities to return",
		},
		"$skip": map[string]interface{}{
			"type":        "integer", 
			"description": "Number of entities to skip",
		},
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": properties,
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityFilter(ctx, entitySetName, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpFilter,
	}
}

// generateCountTool creates a count tool for an entity set
func (b *ODataMCPBridge) generateCountTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpCount, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Get count of %s entities with optional filter", entitySetName)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"$filter": map[string]interface{}{
					"type":        "string",
					"description": "OData filter expression",
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityCount(ctx, entitySetName, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpCount,
	}
}

// generateSearchTool creates a search tool for an entity set
func (b *ODataMCPBridge) generateSearchTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpSearch, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Full-text search %s entities", entitySetName)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"search": map[string]interface{}{
					"type":        "string",
					"description": "Search query string",
				},
				"$select": map[string]interface{}{
					"type":        "string",
					"description": "Comma-separated list of properties to select",
				},
				"$top": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of entities to return",
				},
			},
			"required": []string{"search"},
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntitySearch(ctx, entitySetName, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpSearch,
	}
}

// generateGetTool creates a get tool for an entity set
func (b *ODataMCPBridge) generateGetTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpGet, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Get a single %s entity by key", entitySetName)

	// Build key properties for input schema
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, keyProp := range entityType.KeyProperties {
		for _, prop := range entityType.Properties {
			if prop.Name == keyProp {
				properties[keyProp] = map[string]interface{}{
					"type":        b.getJSONSchemaType(prop.Type),
					"description": fmt.Sprintf("Key property: %s", keyProp),
				}
				required = append(required, keyProp)
				break
			}
		}
	}

	// Add optional query parameters
	properties["$select"] = map[string]interface{}{
		"type":        "string",
		"description": "Comma-separated list of properties to select",
	}
	properties["$expand"] = map[string]interface{}{
		"type":        "string", 
		"description": "Navigation properties to expand",
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityGet(ctx, entitySetName, entityType, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpGet,
	}
}

// generateCreateTool creates a create tool for an entity set
func (b *ODataMCPBridge) generateCreateTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpCreate, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Create a new %s entity", entitySetName)

	// Build properties for input schema based on entity type
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, prop := range entityType.Properties {
		// Skip key properties that are auto-generated
		if prop.IsKey {
			continue
		}

		properties[prop.Name] = map[string]interface{}{
			"type":        b.getJSONSchemaType(prop.Type),
			"description": fmt.Sprintf("Property: %s", prop.Name),
		}

		if !prop.Nullable {
			required = append(required, prop.Name)
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		inputSchema["required"] = required
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: inputSchema,
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityCreate(ctx, entitySetName, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpCreate,
	}
}

// generateUpdateTool creates an update tool for an entity set
func (b *ODataMCPBridge) generateUpdateTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpUpdate, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Update an existing %s entity", entitySetName)

	// Build properties for input schema
	properties := make(map[string]interface{})
	required := make([]string, 0)

	// Add key properties (required)
	for _, keyProp := range entityType.KeyProperties {
		for _, prop := range entityType.Properties {
			if prop.Name == keyProp {
				properties[keyProp] = map[string]interface{}{
					"type":        b.getJSONSchemaType(prop.Type),
					"description": fmt.Sprintf("Key property: %s", keyProp),
				}
				required = append(required, keyProp)
				break
			}
		}
	}

	// Add updatable properties (optional)
	for _, prop := range entityType.Properties {
		if !prop.IsKey {
			properties[prop.Name] = map[string]interface{}{
				"type":        b.getJSONSchemaType(prop.Type),
				"description": fmt.Sprintf("Property: %s", prop.Name),
			}
		}
	}

	// Add method parameter
	properties["_method"] = map[string]interface{}{
		"type":        "string",
		"description": "HTTP method to use (PUT, PATCH, or MERGE)",
		"enum":        []string{"PUT", "PATCH", "MERGE"},
		"default":     "PUT",
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityUpdate(ctx, entitySetName, entityType, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpUpdate,
	}
}

// generateDeleteTool creates a delete tool for an entity set
func (b *ODataMCPBridge) generateDeleteTool(entitySetName string, entitySet *models.EntitySet, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpDelete, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Delete a %s entity", entitySetName)

	// Build key properties for input schema
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, keyProp := range entityType.KeyProperties {
		for _, prop := range entityType.Properties {
			if prop.Name == keyProp {
				properties[keyProp] = map[string]interface{}{
					"type":        b.getJSONSchemaType(prop.Type),
					"description": fmt.Sprintf("Key property: %s", keyProp),
				}
				required = append(required, keyProp)
				break
			}
		}
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]interface{}{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleEntityDelete(ctx, entitySetName, entityType, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpDelete,
	}
}

// generateFunctionTool creates a tool for a function import
func (b *ODataMCPBridge) generateFunctionTool(functionName string, function *models.FunctionImport) {
	toolName := b.formatToolName(functionName, "")

	description := fmt.Sprintf("Call function: %s", functionName)

	// Build properties for input schema based on function parameters
	properties := make(map[string]interface{})
	required := make([]string, 0)

	for _, param := range function.Parameters {
		if param.Mode == "In" || param.Mode == "InOut" {
			properties[param.Name] = map[string]interface{}{
				"type":        b.getJSONSchemaType(param.Type),
				"description": fmt.Sprintf("Parameter: %s", param.Name),
			}

			if !param.Nullable {
				required = append(required, param.Name)
			}
		}
	}

	inputSchema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		inputSchema["required"] = required
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: inputSchema,
	}

	handler := func(ctx context.Context, args map[string]interface{}) (interface{}, error) {
		return b.handleFunctionCall(ctx, functionName, function, args)
	}

	b.server.AddTool(tool, handler)

	// Track tool info
	b.tools[toolName] = &models.ToolInfo{
		Name:        toolName,
		Description: description,
		Function:    functionName,
	}
}

// formatToolName formats a tool name with prefix/postfix
func (b *ODataMCPBridge) formatToolName(operation, entityName string) string {
	var name string

	if entityName != "" {
		if b.config.UsePostfix() {
			name = fmt.Sprintf("%s_%s", operation, entityName)
		} else {
			name = fmt.Sprintf("%s_%s", entityName, operation)
		}
	} else {
		name = operation
	}

	// Apply prefix/postfix
	if b.config.UsePostfix() && b.config.ToolPostfix != "" {
		name = fmt.Sprintf("%s_%s", name, b.config.ToolPostfix)
	} else if !b.config.UsePostfix() && b.config.ToolPrefix != "" {
		name = fmt.Sprintf("%s_%s", b.config.ToolPrefix, name)
	}

	// Apply default postfix if none specified
	if b.config.UsePostfix() && b.config.ToolPostfix == "" {
		serviceID := constants.FormatServiceID(b.config.ServiceURL)
		name = fmt.Sprintf("%s_for_%s", name, serviceID)
	}

	return name
}

// getJSONSchemaType converts OData type to JSON schema type
func (b *ODataMCPBridge) getJSONSchemaType(odataType string) string {
	switch odataType {
	case "Edm.String", "Edm.Guid", "Edm.DateTime", "Edm.DateTimeOffset", "Edm.Time", "Edm.Binary":
		return "string"
	case "Edm.Int16", "Edm.Int32", "Edm.Int64", "Edm.Byte", "Edm.SByte":
		return "integer"
	case "Edm.Single", "Edm.Double", "Edm.Decimal":
		return "number"
	case "Edm.Boolean":
		return "boolean"
	default:
		return "string"
	}
}

// Run starts the MCP bridge
func (b *ODataMCPBridge) Run() error {
	b.mu.Lock()
	if b.running {
		b.mu.Unlock()
		return fmt.Errorf("bridge is already running")
	}
	b.running = true
	b.mu.Unlock()

	// Start MCP server
	return b.server.Run()
}

// Stop stops the MCP bridge
func (b *ODataMCPBridge) Stop() {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.running {
		return
	}

	b.running = false
	close(b.stopChan)
	b.server.Stop()
}

// GetTraceInfo returns comprehensive trace information
func (b *ODataMCPBridge) GetTraceInfo() (*models.TraceInfo, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	authType := "None (anonymous)"
	if b.config.HasBasicAuth() {
		authType = fmt.Sprintf("Basic (user: %s)", b.config.Username)
	} else if b.config.HasCookieAuth() {
		authType = fmt.Sprintf("Cookie (%d cookies)", len(b.config.Cookies))
	}

	toolNaming := "Postfix"
	if !b.config.UsePostfix() {
		toolNaming = "Prefix"
	}

	tools := make([]models.ToolInfo, 0, len(b.tools))
	for _, tool := range b.tools {
		tools = append(tools, *tool)
	}

	return &models.TraceInfo{
		ServiceURL:      b.config.ServiceURL,
		MCPName:         constants.MCPServerName,
		ToolNaming:      toolNaming,
		ToolPrefix:      b.config.ToolPrefix,
		ToolPostfix:     b.config.ToolPostfix,
		ToolShrink:      b.config.ToolShrink,
		SortTools:       b.config.SortTools,
		EntityFilter:    b.config.AllowedEntities,
		FunctionFilter:  b.config.AllowedFunctions,
		Authentication:  authType,
		MetadataSummary: models.MetadataSummary{
			EntityTypes:     len(b.metadata.EntityTypes),
			EntitySets:      len(b.metadata.EntitySets),
			FunctionImports: len(b.metadata.FunctionImports),
		},
		RegisteredTools: tools,
		TotalTools:      len(tools),
	}, nil
}

// Handler implementations would go here...
// These would be the actual implementations that call the OData client
// and return formatted responses. For brevity, I'm showing the signatures:

func (b *ODataMCPBridge) handleServiceInfo(ctx context.Context, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement service info handler
	return "Service info not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityFilter(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity filter handler
	return "Entity filter not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityCount(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity count handler
	return "Entity count not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntitySearch(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity search handler
	return "Entity search not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityGet(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity get handler
	return "Entity get not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityCreate(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity create handler
	return "Entity create not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityUpdate(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity update handler
	return "Entity update not yet implemented", nil
}

func (b *ODataMCPBridge) handleEntityDelete(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement entity delete handler
	return "Entity delete not yet implemented", nil
}

func (b *ODataMCPBridge) handleFunctionCall(ctx context.Context, functionName string, function *models.FunctionImport, args map[string]interface{}) (interface{}, error) {
	// TODO: Implement function call handler
	return "Function call not yet implemented", nil
}