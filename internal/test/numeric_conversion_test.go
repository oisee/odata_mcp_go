package test

import (
	"encoding/json"
	"testing"

	"github.com/odata-mcp/go/internal/utils"
)

func TestIsLikelyDecimalField(t *testing.T) {
	tests := []struct {
		field    string
		expected bool
	}{
		// Should be detected as decimal
		{"Quantity", true},
		{"quantity", true},
		{"TotalAmount", true},
		{"NetPrice", true},
		{"UnitCost", true},
		{"DiscountRate", true},
		{"TaxPercentage", true},
		{"item_qty", true},
		{"price_amt", true},
		{"Weight", true},
		{"Volume", true},
		
		// Should NOT be detected as decimal
		{"ProductID", false},
		{"CustomerName", false},
		{"Description", false},
		{"Status", false},
		{"Category", false},
		{"Email", false},
	}

	for _, test := range tests {
		result := utils.IsLikelyDecimalField(test.field)
		if result != test.expected {
			t.Errorf("IsLikelyDecimalField(%s) = %v, expected %v", test.field, result, test.expected)
		}
	}
}

func TestConvertNumericToString(t *testing.T) {
	tests := []struct {
		input    interface{}
		expected string
		isString bool
	}{
		{int(42), "42", true},
		{int64(999999), "999999", true},
		{float32(123.45), "123.45", true},
		{float64(999.99), "999.99", true},
		{uint(100), "100", true},
		{"already string", "already string", true},
		{true, "", false}, // bool should not convert
		{nil, "", false},  // nil should not convert
	}

	for _, test := range tests {
		result := utils.ConvertNumericToString(test.input)
		
		if test.isString {
			str, ok := result.(string)
			if !ok {
				t.Errorf("ConvertNumericToString(%v) did not return string", test.input)
				continue
			}
			if str != test.expected {
				t.Errorf("ConvertNumericToString(%v) = %s, expected %s", test.input, str, test.expected)
			}
		} else {
			if result != test.input {
				t.Errorf("ConvertNumericToString(%v) should not have converted", test.input)
			}
		}
	}
}

func TestConvertNumericsInMap(t *testing.T) {
	input := map[string]interface{}{
		"SalesOrderID": "12345",
		"Quantity":     1,
		"Price":        99.99,
		"Description":  "Test Product",
		"IsActive":     true,
		"ItemCount":    5,
	}

	result := utils.ConvertNumericsInMap(input)

	// Check conversions
	if qty, ok := result["Quantity"].(string); !ok || qty != "1" {
		t.Errorf("Quantity was not converted correctly: %v", result["Quantity"])
	}

	if price, ok := result["Price"].(string); !ok || price != "99.99" {
		t.Errorf("Price was not converted correctly: %v", result["Price"])
	}

	if count, ok := result["ItemCount"].(string); !ok || count != "5" {
		t.Errorf("ItemCount was not converted correctly: %v", result["ItemCount"])
	}

	// Check non-conversions
	if desc, ok := result["Description"].(string); !ok || desc != "Test Product" {
		t.Errorf("Description should remain unchanged: %v", result["Description"])
	}

	if active, ok := result["IsActive"].(bool); !ok || !active {
		t.Errorf("IsActive should remain bool: %v", result["IsActive"])
	}
}

func TestConvertNumericsInNestedStructure(t *testing.T) {
	input := map[string]interface{}{
		"OrderID": "12345",
		"Total":   500.50,
		"Items": []interface{}{
			map[string]interface{}{
				"ItemID":   1,
				"Quantity": 5,
				"Price":    100.10,
			},
		},
		"Customer": map[string]interface{}{
			"ID":      "C001",
			"Balance": 1000.00,
		},
	}

	result := utils.ConvertNumericsInMap(input)

	// Check nested conversions
	items, ok := result["Items"].([]interface{})
	if !ok || len(items) != 1 {
		t.Fatalf("Items not converted correctly")
	}

	item := items[0].(map[string]interface{})
	if qty, ok := item["Quantity"].(string); !ok || qty != "5" {
		t.Errorf("Nested Quantity not converted: %v", item["Quantity"])
	}

	customer := result["Customer"].(map[string]interface{})
	if balance, ok := customer["Balance"].(string); !ok || balance != "1000" {
		t.Errorf("Nested Balance not converted: %v", customer["Balance"])
	}
}

func TestJSONMarshalAfterConversion(t *testing.T) {
	input := map[string]interface{}{
		"Quantity": 10,
		"Price":    99.99,
	}

	converted := utils.ConvertNumericsInMap(input)
	jsonData, err := json.Marshal(converted)
	if err != nil {
		t.Fatalf("JSON marshal failed: %v", err)
	}

	expected := `{"Price":"99.99","Quantity":"10"}`
	if string(jsonData) != expected {
		t.Errorf("JSON output incorrect.\nGot:      %s\nExpected: %s", jsonData, expected)
	}

	// Verify the JSON contains strings, not numbers
	var parsed map[string]interface{}
	json.Unmarshal(jsonData, &parsed)
	
	if _, ok := parsed["Quantity"].(string); !ok {
		t.Errorf("Quantity is not a string in JSON output")
	}
	if _, ok := parsed["Price"].(string); !ok {
		t.Errorf("Price is not a string in JSON output")
	}
}