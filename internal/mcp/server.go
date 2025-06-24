package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"sync"

	"github.com/odata-mcp/go/internal/constants"
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
}

// Response represents an outgoing MCP response
type Response struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *Error      `json:"error,omitempty"`
}

// Error represents an MCP error
type Error struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// Notification represents an MCP notification (no ID)
type Notification struct {
	JSONRPC string                 `json:"jsonrpc"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// Server represents an MCP server
type Server struct {
	name        string
	version     string
	tools       map[string]*Tool
	toolOrder   []string    // Maintains insertion order
	handlers    map[string]ToolHandler
	input       io.Reader
	output      io.Writer
	ctx         context.Context
	cancel      context.CancelFunc
	mu          sync.RWMutex
	initialized bool
}

// NewServer creates a new MCP server
func NewServer(name, version string) *Server {
	// Disable logging to avoid contaminating stdio communication
	log.SetOutput(ioutil.Discard)
	
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		name:     name,
		version:  version,
		tools:     make(map[string]*Tool),
		toolOrder: make([]string, 0),
		handlers:  make(map[string]ToolHandler),
		input:    os.Stdin,
		output:   os.Stdout,
		ctx:      ctx,
		cancel:   cancel,
	}
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

// SetIO sets the input and output streams for the server
func (s *Server) SetIO(input io.Reader, output io.Writer) {
	s.input = input
	s.output = output
}

// Run starts the MCP server
func (s *Server) Run() error {
	scanner := bufio.NewScanner(s.input)
	// Increase buffer size to handle large messages (10MB)
	const maxScanTokenSize = 10 * 1024 * 1024
	buf := make([]byte, maxScanTokenSize)
	scanner.Buffer(buf, maxScanTokenSize)
	
	for scanner.Scan() {
		select {
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
		}
		
		line := scanner.Text()
		if line == "" {
			continue
		}
		
		if err := s.handleMessage(line); err != nil {
			// Error already sent as JSON-RPC response, don't log to stdout/stderr
		}
	}
	
	return scanner.Err()
}

// Stop stops the MCP server
func (s *Server) Stop() {
	s.cancel()
}

// handleMessage processes a single JSON-RPC message
func (s *Server) handleMessage(line string) error {
	// Parse as generic JSON first to check structure
	var rawMsg map[string]interface{}
	if err := json.Unmarshal([]byte(line), &rawMsg); err != nil {
		// Can't send error response if we can't parse JSON
		return err
	}
	
	// Check if it's a notification (no ID) or request (has ID)
	var req Request
	if err := json.Unmarshal([]byte(line), &req); err != nil {
		// Try to get ID from raw message for error response
		var id interface{}
		if rawID, exists := rawMsg["id"]; exists {
			id = rawID
		}
		return s.sendError(id, -32700, "Parse error", err.Error())
	}
	
	// Handle notifications differently (no response expected)
	if req.Method == "initialized" {
		return s.handleInitialized(&req)
	}
	
	// For requests, ensure we have an ID (except notifications)
	if req.ID == nil && req.Method != "initialized" {
		return s.sendError(1, -32600, "Invalid request", "Missing ID for request")
	}
	
	switch req.Method {
	case "initialize":
		return s.handleInitialize(&req)
	case "tools/list":
		return s.handleToolsList(&req)
	case "tools/call":
		return s.handleToolsCall(&req)
	case "ping":
		return s.handlePing(&req)
	default:
		return s.sendError(req.ID, -32601, "Method not found", req.Method)
	}
}

// handleInitialize handles the initialize request
func (s *Server) handleInitialize(req *Request) error {
	result := map[string]interface{}{
		"protocolVersion": constants.MCPProtocolVersion,
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]interface{}{
			"name":    s.name,
			"version": s.version,
		},
	}
	
	return s.sendResponse(req.ID, result)
}

// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(req *Request) error {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()
	return nil
}

// handleToolsList handles the tools/list request
func (s *Server) handleToolsList(req *Request) error {
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
	
	return s.sendResponse(req.ID, result)
}

// handleToolsCall handles the tools/call request
func (s *Server) handleToolsCall(req *Request) error {
	params, ok := req.Params["arguments"].(map[string]interface{})
	if !ok {
		params = make(map[string]interface{})
	}
	
	name, ok := req.Params["name"].(string)
	if !ok {
		return s.sendError(req.ID, -32602, "Invalid params", "Missing tool name")
	}
	
	s.mu.RLock()
	handler, exists := s.handlers[name]
	s.mu.RUnlock()
	
	if !exists {
		return s.sendError(req.ID, -32602, "Invalid params", fmt.Sprintf("Tool not found: %s", name))
	}
	
	result, err := handler(s.ctx, params)
	if err != nil {
		// Map OData errors to appropriate MCP error codes and provide detailed context
		errorCode, errorMessage, errorData := s.categorizeError(err, name)
		return s.sendError(req.ID, errorCode, errorMessage, errorData)
	}
	
	response := map[string]interface{}{
		"content": []map[string]interface{}{
			{
				"type": "text",
				"text": result,
			},
		},
	}
	
	return s.sendResponse(req.ID, response)
}

// handlePing handles the ping request
func (s *Server) handlePing(req *Request) error {
	result := map[string]interface{}{}
	return s.sendResponse(req.ID, result)
}

// sendResponse sends a JSON-RPC response
func (s *Server) sendResponse(id interface{}, result interface{}) error {
	response := Response{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	
	data, err := json.Marshal(response)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(s.output, "%s\n", data)
	return err
}

// sendError sends a JSON-RPC error response
func (s *Server) sendError(id interface{}, code int, message, data string) error {
	response := Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &Error{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	
	responseData, err := json.Marshal(response)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(s.output, "%s\n", responseData)
	return err
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

// sendNotification sends a JSON-RPC notification
func (s *Server) sendNotification(method string, params map[string]interface{}) error {
	notification := Notification{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	
	data, err := json.Marshal(notification)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(s.output, "%s\n", data)
	return err
}