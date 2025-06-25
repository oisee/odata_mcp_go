package main

import (
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

	fmt.Println("=== Getting Valid Test Data ===\n")

	// 1. Get valid customers
	fmt.Println("1. Valid Customers:")
	fmt.Println(strings.Repeat("-", 60))
	getValidCustomers(serviceURL, username, password)

	// 2. Get valid products
	fmt.Println("\n2. Valid Products:")
	fmt.Println(strings.Repeat("-", 60))
	getValidProducts(serviceURL, username, password)

	// 3. Look at a complete existing order
	fmt.Println("\n3. Examining existing order structure:")
	fmt.Println(strings.Repeat("-", 60))
	examineExistingOrder(serviceURL, username, password)
}

func getValidCustomers(serviceURL, username, password string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/BusinessPartnerSet?$top=5&$format=json", strings.TrimSuffix(serviceURL, "/"))
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if results, ok := d["results"].([]interface{}); ok {
				for _, bp := range results {
					if b, ok := bp.(map[string]interface{}); ok {
						fmt.Printf("ID: %-12s Name: %-30s Role: %s\n", 
							b["BusinessPartnerID"], b["CompanyName"], b["BusinessPartnerRole"])
					}
				}
			}
		}
	}
}

func getValidProducts(serviceURL, username, password string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/ProductSet?$top=10&$filter=ProductID lt 'HT-1010'&$format=json", strings.TrimSuffix(serviceURL, "/"))
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if results, ok := d["results"].([]interface{}); ok {
				for _, product := range results {
					if p, ok := product.(map[string]interface{}); ok {
						fmt.Printf("ID: %-8s Name: %-40s Price: %v %s\n", 
							p["ProductID"], p["Name"], p["Price"], p["CurrencyCode"])
					}
				}
			}
		}
	}
}

func examineExistingOrder(serviceURL, username, password string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Get a recent order with line items
	url := fmt.Sprintf("%s/SalesOrderSet?$top=1&$expand=ToLineItems&$format=json", strings.TrimSuffix(serviceURL, "/"))
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	var result map[string]interface{}
	if json.Unmarshal(body, &result) == nil {
		if d, ok := result["d"].(map[string]interface{}); ok {
			if results, ok := d["results"].([]interface{}); ok && len(results) > 0 {
				if order, ok := results[0].(map[string]interface{}); ok {
					fmt.Printf("Order: %v\n", order["SalesOrderID"])
					fmt.Printf("Customer: %v\n", order["CustomerID"])
					fmt.Printf("Status: %v\n", order["LifecycleStatus"])
					fmt.Printf("Currency: %v\n", order["CurrencyCode"])
					
					if lineItems, ok := order["ToLineItems"].(map[string]interface{}); ok {
						if items, ok := lineItems["results"].([]interface{}); ok {
							fmt.Printf("\nLine Items (%d):\n", len(items))
							for i, item := range items {
								if it, ok := item.(map[string]interface{}); ok {
									fmt.Printf("  [%d] Position: %v, Product: %v, Qty: %v, Delivery: %v\n",
										i+1, it["ItemPosition"], it["ProductID"], it["Quantity"], it["DeliveryDate"])
								}
								if i >= 2 {
									break
								}
							}
						}
					}
				}
			}
		}
	}
}