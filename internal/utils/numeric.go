package utils

import (
	"fmt"
	"reflect"
	"strings"
)

// ConvertNumericFieldsToStrings converts numeric fields to strings for Edm.Decimal types
// This is required because SAP OData v2 expects Edm.Decimal values as JSON strings
func ConvertNumericFieldsToStrings(data map[string]interface{}, decimalFields []string) map[string]interface{} {
	result := make(map[string]interface{})
	
	// Create a set for faster lookup
	decimalFieldSet := make(map[string]bool)
	for _, field := range decimalFields {
		decimalFieldSet[field] = true
		// Also check with case variations
		decimalFieldSet[strings.ToLower(field)] = true
		decimalFieldSet[strings.ToUpper(field)] = true
	}
	
	for key, value := range data {
		// Check if this field should be converted
		if decimalFieldSet[key] || IsLikelyDecimalField(key) {
			result[key] = ConvertToString(value)
		} else {
			// Recursively handle nested structures
			result[key] = ConvertNumericValue(value, decimalFields)
		}
	}
	
	return result
}

// ConvertNumericValue handles conversion of a single value, including nested structures
func ConvertNumericValue(value interface{}, decimalFields []string) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		// Recursively convert nested map
		return ConvertNumericFieldsToStrings(v, decimalFields)
		
	case []interface{}:
		// Convert each item in array
		result := make([]interface{}, len(v))
		for i, item := range v {
			// For items in arrays, also check if they're maps that need conversion
			if itemMap, ok := item.(map[string]interface{}); ok {
				result[i] = ConvertNumericFieldsToStrings(itemMap, decimalFields)
			} else {
				result[i] = ConvertNumericValue(item, decimalFields)
			}
		}
		return result
		
	default:
		// Return other types as-is
		return value
	}
}

// ConvertToString converts numeric types to their string representation
func ConvertToString(value interface{}) string {
	if value == nil {
		return ""
	}
	
	// Check if it's already a string
	if str, ok := value.(string); ok {
		return str
	}
	
	// Use reflection to handle all numeric types
	v := reflect.ValueOf(value)
	switch v.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return fmt.Sprintf("%d", v.Int())
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return fmt.Sprintf("%d", v.Uint())
	case reflect.Float32:
		// For float32, limit precision to avoid floating point artifacts
		f := v.Float()
		// Check if it's a whole number
		if f == float64(int64(f)) {
			return fmt.Sprintf("%.0f", f)
		}
		// Use limited precision for float32 to avoid artifacts
		return fmt.Sprintf("%.6g", f)
	case reflect.Float64:
		// For float64, use more precision and avoid scientific notation
		f := v.Float()
		// Check if it's a whole number
		if f == float64(int64(f)) {
			return fmt.Sprintf("%.0f", f)
		}
		// Otherwise use appropriate decimal places
		return fmt.Sprintf("%g", f)
	case reflect.Bool:
		return fmt.Sprintf("%t", v.Bool())
	default:
		// Fallback to string representation
		return fmt.Sprintf("%v", value)
	}
}

// IsLikelyDecimalField checks if a field name is likely to contain a decimal value
func IsLikelyDecimalField(fieldName string) bool {
	// Common patterns for decimal fields in SAP
	decimalPatterns := []string{
		"amount", "Amount", "AMOUNT",
		"price", "Price", "PRICE",
		"cost", "Cost", "COST",
		"value", "Value", "VALUE",
		"quantity", "Quantity", "QUANTITY",
		"qty", "Qty", "QTY",
		"rate", "Rate", "RATE",
		"percent", "Percent", "PERCENT",
		"discount", "Discount", "DISCOUNT",
		"tax", "Tax", "TAX",
		"total", "Total", "TOTAL",
		"sum", "Sum", "SUM",
		"balance", "Balance", "BALANCE",
		"weight", "Weight", "WEIGHT",
		"volume", "Volume", "VOLUME",
		"NetAmount", "GrossAmount", "TaxAmount",
		"UnitPrice", "ExtendedPrice",
		"OrderQuantity", "DeliveredQuantity",
	}
	
	fieldLower := strings.ToLower(fieldName)
	for _, pattern := range decimalPatterns {
		if strings.Contains(fieldLower, strings.ToLower(pattern)) {
			return true
		}
	}
	
	// Also check if field ends with common numeric suffixes
	numericSuffixes := []string{"_amt", "_amount", "_qty", "_quantity", "_price", "_cost", "_value", "_total"}
	for _, suffix := range numericSuffixes {
		if strings.HasSuffix(fieldLower, suffix) {
			return true
		}
	}
	
	return false
}

// ConvertEntityDataForOData prepares entity data for OData by converting numeric fields
// based on the entity type's property definitions
func ConvertEntityDataForOData(data map[string]interface{}, entityType interface{}) map[string]interface{} {
	// For now, use heuristic-based conversion
	// In a full implementation, this would use entityType metadata to determine
	// which fields are Edm.Decimal and need string conversion
	
	result := make(map[string]interface{})
	
	for key, value := range data {
		if IsLikelyDecimalField(key) {
			// Convert numeric values to strings for decimal fields
			switch v := value.(type) {
			case int, int8, int16, int32, int64,
				uint, uint8, uint16, uint32, uint64,
				float32, float64:
				result[key] = ConvertToString(v)
			default:
				result[key] = value
			}
		} else {
			// Handle nested structures
			switch v := value.(type) {
			case map[string]interface{}:
				// Recursively convert nested objects
				result[key] = ConvertEntityDataForOData(v, nil)
			case []interface{}:
				// Convert arrays of objects
				arr := make([]interface{}, len(v))
				for i, item := range v {
					if itemMap, ok := item.(map[string]interface{}); ok {
						arr[i] = ConvertEntityDataForOData(itemMap, nil)
					} else {
						arr[i] = item
					}
				}
				result[key] = arr
			default:
				result[key] = value
			}
		}
	}
	
	return result
}