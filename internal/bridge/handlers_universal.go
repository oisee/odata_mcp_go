// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/zmcp/odata-mcp/internal/mcp"
	"github.com/zmcp/odata-mcp/internal/models"
)

// generateUniversalTool creates a single universal OData tool
// This is used when --universal flag is set to reduce tool count for large services
func (b *ODataMCPBridge) generateUniversalTool() {
	toolName := b.formatToolName("OData", "")

	description := b.generateUniversalDescription()

	tool := &mcp.Tool{
		Name:        toolName,
		Description: description,
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"action": map[string]any{
					"type":        "string",
					"description": "Operation: list|get|create|update|delete|count|search|call",
					"enum":        []string{"list", "get", "create", "update", "delete", "count", "search", "call"},
				},
				"target": map[string]any{
					"type":        "string",
					"description": "Entity set name (e.g., 'Products') or function name",
				},
				"params": map[string]any{
					"type":        "object",
					"description": "Action-specific parameters (filter, select, expand, orderby, top, skip, key, data, method)",
				},
			},
			"required": []string{"action", "target"},
		},
	}

	handler := func(ctx context.Context, args map[string]any) (any, error) {
		return b.handleUniversalTool(ctx, args)
	}

	b.registerTool(tool, handler, &models.ToolInfo{
		Name:        toolName,
		Description: description,
		Operation:   "universal",
	})
}

// generateUniversalDescription generates a description listing all entities and functions
func (b *ODataMCPBridge) generateUniversalDescription() string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("OData service: %s\n\n", b.config.ServiceURL))

	// List entities with capabilities (sorted)
	sb.WriteString("Entities:\n")
	entityNames := make([]string, 0, len(b.metadata.EntitySets))
	for name := range b.metadata.EntitySets {
		if b.shouldIncludeEntity(name) {
			entityNames = append(entityNames, name)
		}
	}
	sort.Strings(entityNames)

	for _, name := range entityNames {
		es := b.metadata.EntitySets[name]
		ops := []string{"list", "get", "count"}
		if es.Searchable {
			ops = append(ops, "search")
		}
		if es.Creatable && !b.config.IsReadOnly() {
			ops = append(ops, "create")
		}
		if es.Updatable && !b.config.IsReadOnly() {
			ops = append(ops, "update")
		}
		if es.Deletable && !b.config.IsReadOnly() {
			ops = append(ops, "delete")
		}
		sb.WriteString(fmt.Sprintf("  %s [%s]\n", name, strings.Join(ops, ",")))
	}

	// List functions (sorted)
	functionNames := make([]string, 0, len(b.metadata.FunctionImports))
	for name := range b.metadata.FunctionImports {
		if b.shouldIncludeFunction(name) {
			functionNames = append(functionNames, name)
		}
	}

	if len(functionNames) > 0 {
		sort.Strings(functionNames)
		sb.WriteString("\nFunctions:\n")
		for _, name := range functionNames {
			fn := b.metadata.FunctionImports[name]
			// Skip modifying functions in read-only mode
			if b.config.ReadOnly || (!b.config.AllowModifyingFunctions() && b.isFunctionModifying(fn)) {
				continue
			}
			sb.WriteString(fmt.Sprintf("  %s(%s)\n", name, b.formatFunctionParams(fn)))
		}
	}

	// Usage examples
	sb.WriteString(`
Actions:
  list   - Query entities with filter/select/expand/orderby/top/skip
  get    - Retrieve single entity by key
  create - Create new entity
  update - Update existing entity (method: PUT|PATCH|MERGE)
  delete - Delete entity by key
  count  - Count entities matching filter
  search - Full-text search (if supported)
  call   - Execute function/action

Examples:
  action="list" target="Products" params={"filter":"Price gt 100","top":10}
  action="get" target="Products" params={"key":{"ID":123}}
  action="create" target="Orders" params={"data":{"CustomerID":"C001"}}
  action="call" target="ReleaseOrder" params={"OrderID":"O001"}
`)

	return sb.String()
}

// formatFunctionParams formats function parameters for the description
func (b *ODataMCPBridge) formatFunctionParams(fn *models.FunctionImport) string {
	var params []string
	for _, p := range fn.Parameters {
		if p.Mode == "In" || p.Mode == "InOut" {
			params = append(params, fmt.Sprintf("%s: %s", p.Name, p.Type))
		}
	}
	return strings.Join(params, ", ")
}

// handleUniversalTool routes universal tool calls to appropriate handlers
func (b *ODataMCPBridge) handleUniversalTool(ctx context.Context, args map[string]any) (any, error) {
	action, ok := args["action"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: action")
	}

	target, ok := args["target"].(string)
	if !ok {
		return nil, fmt.Errorf("missing required parameter: target")
	}

	// Get params (optional)
	params, _ := args["params"].(map[string]any)
	if params == nil {
		params = make(map[string]any)
	}

	// Check if target is an entity set or function
	entitySet, isEntity := b.metadata.EntitySets[target]
	function, isFunction := b.metadata.FunctionImports[target]

	if !isEntity && !isFunction {
		return nil, fmt.Errorf("unknown target: %s (not an entity set or function)", target)
	}

	// Route to appropriate handler
	switch action {
	case "list":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		return b.handleEntityFilter(ctx, target, params)

	case "get":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		entityType := b.lookupEntityType(entitySet.EntityType)
		if entityType == nil {
			return nil, fmt.Errorf("entity type not found for %s", target)
		}
		// Extract key from params
		if keyObj, ok := params["key"].(map[string]any); ok {
			// Merge key into params for the handler
			for k, v := range keyObj {
				params[k] = v
			}
		}
		return b.handleEntityGet(ctx, target, entityType, params)

	case "create":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		if !entitySet.Creatable {
			return nil, fmt.Errorf("%s does not support create", target)
		}
		if b.config.IsReadOnly() {
			return nil, fmt.Errorf("create operation not allowed in read-only mode")
		}
		// Extract data from params
		if dataObj, ok := params["data"].(map[string]any); ok {
			return b.handleEntityCreate(ctx, target, dataObj)
		}
		return b.handleEntityCreate(ctx, target, params)

	case "update":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		if !entitySet.Updatable {
			return nil, fmt.Errorf("%s does not support update", target)
		}
		if b.config.IsReadOnly() {
			return nil, fmt.Errorf("update operation not allowed in read-only mode")
		}
		entityType := b.lookupEntityType(entitySet.EntityType)
		if entityType == nil {
			return nil, fmt.Errorf("entity type not found for %s", target)
		}
		// Extract key and data from params
		mergedParams := make(map[string]any)
		if keyObj, ok := params["key"].(map[string]any); ok {
			for k, v := range keyObj {
				mergedParams[k] = v
			}
		}
		if dataObj, ok := params["data"].(map[string]any); ok {
			for k, v := range dataObj {
				mergedParams[k] = v
			}
		}
		if method, ok := params["method"].(string); ok {
			mergedParams["_method"] = method
		}
		return b.handleEntityUpdate(ctx, target, entityType, mergedParams)

	case "delete":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		if !entitySet.Deletable {
			return nil, fmt.Errorf("%s does not support delete", target)
		}
		if b.config.IsReadOnly() {
			return nil, fmt.Errorf("delete operation not allowed in read-only mode")
		}
		entityType := b.lookupEntityType(entitySet.EntityType)
		if entityType == nil {
			return nil, fmt.Errorf("entity type not found for %s", target)
		}
		// Extract key from params
		if keyObj, ok := params["key"].(map[string]any); ok {
			for k, v := range keyObj {
				params[k] = v
			}
		}
		return b.handleEntityDelete(ctx, target, entityType, params)

	case "count":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		return b.handleEntityCount(ctx, target, params)

	case "search":
		if !isEntity {
			return nil, fmt.Errorf("%s is not an entity set", target)
		}
		if !entitySet.Searchable {
			return nil, fmt.Errorf("%s does not support search", target)
		}
		return b.handleEntitySearch(ctx, target, params)

	case "call":
		if !isFunction {
			return nil, fmt.Errorf("%s is not a function", target)
		}
		// Check read-only mode for modifying functions
		if b.config.ReadOnly || (!b.config.AllowModifyingFunctions() && b.isFunctionModifying(function)) {
			return nil, fmt.Errorf("function %s not allowed in read-only mode", target)
		}
		return b.handleFunctionCall(ctx, target, function, params)

	default:
		return nil, fmt.Errorf("unknown action: %s (valid: list, get, create, update, delete, count, search, call)", action)
	}
}

// handleUniversalServiceInfo handles the service_info action for universal mode
func (b *ODataMCPBridge) handleUniversalServiceInfo() (any, error) {
	info := map[string]any{
		"service_url":      b.config.ServiceURL,
		"entity_sets":      len(b.metadata.EntitySets),
		"entity_types":     len(b.metadata.EntityTypes),
		"function_imports": len(b.metadata.FunctionImports),
		"schema_namespace": b.metadata.SchemaNamespace,
		"container_name":   b.metadata.ContainerName,
		"version":          b.metadata.Version,
		"mode":             "universal",
	}

	// Add entity list with capabilities
	entities := make(map[string][]string)
	for name, es := range b.metadata.EntitySets {
		if !b.shouldIncludeEntity(name) {
			continue
		}
		ops := []string{"list", "get", "count"}
		if es.Searchable {
			ops = append(ops, "search")
		}
		if es.Creatable && !b.config.IsReadOnly() {
			ops = append(ops, "create")
		}
		if es.Updatable && !b.config.IsReadOnly() {
			ops = append(ops, "update")
		}
		if es.Deletable && !b.config.IsReadOnly() {
			ops = append(ops, "delete")
		}
		entities[name] = ops
	}
	info["entities"] = entities

	// Add function list
	functions := make([]string, 0)
	for name, fn := range b.metadata.FunctionImports {
		if !b.shouldIncludeFunction(name) {
			continue
		}
		if b.config.ReadOnly || (!b.config.AllowModifyingFunctions() && b.isFunctionModifying(fn)) {
			continue
		}
		functions = append(functions, name)
	}
	sort.Strings(functions)
	info["functions"] = functions

	// Add hints
	hints := b.hintManager.GetHints(b.config.ServiceURL)
	if hints != nil {
		info["implementation_hints"] = hints
	}

	response, err := json.Marshal(info)
	if err != nil {
		return "Error formatting service info", err
	}

	return string(response), nil
}
