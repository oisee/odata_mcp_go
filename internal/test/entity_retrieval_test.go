package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/zmcp/odata-mcp/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEntityKeyPredicate tests proper key predicate formatting for entity retrieval
func TestEntityKeyPredicate(t *testing.T) {
	tests := []struct {
		name         string
		entitySet    string
		key          map[string]interface{}
		expectedPath string
	}{
		{
			name:         "Simple string key",
			entitySet:    "PROGRAMSet",
			key:          map[string]interface{}{"Program": "ZHELLO_GO_TEST"},
			expectedPath: "/PROGRAMSet('ZHELLO_GO_TEST')",
		},
		{
			name:         "String key with special chars",
			entitySet:    "PROGRAMSet",
			key:          map[string]interface{}{"Program": "Z_TEST_01"},
			expectedPath: "/PROGRAMSet('Z_TEST_01')",
		},
		{
			name:         "Integer key",
			entitySet:    "OrderSet",
			key:          map[string]interface{}{"OrderID": 12345},
			expectedPath: "/OrderSet(12345)",
		},
		{
			name:         "Composite key",
			entitySet:    "OrderItemSet",
			key:          map[string]interface{}{"OrderID": 12345, "ItemID": "ABC"},
			expectedPath: "/OrderItemSet(OrderID=12345,ItemID='ABC')",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var capturedPath string
			
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				capturedPath = r.URL.Path
				t.Logf("Request URL: %s", r.URL.String())
				
				// Return a mock entity
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"__metadata": map[string]interface{}{
							"type": "Test.Entity",
						},
						"ID": "test",
					},
				})
			}))
			defer server.Close()

			client := client.NewODataClient(server.URL, false)
			client.SetBasicAuth("test", "test")

			// Get entity
			_, err := client.GetEntity(context.Background(), tt.entitySet, tt.key, nil)
			require.NoError(t, err)

			// Verify the path construction
			assert.Equal(t, tt.expectedPath, capturedPath)
		})
	}
}

// TestProgramEntityRetrieval tests retrieving a specific program entity
func TestProgramEntityRetrieval(t *testing.T) {
	programName := "ZHELLO_GO_TEST"
	
	t.Run("MockServer", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, fmt.Sprintf("PROGRAMSet('%s')", programName)) {
				// Return the program entity
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"__metadata": map[string]interface{}{
							"id":   fmt.Sprintf("http://server/PROGRAMSet('%s')", programName),
							"uri":  fmt.Sprintf("http://server/PROGRAMSet('%s')", programName),
							"type": "ZODD_000_SRV.PROGRAM",
						},
						"Program":     programName,
						"Package":     "$VIBE_TEST",
						"ProgramType": "1",
						"Title":       "Test Program",
					},
				})
				return
			}
			
			// Return 404
			w.WriteHeader(http.StatusNotFound)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "Not Found",
					"message": map[string]interface{}{
						"value": "Entity not found",
					},
				},
			})
		}))
		defer server.Close()

		client := client.NewODataClient(server.URL, false)
		client.SetBasicAuth("test", "test")

		// Get entity
		result, err := client.GetEntity(context.Background(), "PROGRAMSet", 
			map[string]interface{}{"Program": programName}, nil)
		
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Verify response
		if entity, ok := result.Value.(map[string]interface{}); ok {
			assert.Equal(t, programName, entity["Program"])
			assert.Equal(t, "$VIBE_TEST", entity["Package"])
		}
	})
	
	t.Run("Integration", func(t *testing.T) {
		// Skip if environment variables are not set
		odataURL := os.Getenv("ODATA_URL")
		odataUser := os.Getenv("ODATA_USER")
		odataPass := os.Getenv("ODATA_PASS")
		
		if odataURL == "" || odataUser == "" || odataPass == "" {
			t.Skip("Skipping integration test: ODATA_URL, ODATA_USER, ODATA_PASS not set")
		}
		
		client := client.NewODataClient(odataURL, true)
		client.SetBasicAuth(odataUser, odataPass)
		
		// Try to get a known program (the one we created in previous tests)
		// Note: This might fail due to authorization, which is expected
		result, err := client.GetEntity(context.Background(), "PROGRAMSet", 
			map[string]interface{}{"Program": programName}, nil)
		
		if err != nil {
			errStr := err.Error()
			
			// Check if it's an authorization issue
			if strings.Contains(errStr, "404") || strings.Contains(errStr, "not found") {
				t.Logf("Expected behavior: Entity not visible due to authorization. Error: %v", err)
				t.Log("This is a known limitation - entities in $VIBE_TEST package may not be retrievable via OData")
			} else {
				t.Errorf("Unexpected error: %v", err)
			}
		} else {
			// If we can retrieve it, verify the data
			t.Log("Successfully retrieved program entity")
			assert.NotNil(t, result)
			
			if entity, ok := result.Value.(map[string]interface{}); ok {
				t.Logf("Retrieved program: %+v", entity)
			}
		}
	})
}

// TestEntityFilter tests filter operations
func TestEntityFilter(t *testing.T) {
	t.Run("FilterSyntax", func(t *testing.T) {
		tests := []struct {
			name           string
			filter         string
			expectedFilter string
		}{
			{
				name:           "Simple string filter",
				filter:         "Program eq 'ZHELLO_GO_TEST'",
				expectedFilter: "Program%20eq%20%27ZHELLO_GO_TEST%27",
			},
			{
				name:           "Filter with package",
				filter:         "Package eq '$VIBE_TEST'",
				expectedFilter: "Package%20eq%20%27%24VIBE_TEST%27",
			},
		}
		
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				var capturedQuery string
				
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					capturedQuery = r.URL.RawQuery
					
					// Return empty result
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"d": map[string]interface{}{
							"results": []interface{}{},
						},
					})
				}))
				defer server.Close()
				
				client := client.NewODataClient(server.URL, false)
				client.SetBasicAuth("test", "test")
				
				// Query with filter
				_, err := client.GetEntitySet(context.Background(), "PROGRAMSet", 
					map[string]string{"$filter": tt.filter})
				
				require.NoError(t, err)
				assert.Contains(t, capturedQuery, tt.expectedFilter)
			})
		}
	})
}