package test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestPROGRAMSetCreation tests creating ABAP programs through OData with CSRF token handling
func TestPROGRAMSetCreation(t *testing.T) {
	// Test data for ABAP program
	testProgram := map[string]interface{}{
		"Package":     "$VIBE_TEST",
		"Program":     "ZTEST_CSRF_" + fmt.Sprintf("%d", time.Now().Unix()),
		"ProgramType": "1",
		"SourceCode":  "REPORT ztest_program.\nWRITE: / 'Test program for CSRF'.",
		"Title":       "Test Program for CSRF Validation",
	}

	t.Run("MockServer", func(t *testing.T) {
		// Create mock server that simulates SAP behavior
		csrfToken := "mock-csrf-token-123"
		tokenRequested := false
		createRequested := false
		
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Log request for debugging
			t.Logf("Mock Request: %s %s", r.Method, r.URL.Path)
			t.Logf("Headers: %+v", r.Header)
			
			// Handle CSRF token fetch
			if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
				tokenRequested = true
				w.Header().Set(constants.CSRFTokenHeader, csrfToken)
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"EntitySets": []string{"PROGRAMSet"},
					},
				})
				return
			}
			
			// Handle metadata
			if strings.HasSuffix(r.URL.Path, "/$metadata") {
				w.Header().Set("Content-Type", "application/xml")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" Version="1.0">
  <edmx:DataServices>
    <Schema xmlns="http://schemas.microsoft.com/ado/2008/09/edm" Namespace="ZODD_000_SRV">
      <EntityType Name="PROGRAM">
        <Key><PropertyRef Name="Program"/></Key>
        <Property Name="Program" Type="Edm.String" Nullable="false" MaxLength="30"/>
        <Property Name="Package" Type="Edm.String" MaxLength="30"/>
        <Property Name="ProgramType" Type="Edm.String" MaxLength="1"/>
        <Property Name="SourceCode" Type="Edm.String"/>
        <Property Name="Title" Type="Edm.String" MaxLength="70"/>
      </EntityType>
      <EntityContainer Name="ZODD_000_SRV_Entities">
        <EntitySet Name="PROGRAMSet" EntityType="ZODD_000_SRV.PROGRAM"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`))
				return
			}
			
			// Handle program creation
			if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "PROGRAMSet") {
				createRequested = true
				
				// Check CSRF token
				if r.Header.Get(constants.CSRFTokenHeader) != csrfToken {
					w.WriteHeader(http.StatusForbidden)
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error": map[string]interface{}{
							"code":    "403",
							"message": map[string]interface{}{
								"lang":  "en",
								"value": "CSRF token validation failed",
							},
						},
					})
					return
				}
				
				// Validate request body
				var requestData map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&requestData); err != nil {
					w.WriteHeader(http.StatusBadRequest)
					return
				}
				
				// Simulate successful creation
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": requestData, // Echo back the created entity
				})
				return
			}
			
			// Default response
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
		}))
		defer server.Close()
		
		// Create client
		client := client.NewODataClient(server.URL, true)
		client.SetBasicAuth("testuser", "testpass")
		
		// Test direct client creation
		result, err := client.CreateEntity(context.Background(), "PROGRAMSet", testProgram)
		require.NoError(t, err)
		assert.NotNil(t, result)
		
		// Verify CSRF token was fetched
		assert.True(t, tokenRequested, "CSRF token should have been requested")
		assert.True(t, createRequested, "Create request should have been made")
		
		// Verify response contains created data
		if data, ok := result.Value.(map[string]interface{}); ok {
			assert.Equal(t, testProgram["Program"], data["Program"])
			assert.Equal(t, testProgram["Package"], data["Package"])
		}
	})
	
	t.Run("Integration", func(t *testing.T) {
		// Skip if environment variables are not set
		odataURL := os.Getenv("ODATA_URL")
		odataUser := os.Getenv("ODATA_USER")
		odataPass := os.Getenv("ODATA_PASS")
		
		if odataURL == "" || odataUser == "" || odataPass == "" {
			t.Skip("Skipping integration test: ODATA_URL, ODATA_USER, ODATA_PASS not set")
		}
		
		// Create real client
		client := client.NewODataClient(odataURL, true)
		client.SetBasicAuth(odataUser, odataPass)
		
		// Try to create a program
		// Note: This may fail due to backend validations (e.g., package restrictions)
		result, err := client.CreateEntity(context.Background(), "PROGRAMSet", testProgram)
		
		if err != nil {
			errStr := err.Error()
			
			// Check for CSRF token issues
			if strings.Contains(errStr, "403") && strings.Contains(errStr, "CSRF") {
				t.Fatalf("CSRF token validation failed - this should not happen with the fix: %v", err)
			}
			
			// Check for backend validation errors (expected)
			if strings.Contains(errStr, "package") || strings.Contains(errStr, "$VIBE_") {
				t.Logf("Expected backend validation error: %v", err)
				t.Log("The CSRF token handling is working correctly, but the backend rejected the request due to package restrictions")
				return
			}
			
			// Other errors
			t.Logf("Create operation failed with: %v", err)
		} else {
			// Success - verify response
			t.Log("Program created successfully!")
			assert.NotNil(t, result)
			
			if data, ok := result.Value.(map[string]interface{}); ok {
				t.Logf("Created program: %+v", data)
			}
		}
	})
}

// TestCSRFTokenRetryScenario tests a specific scenario where CSRF token fails and needs retry
func TestCSRFTokenRetryScenario(t *testing.T) {
	// This test simulates the exact scenario from the bug report
	csrfToken := "sap-csrf-token"
	tokenFetches := 0
	createAttempts := 0
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SAP-style CSRF token fetch
		if r.Header.Get("X-CSRF-Token") == "Fetch" {
			tokenFetches++
			w.Header().Set("X-CSRF-Token", csrfToken)
			w.Header().Set("Set-Cookie", "sap-usercontext=sap-client=100; path=/")
			w.WriteHeader(http.StatusOK)
			// SAP returns service document on token fetch
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"EntitySets": []interface{}{
						map[string]interface{}{"name": "PROGRAMSet"},
					},
				},
			})
			return
		}
		
		// Program creation endpoint
		if r.Method == "POST" && strings.Contains(r.URL.Path, "PROGRAMSet") {
			createAttempts++
			
			// First attempt always fails with CSRF error (simulating expired token)
			if createAttempts == 1 || r.Header.Get("X-CSRF-Token") != csrfToken {
				w.WriteHeader(http.StatusForbidden)
				// SAP-style error response
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code": "005056A509B11EE1B9A8FEC11C23378E",
						"message": map[string]interface{}{
							"lang":  "en",
							"value": "CSRF token validation failed",
						},
						"innererror": map[string]interface{}{
							"transactionid": "4DD97CCD6B390040E00684E9F2C3E12B",
							"timestamp":     "20250622080201.9627920",
						},
					},
				})
				return
			}
			
			// Success on retry with valid token
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"__metadata": map[string]interface{}{
						"type": "ZODD_000_SRV.PROGRAM",
						"uri":  fmt.Sprintf("%s/PROGRAMSet('%s')", r.Host, body["Program"]),
					},
					"Program":     body["Program"],
					"Package":     body["Package"],
					"ProgramType": body["ProgramType"],
					"SourceCode":  body["SourceCode"],
					"Title":       body["Title"],
				},
			})
			return
		}
	}))
	defer server.Close()
	
	// Create client with verbose logging
	client := client.NewODataClient(server.URL, true)
	client.SetBasicAuth("testuser", "testpass")
	
	// Attempt to create program - should handle CSRF retry automatically
	programData := map[string]interface{}{
		"Package":     "$VIBE_TEST",
		"Program":     "ZBUG_REPRO",
		"ProgramType": "1",
		"SourceCode":  "REPORT zbug_repro.\nWRITE: / 'Bug reproduction'.",
		"Title":       "Bug Reproduction Program",
	}
	
	result, err := client.CreateEntity(context.Background(), "PROGRAMSet", programData)
	require.NoError(t, err, "Should handle CSRF retry automatically")
	assert.NotNil(t, result)
	
	// Verify the retry flow
	assert.Equal(t, 2, tokenFetches, "Should fetch token twice (initial + after 403)")
	assert.Equal(t, 2, createAttempts, "Should attempt create twice (initial fail + retry)")
	
	// Verify created entity
	if data, ok := result.Value.(map[string]interface{}); ok {
		assert.Equal(t, "ZBUG_REPRO", data["Program"])
		assert.Equal(t, "$VIBE_TEST", data["Package"])
		t.Logf("Successfully created program: %+v", data)
	}
}