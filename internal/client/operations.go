// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/models"
)

// GetEntitySet retrieves entities from an entity set
func (c *ODataClient) GetEntitySet(ctx context.Context, entitySet string, options map[string]string) (*models.ODataResponse, error) {
	endpoint := entitySet

	// Build query parameters with standard OData v2 parameters
	params := url.Values{}

	// Always add JSON format for consistent responses (v2 only)
	if !c.isV4 {
		params.Add(constants.QueryFormat, "json")
	}

	// Add inline count for pagination support unless explicitly requesting count only
	// OData v4 uses $count=true instead of $inlinecount
	if !c.isV4 {
		if _, hasInlineCount := options[constants.QueryInlineCount]; !hasInlineCount {
			params.Add(constants.QueryInlineCount, "allpages")
		}
	}

	// Add user-provided parameters
	for key, value := range options {
		if value != "" {
			// Handle v2 to v4 query parameter translation
			if c.isV4 && key == constants.QueryInlineCount {
				// Translate $inlinecount to $count for v4
				if value == "allpages" {
					params.Set(constants.QueryCount, "true")
				} else if value == "none" {
					params.Set(constants.QueryCount, "false")
				}
				// Skip adding $inlinecount for v4
				continue
			}
			params.Set(key, value) // Use Set to override defaults if needed
		}
	}

	if len(params) > 0 {
		endpoint += "?" + encodeQueryParams(params)
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
			endpoint += "?" + encodeQueryParams(params)
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
