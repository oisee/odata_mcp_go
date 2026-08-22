// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/zmcp/odata-mcp/internal/models"
)

// parseODataResponse parses an OData response
func (c *ODataClient) parseODataResponse(resp *http.Response) (*models.ODataResponse, error) {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, c.parseErrorFromBody(body, resp.StatusCode)
	}

	// Handle empty responses (e.g., from DELETE operations)
	if len(body) == 0 {
		return &models.ODataResponse{}, nil
	}

	// Log raw response for debugging
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Raw response: %s\n", string(body))
	}

	// Parse using the appropriate parser
	parsedResponse, err := parseODataResponse(body, c.isV4)
	if err != nil {
		return nil, err
	}

	// Convert to ODataResponse model
	var odataResp models.ODataResponse

	switch v := parsedResponse.(type) {
	case map[string]interface{}:
		// Check for v4 format
		if c.isV4 {
			// OData v4 format
			if value, ok := v["value"]; ok {
				odataResp.Value = value
			} else {
				// Single entity
				odataResp.Value = v
			}
			if count, ok := v["@odata.count"]; ok {
				switch c := count.(type) {
				case float64:
					countInt := int64(c)
					odataResp.Count = &countInt
				case string:
					// Handle string count (common in v2)
					var countInt int64
					if _, err := fmt.Sscanf(c, "%d", &countInt); err == nil {
						odataResp.Count = &countInt
					}
				}
			}
			if nextLink, ok := v["@odata.nextLink"]; ok {
				if nextLinkStr, ok := nextLink.(string); ok {
					odataResp.NextLink = nextLinkStr
				}
			}
			if context, ok := v["@odata.context"]; ok {
				if contextStr, ok := context.(string); ok {
					odataResp.Context = contextStr
				}
			}
		} else {
			// OData v2 format (already normalized by parseODataResponseBody)
			if value, ok := v["value"]; ok {
				odataResp.Value = value
			} else {
				// Single entity
				odataResp.Value = v
			}
			if count, ok := v["@odata.count"]; ok {
				switch c := count.(type) {
				case float64:
					countInt := int64(c)
					odataResp.Count = &countInt
				case string:
					// Handle string count (common in v2)
					var countInt int64
					if _, err := fmt.Sscanf(c, "%d", &countInt); err == nil {
						odataResp.Count = &countInt
					}
				}
			}
			if nextLink, ok := v["@odata.nextLink"]; ok {
				if nextLinkStr, ok := nextLink.(string); ok {
					odataResp.NextLink = nextLinkStr
				}
			}
		}
	default:
		// Direct value
		odataResp.Value = parsedResponse
	}

	// Process GUIDs if needed (to be implemented)
	c.optimizeResponse(&odataResp)

	return &odataResp, nil
}

// parseError parses error from HTTP response
func (c *ODataClient) parseError(resp *http.Response) error {
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("HTTP %d: failed to read error response", resp.StatusCode)
	}

	return c.parseErrorFromBody(body, resp.StatusCode)
}

// parseErrorFromBody parses error from response body
func (c *ODataClient) parseErrorFromBody(body []byte, statusCode int) error {
	// Try to parse as JSON error - handle both v2 (nested message) and v4 (flat message) formats
	var rawError struct {
		Error struct {
			Code    string      `json:"code"`
			Message interface{} `json:"message"` // Can be string (v4) or object (v2)
			Target  string      `json:"target,omitempty"`
			Details []struct {
				Code    string `json:"code,omitempty"`
				Message string `json:"message"`
				Target  string `json:"target,omitempty"`
			} `json:"details,omitempty"`
			InnerError map[string]interface{} `json:"innererror,omitempty"`
			Severity   string                 `json:"severity,omitempty"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &rawError); err == nil && rawError.Error.Code != "" {
		// Extract message - handle both formats
		var message string
		switch m := rawError.Error.Message.(type) {
		case string:
			// v4 format: "message": "error text"
			message = m
		case map[string]interface{}:
			// v2 format: "message": {"lang": "en", "value": "error text"}
			if val, ok := m["value"].(string); ok {
				message = val
			} else {
				// Fallback to string representation
				msgBytes, _ := json.Marshal(m)
				message = string(msgBytes)
			}
		default:
			message = fmt.Sprintf("%v", m)
		}

		// Build ODataError for detailed error handling
		odataErr := &models.ODataError{
			Code:       rawError.Error.Code,
			Message:    message,
			Target:     rawError.Error.Target,
			InnerError: rawError.Error.InnerError,
			Severity:   rawError.Error.Severity,
		}
		for _, d := range rawError.Error.Details {
			odataErr.Details = append(odataErr.Details, models.ODataErrorDetail{
				Code:    d.Code,
				Message: d.Message,
				Target:  d.Target,
			})
		}
		return c.buildDetailedError(odataErr, statusCode, body)
	}

	// Fallback to generic error
	return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
}

// buildDetailedError creates a comprehensive error message from OData error details
func (c *ODataClient) buildDetailedError(odataErr *models.ODataError, statusCode int, rawBody []byte) error {
	var errMsg strings.Builder

	// Start with basic error info
	errMsg.WriteString(fmt.Sprintf("OData error (HTTP %d)", statusCode))

	// Add error code if available
	if odataErr.Code != "" {
		errMsg.WriteString(fmt.Sprintf(" [%s]", odataErr.Code))
	}

	// Add main message
	errMsg.WriteString(fmt.Sprintf(": %s", odataErr.Message))

	// Add target if available (which field/entity caused the error)
	if odataErr.Target != "" {
		errMsg.WriteString(fmt.Sprintf(" (target: %s)", odataErr.Target))
	}

	// Add severity if available
	if odataErr.Severity != "" {
		errMsg.WriteString(fmt.Sprintf(" [severity: %s]", odataErr.Severity))
	}

	// Add details if available
	if len(odataErr.Details) > 0 {
		errMsg.WriteString(" | Details: ")
		for i, detail := range odataErr.Details {
			if i > 0 {
				errMsg.WriteString("; ")
			}
			errMsg.WriteString(detail.Message)
			if detail.Target != "" {
				errMsg.WriteString(fmt.Sprintf(" (target: %s)", detail.Target))
			}
		}
	}

	// Add inner error info if available and verbose mode is on
	if c.verbose && len(odataErr.InnerError) > 0 {
		errMsg.WriteString(" | Inner error: ")
		if innerErrBytes, err := json.Marshal(odataErr.InnerError); err == nil {
			errMsg.WriteString(string(innerErrBytes))
		}
	}

	return fmt.Errorf("%s", errMsg.String())
}

// optimizeResponse applies optimizations to the response
func (c *ODataClient) optimizeResponse(resp *models.ODataResponse) {
	// TODO: Implement GUID conversion and other optimizations
	// This would include the sophisticated response optimization logic
	// from the Python version
}
