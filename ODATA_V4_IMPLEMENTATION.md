# OData v4 Implementation Summary

This document describes the OData v4 support implementation for the OData MCP Bridge.

## Overview

The OData MCP Bridge now supports both OData v2 and v4 services. The implementation automatically detects the OData version from the metadata and adjusts its behavior accordingly.

## Key Changes

### 1. Metadata Parsing

- **New Parser**: Added `parser_v4.go` with complete OData v4 metadata parsing support
- **Auto-detection**: The main `ParseMetadata` function automatically detects v2 vs v4 based on XML namespaces
- **Version Detection**: 
  - v2: Uses namespace `http://schemas.microsoft.com/ado/2007/06/edmx`
  - v4: Uses namespace `http://docs.oasis-open.org/odata/ns/edmx`

### 2. Type System Updates

- **New v4 Types**: Added support for OData v4 specific types:
  - `Edm.Date` - Date without time
  - `Edm.TimeOfDay` - Time without date
  - `Edm.Duration` - ISO 8601 duration (replaces v2's `Edm.Time`)
  - `Edm.Stream` - Binary stream references

- **Updated Navigation Properties**: v4 uses different attributes:
  - v2: `Relationship`, `ToRole`, `FromRole`
  - v4: `Type`, `Partner`, `Nullable`

### 3. Client Updates

- **Version Tracking**: ODataClient now tracks whether it's connected to a v4 service
- **Request Headers**: 
  - v2: Uses `application/json`
  - v4: Uses `application/json;odata.metadata=minimal`

- **Query Parameters**:
  - v2: Uses `$format=json` and `$inlinecount=allpages`
  - v4: Omits `$format` (JSON is default) and uses `$count=true` instead

### 4. Response Handling

- **Response Parser**: Added `response_parser.go` to handle both v2 and v4 response formats
- **Format Differences**:
  - v2: Wraps responses in `{ "d": { ... } }`
  - v4: Direct JSON with OData annotations (`@odata.context`, `@odata.count`, etc.)

### 5. Feature Support

#### OData v4 Features Supported:
- ✅ Automatic version detection
- ✅ v4 metadata parsing
- ✅ v4 response format handling
- ✅ New data types (Date, TimeOfDay, Duration, Stream)
- ✅ `$count` query option
- ✅ `contains()` filter function
- ✅ Navigation property bindings
- ✅ Actions and Functions (mapped to function imports)

#### Limitations:
- Delta queries not yet implemented
- Batch requests use existing v2 approach
- Some v4-specific query options (`$apply`, `$compute`) not exposed in tools

## Testing

### Unit Tests
- `odata_v4_test.go`: Comprehensive unit tests for v4 parsing and handling
- Tests cover metadata parsing, response handling, type mapping, and version detection

### Integration Tests
- `northwind_v4_integration_test.go`: Tests against live Northwind v4 service
- Validates metadata fetching, entity queries, filtering, and navigation

### Test Results
All tests pass successfully, confirming proper v4 support:
```
PASS: TestODataV4MetadataParsing
PASS: TestODataV4ResponseHandling  
PASS: TestODataV4vsV2Detection
PASS: TestODataV4NewTypes
PASS: TestNorthwindV4Integration
```

## Usage Examples

### CLI Usage
```bash
# v2 service
./odata-mcp https://services.odata.org/V2/Northwind/Northwind.svc/

# v4 service (automatically detected)
./odata-mcp https://services.odata.org/V4/Northwind/Northwind.svc/
```

### Claude Desktop Configuration
```json
{
    "mcpServers": {
        "odata-v4-service": {
            "command": "odata-mcp",
            "args": [
                "--service",
                "https://services.odata.org/V4/Northwind/Northwind.svc/"
            ]
        }
    }
}
```

## Implementation Details

### Version Detection Flow
1. Client fetches `$metadata` endpoint
2. `ParseMetadata` checks XML namespace to determine version
3. Calls appropriate parser (`ParseMetadataV2` or `ParseMetadataV4`)
4. Sets `isV4` flag on client for request/response handling

### Request Adaptation
- v4 clients omit `$format=json` parameter
- v4 clients skip `$inlinecount` parameter  
- v4 clients use appropriate Accept header

### Response Normalization
- Both v2 and v4 responses are normalized to consistent internal format
- Tools work identically regardless of OData version
- Version-specific features are abstracted away

## Future Enhancements

1. **Delta Query Support**: Implement v4 delta query functionality
2. **Advanced Query Options**: Expose `$apply`, `$compute`, `$search`
3. **Batch Operations**: Update batch handling for v4 format
4. **Type Definitions**: Support v4 type definitions and enums
5. **Annotations**: Parse and utilize v4 annotations for richer metadata