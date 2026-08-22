// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package metadata

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseMetadata_SingleSchema tests parsing of standard single-schema metadata (Northwind style)
func TestParseMetadata_SingleSchema(t *testing.T) {
	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="NorthwindModel" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ProductID"/>
        </Key>
        <Property Name="ProductID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="ProductName" Type="Edm.String" Nullable="false" MaxLength="40"/>
        <Property Name="UnitPrice" Type="Edm.Decimal" Precision="19" Scale="4"/>
      </EntityType>
      <EntityType Name="Category">
        <Key>
          <PropertyRef Name="CategoryID"/>
        </Key>
        <Property Name="CategoryID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="CategoryName" Type="Edm.String" Nullable="false" MaxLength="15"/>
      </EntityType>
      <EntityContainer Name="NorthwindEntities" m:IsDefaultEntityContainer="true">
        <EntitySet Name="Products" EntityType="NorthwindModel.Product"/>
        <EntitySet Name="Categories" EntityType="NorthwindModel.Category"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	result, err := ParseMetadata([]byte(metadata), "http://example.com/odata")
	require.NoError(t, err)

	// Should have 2 entity types (stored with qualified names)
	assert.Len(t, result.EntityTypes, 2)
	assert.Contains(t, result.EntityTypes, "NorthwindModel.Product")
	assert.Contains(t, result.EntityTypes, "NorthwindModel.Category")

	// Should have 2 entity sets
	assert.Len(t, result.EntitySets, 2)
	assert.Contains(t, result.EntitySets, "Products")
	assert.Contains(t, result.EntitySets, "Categories")

	// Entity set keeps original reference (qualified name from metadata)
	assert.Equal(t, "NorthwindModel.Product", result.EntitySets["Products"].EntityType)
	assert.Equal(t, "NorthwindModel.Category", result.EntitySets["Categories"].EntityType)

	// Namespace should be captured
	assert.Equal(t, "NorthwindModel", result.SchemaNamespace)
	assert.Equal(t, "NorthwindEntities", result.ContainerName)
}

// TestParseMetadata_MultiSchema tests parsing of SAP-style multi-schema metadata
// SAP OData services typically split EntityTypes and EntityContainer across multiple schemas
func TestParseMetadata_MultiSchema(t *testing.T) {
	// This is the SAP pattern: EntityTypes in one schema, EntityContainer in another
	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="API_PRODUCT_SRV_Entities" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="A_Product">
        <Key>
          <PropertyRef Name="Product"/>
        </Key>
        <Property Name="Product" Type="Edm.String" Nullable="false" MaxLength="40"/>
        <Property Name="ProductType" Type="Edm.String" MaxLength="4"/>
        <Property Name="BaseUnit" Type="Edm.String" MaxLength="3"/>
      </EntityType>
      <EntityType Name="A_ProductDescription">
        <Key>
          <PropertyRef Name="Product"/>
          <PropertyRef Name="Language"/>
        </Key>
        <Property Name="Product" Type="Edm.String" Nullable="false" MaxLength="40"/>
        <Property Name="Language" Type="Edm.String" Nullable="false" MaxLength="2"/>
        <Property Name="ProductDescription" Type="Edm.String" MaxLength="40"/>
      </EntityType>
    </Schema>
    <Schema Namespace="API_PRODUCT_SRV" xmlns="http://schemas.microsoft.com/ado/2008/09/edm"
            xmlns:sap="http://www.sap.com/Protocols/SAPData">
      <EntityContainer Name="API_PRODUCT_SRV" m:IsDefaultEntityContainer="true">
        <EntitySet Name="A_Product" EntityType="API_PRODUCT_SRV_Entities.A_Product"
                   sap:creatable="true" sap:updatable="true" sap:deletable="false"/>
        <EntitySet Name="A_ProductDescription" EntityType="API_PRODUCT_SRV_Entities.A_ProductDescription"
                   sap:creatable="true" sap:updatable="true" sap:deletable="true"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	result, err := ParseMetadata([]byte(metadata), "http://sap-host:50000/sap/opu/odata/sap/API_PRODUCT_SRV")
	require.NoError(t, err)

	// Should have 2 entity types (stored with qualified names from first schema)
	assert.Len(t, result.EntityTypes, 2, "Should parse entity types from first schema")
	assert.Contains(t, result.EntityTypes, "API_PRODUCT_SRV_Entities.A_Product", "Should have A_Product entity type")
	assert.Contains(t, result.EntityTypes, "API_PRODUCT_SRV_Entities.A_ProductDescription", "Should have A_ProductDescription entity type")

	// Should have 2 entity sets (from second schema)
	assert.Len(t, result.EntitySets, 2, "Should parse entity sets from second schema")
	assert.Contains(t, result.EntitySets, "A_Product", "Should have A_Product entity set")
	assert.Contains(t, result.EntitySets, "A_ProductDescription", "Should have A_ProductDescription entity set")

	// Entity sets keep original reference (qualified name from metadata)
	productSet := result.EntitySets["A_Product"]
	assert.Equal(t, "API_PRODUCT_SRV_Entities.A_Product", productSet.EntityType, "EntityType should be qualified name")

	// SAP attributes should be parsed
	assert.True(t, productSet.Creatable)
	assert.True(t, productSet.Updatable)
	assert.False(t, productSet.Deletable)

	// Container name from second schema
	assert.Equal(t, "API_PRODUCT_SRV", result.ContainerName)
}

// TestParseMetadata_MultiSchema_FunctionImports tests function imports in multi-schema
func TestParseMetadata_MultiSchema_FunctionImports(t *testing.T) {
	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="GWSAMPLE_BASIC_Entities" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="BusinessPartner">
        <Key>
          <PropertyRef Name="BusinessPartnerID"/>
        </Key>
        <Property Name="BusinessPartnerID" Type="Edm.String" Nullable="false" MaxLength="10"/>
        <Property Name="CompanyName" Type="Edm.String" MaxLength="80"/>
      </EntityType>
    </Schema>
    <Schema Namespace="GWSAMPLE_BASIC" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityContainer Name="GWSAMPLE_BASIC" m:IsDefaultEntityContainer="true">
        <EntitySet Name="BusinessPartnerSet" EntityType="GWSAMPLE_BASIC_Entities.BusinessPartner"/>
        <FunctionImport Name="RegenerateAllData" ReturnType="Edm.String" m:HttpMethod="POST">
          <Parameter Name="NoOfSalesOrders" Type="Edm.Int32" Mode="In"/>
        </FunctionImport>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	result, err := ParseMetadata([]byte(metadata), "http://sap-host:50000/sap/opu/odata/IWBEP/GWSAMPLE_BASIC")
	require.NoError(t, err)

	// Should have entity types from first schema (qualified name)
	assert.Contains(t, result.EntityTypes, "GWSAMPLE_BASIC_Entities.BusinessPartner")

	// Should have entity sets from second schema
	assert.Contains(t, result.EntitySets, "BusinessPartnerSet")

	// Should have function imports from second schema
	assert.Len(t, result.FunctionImports, 1)
	assert.Contains(t, result.FunctionImports, "RegenerateAllData")

	fn := result.FunctionImports["RegenerateAllData"]
	assert.Equal(t, "POST", fn.HTTPMethod)
	assert.Len(t, fn.Parameters, 1)
	assert.Equal(t, "NoOfSalesOrders", fn.Parameters[0].Name)
}

// TestParseMetadata_ThreeSchemas tests an edge case with 3 schemas
func TestParseMetadata_ThreeSchemas(t *testing.T) {
	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0" xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx">
  <edmx:DataServices m:DataServiceVersion="2.0" xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata">
    <Schema Namespace="CommonTypes" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Address">
        <Key>
          <PropertyRef Name="AddressID"/>
        </Key>
        <Property Name="AddressID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Street" Type="Edm.String" MaxLength="100"/>
      </EntityType>
    </Schema>
    <Schema Namespace="BusinessTypes" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Customer">
        <Key>
          <PropertyRef Name="CustomerID"/>
        </Key>
        <Property Name="CustomerID" Type="Edm.String" Nullable="false" MaxLength="10"/>
        <Property Name="Name" Type="Edm.String" MaxLength="80"/>
      </EntityType>
    </Schema>
    <Schema Namespace="ServiceContainer" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityContainer Name="MainContainer" m:IsDefaultEntityContainer="true">
        <EntitySet Name="Addresses" EntityType="CommonTypes.Address"/>
        <EntitySet Name="Customers" EntityType="BusinessTypes.Customer"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	result, err := ParseMetadata([]byte(metadata), "http://example.com/odata")
	require.NoError(t, err)

	// Should have entity types from both first and second schemas (qualified names)
	assert.Len(t, result.EntityTypes, 2)
	assert.Contains(t, result.EntityTypes, "CommonTypes.Address")
	assert.Contains(t, result.EntityTypes, "BusinessTypes.Customer")

	// Should have entity sets from third schema
	assert.Len(t, result.EntitySets, 2)
	assert.Contains(t, result.EntitySets, "Addresses")
	assert.Contains(t, result.EntitySets, "Customers")

	// Entity sets keep original reference (qualified name from metadata)
	assert.Equal(t, "CommonTypes.Address", result.EntitySets["Addresses"].EntityType)
	assert.Equal(t, "BusinessTypes.Customer", result.EntitySets["Customers"].EntityType)
}

// TestParseMetadata_V4_SearchRestrictions tests parsing of OData v4 SearchRestrictions annotation
func TestParseMetadata_V4_SearchRestrictions(t *testing.T) {
	metadata := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="4.0" xmlns:edmx="http://docs.oasis-open.org/odata/ns/edmx">
  <edmx:DataServices>
    <Schema Namespace="TestService" xmlns="http://docs.oasis-open.org/odata/ns/edm">
      <EntityType Name="Product">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
      </EntityType>
      <EntityType Name="Category">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.Int32" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
      </EntityType>
      <EntityContainer Name="Container">
        <EntitySet Name="Products" EntityType="TestService.Product">
          <Annotation Term="Org.OData.Capabilities.V1.SearchRestrictions">
            <Record>
              <PropertyValue Property="Searchable" Bool="true"/>
            </Record>
          </Annotation>
        </EntitySet>
        <EntitySet Name="Categories" EntityType="TestService.Category">
          <Annotation Term="Org.OData.Capabilities.V1.SearchRestrictions">
            <Record>
              <PropertyValue Property="Searchable" Bool="false"/>
            </Record>
          </Annotation>
        </EntitySet>
        <EntitySet Name="Orders" EntityType="TestService.Product">
        </EntitySet>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	result, err := ParseMetadata([]byte(metadata), "http://example.com/odata")
	require.NoError(t, err)

	// Products should be searchable (annotation says true)
	assert.True(t, result.EntitySets["Products"].Searchable, "Products should be searchable")

	// Categories should not be searchable (annotation says false)
	assert.False(t, result.EntitySets["Categories"].Searchable, "Categories should not be searchable")

	// Orders should not be searchable (no annotation, defaults to false)
	assert.False(t, result.EntitySets["Orders"].Searchable, "Orders should default to not searchable")
}
