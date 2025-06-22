package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/odata-mcp/go/internal/constants"
	"github.com/odata-mcp/go/internal/metadata"
	"github.com/odata-mcp/go/internal/models"
)

// ODataClient handles HTTP communication with OData services
type ODataClient struct {
	baseURL       string
	httpClient    *http.Client
	cookies       map[string]string
	username      string
	password      string
	csrfToken     string
	verbose       bool
	sessionCookies []*http.Cookie // Track session cookies from server
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
	}
}

// SetBasicAuth configures basic authentication
func (c *ODataClient) SetBasicAuth(username, password string) {
	c.username = username
	c.password = password
}

// SetCookies configures cookie authentication
func (c *ODataClient) SetCookies(cookies map[string]string) {
	c.cookies = cookies
}

// buildRequest creates an HTTP request with proper headers and authentication
func (c *ODataClient) buildRequest(ctx context.Context, method, endpoint string, body io.Reader) (*http.Request, error) {
	fullURL := c.baseURL + strings.TrimPrefix(endpoint, "/")
	
	req, err := http.NewRequestWithContext(ctx, method, fullURL, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set standard headers
	req.Header.Set(constants.UserAgent, constants.DefaultUserAgent)
	req.Header.Set(constants.Accept, constants.ContentTypeJSON)

	// Set authentication
	if c.username != "" && c.password != "" {
		req.SetBasicAuth(c.username, c.password)
	}

	// Set cookies
	for name, value := range c.cookies {
		req.AddCookie(&http.Cookie{
			Name:  name,
			Value: value,
		})
	}
	
	// Add session cookies received from server
	for _, cookie := range c.sessionCookies {
		req.AddCookie(cookie)
	}

	// Set CSRF token if available
	if c.csrfToken != "" {
		req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
		if c.verbose {
			// Show first 20 chars of token like Python does
			tokenPreview := c.csrfToken
			if len(tokenPreview) > 20 {
				tokenPreview = tokenPreview[:20] + "..."
			}
			fmt.Fprintf(os.Stderr, "[VERBOSE] Adding CSRF token to request: %s\n", tokenPreview)
		}
	}

	return req, nil
}

// doRequest executes an HTTP request and handles common errors
func (c *ODataClient) doRequest(req *http.Request) (*http.Response, error) {
	// For requests with body, we need to save it for potential retry
	var bodyBytes []byte
	if req.Body != nil && req.ContentLength > 0 {
		var err error
		bodyBytes, err = io.ReadAll(req.Body)
		if err != nil {
			return nil, fmt.Errorf("failed to read request body: %w", err)
		}
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
	}
	
	return c.doRequestWithRetry(req, bodyBytes, false)
}

// doRequestWithRetry executes an HTTP request with CSRF retry logic
func (c *ODataClient) doRequestWithRetry(req *http.Request, bodyBytes []byte, isRetry bool) (*http.Response, error) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] %s %s\n", req.Method, req.URL.String())
	}

	// Reset body if we have it (for retry scenarios)
	if bodyBytes != nil && len(bodyBytes) > 0 {
		req.Body = io.NopCloser(bytes.NewReader(bodyBytes))
		req.ContentLength = int64(len(bodyBytes))
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Check if this is a modifying operation
	modifyingMethods := []string{"POST", "PUT", "MERGE", "PATCH", "DELETE"}
	isModifying := false
	for _, m := range modifyingMethods {
		if req.Method == m {
			isModifying = true
			break
		}
	}

	// Handle CSRF token validation failure (Python-style)
	if resp.StatusCode == http.StatusForbidden && isModifying && !isRetry {
		// Read response body to check for CSRF-related errors
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		bodyStr := string(body)
		
		csrfFailed := strings.Contains(bodyStr, "CSRF token validation failed") ||
			strings.Contains(strings.ToLower(bodyStr), "csrf") ||
			strings.EqualFold(resp.Header.Get("x-csrf-token"), "required")
		
		if csrfFailed {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token validation failed, attempting to refetch...\n")
			}
			
			// Clear the invalid token
			c.csrfToken = ""
			
			// Try to fetch new CSRF token
			if err := c.fetchCSRFToken(req.Context()); err != nil {
				// Return original error with CSRF context
				return nil, fmt.Errorf("CSRF token required but refetch failed. Status: %d. Response: %s", resp.StatusCode, bodyStr)
			}

			// Retry original request with new CSRF token
			req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Retrying request with new CSRF token...\n")
			}
			return c.doRequestWithRetry(req, bodyBytes, true)
		}
		
		// Not a CSRF error, recreate response with body
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	return resp, nil
}

// fetchCSRFToken fetches a CSRF token from the service
func (c *ODataClient) fetchCSRFToken(ctx context.Context) error {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Fetching CSRF token...\n")
	}
	
	// Clear any existing CSRF token (Python behavior)
	c.csrfToken = ""
	
	// Use service root for CSRF token fetching (more reliable than empty string)
	req, err := c.buildRequest(ctx, constants.GET, "", nil)
	if err != nil {
		return err
	}

	req.Header.Set(constants.CSRFTokenHeader, constants.CSRFTokenFetch)
	
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch request: %s %s\n", req.Method, req.URL.String())
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch headers: %v\n", req.Header)
	}

	// Don't use doRequest here to avoid retry loops - fetch token requests shouldn't retry
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CSRF token request failed: %w", err)
	}
	defer resp.Body.Close()
	
	// Store any session cookies from the response
	if cookies := resp.Cookies(); len(cookies) > 0 {
		c.sessionCookies = append(c.sessionCookies, cookies...)
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Received %d session cookies during token fetch\n", len(cookies))
			for _, cookie := range cookies {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Cookie: %s=%s (Path=%s)\n", cookie.Name, cookie.Value[:min(len(cookie.Value), 20)]+"...", cookie.Path)
			}
		}
	}
	
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch response status: %d\n", resp.StatusCode)
		fmt.Fprintf(os.Stderr, "[VERBOSE] Token fetch response headers: %v\n", resp.Header)
	}

	// Check both possible header names (case variations)
	token := resp.Header.Get(constants.CSRFTokenHeader)
	if token == "" {
		token = resp.Header.Get(constants.CSRFTokenHeaderLower)
	}

	// Additional header variations that some SAP systems use
	if token == "" {
		token = resp.Header.Get("x-csrf-token")
	}
	if token == "" {
		token = resp.Header.Get("X-Csrf-Token")
	}

	if token == "" || token == constants.CSRFTokenFetch {
		return fmt.Errorf("CSRF token not found in response headers")
	}

	c.csrfToken = token
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token fetched successfully: %s...\n", token[:min(len(token), 20)])
	}

	return nil
}

// Helper function for min
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// GetMetadata fetches and parses the OData service metadata
func (c *ODataClient) GetMetadata(ctx context.Context) (*models.ODataMetadata, error) {
	req, err := c.buildRequest(ctx, constants.GET, constants.MetadataEndpoint, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.Accept, constants.ContentTypeXML)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read metadata response: %w", err)
	}

	// Parse metadata XML (to be implemented)
	metadata, err := c.parseMetadataXML(body)
	if err != nil {
		// Fallback to service document if metadata parsing fails
		return c.getServiceDocument(ctx)
	}

	return metadata, nil
}

// GetEntitySet retrieves entities from an entity set
func (c *ODataClient) GetEntitySet(ctx context.Context, entitySet string, options map[string]string) (*models.ODataResponse, error) {
	endpoint := entitySet
	
	// Build query parameters with standard OData v2 parameters
	params := url.Values{}
	
	// Always add JSON format for consistent responses
	params.Add(constants.QueryFormat, "json")
	
	// Add inline count for pagination support unless explicitly requesting count only
	if _, hasInlineCount := options[constants.QueryInlineCount]; !hasInlineCount {
		params.Add(constants.QueryInlineCount, "allpages")
	}
	
	// Add user-provided parameters
	for key, value := range options {
		if value != "" {
			params.Set(key, value) // Use Set to override defaults if needed
		}
	}
	
	if len(params) > 0 {
		endpoint += "?" + params.Encode()
	}

	req, err := c.buildRequest(ctx, constants.GET, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// GetEntity retrieves a single entity by key
func (c *ODataClient) GetEntity(ctx context.Context, entitySet string, key map[string]interface{}, options map[string]string) (*models.ODataResponse, error) {
	// Build key predicate
	keyPredicate := c.buildKeyPredicate(key)
	endpoint := fmt.Sprintf("%s(%s)", entitySet, keyPredicate)

	// Build query parameters
	if len(options) > 0 {
		params := url.Values{}
		for k, v := range options {
			if v != "" {
				params.Add(k, v)
			}
		}
		if len(params) > 0 {
			endpoint += "?" + params.Encode()
		}
	}

	req, err := c.buildRequest(ctx, constants.GET, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// CreateEntity creates a new entity
func (c *ODataClient) CreateEntity(ctx context.Context, entitySet string, data map[string]interface{}) (*models.ODataResponse, error) {
	// Always fetch a fresh CSRF token for modifying operations (Python behavior)
	if err := c.fetchCSRFToken(ctx); err != nil {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to fetch CSRF token, proceeding without it: %v\n", err)
		}
		// Continue without token - some services might not require it
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity data: %w", err)
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Creating entity with data: %s\n", string(jsonData))
	}

	req, err := c.buildRequest(ctx, constants.POST, entitySet, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.ContentType, constants.ContentTypeJSON)
	// Explicitly set content length to avoid any body length issues
	req.ContentLength = int64(len(jsonData))

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// UpdateEntity updates an existing entity
func (c *ODataClient) UpdateEntity(ctx context.Context, entitySet string, key map[string]interface{}, data map[string]interface{}, method string) (*models.ODataResponse, error) {
	// Always fetch a fresh CSRF token for modifying operations (Python behavior)
	if err := c.fetchCSRFToken(ctx); err != nil {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to fetch CSRF token, proceeding without it: %v\n", err)
		}
		// Continue without token - some services might not require it
	}

	keyPredicate := c.buildKeyPredicate(key)
	endpoint := fmt.Sprintf("%s(%s)", entitySet, keyPredicate)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity data: %w", err)
	}

	if method == "" {
		method = constants.PUT
	}

	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Updating entity with data: %s\n", string(jsonData))
	}

	req, err := c.buildRequest(ctx, method, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.ContentType, constants.ContentTypeJSON)
	// Explicitly set content length to avoid any body length issues
	req.ContentLength = int64(len(jsonData))

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// DeleteEntity deletes an entity
func (c *ODataClient) DeleteEntity(ctx context.Context, entitySet string, key map[string]interface{}) (*models.ODataResponse, error) {
	// Always fetch a fresh CSRF token for modifying operations (Python behavior)
	if err := c.fetchCSRFToken(ctx); err != nil {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to fetch CSRF token, proceeding without it: %v\n", err)
		}
		// Continue without token - some services might not require it
	}

	keyPredicate := c.buildKeyPredicate(key)
	endpoint := fmt.Sprintf("%s(%s)", entitySet, keyPredicate)

	req, err := c.buildRequest(ctx, constants.DELETE, endpoint, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// CallFunction calls a function import
func (c *ODataClient) CallFunction(ctx context.Context, functionName string, parameters map[string]interface{}, method string) (*models.ODataResponse, error) {
	endpoint := functionName

	var req *http.Request
	var err error

	if method == constants.GET {
		// For GET requests, add parameters to URL with proper OData formatting
		if len(parameters) > 0 {
			var paramStrings []string
			for key, value := range parameters {
				paramStrings = append(paramStrings, c.formatFunctionParameter(key, value))
			}
			endpoint += "?" + strings.Join(paramStrings, "&")
		}
		req, err = c.buildRequest(ctx, constants.GET, endpoint, nil)
	} else {
		// Always fetch a fresh CSRF token for modifying operations (Python behavior)
		if err := c.fetchCSRFToken(ctx); err != nil {
			if c.verbose {
				fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to fetch CSRF token, proceeding without it: %v\n", err)
			}
			// Continue without token - some services might not require it
		}

		// For POST requests, send parameters in body
		jsonData, marshalErr := json.Marshal(parameters)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal function parameters: %w", marshalErr)
		}

		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Calling function with data: %s\n", string(jsonData))
		}

		req, err = c.buildRequest(ctx, constants.POST, endpoint, bytes.NewReader(jsonData))
		if err == nil {
			req.Header.Set(constants.ContentType, constants.ContentTypeJSON)
			// Explicitly set content length to avoid any body length issues
			req.ContentLength = int64(len(jsonData))
		}
	}

	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// buildKeyPredicate builds OData key predicate from key-value pairs
func (c *ODataClient) buildKeyPredicate(key map[string]interface{}) string {
	if len(key) == 1 {
		// Single key
		for _, value := range key {
			return c.formatKeyValue(value)
		}
	}

	// Composite key
	var parts []string
	for k, v := range key {
		parts = append(parts, fmt.Sprintf("%s=%s", k, c.formatKeyValue(v)))
	}
	return strings.Join(parts, ",")
}

// formatKeyValue formats a key value for OData URL
func (c *ODataClient) formatKeyValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		// For key predicates, don't URL encode the value inside quotes
		// URL encoding happens at the full URL level
		return fmt.Sprintf("'%s'", v)
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("'%s'", fmt.Sprintf("%v", v))
	}
}

// formatFunctionParameter formats a function parameter for OData URL
func (c *ODataClient) formatFunctionParameter(key string, value interface{}) string {
	switch v := value.(type) {
	case string:
		// OData requires string parameters to be single-quoted
		// URL encode the value but not the quotes
		return fmt.Sprintf("%s='%s'", key, url.QueryEscape(v))
	case int, int32, int64:
		return fmt.Sprintf("%s=%d", key, v)
	case float32, float64:
		return fmt.Sprintf("%s=%g", key, v)
	case bool:
		return fmt.Sprintf("%s=%t", key, v)
	default:
		// Default to string representation with quotes
		return fmt.Sprintf("%s='%s'", key, url.QueryEscape(fmt.Sprintf("%v", v)))
	}
}

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

	// OData v2 typically wraps results in a "d" property
	var wrapper struct {
		D json.RawMessage `json:"d"`
	}
	
	if err := json.Unmarshal(body, &wrapper); err != nil {
		// Try direct parsing if no wrapper
		var odataResp models.ODataResponse
		if err := json.Unmarshal(body, &odataResp); err != nil {
			return nil, fmt.Errorf("failed to parse OData response: %w", err)
		}
		c.optimizeResponse(&odataResp)
		return &odataResp, nil
	}

	// Parse the wrapped response
	var odataResp models.ODataResponse
	if len(wrapper.D) > 0 {
		if c.verbose {
			fmt.Fprintf(os.Stderr, "[VERBOSE] Wrapped content: %s\n", string(wrapper.D))
		}
		
		// OData v2 responses typically have a structure like:
		// { "d": { "results": [...], "__count": "N" } } for collections
		// { "d": { ...entity properties... } } for single entities
		
		// First check if it's a collection response
		var collectionCheck struct {
			Results json.RawMessage `json:"results"`
		}
		if err := json.Unmarshal(wrapper.D, &collectionCheck); err == nil && len(collectionCheck.Results) > 0 {
			// It's a collection - parse as such
			var collection struct {
				Results []json.RawMessage `json:"results"`
				Count   string           `json:"__count,omitempty"`
			}
			if err := json.Unmarshal(wrapper.D, &collection); err != nil {
				return nil, fmt.Errorf("failed to parse collection response: %w", err)
			}
			
			// Convert raw messages to interface{}
			entities := make([]interface{}, len(collection.Results))
			for i, raw := range collection.Results {
				var entity interface{}
				if err := json.Unmarshal(raw, &entity); err != nil {
					return nil, fmt.Errorf("failed to parse entity %d: %w", i, err)
				}
				entities[i] = entity
			}
			odataResp.Value = entities
			
			if collection.Count != "" {
				var count int64
				fmt.Sscanf(collection.Count, "%d", &count)
				odataResp.Count = &count
			}
		} else {
			// It's a single entity - parse the entity directly
			var entity interface{}
			if err := json.Unmarshal(wrapper.D, &entity); err != nil {
				return nil, fmt.Errorf("failed to parse single entity response: %w", err)
			}
			// For single entities, put the entity directly in Value (not wrapped in array)
			odataResp.Value = entity
		}
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
	// Try to parse as JSON error
	var errorResp struct {
		Error *models.ODataError `json:"error"`
	}

	if err := json.Unmarshal(body, &errorResp); err == nil && errorResp.Error != nil {
		return c.buildDetailedError(errorResp.Error, statusCode, body)
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
	
	return fmt.Errorf(errMsg.String())
}

// optimizeResponse applies optimizations to the response
func (c *ODataClient) optimizeResponse(resp *models.ODataResponse) {
	// TODO: Implement GUID conversion and other optimizations
	// This would include the sophisticated response optimization logic
	// from the Python version
}

// parseMetadataXML parses OData metadata XML
func (c *ODataClient) parseMetadataXML(data []byte) (*models.ODataMetadata, error) {
	return metadata.ParseMetadata(data, c.baseURL)
}

// getServiceDocument gets the service document as fallback
func (c *ODataClient) getServiceDocument(ctx context.Context) (*models.ODataMetadata, error) {
	req, err := c.buildRequest(ctx, constants.GET, "", nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.Accept, constants.ContentTypeJSON)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, c.parseError(resp)
	}

	// For now, return a minimal metadata structure
	// In a full implementation, this would parse the service document
	metadata := &models.ODataMetadata{
		ServiceRoot:     c.baseURL,
		EntityTypes:     make(map[string]*models.EntityType),
		EntitySets:      make(map[string]*models.EntitySet),
		FunctionImports: make(map[string]*models.FunctionImport),
		Version:         "2.0",
		ParsedAt:        time.Now(),
	}

	return metadata, nil
}