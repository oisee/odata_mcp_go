package test

import (
	"testing"

	"github.com/zmcp/odata-mcp/internal/metadata"
)

func TestSAPLabelParsingV2(t *testing.T) {
	xmlData := `<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx Version="1.0"
  xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx"
  xmlns:m="http://schemas.microsoft.com/ado/2007/08/dataservices/metadata"
  xmlns:sap="http://www.sap.com/Protocols/SAPData">
  <edmx:DataServices m:DataServiceVersion="2.0">
    <Schema Namespace="Test" xmlns="http://schemas.microsoft.com/ado/2008/09/edm">
      <EntityType Name="Foo">
        <Key>
          <PropertyRef Name="ID"/>
        </Key>
        <Property Name="ID" Type="Edm.String" Nullable="false" sap:label="Identifier"/>
        <Property Name="Bar" Type="Edm.String" sap:label="Bar Label"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="Foos" EntityType="Test.Foo"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`

	meta, err := metadata.ParseMetadata([]byte(xmlData), "http://example.com/")
	if err != nil {
		t.Fatalf("ParseMetadata failed: %v", err)
	}

	et := meta.EntityTypes["Foo"]
	if et == nil {
		t.Fatalf("Entity type Foo not found")
	}

	var barLabel *string
	for _, prop := range et.Properties {
		if prop.Name == "Bar" {
			barLabel = prop.SAPLabel
			break
		}
	}

	if barLabel == nil || *barLabel != "Bar Label" {
		t.Fatalf("Expected SAP label 'Bar Label', got %v", barLabel)
	}
}
