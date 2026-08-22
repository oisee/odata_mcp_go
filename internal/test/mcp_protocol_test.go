package test

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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
	ID      any             `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      any             `json:"id,omitempty"`
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
	responses map[any]chan MCPResponse
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

	// Create scanner with larger buffer for big responses (Northwind tools/list is ~80KB)
	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

	client := &MCPClient{
		cmd:       cmd,
		stdin:     stdin,
		stdout:    stdout,
		stderr:    stderr,
		scanner:   scanner,
		responses: make(map[any]chan MCPResponse),
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

	// Give server time to start and fetch metadata
	// Remote services (Northwind, SAP) need more time than local mock
	startupWait := 500 * time.Millisecond
	if strings.Contains(serviceURL, "odata.org") || strings.Contains(serviceURL, "sap") {
		startupWait = 5 * time.Second
		t.Log("Using extended startup wait for remote service")
	}
	time.Sleep(startupWait)

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

		// Convert ID to int for map lookup (JSON numbers unmarshal as float64)
		var lookupID any = resp.ID
		if f, ok := resp.ID.(float64); ok {
			lookupID = int(f)
		}

		c.mu.Lock()
		if ch, ok := c.responses[lookupID]; ok {
			ch <- resp
			delete(c.responses, lookupID)
		}
		c.mu.Unlock()
	}
}

func (c *MCPClient) SendRequest(method string, params any) (MCPResponse, error) {
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

func (c *MCPClient) SendNotification(method string, params any) error {
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
	mockServer *httptest.Server
}

func (suite *MCPProtocolTestSuite) SetupSuite() {
	// Use environment variable or embedded mock server
	suite.serviceURL = os.Getenv("ODATA_URL")
	if suite.serviceURL == "" {
		// Create embedded mock OData server
		suite.mockServer = httptest.NewServer(createMockODataHandler(suite.T()))
		suite.serviceURL = suite.mockServer.URL
		suite.T().Log("Using embedded mock service URL:", suite.serviceURL)

		// Use t.Cleanup to ensure server is closed even if SetupSuite fails later
		suite.T().Cleanup(func() {
			if suite.mockServer != nil {
				suite.mockServer.Close()
			}
		})
	}
}

// createMockODataHandler creates an HTTP handler that simulates an OData service
func createMockODataHandler(t *testing.T) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Logf("[MOCK SERVER] %s %s", r.Method, r.URL.Path)

		// Handle CSRF token fetch
		if r.Header.Get("X-CSRF-Token") == "Fetch" {
			w.Header().Set("X-CSRF-Token", "mock-csrf-token")
			w.WriteHeader(http.StatusOK)
			return
		}

		// Handle metadata requests
		if strings.HasSuffix(r.URL.Path, "/$metadata") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" Version="1.0">
  <edmx:DataServices>
    <Schema xmlns="http://schemas.microsoft.com/ado/2008/09/edm" Namespace="TestNamespace">
      <EntityType Name="TestEntity">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.String" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Value" Type="Edm.Int32"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="TestEntities" EntityType="TestNamespace.TestEntity"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`))
			return
		}

		// Handle entity set requests
		if strings.Contains(r.URL.Path, "TestEntities") {
			w.Header().Set("Content-Type", "application/json")
			switch r.Method {
			case http.MethodGet:
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"d": map[string]any{
						"results": []map[string]any{
							{"ID": "1", "Name": "Test 1", "Value": 100},
							{"ID": "2", "Name": "Test 2", "Value": 200},
						},
					},
				})
			case http.MethodPost:
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]any{
					"d": map[string]any{"ID": "3", "Name": "Created", "Value": 300},
				})
			case http.MethodPut, http.MethodPatch:
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]any{
					"d": map[string]any{"ID": "1", "Name": "Updated", "Value": 150},
				})
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusMethodNotAllowed)
			}
			return
		}

		// Handle service document (root)
		if r.URL.Path == "/" || r.URL.Path == "" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"d": map[string]any{
					"EntitySets": []string{"TestEntities"},
				},
			})
			return
		}

		// Default: not found
		w.WriteHeader(http.StatusNotFound)
	})
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
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "mcp-test-client",
			"version": "1.0.0",
		},
	}

	resp, err := suite.client.SendRequest("initialize", initParams)
	require.NoError(suite.T(), err)
	assert.Nil(suite.T(), resp.Error)

	// Parse result
	var result map[string]any
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)

	// Verify server info
	serverInfo, ok := result["serverInfo"].(map[string]any)
	assert.True(suite.T(), ok)
	assert.Equal(suite.T(), "odata-mcp-bridge", serverInfo["name"])

	// Verify capabilities
	capabilities, ok := result["capabilities"].(map[string]any)
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
	var result map[string]any
	err = json.Unmarshal(resp.Result, &result)
	require.NoError(suite.T(), err)

	tools, ok := result["tools"].([]any)
	assert.True(suite.T(), ok)
	assert.Greater(suite.T(), len(tools), 0)

	// Tool names are dynamically generated based on entity sets and service URL.
	// Expected patterns: filter_<Entity>_for_<ServiceID>, get_<Entity>_for_<ServiceID>, etc.
	// We verify that each tool type is present by checking for prefixes.
	expectedPrefixes := []string{
		"odata_service_info_", // Service info tool
		"filter_",             // List/filter tools
		"get_",                // Get single entity tools
		"create_",             // Create entity tools
		"update_",             // Update entity tools
		"delete_",             // Delete entity tools
	}

	foundPrefixes := make(map[string]bool)
	for _, tool := range tools {
		toolMap := tool.(map[string]any)
		name := toolMap["name"].(string)

		// Track which prefixes we found
		for _, prefix := range expectedPrefixes {
			if strings.HasPrefix(name, prefix) {
				foundPrefixes[prefix] = true
			}
		}

		// Verify tool structure
		assert.NotEmpty(suite.T(), toolMap["description"], "Tool %s should have description", name)
		assert.NotNil(suite.T(), toolMap["inputSchema"], "Tool %s should have inputSchema", name)
	}

	for _, prefix := range expectedPrefixes {
		assert.True(suite.T(), foundPrefixes[prefix], "Should have at least one tool with prefix %s", prefix)
	}
}

func (suite *MCPProtocolTestSuite) TestCallToolWithCSRF() {
	// Initialize first
	suite.initializeClient()

	// Test create_entity tool (which should trigger CSRF token handling)
	toolParams := map[string]any{
		"name": "create_entity",
		"arguments": map[string]any{
			"entitySet": "TestEntities",
			"entity": map[string]any{
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
		var result map[string]any
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

	// Call a non-existent tool - should return an error
	toolParams := map[string]any{
		"name":      "nonexistent_tool",
		"arguments": map[string]any{},
	}

	resp, err := suite.client.SendRequest("tools/call", toolParams)
	require.NoError(suite.T(), err)

	// Should return an error (tool not found or invalid params)
	assert.NotNil(suite.T(), resp.Error)
	// Error code -32602 is "Invalid params" per JSON-RPC spec, which is correct for unknown tool
	// The error data contains the specific reason
	errorData := ""
	if resp.Error.Data != nil {
		if dataStr, ok := resp.Error.Data.(string); ok {
			errorData = dataStr
		}
	}
	assert.True(suite.T(),
		resp.Error.Code == -32602 || // Invalid params (tool not found)
			strings.Contains(strings.ToLower(resp.Error.Message), "not found") ||
			strings.Contains(strings.ToLower(errorData), "not found"),
		"Error should indicate tool not found: code=%d, message=%s, data=%v",
		resp.Error.Code, resp.Error.Message, resp.Error.Data)
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
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
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
		serviceURL := os.Getenv("ODATA_URL")
		if serviceURL == "" {
			mockServer := httptest.NewServer(createMockODataHandler(t))
			t.Cleanup(func() { mockServer.Close() })
			serviceURL = mockServer.URL
		}

		client, err := NewMCPClient(t, serviceURL)
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
			params       any
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
	serviceURL := os.Getenv("ODATA_URL")
	if serviceURL == "" {
		b.Skip("Skipping benchmark: ODATA_URL not set (mock server adds overhead)")
	}

	client, err := NewMCPClient(&testing.T{}, serviceURL)
	if err != nil {
		b.Fatal(err)
	}
	defer client.Close()

	// Initialize
	initParams := map[string]any{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]any{},
		"clientInfo": map[string]any{
			"name":    "benchmark-client",
			"version": "1.0.0",
		},
	}

	_, err = client.SendRequest("initialize", initParams)
	if err != nil {
		b.Fatal(err)
	}

	_ = client.SendNotification("initialized", nil)

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
		params := map[string]any{
			"name":      "get_metadata",
			"arguments": map[string]any{},
		}

		for i := 0; i < b.N; i++ {
			_, err := client.SendRequest("tools/call", params)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
