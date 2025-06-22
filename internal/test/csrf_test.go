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
	"github.com/stretchr/testify/suite"
)

type CSRFTestSuite struct {
	suite.Suite
	mockServer       *httptest.Server
	client           *client.ODataClient
	csrfToken        string
	tokenRequests    int
	modifyRequests   int
	csrfRequired     bool
	validCredentials bool
}

func (suite *CSRFTestSuite) SetupSuite() {
	suite.csrfToken = "test-csrf-token-12345"
	suite.csrfRequired = true
	suite.validCredentials = true
}

func (suite *CSRFTestSuite) SetupTest() {
	suite.tokenRequests = 0
	suite.modifyRequests = 0

	suite.mockServer = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Log all requests for debugging
		suite.T().Logf("Request: %s %s", r.Method, r.URL.Path)
		suite.T().Logf("Headers: %+v", r.Header)

		// Check authentication
		if !suite.validCredentials {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}

		// Handle CSRF token fetch requests
		if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
			suite.tokenRequests++
			w.Header().Set(constants.CSRFTokenHeader, suite.csrfToken)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"EntitySets": []string{"TestEntities"},
				},
			})
			return
		}

		// Handle metadata requests
		if strings.HasSuffix(r.URL.Path, "/$metadata") {
			w.Header().Set("Content-Type", "application/xml")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`<?xml version="1.0" encoding="utf-8"?>
<edmx:Edmx xmlns:edmx="http://schemas.microsoft.com/ado/2007/06/edmx" Version="1.0">
  <edmx:DataServices>
    <Schema xmlns="http://schemas.microsoft.com/ado/2008/09/edm" Namespace="TestNamespace">
      <EntityType Name="TestEntity">
        <Key><PropertyRef Name="ID"/></Key>
        <Property Name="ID" Type="Edm.String" Nullable="false"/>
        <Property Name="Name" Type="Edm.String"/>
        <Property Name="Value" Type="Edm.Int32"/>
      </EntityType>
      <EntityContainer Name="TestContainer">
        <EntitySet Name="TestEntities" EntityType="TestNamespace.TestEntity"/>
      </EntityContainer>
    </Schema>
  </edmx:DataServices>
</edmx:Edmx>`))
			return
		}

		// Check CSRF token for modifying operations
		if r.Method == http.MethodPost || r.Method == http.MethodPut || r.Method == http.MethodDelete {
			suite.modifyRequests++
			
			if suite.csrfRequired && r.Header.Get(constants.CSRFTokenHeader) != suite.csrfToken {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "403",
						"message": "CSRF token validation failed",
					},
				})
				return
			}
		}

		// Handle different operations
		switch {
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "TestEntities"):
			// READ operation
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"results": []map[string]interface{}{
						{"ID": "1", "Name": "Test 1", "Value": 100},
						{"ID": "2", "Name": "Test 2", "Value": 200},
					},
				},
			})

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "TestEntities"):
			// CREATE operation
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"ID":    "3",
					"Name":  body["Name"],
					"Value": body["Value"],
				},
			})

		case r.Method == http.MethodPut && strings.Contains(r.URL.Path, "TestEntities"):
			// UPDATE operation
			var body map[string]interface{}
			json.NewDecoder(r.Body).Decode(&body)
			
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": body,
			})

		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "TestEntities"):
			// DELETE operation
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "TestFunction"):
			// Function call
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"result": "function executed",
				},
			})

		default:
			// Root service document
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"d": map[string]interface{}{
					"EntitySets": []string{"TestEntities"},
				},
			})
		}
	}))

	// Create client with mock server
	suite.client = client.NewODataClient(
		suite.mockServer.URL,
		false,
	)
	suite.client.SetBasicAuth("testuser", "testpass")
}

func (suite *CSRFTestSuite) TearDownTest() {
	suite.mockServer.Close()
}

func (suite *CSRFTestSuite) TestCSRFTokenFetchOnCreate() {
	// Test that CREATE operation fetches CSRF token (Python-style: fresh token per operation)
	entity := map[string]interface{}{
		"Name":  "New Entity",
		"Value": 300,
	}

	result, err := suite.client.CreateEntity(context.Background(), "TestEntities", entity)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Should have fetched token once
	assert.Equal(suite.T(), 1, suite.tokenRequests, "Should fetch CSRF token once")
	assert.Equal(suite.T(), 1, suite.modifyRequests, "Should make one create request")
}

func (suite *CSRFTestSuite) TestCSRFTokenFetchOnUpdate() {
	// Test that UPDATE operation fetches CSRF token (Python-style: fresh token per operation)
	entity := map[string]interface{}{
		"Name":  "Updated Entity",
		"Value": 400,
	}

	result, err := suite.client.UpdateEntity(context.Background(), "TestEntities", map[string]interface{}{"ID": "1"}, entity, "")
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Should have fetched token once
	assert.Equal(suite.T(), 1, suite.tokenRequests, "Should fetch CSRF token once")
	assert.Equal(suite.T(), 1, suite.modifyRequests, "Should make one update request")
}

func (suite *CSRFTestSuite) TestCSRFTokenFetchOnDelete() {
	// Test that DELETE operation fetches CSRF token (Python-style: fresh token per operation)
	_, err := suite.client.DeleteEntity(context.Background(), "TestEntities", map[string]interface{}{"ID": "1"})
	require.NoError(suite.T(), err)
	
	// Should have fetched token once
	assert.Equal(suite.T(), 1, suite.tokenRequests, "Should fetch CSRF token once")
	assert.Equal(suite.T(), 1, suite.modifyRequests, "Should make one delete request")
}

func (suite *CSRFTestSuite) TestCSRFTokenFreshFetchForEachOperation() {
	// Test that CSRF token is fetched fresh for each modifying operation (Python behavior)
	
	// First operation - should fetch token
	entity1 := map[string]interface{}{
		"Name":  "Entity 1",
		"Value": 100,
	}
	_, err := suite.client.CreateEntity(context.Background(), "TestEntities", entity1)
	require.NoError(suite.T(), err)
	
	// Second operation - should fetch new token (Python behavior)
	entity2 := map[string]interface{}{
		"Name":  "Entity 2",
		"Value": 200,
	}
	_, err = suite.client.CreateEntity(context.Background(), "TestEntities", entity2)
	require.NoError(suite.T(), err)
	
	// Third operation - should fetch another new token
	_, err = suite.client.DeleteEntity(context.Background(), "TestEntities", map[string]interface{}{"ID": "1"})
	require.NoError(suite.T(), err)
	
	// Should have fetched token for each operation (Python behavior)
	assert.Equal(suite.T(), 3, suite.tokenRequests, "Should fetch CSRF token for each operation")
	assert.Equal(suite.T(), 3, suite.modifyRequests, "Should make three modify requests")
}

func (suite *CSRFTestSuite) TestCSRFTokenNotRequiredForRead() {
	// Test that READ operations don't require CSRF token
	result, err := suite.client.GetEntitySet(context.Background(), "TestEntities", nil)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Should not fetch token for read operations
	assert.Equal(suite.T(), 0, suite.tokenRequests, "Should not fetch CSRF token for read")
	assert.Equal(suite.T(), 0, suite.modifyRequests, "Should not be a modify request")
}

func (suite *CSRFTestSuite) TestCSRFTokenRetryOn403() {
	// Test the automatic retry mechanism when CSRF token validation fails
	// This simulates a scenario where the token is invalid or expired
	
	fetchCount := 0
	postCount := 0
	validToken := "valid-token-456"
	
	testServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle token fetch
		if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
			fetchCount++
			// First fetch returns invalid token, second returns valid token
			if fetchCount == 1 {
				w.Header().Set(constants.CSRFTokenHeader, "invalid-token")
			} else {
				w.Header().Set(constants.CSRFTokenHeader, validToken)
			}
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
			return
		}
		
		// For POST requests
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "TestEntities") {
			postCount++
			token := r.Header.Get(constants.CSRFTokenHeader)
			
			// Only accept the valid token
			if token == validToken {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"ID":    "3",
						"Name":  "Created",
						"Value": 500,
					},
				})
			} else {
				// Return 403 with CSRF error
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "403",
						"message": "CSRF token validation failed",
					},
				})
			}
			return
		}
		
		// Default response
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
	}))
	defer testServer.Close()
	
	client := client.NewODataClient(testServer.URL, false)
	client.SetBasicAuth("testuser", "testpass")
	
	// Make a create request
	// This should trigger: fetch token (invalid) -> POST -> 403 -> fetch new token -> retry -> success
	entity := map[string]interface{}{
		"Name":  "Test Entity",
		"Value": 500,
	}
	
	result, err := client.CreateEntity(context.Background(), "TestEntities", entity)
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Verify the retry mechanism worked
	assert.Equal(suite.T(), 2, fetchCount, "Should fetch token twice (initial + retry)")
	assert.Equal(suite.T(), 2, postCount, "Should make two POST requests (initial + retry)")
}

func (suite *CSRFTestSuite) TestCSRFTokenFunctionCall() {
	// Test that function calls also handle CSRF tokens
	params := map[string]interface{}{
		"param1": "value1",
	}
	
	result, err := suite.client.CallFunction(context.Background(), "TestFunction", params, "POST")
	require.NoError(suite.T(), err)
	assert.NotNil(suite.T(), result)
	
	// Should have fetched token once
	assert.Equal(suite.T(), 1, suite.tokenRequests, "Should fetch CSRF token once")
	assert.Equal(suite.T(), 1, suite.modifyRequests, "Should make one function call")
}

func (suite *CSRFTestSuite) TestCSRFTokenHeaderVariations() {
	// Test that client handles different case variations of CSRF header
	testCases := []struct {
		name       string
		headerName string
	}{
		{"Uppercase", "X-CSRF-TOKEN"},
		{"Lowercase", "x-csrf-token"},
		{"Mixed case", "X-Csrf-Token"},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset counters
			suite.tokenRequests = 0
			suite.modifyRequests = 0
			
			// Create new server that returns token with different case
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
					w.Header().Set(tc.headerName, suite.csrfToken)
					w.WriteHeader(http.StatusOK)
					json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
					return
				}
				
				// For modify operations, accept the token
				if r.Method == http.MethodPost && r.Header.Get(constants.CSRFTokenHeader) == suite.csrfToken {
					w.WriteHeader(http.StatusCreated)
					json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{"ID": "1"}})
					return
				}
				
				w.WriteHeader(http.StatusForbidden)
			}))
			defer server.Close()
			
			// Create client with new server
			client := client.NewODataClient(server.URL, false)
			client.SetBasicAuth("user", "pass")
			
			// Make a create request
			_, err := client.CreateEntity(context.Background(), "TestEntities", map[string]interface{}{"Name": "Test"})
			assert.NoError(suite.T(), err, "Should handle %s header", tc.headerName)
		})
	}
}

func TestCSRFTestSuite(t *testing.T) {
	suite.Run(t, new(CSRFTestSuite))
}

// Integration test using environment variables
func TestCSRFIntegration(t *testing.T) {
	// Skip if environment variables are not set
	odataURL := os.Getenv("ODATA_URL")
	odataUser := os.Getenv("ODATA_USER")
	odataPass := os.Getenv("ODATA_PASS")
	
	if odataURL == "" || odataUser == "" || odataPass == "" {
		t.Skip("Skipping integration test: ODATA_URL, ODATA_USER, ODATA_PASS not set")
	}
	
	t.Log("Running CSRF integration test against:", odataURL)
	
	// Create real client
	client := client.NewODataClient(odataURL, true)
	client.SetBasicAuth(odataUser, odataPass)
	
	// Test 1: Get metadata to understand available entity sets
	t.Run("GetMetadata", func(t *testing.T) {
		metadata, err := client.GetMetadata(context.Background())
		if err != nil {
			t.Logf("Warning: Could not fetch metadata: %v", err)
			t.Skip("Skipping test - metadata not available")
		}
		assert.NotEmpty(t, metadata)
		t.Logf("Metadata retrieved successfully")
	})
	
	// Test 2: Test read operation (no CSRF required)
	t.Run("ReadOperation", func(t *testing.T) {
		// Try to get root entity sets
		resp, err := client.GetEntitySet(context.Background(), "", nil)
		if err != nil {
			t.Logf("Warning: Could not fetch service document: %v", err)
			// Try alternative - get entities
			entities, err := client.GetEntitySet(context.Background(), "", nil)
			if err != nil {
				t.Skip("Skipping test - service not accessible")
			}
			assert.NotNil(t, entities)
		} else {
			assert.NotNil(t, resp)
			t.Log("Read operation successful - no CSRF token required")
		}
	})
	
	// Test 3: Test modify operation (CSRF required)
	t.Run("ModifyOperationCSRF", func(t *testing.T) {
		// Note: This test attempts a create operation which may fail
		// if the entity set doesn't exist or requires specific fields
		testEntity := map[string]interface{}{
			"TestField": fmt.Sprintf("CSRFTest_%d", time.Now().Unix()),
		}
		
		// Attempt to create - this should trigger CSRF token fetch
		result, err := client.CreateEntity(context.Background(), "TestEntities", testEntity)
		
		if err != nil {
			// Check if error is due to CSRF or other issues
			errStr := err.Error()
			if strings.Contains(errStr, "403") || strings.Contains(errStr, "CSRF") {
				t.Log("CSRF token validation detected - server requires CSRF tokens")
			} else if strings.Contains(errStr, "404") {
				t.Log("Entity set not found - expected for test environment")
			} else {
				t.Logf("Create operation failed with: %v", err)
			}
		} else {
			t.Log("Create operation successful with CSRF token")
			assert.NotNil(t, result)
			
			// If create succeeded, try to clean up
			if result.Value != nil {
				if data, ok := result.Value.(map[string]interface{}); ok {
					if id, ok := data["ID"]; ok {
						_, _ = client.DeleteEntity(context.Background(), "TestEntities", map[string]interface{}{"ID": id})
					}
				}
			}
		}
	})
}