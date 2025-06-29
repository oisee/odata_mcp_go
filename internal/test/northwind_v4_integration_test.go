package test

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/odata-mcp/go/internal/bridge"
	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/config"
	"github.com/odata-mcp/go/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNorthwindV4Integration tests against the real Northwind OData v4 service
func TestNorthwindV4Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	
	// Skip if offline
	if os.Getenv("OFFLINE_TESTS") == "true" {
		t.Skip("Skipping online integration test")
	}
	
	serviceURL := "https://services.odata.org/V4/Northwind/Northwind.svc/"
	
	t.Run("DirectClient", func(t *testing.T) {
		// Create client
		odataClient := client.NewODataClient(serviceURL, true)
		ctx := context.Background()
		
		// Test metadata fetch
		t.Run("Metadata", func(t *testing.T) {
			meta, err := odataClient.GetMetadata(ctx)
			require.NoError(t, err)
			
			// Verify it's v4
			assert.Equal(t, "4.0", meta.Version)
			
			// Check some expected entity sets
			assert.Contains(t, meta.EntitySets, "Products")
			assert.Contains(t, meta.EntitySets, "Categories")
			assert.Contains(t, meta.EntitySets, "Orders")
			assert.Contains(t, meta.EntitySets, "Customers")
			
			// Verify v4 specific features
			productType := meta.EntityTypes["Product"]
			require.NotNil(t, productType)
			
			// Check navigation properties have v4 attributes
			var categoryNav *models.NavigationProperty
			for _, nav := range productType.NavigationProps {
				if nav.Name == "Category" {
					categoryNav = nav
					break
				}
			}
			require.NotNil(t, categoryNav, "Category navigation property not found")
			assert.NotEmpty(t, categoryNav.Type, "Navigation property should have Type in v4")
			assert.NotEmpty(t, categoryNav.Partner, "Navigation property should have Partner in v4")
		})
		
		// Test entity retrieval
		t.Run("GetProducts", func(t *testing.T) {
			resp, err := odataClient.GetEntitySet(ctx, "Products", map[string]string{
				"$top": "5",
			})
			require.NoError(t, err)
			
			// Check v4 response format
			assert.NotEmpty(t, resp.Context, "Should have @odata.context")
			assert.Contains(t, resp.Context, "$metadata#Products")
			
			// Check data
			values, ok := resp.Value.([]interface{})
			require.True(t, ok, "Value should be an array")
			assert.Len(t, values, 5, "Should return 5 products")
			
			// Check first product has v4 annotations
			if len(values) > 0 {
				product, ok := values[0].(map[string]interface{})
				require.True(t, ok)
				// Note: Minimal metadata mode might not include @odata.id
				assert.NotNil(t, product["ProductID"])
				assert.NotNil(t, product["ProductName"])
			}
		})
		
		// Test single entity
		t.Run("GetSingleProduct", func(t *testing.T) {
			resp, err := odataClient.GetEntity(ctx, "Products", map[string]interface{}{
				"ProductID": 1,
			}, nil)
			require.NoError(t, err)
			
			// Check v4 response format
			assert.NotEmpty(t, resp.Context, "Should have @odata.context")
			
			// Check entity
			entity, ok := resp.Value.(map[string]interface{})
			require.True(t, ok, "Value should be a map")
			assert.Equal(t, float64(1), entity["ProductID"])
			assert.NotNil(t, entity["ProductName"])
		})
		
		// Test v4 specific query features
		t.Run("V4QueryFeatures", func(t *testing.T) {
			// Test $count
			resp, err := odataClient.GetEntitySet(ctx, "Products", map[string]string{
				"$count": "true",
				"$top":   "10",
			})
			require.NoError(t, err)
			
			assert.NotNil(t, resp.Count, "Should include count with $count=true")
			assert.Greater(t, *resp.Count, int64(10), "Total count should be greater than page size")
			
			// Test $filter with v4 functions
			resp, err = odataClient.GetEntitySet(ctx, "Products", map[string]string{
				"$filter": "contains(ProductName, 'Chef')",
				"$select": "ProductID,ProductName",
			})
			require.NoError(t, err)
			
			values, ok := resp.Value.([]interface{})
			require.True(t, ok)
			assert.Greater(t, len(values), 0, "Should find products containing 'Chef'")
			
			// Verify only selected fields are returned
			if len(values) > 0 {
				product := values[0].(map[string]interface{})
				assert.Contains(t, product, "ProductID")
				assert.Contains(t, product, "ProductName")
				// Other fields might still be included depending on service implementation
			}
		})
		
		// Test navigation properties (v4 style)
		t.Run("NavigationProperties", func(t *testing.T) {
			// Expand category
			resp, err := odataClient.GetEntity(ctx, "Products", map[string]interface{}{
				"ProductID": 1,
			}, map[string]string{
				"$expand": "Category",
			})
			require.NoError(t, err)
			
			entity, ok := resp.Value.(map[string]interface{})
			require.True(t, ok)
			
			// Check if category is expanded
			if category, ok := entity["Category"]; ok && category != nil {
				categoryMap, ok := category.(map[string]interface{})
				require.True(t, ok, "Category should be a map")
				assert.NotNil(t, categoryMap["CategoryName"])
			}
		})
	})
	
	t.Run("MCPBridge", func(t *testing.T) {
		// Create config for v4 service
		cfg := &config.Config{
			ServiceURL: serviceURL,
			Verbose:    false,
		}
		
		// Create bridge
		bridge, err := bridge.NewODataMCPBridge(cfg)
		require.NoError(t, err)
		require.NotNil(t, bridge)
		
		// Get trace info to verify tools
		traceInfo, err := bridge.GetTraceInfo()
		require.NoError(t, err)
		
		// Verify tools were generated
		assert.Greater(t, len(traceInfo.RegisteredTools), 10, "Should generate multiple tools")
		
		// Check for expected tools
		toolNames := make(map[string]bool)
		for _, tool := range traceInfo.RegisteredTools {
			toolNames[tool.Name] = true
		}
		
		// Debug: print all tool names
		t.Logf("Generated tools: %v", toolNames)
		
		// Check entity tools - tools may have postfix by default
		hasFilterProducts := false
		hasGetProducts := false
		hasFilterCategories := false
		
		for name := range toolNames {
			if strings.Contains(name, "filter") && strings.Contains(name, "Products") {
				hasFilterProducts = true
			}
			if strings.Contains(name, "get") && strings.Contains(name, "Products") {
				hasGetProducts = true
			}
			if strings.Contains(name, "filter") && strings.Contains(name, "Categories") {
				hasFilterCategories = true
			}
		}
		
		assert.True(t, hasFilterProducts, "Should have a filter Products tool")
		assert.True(t, hasGetProducts, "Should have a get Products tool") 
		assert.True(t, hasFilterCategories, "Should have a filter Categories tool")
		
		// The bridge would be used with MCP server for actual tool execution
		// Here we just verify the tools were generated correctly
	})
}