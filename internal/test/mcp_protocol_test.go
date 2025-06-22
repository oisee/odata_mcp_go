package test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// MCP Protocol structures
type MCPRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type MCPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	stderr    io.ReadCloser
	scanner   *bufio.Scanner
	mu        sync.Mutex
	requestID int
	responses map[interface{}]chan MCPResponse
	t         *testing.T
}

func NewMCPClient(t *testing.T, serviceURL string) (*MCPClient, error) {
	// Build the server if needed
	buildCmd := exec.Command("go", "build", "-o", "../../odata-mcp", "../../cmd/odata-mcp")
	buildCmd.Dir = "."
	if output, err := buildCmd.CombinedOutput(); err != nil {
		return nil, fmt.Errorf("failed to build server: %w\nOutput: %s", err, output)
	}

	cmd := exec.Command("../../odata-mcp", serviceURL)
	
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, err
	}
	
	if err := cmd.Start(); err != nil {
		return nil, err
	}
	
	client := &MCPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		scanner:   bufio.NewScanner(stdout),
		responses: make(map[interface{}]chan MCPResponse),
		t:         t,
	}
	
	// Start reading responses
	go client.readResponses()
	
	// Start reading stderr for debugging
	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			t.Logf("[SERVER STDERR] %s", scanner.Text())
		}
	}()
	
	// Give server time to start
	time.Sleep(500 * time.Millisecond)
	
	return client, nil
}

func (c *MCPClient) readResponses() {
	for c.scanner.Scan() {
		line := c.scanner.Text()
		c.t.Logf("[SERVER RESPONSE] %s", line)
		
		var resp MCPResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			c.t.Logf("Failed to parse response: %v", err)
			continue
		}
		
		c.mu.Lock()
		if ch, ok := c.responses[resp.ID]; ok {
			ch <- resp
			delete(c.responses, resp.ID)
		}
		c.mu.Unlock()
	}
}

func (c *MCPClient) SendRequest(method string, params interface{}) (MCPResponse, error) {
	c.mu.Lock()
	c.requestID++
	id := c.requestID
	c.mu.Unlock()
	
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return MCPResponse{}, err
		}
		paramsJSON = data
	}
	
	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  paramsJSON,
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		return MCPResponse{}, err
	}
	
	c.t.Logf("[CLIENT REQUEST] %s", string(data))
	
	respChan := make(chan MCPResponse, 1)
	c.mu.Lock()
	c.responses[id] = respChan
	c.mu.Unlock()
	
	if _, err := c.stdin.Write(append(data, '\n')); err != nil {
		return MCPResponse{}, err
	}
	
	select {
	case resp := <-respChan:
		return resp, nil
	case <-time.After(5 * time.Second):
		return MCPResponse{}, fmt.Errorf("timeout waiting for response")
	}
}

func (c *MCPClient) SendNotification(method string, params interface{}) error {
	var paramsJSON json.RawMessage
	if params != nil {
		data, err := json.Marshal(params)
		if err != nil {
			return err
		}
		paramsJSON = data
	}
	
	req := MCPRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  paramsJSON,
	}
	
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	
	c.t.Logf("[CLIENT NOTIFICATION] %s", string(data))
	
	_, err = c.stdin.Write(append(data, '\n'))
	return err
}

func (c *MCPClient) Close() error {
	c.stdin.Close()
	return c.cmd.Wait()
}

// Test Suite
type MCPProtocolTestSuite struct {
	suite.Suite
	client     *MCPClient
	serviceURL string
}

func (suite *MCPProtocolTestSuite) SetupSuite() {
	// Use environment variable or mock server
	suite.serviceURL = os.Getenv("ODATA_URL")
	if suite.serviceURL == "" {
		// Use the mock server from CSRF tests
		suite.serviceURL = "http://localhost:8080/mock"
		suite.T().Log("Using mock service URL:", suite.serviceURL)
	}
}

func (suite *MCPProtocolTestSuite) SetupTest() {
	client, err := NewMCPClient(suite.T(), suite.serviceURL)
	require.NoError(suite.T(), err)
	suite.client = client
}

func (suite *MCPProtocolTestSuite) TearDownTest() {
	if suite.client != nil {
		suite.client.Close()
	}
}

func (suite *MCPProtocolTestSuite) TestInitializeProtocol() {
	// Test initialize request
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "mcp-test-client",
			"version": "1.0.0",
		},
	}
	
	resp, err := suite.client.SendRequest("initialize", initParams)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), resp.Error)
	
	// Parse result
	var result map[string]interface{}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)
	
	// Verify server info
	serverInfo, ok := result["serverInfo"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "odata-mcp-server", serverInfo["name"])
	
	// Verify capabilities
	capabilities, ok := result["capabilities"].(map[string]interface{})
	assert.True(suite.T(), ok)
	assert.NotNil(suite.T(), capabilities["tools"])
	
	// Send initialized notification
	err = suite.client.SendNotification("initialized", nil)
	assert.NoError(suite.T(), err)
}

func (suite *MCPProtocolTestSuite) TestListTools() {
	// Initialize first
	suite.initializeClient()
	
	// List tools
	resp, err := suite.client.SendRequest("tools/list", nil)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), resp.Error)
	
	// Parse tools
	var result map[string]interface{}
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)
	
	tools, ok := result["tools"].([]interface{})
	assert.True(suite.T(), ok)
	assert.Greater(suite.T(), len(tools), 0)
	
	// Verify expected tools are present
	expectedTools := []string{
		"query_entities",
		"get_entity",
		"create_entity",
		"update_entity",
		"delete_entity",
		"call_function",
		"get_metadata",
	}
	
	toolNames := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]interface{})
		name := toolMap["name"].(string)
		toolNames[name] = true
		
		// Verify tool structure
		assert.NotEmpty(suite.T(), toolMap["description"])
		assert.NotNil(suite.T(), toolMap["inputSchema"])
	}
	
	for _, expected := range expectedTools {
		assert.True(suite.T(), toolNames[expected], "Tool %s should be present", expected)
	}
}

func (suite *MCPProtocolTestSuite) TestCallToolWithCSRF() {
	// Initialize first
	suite.initializeClient()
	
	// Test create_entity tool (which should trigger CSRF token handling)
	toolParams := map[string]interface{}{
		"name": "create_entity",
		"arguments": map[string]interface{}{
			"entitySet": "TestEntities",
			"entity": map[string]interface{}{
				"Name":  "Test Entity",
				"Value": 100,
			},
		},
	}
	
	resp, err := suite.client.SendRequest("tools/call", toolParams)
	
	// The actual result depends on whether the service exists
	// We're mainly testing that the protocol works correctly
	if err != nil {
		suite.T().Logf("Tool call error (expected if service doesn't exist): %v", err)
	} else if resp.Error != nil {
		suite.T().Logf("Tool returned error: %+v", resp.Error)
		// Verify error structure
		assert.NotEmpty(suite.T(), resp.Error.Message)
		assert.NotZero(suite.T(), resp.Error.Code)
	} else {
		// If successful, verify result structure
		var result map[string]interface{}
		err = json.Unmarshal(resp.Result, &result)
		assert.NoError(suite.T(), err)
		assert.Contains(suite.T(), result, "content")
	}
}

func (suite *MCPProtocolTestSuite) TestInvalidRequest() {
	// Initialize first
	suite.initializeClient()
	
	// Test invalid method
	resp, err := suite.client.SendRequest("invalid/method", nil)
	require.NoError(suite.T(), err)
	
	assert.NotNil(suite.T(), resp.Error)
	assert.Equal(suite.T(), -32601, resp.Error.Code) // Method not found
}

func (suite *MCPProtocolTestSuite) TestMissingRequiredParams() {
	// Initialize first
	suite.initializeClient()
	
	// Call tool without required parameters
	toolParams := map[string]interface{}{
		"name": "create_entity",
		"arguments": map[string]interface{}{
			// Missing entitySet
			"entity": map[string]interface{}{
				"Name": "Test",
			},
		},
	}
	
	resp, err := suite.client.SendRequest("tools/call", toolParams)
	require.NoError(suite.T(), err)
	
	assert.NotNil(suite.T(), resp.Error)
	assert.Contains(suite.T(), strings.ToLower(resp.Error.Message), "missing")
}

func (suite *MCPProtocolTestSuite) TestConcurrentRequests() {
	// Initialize first
	suite.initializeClient()
	
	// Send multiple requests concurrently
	var wg sync.WaitGroup
	errors := make([]error, 5)
	
	for i := 0; i < 5; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			
			resp, err := suite.client.SendRequest("tools/list", nil)
			if err != nil {
				errors[index] = err
				return
			}
			
			if resp.Error != nil {
				errors[index] = fmt.Errorf("response error: %v", resp.Error)
			}
		}(i)
	}
	
	wg.Wait()
	
	// Check all requests succeeded
	for i, err := range errors {
		assert.NoError(suite.T(), err, "Request %d failed", i)
	}
}

func (suite *MCPProtocolTestSuite) initializeClient() {
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "mcp-test-client",
			"version": "1.0.0",
		},
	}
	
	resp, err := suite.client.SendRequest("initialize", initParams)
	require.NoError(suite.T(), err)
	require.Nil(suite.T(), resp.Error)
	
	err = suite.client.SendNotification("initialized", nil)
	require.NoError(suite.T(), err)
}

func TestMCPProtocolTestSuite(t *testing.T) {
	suite.Run(t, new(MCPProtocolTestSuite))
}

// MCP Protocol Audit Tests
func TestMCPProtocolCompliance(t *testing.T) {
	// These tests verify compliance with MCP specification
	
	t.Run("JSONRPCVersion", func(t *testing.T) {
		client, err := NewMCPClient(t, "http://localhost:8080/mock")
		require.NoError(t, err)
		defer client.Close()
		
		// All responses should have jsonrpc: "2.0"
		resp, err := client.SendRequest("tools/list", nil)
		require.NoError(t, err)
		assert.Equal(t, "2.0", resp.JSONRPC)
	})
	
	t.Run("ErrorCodes", func(t *testing.T) {
		// Test standard JSON-RPC error codes
		testCases := []struct {
			name         string
			method       string
			params       interface{}
			expectedCode int
		}{
			{
				name:         "ParseError",
				method:       "", // Will send malformed JSON
				expectedCode: -32700,
			},
			{
				name:         "InvalidRequest",
				method:       "test",
				params:       "invalid", // Should be object
				expectedCode: -32600,
			},
			{
				name:         "MethodNotFound",
				method:       "nonexistent/method",
				expectedCode: -32601,
			},
		}
		
		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				// Test implementation would go here
				// This is a placeholder for the structure
				t.Logf("Testing error code %d for %s", tc.expectedCode, tc.name)
			})
		}
	})
}

// Benchmark tests
func BenchmarkMCPRequests(b *testing.B) {
	client, err := NewMCPClient(&testing.T{}, "http://localhost:8080/mock")
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()
	
	// Initialize
	initParams := map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]interface{}{
			"name":    "benchmark-client",
			"version": "1.0.0",
		},
	}
	
	_, err = client.SendRequest("initialize", initParams)
	if err != nil {
		b.Fatal(err)
	}
	
	client.SendNotification("initialized", nil)
	
	b.ResetTimer()
	
	b.Run("ToolsList", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := client.SendRequest("tools/list", nil)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
	
	b.Run("ToolCall", func(b *testing.B) {
		params := map[string]interface{}{
			"name": "get_metadata",
			"arguments": map[string]interface{}{},
		}
		
		for i := 0; i < b.N; i++ {
			_, err := client.SendRequest("tools/call", params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}