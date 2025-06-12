package constants

import "fmt"

// OData XML namespaces
const (
	EdmNamespace  = "http://schemas.microsoft.com/ado/2006/04/edm"
	EdmxNamespace = "http://schemas.microsoft.com/ado/2007/06/edmx"
	SAPNamespace  = "http://www.sap.com/Protocols/SAPData"
	AtomNamespace = "http://www.w3.org/2005/Atom"
	AppNamespace  = "http://www.w3.org/2007/app"
)

// OData primitive type mappings to Go types
var ODataTypeMap = map[string]string{
	"Edm.String":           "string",
	"Edm.Int16":            "int16",
	"Edm.Int32":            "int32",
	"Edm.Int64":            "int64",
	"Edm.Boolean":          "bool",
	"Edm.Byte":             "byte",
	"Edm.SByte":            "int8",
	"Edm.Single":           "float32",
	"Edm.Double":           "float64",
	"Edm.Decimal":          "string", // Use string for precision
	"Edm.DateTime":         "string", // ISO 8601 string
	"Edm.DateTimeOffset":   "string", // ISO 8601 string with timezone
	"Edm.Time":             "string", // Duration string
	"Edm.Guid":             "string", // UUID string
	"Edm.Binary":           "string", // Base64 encoded string
}

// HTTP methods supported by OData
const (
	GET    = "GET"
	POST   = "POST"
	PUT    = "PUT"
	PATCH  = "PATCH"
	MERGE  = "MERGE"
	DELETE = "DELETE"
)

// OData system query options
const (
	QueryFilter    = "$filter"
	QuerySelect    = "$select"
	QueryExpand    = "$expand"
	QueryOrderBy   = "$orderby"
	QueryTop       = "$top"
	QuerySkip      = "$skip"
	QueryCount     = "$count"
	QuerySearch    = "$search"
	QueryFormat    = "$format"
	QuerySkipToken = "$skiptoken"
	QueryInlineCount = "$inlinecount"
)

// SAP-specific query options
const (
	SAPQuerySearch = "search"
)

// CSRF Token headers (SAP-specific)
const (
	CSRFTokenHeader      = "X-CSRF-Token"
	CSRFTokenFetch       = "Fetch"
	CSRFTokenHeaderLower = "x-csrf-token"
)

// HTTP headers
const (
	ContentType     = "Content-Type"
	Accept          = "Accept"
	Authorization   = "Authorization"
	UserAgent       = "User-Agent"
	IfMatch         = "If-Match"
	IfNoneMatch     = "If-None-Match"
)

// Content types
const (
	ContentTypeJSON       = "application/json"
	ContentTypeXML        = "application/xml"
	ContentTypeAtomXML    = "application/atom+xml"
	ContentTypeFormURL    = "application/x-www-form-urlencoded"
	ContentTypeODataJSON  = "application/json;odata=verbose"
	ContentTypeODataAtom  = "application/atom+xml;type=entry"
)

// OData metadata endpoints
const (
	MetadataEndpoint     = "$metadata"
	ServiceDocEndpoint   = ""
	BatchEndpoint        = "$batch"
)

// Tool operation types
const (
	OpFilter = "filter"
	OpCount  = "count"
	OpSearch = "search"
	OpGet    = "get"
	OpCreate = "create"
	OpUpdate = "update"
	OpDelete = "delete"
	OpInfo   = "info"
)

// Tool operation names (for shrinking)
var ToolOperationNames = map[string]string{
	OpFilter: "filter",
	OpCount:  "count",
	OpSearch: "search",
	OpGet:    "get",
	OpCreate: "create",
	OpUpdate: "update",
	OpDelete: "delete",
	OpInfo:   "info",
}

// Shortened tool operation names
var ShortenedToolOperationNames = map[string]string{
	OpFilter: "filter",
	OpCount:  "count",
	OpSearch: "search",
	OpGet:    "get",
	OpCreate: "create",
	OpUpdate: "upd",
	OpDelete: "del",
	OpInfo:   "info",
}

// Error messages
const (
	ErrInvalidServiceURL    = "invalid service URL"
	ErrMetadataNotFound     = "metadata not found"
	ErrEntitySetNotFound    = "entity set not found"
	ErrEntityTypeNotFound   = "entity type not found"
	ErrFunctionNotFound     = "function import not found"
	ErrAuthenticationFailed = "authentication failed"
	ErrCSRFTokenFailed      = "CSRF token fetch failed"
	ErrRequestFailed        = "HTTP request failed"
	ErrResponseParseFailed  = "response parsing failed"
)

// Default values
const (
	DefaultUserAgent          = "OData-MCP-Bridge/1.0 (Go)"
	DefaultTimeout            = 30 // seconds
	DefaultMaxResponseSize    = 10 * 1024 * 1024 // 10MB
	DefaultMaxItems           = 1000
	DefaultToolNameMaxLength  = 64
)

// MCP-specific constants
const (
	MCPProtocolVersion = "2024-11-05"
	MCPServerName      = "odata-mcp-bridge"
	MCPServerVersion   = "1.0.0"
)

// GetGoType returns the Go type for an OData type
func GetGoType(odataType string) string {
	if goType, ok := ODataTypeMap[odataType]; ok {
		return goType
	}
	return "interface{}" // fallback for unknown types
}

// GetToolOperationName returns the operation name for tools
func GetToolOperationName(operation string, shrink bool) string {
	if shrink {
		if name, ok := ShortenedToolOperationNames[operation]; ok {
			return name
		}
	}
	if name, ok := ToolOperationNames[operation]; ok {
		return name
	}
	return operation
}

// FormatServiceID extracts a service identifier from a service URL for tool naming
func FormatServiceID(serviceURL string) string {
	// Simple extraction - in a real implementation, this would be more sophisticated
	// to match the Python version's service identification logic
	if len(serviceURL) > 50 {
		return "service"
	}
	return fmt.Sprintf("svc_%d", len(serviceURL)%1000)
}