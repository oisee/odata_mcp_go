package test

import (
	"encoding/json"
	"testing"

	"github.com/odata-mcp/go/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertNumericFieldsToStrings(t *testing.T) {
	tests := []struct {
		name          string
		input         map[string]interface{}
		decimalFields []string
		expected      map[string]interface{}
	}{
		{
			name: "Convert specified decimal fields",
			input: map[string]interface{}{
				"OrderID":   "001",
				"Quantity":  10,
				"UnitPrice": 25.50,
				"Status":    "Active",
			},
			decimalFields: []string{"Quantity", "UnitPrice"},
			expected: map[string]interface{}{
				"OrderID":   "001",
				"Quantity":  "10",
				"UnitPrice": "25.5",
				"Status":    "Active",
			},
		},
		{
			name: "Handle various numeric types",
			input: map[string]interface{}{
				"IntValue":    int32(42),
				"FloatValue":  float32(3.14),
				"DoubleValue": 2.71828,
				"LargeInt":    int64(1000000),
			},
			decimalFields: []string{"IntValue", "FloatValue", "DoubleValue", "LargeInt"},
			expected: map[string]interface{}{
				"IntValue":    "42",
				"FloatValue":  "3.14",  // float32 may have precision issues
				"DoubleValue": "2.71828",
				"LargeInt":    "1000000",
			},
		},
		{
			name: "Preserve non-numeric values",
			input: map[string]interface{}{
				"Name":     "Product A",
				"Quantity": "already a string",
				"IsActive": true,
				"Data":     nil,
			},
			decimalFields: []string{"Quantity"},
			expected: map[string]interface{}{
				"Name":     "Product A",
				"Quantity": "already a string",
				"IsActive": true,
				"Data":     nil,
			},
		},
		{
			name: "Handle nested structures",
			input: map[string]interface{}{
				"OrderID": "001",
				"Items": []interface{}{
					map[string]interface{}{
						"ItemID":   "I001",
						"Quantity": 5,
						"Price":    19.99,
					},
				},
			},
			decimalFields: []string{"Quantity", "Price"},
			expected: map[string]interface{}{
				"OrderID": "001",
				"Items": []interface{}{
					map[string]interface{}{
						"ItemID":   "I001",
						"Quantity": "5",
						"Price":    "19.99",
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ConvertNumericFieldsToStrings(tt.input, tt.decimalFields)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertToString(t *testing.T) {
	tests := []struct {
		name     string
		input    interface{}
		expected string
	}{
		{"nil value", nil, ""},
		{"string value", "test", "test"},
		{"int value", 42, "42"},
		{"int64 value", int64(1234567890), "1234567890"},
		{"float32 value", float32(3.14), "3.14"},  // Note: float32 may show more precision
		{"float64 whole number", 100.0, "100"},
		{"float64 decimal", 25.75, "25.75"},
		{"bool true", true, "true"},
		{"bool false", false, "false"},
		{"large number avoiding scientific notation", 1000000.0, "1000000"},
		{"small decimal", 0.00001, "1e-05"}, // Go's %g format may use scientific notation for very small numbers
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.ConvertToString(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsLikelyDecimalField(t *testing.T) {
	tests := []struct {
		name     string
		field    string
		expected bool
	}{
		{"Quantity field", "Quantity", true},
		{"quantity lowercase", "quantity", true},
		{"OrderQuantity", "OrderQuantity", true},
		{"NetAmount", "NetAmount", true},
		{"UnitPrice", "UnitPrice", true},
		{"TaxAmount", "TaxAmount", true},
		{"discount_amt", "discount_amt", true},
		{"total_cost", "total_cost", true},
		{"Regular field", "Name", false},
		{"Description", "Description", false},
		{"Status", "Status", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := utils.IsLikelyDecimalField(tt.field)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConvertEntityDataForOData(t *testing.T) {
	input := map[string]interface{}{
		"OrderID":      "001",
		"CustomerName": "John Doe",
		"Quantity":     10,
		"UnitPrice":    25.50,
		"TaxAmount":    2.55,
		"Status":       "Active",
		"IsUrgent":     true,
	}

	result := utils.ConvertEntityDataForOData(input, nil)

	// Numeric fields that look like decimals should be converted to strings
	assert.Equal(t, "10", result["Quantity"])
	assert.Equal(t, "25.5", result["UnitPrice"])
	assert.Equal(t, "2.55", result["TaxAmount"])

	// Non-decimal fields should remain unchanged
	assert.Equal(t, "001", result["OrderID"])
	assert.Equal(t, "John Doe", result["CustomerName"])
	assert.Equal(t, "Active", result["Status"])
	assert.Equal(t, true, result["IsUrgent"])
}

func TestJSONMarshalingAfterConversion(t *testing.T) {
	// This test verifies that after conversion, JSON marshaling produces strings for numeric fields
	input := map[string]interface{}{
		"ProductID": "P001",
		"Quantity":  5,
		"Price":     19.99,
	}

	// Convert numeric fields
	converted := utils.ConvertEntityDataForOData(input, nil)

	// Marshal to JSON
	jsonData, err := json.Marshal(converted)
	require.NoError(t, err)

	// The JSON should have numeric values as strings
	expected := `{"Price":"19.99","ProductID":"P001","Quantity":"5"}`
	
	// Parse both to compare (to handle key ordering)
	var expectedMap, actualMap map[string]interface{}
	err = json.Unmarshal([]byte(expected), &expectedMap)
	require.NoError(t, err)
	err = json.Unmarshal(jsonData, &actualMap)
	require.NoError(t, err)

	assert.Equal(t, expectedMap, actualMap)
}