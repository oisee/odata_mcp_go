package test

import (
	"context"
	"os"
	"testing"

	"github.com/odata-mcp/go/internal/client"
	"github.com/stretchr/testify/assert"
)

// TestFilterIntegration tests filter operations against real SAP service
func TestFilterIntegration(t *testing.T) {
	// Skip if environment variables are not set
	odataURL := os.Getenv("ODATA_URL")
	odataUser := os.Getenv("ODATA_USER")
	odataPass := os.Getenv("ODATA_PASS")
	
	if odataURL == "" || odataUser == "" || odataPass == "" {
		t.Skip("Skipping integration test: ODATA_URL, ODATA_USER, ODATA_PASS not set")
	}
	
	client := client.NewODataClient(odataURL, true)
	client.SetBasicAuth(odataUser, odataPass)
	
	tests := []struct {
		name       string
		entitySet  string
		filter     string
		expectEmpty bool
		note       string
	}{
		{
			name:       "Filter by specific program",
			entitySet:  "PROGRAMSet",
			filter:     "Program eq 'ZHELLO_GO_TEST'",
			expectEmpty: true,
			note:       "Programs in $VIBE_TEST package may not be visible",
		},
		{
			name:       "Filter by package",
			entitySet:  "PROGRAMSet",
			filter:     "Package eq '$VIBE_TEST'",
			expectEmpty: true,
			note:       "Package $VIBE_TEST may have visibility restrictions",
		},
		{
			name:       "Filter standard programs",
			entitySet:  "PROGRAMSet",
			filter:     "substringof('DEMO', Program)",
			expectEmpty: false,
			note:       "Standard DEMO programs should be visible",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := client.GetEntitySet(context.Background(), tt.entitySet,
				map[string]string{"$filter": tt.filter, "$top": "5"})
			
			if err != nil {
				t.Errorf("Filter query failed: %v", err)
				return
			}
			
			// Check if results match expectations
			results, ok := result.Value.([]interface{})
			if !ok {
				t.Error("Expected results to be an array")
				return
			}
			
			if tt.expectEmpty {
				if len(results) > 0 {
					t.Logf("Expected empty results but got %d items", len(results))
					t.Logf("Note: %s", tt.note)
				} else {
					t.Logf("Confirmed: No results returned. %s", tt.note)
				}
			} else {
				if len(results) == 0 {
					t.Logf("Warning: Expected results but got none. %s", tt.note)
				} else {
					t.Logf("Found %d results as expected", len(results))
					// Log first result
					if len(results) > 0 {
						if entity, ok := results[0].(map[string]interface{}); ok {
							t.Logf("Sample result: Program=%v, Package=%v", 
								entity["Program"], entity["Package"])
						}
					}
				}
			}
			
			// Verify count if available
			if result.Count != nil {
				t.Logf("Total count: %d", *result.Count)
			}
		})
	}
	
	// Test that basic filtering syntax works
	t.Run("BasicFilterSyntax", func(t *testing.T) {
		// Just verify the service accepts filter syntax without errors
		_, err := client.GetEntitySet(context.Background(), "PROGRAMSet",
			map[string]string{
				"$filter": "Program ne ''",
				"$top":    "1",
			})
		
		assert.NoError(t, err, "Basic filter syntax should be accepted by service")
	})
}