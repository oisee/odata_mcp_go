package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"strings"
	"sync"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/transport"
)

// Tool represents an MCP tool
type Tool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"inputSchema"`
}

// ToolHandler is a function that handles tool execution
type ToolHandler func(ctx context.Context, args map[string]interface{}) (interface{}, error)

// Request represents an incoming MCP request
type Request struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      interface{}            `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	ctx     context.Context        // Request context from transport (not serialized)
}

// Server represents an MCP server
type Server struct {
	name            string
	version         string
	protocolVersion string // MCP protocol version (can be overridden)
	tools           map[string]*Tool
	toolOrder       []string // Maintains insertion order
	handlers        map[string]ToolHandler
	transport       transport.Transport
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	initialized     bool
}

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	// Disable logging to avoid contaminating stdio communication
	log.SetOutput(io.Discard)

	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		name:            name,
		version:         version,
		protocolVersion: constants.MCPProtocolVersion, // Default protocol version
		tools:           make(map[string]*Tool),
		toolOrder:       make([]string, 0),
		handlers:        make(map[string]ToolHandler),
		ctx:             ctx,
		cancel:          cancel,
	}
}

// SetProtocolVersion sets the MCP protocol version to use
func (s *Server) SetProtocolVersion(version string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.protocolVersion = version
}

// AddTool registers a new tool with the server
func (s *Server) AddTool(tool *Tool, handler ToolHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Only add to order if it's a new tool
	if _, exists := s.tools[tool.Name]; !exists {
		s.toolOrder = append(s.toolOrder, tool.Name)
	}

	s.tools[tool.Name] = tool
	s.handlers[tool.Name] = handler
}

// RemoveTool removes a tool from the server
func (s *Server) RemoveTool(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.tools, name)
	delete(s.handlers, name)

	// Remove from order slice
	for i, toolName := range s.toolOrder {
		if toolName == name {
			s.toolOrder = append(s.toolOrder[:i], s.toolOrder[i+1:]...)
			break
		}
	}
}

// GetTools returns all registered tools in insertion order
func (s *Server) GetTools() []*Tool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	tools := make([]*Tool, 0, len(s.tools))
	for _, name := range s.toolOrder {
		if tool, exists := s.tools[name]; exists {
			tools = append(tools, tool)
		}
	}
	return tools
}

// SetTransport sets the transport for the server
func (s *Server) SetTransport(t interface{}) {
	if trans, ok := t.(transport.Transport); ok {
		s.transport = trans
	}
}

// Run starts the MCP server
func (s *Server) Run() error {
	if s.transport == nil {
		return fmt.Errorf("transport not set")
	}

	// Start the transport with our message handler
	return s.transport.Start(s.ctx)
}

// HandleMessage processes incoming transport messages
func (s *Server) HandleMessage(ctx context.Context, msg *transport.Message) (*transport.Message, error) {
	// Validate JSON-RPC version
	if msg.JSONRPC != "2.0" {
		return s.createErrorResponse(msg.ID, -32600, "Invalid Request", "JSON-RPC version must be 2.0"), nil
	}

	// Convert transport message to internal request
	req := &Request{
		JSONRPC: msg.JSONRPC,
		ID:      msg.ID,
		Method:  msg.Method,
		ctx:     ctx, // Store the request context for passing to handlers
	}

	// Parse params if present
	if len(msg.Params) > 0 {
		var params map[string]interface{}
		if err := json.Unmarshal(msg.Params, &params); err != nil {
			return s.createErrorResponse(msg.ID, -32700, "Parse error", err.Error()), nil
		}
		req.Params = params
	} else {
		// Initialize empty params map
		req.Params = make(map[string]interface{})
	}

	// Handle notifications (no response expected)
	if req.Method == "initialized" {
		s.handleInitialized(req)
		return nil, nil
	}

	// Handle requests
	switch req.Method {
	case "initialize":
		return s.handleInitializeV2(req)
	case "tools/list":
		return s.handleToolsListV2(req)
	case "tools/call":
		return s.handleToolsCallV2(req)
	case "resources/list":
		return s.handleResourcesListV2(req)
	case "prompts/list":
		return s.handlePromptsListV2(req)
	case "ping":
		return s.handlePingV2(req)
	default:
		return s.createErrorResponse(req.ID, -32601, "Method not found", req.Method), nil
	}
}

// Stop stops the MCP server
func (s *Server) Stop() {
	s.cancel()
}

// createErrorResponse creates an error response message
func (s *Server) createErrorResponse(id interface{}, code int, message, data string) *transport.Message {
	var idBytes json.RawMessage

	// Handle different ID types
	switch v := id.(type) {
	case json.RawMessage:
		// Check if it's null
		if string(v) == "null" || len(v) == 0 {
			idBytes = json.RawMessage("0")
		} else {
			idBytes = v
		}
	case nil:
		idBytes = json.RawMessage("0")
	default:
		idBytes, _ = json.Marshal(id)
	}

	return &transport.Message{
		JSONRPC: "2.0",
		ID:      idBytes,
		Error: &transport.Error{
			Code:    code,
			Message: message,
			Data:    json.RawMessage(fmt.Sprintf(`"%s"`, data)),
		},
	}
}

// createResponse creates a success response message
func (s *Server) createResponse(id interface{}, result interface{}) (*transport.Message, error) {
	var idBytes json.RawMessage

	// Handle different ID types - convert null to 0 for Claude Desktop compatibility
	switch v := id.(type) {
	case json.RawMessage:
		// Check if it's null and convert to 0 for Claude Desktop
		if string(v) == "null" || len(v) == 0 {
			idBytes = json.RawMessage("0")
		} else {
			idBytes = v
		}
	case nil:
		idBytes = json.RawMessage("0")
	default:
		idBytes, _ = json.Marshal(id)
	}

	resultBytes, err := json.Marshal(result)
	if err != nil {
		return nil, err
	}

	return &transport.Message{
		JSONRPC: "2.0",
		ID:      idBytes,
		Result:  resultBytes,
	}, nil
}

// handleInitializeV2 handles the initialize request for transport
func (s *Server) handleInitializeV2(req *Request) (*transport.Message, error) {
	// Order fields to match AI Foundry client expectations
	result := map[string]interface{}{
		"capabilities": map[string]interface{}{
			"prompts": map[string]interface{}{
				"listChanged": false,
			},
			"resources": map[string]interface{}{
				"listChanged": false,
				"subscribe":   false,
			},
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"protocolVersion": s.protocolVersion, // Use configurable version
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}

	return s.createResponse(req.ID, result)
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(req *Request) {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()
}

// handleToolsListV2 handles the tools/list request for transport
func (s *Server) handleToolsListV2(req *Request) (*transport.Message, error) {
	s.mu.RLock()
	tools := make([]*Tool, 0, len(s.tools))
	// Use the ordered list to maintain insertion order
	for _, name := range s.toolOrder {
		if tool, exists := s.tools[name]; exists {
			tools = append(tools, tool)
		}
	}
	s.mu.RUnlock()

	result := map[string]interface{}{
		"tools": tools,
	}

	return s.createResponse(req.ID, result)
}

// handleToolsCallV2 handles the tools/call request for transport
func (s *Server) handleToolsCallV2(req *Request) (*transport.Message, error) {
	params, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}

	name, ok := req.Params["name"].(string)
	if !ok {
		return s.createErrorResponse(req.ID, -32602, "Invalid params", "Missing tool name"), nil
	}

	s.mu.RLock()
	handler, exists := s.handlers[name]
	s.mu.RUnlock()

	if !exists {
		return s.createErrorResponse(req.ID, -32602, "Invalid params", fmt.Sprintf("Tool not found: %s", name)), nil
	}

	// Use request context (from HTTP request) instead of server context to preserve headers
	handlerCtx := req.ctx
	if handlerCtx == nil {
		// Fallback to server context if no request context available (e.g., stdio transport)
		handlerCtx = s.ctx
	}

	result, err := handler(handlerCtx, params)
	if err != nil {
		// Map OData errors to appropriate MCP error codes and provide detailed context
		errorCode, errorMessage, errorData := s.categorizeError(err, name)
		return s.createErrorResponse(req.ID, errorCode, errorMessage, errorData), nil
	}

	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	}

	return s.createResponse(req.ID, response)
}

// handlePingV2 handles the ping request for transport
func (s *Server) handlePingV2(req *Request) (*transport.Message, error) {
	result := map[string]interface{}{}
	return s.createResponse(req.ID, result)
}

// handleResourcesListV2 handles the resources/list request for transport
func (s *Server) handleResourcesListV2(req *Request) (*transport.Message, error) {
	// OData MCP bridge doesn't provide resources, only tools
	result := map[string]interface{}{
		"resources": []interface{}{},
	}
	return s.createResponse(req.ID, result)
}

// handlePromptsListV2 handles the prompts/list request for transport
func (s *Server) handlePromptsListV2(req *Request) (*transport.Message, error) {
	// OData MCP bridge doesn't provide prompts, only tools
	result := map[string]interface{}{
		"prompts": []interface{}{},
	}
	return s.createResponse(req.ID, result)
}

// SendNotification sends a notification through the transport
func (s *Server) SendNotification(method string, params interface{}) error {
	if s.transport == nil {
		return fmt.Errorf("transport not set")
	}

	paramsBytes, err := json.Marshal(params)
	if err != nil {
		return err
	}

	msg := &transport.Message{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsBytes,
	}

	return s.transport.WriteMessage(msg)
}

// categorizeError maps OData errors to appropriate MCP error codes and enhances error messages
func (s *Server) categorizeError(err error, toolName string) (int, string, string) {
	errStr := err.Error()

	// Create a comprehensive error message that includes both context and details
	// The MCP client will see this as the main error message
	fullErrorMessage := fmt.Sprintf("OData MCP tool '%s' failed: %s", toolName, errStr)

	// Create structured data for programmatic use (though most clients ignore this)
	errorData := fmt.Sprintf("{\"tool\":\"%s\",\"original_error\":\"%s\"}", toolName, errStr)

	// Check for specific OData error patterns and map to appropriate MCP codes
	switch {
	case strings.Contains(errStr, "HTTP 400") || strings.Contains(errStr, "Bad Request"):
		return -32602, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 401") || strings.Contains(errStr, "Unauthorized"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 403") || strings.Contains(errStr, "Forbidden"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 404") || strings.Contains(errStr, "Not Found"):
		return -32602, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 409") || strings.Contains(errStr, "Conflict"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 422") || strings.Contains(errStr, "Unprocessable"):
		return -32602, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 429") || strings.Contains(errStr, "Too Many Requests"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 500") || strings.Contains(errStr, "Internal Server Error"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 502") || strings.Contains(errStr, "Bad Gateway"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "HTTP 503") || strings.Contains(errStr, "Service Unavailable"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "CSRF token"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "timeout") || strings.Contains(errStr, "deadline exceeded"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "connection refused") || strings.Contains(errStr, "network"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "invalid metadata") || strings.Contains(errStr, "metadata"):
		return -32603, fullErrorMessage, errorData

	case strings.Contains(errStr, "invalid entity") || strings.Contains(errStr, "entity not found"):
		return -32602, fullErrorMessage, errorData

	default:
		// Generic internal error with full context
		return -32603, fullErrorMessage, errorData
	}
}
