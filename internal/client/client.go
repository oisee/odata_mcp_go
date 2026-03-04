// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"net/http"
	"strings"
	"time"

	"github.com/icholy/digest"
	"github.com/zmcp/odata-mcp/internal/constants"
)

// Context key for HTTP headers passed from MCP server
type contextKey string

const HTTPHeadersContextKey contextKey = "mcp-http-headers"

// ODataClient handles HTTP communication with OData services
type ODataClient struct {
	baseURL        string
	httpClient     *http.Client
	cookies        map[string]string
	username       string
	password       string
	authType       string // "basic" or "digest"
	csrfToken      string
	verbose        bool
	sessionCookies []*http.Cookie // Track session cookies from server
	isV4           bool           // Whether the service is OData v4
}

// NewODataClient creates a new OData client
func NewODataClient(baseURL string, verbose bool) *ODataClient {
	// Ensure base URL ends with /
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}

	return &ODataClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: time.Duration(constants.DefaultTimeout) * time.Second,
		},
		verbose: verbose,
		isV4:    false, // Will be determined when fetching metadata
	}
}

// SetBasicAuth configures basic authentication
func (c *ODataClient) SetBasicAuth(username, password string) {
	c.username = username
	c.password = password
}

// SetDigestAuth configures HTTP Digest authentication
func (c *ODataClient) SetDigestAuth(username, password string) {
	c.username = username
	c.password = password
	c.authType = "digest"
	c.httpClient.Transport = &digest.Transport{
		Username: username,
		Password: password,
	}
}

// SetCookies configures cookie authentication
func (c *ODataClient) SetCookies(cookies map[string]string) {
	c.cookies = cookies
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// shouldForwardHeader determines if a header should be forwarded from MCP to OData service
func shouldForwardHeader(headerName string) bool {
	// Normalize to lowercase for comparison
	lower := strings.ToLower(headerName)

	// Allow authentication headers
	if lower == "authorization" || lower == "cookie" {
		return true
	}

	// Allow custom headers (X- prefix)
	if strings.HasPrefix(lower, "x-") {
		return true
	}

	// Block hop-by-hop headers and other problematic headers
	blockedHeaders := []string{
		"host",
		"connection",
		"keep-alive",
		"transfer-encoding",
		"upgrade",
		"proxy-authenticate",
		"proxy-authorization",
		"te",
		"trailer",
		"content-length", // Will be set by http.Client
		"content-type",   // Set by specific methods
		"accept",         // Set by buildRequest
		"user-agent",     // Set by buildRequest
	}

	for _, blocked := range blockedHeaders {
		if lower == blocked {
			return false
		}
	}

	// Allow other headers by default
	return true
}
