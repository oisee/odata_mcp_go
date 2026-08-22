package client

import (
	"encoding/json"
	"fmt"
)

// parseODataResponse parses OData responses, handling both v2 and v4 formats
// Note: Error responses should be handled by the caller before calling this function
// (check HTTP status code and use parseErrorFromBody in response.go)
func parseODataResponse(data []byte, isV4 bool) (any, error) {
	// Try to parse as a generic map first
	var rawResponse map[string]any
	if err := json.Unmarshal(data, &rawResponse); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	if isV4 {
		return parseV4Response(rawResponse), nil
	}
	return parseV2Response(rawResponse), nil
}

// parseV2Response handles OData v2 response format
func parseV2Response(response map[string]any) any {
	// OData v2 wraps results in a "d" property
	if d, ok := response["d"]; ok {
		if dMap, ok := d.(map[string]any); ok {
			// Check if it's a collection
			if results, ok := dMap["results"]; ok {
				normalized := map[string]any{
					"value": results,
				}
				// Include count if present
				if count, ok := dMap["__count"]; ok {
					normalized["@odata.count"] = count
				}
				// Include next link if present
				if next, ok := dMap["__next"]; ok {
					normalized["@odata.nextLink"] = next
				}
				return normalized
			}
			// Single entity
			return d
		}
		return d
	}
	return response
}

// parseV4Response handles OData v4 response format
func parseV4Response(response map[string]any) any {
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

