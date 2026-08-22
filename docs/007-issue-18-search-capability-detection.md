# Issue #18: $search Capability Detection

**Date:** 2026-01-31
**Document ID:** 007
**Subject:** Fix for OData $search causing timeouts on unsupported services
**Related:** GitHub Issue #18, OData v4 Capabilities Vocabulary
**Status:** ✅ IMPLEMENTED

---

## Problem

Claude was inconsistently adding `$search` parameter to OData queries. When the service doesn't support full-text search, this caused timeouts.

**Example from issue:**
```
// Works (no search)
GET /Customers

// Fails with timeout (search not supported)
GET /Products?$search=*
```

The user noted: *"$search is a standard OData full-text-search option which I don't think we support."*

---

## Root Cause

The v4 metadata parser defaulted `Searchable` to `true` for all entity sets:

```go
// parser_v4.go - BEFORE
return &models.EntitySet{
    ...
    Searchable: true,  // Assumed all v4 services support $search
}
```

This caused search tools to be generated for every entity set, even when the service doesn't implement `$search`.

**v2 was correct:** It only enabled search when SAP explicitly declared `sap:searchable="true"` in metadata.

---

## Solution

### 1. Parse OData v4 Capability Annotations

OData v4 services can declare search support via the Capabilities vocabulary:

```xml
<EntitySet Name="Products" EntityType="NS.Product">
  <Annotation Term="Org.OData.Capabilities.V1.SearchRestrictions">
    <Record>
      <PropertyValue Property="Searchable" Bool="true"/>
    </Record>
  </Annotation>
</EntitySet>
```

Added annotation parsing to `parser_v4.go`:

```go
// AnnotationV4 represents an OData v4 annotation
type AnnotationV4 struct {
    Term   string           `xml:"Term,attr"`
    Record *AnnotationRecord `xml:"Record"`
}

// Check for SearchRestrictions annotation
for _, ann := range es.Annotations {
    if strings.HasSuffix(ann.Term, "SearchRestrictions") {
        if ann.Record != nil {
            for _, pv := range ann.Record.PropertyValues {
                if pv.Property == "Searchable" {
                    searchable = pv.Bool == "true"
                }
            }
        }
    }
}
```

### 2. Conservative Default

If no annotation is present, default to `false`:

```go
// parser_v4.go - AFTER
searchable := false // Default to false (conservative)

// ... check annotations ...

return &models.EntitySet{
    ...
    Searchable: searchable,
}
```

---

## Behavior Matrix

| Version | Metadata Declaration | Result |
|---------|---------------------|--------|
| **v2** | `sap:searchable="true"` | Search enabled |
| **v2** | `sap:searchable="false"` or missing | Search disabled |
| **v4** | `SearchRestrictions.Searchable=true` | Search enabled |
| **v4** | `SearchRestrictions.Searchable=false` | Search disabled |
| **v4** | No annotation | Search disabled (conservative) |

---

## Files Changed

| File | Change |
|------|--------|
| `internal/metadata/parser_v4.go` | Added `AnnotationV4`, `AnnotationRecord`, `AnnotationPropValue` structs |
| `internal/metadata/parser_v4.go` | Updated `parseEntitySetV4()` to check annotations |
| `internal/metadata/parser_test.go` | Added `TestParseMetadata_V4_SearchRestrictions` |

---

## Testing

```bash
go test ./internal/metadata/... -v -run SearchRestrictions
```

Test verifies:
- Entity set with `Searchable=true` annotation → search enabled
- Entity set with `Searchable=false` annotation → search disabled
- Entity set without annotation → search disabled (default)

---

## Impact

- Services that support `$search` and declare it via annotations: **No change** (works)
- Services that support `$search` but don't declare it: **Search tool not generated** (user can use filter instead)
- Services that don't support `$search`: **Fixed** (no more timeouts)

The conservative default prioritizes reliability over functionality. Most OData services don't implement full-text search, so this matches the common case.
