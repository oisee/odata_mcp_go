// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package client

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/metadata"
	"github.com/zmcp/odata-mcp/internal/models"
)

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
	meta, err := c.parseMetadataXML(body)
	if err != nil {
		// Fallback to service document if metadata parsing fails
		return c.getServiceDocument(ctx)
	}

	return meta, nil
}

// parseMetadataXML parses OData metadata XML
func (c *ODataClient) parseMetadataXML(data []byte) (*models.ODataMetadata, error) {
	meta, err := metadata.ParseMetadata(data, c.baseURL)
	if err != nil {
		return nil, err
	}

	// Set the client's v4 flag based on metadata version
	c.isV4 = meta.Version == "4.0" || meta.Version == "4.01"

	return meta, nil
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
	meta := &models.ODataMetadata{
		ServiceRoot:     c.baseURL,
		EntityTypes:     make(map[string]*models.EntityType),
		EntitySets:      make(map[string]*models.EntitySet),
		FunctionImports: make(map[string]*models.FunctionImport),
		Version:         "2.0",
		ParsedAt:        time.Now(),
	}

	return meta, nil
}
