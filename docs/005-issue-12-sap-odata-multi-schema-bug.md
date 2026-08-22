# Issue #12: SAP OData Multi-Schema Parser Bug

**Date:** 2026-01-31
**Document ID:** 005
**Subject:** SAP OData services not generating EntitySet tools
**GitHub Issue:** https://github.com/oisee/odata_mcp_go/issues/12
**Reporter:** Emily Celen
**Status:** ✅ IMPLEMENTED

---

## Problem Summary

SAP OData services only show the generic `odata_service_info_for_sap` tool instead of discovering EntitySets, while Northwind service works correctly.

**Expected:** All EntitySets discovered as individual tools
**Actual:** Only `odata_service_info_for_sap` appears

---

## Root Cause

The metadata parser at `internal/metadata/parser.go` only handles a **single Schema element**:

```go
// Line 22-24 - BUG: Only reads first Schema
type DataServices struct {
    XMLName xml.Name `xml:"DataServices"`
    Schema  Schema   `xml:"Schema"`  // Single schema only!
}
```

**SAP OData services typically use MULTIPLE schemas:**

```xml
<edmx:DataServices>
  <!-- Schema 1: Entity Types -->
  <Schema Namespace="API_PRODUCT_SRV_Entities">
    <EntityType Name="A_Product">...</EntityType>
    <EntityType Name="A_ProductDescription">...</EntityType>
  </Schema>

  <!-- Schema 2: Entity Container -->
  <Schema Namespace="API_PRODUCT_SRV">
    <EntityContainer Name="API_PRODUCT_SRV">
      <EntitySet Name="A_Product" EntityType="API_PRODUCT_SRV_Entities.A_Product"/>
    </EntityContainer>
  </Schema>
</edmx:DataServices>
```

**Northwind uses a single schema** - which is why it works.

---

## Failure Flow

```
1. Parser reads metadata XML
2. Only first Schema element is parsed
3. If EntityTypes are in Schema 1 and EntityContainer in Schema 2:
   - EntityTypes are captured
   - EntityContainer is MISSED (or vice versa)
4. generateEntitySetTools() at bridge.go:323 fails silently:

   entityType, exists := b.metadata.EntityTypes[entitySet.EntityType]
   if !exists {
       return  // Silent failure - no tools generated
   }

5. Only service_info tool remains
```

---

## Fix

### Change 1: Parser Types (`internal/metadata/parser.go`)

```go
// BEFORE (line 22-24)
type DataServices struct {
    XMLName xml.Name `xml:"DataServices"`
    Schema  Schema   `xml:"Schema"`
}

// AFTER
type DataServices struct {
    XMLName xml.Name `xml:"DataServices"`
    Schemas []Schema `xml:"Schema"`  // Multiple schemas
}
```

### Change 2: ParseMetadata Function

```go
func ParseMetadata(data []byte, serviceRoot string) (*models.ODataMetadata, error) {
    if IsODataV4(data) {
        return ParseMetadataV4(data, serviceRoot)
    }

    var edmx EDMX
    if err := xml.Unmarshal(data, &edmx); err != nil {
        return nil, fmt.Errorf("failed to parse metadata XML: %w", err)
    }

    metadata := &models.ODataMetadata{
        ServiceRoot:     serviceRoot,
        EntityTypes:     make(map[string]*models.EntityType),
        EntitySets:      make(map[string]*models.EntitySet),
        FunctionImports: make(map[string]*models.FunctionImport),
        ParsedAt:        time.Now(),
    }

    // Merge all schemas
    for _, schema := range edmx.DataServices.Schemas {
        // Capture namespace from first schema (or schema with container)
        if metadata.SchemaNamespace == "" {
            metadata.SchemaNamespace = schema.Namespace
        }

        // Parse entity types from ALL schemas
        for _, et := range schema.EntityTypes {
            entityType := parseEntityType(et)
            // Store with namespace prefix for cross-schema lookup
            metadata.EntityTypes[et.Name] = entityType
            metadata.EntityTypes[schema.Namespace+"."+et.Name] = entityType
        }

        // Parse entity container (usually only in one schema)
        if schema.EntityContainer.Name != "" {
            metadata.ContainerName = schema.EntityContainer.Name

            for _, es := range schema.EntityContainer.EntitySets {
                entitySet := parseEntitySet(es, schema.Namespace)
                metadata.EntitySets[es.Name] = entitySet
            }

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
```

### Change 3: EntityType Lookup in Bridge

The bridge at `bridge.go:323` looks up entity types by name. SAP uses qualified names like `API_PRODUCT_SRV_Entities.A_Product`. Update lookup:

```go
func (b *ODataMCPBridge) generateEntitySetTools(entitySetName string, entitySet *models.EntitySet) {
    // Try direct lookup first
    entityType, exists := b.metadata.EntityTypes[entitySet.EntityType]

    // If not found, try without namespace prefix
    if !exists {
        parts := strings.Split(entitySet.EntityType, ".")
        shortName := parts[len(parts)-1]
        entityType, exists = b.metadata.EntityTypes[shortName]
    }

    if !exists {
        if b.config.Verbose {
            fmt.Printf("[VERBOSE] Entity type not found for %s: %s\n",
                entitySetName, entitySet.EntityType)
        }
        return
    }

    // ... rest of tool generation
}
```

---

## Estimated Changes

| File | Change | LOC |
|------|--------|-----|
| `internal/metadata/parser.go` | Multi-schema support | ~40 |
| `internal/bridge/bridge.go` | Qualified name lookup | ~10 |
| `internal/metadata/parser_test.go` | Test with SAP metadata | ~50 |

**Total:** ~100 LOC

---

## Testing

1. Add test case with SAP-style multi-schema metadata
2. Verify Northwind still works (regression)
3. Test with real SAP OData service (API_PRODUCT_SRV)

---

## Verification

After fix, SAP service should show tools like:
- `filter_A_Product_for_sap`
- `get_A_Product_for_sap`
- `filter_A_ProductDescription_for_sap`
- etc.

Instead of just:
- `odata_service_info_for_sap`
