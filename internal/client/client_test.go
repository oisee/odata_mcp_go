package client

import (
	"net/url"
	"strings"
	"testing"

	"github.com/zmcp/odata-mcp/internal/models"
)

func TestEncodeQueryParams(t *testing.T) {
	tests := []struct {
		name     string
		params   url.Values
		expected string
	}{
		{
			name: "Simple filter with spaces",
			params: url.Values{
				"$filter": []string{"Name eq 'Test Value'"},
			},
			expected: "%24filter=Name%20eq%20%27Test%20Value%27",
		},
		{
			name: "Multiple parameters with spaces",
			params: url.Values{
				"$filter": []string{"Category eq 'Test Category'"},
				"$select": []string{"ID, Name, Description"},
			},
			expected: "%24filter=Category%20eq%20%27Test%20Category%27&%24select=ID%2C%20Name%2C%20Description",
		},
		{
			name: "Special characters",
			params: url.Values{
				"$filter": []string{"Code eq '$TEST_CODE'"},
			},
			expected: "%24filter=Code%20eq%20%27%24TEST_CODE%27",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := encodeQueryParams(tt.params)
			if result != tt.expected {
				t.Errorf("encodeQueryParams() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatKeyValue(t *testing.T) {
	c := &ODataClient{}

	tests := []struct {
		name     string
		value    interface{}
		expected string
	}{
		{
			name:     "Regular string",
			value:    "test-value",
			expected: "'test-value'",
		},
		{
			name:     "GUID value",
			value:    models.GUIDValue("79cebabb-7e17-1eef-a5f0-30809b11535e"),
			expected: "guid'79cebabb-7e17-1eef-a5f0-30809b11535e'",
		},
		{
			name:     "Integer",
			value:    42,
			expected: "42",
		},
		{
			name:     "Boolean true",
			value:    true,
			expected: "true",
		},
		{
			name:     "Boolean false",
			value:    false,
			expected: "false",
		},
		{
			name:     "Float",
			value:    3.14,
			expected: "3.14",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.formatKeyValue(tt.value)
			if result != tt.expected {
				t.Errorf("formatKeyValue(%v) = %v, want %v", tt.value, result, tt.expected)
			}
		})
	}
}

func TestBuildKeyPredicate(t *testing.T) {
	c := &ODataClient{}

	tests := []struct {
		name     string
		key      map[string]interface{}
		expected string
	}{
		{
			name:     "Single string key",
			key:      map[string]interface{}{"ID": "123"},
			expected: "'123'",
		},
		{
			name:     "Single GUID key",
			key:      map[string]interface{}{"ID": models.GUIDValue("79cebabb-7e17-1eef-a5f0-30809b11535e")},
			expected: "guid'79cebabb-7e17-1eef-a5f0-30809b11535e'",
		},
		{
			name:     "Single integer key",
			key:      map[string]interface{}{"ID": 42},
			expected: "42",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := c.buildKeyPredicate(tt.key)
			if result != tt.expected {
				t.Errorf("buildKeyPredicate(%v) = %v, want %v", tt.key, result, tt.expected)
			}
		})
	}
}

func TestBuildKeyPredicateComposite(t *testing.T) {
	c := &ODataClient{}

	// Composite key with two GUIDs (like the issue #16 example)
	key := map[string]interface{}{
		"ChemicalUUID":         models.GUIDValue("06188b1f-4e75-1fe0-a0c7-ec4e1da95d29"),
		"ChemicalRevisionUUID": models.GUIDValue("06188b1f-4e75-1fe0-a0c7-ec4e1da97d29"),
	}

	result := c.buildKeyPredicate(key)

	// Check that both parts are present (order may vary due to map iteration)
	if !strings.Contains(result, "ChemicalUUID=guid'06188b1f-4e75-1fe0-a0c7-ec4e1da95d29'") {
		t.Errorf("buildKeyPredicate missing ChemicalUUID, got: %v", result)
	}
	if !strings.Contains(result, "ChemicalRevisionUUID=guid'06188b1f-4e75-1fe0-a0c7-ec4e1da97d29'") {
		t.Errorf("buildKeyPredicate missing ChemicalRevisionUUID, got: %v", result)
	}
	if !strings.Contains(result, ",") {
		t.Errorf("buildKeyPredicate should have comma separator, got: %v", result)
	}
}
