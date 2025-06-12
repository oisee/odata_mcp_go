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
	baseURL    string
	httpClient *http.Client
	cookies    map[string]string
	username   string
	password   string
	csrfToken  string
	verbose    bool
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

	// Set CSRF token if available
	if c.csrfToken != "" {
		req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
	}

	return req, nil
}

// doRequest executes an HTTP request and handles common errors
func (c *ODataClient) doRequest(req *http.Request) (*http.Response, error) {
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] %s %s\n", req.Method, req.URL.String())
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}

	// Handle CSRF token requirement (SAP-specific)
	if resp.StatusCode == http.StatusForbidden && c.csrfToken == "" {
		resp.Body.Close()
		
		// Try to fetch CSRF token
		if err := c.fetchCSRFToken(req.Context()); err != nil {
			return nil, fmt.Errorf("failed to fetch CSRF token: %w", err)
		}

		// Retry original request with CSRF token
		req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
		return c.doRequest(req)
	}

	return resp, nil
}

// fetchCSRFToken fetches a CSRF token from the service
func (c *ODataClient) fetchCSRFToken(ctx context.Context) error {
	req, err := c.buildRequest(ctx, constants.GET, "", nil)
	if err != nil {
		return err
	}

	req.Header.Set(constants.CSRFTokenHeader, constants.CSRFTokenFetch)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("CSRF token request failed: %w", err)
	}
	defer resp.Body.Close()

	token := resp.Header.Get(constants.CSRFTokenHeader)
	if token == "" {
		token = resp.Header.Get(constants.CSRFTokenHeaderLower)
	}

	if token == "" {
		return fmt.Errorf("CSRF token not found in response headers")
	}

	c.csrfToken = token
	if c.verbose {
		fmt.Fprintf(os.Stderr, "[VERBOSE] CSRF token fetched successfully\n")
	}

	return nil
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
	
	// Build query parameters
	if len(options) > 0 {
		params := url.Values{}
		for key, value := range options {
			if value != "" {
				params.Add(key, value)
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
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity data: %w", err)
	}

	req, err := c.buildRequest(ctx, constants.POST, entitySet, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.ContentType, constants.ContentTypeJSON)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// UpdateEntity updates an existing entity
func (c *ODataClient) UpdateEntity(ctx context.Context, entitySet string, key map[string]interface{}, data map[string]interface{}, method string) (*models.ODataResponse, error) {
	keyPredicate := c.buildKeyPredicate(key)
	endpoint := fmt.Sprintf("%s(%s)", entitySet, keyPredicate)

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal entity data: %w", err)
	}

	if method == "" {
		method = constants.PUT
	}

	req, err := c.buildRequest(ctx, method, endpoint, bytes.NewReader(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set(constants.ContentType, constants.ContentTypeJSON)

	resp, err := c.doRequest(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return c.parseODataResponse(resp)
}

// DeleteEntity deletes an entity
func (c *ODataClient) DeleteEntity(ctx context.Context, entitySet string, key map[string]interface{}) (*models.ODataResponse, error) {
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
		// For GET requests, add parameters to URL
		if len(parameters) > 0 {
			params := url.Values{}
			for key, value := range parameters {
				params.Add(key, fmt.Sprintf("%v", value))
			}
			endpoint += "?" + params.Encode()
		}
		req, err = c.buildRequest(ctx, constants.GET, endpoint, nil)
	} else {
		// For POST requests, send parameters in body
		jsonData, marshalErr := json.Marshal(parameters)
		if marshalErr != nil {
			return nil, fmt.Errorf("failed to marshal function parameters: %w", marshalErr)
		}
		req, err = c.buildRequest(ctx, constants.POST, endpoint, bytes.NewReader(jsonData))
		if err == nil {
			req.Header.Set(constants.ContentType, constants.ContentTypeJSON)
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
		return fmt.Sprintf("'%s'", url.QueryEscape(v))
	case int, int32, int64:
		return fmt.Sprintf("%d", v)
	case float32, float64:
		return fmt.Sprintf("%g", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("'%s'", url.QueryEscape(fmt.Sprintf("%v", v)))
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

	var odataResp models.ODataResponse
	if err := json.Unmarshal(body, &odataResp); err != nil {
		return nil, fmt.Errorf("failed to parse OData response: %w", err)
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
		return fmt.Errorf("OData error: %s", errorResp.Error.Message)
	}

	// Fallback to generic error
	return fmt.Errorf("HTTP %d: %s", statusCode, string(body))
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
	// TODO: Implement service document parsing as fallback
	// This would be used when metadata parsing fails
	return nil, fmt.Errorf("service document parsing not yet implemented")
}