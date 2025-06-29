package test

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/metadata"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestODataV4MetadataParsing tests parsing of OData v4 metadata
func TestODataV4MetadataParsing(t *testing.T) {
	// Sample OData v4 metadata
	v4Metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="NorthwindModel" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ProductID" />
        </Key>
        <Property Name="ProductID" Type="Edm.Int32" Nullable="false" />
        <Property Name="ProductName" Type="Edm.String" />
        <Property Name="UnitPrice" Type="Edm.Decimal" />
        <Property Name="Discontinued" Type="Edm.Boolean" Nullable="false" />
        <NavigationProperty Name="Category" Type="NorthwindModel.Category" Partner="Products" />
      </EntityType>
      <EntityType Name="Category">
        <Key>
          <PropertyRef Name="CategoryID" />
        </Key>
        <Property Name="CategoryID" Type="Edm.Int32" Nullable="false" />
        <Property Name="CategoryName" Type="Edm.String" />
        <NavigationProperty Name="Products" Type="Collection(NorthwindModel.Product)" Partner="Category" />
      </EntityType>
      <EntityContainer Name="NorthwindEntities">
        <EntitySet Name="Products" EntityType="NorthwindModel.Product">
          <NavigationPropertyBinding Path="Category" Target="Categories" />
        </EntitySet>
        <EntitySet Name="Categories" EntityType="NorthwindModel.Category">
          <NavigationPropertyBinding Path="Products" Target="Products" />
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	// Parse metadata
	meta, err := metadata.ParseMetadata([]byte(v4Metadata), "http://example.com/odata/")
	require.NoError(t, err)
	
	// Verify version
	assert.Equal(t, "4.0", meta.Version)
	
	// Verify entity types
	assert.Len(t, meta.EntityTypes, 2)
	
	productType := meta.EntityTypes["Product"]
	require.NotNil(t, productType)
	assert.Equal(t, "Product", productType.Name)
	assert.Len(t, productType.Properties, 4)
	assert.Len(t, productType.NavigationProps, 1)
	
	// Check navigation property has v4 attributes
	navProp := productType.NavigationProps[0]
	assert.Equal(t, "Category", navProp.Name)
	assert.Equal(t, "NorthwindModel.Category", navProp.Type)
	assert.Equal(t, "Products", navProp.Partner)
	
	// Verify entity sets
	assert.Len(t, meta.EntitySets, 2)
	productSet := meta.EntitySets["Products"]
	require.NotNil(t, productSet)
	assert.Equal(t, "Product", productSet.EntityType)
}

// TestODataV4ResponseHandling tests handling of OData v4 responses
func TestODataV4ResponseHandling(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/$metadata":
			// Return v4 metadata
			w.Header().Set("Content-Type", "application/xml")
			xml.NewEncoder(w).Encode(struct {
				XMLName xml.Name `xml:"edmx:Edmx"`
				Version string   `xml:"Version,attr"`
				DataServices struct {
					Schema struct {
						Namespace string `xml:"Namespace,attr"`
						EntityContainer struct {
							Name string `xml:"Name,attr"`
						} `xml:"EntityContainer"`
					} `xml:"Schema"`
				} `xml:"DataServices"`
			}{
				Version: "4.0",
				DataServices: struct {
					Schema struct {
						Namespace string `xml:"Namespace,attr"`
						EntityContainer struct {
							Name string `xml:"Name,attr"`
						} `xml:"EntityContainer"`
					} `xml:"Schema"`
				}{
					Schema: struct {
						Namespace string `xml:"Namespace,attr"`
						EntityContainer struct {
							Name string `xml:"Name,attr"`
						} `xml:"EntityContainer"`
					}{
						Namespace: "Test",
						EntityContainer: struct {
							Name string `xml:"Name,attr"`
						}{
							Name: "TestContainer",
						},
					},
				},
			})
			
		case "/Products":
			// Return v4 collection response
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"@odata.context": "http://" + r.Host + "/$metadata#Products",
				"@odata.count":   2,
				"value": []map[string]interface{}{
					{
						"@odata.id":   "http://" + r.Host + "/Products(1)",
						"ProductID":   1,
						"ProductName": "Chai",
						"UnitPrice":   18.00,
					},
					{
						"@odata.id":   "http://" + r.Host + "/Products(2)",
						"ProductID":   2,
						"ProductName": "Chang",
						"UnitPrice":   19.00,
					},
				},
				"@odata.nextLink": "http://" + r.Host + "/Products?$skip=2",
			})
			
		case "/Products(1)":
			// Return v4 single entity response
			w.Header().Set("Content-Type", "application/json;odata.metadata=minimal")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"@odata.context": "http://" + r.Host + "/$metadata#Products/$entity",
				"@odata.id":      "http://" + r.Host + "/Products(1)",
				"@odata.etag":    "W/\"1\"",
				"ProductID":      1,
				"ProductName":    "Chai",
				"UnitPrice":      18.00,
				"Discontinued":   false,
			})
			
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()
	
	// Create client
	odataClient := client.NewODataClient(server.URL, false)
	ctx := context.Background()
	
	// Fetch metadata to set v4 flag
	meta, err := odataClient.GetMetadata(ctx)
	require.NoError(t, err)
	assert.Equal(t, "4.0", meta.Version)
	
	// Test collection response
	resp, err := odataClient.GetEntitySet(ctx, "Products", nil)
	require.NoError(t, err)
	
	assert.NotNil(t, resp.Count)
	assert.Equal(t, int64(2), *resp.Count)
	assert.Contains(t, resp.Context, "/$metadata#Products")
	assert.Contains(t, resp.NextLink, "/Products?$skip=2")
	
	// Verify collection values
	values, ok := resp.Value.([]interface{})
	require.True(t, ok)
	assert.Len(t, values, 2)
	
	// Test single entity response
	resp, err = odataClient.GetEntity(ctx, "Products", map[string]interface{}{"ProductID": 1}, nil)
	require.NoError(t, err)
	
	assert.Contains(t, resp.Context, "/$metadata#Products/$entity")
	
	// Verify single entity
	entity, ok := resp.Value.(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, float64(1), entity["ProductID"])
	assert.Equal(t, "Chai", entity["ProductName"])
}

// TestODataV4vsV2Detection tests automatic detection of OData version
func TestODataV4vsV2Detection(t *testing.T) {
	tests := []struct {
		name     string
		metadata string
		expected string
		isV4     bool
	}{
		{
			name: "OData v2",
			metadata: `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="Test" xmlns="http://schemas.microsoft.com/ado/2009/11/edm">
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`,
			expected: "1.0",
			isV4:     false,
		},
		{
			name: "OData v4.0",
			metadata: `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="TestContainer" />
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`,
			expected: "4.0",
			isV4:     true,
		},
		{
			name: "OData v4.01",
			metadata: `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.01" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityContainer Name="TestContainer" />
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`,
			expected: "4.01",
			isV4:     true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.isV4, metadata.IsODataV4([]byte(tt.metadata)))
			
			meta, err := metadata.ParseMetadata([]byte(tt.metadata), "http://example.com/")
			require.NoError(t, err)
			assert.Equal(t, tt.expected, meta.Version)
		})
	}
}

// TestODataV4NewTypes tests handling of new v4 data types
func TestODataV4NewTypes(t *testing.T) {
	v4Metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="Test" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Event">
        <Key>
          <PropertyRef Name="ID" />
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false" />
        <Property Name="Title" Type="Edm.String" />
        <Property Name="EventDate" Type="Edm.Date" />
        <Property Name="StartTime" Type="Edm.TimeOfDay" />
        <Property Name="Duration" Type="Edm.Duration" />
        <Property Name="Timestamp" Type="Edm.DateTimeOffset" />
        <Property Name="Logo" Type="Edm.Stream" />
      </EntityType>
      <EntityContainer Name="Container">
        <EntitySet Name="Events" EntityType="Test.Event" />
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta, err := metadata.ParseMetadata([]byte(v4Metadata), "http://example.com/")
	require.NoError(t, err)
	
	eventType := meta.EntityTypes["Event"]
	require.NotNil(t, eventType)
	
	// Check new v4 types
	typeMap := make(map[string]string)
	for _, prop := range eventType.Properties {
		typeMap[prop.Name] = prop.Type
	}
	
	assert.Equal(t, "Edm.Date", typeMap["EventDate"])
	assert.Equal(t, "Edm.TimeOfDay", typeMap["StartTime"])
	assert.Equal(t, "Edm.Duration", typeMap["Duration"])
	assert.Equal(t, "Edm.DateTimeOffset", typeMap["Timestamp"])
	assert.Equal(t, "Edm.Stream", typeMap["Logo"])
}