package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/config"
	"github.com/odata-mcp/go/internal/constants"
	"github.com/odata-mcp/go/internal/mcp"
	"github.com/odata-mcp/go/internal/models"
	"github.com/odata-mcp/go/internal/utils"
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
	// 1. Generate service info tool first
	b.generateServiceInfoTool()

	// 2. Generate entity set tools in alphabetical order
	entityNames := make([]string, 0, len(b.metadata.EntitySets))
	for name := range b.metadata.EntitySets {
		if b.shouldIncludeEntity(name) {
			entityNames = append(entityNames, name)
		}
	}
	sort.Strings(entityNames)
	
	for _, name := range entityNames {
		entitySet := b.metadata.EntitySets[name]
		b.generateEntitySetTools(name, entitySet)
	}

	// 3. Generate function import tools in alphabetical order
	functionNames := make([]string, 0, len(b.metadata.FunctionImports))
	for name := range b.metadata.FunctionImports {
		if b.shouldIncludeFunction(name) {
			functionNames = append(functionNames, name)
		}
	}
	sort.Strings(functionNames)
	
	for _, name := range functionNames {
		function := b.metadata.FunctionImports[name]
		b.generateFunctionTool(name, function)
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

	// Build input schema with standard OData parameters
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
	includeMetadata := false
	if val, ok := args["include_metadata"].(bool); ok {
		includeMetadata = val
	}

	info := map[string]interface{}{
		"service_url": b.config.ServiceURL,
		"entity_sets": len(b.metadata.EntitySets),
		"entity_types": len(b.metadata.EntityTypes),
		"function_imports": len(b.metadata.FunctionImports),
		"schema_namespace": b.metadata.SchemaNamespace,
		"container_name": b.metadata.ContainerName,
		"version": b.metadata.Version,
		"parsed_at": b.metadata.ParsedAt.Format("2006-01-02T15:04:05Z"),
	}

	if includeMetadata {
		info["entity_sets_detail"] = b.metadata.EntitySets
		info["entity_types_detail"] = b.metadata.EntityTypes
		info["function_imports_detail"] = b.metadata.FunctionImports
	}

	response, err := json.Marshal(info)
	if err != nil {
		return "Error formatting service info", err
	}

	return string(response), nil
}

func (b *ODataMCPBridge) handleEntityFilter(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// Build query options from arguments using standard OData parameters
	options := make(map[string]string)
	
	// Handle each OData parameter
	if filter, ok := args["$filter"].(string); ok && filter != "" {
		options[constants.QueryFilter] = filter
	}
	if selectParam, ok := args["$select"].(string); ok && selectParam != "" {
		options[constants.QuerySelect] = selectParam
	}
	if expand, ok := args["$expand"].(string); ok && expand != "" {
		options[constants.QueryExpand] = expand
	}
	if orderby, ok := args["$orderby"].(string); ok && orderby != "" {
		options[constants.QueryOrderBy] = orderby
	}
	if top, ok := args["$top"].(float64); ok {
		options[constants.QueryTop] = fmt.Sprintf("%d", int(top))
	}
	if skip, ok := args["$skip"].(float64); ok {
		options[constants.QuerySkip] = fmt.Sprintf("%d", int(skip))
	}
	
	// Call OData client to get entity set
	response, err := b.client.GetEntitySet(ctx, entitySetName, options)
	if err != nil {
		if b.config.VerboseErrors {
			return nil, fmt.Errorf("failed to filter entities from %s with options %v: %w", entitySetName, options, err)
		}
		return nil, fmt.Errorf("failed to filter entities: %w", err)
	}
	
	// Enhance response based on configuration
	enhancedResponse := b.enhanceResponse(response, options)
	
	// Format response as JSON string
	result, err := json.Marshal(enhancedResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}

// enhanceResponse enhances OData response based on configuration options
func (b *ODataMCPBridge) enhanceResponse(response *models.ODataResponse, options map[string]string) *models.ODataResponse {
	enhanced := &models.ODataResponse{
		Context:  response.Context,
		Count:    response.Count,
		NextLink: response.NextLink,
		Value:    response.Value,
		Error:    response.Error,
		Metadata: response.Metadata,
	}
	
	// Apply size limits first to prevent large responses
	enhanced = b.applySizeLimits(enhanced)
	
	// Add pagination hints if enabled
	if b.config.PaginationHints && response.Value != nil {
		pagination := &models.PaginationInfo{}
		
		// Set total count if available
		if response.Count != nil {
			pagination.TotalCount = response.Count
		}
		
		// Calculate current count
		if resultArray, ok := response.Value.([]interface{}); ok {
			pagination.CurrentCount = len(resultArray)
		} else {
			pagination.CurrentCount = 1 // Single entity
		}
		
		// Parse skip and top from options
		skip := 0
		top := 0
		if skipStr, exists := options[constants.QuerySkip]; exists {
			fmt.Sscanf(skipStr, "%d", &skip)
		}
		if topStr, exists := options[constants.QueryTop]; exists {
			fmt.Sscanf(topStr, "%d", &top)
		}
		
		pagination.Skip = skip
		pagination.Top = top
		
		// Determine if there are more results
		if pagination.TotalCount != nil && top > 0 {
			pagination.HasMore = int64(skip+pagination.CurrentCount) < *pagination.TotalCount
			
			// Generate suggested next call if there are more results
			if pagination.HasMore {
				nextSkip := skip + pagination.CurrentCount
				suggestedCall := fmt.Sprintf("Use $skip=%d and $top=%d for next page", nextSkip, top)
				pagination.SuggestedNextCall = &suggestedCall
			}
		}
		
		enhanced.Pagination = pagination
	}
	
	// Process legacy dates if enabled
	if b.config.LegacyDates {
		enhanced.Value = b.convertLegacyDates(enhanced.Value)
	}
	
	// Strip metadata if not requested
	if !b.config.ResponseMetadata {
		enhanced.Value = b.stripMetadata(enhanced.Value)
	}
	
	return enhanced
}

// applySizeLimits enforces response size and item count limits
func (b *ODataMCPBridge) applySizeLimits(response *models.ODataResponse) *models.ODataResponse {
	if response.Value == nil {
		return response
	}
	
	// Apply item count limit
	if b.config.MaxItems > 0 {
		if resultArray, ok := response.Value.([]interface{}); ok {
			if len(resultArray) > b.config.MaxItems {
				// Truncate to max items and add warning
				truncated := resultArray[:b.config.MaxItems]
				
				// Update response
				newResponse := &models.ODataResponse{
					Context:  response.Context,
					Count:    response.Count,
					NextLink: response.NextLink,
					Value:    truncated,
					Error:    response.Error,
					Metadata: response.Metadata,
				}
				
				// Add truncation warning
				if newResponse.Metadata == nil {
					newResponse.Metadata = make(map[string]interface{})
				}
				newResponse.Metadata["truncated"] = true
				newResponse.Metadata["original_count"] = len(resultArray)
				newResponse.Metadata["max_items"] = b.config.MaxItems
				newResponse.Metadata["warning"] = fmt.Sprintf("Response truncated from %d to %d items due to size limits", len(resultArray), b.config.MaxItems)
				
				return newResponse
			}
		}
	}
	
	// Apply response size limit
	if b.config.MaxResponseSize > 0 {
		// Estimate response size by marshaling to JSON
		jsonData, err := json.Marshal(response.Value)
		if err == nil && len(jsonData) > b.config.MaxResponseSize {
			// If it's an array, try to reduce items
                       if resultArray, ok := response.Value.([]interface{}); ok {
                               if len(resultArray) == 0 {
                                       return response
                               }

                               // Calculate how many items we can fit
                               avgItemSize := len(jsonData) / len(resultArray)
                               if avgItemSize == 0 {
                                       return response
                               }
                               maxItems := b.config.MaxResponseSize / avgItemSize
                               if maxItems < 1 {
                                       maxItems = 1
                               }
				
				// Truncate to fit size limit
				truncated := resultArray[:maxItems]
				
				// Update response
				newResponse := &models.ODataResponse{
					Context:  response.Context,
					Count:    response.Count,
					NextLink: response.NextLink,
					Value:    truncated,
					Error:    response.Error,
					Metadata: response.Metadata,
				}
				
				// Add truncation warning
				if newResponse.Metadata == nil {
					newResponse.Metadata = make(map[string]interface{})
				}
				newResponse.Metadata["truncated"] = true
				newResponse.Metadata["original_count"] = len(resultArray)
				newResponse.Metadata["truncated_count"] = len(truncated)
				newResponse.Metadata["max_response_size"] = b.config.MaxResponseSize
				newResponse.Metadata["warning"] = fmt.Sprintf("Response truncated from %d to %d items due to response size limit (%d bytes)", len(resultArray), len(truncated), b.config.MaxResponseSize)
				
				return newResponse
			}
		}
	}
	
	return response
}

// convertLegacyDates converts date fields to epoch timestamp format (/Date(1234567890000)/)
func (b *ODataMCPBridge) convertLegacyDates(data interface{}) interface{} {
	if !b.config.LegacyDates {
		return data
	}
	
	// Convert from OData legacy format to ISO for display
	return utils.ConvertDatesInResponse(data, true)
}

// stripMetadata removes __metadata blocks from entities unless specifically requested
func (b *ODataMCPBridge) stripMetadata(data interface{}) interface{} {
	switch v := data.(type) {
	case []interface{}:
		// Handle array of entities
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = b.stripMetadata(item)
		}
		return result
	case map[string]interface{}:
		// Handle single entity
		result := make(map[string]interface{})
		for key, value := range v {
			if key != "__metadata" {
				result[key] = b.stripMetadata(value)
			}
		}
		return result
	default:
		return data
	}
}

func (b *ODataMCPBridge) handleEntityCount(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// Build query options - for count we typically only need filter
	options := make(map[string]string)
	
	if filter, ok := args["$filter"].(string); ok && filter != "" {
		options[constants.QueryFilter] = filter
	}
	
	// Add $inlinecount=allpages to get inline count (OData v2 syntax)
	options[constants.QueryInlineCount] = "allpages"
	options[constants.QueryTop] = "0" // We only want the count, not the data
	
	// Call OData client to get count
	response, err := b.client.GetEntitySet(ctx, entitySetName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity count: %w", err)
	}
	
	// Extract count from response
	count := int64(0)
	if response.Count != nil {
		count = *response.Count
	}
	
	// Return count as formatted string
	return fmt.Sprintf(`{"count": %d}`, count), nil
}

func (b *ODataMCPBridge) handleEntitySearch(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// Get search term
	searchTerm, ok := args["search"].(string)
	if !ok {
		searchTerm, ok = args["search_term"].(string)
		if !ok {
			return nil, fmt.Errorf("missing required parameter: search_term")
		}
	}
	
	// Build query options
	options := make(map[string]string)
	options[constants.QuerySearch] = searchTerm
	
	// Handle optional parameters
	if top, ok := args["$top"].(float64); ok {
		options[constants.QueryTop] = fmt.Sprintf("%d", int(top))
	}
	if skip, ok := args["$skip"].(float64); ok {
		options[constants.QuerySkip] = fmt.Sprintf("%d", int(skip))
	}
	
	// Call OData client to search entities
	response, err := b.client.GetEntitySet(ctx, entitySetName, options)
	if err != nil {
		return nil, fmt.Errorf("failed to search entities: %w", err)
	}
	
	// Format response as JSON string
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}

func (b *ODataMCPBridge) handleEntityGet(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// Build key values from arguments
	key := make(map[string]interface{})
	for _, keyProp := range entityType.KeyProperties {
		if value, exists := args[keyProp]; exists {
			key[keyProp] = value
		} else {
			return nil, fmt.Errorf("missing required key property: %s", keyProp)
		}
	}
	
	// Build query options for expand/select
	options := make(map[string]string)
	if selectParam, ok := args["$select"].(string); ok && selectParam != "" {
		options[constants.QuerySelect] = selectParam
	}
	if expand, ok := args["$expand"].(string); ok && expand != "" {
		options[constants.QueryExpand] = expand
	}
	
	// Call OData client to get entity
	response, err := b.client.GetEntity(ctx, entitySetName, key, options)
	if err != nil {
		return nil, fmt.Errorf("failed to get entity: %w", err)
	}
	
	// Format response as JSON string
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}

func (b *ODataMCPBridge) handleEntityCreate(ctx context.Context, entitySetName string, args map[string]interface{}) (interface{}, error) {
	// All arguments are the entity data (excluding system parameters)
	entityData := make(map[string]interface{})
	for k, v := range args {
		// Skip any system parameters (starting with $)
		if !strings.HasPrefix(k, "$") {
			entityData[k] = v
		}
	}
	
	// Convert numeric fields to strings for Edm.Decimal compatibility
	// This prevents "Failed to read property 'Quantity' at offset" errors
	entityData = utils.ConvertEntityDataForOData(entityData, nil)
	
	// Convert date fields to OData legacy format if needed
	if b.config.LegacyDates {
		entityData = utils.ConvertDatesInMap(entityData, false) // false = convert ISO to legacy
	}
	
	// Call OData client to create entity
	response, err := b.client.CreateEntity(ctx, entitySetName, entityData)
	if err != nil {
		return nil, fmt.Errorf("failed to create entity: %w", err)
	}
	
	// Enhance response (includes date conversion if enabled)
	response = b.enhanceResponse(response, make(map[string]string))
	
	// Format response as JSON string
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}

func (b *ODataMCPBridge) handleEntityUpdate(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// Extract key values and method
	key := make(map[string]interface{})
	updateData := make(map[string]interface{})
	method := constants.PUT // default method
	
	for k, v := range args {
		if k == "_method" {
			if m, ok := v.(string); ok {
				method = m
			}
			continue
		}
		
		// Check if this is a key property
		isKey := false
		for _, keyProp := range entityType.KeyProperties {
			if k == keyProp {
				key[k] = v
				isKey = true
				break
			}
		}
		
		// If not a key, it's update data
		if !isKey && !strings.HasPrefix(k, "$") {
			updateData[k] = v
		}
	}
	
	// Verify we have all required key properties
	for _, keyProp := range entityType.KeyProperties {
		if _, exists := key[keyProp]; !exists {
			return nil, fmt.Errorf("missing required key property: %s", keyProp)
		}
	}
	
	// Convert numeric fields to strings for Edm.Decimal compatibility
	// This prevents "Failed to read property 'Quantity' at offset" errors
	updateData = utils.ConvertEntityDataForOData(updateData, nil)
	
	// Convert date fields to OData legacy format if needed
	if b.config.LegacyDates {
		updateData = utils.ConvertDatesInMap(updateData, false) // false = convert ISO to legacy
	}
	
	// Call OData client to update entity
	response, err := b.client.UpdateEntity(ctx, entitySetName, key, updateData, method)
	if err != nil {
		return nil, fmt.Errorf("failed to update entity: %w", err)
	}
	
	// Enhance response (includes date conversion if enabled)
	response = b.enhanceResponse(response, make(map[string]string))
	
	// Format response as JSON string
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}

func (b *ODataMCPBridge) handleEntityDelete(ctx context.Context, entitySetName string, entityType *models.EntityType, args map[string]interface{}) (interface{}, error) {
	// Build key values from arguments
	key := make(map[string]interface{})
	for _, keyProp := range entityType.KeyProperties {
		if value, exists := args[keyProp]; exists {
			key[keyProp] = value
		} else {
			return nil, fmt.Errorf("missing required key property: %s", keyProp)
		}
	}
	
	// Call OData client to delete entity
	_, err := b.client.DeleteEntity(ctx, entitySetName, key)
	if err != nil {
		return nil, fmt.Errorf("failed to delete entity: %w", err)
	}
	
	// For successful deletes, return a simple success message
	return `{"status": "success", "message": "Entity deleted successfully"}`, nil
}

func (b *ODataMCPBridge) handleFunctionCall(ctx context.Context, functionName string, function *models.FunctionImport, args map[string]interface{}) (interface{}, error) {
	// Build parameters from arguments
	parameters := make(map[string]interface{})
	for _, param := range function.Parameters {
		if param.Mode == "In" || param.Mode == "InOut" {
			if value, exists := args[param.Name]; exists {
				parameters[param.Name] = value
			} else if !param.Nullable {
				return nil, fmt.Errorf("missing required parameter: %s", param.Name)
			}
		}
	}
	
	// Determine HTTP method (default to GET if not specified)
	method := function.HTTPMethod
	if method == "" {
		method = constants.GET
	}
	
	// Call OData client to execute function
	response, err := b.client.CallFunction(ctx, functionName, parameters, method)
	if err != nil {
		return nil, fmt.Errorf("failed to call function: %w", err)
	}
	
	// Format response as JSON string
	result, err := json.Marshal(response)
	if err != nil {
		return nil, fmt.Errorf("failed to format response: %w", err)
	}
	
	return string(result), nil
}