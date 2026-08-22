package metadata

import (
	"encoding/xml"
	"fmt"
	"slices"
	"time"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/models"
)

// EDMX represents the root EDMX document
type EDMX struct {
	XMLName      xml.Name     `xml:"Edmx"`
	Version      string       `xml:"Version,attr"`
	DataServices DataServices `xml:"DataServices"`
}

// DataServices contains the schemas (SAP services often have multiple schemas)
type DataServices struct {
	XMLName xml.Name `xml:"DataServices"`
	Schemas []Schema `xml:"Schema"`
}

// Schema contains entity types, entity sets, and function imports
type Schema struct {
	XMLName         xml.Name         `xml:"Schema"`
	Namespace       string           `xml:"Namespace,attr"`
	EntityTypes     []EntityType     `xml:"EntityType"`
	EntityContainer EntityContainer  `xml:"EntityContainer"`
	FunctionImports []FunctionImport `xml:"FunctionImport"`
}

// EntityType represents an OData entity type
type EntityType struct {
	XMLName              xml.Name             `xml:"EntityType"`
	Name                 string               `xml:"Name,attr"`
	Key                  Key                  `xml:"Key"`
	Properties           []Property           `xml:"Property"`
	NavigationProperties []NavigationProperty `xml:"NavigationProperty"`
}

// Key contains key properties
type Key struct {
	XMLName      xml.Name      `xml:"Key"`
	PropertyRefs []PropertyRef `xml:"PropertyRef"`
}

// PropertyRef references a key property
type PropertyRef struct {
	XMLName xml.Name `xml:"PropertyRef"`
	Name    string   `xml:"Name,attr"`
}

// Property represents an entity property
type Property struct {
	XMLName   xml.Name `xml:"Property"`
	Name      string   `xml:"Name,attr"`
	Type      string   `xml:"Type,attr"`
	Nullable  string   `xml:"Nullable,attr"`
	MaxLength string   `xml:"MaxLength,attr"`
	Precision string   `xml:"Precision,attr"`
	Scale     string   `xml:"Scale,attr"`
}

// NavigationProperty represents a navigation property
type NavigationProperty struct {
	XMLName      xml.Name `xml:"NavigationProperty"`
	Name         string   `xml:"Name,attr"`
	Relationship string   `xml:"Relationship,attr"`
	ToRole       string   `xml:"ToRole,attr"`
	FromRole     string   `xml:"FromRole,attr"`
}

// EntityContainer contains entity sets and function imports
type EntityContainer struct {
	XMLName         xml.Name         `xml:"EntityContainer"`
	Name            string           `xml:"Name,attr"`
	EntitySets      []EntitySet      `xml:"EntitySet"`
	FunctionImports []FunctionImport `xml:"FunctionImport"`
}

// EntitySet represents an OData entity set
type EntitySet struct {
	XMLName    xml.Name `xml:"EntitySet"`
	Name       string   `xml:"Name,attr"`
	EntityType string   `xml:"EntityType,attr"`
	// SAP-specific attributes (use full namespace URL for Go xml parser)
	Creatable  string `xml:"http://www.sap.com/Protocols/SAPData creatable,attr"`
	Updatable  string `xml:"http://www.sap.com/Protocols/SAPData updatable,attr"`
	Deletable  string `xml:"http://www.sap.com/Protocols/SAPData deletable,attr"`
	Searchable string `xml:"http://www.sap.com/Protocols/SAPData searchable,attr"`
	Pageable   string `xml:"http://www.sap.com/Protocols/SAPData pageable,attr"`
}

// FunctionImport represents an OData function import
type FunctionImport struct {
	XMLName    xml.Name    `xml:"FunctionImport"`
	Name       string      `xml:"Name,attr"`
	ReturnType string      `xml:"ReturnType,attr"`
	HTTPMethod string      `xml:"http://schemas.microsoft.com/ado/2007/08/dataservices/metadata HttpMethod,attr"`
	Parameters []Parameter `xml:"Parameter"`
}

// Parameter represents a function parameter
type Parameter struct {
	XMLName  xml.Name `xml:"Parameter"`
	Name     string   `xml:"Name,attr"`
	Type     string   `xml:"Type,attr"`
	Mode     string   `xml:"Mode,attr"`
	Nullable string   `xml:"Nullable,attr"`
}

// ParseMetadata parses OData metadata XML and returns structured metadata
// It automatically detects whether the metadata is v2 or v4 and uses the appropriate parser
func ParseMetadata(data []byte, serviceRoot string) (*models.ODataMetadata, error) {
	// Check if it's OData v4
	if IsODataV4(data) {
		return ParseMetadataV4(data, serviceRoot)
	}

	// Parse as OData v2
	var edmx EDMX
	if err := xml.Unmarshal(data, &edmx); err != nil {
		return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
	}

	metadata := &models.ODataMetadata{
		ServiceRoot:     serviceRoot,
		EntityTypes:     make(map[string]*models.EntityType),
		EntitySets:      make(map[string]*models.EntitySet),
		FunctionImports: make(map[string]*models.FunctionImport),
		Version:         edmx.Version,
		ParsedAt:        time.Now(),
	}

	// Iterate over all schemas (SAP services often split EntityTypes and EntityContainer across schemas)
	for _, schema := range edmx.DataServices.Schemas {
		// Capture namespace from first schema that has one
		if metadata.SchemaNamespace == "" && schema.Namespace != "" {
			metadata.SchemaNamespace = schema.Namespace
		}

		// Parse entity types from this schema
		for _, et := range schema.EntityTypes {
			entityType := parseEntityType(et)
			// Store with qualified name (namespace.Name) to handle collisions
			if schema.Namespace != "" {
				metadata.EntityTypes[schema.Namespace+"."+et.Name] = entityType
			} else {
				metadata.EntityTypes[et.Name] = entityType
			}
		}

		// Parse entity container if present in this schema
		if schema.EntityContainer.Name != "" {
			metadata.ContainerName = schema.EntityContainer.Name

			// Parse entity sets
			for _, es := range schema.EntityContainer.EntitySets {
				entitySet := parseEntitySet(es)
				metadata.EntitySets[es.Name] = entitySet
			}

			// Parse function imports from entity container
			for _, fi := range schema.EntityContainer.FunctionImports {
				functionImport := parseFunctionImport(fi)
				metadata.FunctionImports[fi.Name] = functionImport
			}
		}

		// Also check for function imports at schema level (SAP pattern)
		for _, fi := range schema.FunctionImports {
			functionImport := parseFunctionImport(fi)
			metadata.FunctionImports[fi.Name] = functionImport
		}
	}

	return metadata, nil
}

// parseEntityType converts XML entity type to model
func parseEntityType(et EntityType) *models.EntityType {
	entityType := &models.EntityType{
		Name:            et.Name,
		Properties:      make([]*models.EntityProperty, 0),
		KeyProperties:   make([]string, 0),
		NavigationProps: make([]*models.NavigationProperty, 0),
	}

	// Parse key properties
	for _, keyRef := range et.Key.PropertyRefs {
		entityType.KeyProperties = append(entityType.KeyProperties, keyRef.Name)
	}

	// Parse properties
	for _, prop := range et.Properties {
		property := &models.EntityProperty{
			Name:     prop.Name,
			Type:     prop.Type,
			Nullable: prop.Nullable != "false", // Default to true if not specified
			IsKey:    slices.Contains(entityType.KeyProperties, prop.Name),
		}
		entityType.Properties = append(entityType.Properties, property)
	}

	// Parse navigation properties
	for _, navProp := range et.NavigationProperties {
		navigationProp := &models.NavigationProperty{
			Name:         navProp.Name,
			Relationship: navProp.Relationship,
			ToRole:       navProp.ToRole,
			FromRole:     navProp.FromRole,
		}
		entityType.NavigationProps = append(entityType.NavigationProps, navigationProp)
	}

	return entityType
}

// parseEntitySet converts XML entity set to model
func parseEntitySet(es EntitySet) *models.EntitySet {
	// Keep EntityType as-is (may be qualified like "Namespace.TypeName" or short like "TypeName")
	// The bridge will handle lookup with fallback
	entitySet := &models.EntitySet{
		Name:       es.Name,
		EntityType: es.EntityType,
		Creatable:  es.Creatable != "false", // Default to true
		Updatable:  es.Updatable != "false", // Default to true
		Deletable:  es.Deletable != "false", // Default to true
		Searchable: es.Searchable == "true", // Default to false
		Pageable:   es.Pageable != "false",  // Default to true
		// SAP-specific fields (set if attribute is present)
		SAPCreatable:  es.Creatable != "",
		SAPUpdatable:  es.Updatable != "",
		SAPDeletable:  es.Deletable != "",
		SAPSearchable: es.Searchable != "",
		SAPPageable:   es.Pageable != "",
	}

	return entitySet
}

// parseFunctionImport converts XML function import to model
func parseFunctionImport(fi FunctionImport) *models.FunctionImport {
	functionImport := &models.FunctionImport{
		Name:       fi.Name,
		HTTPMethod: fi.HTTPMethod,
		Parameters: make([]*models.FunctionParameter, 0),
	}

	if fi.ReturnType != "" {
		functionImport.ReturnType = fi.ReturnType
	}

	// Default HTTP method to GET if not specified
	if functionImport.HTTPMethod == "" {
		functionImport.HTTPMethod = constants.GET
	}

	// Parse parameters
	for _, param := range fi.Parameters {
		parameter := &models.FunctionParameter{
			Name:     param.Name,
			Type:     param.Type,
			Mode:     param.Mode,
			Nullable: param.Nullable != "false", // Default to true
		}

		// Default mode to In if not specified
		if parameter.Mode == "" {
			parameter.Mode = "In"
		}

		functionImport.Parameters = append(functionImport.Parameters, parameter)
	}

	return functionImport
}
