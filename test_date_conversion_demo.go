package main

import (
	"encoding/json"
	"fmt"
	
	"github.com/odata-mcp/go/internal/utils"
)

func main() {
	fmt.Println("=== SAP OData Date Conversion Demo ===\n")

	// Example order data from SAP
	orderData := map[string]interface{}{
		"OrderID": "0500000001",
		"CustomerID": "0000000001",
		"OrderDate": "2024-12-25T10:30:00Z",
		"DeliveryDate": "2025-01-01T00:00:00Z",
		"NetAmount": 1500.00,
		"Currency": "USD",
		"Items": []interface{}{
			map[string]interface{}{
				"ItemID": "10",
				"ProductID": "HT-1001",
				"Quantity": 2,
				"CreatedAt": "2024-12-25T10:30:00Z",
			},
		},
	}

	fmt.Println("1. Original data (ISO format from user input):")
	printJSON(orderData)

	// Convert to SAP legacy format for sending to OData service
	sapData := utils.ConvertDatesInMap(orderData, false) // false = ISO to legacy
	fmt.Println("\n2. Converted to SAP legacy format (for CREATE/UPDATE):")
	printJSON(sapData)

	// Simulate SAP response with legacy dates
	sapResponse := map[string]interface{}{
		"d": map[string]interface{}{
			"results": []interface{}{
				map[string]interface{}{
					"OrderID": "0500000001",
					"CustomerID": "0000000001",
					"OrderDate": "/Date(1735125000000)/",
					"DeliveryDate": "/Date(1735689600000)/",
					"NetAmount": "1500.00",
					"Currency": "USD",
					"CreatedAt": "/Date(1735125000000)/",
					"ModifiedAt": "/Date(1735125060000)/",
				},
			},
		},
	}

	fmt.Println("\n3. SAP OData response (legacy format):")
	printJSON(sapResponse)

	// Convert back to ISO for display
	displayData := utils.ConvertDatesInResponse(sapResponse, true) // true = legacy to ISO
	fmt.Println("\n4. Converted back to ISO format (for display):")
	printJSON(displayData)

	// Show specific date conversions
	fmt.Println("\n5. Date conversion examples:")
	examples := []string{
		"2024-12-25T10:30:00Z",
		"2024-12-25T10:30:00+01:00",
		"2024-12-25",
		"2024-12-25T00:00:00",
	}

	for _, iso := range examples {
		legacy := utils.ConvertISOToODataLegacy(iso)
		back := utils.ConvertODataLegacyToISO(legacy)
		fmt.Printf("\n   ISO: %s\n   -> Legacy: %s\n   -> Back to ISO: %s\n", iso, legacy, back)
	}

	// Test edge cases
	fmt.Println("\n6. Edge cases:")
	
	// Null date
	nullData := map[string]interface{}{
		"OrderID": "123",
		"OrderDate": nil,
		"DeliveryDate": "",
	}
	converted := utils.ConvertDatesInMap(nullData, false)
	fmt.Printf("\n   Null/empty dates: %v\n   Converted: %v\n", nullData, converted)

	// Invalid format
	invalidData := map[string]interface{}{
		"OrderDate": "25/12/2024", // Wrong format
		"DeliveryDate": "not a date",
	}
	converted2 := utils.ConvertDatesInMap(invalidData, false)
	fmt.Printf("\n   Invalid dates: %v\n   Converted (unchanged): %v\n", invalidData, converted2)
}

func printJSON(data interface{}) {
	b, _ := json.MarshalIndent(data, "", "  ")
	fmt.Println(string(b))
}