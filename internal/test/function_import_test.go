package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/odata-mcp/go/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestFunctionImportURIEncoding tests proper URI encoding for function imports
func TestFunctionImportURIEncoding(t *testing.T) {
	tests := []struct {
		name           string
		functionName   string
		parameters     map[string]interface{}
		expectedPath   string
		expectedParams string
	}{
		{
			name:           "String parameter",
			functionName:   "ACTIVATE_PROGRAM",
			parameters:     map[string]interface{}{"Program": "ZHELLO_GO_TEST"},
			expectedPath:   "/ACTIVATE_PROGRAM",
			expectedParams: "Program='ZHELLO_GO_TEST'",
		},
		{
			name:           "String with spaces",
			functionName:   "SEARCH_PROGRAM",
			parameters:     map[string]interface{}{"Query": "hello world"},
			expectedPath:   "/SEARCH_PROGRAM",
			expectedParams: "Query='hello+world'",
		},
		{
			name:           "Multiple parameters",
			functionName:   "CREATE_OBJECT",
			parameters:     map[string]interface{}{"Name": "Test Object", "Type": "Report", "Version": 1},
			expectedPath:   "/CREATE_OBJECT",
			expectedParams: "Name='Test+Object'&Type='Report'&Version=1",
		},
		{
			name:           "Boolean parameter",
			functionName:   "SET_ACTIVE",
			parameters:     map[string]interface{}{"Program": "ZTEST", "Active": true},
			expectedPath:   "/SET_ACTIVE",
			expectedParams: "Program='ZTEST'&Active=true",
		},
		{
			name:           "Special characters",
			functionName:   "UPDATE_PROGRAM",
			parameters:     map[string]interface{}{"Program": "Z$TEST#01", "Description": "Test & Demo"},
			expectedPath:   "/UPDATE_PROGRAM",
			expectedParams: "Program='Z%24TEST%2301'&Description='Test+%26+Demo'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedURL string
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedURL = r.URL.String()
				
				// Return a success response
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"Success": true,
					},
				})
			}))
			defer server.Close()

			client := client.NewODataClient(server.URL, false)
			client.SetBasicAuth("test", "test")

			// Call the function
			_, err := client.CallFunction(context.Background(), tt.functionName, tt.parameters, "GET")
			require.NoError(t, err)

			// Verify the URL construction
			assert.Equal(t, tt.expectedPath, strings.Split(capturedURL, "?")[0])
			
			if tt.expectedParams != "" {
				// Extract query string
				parts := strings.Split(capturedURL, "?")
				require.Len(t, parts, 2, "Expected query parameters")
				
				// Check that all expected parameters are present
				// Note: Order may vary, so we check each parameter individually
				queryString := parts[1]
				for _, param := range strings.Split(tt.expectedParams, "&") {
					assert.Contains(t, queryString, param, "Missing parameter: %s", param)
				}
			}
		})
	}
}

// TestActivateProgramFunction tests the specific ACTIVATE_PROGRAM function
func TestActivateProgramFunction(t *testing.T) {
	programName := "ZHELLO_GO_TEST"
	activateCalled := false
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "ACTIVATE_PROGRAM") {
			activateCalled = true
			
			// Verify the program parameter is properly formatted
			programParam := r.URL.RawQuery
			
			// The parameter should be in the format: Program='ZHELLO_GO_TEST'
			assert.Contains(t, programParam, "Program='ZHELLO_GO_TEST'")
			
			// Return activation result
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"ACTIVATE_PROGRAM": map[string]interface{}{
						"Log": "Ok",
					},
				},
			})
			return
		}
		
		// Default response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
	}))
	defer server.Close()

	client := client.NewODataClient(server.URL, false)
	client.SetBasicAuth("test", "test")

	// Call ACTIVATE_PROGRAM
	result, err := client.CallFunction(context.Background(), "ACTIVATE_PROGRAM", 
		map[string]interface{}{"Program": programName}, "GET")
	
	require.NoError(t, err)
	assert.True(t, activateCalled, "ACTIVATE_PROGRAM should have been called")
	assert.NotNil(t, result)
}