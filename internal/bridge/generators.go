// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"fmt"
	"os"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
)

// generateServiceInfoTool creates a tool to get service information
func (b *ODataMCPBridge) generateServiceInfoTool() {
	toolName := b.formatToolName("odata_service_info", "")

	tool := &mcp.Tool{
		Name:        toolName,
		Description: "Get information about the OData service including metadata, entity sets, and capabilities",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"include_metadata": map[string]any{
					"type":        "boolean",
					"description": "Include detailed metadata information",
					"default":     false,
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleServiceInfo(ctx, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: tool.Description,
		Operation:   constants.OpInfo,
	})
}

// generateEntitySetTools creates tools for an entity set
func (b *ODataMCPBridge) generateEntitySetTools(entitySetName string, entitySet *models.EntitySet) {
	// Get entity type with fallback lookup
	entityType := b.lookupEntityType(entitySet.EntityType)
	if entityType == nil {
		// Always log a warning when entity type is not found - this indicates a potential bug
		fmt.Fprintf(os.Stderr, "[WARNING] Entity type not found for entity set %s: %s (tools will not be generated)\n", entitySetName, entitySet.EntityType)
		return
	}

	// Generate filter/list tool
	if b.config.IsOperationEnabled('F') {
		b.generateFilterTool(entitySetName, entityType)
	}

	// Generate count tool (consider it part of filter/read operations)
	if b.config.IsOperationEnabled('F') {
		b.generateCountTool(entitySetName)
	}

	// Generate search tool if supported
	if entitySet.Searchable && b.config.IsOperationEnabled('S') {
		b.generateSearchTool(entitySetName)
	}

	// Generate get tool
	if b.config.IsOperationEnabled('G') {
		b.generateGetTool(entitySetName, entityType)
	}

	// Generate create tool if allowed and not in read-only mode
	if entitySet.Creatable && !b.config.IsReadOnly() && b.config.IsOperationEnabled('C') {
		b.generateCreateTool(entitySetName, entityType)
	}

	// Generate update tool if allowed and not in read-only mode
	if entitySet.Updatable && !b.config.IsReadOnly() && b.config.IsOperationEnabled('U') {
		b.generateUpdateTool(entitySetName, entityType)
	}

	// Generate delete tool if allowed and not in read-only mode
	if entitySet.Deletable && !b.config.IsReadOnly() && b.config.IsOperationEnabled('D') {
		b.generateDeleteTool(entitySetName, entityType)
	}
}

// generateFilterTool creates a filter/list tool for an entity set
func (b *ODataMCPBridge) generateFilterTool(entitySetName string, _ *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpFilter, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("List/filter %s entities with OData query options", entitySetName)

	// Build input schema with standard OData parameters
	properties := map[string]any{
		b.getParameterName("$filter"): map[string]any{
			"type":        "string",
			"description": "OData filter expression",
		},
		b.getParameterName("$select"): map[string]any{
			"type":        "string",
			"description": "Comma-separated list of properties to select",
		},
		b.getParameterName("$expand"): map[string]any{
			"type":        "string",
			"description": "Navigation properties to expand",
		},
		b.getParameterName("$orderby"): map[string]any{
			"type":        "string",
			"description": "Properties to order by",
		},
		b.getParameterName("$top"): map[string]any{
			"type":        "integer",
			"description": "Maximum number of entities to return",
		},
		b.getParameterName("$skip"): map[string]any{
			"type":        "integer",
			"description": "Number of entities to skip",
		},
		b.getParameterName("$count"): map[string]any{
			"type":        "boolean",
			"description": "Include total count of matching entities (v4) or use $inlinecount for v2",
		},
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": properties,
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityFilter(ctx, entitySetName, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpFilter,
	})
}

// generateCountTool creates a count tool for an entity set
func (b *ODataMCPBridge) generateCountTool(entitySetName string) {
	opName := constants.GetToolOperationName(constants.OpCount, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Get count of %s entities with optional filter", entitySetName)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				b.getParameterName("$filter"): map[string]any{
					"type":        "string",
					"description": "OData filter expression",
				},
			},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityCount(ctx, entitySetName, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpCount,
	})
}

// generateSearchTool creates a search tool for an entity set
func (b *ODataMCPBridge) generateSearchTool(entitySetName string) {
	opName := constants.GetToolOperationName(constants.OpSearch, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Full-text search %s entities", entitySetName)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"search": map[string]any{
					"type":        "string",
					"description": "Search query string",
				},
				b.getParameterName("$select"): map[string]any{
					"type":        "string",
					"description": "Comma-separated list of properties to select",
				},
				b.getParameterName("$top"): map[string]any{
					"type":        "integer",
					"description": "Maximum number of entities to return",
				},
			},
			"required": []string{"search"},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntitySearch(ctx, entitySetName, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpSearch,
	})
}

// generateGetTool creates a get tool for an entity set
func (b *ODataMCPBridge) generateGetTool(entitySetName string, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpGet, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Get a single %s entity by key", entitySetName)

	// Build key properties for input schema using helper
	properties, required := b.buildKeySchema(entityType)

	// Add optional query parameters
	properties[b.getParameterName("$select")] = map[string]any{
		"type":        "string",
		"description": "Comma-separated list of properties to select",
	}
	properties[b.getParameterName("$expand")] = map[string]any{
		"type":        "string",
		"description": "Navigation properties to expand",
	}

	inputSchema := map[string]any{
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

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityGet(ctx, entitySetName, entityType, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpGet,
	})
}

// generateCreateTool creates a create tool for an entity set
func (b *ODataMCPBridge) generateCreateTool(entitySetName string, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpCreate, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Create a new %s entity", entitySetName)

	// Build properties for input schema based on entity type
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, prop := range entityType.Properties {
		// Skip key properties that are auto-generated
		if prop.IsKey {
			continue
		}

		properties[prop.Name] = map[string]any{
			"type":        b.getJSONSchemaType(prop.Type),
			"description": fmt.Sprintf("Property: %s", prop.Name),
		}

		if !prop.Nullable {
			required = append(required, prop.Name)
		}
	}

	inputSchema := map[string]any{
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

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityCreate(ctx, entitySetName, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpCreate,
	})
}

// generateUpdateTool creates an update tool for an entity set
func (b *ODataMCPBridge) generateUpdateTool(entitySetName string, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpUpdate, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Update an existing %s entity", entitySetName)

	// Build key properties for input schema using helper
	properties, required := b.buildKeySchema(entityType)

	// Add updatable properties (optional)
	for _, prop := range entityType.Properties {
		if !prop.IsKey {
			properties[prop.Name] = map[string]any{
				"type":        b.getJSONSchemaType(prop.Type),
				"description": fmt.Sprintf("Property: %s", prop.Name),
			}
		}
	}

	// Add method parameter
	properties["_method"] = map[string]any{
		"type":        "string",
		"description": "HTTP method to use (PUT, PATCH, or MERGE)",
		"enum":        []string{"PUT", "PATCH", "MERGE"},
		"default":     "PUT",
	}

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityUpdate(ctx, entitySetName, entityType, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpUpdate,
	})
}

// generateDeleteTool creates a delete tool for an entity set
func (b *ODataMCPBridge) generateDeleteTool(entitySetName string, entityType *models.EntityType) {
	opName := constants.GetToolOperationName(constants.OpDelete, b.config.ToolShrink)
	toolName := b.formatToolName(opName, entitySetName)

	description := fmt.Sprintf("Delete a %s entity", entitySetName)

	// Build key properties for input schema using helper
	properties, required := b.buildKeySchema(entityType)

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type":       "object",
			"properties": properties,
			"required":   required,
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleEntityDelete(ctx, entitySetName, entityType, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		EntitySet:   entitySetName,
		Operation:   constants.OpDelete,
	})
}

// generateFunctionTool creates a tool for a function import
func (b *ODataMCPBridge) generateFunctionTool(functionName string, function *models.FunctionImport) {
	toolName := b.formatToolName(functionName, "")

	description := fmt.Sprintf("Call function: %s", functionName)

	// Build properties for input schema based on function parameters
	properties := make(map[string]any)
	required := make([]string, 0)

	for _, param := range function.Parameters {
		if param.Mode == "In" || param.Mode == "InOut" {
			properties[param.Name] = map[string]any{
				"type":        b.getJSONSchemaType(param.Type),
				"description": fmt.Sprintf("Parameter: %s", param.Name),
			}

			if !param.Nullable {
				required = append(required, param.Name)
			}
		}
	}

	inputSchema := map[string]any{
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

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleFunctionCall(ctx, functionName, function, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		Function:    functionName,
	})
}
