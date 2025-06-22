package test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/odata-mcp/go/internal/client"
	"github.com/odata-mcp/go/internal/constants"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCSRFTokenRetryMechanism tests the automatic retry on 403 in doRequest
// This specifically tests the retry logic that happens when a token becomes invalid
func TestCSRFTokenRetryMechanism(t *testing.T) {
	tokenFetchCount := 0
	postCount := 0
	validToken := "valid-token-789"
	
	// Track which token was used for each request
	tokensUsed := []string{}
	
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Handle CSRF token fetch
		if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
			tokenFetchCount++
			// Always return a valid token
			w.Header().Set(constants.CSRFTokenHeader, validToken)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
			return
		}
		
		// For POST requests
		if r.Method == http.MethodPost {
			postCount++
			token := r.Header.Get(constants.CSRFTokenHeader)
			tokensUsed = append(tokensUsed, token)
			
			// First request with any token fails with CSRF error
			// This simulates token expiry
			if postCount == 1 {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "403",
						"message": "CSRF token validation failed",
					},
				})
				return
			}
			
			// Subsequent requests succeed if they have the valid token
			if token == validToken {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{"ID": "1"},
				})
			} else {
				w.WriteHeader(http.StatusForbidden)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error": map[string]interface{}{
						"code":    "403", 
						"message": "Invalid token",
					},
				})
			}
		}
	}))
	defer server.Close()
	
	client := client.NewODataClient(server.URL, false)
	client.SetBasicAuth("user", "pass")
	
	// Make a create request
	// This should: fetch token -> POST (fail with 403) -> fetch new token -> POST (succeed)
	_, err := client.CreateEntity(context.Background(), "TestEntities", map[string]interface{}{"Name": "Test"})
	require.NoError(t, err)
	
	// Verify the flow
	assert.Equal(t, 2, tokenFetchCount, "Should fetch token twice (initial + after 403)")
	assert.Equal(t, 2, postCount, "Should make 2 POST requests (initial fail + retry)")
	assert.Equal(t, 2, len(tokensUsed), "Should have used 2 tokens")
	assert.Equal(t, validToken, tokensUsed[0], "First request should use the fetched token")
	assert.Equal(t, validToken, tokensUsed[1], "Retry should use the new token")
}

// TestCSRFProactiveFetch tests that modifying operations fetch tokens before making requests
func TestCSRFProactiveFetch(t *testing.T) {
	csrfToken := "proactive-token"
	fetchedBeforeRequest := false
	createRequestReceived := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Token fetch should happen before the actual request
		if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
			fetchedBeforeRequest = !createRequestReceived
			w.Header().Set(constants.CSRFTokenHeader, csrfToken)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
			return
		}

		// Create request
		if r.Method == http.MethodPost {
			createRequestReceived = true
			if r.Header.Get(constants.CSRFTokenHeader) == csrfToken {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"d": map[string]interface{}{
						"ID": "new",
						"Name": "Created",
					},
				})
			} else {
				w.WriteHeader(http.StatusForbidden)
			}
		}
	}))
	defer server.Close()

	client := client.NewODataClient(server.URL, false)
	client.SetBasicAuth("user", "pass")

	// CreateEntity should fetch token before making the request
	result, err := client.CreateEntity(context.Background(), "TestEntities", map[string]interface{}{
		"Name": "Test",
	})
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.True(t, fetchedBeforeRequest, "Token should be fetched before the create request")
	assert.True(t, createRequestReceived, "Create request should be received")
}

// TestCSRFTokenExpiry tests behavior when token expires between requests
func TestCSRFTokenExpiry(t *testing.T) {
	validToken := "valid-token-123"
	currentToken := validToken
	tokenFetchCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get(constants.CSRFTokenHeader) == constants.CSRFTokenFetch {
			tokenFetchCount++
			// On second fetch, return a new token
			if tokenFetchCount > 1 {
				currentToken = "new-token-456"
			}
			w.Header().Set(constants.CSRFTokenHeader, currentToken)
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{}})
			return
		}

		// First request succeeds with initial token
		if tokenFetchCount == 1 && r.Header.Get(constants.CSRFTokenHeader) == validToken {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{"ID": "1"}})
			return
		}

		// After token "expires", only new token works
		if tokenFetchCount > 1 && r.Header.Get(constants.CSRFTokenHeader) == currentToken {
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(map[string]interface{}{"d": map[string]interface{}{"ID": "2"}})
			return
		}

		// Old token returns 403
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code": "403",
				"message": "CSRF token expired",
			},
		})
	}))
	defer server.Close()

	client := client.NewODataClient(server.URL, false)
	client.SetBasicAuth("user", "pass")

	// First request - should fetch token and succeed
	_, err := client.CreateEntity(context.Background(), "TestEntities", map[string]interface{}{"Name": "First"})
	require.NoError(t, err)
	assert.Equal(t, 1, tokenFetchCount)

	// Second request - Python behavior: always fetches fresh token
	_, err = client.CreateEntity(context.Background(), "TestEntities", map[string]interface{}{"Name": "Second"})
	require.NoError(t, err)
	assert.Equal(t, 2, tokenFetchCount, "Should fetch new token for each operation (Python behavior)")
}