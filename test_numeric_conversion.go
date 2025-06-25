package main

import (
	"encoding/json"
	"fmt"
	
	"github.com/odata-mcp/go/internal/utils"
)

func main() {
	fmt.Println("=== SAP OData Numeric Field Conversion Demo ===\n")

	// Example sales order line item data
	lineItem := map[string]interface{}{
		"SalesOrderID": "0500010047",
		"ProductID": "HT-1251",
		"Quantity": 1,              // numeric
		"QuantityUnit": "EA",
		"UnitPrice": 989.99,        // numeric
		"NetAmount": 989.99,        // numeric
		"TaxAmount": 98.99,         // numeric
		"GrossAmount": 1088.98,     // numeric
		"DeliveryDate": "2025-06-24T00:00:00",
		"ItemPosition": 10,         // numeric
		"Description": "Laptop",
	}

	fmt.Println("1. Original data (numeric values as JSON numbers):")
	printJSON(lineItem)

	// Convert numerics to strings for SAP
	converted := utils.ConvertNumericsInMap(lineItem)
	fmt.Println("\n2. After numeric conversion (for SAP OData v2):")
	printJSON(converted)

	// Test field name detection
	fmt.Println("\n3. Field name detection test:")
	testFields := []string{
		"Quantity",
		"QuantityUnit",
		"NetAmount",
		"Description",
		"TotalPrice",
		"ItemCount",
		"ProductID",
		"DiscountRate",
		"CustomerName",
		"TaxPercentage",
	}

	for _, field := range testFields {
		isDecimal := utils.IsLikelyDecimalField(field)
		fmt.Printf("   %-20s -> %v\n", field, isDecimal)
	}

	// Test various numeric types
	fmt.Println("\n4. Numeric type conversion test:")
	testValues := map[string]interface{}{
		"int":     42,
		"int64":   int64(999999999),
		"float32": float32(123.45),
		"float64": 9999.99999,
		"string":  "already string",
		"bool":    true,
		"nil":     nil,
	}

	for name, value := range testValues {
		converted := utils.ConvertNumericToString(value)
		fmt.Printf("   %-10s %T(%v) -> %T(%v)\n", name+":", value, value, converted, converted)
	}

	// Test nested structure
	fmt.Println("\n5. Nested structure test:")
	order := map[string]interface{}{
		"OrderID": "12345",
		"TotalAmount": 5000.50,
		"Items": []interface{}{
			map[string]interface{}{
				"ItemID": 1,
				"Quantity": 5,
				"Price": 100.10,
			},
			map[string]interface{}{
				"ItemID": 2,
				"Quantity": 10,
				"Price": 400.00,
			},
		},
		"Customer": map[string]interface{}{
			"ID": "C001",
			"CreditLimit": 10000.00,
			"Balance": 2500.50,
		},
	}

	convertedOrder := utils.ConvertNumericsInMap(order)
	fmt.Println("Original nested structure:")
	printJSON(order)
	fmt.Println("\nAfter conversion:")
	printJSON(convertedOrder)
}

func printJSON(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}