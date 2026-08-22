// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zmcp/odata-mcp/internal/client"
)

// TestErrorResponseTiming verifies that OData errors are returned immediately
// without waiting for timeout (addresses issues #17 and #19)
func TestErrorResponseTiming(t *testing.T) {
	tests := []struct {
		name           string
		statusCode     int
		responseBody   string
		expectedErrMsg string
	}{
		{
			name:           "404 Not Found - immediate response",
			statusCode:     http.StatusNotFound,
			responseBody:   `{"error":{"code":"404","message":"Resource not found for the segment 'InvalidEntity'"}}`,
			expectedErrMsg: "Resource not found",
		},
		{
			name:           "400 Bad Request - immediate response",
			statusCode:     http.StatusBadRequest,
			responseBody:   `{"error":{"code":"400","message":"Invalid filter expression"}}`,
			expectedErrMsg: "Invalid filter",
		},
		{
			name:           "500 Internal Server Error - immediate response",
			statusCode:     http.StatusInternalServerError,
			responseBody:   `{"error":{"code":"500","message":"Internal server error occurred"}}`,
			expectedErrMsg: "Internal server error",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create mock server that returns error immediately
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				// Return error immediately without delay
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(tc.statusCode)
				w.Write([]byte(tc.responseBody))
			}))
			defer server.Close()

			// Create client
			odataClient := client.NewODataClient(server.URL, false)

			// Measure time for error response
			ctx := context.Background()
			start := time.Now()

			_, err := odataClient.GetEntitySet(ctx, "TestEntity", nil)

			elapsed := time.Since(start)

			// Verify error was returned
			if err == nil {
				t.Fatal("Expected error but got nil")
			}

			// Verify error message contains expected text
			if tc.expectedErrMsg != "" && !containsIgnoreCase(err.Error(), tc.expectedErrMsg) {
				t.Errorf("Error message %q should contain %q", err.Error(), tc.expectedErrMsg)
			}

			// Verify response was immediate (less than 1 second)
			// If there's a timeout issue, this would take 30+ seconds
			if elapsed > 2*time.Second {
				t.Errorf("Error response took %v, expected immediate response (< 2s)", elapsed)
			}

			t.Logf("Error response received in %v: %v", elapsed, err)
		})
	}
}

// TestErrorResponseContent verifies that actual SAP error messages are preserved
// (addresses issue #19 - error message should not be lost)
func TestErrorResponseContent(t *testing.T) {
	sapErrorBody := `{
		"error": {
			"code": "SY/530",
			"message": {
				"lang": "en",
				"value": "Resource not found for the segment 'Materialbycustomer'"
			},
			"innererror": {
				"transactionid": "ABC123",
				"timestamp": "20240101120000"
			}
		}
	}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(sapErrorBody))
	}))
	defer server.Close()

	odataClient := client.NewODataClient(server.URL, true) // Enable verbose to get inner error

	ctx := context.Background()
	_, err := odataClient.GetEntitySet(ctx, "Materialbycustomer", nil)

	if err == nil {
		t.Fatal("Expected error but got nil")
	}

	// The error message should contain the actual SAP error, not a generic timeout
	errStr := err.Error()

	// Should NOT contain timeout message
	if containsIgnoreCase(errStr, "timeout") || containsIgnoreCase(errStr, "timed out") {
		t.Errorf("Error should not be timeout, got: %v", err)
	}

	// Should contain actual error details
	if !containsIgnoreCase(errStr, "Materialbycustomer") && !containsIgnoreCase(errStr, "Resource not found") {
		t.Errorf("Error should contain SAP error message, got: %v", err)
	}

	t.Logf("SAP error properly propagated: %v", err)
}

func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) && findSubstringIgnoreCase(s, substr)
}

func findSubstringIgnoreCase(s, substr string) bool {
	sLower := stringToLower(s)
	substrLower := stringToLower(substr)
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		if sLower[i:i+len(substrLower)] == substrLower {
			return true
		}
	}
	return false
}

func stringToLower(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
