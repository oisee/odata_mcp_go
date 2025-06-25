package main

import (
	"encoding/json"
	"fmt"
	
	"github.com/odata-mcp/go/internal/utils"
)

func main() {
	fmt.Println("=== JSON Output Test for SAP OData ===\n")

	// Simulate what would be sent to SAP for a line item creation
	lineItem := map[string]interface{}{
		"SalesOrderID": "0500010047",
		"ProductID": "HT-1251",
		"Quantity": 1,
		"QuantityUnit": "EA",
		"DeliveryDate": "2025-06-24T00:00:00",
	}

	fmt.Println("1. Before conversion:")
	jsonBefore, _ := json.Marshal(lineItem)
	fmt.Printf("JSON: %s\n", jsonBefore)

	// Apply conversions
	converted := utils.ConvertNumericsInMap(lineItem)
	
	fmt.Println("\n2. After numeric conversion:")
	jsonAfter, _ := json.Marshal(converted)
	fmt.Printf("JSON: %s\n", jsonAfter)

	// Check specific field
	fmt.Println("\n3. Quantity field analysis:")
	fmt.Printf("Before: Type=%T, Value=%v\n", lineItem["Quantity"], lineItem["Quantity"])
	fmt.Printf("After:  Type=%T, Value=%v\n", converted["Quantity"], converted["Quantity"])
	
	// Verify JSON encoding
	fmt.Println("\n4. JSON encoding verification:")
	var parsed map[string]interface{}
	json.Unmarshal(jsonAfter, &parsed)
	
	for key, value := range parsed {
		fmt.Printf("   %s: %T = %v\n", key, value, value)
	}
	
	// Show what SAP expects vs what we're sending
	fmt.Println("\n5. SAP expectation vs our output:")
	fmt.Println("SAP expects for Quantity field: \"Quantity\": \"1\"")
	fmt.Printf("We are sending: ")
	if qty, ok := converted["Quantity"].(string); ok {
		fmt.Printf("\"Quantity\": \"%s\" ✓\n", qty)
	} else {
		fmt.Printf("\"Quantity\": %v ✗\n", converted["Quantity"])
	}
}