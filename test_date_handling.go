package main

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// Test different date formats used in OData services
func main() {
	fmt.Println("=== OData Date Format Testing ===\n")

	// Common date formats in OData
	testDates := map[string]string{
		"ISO 8601":                "2024-12-25T10:30:00",
		"ISO 8601 with Z":         "2024-12-25T10:30:00Z",
		"ISO 8601 with offset":    "2024-12-25T10:30:00+01:00",
		"OData v2 Legacy":         "/Date(1735125000000)/",
		"OData v2 with offset":    "/Date(1735125000000+0100)/",
		"Date only (ISO)":         "2024-12-25",
		"Time only":               "PT10H30M",
		"Invalid format":          "25/12/2024",
	}

	fmt.Println("1. Parsing different date formats:")
	fmt.Println(strings.Repeat("-", 60))
	for name, dateStr := range testDates {
		fmt.Printf("%-25s: %s\n", name, dateStr)
		
		// Try to detect and parse the format
		if isODataLegacyDate(dateStr) {
			if ts, offset, ok := parseODataLegacyDate(dateStr); ok {
				fmt.Printf("  -> Parsed as epoch: %d ms, offset: %s\n", ts, offset)
				fmt.Printf("  -> As time: %s\n", time.UnixMilli(ts).UTC())
			}
		} else if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			fmt.Printf("  -> Parsed as RFC3339: %s\n", t)
		} else if t, err := time.Parse("2006-01-02", dateStr); err == nil {
			fmt.Printf("  -> Parsed as date only: %s\n", t)
		} else {
			fmt.Printf("  -> Could not parse\n")
		}
		fmt.Println()
	}

	// Test conversion functions
	fmt.Println("\n2. Converting between formats:")
	fmt.Println(strings.Repeat("-", 60))
	
	// ISO to OData legacy
	iso := "2024-12-25T10:30:00Z"
	if legacy := convertISOToODataLegacy(iso); legacy != "" {
		fmt.Printf("ISO to Legacy: %s -> %s\n", iso, legacy)
	}
	
	// OData legacy to ISO
	legacy := "/Date(1735125000000)/"
	if iso := convertODataLegacyToISO(legacy); iso != "" {
		fmt.Printf("Legacy to ISO: %s -> %s\n", legacy, iso)
	}

	// Test with actual SAP response data
	fmt.Println("\n3. Processing SAP OData response:")
	fmt.Println(strings.Repeat("-", 60))
	
	sapResponse := map[string]interface{}{
		"OrderID": "12345",
		"CreatedAt": "/Date(1735125000000)/",
		"DeliveryDate": "/Date(1735729800000)/",
		"LastModified": "/Date(1735125000000+0100)/",
		"Amount": 99.99,
		"Items": []interface{}{
			map[string]interface{}{
				"ItemID": "001",
				"CreatedDate": "/Date(1735125000000)/",
			},
		},
	}
	
	fmt.Println("Original SAP response:")
	printJSON(sapResponse)
	
	// Convert dates for display
	converted := convertDatesInResponse(sapResponse, true) // true = legacy to ISO
	fmt.Println("\nConverted to ISO format:")
	printJSON(converted)
	
	// Convert back for sending to SAP
	reconverted := convertDatesInResponse(converted, false) // false = ISO to legacy
	fmt.Println("\nConverted back to legacy format:")
	printJSON(reconverted)
}

// Helper functions for date handling

func isODataLegacyDate(s string) bool {
	return strings.HasPrefix(s, "/Date(") && strings.HasSuffix(s, ")/")
}

func parseODataLegacyDate(s string) (int64, string, bool) {
	// Pattern: /Date(milliseconds[+/-offset])/
	re := regexp.MustCompile(`/Date\((\d+)([\+\-]\d{4})?\)/`)
	matches := re.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, "", false
	}
	
	ms, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, "", false
	}
	
	offset := ""
	if len(matches) > 2 {
		offset = matches[2]
	}
	
	return ms, offset, true
}

func convertISOToODataLegacy(iso string) string {
	t, err := time.Parse(time.RFC3339, iso)
	if err != nil {
		// Try without timezone
		t, err = time.Parse("2006-01-02T15:04:05", iso)
		if err != nil {
			return ""
		}
	}
	
	ms := t.UnixMilli()
	return fmt.Sprintf("/Date(%d)/", ms)
}

func convertODataLegacyToISO(legacy string) string {
	ms, _, ok := parseODataLegacyDate(legacy)
	if !ok {
		return ""
	}
	
	t := time.UnixMilli(ms).UTC()
	return t.Format(time.RFC3339)
}

func convertDatesInResponse(data interface{}, legacyToISO bool) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			if str, ok := value.(string); ok {
				if legacyToISO && isODataLegacyDate(str) {
					if converted := convertODataLegacyToISO(str); converted != "" {
						result[key] = converted
					} else {
						result[key] = value
					}
				} else if !legacyToISO && isISODateTime(str) {
					if converted := convertISOToODataLegacy(str); converted != "" {
						result[key] = converted
					} else {
						result[key] = value
					}
				} else {
					result[key] = convertDatesInResponse(value, legacyToISO)
				}
			} else {
				result[key] = convertDatesInResponse(value, legacyToISO)
			}
		}
		return result
		
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = convertDatesInResponse(item, legacyToISO)
		}
		return result
		
	default:
		return data
	}
}

func isISODateTime(s string) bool {
	// Simple check for ISO 8601 format
	if len(s) < 10 {
		return false
	}
	// Check for YYYY-MM-DD pattern at start
	if s[4] != '-' || s[7] != '-' {
		return false
	}
	// Check if it has time component
	return len(s) > 10 && s[10] == 'T'
}

func printJSON(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}