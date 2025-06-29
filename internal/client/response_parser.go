package client

import (
	"encoding/json"
	"fmt"
	"strings"
)

// parseODataResponse parses OData responses, handling both v2 and v4 formats
func parseODataResponse(data []byte, isV4 bool) (interface{}, error) {
	// Try to parse as a generic map first
	var rawResponse map[string]interface{}
	if err := json.Unmarshal(data, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	// Check for error response
	if errorData, ok := rawResponse["error"]; ok {
		return nil, parseODataError(errorData)
	}

	if isV4 {
		return parseV4Response(rawResponse), nil
	}
	return parseV2Response(rawResponse), nil
}

// parseV2Response handles OData v2 response format
func parseV2Response(response map[string]interface{}) interface{} {
	// OData v2 wraps results in a "d" property
	if d, ok := response["d"]; ok {
		if dMap, ok := d.(map[string]interface{}); ok {
			// Check if it's a collection
			if results, ok := dMap["results"]; ok {
				return map[string]interface{}{
					"value": results,
					"@odata.count": dMap["__count"],
					"@odata.nextLink": dMap["__next"],
				}
			}
			// Single entity
			return d
		}
		return d
	}
	return response
}

// parseV4Response handles OData v4 response format
func parseV4Response(response map[string]interface{}) interface{} {
	// OData v4 uses standard properties without wrapping
	// Collections use "value" property
	if _, hasValue := response["value"]; hasValue {
		// It's already in v4 format
		return response
	}
	
	// Check if it's a single entity (has @odata.context)
	if _, hasContext := response["@odata.context"]; hasContext {
		return response
	}
	
	// Otherwise return as-is
	return response
}

// parseODataError parses OData error responses
func parseODataError(errorData interface{}) error {
	errorBytes, _ := json.Marshal(errorData)
	var odataError struct {
		Code    string `json:"code"`
		Message struct {
			Lang  string `json:"lang"`
			Value string `json:"value"`
		} `json:"message"`
		InnerError interface{} `json:"innererror"`
	}
	
	if err := json.Unmarshal(errorBytes, &odataError); err == nil && odataError.Message.Value != "" {
		return fmt.Errorf("OData error %s: %s", odataError.Code, odataError.Message.Value)
	}
	
	// Try v4 error format
	var v4Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Target  string `json:"target"`
		Details []struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Target  string `json:"target"`
		} `json:"details"`
	}
	
	if err := json.Unmarshal(errorBytes, &v4Error); err == nil && v4Error.Message != "" {
		return fmt.Errorf("OData error %s: %s", v4Error.Code, v4Error.Message)
	}
	
	return fmt.Errorf("OData error: %v", errorData)
}

// normalizeODataResponse normalizes the response to a consistent format
func normalizeODataResponse(data interface{}, isV4 bool) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		if isV4 {
			return v // v4 is already normalized
		}
		// For v2, unwrap the "d" wrapper
		if d, ok := v["d"]; ok {
			return d
		}
		return v
	default:
		return v
	}
}

// extractEntityKey extracts the key from an entity response
func extractEntityKey(entity map[string]interface{}, keyProperties []string) (string, error) {
	if len(keyProperties) == 0 {
		return "", fmt.Errorf("no key properties defined")
	}
	
	if len(keyProperties) == 1 {
		// Single key
		key := keyProperties[0]
		if val, ok := entity[key]; ok {
			return formatKeyValue(val), nil
		}
		return "", fmt.Errorf("key property %s not found", key)
	}
	
	// Composite key
	var keyParts []string
	for _, key := range keyProperties {
		if val, ok := entity[key]; ok {
			keyParts = append(keyParts, fmt.Sprintf("%s=%s", key, formatKeyValue(val)))
		} else {
			return "", fmt.Errorf("key property %s not found", key)
		}
	}
	return strings.Join(keyParts, ","), nil
}

// formatKeyValue formats a key value for use in URLs
func formatKeyValue(val interface{}) string {
	switch v := val.(type) {
	case string:
		// String keys need to be quoted
		return fmt.Sprintf("'%s'", v)
	case float64:
		// Check if it's actually an integer
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%v", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}