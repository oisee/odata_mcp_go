// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"

	"github.com/zmcp/odata-mcp/internal/client"
	"github.com/zmcp/odata-mcp/internal/config"
	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/hint"
	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
	"github.com/zmcp/odata-mcp/internal/transport"
)

// ODataMCPBridge connects OData services to MCP
type ODataMCPBridge struct {
	config      *config.Config
	client      *client.ODataClient
	server      *mcp.Server
	metadata    *models.ODataMetadata
	tools       map[string]*models.ToolInfo
	hintManager *hint.Manager
	mu          sync.RWMutex
	running     bool
	stopChan    chan struct{}
}

// NewODataMCPBridge creates a new bridge instance
func NewODataMCPBridge(cfg *config.Config) (*ODataMCPBridge, error) {
	// Create OData client
	odataClient := client.NewODataClient(cfg.ServiceURL, cfg.Verbose)

	// Configure authentication
	if cfg.HasDigestAuth() {
		odataClient.SetDigestAuth(cfg.Username, cfg.Password)
	} else if cfg.HasBasicAuth() {
		odataClient.SetBasicAuth(cfg.Username, cfg.Password)
	} else if cfg.HasCookieAuth() {
		odataClient.SetCookies(cfg.Cookies)
	}

	// Create MCP server
	mcpServer := mcp.NewServer(constants.MCPServerName, constants.MCPServerVersion)

	// Set protocol version if specified
	if cfg.ProtocolVersion != "" {
		mcpServer.SetProtocolVersion(cfg.ProtocolVersion)
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using MCP protocol version: %s\n", cfg.ProtocolVersion)
		}
	}

	// Create hint manager
	hintMgr := hint.NewManager()

	// Load hints from file if specified or default location
	if err := hintMgr.LoadFromFile(cfg.HintsFile); err != nil {
		if cfg.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to load hints file: %v\n", err)
		}
	}

	// Set CLI hint if provided
	if cfg.Hint != "" {
		if err := hintMgr.SetCLIHint(cfg.Hint); err != nil {
			if cfg.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to parse CLI hint: %v\n", err)
			}
		}
	}

	bridge := &ODataMCPBridge{
		config:      cfg,
		client:      odataClient,
		server:      mcpServer,
		tools:       make(map[string]*models.ToolInfo),
		hintManager: hintMgr,
		stopChan:    make(chan struct{}),
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
	// Check if universal tool mode is enabled
	if b.config.UniversalTool {
		if b.config.Verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Using universal tool mode (single tool instead of per-entity tools)\n")
		}
		b.generateUniversalTool()
		return nil
	}

	// Standard multi-tool mode
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

		// Check if actions are enabled
		if !b.config.IsOperationEnabled('A') {
			if b.config.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Skipping function %s - actions are disabled\n", name)
			}
			continue
		}

		// Skip modifying functions in read-only mode unless functions are allowed
		if b.config.ReadOnly || (!b.config.AllowModifyingFunctions() && b.isFunctionModifying(function)) {
			if b.config.Verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Skipping function %s in read-only mode (HTTP method: %s)\n", name, function.HTTPMethod)
			}
			continue
		}
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
	if prefix, found := strings.CutSuffix(pattern, "*"); found {
		return strings.HasPrefix(name, prefix)
	}

	if suffix, found := strings.CutPrefix(pattern, "*"); found {
		return strings.HasSuffix(name, suffix)
	}

	return false
}

// isFunctionModifying determines if a function import performs modifying operations
func (b *ODataMCPBridge) isFunctionModifying(function *models.FunctionImport) bool {
	// Check HTTP method - POST is typically used for modifying operations
	// GET is typically read-only
	httpMethod := strings.ToUpper(function.HTTPMethod)
	if httpMethod == "GET" {
		return false
	}

	// For v4, actions are typically modifying, functions are typically read-only
	if function.IsAction {
		return true
	}

	// If HTTP method is POST, PUT, PATCH, DELETE, or MERGE, it's modifying
	return httpMethod == "POST" || httpMethod == "PUT" ||
		httpMethod == "PATCH" || httpMethod == "DELETE" ||
		httpMethod == "MERGE"
}

// getParameterName returns the parameter name based on ClaudeCodeFriendly setting
func (b *ODataMCPBridge) getParameterName(odataParam string) string {
	if b.config.ClaudeCodeFriendly {
		if stripped, found := strings.CutPrefix(odataParam, "$"); found {
			return stripped
		}
	}
	return odataParam
}

// mapParameterToOData maps a parameter name back to its OData equivalent
func (b *ODataMCPBridge) mapParameterToOData(param string) string {
	if b.config.ClaudeCodeFriendly {
		// Map friendly names back to OData names
		switch param {
		case "filter":
			return "$filter"
		case "select":
			return "$select"
		case "expand":
			return "$expand"
		case "orderby":
			return "$orderby"
		case "top":
			return "$top"
		case "skip":
			return "$skip"
		case "count":
			return "$count"
		case "search":
			return "$search"
		case "format":
			return "$format"
		default:
			// If it doesn't match known OData params, return as-is
			return param
		}
	}
	// If not in Claude-friendly mode, check if we need to add $ prefix
	if !strings.HasPrefix(param, "$") && !strings.HasPrefix(param, "_") {
		// Check if this is a known OData parameter without prefix
		switch param {
		case "filter", "select", "expand", "orderby", "top", "skip", "count", "search", "format":
			return "$" + param
		}
	}
	return param
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

// lookupEntityType finds an entity type by name with fallback.
// It tries the qualified name first (e.g., "Namespace.TypeName"), then falls back
// to the short name (e.g., "TypeName") for simpler metadata formats.
func (b *ODataMCPBridge) lookupEntityType(typeName string) *models.EntityType {
	// Try direct lookup (works for qualified names or single-schema services)
	if entityType, exists := b.metadata.EntityTypes[typeName]; exists {
		return entityType
	}

	// Fallback: extract short name and try again
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		shortName := parts[len(parts)-1]
		if entityType, exists := b.metadata.EntityTypes[shortName]; exists {
			return entityType
		}
	}

	return nil
}

// buildKeySchema builds JSON schema properties for entity key properties.
// This is a helper to avoid repeating key property lookup logic.
func (b *ODataMCPBridge) buildKeySchema(entityType *models.EntityType) (properties map[string]any, required []string) {
	properties = make(map[string]any)
	required = make([]string, 0, len(entityType.KeyProperties))

	for _, keyProp := range entityType.KeyProperties {
		for _, prop := range entityType.Properties {
			if prop.Name == keyProp {
				properties[keyProp] = map[string]any{
					"type":        b.getJSONSchemaType(prop.Type),
					"description": fmt.Sprintf("Key property: %s", keyProp),
				}
				required = append(required, keyProp)
				break
			}
		}
	}

	return properties, required
}

// registerTool registers a tool with the MCP server and tracks it.
// This is a helper to avoid repeating tool registration logic.
func (b *ODataMCPBridge) registerTool(tool *mcp.Tool, handler func(context.Context, map[string]any) (any, error), info *models.ToolInfo) {
	b.server.AddTool(tool, handler)
	b.tools[tool.Name] = info
}

// GetServer returns the MCP server instance
func (b *ODataMCPBridge) GetServer() *mcp.Server {
	return b.server
}

// SetTransport sets the transport for the MCP server
func (b *ODataMCPBridge) SetTransport(t any) {
	b.server.SetTransport(t)
}

// HandleMessage delegates message handling to the MCP server
func (b *ODataMCPBridge) HandleMessage(ctx context.Context, msg any) (any, error) {
	// Convert any to *transport.Message
	if transportMsg, ok := msg.(*transport.Message); ok {
		return b.server.HandleMessage(ctx, transportMsg)
	}
	return nil, fmt.Errorf("invalid message type")
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
	if b.config.HasDigestAuth() {
		authType = fmt.Sprintf("Digest (user: %s)", b.config.Username)
	} else if b.config.HasBasicAuth() {
		authType = fmt.Sprintf("Basic (user: %s)", b.config.Username)
	} else if b.config.HasCookieAuth() {
		authType = fmt.Sprintf("Cookie (%d cookies)", len(b.config.Cookies))
	}

	toolNaming := "Postfix"
	if !b.config.UsePostfix() {
		toolNaming = "Prefix"
	}

	readOnlyMode := ""
	if b.config.ReadOnly {
		readOnlyMode = "Full read-only (no modifying operations)"
	} else if b.config.ReadOnlyButFunctions {
		readOnlyMode = "Read-only except functions"
	}

	operationFilter := ""
	if b.config.EnableOps != "" {
		operationFilter = fmt.Sprintf("Enabled: %s", strings.ToUpper(b.config.EnableOps))
	} else if b.config.DisableOps != "" {
		operationFilter = fmt.Sprintf("Disabled: %s", strings.ToUpper(b.config.DisableOps))
	}

	// Get the actual tools from the MCP server to include full schema info
	mcpTools := b.server.GetTools()
	tools := make([]models.ToolInfo, 0, len(mcpTools))

	for _, mcpTool := range mcpTools {
		// Find the corresponding tool info
		var toolInfo *models.ToolInfo
		for _, ti := range b.tools {
			if ti.Name == mcpTool.Name {
				toolInfo = ti
				break
			}
		}

		if toolInfo != nil {
			// Create a copy with properties from the MCP tool
			info := *toolInfo
			info.Properties = mcpTool.InputSchema
			tools = append(tools, info)
		}
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
		OperationFilter: operationFilter,
		Authentication:  authType,
		ReadOnlyMode:    readOnlyMode,
		MetadataSummary: models.MetadataSummary{
			EntityTypes:     len(b.metadata.EntityTypes),
			EntitySets:      len(b.metadata.EntitySets),
			FunctionImports: len(b.metadata.FunctionImports),
		},
		RegisteredTools: tools,
		TotalTools:      len(tools),
	}, nil
}
