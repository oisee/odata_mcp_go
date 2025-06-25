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

	fmt.Println("=== SAP Order Investigation ===\n")

	// 1. Check if the sales order exists
	fmt.Println("1. Checking if SalesOrderID 0500000001 exists...")
	checkSalesOrder(serviceURL, username, password, "0500000001")

	// 2. Get existing orders to find a valid one
	fmt.Println("\n2. Getting list of existing sales orders...")
	getExistingSalesOrders(serviceURL, username, password)

	// 3. Check what line items can be added to
	fmt.Println("\n3. Checking line items for an existing order...")
	checkLineItemsForOrder(serviceURL, username, password, "0500000000")

	// 4. Check product availability
	fmt.Println("\n4. Checking if product HT-1001 exists...")
	checkProduct(serviceURL, username, password, "HT-1001")
}

func checkSalesOrder(serviceURL, username, password, orderID string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/SalesOrderSet('%s')?$format=json", strings.TrimSuffix(serviceURL, "/"), orderID)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	if resp.StatusCode == 404 {
		fmt.Printf("Sales Order %s NOT FOUND\n", orderID)
	} else if resp.StatusCode == 200 {
		fmt.Printf("Sales Order %s EXISTS\n", orderID)
		
		// Parse and show relevant details
		var result map[string]interface{}
		if json.Unmarshal(body, &result) == nil {
			if d, ok := result["d"].(map[string]interface{}); ok {
				fmt.Printf("  Customer: %v\n", d["CustomerID"])
				fmt.Printf("  Status: %v\n", d["LifecycleStatus"])
				fmt.Printf("  Created: %v\n", d["CreatedAt"])
			}
		}
	} else {
		fmt.Printf("Status %d: %s\n", resp.StatusCode, string(body))
	}
}

func getExistingSalesOrders(serviceURL, username, password string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/SalesOrderSet?$top=5&$orderby=SalesOrderID desc&$format=json", strings.TrimSuffix(serviceURL, "/"))
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
				fmt.Printf("Found %d sales orders:\n", len(results))
				for _, order := range results {
					if o, ok := order.(map[string]interface{}); ok {
						fmt.Printf("  - %v (Customer: %v, Status: %v)\n", 
							o["SalesOrderID"], o["CustomerID"], o["LifecycleStatus"])
					}
				}
			}
		}
	}
}

func checkLineItemsForOrder(serviceURL, username, password, orderID string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/SalesOrderSet('%s')/ToLineItems?$format=json", strings.TrimSuffix(serviceURL, "/"), orderID)
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
				fmt.Printf("Order %s has %d line items:\n", orderID, len(results))
				for i, item := range results {
					if it, ok := item.(map[string]interface{}); ok {
						fmt.Printf("  - Position %v: Product %v, Qty: %v %v\n", 
							it["ItemPosition"], it["ProductID"], it["Quantity"], it["QuantityUnit"])
					}
					if i >= 2 {
						fmt.Println("  ... and more")
						break
					}
				}
			}
		}
	}
}

func checkProduct(serviceURL, username, password, productID string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	url := fmt.Sprintf("%s/ProductSet('%s')?$format=json", strings.TrimSuffix(serviceURL, "/"), productID)
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	
	if resp.StatusCode == 404 {
		fmt.Printf("Product %s NOT FOUND\n", productID)
	} else if resp.StatusCode == 200 {
		fmt.Printf("Product %s EXISTS\n", productID)
		
		var result map[string]interface{}
		if json.Unmarshal(body, &result) == nil {
			if d, ok := result["d"].(map[string]interface{}); ok {
				fmt.Printf("  Name: %v\n", d["Name"])
				fmt.Printf("  Category: %v\n", d["Category"])
				fmt.Printf("  Price: %v %v\n", d["Price"], d["CurrencyCode"])
			}
		}
	}
}