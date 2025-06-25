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

	fmt.Println("=== SAP Order Line Item Creation - Corrected Format ===\n")

	// Fetch CSRF token
	token, cookies := fetchCSRFToken(serviceURL, username, password)
	fmt.Printf("CSRF Token: %s\n", token)
	fmt.Printf("Cookies: %d received\n\n", len(cookies))

	// Calculate delivery date (7 days from now)
	deliveryDate := time.Now().AddDate(0, 0, 7)
	deliveryDateMs := deliveryDate.UnixMilli()

	// Test with corrected format based on metadata analysis
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Test 1: Minimal required fields only",
			data: map[string]interface{}{
				"SalesOrderID": "0500000001",
				"ItemPosition": "0000000010",
				"ProductID":    "HT-1001",
				"DeliveryDate": fmt.Sprintf("/Date(%d)/", deliveryDateMs),
				"Quantity":     "1.000",
			},
		},
		{
			name: "Test 2: With optional fields",
			data: map[string]interface{}{
				"SalesOrderID": "0500000001",
				"ItemPosition": "0000000020",
				"ProductID":    "HT-1001",
				"DeliveryDate": fmt.Sprintf("/Date(%d)/", deliveryDateMs),
				"Quantity":     "2.000",
				"QuantityUnit": "EA",
				"Note":         "Test item created via MCP",
				"NoteLanguage": "EN",
			},
		},
		{
			name: "Test 3: With amounts and currency",
			data: map[string]interface{}{
				"SalesOrderID": "0500000001",
				"ItemPosition": "0000000030",
				"ProductID":    "HT-1001",
				"DeliveryDate": fmt.Sprintf("/Date(%d)/", deliveryDateMs),
				"Quantity":     "1.000",
				"QuantityUnit": "EA",
				"CurrencyCode": "USD",
				"NetAmount":    "100.000",
				"TaxAmount":    "19.000",
				"GrossAmount":  "119.000",
			},
		},
	}

	for _, test := range testCases {
		fmt.Printf("\n%s\n", test.name)
		fmt.Println(strings.Repeat("-", len(test.name)))
		
		jsonData, _ := json.MarshalIndent(test.data, "", "  ")
		fmt.Printf("Request body:\n%s\n", jsonData)
		
		resp, statusCode, err := createLineItem(serviceURL, username, password, token, cookies, test.data)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Status: %d\n", statusCode)
			if statusCode < 300 {
				fmt.Printf("Success! Response:\n%s\n", resp)
			} else {
				fmt.Printf("Failed. Response:\n%s\n", resp)
			}
		}
		
		// Small delay between requests
		time.Sleep(1 * time.Second)
	}
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
	
	// Add cookies
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	
	fmt.Printf("\nSending request to: %s\n", url)
	fmt.Printf("Headers:\n")
	for k, v := range req.Header {
		if k != "Authorization" { // Don't print auth header
			fmt.Printf("  %s: %s\n", k, v)
		}
	}
	
	resp, err := client.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	return string(body), resp.StatusCode, nil
}