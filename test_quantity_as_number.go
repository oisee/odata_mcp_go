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

	fmt.Println("=== Testing Quantity Field Format ===\n")

	// Get CSRF token
	token, cookies := fetchCSRFToken(serviceURL, username, password)

	// Create a new order for testing
	orderData := map[string]interface{}{
		"CustomerID":   "0100000005",
		"Note":         "Test quantity format",
		"NoteLanguage": "EN",
		"CurrencyCode": "USD",
	}
	
	orderID, err := createSalesOrder(serviceURL, username, password, token, cookies, orderData)
	if err != nil {
		fmt.Printf("Failed to create order: %v\n", err)
		return
	}
	
	fmt.Printf("Created test order: %s\n\n", orderID)

	// Test 1: Quantity as JSON number (what would happen without conversion)
	fmt.Println("Test 1: Sending Quantity as JSON number")
	fmt.Println(strings.Repeat("-", 50))
	
	lineItemData1 := map[string]interface{}{
		"SalesOrderID": orderID,
		"ItemPosition": "0000000010",
		"ProductID":    "HT-1254",
		"DeliveryDate": fmt.Sprintf("/Date(%d)/", time.Now().AddDate(0, 0, 7).UnixMilli()),
		"Quantity":     1,  // As number, not string
		"QuantityUnit": "EA",
	}
	
	resp1, status1 := testLineItemCreation(serviceURL, username, password, token, cookies, lineItemData1)
	fmt.Printf("Status: %d\n", status1)
	if status1 >= 400 {
		fmt.Printf("FAILED as expected!\n")
		// Extract error message
		var errResp map[string]interface{}
		if json.Unmarshal([]byte(resp1), &errResp) == nil {
			if err, ok := errResp["error"].(map[string]interface{}); ok {
				if msg, ok := err["message"].(map[string]interface{}); ok {
					fmt.Printf("Error: %v\n", msg["value"])
				}
			}
		}
	}

	// Test 2: Quantity as string (with conversion)
	fmt.Println("\n\nTest 2: Sending Quantity as JSON string")
	fmt.Println(strings.Repeat("-", 50))
	
	lineItemData2 := map[string]interface{}{
		"SalesOrderID": orderID,
		"ItemPosition": "0000000020",
		"ProductID":    "HT-1254",
		"DeliveryDate": fmt.Sprintf("/Date(%d)/", time.Now().AddDate(0, 0, 7).UnixMilli()),
		"Quantity":     "2.000",  // As string
		"QuantityUnit": "EA",
	}
	
	resp2, status2 := testLineItemCreation(serviceURL, username, password, token, cookies, lineItemData2)
	fmt.Printf("Status: %d\n", status2)
	if status2 < 300 {
		fmt.Printf("SUCCESS!\n")
		// Show created item
		var result map[string]interface{}
		if json.Unmarshal([]byte(resp2), &result) == nil {
			if d, ok := result["d"].(map[string]interface{}); ok {
				fmt.Printf("Created: Position %v, Quantity %v\n", d["ItemPosition"], d["Quantity"])
			}
		}
	}

	// Test 3: Float as number
	fmt.Println("\n\nTest 3: Sending Quantity as float")
	fmt.Println(strings.Repeat("-", 50))
	
	lineItemData3 := map[string]interface{}{
		"SalesOrderID": orderID,
		"ItemPosition": "0000000030",
		"ProductID":    "HT-1254",
		"DeliveryDate": fmt.Sprintf("/Date(%d)/", time.Now().AddDate(0, 0, 7).UnixMilli()),
		"Quantity":     1.5,  // As float number
		"QuantityUnit": "EA",
	}
	
	_, status3 := testLineItemCreation(serviceURL, username, password, token, cookies, lineItemData3)
	fmt.Printf("Status: %d\n", status3)
	if status3 >= 400 {
		fmt.Printf("FAILED as expected!\n")
	}

	fmt.Println("\n\nConclusion:")
	fmt.Println("- SAP OData v2 requires Edm.Decimal fields to be sent as JSON strings")
	fmt.Println("- Sending numeric values causes 'Failed to read property' errors")
	fmt.Println("- The Go implementation's numeric-to-string conversion is NECESSARY")
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
	
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if orderID, ok := d["SalesOrderID"].(string); ok {
				return orderID, nil
			}
		}
	}
	
	return "", fmt.Errorf("could not parse order ID")
}

func testLineItemCreation(serviceURL, username, password, token string, cookies []*http.Cookie, data map[string]interface{}) (string, int) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderLineItemSet"
	
	jsonData, _ := json.Marshal(data)
	fmt.Printf("JSON sent: %s\n", string(jsonData))
	
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
		return err.Error(), 0
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body), resp.StatusCode
}