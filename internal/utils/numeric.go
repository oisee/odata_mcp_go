package utils

import (
	"fmt"
	"strconv"
	"strings"
)

// Common numeric field patterns that typically use Edm.Decimal in SAP
var decimalFieldPatterns = []string{
	"quantity", "qty",
	"amount", "amt",
	"price", "cost",
	"value", "val",
	"total", "sum",
	"net", "gross",
	"tax", "vat",
	"discount", "disc",
	"rate", "percentage", "percent", "pct",
	"weight", "wgt",
	"volume", "vol",
	"size", "length", "width", "height",
	"balance", "credit", "debit",
	"fee", "charge",
	"margin", "profit",
	"salary", "wage", "pay",
	"budget", "revenue",
	"score", "points",
	"units", "count",
}

// IsLikelyDecimalField checks if a field name suggests it should be Edm.Decimal
func IsLikelyDecimalField(fieldName string) bool {
	lowerName := strings.ToLower(fieldName)
	
	for _, pattern := range decimalFieldPatterns {
		if strings.Contains(lowerName, pattern) {
			return true
		}
	}
	
	// Also check for fields ending with common numeric suffixes
	numericSuffixes := []string{"_qty", "_amt", "_val", "_no", "_num", "_count"}
	for _, suffix := range numericSuffixes {
		if strings.HasSuffix(lowerName, suffix) {
			return true
		}
	}
	
	return false
}

// ConvertNumericToString converts numeric values to strings for OData v2 compatibility
func ConvertNumericToString(value interface{}) interface{} {
	switch v := value.(type) {
	case int:
		return strconv.Itoa(v)
	case int8:
		return strconv.FormatInt(int64(v), 10)
	case int16:
		return strconv.FormatInt(int64(v), 10)
	case int32:
		return strconv.FormatInt(int64(v), 10)
	case int64:
		return strconv.FormatInt(v, 10)
	case uint:
		return strconv.FormatUint(uint64(v), 10)
	case uint8:
		return strconv.FormatUint(uint64(v), 10)
	case uint16:
		return strconv.FormatUint(uint64(v), 10)
	case uint32:
		return strconv.FormatUint(uint64(v), 10)
	case uint64:
		return strconv.FormatUint(v, 10)
	case float32:
		// Use FormatFloat with 'f' to avoid scientific notation
		// -1 precision means use the smallest number of digits necessary
		return strconv.FormatFloat(float64(v), 'f', -1, 32)
	case float64:
		// For float64, also avoid scientific notation
		return strconv.FormatFloat(v, 'f', -1, 64)
	default:
		// Return as-is for non-numeric types
		return value
	}
}

// ConvertNumericsInMap converts numeric fields to strings based on field names
func ConvertNumericsInMap(data map[string]interface{}) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		// Skip system fields
		if strings.HasPrefix(key, "$") || strings.HasPrefix(key, "__") {
			result[key] = value
			continue
		}
		
		// Check if this field should be converted
		if IsLikelyDecimalField(key) {
			// Convert numeric value to string
			result[key] = ConvertNumericValue(value, true)
		} else {
			// Recursively process nested structures
			result[key] = ConvertNumericValue(value, false)
		}
	}
	
	return result
}

// ConvertNumericValue converts a single value, handling nested structures
func ConvertNumericValue(value interface{}, forceConvert bool) interface{} {
	switch v := value.(type) {
	case map[string]interface{}:
		// Recursively convert nested map
		return ConvertNumericsInMap(v)
		
	case []interface{}:
		// Convert each item in array
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = ConvertNumericValue(item, false)
		}
		return result
		
	default:
		// For scalar values, convert if needed
		if forceConvert {
			return ConvertNumericToString(v)
		}
		return v
	}
}

// FormatDecimalString ensures a numeric string has proper decimal formatting
func FormatDecimalString(s string) string {
	// If it's already a properly formatted decimal, return as-is
	if strings.Contains(s, ".") {
		return s
	}
	
	// For integers, we could add .00 but SAP usually accepts integers as-is
	return s
}

// ParseDecimalString attempts to parse a decimal string into a float64
func ParseDecimalString(s string) (float64, error) {
	// Remove any whitespace
	s = strings.TrimSpace(s)
	
	// Handle empty string
	if s == "" {
		return 0, fmt.Errorf("empty string")
	}
	
	// Parse as float
	return strconv.ParseFloat(s, 64)
}