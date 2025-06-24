package utils

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	// Regex for parsing OData v2 legacy date format: /Date(milliseconds[+/-offset])/
	odataLegacyDateRegex = regexp.MustCompile(`^/Date\((-?\d+)([\+\-]\d{4})?\)/$`)
	
	// Common date field names that typically contain dates in SAP systems
	dateFieldPatterns = []string{
		"Date", "date", "DATE",
		"Time", "time", "TIME",
		"At", "at", "AT",
		"On", "on", "ON",
		"Created", "created", "CREATED",
		"Modified", "modified", "MODIFIED",
		"Updated", "updated", "UPDATED",
		"Changed", "changed", "CHANGED",
		"Valid", "valid", "VALID",
		"Expired", "expired", "EXPIRED",
		"Start", "start", "START",
		"End", "end", "END",
		"From", "from", "FROM",
		"To", "to", "TO",
		"Since", "since", "SINCE",
		"Until", "until", "UNTIL",
		"Delivery", "delivery", "DELIVERY",
		"Due", "due", "DUE",
		"Posted", "posted", "POSTED",
		"Timestamp", "timestamp", "TIMESTAMP",
	}
)

// IsODataLegacyDate checks if a string is in OData v2 legacy date format
func IsODataLegacyDate(s string) bool {
	return odataLegacyDateRegex.MatchString(s)
}

// ParseODataLegacyDate extracts milliseconds and offset from OData legacy date
func ParseODataLegacyDate(s string) (milliseconds int64, offset string, ok bool) {
	matches := odataLegacyDateRegex.FindStringSubmatch(s)
	if len(matches) < 2 {
		return 0, "", false
	}
	
	ms, err := strconv.ParseInt(matches[1], 10, 64)
	if err != nil {
		return 0, "", false
	}
	
	if len(matches) > 2 && matches[2] != "" {
		offset = matches[2]
	}
	
	return ms, offset, true
}

// ConvertODataLegacyToISO converts OData legacy date to ISO 8601 format
func ConvertODataLegacyToISO(legacy string) string {
	ms, _, ok := ParseODataLegacyDate(legacy)
	if !ok {
		return legacy // Return as-is if not valid legacy format
	}
	
	t := time.UnixMilli(ms).UTC()
	return t.Format(time.RFC3339)
}

// ConvertISOToODataLegacy converts ISO 8601 date to OData legacy format
func ConvertISOToODataLegacy(iso string) string {
	// Try various ISO formats
	formats := []string{
		time.RFC3339,
		time.RFC3339Nano,
		"2006-01-02T15:04:05",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02",
	}
	
	var t time.Time
	var err error
	
	for _, format := range formats {
		t, err = time.Parse(format, iso)
		if err == nil {
			break
		}
	}
	
	if err != nil {
		return iso // Return as-is if not valid ISO format
	}
	
	ms := t.UnixMilli()
	return fmt.Sprintf("/Date(%d)/", ms)
}

// IsISODateTime checks if a string appears to be an ISO 8601 datetime
func IsISODateTime(s string) bool {
	if len(s) < 10 {
		return false
	}
	
	// Check for YYYY-MM-DD pattern
	if len(s) >= 10 && s[4] == '-' && s[7] == '-' {
		// Could be date only or datetime
		if len(s) == 10 {
			return true // Date only
		}
		if len(s) > 10 && (s[10] == 'T' || s[10] == ' ') {
			return true // DateTime
		}
	}
	
	return false
}

// IsLikelyDateField checks if a field name is likely to contain a date value
func IsLikelyDateField(fieldName string) bool {
	for _, pattern := range dateFieldPatterns {
		if strings.Contains(fieldName, pattern) {
			return true
		}
	}
	return false
}

// ConvertDatesInMap recursively converts date values in a map
// If toISO is true, converts from OData legacy to ISO format
// If toISO is false, converts from ISO to OData legacy format
func ConvertDatesInMap(data map[string]interface{}, toISO bool) map[string]interface{} {
	result := make(map[string]interface{})
	
	for key, value := range data {
		result[key] = ConvertDateValue(value, toISO, key)
	}
	
	return result
}

// ConvertDateValue converts a single value, handling nested structures
func ConvertDateValue(value interface{}, toISO bool, fieldName string) interface{} {
	switch v := value.(type) {
	case string:
		// Check if it's a date value that needs conversion
		if toISO && IsODataLegacyDate(v) {
			return ConvertODataLegacyToISO(v)
		} else if !toISO && IsISODateTime(v) && IsLikelyDateField(fieldName) {
			return ConvertISOToODataLegacy(v)
		}
		return v
		
	case map[string]interface{}:
		// Recursively convert nested map
		return ConvertDatesInMap(v, toISO)
		
	case []interface{}:
		// Convert each item in array
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = ConvertDateValue(item, toISO, "")
		}
		return result
		
	default:
		// Return other types as-is
		return value
	}
}

// ConvertDatesInResponse converts all date fields in an OData response
func ConvertDatesInResponse(response interface{}, toISO bool) interface{} {
	return ConvertDateValue(response, toISO, "")
}

// FormatDateForOData formats a time.Time for OData based on the type
func FormatDateForOData(t time.Time, edmType string, useLegacyFormat bool) string {
	switch edmType {
	case "Edm.DateTime":
		if useLegacyFormat {
			return fmt.Sprintf("/Date(%d)/", t.UnixMilli())
		}
		return t.Format("2006-01-02T15:04:05")
		
	case "Edm.DateTimeOffset":
		if useLegacyFormat {
			// Include timezone offset in legacy format
			_, offset := t.Zone()
			offsetHours := offset / 3600
			offsetMinutes := (offset % 3600) / 60
			sign := "+"
			if offset < 0 {
				sign = "-"
				offsetHours = -offsetHours
				offsetMinutes = -offsetMinutes
			}
			return fmt.Sprintf("/Date(%d%s%02d%02d)/", t.UnixMilli(), sign, offsetHours, offsetMinutes)
		}
		return t.Format(time.RFC3339)
		
	case "Edm.Date":
		return t.Format("2006-01-02")
		
	case "Edm.Time":
		// OData v2 uses ISO 8601 duration format for time
		return fmt.Sprintf("PT%dH%dM%dS", t.Hour(), t.Minute(), t.Second())
		
	default:
		return t.Format(time.RFC3339)
	}
}