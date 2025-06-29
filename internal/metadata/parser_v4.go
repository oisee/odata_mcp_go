package metadata

import (
	"encoding/xml"
	"fmt"
	"strings"
	"time"

	"github.com/odata-mcp/go/internal/models"
)

// EDMXV4 represents the root EDMX document for OData v4
type EDMXV4 struct {
	XMLName      xml.Name       `xml:"Edmx"`
	Version      string         `xml:"Version,attr"`
	DataServices DataServicesV4 `xml:"DataServices"`
}

// DataServicesV4 contains the schema for OData v4
type DataServicesV4 struct {
	XMLName xml.Name   `xml:"DataServices"`
	Schemas []SchemaV4 `xml:"Schema"`
}

// SchemaV4 contains entity types, entity sets, and function imports for OData v4
type SchemaV4 struct {
	XMLName          xml.Name           `xml:"Schema"`
	Namespace        string             `xml:"Namespace,attr"`
	EntityTypes      []EntityTypeV4     `xml:"EntityType"`
	ComplexTypes     []ComplexTypeV4    `xml:"ComplexType"`
	EnumTypes        []EnumTypeV4       `xml:"EnumType"`
	EntityContainers []EntityContainerV4 `xml:"EntityContainer"`
	Functions        []FunctionV4        `xml:"Function"`
	Actions          []ActionV4          `xml:"Action"`
}

// EntityTypeV4 represents an OData v4 entity type
type EntityTypeV4 struct {
	XMLName              xml.Name               `xml:"EntityType"`
	Name                 string                 `xml:"Name,attr"`
	BaseType             string                 `xml:"BaseType,attr"`
	Abstract             string                 `xml:"Abstract,attr"`
	OpenType             string                 `xml:"OpenType,attr"`
	Key                  KeyV4                  `xml:"Key"`
	Properties           []PropertyV4           `xml:"Property"`
	NavigationProperties []NavigationPropertyV4 `xml:"NavigationProperty"`
}

// ComplexTypeV4 represents an OData v4 complex type
type ComplexTypeV4 struct {
	XMLName              xml.Name               `xml:"ComplexType"`
	Name                 string                 `xml:"Name,attr"`
	BaseType             string                 `xml:"BaseType,attr"`
	Abstract             string                 `xml:"Abstract,attr"`
	OpenType             string                 `xml:"OpenType,attr"`
	Properties           []PropertyV4           `xml:"Property"`
	NavigationProperties []NavigationPropertyV4 `xml:"NavigationProperty"`
}

// EnumTypeV4 represents an OData v4 enum type
type EnumTypeV4 struct {
	XMLName      xml.Name      `xml:"EnumType"`
	Name         string        `xml:"Name,attr"`
	UnderlyingType string      `xml:"UnderlyingType,attr"`
	IsFlags      string        `xml:"IsFlags,attr"`
	Members      []EnumMemberV4 `xml:"Member"`
}

// EnumMemberV4 represents a member of an enum type
type EnumMemberV4 struct {
	XMLName xml.Name `xml:"Member"`
	Name    string   `xml:"Name,attr"`
	Value   string   `xml:"Value,attr"`
}

// KeyV4 contains key properties for OData v4
type KeyV4 struct {
	XMLName      xml.Name       `xml:"Key"`
	PropertyRefs []PropertyRefV4 `xml:"PropertyRef"`
}

// PropertyRefV4 references a key property in OData v4
type PropertyRefV4 struct {
	XMLName xml.Name `xml:"PropertyRef"`
	Name    string   `xml:"Name,attr"`
}

// PropertyV4 represents an entity property in OData v4
type PropertyV4 struct {
	XMLName       xml.Name `xml:"Property"`
	Name          string   `xml:"Name,attr"`
	Type          string   `xml:"Type,attr"`
	Nullable      string   `xml:"Nullable,attr"`
	MaxLength     string   `xml:"MaxLength,attr"`
	Precision     string   `xml:"Precision,attr"`
	Scale         string   `xml:"Scale,attr"`
	Unicode       string   `xml:"Unicode,attr"`
	DefaultValue  string   `xml:"DefaultValue,attr"`
}

// NavigationPropertyV4 represents a navigation property in OData v4
type NavigationPropertyV4 struct {
	XMLName              xml.Name `xml:"NavigationProperty"`
	Name                 string   `xml:"Name,attr"`
	Type                 string   `xml:"Type,attr"`
	Nullable             string   `xml:"Nullable,attr"`
	Partner              string   `xml:"Partner,attr"`
	ContainsTarget       string   `xml:"ContainsTarget,attr"`
}

// EntityContainerV4 contains entity sets and singletons for OData v4
type EntityContainerV4 struct {
	XMLName              xml.Name               `xml:"EntityContainer"`
	Name                 string                 `xml:"Name,attr"`
	Extends              string                 `xml:"Extends,attr"`
	EntitySets           []EntitySetV4          `xml:"EntitySet"`
	Singletons           []SingletonV4          `xml:"Singleton"`
	FunctionImports      []FunctionImportV4     `xml:"FunctionImport"`
	ActionImports        []ActionImportV4       `xml:"ActionImport"`
}

// EntitySetV4 represents an OData v4 entity set
type EntitySetV4 struct {
	XMLName                  xml.Name                     `xml:"EntitySet"`
	Name                     string                       `xml:"Name,attr"`
	EntityType               string                       `xml:"EntityType,attr"`
	NavigationPropertyBindings []NavigationPropertyBinding `xml:"NavigationPropertyBinding"`
}

// SingletonV4 represents an OData v4 singleton
type SingletonV4 struct {
	XMLName                  xml.Name                     `xml:"Singleton"`
	Name                     string                       `xml:"Name,attr"`
	Type                     string                       `xml:"Type,attr"`
	NavigationPropertyBindings []NavigationPropertyBinding `xml:"NavigationPropertyBinding"`
}

// NavigationPropertyBinding represents a navigation property binding
type NavigationPropertyBinding struct {
	XMLName xml.Name `xml:"NavigationPropertyBinding"`
	Path    string   `xml:"Path,attr"`
	Target  string   `xml:"Target,attr"`
}

// FunctionImportV4 represents an OData v4 function import
type FunctionImportV4 struct {
	XMLName      xml.Name `xml:"FunctionImport"`
	Name         string   `xml:"Name,attr"`
	Function     string   `xml:"Function,attr"`
	EntitySet    string   `xml:"EntitySet,attr"`
	IncludeInServiceDocument string `xml:"IncludeInServiceDocument,attr"`
}

// ActionImportV4 represents an OData v4 action import
type ActionImportV4 struct {
	XMLName      xml.Name `xml:"ActionImport"`
	Name         string   `xml:"Name,attr"`
	Action       string   `xml:"Action,attr"`
	EntitySet    string   `xml:"EntitySet,attr"`
}

// FunctionV4 represents an OData v4 function
type FunctionV4 struct {
	XMLName      xml.Name      `xml:"Function"`
	Name         string        `xml:"Name,attr"`
	IsBound      string        `xml:"IsBound,attr"`
	IsComposable string        `xml:"IsComposable,attr"`
	Parameters   []ParameterV4 `xml:"Parameter"`
	ReturnType   ReturnTypeV4  `xml:"ReturnType"`
}

// ActionV4 represents an OData v4 action
type ActionV4 struct {
	XMLName      xml.Name      `xml:"Action"`
	Name         string        `xml:"Name,attr"`
	IsBound      string        `xml:"IsBound,attr"`
	Parameters   []ParameterV4 `xml:"Parameter"`
	ReturnType   *ReturnTypeV4 `xml:"ReturnType"`
}

// ParameterV4 represents a function/action parameter in OData v4
type ParameterV4 struct {
	XMLName  xml.Name `xml:"Parameter"`
	Name     string   `xml:"Name,attr"`
	Type     string   `xml:"Type,attr"`
	Nullable string   `xml:"Nullable,attr"`
}

// ReturnTypeV4 represents a function/action return type in OData v4
type ReturnTypeV4 struct {
	XMLName  xml.Name `xml:"ReturnType"`
	Type     string   `xml:"Type,attr"`
	Nullable string   `xml:"Nullable,attr"`
}

// ParseMetadataV4 parses OData v4 metadata XML and returns structured metadata
func ParseMetadataV4(data []byte, serviceRoot string) (*models.ODataMetadata, error) {
	var edmx EDMXV4
	if err := xml.Unmarshal(data, &edmx); err != nil {
		return nil, fmt.Errorf("failed to parse v4 metadata XML: %w", err)
	}

	if len(edmx.DataServices.Schemas) == 0 {
		return nil, fmt.Errorf("no schemas found in metadata")
	}

	// Find the main schema and container
	var mainSchema *SchemaV4
	var mainContainer *EntityContainerV4
	
	for i := range edmx.DataServices.Schemas {
		schema := &edmx.DataServices.Schemas[i]
		if len(schema.EntityContainers) > 0 {
			mainSchema = schema
			mainContainer = &schema.EntityContainers[0]
			break
		}
	}
	
	if mainSchema == nil || mainContainer == nil {
		return nil, fmt.Errorf("no entity container found in metadata")
	}
	
	metadata := &models.ODataMetadata{
		ServiceRoot:     serviceRoot,
		EntityTypes:     make(map[string]*models.EntityType),
		EntitySets:      make(map[string]*models.EntitySet),
		FunctionImports: make(map[string]*models.FunctionImport),
		SchemaNamespace: mainSchema.Namespace,
		ContainerName:   mainContainer.Name,
		Version:         edmx.Version,
		ParsedAt:        time.Now(),
	}

	// Parse entity types from all schemas
	for _, schema := range edmx.DataServices.Schemas {
		for _, et := range schema.EntityTypes {
			entityType := parseEntityTypeV4(et)
			metadata.EntityTypes[et.Name] = entityType
		}
	}

	// Parse entity sets
	for _, es := range mainContainer.EntitySets {
		entitySet := parseEntitySetV4(es, mainSchema.Namespace)
		metadata.EntitySets[es.Name] = entitySet
	}

	// Parse function imports
	for _, fi := range mainContainer.FunctionImports {
		functionImport := parseFunctionImportV4(fi, mainSchema.Functions)
		if functionImport != nil {
			metadata.FunctionImports[fi.Name] = functionImport
		}
	}

	// Parse action imports as function imports (for compatibility)
	for _, ai := range mainContainer.ActionImports {
		actionImport := parseActionImportV4(ai, mainSchema.Actions)
		if actionImport != nil {
			metadata.FunctionImports[ai.Name] = actionImport
		}
	}

	return metadata, nil
}

// parseEntityTypeV4 converts XML entity type to model for OData v4
func parseEntityTypeV4(et EntityTypeV4) *models.EntityType {
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
			Type:     normalizeTypeV4(prop.Type),
			Nullable: prop.Nullable != "false",
			IsKey:    contains(entityType.KeyProperties, prop.Name),
		}
		entityType.Properties = append(entityType.Properties, property)
	}

	// Parse navigation properties
	for _, navProp := range et.NavigationProperties {
		navigationProp := &models.NavigationProperty{
			Name:     navProp.Name,
			Type:     navProp.Type,
			Partner:  navProp.Partner,
			Nullable: navProp.Nullable != "false",
		}
		entityType.NavigationProps = append(entityType.NavigationProps, navigationProp)
	}

	return entityType
}

// parseEntitySetV4 converts XML entity set to model for OData v4
func parseEntitySetV4(es EntitySetV4, namespace string) *models.EntitySet {
	// Remove namespace prefix from entity type if present
	entityTypeName := es.EntityType
	if strings.Contains(entityTypeName, ".") {
		parts := strings.Split(entityTypeName, ".")
		entityTypeName = parts[len(parts)-1]
	}

	return &models.EntitySet{
		Name:       es.Name,
		EntityType: entityTypeName,
		// OData v4 doesn't have explicit CRUD capability attributes in metadata
		// We assume all operations are allowed unless restricted by service
		Creatable:  true,
		Updatable:  true,
		Deletable:  true,
		Searchable: true,
		Pageable:   true,
	}
}

// parseFunctionImportV4 converts XML function import to model for OData v4
func parseFunctionImportV4(fi FunctionImportV4, functions []FunctionV4) *models.FunctionImport {
	// Find the corresponding function definition
	var function *FunctionV4
	functionName := fi.Function
	if strings.Contains(functionName, ".") {
		parts := strings.Split(functionName, ".")
		functionName = parts[len(parts)-1]
	}
	
	for i := range functions {
		if functions[i].Name == functionName {
			function = &functions[i]
			break
		}
	}
	
	if function == nil {
		return nil
	}

	functionImport := &models.FunctionImport{
		Name:       fi.Name,
		HTTPMethod: "GET", // Functions are always GET in OData v4
		Parameters: make([]*models.FunctionParameter, 0),
	}

	// Parse return type
	if function.ReturnType.Type != "" {
		functionImport.ReturnType = normalizeTypeV4(function.ReturnType.Type)
	}

	// Parse parameters
	for _, param := range function.Parameters {
		if param.Name == "bindingParameter" {
			continue // Skip binding parameters
		}
		parameter := &models.FunctionParameter{
			Name:     param.Name,
			Type:     normalizeTypeV4(param.Type),
			Nullable: param.Nullable != "false",
		}
		functionImport.Parameters = append(functionImport.Parameters, parameter)
	}

	return functionImport
}

// parseActionImportV4 converts XML action import to model for OData v4
func parseActionImportV4(ai ActionImportV4, actions []ActionV4) *models.FunctionImport {
	// Find the corresponding action definition
	var action *ActionV4
	actionName := ai.Action
	if strings.Contains(actionName, ".") {
		parts := strings.Split(actionName, ".")
		actionName = parts[len(parts)-1]
	}
	
	for i := range actions {
		if actions[i].Name == actionName {
			action = &actions[i]
			break
		}
	}
	
	if action == nil {
		return nil
	}

	actionImport := &models.FunctionImport{
		Name:       ai.Name,
		HTTPMethod: "POST", // Actions are always POST in OData v4
		Parameters: make([]*models.FunctionParameter, 0),
	}

	// Parse return type
	if action.ReturnType != nil && action.ReturnType.Type != "" {
		actionImport.ReturnType = normalizeTypeV4(action.ReturnType.Type)
	}

	// Parse parameters
	for _, param := range action.Parameters {
		if param.Name == "bindingParameter" {
			continue // Skip binding parameters
		}
		parameter := &models.FunctionParameter{
			Name:     param.Name,
			Type:     normalizeTypeV4(param.Type),
			Nullable: param.Nullable != "false",
		}
		actionImport.Parameters = append(actionImport.Parameters, parameter)
	}

	return actionImport
}

// normalizeTypeV4 normalizes OData v4 type names
func normalizeTypeV4(typeName string) string {
	// Handle collection types
	if strings.HasPrefix(typeName, "Collection(") && strings.HasSuffix(typeName, ")") {
		innerType := typeName[11 : len(typeName)-1]
		return "Collection(" + normalizeTypeV4(innerType) + ")"
	}
	
	// If it's already an Edm type, return as-is
	if strings.HasPrefix(typeName, "Edm.") {
		return typeName
	}
	
	// Remove namespace prefix if present but keep Edm prefix
	if strings.Contains(typeName, ".") {
		parts := strings.Split(typeName, ".")
		if len(parts) == 2 && parts[0] == "Edm" {
			// It's Edm.Type format, keep it
			return typeName
		}
		// For complex types, keep just the type name
		return parts[len(parts)-1]
	}
	
	return typeName
}

// IsODataV4 checks if the metadata is OData v4
func IsODataV4(data []byte) bool {
	var edmx EDMXV4
	if err := xml.Unmarshal(data, &edmx); err != nil {
		return false
	}
	return edmx.Version == "4.0" || edmx.Version == "4.01"
}