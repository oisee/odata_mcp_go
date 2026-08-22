// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"encoding/json"
	"fmt"

	"github.com/zmcp/odata-mcp/internal/constants"
	"github.com/zmcp/odata-mcp/internal/models"
	"github.com/zmcp/odata-mcp/internal/utils"
)

// enhanceResponse enhances OData response based on configuration options
func (b *ODataMCPBridge) enhanceResponse(response *models.ODataResponse, options map[string]string) *models.ODataResponse {
	enhanced := &models.ODataResponse{
		Context:  response.Context,
		Count:    response.Count,
		NextLink: response.NextLink,
		Value:    response.Value,
		Error:    response.Error,
		Metadata: response.Metadata,
	}

	// Apply size limits first to prevent large responses
	enhanced = b.applySizeLimits(enhanced)

	// Add pagination hints if enabled
	if b.config.PaginationHints && response.Value != nil {
		pagination := &models.PaginationInfo{}

		// Set total count if available
		if response.Count != nil {
			pagination.TotalCount = response.Count
		}

		// Calculate current count
		if resultArray, ok := response.Value.([]any); ok {
			pagination.CurrentCount = len(resultArray)
		} else {
			pagination.CurrentCount = 1 // Single entity
		}

		// Parse skip and top from options
		skip := 0
		top := 0
		if skipStr, exists := options[constants.QuerySkip]; exists {
			_, _ = fmt.Sscanf(skipStr, "%d", &skip)
		}
		if topStr, exists := options[constants.QueryTop]; exists {
			_, _ = fmt.Sscanf(topStr, "%d", &top)
		}

		pagination.Skip = skip
		pagination.Top = top

		// Determine if there are more results
		if pagination.TotalCount != nil && top > 0 {
			pagination.HasMore = int64(skip+pagination.CurrentCount) < *pagination.TotalCount

			// Generate suggested next call if there are more results
			if pagination.HasMore {
				nextSkip := skip + pagination.CurrentCount
				suggestedCall := fmt.Sprintf("Use $skip=%d and $top=%d for next page", nextSkip, top)
				pagination.SuggestedNextCall = &suggestedCall
			}
		}

		enhanced.Pagination = pagination
	}

	// Process legacy dates if enabled
	if b.config.LegacyDates {
		enhanced.Value = b.convertLegacyDates(enhanced.Value)
	}

	// Strip metadata if not requested
	if !b.config.ResponseMetadata {
		enhanced.Value = b.stripMetadata(enhanced.Value)
	}

	return enhanced
}

// applySizeLimits enforces response size and item count limits
func (b *ODataMCPBridge) applySizeLimits(response *models.ODataResponse) *models.ODataResponse {
	if response.Value == nil {
		return response
	}

	// Apply item count limit
	if b.config.MaxItems > 0 {
		if resultArray, ok := response.Value.([]any); ok {
			if len(resultArray) > b.config.MaxItems {
				// Truncate to max items and add warning
				truncated := resultArray[:b.config.MaxItems]

				// Update response
				newResponse := &models.ODataResponse{
					Context:  response.Context,
					Count:    response.Count,
					NextLink: response.NextLink,
					Value:    truncated,
					Error:    response.Error,
					Metadata: response.Metadata,
				}

				// Add truncation warning
				if newResponse.Metadata == nil {
					newResponse.Metadata = make(map[string]any)
				}
				newResponse.Metadata["truncated"] = true
				newResponse.Metadata["original_count"] = len(resultArray)
				newResponse.Metadata["max_items"] = b.config.MaxItems
				newResponse.Metadata["warning"] = fmt.Sprintf("Response truncated from %d to %d items due to size limits", len(resultArray), b.config.MaxItems)

				return newResponse
			}
		}
	}

	// Apply response size limit
	if b.config.MaxResponseSize > 0 {
		// Estimate response size by marshaling to JSON
		jsonData, err := json.Marshal(response.Value)
		if err == nil && len(jsonData) > b.config.MaxResponseSize {
			// If it's an array, try to reduce items
			if resultArray, ok := response.Value.([]any); ok {
				if len(resultArray) == 0 {
					return response
				}

				// Calculate how many items we can fit
				avgItemSize := len(jsonData) / len(resultArray)
				if avgItemSize == 0 {
					return response
				}
				maxItems := b.config.MaxResponseSize / avgItemSize
				if maxItems < 1 {
					maxItems = 1
				}

				// Truncate to fit size limit
				truncated := resultArray[:maxItems]

				// Update response
				newResponse := &models.ODataResponse{
					Context:  response.Context,
					Count:    response.Count,
					NextLink: response.NextLink,
					Value:    truncated,
					Error:    response.Error,
					Metadata: response.Metadata,
				}

				// Add truncation warning
				if newResponse.Metadata == nil {
					newResponse.Metadata = make(map[string]any)
				}
				newResponse.Metadata["truncated"] = true
				newResponse.Metadata["original_count"] = len(resultArray)
				newResponse.Metadata["truncated_count"] = len(truncated)
				newResponse.Metadata["max_response_size"] = b.config.MaxResponseSize
				newResponse.Metadata["warning"] = fmt.Sprintf("Response truncated from %d to %d items due to response size limit (%d bytes)", len(resultArray), len(truncated), b.config.MaxResponseSize)

				return newResponse
			}
		}
	}

	return response
}

// convertLegacyDates converts date fields to epoch timestamp format (/Date(1234567890000)/)
func (b *ODataMCPBridge) convertLegacyDates(data any) any {
	if !b.config.LegacyDates {
		return data
	}

	// Convert from OData legacy format to ISO for display
	return utils.ConvertDatesInResponse(data, true)
}

// stripMetadata removes __metadata blocks from entities unless specifically requested
func (b *ODataMCPBridge) stripMetadata(data any) any {
	switch v := data.(type) {
	case []any:
		// Handle array of entities
		result := make([]any, len(v))
		for i, item := range v {
			result[i] = b.stripMetadata(item)
		}
		return result
	case map[string]any:
		// Handle single entity
		result := make(map[string]any)
		for key, value := range v {
			if key != "__metadata" {
				result[key] = b.stripMetadata(value)
			}
		}
		return result
	default:
		return data
	}
}
