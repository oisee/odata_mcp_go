# SAP OData Numeric Field Handling

## Problem

SAP OData v2 services expect `Edm.Decimal` fields to be sent as JSON strings, not JSON numbers. Sending numeric values as JSON numbers results in errors like:

```
Failed to read property 'Quantity' at offset '71'
```

## Solution

The Go implementation now automatically converts numeric values to strings for fields that are likely to be `Edm.Decimal` types in SAP.

## How It Works

### 1. Field Name Detection

The implementation uses heuristic-based field name matching to identify decimal fields. Fields containing these patterns are automatically converted:

- `quantity`, `qty`
- `amount`, `amt`
- `price`, `cost`
- `value`, `val`
- `total`, `sum`
- `net`, `gross`
- `tax`, `vat`
- `discount`, `rate`, `percentage`
- `weight`, `volume`, `size`
- And many more...

### 2. Automatic Conversion

When creating or updating entities:

**Input (from user):**
```json
{
  "Quantity": 1,
  "Price": 99.99,
  "Description": "Laptop"
}
```

**Output (sent to SAP):**
```json
{
  "Quantity": "1",
  "Price": "99.99",
  "Description": "Laptop"
}
```

### 3. Type Support

All numeric types are converted to strings:
- `int`, `int8`, `int16`, `int32`, `int64`
- `uint`, `uint8`, `uint16`, `uint32`, `uint64`
- `float32`, `float64`

Non-numeric types (strings, bools, nil) are left unchanged.

### 4. Nested Structure Support

The conversion works recursively through:
- Nested objects
- Arrays of objects
- Complex hierarchical structures

## Examples

### Creating a Sales Order Line Item

**Before (would fail):**
```json
{
  "SalesOrderID": "0500010047",
  "ProductID": "HT-1251",
  "Quantity": 1,
  "QuantityUnit": "EA",
  "DeliveryDate": "2025-06-24T00:00:00"
}
```

**After conversion (works):**
```json
{
  "SalesOrderID": "0500010047",
  "ProductID": "HT-1251",
  "Quantity": "1",
  "QuantityUnit": "EA",
  "DeliveryDate": "/Date(1750377600000)/"
}
```

Note: Date conversion also happens if legacy dates are enabled (default for SAP).

## Testing

Run the included tests:

```bash
# Unit tests
go test ./internal/test/numeric_conversion_test.go -v

# Demo
go run test_numeric_conversion.go
go run test_json_output.go
```

## Comparison with Python Implementation

The Python implementation likely performs similar conversions. Both implementations now:
- Detect decimal fields by name patterns
- Convert numeric values to strings
- Handle nested structures
- Maintain compatibility with SAP OData v2

## Known Limitations

1. The heuristic approach may miss some decimal fields with unusual names
2. Fields explicitly marked as `Edm.Int32` or other non-decimal types are still converted if their names match patterns
3. No metadata-based type detection (would require parsing OData metadata)

## Future Improvements

1. Parse OData metadata to get exact field types
2. Allow configuration to override field detection
3. Add support for custom field patterns
4. Provide option to disable automatic conversion