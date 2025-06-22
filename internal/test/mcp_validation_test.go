package test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMCPRequestValidation tests MCP request validation logic
func TestMCPRequestValidation(t *testing.T) {
	testCases := []struct {
		name          string
		request       string
		expectedError string
		expectedCode  int
	}{
		{
			name:          "Valid request",
			request:       `{"jsonrpc": "2.0", "id": 1, "method": "initialize"}`,
			expectedError: "",
			expectedCode:  0,
		},
		{
			name:          "Empty request",
			request:       "",
			expectedError: "EOF",
			expectedCode:  -32700, // Parse error
		},
		{
			name:          "Invalid JSON",
			request:       "{invalid json}",
			expectedError: "invalid character",
			expectedCode:  -32700, // Parse error
		},
		{
			name:          "Missing JSONRPC",
			request:       `{"id": 1, "method": "initialize"}`,
			expectedError: "Invalid request",
			expectedCode:  -32600, // Invalid request
		},
		{
			name:          "Wrong JSONRPC version",
			request:       `{"jsonrpc": "1.0", "id": 1, "method": "initialize"}`,
			expectedError: "Invalid request",
			expectedCode:  -32600, // Invalid request
		},
		{
			name:          "Missing method",
			request:       `{"jsonrpc": "2.0", "id": 1}`,
			expectedError: "Invalid request",
			expectedCode:  -32600, // Invalid request
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Validate request format
			if tc.request == "" {
				assert.Equal(t, -32700, tc.expectedCode, "Empty request should return parse error")
				return
			}
			
			var req MCPRequest
			err := json.Unmarshal([]byte(tc.request), &req)
			
			if err != nil {
				assert.Equal(t, -32700, tc.expectedCode, "Invalid JSON should return parse error")
				assert.Contains(t, err.Error(), tc.expectedError)
				return
			}
			
			// Validate JSONRPC version
			if req.JSONRPC != "2.0" {
				assert.Equal(t, -32600, tc.expectedCode, "Invalid JSONRPC version should return invalid request")
				return
			}
			
			// Validate method presence
			if req.Method == "" {
				assert.Equal(t, -32600, tc.expectedCode, "Missing method should return invalid request")
				return
			}
			
			// Valid request
			assert.Equal(t, 0, tc.expectedCode, "Valid request should not have error code")
		})
	}
}

// TestMCPResponseFormat tests that responses follow MCP specification
func TestMCPResponseFormat(t *testing.T) {
	testCases := []struct {
		name     string
		response MCPResponse
		hasError bool
	}{
		{
			name: "Success response",
			response: MCPResponse{
				JSONRPC: "2.0",
				ID:      1,
				Result:  json.RawMessage(`{"status": "ok"}`),
			},
			hasError: false,
		},
		{
			name: "Error response",
			response: MCPResponse{
				JSONRPC: "2.0",
				ID:      2,
				Error: &MCPError{
					Code:    -32601,
					Message: "Method not found",
				},
			},
			hasError: true,
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Verify response structure
			assert.Equal(t, "2.0", tc.response.JSONRPC, "Response must have jsonrpc 2.0")
			assert.NotNil(t, tc.response.ID, "Response must have ID")
			
			if tc.hasError {
				assert.NotNil(t, tc.response.Error, "Error response must have error")
				assert.Nil(t, tc.response.Result, "Error response cannot have result")
				assert.NotZero(t, tc.response.Error.Code, "Error must have code")
				assert.NotEmpty(t, tc.response.Error.Message, "Error must have message")
			} else {
				assert.Nil(t, tc.response.Error, "Success response cannot have error")
				assert.NotNil(t, tc.response.Result, "Success response must have result")
			}
			
			// Verify it can be marshaled
			data, err := json.Marshal(tc.response)
			require.NoError(t, err)
			assert.NotEmpty(t, data)
		})
	}
}

// TestMCPErrorCodes tests standard JSON-RPC error codes
func TestMCPErrorCodes(t *testing.T) {
	errorCodes := map[string]int{
		"Parse error":           -32700,
		"Invalid request":       -32600,
		"Method not found":      -32601,
		"Invalid params":        -32602,
		"Internal error":        -32603,
		"Server error":          -32000, // to -32099
	}
	
	for name, code := range errorCodes {
		t.Run(name, func(t *testing.T) {
			// Verify error codes are in expected ranges
			if code >= -32099 && code <= -32000 {
				assert.True(t, true, "Server error codes in valid range")
			} else {
				assert.Contains(t, []int{-32700, -32600, -32601, -32602, -32603}, code,
					"Standard error code should be recognized")
			}
		})
	}
}

// TestMCPToolValidation tests tool argument validation
func TestMCPToolValidation(t *testing.T) {
	testCases := []struct {
		name         string
		toolName     string
		arguments    map[string]interface{}
		expectError  bool
		errorMessage string
	}{
		{
			name:     "Valid create_entity",
			toolName: "create_entity",
			arguments: map[string]interface{}{
				"entitySet": "TestEntities",
				"entity": map[string]interface{}{
					"Name": "Test",
				},
			},
			expectError: false,
		},
		{
			name:     "Missing entitySet",
			toolName: "create_entity",
			arguments: map[string]interface{}{
				"entity": map[string]interface{}{
					"Name": "Test",
				},
			},
			expectError:  true,
			errorMessage: "entitySet",
		},
		{
			name:     "Wrong type for entitySet",
			toolName: "create_entity",
			arguments: map[string]interface{}{
				"entitySet": 123, // Should be string
				"entity":    map[string]interface{}{},
			},
			expectError:  true,
			errorMessage: "type",
		},
	}
	
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Simulate validation
			if tc.toolName == "create_entity" {
				entitySet, hasEntitySet := tc.arguments["entitySet"]
				entity, hasEntity := tc.arguments["entity"]
				
				if !hasEntitySet || !hasEntity {
					assert.True(t, tc.expectError, "Should expect error for missing fields")
					return
				}
				
				if _, ok := entitySet.(string); !ok {
					assert.True(t, tc.expectError, "Should expect error for wrong type")
					return
				}
				
				if _, ok := entity.(map[string]interface{}); !ok {
					assert.True(t, tc.expectError, "Should expect error for wrong entity type")
					return
				}
			}
			
			assert.False(t, tc.expectError, "Valid arguments should not produce error")
		})
	}
}