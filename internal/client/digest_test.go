package client

import (
	"context"
	"crypto/md5"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetDigestAuth(t *testing.T) {
	c := NewODataClient("http://example.com/odata/", false)
	c.SetDigestAuth("admin", "secret")

	assert.Equal(t, "digest", c.authType)
	assert.Equal(t, "admin", c.username)
	assert.Equal(t, "secret", c.password)
	assert.NotNil(t, c.httpClient.Transport, "transport should be wrapped with digest")
}

func TestDigestAuthDoesNotSetBasicHeader(t *testing.T) {
	// Create a test server that captures the Authorization header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"d":{"results":[]}}`))
	}))
	defer server.Close()

	c := NewODataClient(server.URL+"/odata/", false)
	c.SetDigestAuth("admin", "secret")

	req, err := c.buildRequest(context.Background(), "GET", "", nil)
	require.NoError(t, err)

	// buildRequest should NOT set Basic auth header for digest clients
	assert.Empty(t, req.Header.Get("Authorization"), "digest client should not set Basic auth in buildRequest")

	// But the username/password should still be stored
	assert.Equal(t, "admin", c.username)
	assert.Equal(t, "secret", c.password)
}

func TestDigestAuthChallengeResponse(t *testing.T) {
	// Simulate a digest auth server:
	// 1st request -> 401 with WWW-Authenticate: Digest ...
	// 2nd request -> 200 if correct Authorization: Digest ... header
	realm := "testrealm"
	nonce := "dcd98b7102dd2f0e8b11d0f600bfb0c093"
	requestCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		authHeader := r.Header.Get("Authorization")

		if !strings.HasPrefix(authHeader, "Digest ") {
			// Send 401 challenge
			w.Header().Set("WWW-Authenticate",
				fmt.Sprintf(`Digest realm="%s", nonce="%s", qop="auth", algorithm=MD5`, realm, nonce))
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("Unauthorized"))
			return
		}

		// Verify it's a Digest response (basic validation)
		if strings.Contains(authHeader, fmt.Sprintf(`realm="%s"`, realm)) &&
			strings.Contains(authHeader, `username="admin"`) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"d":{"results":[{"ID":1,"Name":"Test"}]}}`))
			return
		}

		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("Bad credentials"))
	}))
	defer server.Close()

	c := NewODataClient(server.URL+"/odata/", false)
	c.SetDigestAuth("admin", "secret")

	req, err := c.buildRequest(context.Background(), "GET", "", nil)
	require.NoError(t, err)

	resp, err := c.httpClient.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode, "digest auth should succeed after challenge")
	// The digest library handles the 401 challenge internally, so the first call
	// should result in 2 actual HTTP requests (challenge + authenticated)
	assert.Equal(t, 2, requestCount, "expected challenge + authenticated request")
}

// helper for MD5 hashing (used in digest auth)
func md5Hash(data string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(data)))
}
