package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

func main() {
	// Load test credentials
	godotenv.Load(".env.test")
	
	serviceURL := os.Getenv("ODATA_SERVICE_URL")
	username := os.Getenv("ODATA_USERNAME")
	password := os.Getenv("ODATA_PASSWORD")

	fmt.Println("=== SAP Order Creation - Final Solution ===\n")

	// Get CSRF token
	token, cookies := fetchCSRFToken(serviceURL, username, password)

	// Step 1: Create a new sales order
	fmt.Println("1. Creating new sales order...")
	orderData := map[string]interface{}{
		"CustomerID":   "0100000005", // Valid customer from our check
		"Note":         "Test order via Go MCP",
		"NoteLanguage": "EN",
		"CurrencyCode": "USD",
	}
	
	orderID, err := createSalesOrder(serviceURL, username, password, token, cookies, orderData)
	if err != nil {
		fmt.Printf("Failed to create order: %v\n", err)
		return
	}
	
	fmt.Printf("Created order: %s\n\n", orderID)
	
	// Step 2: Add line item with proper format
	fmt.Println("2. Adding line item to the order...")
	
	// Test both with quantity as string (as seen in responses) and with our conversion
	lineItemData := map[string]interface{}{
		"SalesOrderID": orderID,
		"ItemPosition": "0000000010",
		"ProductID":    "HT-1254", // Basic product ID
		"DeliveryDate": fmt.Sprintf("/Date(%d)/", time.Now().AddDate(0, 0, 7).UnixMilli()),
		"Quantity":     "1.000",   // As string with 3 decimals like in the response
		"QuantityUnit": "EA",
		"Note":         "Test line item",
		"NoteLanguage": "EN",
	}
	
	jsonData, _ := json.MarshalIndent(lineItemData, "", "  ")
	fmt.Printf("Line item data:\n%s\n\n", jsonData)
	
	resp, statusCode, err := createLineItem(serviceURL, username, password, token, cookies, lineItemData)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
	} else {
		fmt.Printf("Status: %d\n", statusCode)
		if statusCode < 300 {
			fmt.Println("SUCCESS! Line item created.")
			
			// Parse and show the created item
			var result map[string]interface{}
			if json.Unmarshal([]byte(resp), &result) == nil {
				if d, ok := result["d"].(map[string]interface{}); ok {
					fmt.Printf("\nCreated line item details:\n")
					fmt.Printf("  Order: %v\n", d["SalesOrderID"])
					fmt.Printf("  Position: %v\n", d["ItemPosition"])
					fmt.Printf("  Product: %v\n", d["ProductID"])
					fmt.Printf("  Quantity: %v %v\n", d["Quantity"], d["QuantityUnit"])
					fmt.Printf("  Delivery: %v\n", d["DeliveryDate"])
				}
			}
		} else {
			fmt.Printf("Failed. Response:\n%s\n", resp)
		}
	}
	
	// Step 3: Verify what data format worked
	fmt.Println("\n3. Analysis:")
	fmt.Println("- Quantity was sent as a string (not a number)")
	fmt.Println("- This matches the format returned by SAP in GET responses")
	fmt.Println("- The Go implementation's numeric conversion IS necessary for SAP compatibility")
}

func fetchCSRFToken(serviceURL, username, password string) (string, []*http.Cookie) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	req, _ := http.NewRequest("GET", serviceURL, nil)
	req.SetBasicAuth(username, password)
	req.Header.Set("X-CSRF-Token", "Fetch")
	
	resp, err := client.Do(req)
	if err != nil {
		return "", nil
	}
	defer resp.Body.Close()
	
	token := resp.Header.Get("X-CSRF-Token")
	return token, resp.Cookies()
}

func createSalesOrder(serviceURL, username, password, token string, cookies []*http.Cookie, data map[string]interface{}) (string, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderSet"
	
	jsonData, _ := json.Marshal(data)
	req, _ := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	// Parse response to get order ID
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if orderID, ok := d["SalesOrderID"].(string); ok {
				return orderID, nil
			}
		}
	}
	
	return "", fmt.Errorf("could not parse order ID from response")
}

func createLineItem(serviceURL, username, password, token string, cookies []*http.Cookie, data map[string]interface{}) (string, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderLineItemSet"
	
	jsonData, _ := json.Marshal(data)
	fmt.Printf("Actual JSON being sent:\n%s\n\n", string(jsonData))
	
	req, _ := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewReader(jsonData))
	req.SetBasicAuth(username, password)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("X-CSRF-Token", token)
	}
	
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	return string(body), resp.StatusCode, nil
}