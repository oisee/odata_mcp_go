// Copyright (c) 2024 OData MCP Contributors
// SPDX-License-Identifier: MIT

package bridge

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/zmcp/odata-mcp/internal/models"
)

// transformFilterForSAP transforms filter strings to handle SAP-specific GUID formatting
// SAP requires GUID values to be prefixed with 'guid' like: guid'069f2c5e-2738-1eeb-b7bd-cd0f34d2052d'
func (b *ODataMCPBridge) transformFilterForSAP(filter string, entitySetName string) string {
	// Only transform if we have metadata and it's a SAP service
	if b.metadata == nil || !b.isSAPService() {
		return filter
	}

	// Find the entity type for this entity set
	var entityType *models.EntityType
	for _, es := range b.metadata.EntitySets {
		if es.Name == entitySetName {
			// Find the corresponding entity type
			for _, et := range b.metadata.EntityTypes {
				if et.Name == es.EntityType {
					entityType = et
					break
				}
			}
			break
		}
	}

	if entityType == nil {
		return filter
	}

	// Build a map of GUID properties
	guidProperties := make(map[string]bool)
	for _, prop := range entityType.Properties {
		if prop.Type == "Edm.Guid" {
			guidProperties[prop.Name] = true
		}
	}

	// If no GUID properties, return unchanged
	if len(guidProperties) == 0 {
		return filter
	}

	// Transform the filter string
	// Look for patterns like: PropertyName eq 'value'
	// and transform to: PropertyName eq guid'value'
	result := filter
	for propName := range guidProperties {
		// Pattern to match: propName eq 'uuid-value'
		// We need to handle various spacing and quote types
		// Note: Pattern includes dynamic propName so regex must be compiled per property
		patterns := []struct {
			regex       string
			replacement string
		}{
			// Handle: PropName eq 'value'
			{
				regex:       `(` + regexp.QuoteMeta(propName) + `\s+eq\s+)'([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12})'`,
				replacement: `${1}guid'${2}'`,
			},
			// Handle: PropName ne 'value'
			{
				regex:       `(` + regexp.QuoteMeta(propName) + `\s+ne\s+)'([a-fA-F0-9]{8}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{4}-[a-fA-F0-9]{12})'`,
				replacement: `${1}guid'${2}'`,
			},
		}

		for _, p := range patterns {
			re := regexp.MustCompile(p.regex)
			result = re.ReplaceAllString(result, p.replacement)
		}
	}

	if b.config.Verbose && result != filter {
		fmt.Fprintf(os.Stderr, "[VERBOSE] Transformed SAP filter: %s -> %s\n", filter, result)
	}

	return result
}

// isSAPService determines if the current service is a SAP OData service
func (b *ODataMCPBridge) isSAPService() bool {
	// Check for SAP-specific hints
	hints := b.hintManager.GetHints(b.config.ServiceURL)
	if hints != nil {
		if st, ok := hints["service_type"].(string); ok {
			return strings.Contains(strings.ToLower(st), "sap")
		}
	}

	// Check for SAP indicators in metadata
	if b.metadata != nil {
		// Check for SAP-specific annotations in entity sets
		for _, es := range b.metadata.EntitySets {
			if es.SAPCreatable || es.SAPUpdatable || es.SAPDeletable || es.SAPSearchable || es.SAPPageable {
				return true
			}
		}

		// Check for SAP namespace
		if strings.Contains(b.metadata.SchemaNamespace, "sap") || strings.Contains(strings.ToLower(b.metadata.SchemaNamespace), "sap") {
			return true
		}
	}

	// Check URL for SAP indicators
	url := strings.ToLower(b.config.ServiceURL)
	return strings.Contains(url, "sap") || strings.Contains(url, "s4hana") || strings.Contains(url, "odata.sap")
}
