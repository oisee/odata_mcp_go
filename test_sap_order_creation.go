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

	fmt.Println("=== SAP Order Creation Test ===\n")
	fmt.Printf("Service: %s\n", serviceURL)
	fmt.Printf("User: %s\n\n", username)

	// First, let's get the metadata to understand the SalesOrderLineItemSet structure
	fmt.Println("1. Fetching metadata for SalesOrderLineItemSet...")
	metadata := fetchMetadata(serviceURL, username, password)
	fmt.Println("Metadata fetched successfully.\n")

	// Try to fetch CSRF token
	fmt.Println("2. Fetching CSRF token...")
	token, cookies := fetchCSRFToken(serviceURL, username, password)
	fmt.Printf("CSRF Token: %s\n", token)
	fmt.Printf("Cookies: %d received\n\n", len(cookies))

	// Test different formats for line item creation
	fmt.Println("3. Testing different line item formats...")
	
	testCases := []struct {
		name string
		data map[string]interface{}
	}{
		{
			name: "Test 1: Quantity as number",
			data: map[string]interface{}{
				"SalesOrderID": "0500000001",
				"ProductID":    "HT-1001",
				"Quantity":     1,
				"QuantityUnit": "EA",
			},
		},
		{
			name: "Test 2: Quantity as string",
			data: map[string]interface{}{
				"SalesOrderID": "0500000001",
				"ProductID":    "HT-1001",
				"Quantity":     "1",
				"QuantityUnit": "EA",
			},
		},
		{
			name: "Test 3: With all possible fields",
			data: map[string]interface{}{
				"SalesOrderID":     "0500000001",
				"ItemPosition":     "0000000010",
				"ProductID":        "HT-1001",
				"Note":             "Test item",
				"NoteLanguage":     "EN",
				"Currency":         "USD",
				"GrossAmount":      "100.00",
				"NetAmount":        "100.00",
				"TaxAmount":        "0.00",
				"DeliveryDate":     "/Date(1735689600000)/",
				"Quantity":         "1.000",
				"QuantityUnit":     "EA",
			},
		},
	}

	for _, test := range testCases {
		fmt.Printf("\n%s\n", test.name)
		fmt.Println(strings.Repeat("-", len(test.name)))
		
		jsonData, _ := json.MarshalIndent(test.data, "", "  ")
		fmt.Printf("Request body:\n%s\n", jsonData)
		
		resp, err := createLineItem(serviceURL, username, password, token, cookies, test.data)
		if err != nil {
			fmt.Printf("Error: %v\n", err)
		} else {
			fmt.Printf("Response: %s\n", resp)
		}
	}

	// Now let's examine what fields are actually required
	fmt.Println("\n4. Examining SalesOrderLineItemSet structure from metadata...")
	examineEntityType(metadata, "SalesOrderLineItem")
}

func fetchMetadata(serviceURL, username, password string) string {
	client := &http.Client{Timeout: 30 * time.Second}
	
	metadataURL := strings.TrimSuffix(serviceURL, "/") + "/$metadata"
	req, _ := http.NewRequest("GET", metadataURL, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		panic(fmt.Sprintf("Failed to fetch metadata: %v", err))
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	return string(body)
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

func createLineItem(serviceURL, username, password, token string, cookies []*http.Cookie, data map[string]interface{}) (string, error) {
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
	
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	return string(body), nil
}

func examineEntityType(metadata, entityName string) {
	// Simple parsing to find required fields
	if strings.Contains(metadata, entityName) {
		fmt.Println("Entity type found in metadata.")
		
		// Look for properties with Nullable="false"
		lines := strings.Split(metadata, "\n")
		inEntity := false
		for _, line := range lines {
			if strings.Contains(line, fmt.Sprintf(`Name="%s"`, entityName)) {
				inEntity = true
			}
			if inEntity && strings.Contains(line, "</EntityType>") {
				break
			}
			if inEntity && strings.Contains(line, "Property") && strings.Contains(line, `Nullable="false"`) {
				fmt.Printf("Required field: %s\n", line)
			}
		}
	}
}