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

	fmt.Println("=== Finding Open Sales Orders ===\n")

	// Get orders with different statuses
	fmt.Println("1. Fetching sales orders by status...")
	orders := getSalesOrdersByStatus(serviceURL, username, password)
	
	// Find an open order
	var openOrderID string
	for _, order := range orders {
		if order["LifecycleStatus"] == "N" || order["LifecycleStatus"] == "O" || order["LifecycleStatus"] == "P" {
			openOrderID = order["SalesOrderID"].(string)
			fmt.Printf("\nFound open order: %s (Status: %s)\n", openOrderID, order["LifecycleStatus"])
			break
		}
	}

	if openOrderID == "" {
		fmt.Println("\nNo open orders found. Creating a new order...")
		newOrderID := createNewSalesOrder(serviceURL, username, password)
		if newOrderID != "" {
			openOrderID = newOrderID
		}
	}

	if openOrderID != "" {
		// Try to add line item to the open order
		fmt.Printf("\n2. Adding line item to order %s...\n", openOrderID)
		
		token, cookies := fetchCSRFToken(serviceURL, username, password)
		
		// Get next item position
		nextPosition := getNextItemPosition(serviceURL, username, password, openOrderID)
		
		lineItemData := map[string]interface{}{
			"SalesOrderID": openOrderID,
			"ItemPosition": nextPosition,
			"ProductID":    "HT-1001",
			"DeliveryDate": fmt.Sprintf("/Date(%d)/", time.Now().AddDate(0, 0, 7).UnixMilli()),
			"Quantity":     "1.000",
			"QuantityUnit": "EA",
		}
		
		jsonData, _ := json.MarshalIndent(lineItemData, "", "  ")
		fmt.Printf("Request body:\n%s\n", jsonData)
		
		resp, statusCode, err := createLineItem(serviceURL, username, password, token, cookies, lineItemData)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("\nStatus: %d\n", statusCode)
			if statusCode < 300 {
				fmt.Printf("Success! Line item created.\n")
				
				// Parse response to show created item
				var result map[string]interface{}
				if json.Unmarshal([]byte(resp), &result) == nil {
					if d, ok := result["d"].(map[string]interface{}); ok {
						fmt.Printf("\nCreated line item:\n")
						fmt.Printf("  Order: %v\n", d["SalesOrderID"])
						fmt.Printf("  Position: %v\n", d["ItemPosition"])
						fmt.Printf("  Product: %v\n", d["ProductID"])
						fmt.Printf("  Quantity: %v %v\n", d["Quantity"], d["QuantityUnit"])
					}
				}
			} else {
				fmt.Printf("Failed. Response:\n%s\n", resp)
			}
		}
	}
}

func getSalesOrdersByStatus(serviceURL, username, password string) []map[string]interface{} {
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Get recent orders
	url := fmt.Sprintf("%s/SalesOrderSet?$top=20&$orderby=CreatedAt desc&$format=json", strings.TrimSuffix(serviceURL, "/"))
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	var orders []map[string]interface{}
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if results, ok := d["results"].([]interface{}); ok {
				fmt.Printf("Found %d orders:\n", len(results))
				for _, order := range results {
					if o, ok := order.(map[string]interface{}); ok {
						orders = append(orders, o)
						fmt.Printf("  - %v (Customer: %v, Status: %v, Created: %v)\n", 
							o["SalesOrderID"], o["CustomerID"], o["LifecycleStatus"], o["CreatedAt"])
					}
				}
			}
		}
	}
	
	return orders
}

func createNewSalesOrder(serviceURL, username, password string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Fetch CSRF token
	token, cookies := fetchCSRFToken(serviceURL, username, password)
	
	// Create minimal order
	orderData := map[string]interface{}{
		"CustomerID": "0100000000", // Use a standard customer
		"Note": "Test order created via MCP",
		"NoteLanguage": "EN",
		"CurrencyCode": "USD",
	}
	
	jsonData, _ := json.Marshal(orderData)
	
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderSet"
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
		fmt.Printf("Failed to create order: %v\n", err)
		return ""
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	if resp.StatusCode < 300 {
		var result map[string]interface{}
		if json.Unmarshal(body, &result) == nil {
			if d, ok := result["d"].(map[string]interface{}); ok {
				orderID := d["SalesOrderID"].(string)
				fmt.Printf("Created new order: %s\n", orderID)
				return orderID
			}
		}
	} else {
		fmt.Printf("Failed to create order. Status %d: %s\n", resp.StatusCode, string(body))
	}
	
	return ""
}

func getNextItemPosition(serviceURL, username, password, orderID string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Get existing line items
	url := fmt.Sprintf("%s/SalesOrderSet('%s')/ToLineItems?$orderby=ItemPosition desc&$top=1&$format=json", 
		strings.TrimSuffix(serviceURL, "/"), orderID)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		return "0000000010" // Default first position
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if results, ok := d["results"].([]interface{}); ok && len(results) > 0 {
				if item, ok := results[0].(map[string]interface{}); ok {
					if pos, ok := item["ItemPosition"].(string); ok {
						// Parse and increment
						var posNum int
						fmt.Sscanf(pos, "%d", &posNum)
						return fmt.Sprintf("%010d", posNum+10)
					}
				}
			}
		}
	}
	
	return "0000000010" // Default first position
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

func createLineItem(serviceURL, username, password, token string, cookies []*http.Cookie, data map[string]interface{}) (string, int, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderLineItemSet"
	
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
		return "", 0, err
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	return string(body), resp.StatusCode, nil
}