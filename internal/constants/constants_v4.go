package constants

// OData v4 XML namespaces
const (
	EdmNamespaceV4  = "http://docs.oasis-open.org/odata/ns/edm"
	EdmxNamespaceV4 = "http://docs.oasis-open.org/odata/ns/edmx"
)

// OData v4 specific type mappings
var ODataTypeMapV4 = map[string]string{
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
	"Edm.Date":             "string", // ISO 8601 date string (new in v4)
	"Edm.TimeOfDay":        "string", // ISO 8601 time string (new in v4)
	"Edm.DateTimeOffset":   "string", // ISO 8601 string with timezone
	"Edm.Duration":         "string", // ISO 8601 duration (replaces Edm.Time)
	"Edm.Guid":             "string", // UUID string
	"Edm.Binary":           "string", // Base64 encoded string
	"Edm.Stream":           "string", // Stream reference (new in v4)
}

// OData v4 content types
const (
	ContentTypeODataJSONV4         = "application/json;odata.metadata=minimal"
	ContentTypeODataJSONFullV4     = "application/json;odata.metadata=full"
	ContentTypeODataJSONNoneV4     = "application/json;odata.metadata=none"
	ContentTypeODataJSONStreamingV4 = "application/json;odata.streaming=true"
)

// OData v4 query options (additional to v2)
const (
	QueryApply   = "$apply"   // Data aggregation transformations
	QueryCompute = "$compute" // Computed properties
	QueryLevels  = "$levels"  // Expand levels for hierarchical data
)

// OData v4 annotations
const (
	ODataContext    = "@odata.context"
	ODataType       = "@odata.type"
	ODataID         = "@odata.id"
	ODataEditLink   = "@odata.editLink"
	ODataCount      = "@odata.count"
	ODataNextLink   = "@odata.nextLink"
	ODataDeltaLink  = "@odata.deltaLink"
)

// IsODataV4Namespace checks if the namespace is OData v4
func IsODataV4Namespace(namespace string) bool {
	return namespace == EdmNamespaceV4 || namespace == EdmxNamespaceV4
}

// GetODataVersion determines the OData version from the namespace
func GetODataVersion(namespace string) string {
	if IsODataV4Namespace(namespace) {
		return "4.0"
	}
	return "2.0"
}