package main

import (
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
)

type Metadata struct {
	XMLName xml.Name `xml:"Edmx"`
	DataServices struct {
		Schema []Schema `xml:"Schema"`
	} `xml:"DataServices"`
}

type Schema struct {
	Namespace   string       `xml:"Namespace,attr"`
	EntityTypes []EntityType `xml:"EntityType"`
	EntityContainer struct {
		Name       string      `xml:"Name,attr"`
		EntitySets []EntitySet `xml:"EntitySet"`
	} `xml:"EntityContainer"`
}

type EntityType struct {
	Name       string     `xml:"Name,attr"`
	Properties []Property `xml:"Property"`
	NavProps   []NavigationProperty `xml:"NavigationProperty"`
}

type Property struct {
	Name     string `xml:"Name,attr"`
	Type     string `xml:"Type,attr"`
	Nullable string `xml:"Nullable,attr"`
	MaxLength string `xml:"MaxLength,attr"`
	Precision string `xml:"Precision,attr"`
	Scale    string `xml:"Scale,attr"`
}

type NavigationProperty struct {
	Name string `xml:"Name,attr"`
}

type EntitySet struct {
	Name       string `xml:"Name,attr"`
	EntityType string `xml:"EntityType,attr"`
}

func main() {
	// Load test credentials
	godotenv.Load(".env.test")
	
	serviceURL := os.Getenv("ODATA_SERVICE_URL")
	username := os.Getenv("ODATA_USERNAME")
	password := os.Getenv("ODATA_PASSWORD")

	fmt.Println("=== SAP Metadata Analysis ===\n")

	// Fetch and parse metadata
	metadata := fetchAndParseMetadata(serviceURL, username, password)
	
	// Find SalesOrderLineItem entity type
	fmt.Println("1. SalesOrderLineItem Entity Type Properties:")
	fmt.Println(strings.Repeat("-", 80))
	
	for _, schema := range metadata.DataServices.Schema {
		for _, entity := range schema.EntityTypes {
			if entity.Name == "SalesOrderLineItem" {
				fmt.Printf("Entity: %s\n\n", entity.Name)
				fmt.Printf("%-20s %-25s %-10s %-10s %-10s\n", "Property", "Type", "Nullable", "Precision", "Scale")
				fmt.Println(strings.Repeat("-", 80))
				
				for _, prop := range entity.Properties {
					nullable := prop.Nullable
					if nullable == "" {
						nullable = "true" // default
					}
					fmt.Printf("%-20s %-25s %-10s %-10s %-10s\n", 
						prop.Name, prop.Type, nullable, prop.Precision, prop.Scale)
				}
				
				fmt.Println("\nNavigation Properties:")
				for _, nav := range entity.NavProps {
					fmt.Printf("  - %s\n", nav.Name)
				}
			}
		}
	}

	// Also check for existing line items to see the format
	fmt.Println("\n\n2. Fetching existing line items to see data format...")
	fetchExistingLineItems(serviceURL, username, password)
}

func fetchAndParseMetadata(serviceURL, username, password string) *Metadata {
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
	
	var metadata Metadata
	if err := xml.Unmarshal(body, &metadata); err != nil {
		// Save for debugging
		ioutil.WriteFile("metadata.xml", body, 0644)
		panic(fmt.Sprintf("Failed to parse metadata: %v", err))
	}
	
	return &metadata
}

func fetchExistingLineItems(serviceURL, username, password string) {
	client := &http.Client{Timeout: 30 * time.Second}
	
	// Fetch first few line items
	url := strings.TrimSuffix(serviceURL, "/") + "/SalesOrderLineItemSet?$top=3&$format=json"
	req, _ := http.NewRequest("GET", url, nil)
	req.SetBasicAuth(username, password)
	
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Failed to fetch line items: %v\n", err)
		return
	}
	defer resp.Body.Close()
	
	body, _ := ioutil.ReadAll(resp.Body)
	fmt.Printf("Sample line items:\n%s\n", string(body))
}