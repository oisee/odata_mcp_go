package models

import "time"

// EntityProperty represents a property of an OData entity type
type EntityProperty struct {
	Name        string  `json:"name"`
	Type        string  `json:"type"`         // OData type (e.g., "Edm.String")
	Nullable    bool    `json:"nullable"`
	IsKey       bool    `json:"is_key"`
	Description *string `json:"description,omitempty"`
}

// EntityType represents an OData entity type definition
type EntityType struct {
	Name           string            `json:"name"`
	Properties     []*EntityProperty `json:"properties"`
	KeyProperties  []string          `json:"key_properties"`
	Description    *string           `json:"description,omitempty"`
	NavigationProps []*NavigationProperty `json:"navigation_properties,omitempty"`
}

// NavigationProperty represents a navigation property in an entity type
type NavigationProperty struct {
	Name         string `json:"name"`
	Relationship string `json:"relationship"`
	ToRole       string `json:"to_role"`
	FromRole     string `json:"from_role"`
}

// EntitySet represents an OData entity set
type EntitySet struct {
	Name         string  `json:"name"`
	EntityType   string  `json:"entity_type"`
	Creatable    bool    `json:"creatable"`
	Updatable    bool    `json:"updatable"`
	Deletable    bool    `json:"deletable"`
	Searchable   bool    `json:"searchable"`
	Pageable     bool    `json:"pageable"`
	Description  *string `json:"description,omitempty"`
}

// FunctionImportParameter represents a parameter for a function import
type FunctionImportParameter struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Mode     string `json:"mode"` // In, Out, InOut
	Nullable bool   `json:"nullable"`
}

// FunctionImport represents an OData function import
type FunctionImport struct {
	Name        string                     `json:"name"`
	HTTPMethod  string                     `json:"http_method"`
	ReturnType  *string                    `json:"return_type,omitempty"`
	Parameters  []*FunctionImportParameter `json:"parameters"`
	Description *string                    `json:"description,omitempty"`
}

// ODataMetadata represents the complete OData service metadata
type ODataMetadata struct {
	ServiceRoot    string                   `json:"service_root"`
	EntityTypes    map[string]*EntityType   `json:"entity_types"`
	EntitySets     map[string]*EntitySet    `json:"entity_sets"`
	FunctionImports map[string]*FunctionImport `json:"function_imports"`
	SchemaNamespace string                   `json:"schema_namespace"`
	ContainerName   string                   `json:"container_name"`
	Version        string                   `json:"version"`
	ParsedAt       time.Time                `json:"parsed_at"`
}

// ODataError represents an OData error response
type ODataError struct {
	Code        string                 `json:"code,omitempty"`
	Message     string                 `json:"message"`
	Details     []ODataErrorDetail     `json:"details,omitempty"`
	InnerError  map[string]interface{} `json:"innererror,omitempty"`
	Target      string                 `json:"target,omitempty"`
	Severity    string                 `json:"severity,omitempty"`
}

// ODataErrorDetail represents detailed error information
type ODataErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
	Target  string `json:"target,omitempty"`
}

// ODataResponse represents a generic OData response
type ODataResponse struct {
	Context   string                 `json:"@odata.context,omitempty"`
	Count     *int64                 `json:"@odata.count,omitempty"`
	NextLink  string                 `json:"@odata.nextLink,omitempty"`
	Value     []interface{}          `json:"value,omitempty"`
	Error     *ODataError            `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"@odata.metadata,omitempty"`
}

// ToolInfo represents information about a generated MCP tool
type ToolInfo struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []ToolParameter        `json:"parameters"`
	EntitySet   string                 `json:"entity_set,omitempty"`
	Operation   string                 `json:"operation,omitempty"`
	Function    string                 `json:"function,omitempty"`
	Properties  map[string]interface{} `json:"properties,omitempty"`
}

// ToolParameter represents a parameter for an MCP tool
type ToolParameter struct {
	Name        string      `json:"name"`
	Type        string      `json:"type"`
	Description string      `json:"description"`
	Required    bool        `json:"required"`
	Default     interface{} `json:"default,omitempty"`
}

// TraceInfo represents comprehensive information for trace mode
type TraceInfo struct {
	ServiceURL       string              `json:"service_url"`
	MCPName          string              `json:"mcp_name"`
	ToolNaming       string              `json:"tool_naming"`
	ToolPrefix       string              `json:"tool_prefix,omitempty"`
	ToolPostfix      string              `json:"tool_postfix,omitempty"`
	ToolShrink       bool                `json:"tool_shrink"`
	SortTools        bool                `json:"sort_tools"`
	EntityFilter     []string            `json:"entity_filter,omitempty"`
	FunctionFilter   []string            `json:"function_filter,omitempty"`
	Authentication   string              `json:"authentication"`
	MetadataSummary  MetadataSummary     `json:"metadata_summary"`
	RegisteredTools  []ToolInfo          `json:"registered_tools"`
	TotalTools       int                 `json:"total_tools"`
}

// MetadataSummary represents a summary of parsed metadata
type MetadataSummary struct {
	EntityTypes      int `json:"entity_types"`
	EntitySets       int `json:"entity_sets"`
	FunctionImports  int `json:"function_imports"`
}