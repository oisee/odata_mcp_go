# CSRF Token Handling: Python vs Go Implementation Comparison

## Overview

This document compares how CSRF token handling is implemented in the Python and Go versions of the OData MCP server, specifically focusing on CREATE/UPDATE/DELETE operations.

## Python Implementation (odata_mcp)

### Token Fetching Strategy: **On-Demand with Retry**

The Python implementation uses an on-demand approach with automatic retry:

1. **Initial Request**: When a modifying operation (POST/PUT/MERGE/DELETE) is initiated, it checks if a CSRF token exists
2. **On-Demand Fetch**: If `requires_csrf=True` is passed to `_make_request()`, it fetches a fresh token before the request
3. **Automatic Retry**: If the request fails with 403 and CSRF-related error, it refetches the token and retries once

### Key Code Sections

```python
# From client.py, line 130-186
def _make_request(self, method: str, url: str, requires_csrf: bool = False, **kwargs) -> requests.Response:
    modifying_methods = ['POST', 'PUT', 'MERGE', 'PATCH', 'DELETE']
    is_modifying = method.upper() in modifying_methods

    # For modifying operations that require CSRF, always fetch a fresh token
    if is_modifying and requires_csrf:
        if not self._fetch_csrf_token():
            self._log_verbose("Failed to fetch CSRF token, proceeding without it")

    # ... make request ...

    # Handle CSRF token issues
    csrf_failed = (
        response.status_code == 403 and
        is_modifying and requires_csrf and
        ('CSRF token validation failed' in response.text or
         'csrf' in response.text.lower() or
         response.headers.get('x-csrf-token', '').lower() == 'required')
    )

    if csrf_failed and not hasattr(response, '_csrf_retry_attempted'):
        # Clear the invalid token and retry once
        self.csrf_token = None
        if self._fetch_csrf_token():
            # Retry request with new token
            response = self.session.request(method, url, **kwargs)
```

### Entity Operations

```python
# All entity operations pass requires_csrf=True
async def create_entity(...):
    response = await asyncio.to_thread(
        self._make_request, 'POST', url, params=params, json=entity_data, requires_csrf=True
    )

async def update_entity(...):
    response = await asyncio.to_thread(
        self._make_request, 'MERGE', url, params=params, json=entity_data, requires_csrf=True
    )

async def delete_entity(...):
    response = await asyncio.to_thread(
        self._make_request, 'DELETE', url, requires_csrf=True
    )
```

## Go Implementation (odata_mcp_go)

### Token Fetching Strategy: **Proactive with Fallback**

The Go implementation uses a proactive approach:

1. **Proactive Fetch**: Before any modifying operation, it checks if a token exists and fetches one if missing
2. **Automatic Fallback**: If a request fails with 403 and no token exists, `doRequest()` automatically fetches a token and retries

### Key Code Sections

```go
// From client.go, line 92-118
func (c *ODataClient) doRequest(req *http.Request) (*http.Response, error) {
    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, fmt.Errorf("HTTP request failed: %w", err)
    }

    // Handle CSRF token requirement (SAP-specific)
    if resp.StatusCode == http.StatusForbidden && c.csrfToken == "" {
        resp.Body.Close()
        
        // Try to fetch CSRF token
        if err := c.fetchCSRFToken(req.Context()); err != nil {
            return nil, fmt.Errorf("failed to fetch CSRF token: %w", err)
        }

        // Retry original request with CSRF token
        req.Header.Set(constants.CSRFTokenHeader, c.csrfToken)
        return c.doRequest(req)
    }

    return resp, nil
}
```

### Entity Operations

```go
// All entity operations proactively fetch CSRF token
func (c *ODataClient) CreateEntity(ctx context.Context, entitySet string, data map[string]interface{}) (*models.ODataResponse, error) {
    // Proactively fetch CSRF token for POST operations
    if c.csrfToken == "" {
        if err := c.fetchCSRFToken(ctx); err != nil {
            if c.verbose {
                fmt.Fprintf(os.Stderr, "[VERBOSE] Failed to fetch CSRF token: %v\n", err)
            }
            // Continue without token - some services might not require it
        }
    }
    // ... continue with request ...
}

// Similar pattern for UpdateEntity and DeleteEntity
```

## Key Differences

### 1. **Fetching Strategy**
- **Python**: On-demand - fetches a fresh token for each modifying request when `requires_csrf=True`
- **Go**: Proactive - attempts to fetch token before the request if missing, but reuses existing tokens

### 2. **Token Reuse**
- **Python**: Always fetches a fresh token for each modifying operation
- **Go**: Reuses the same token across multiple operations until it fails

### 3. **Error Handling**
- **Python**: 
  - Explicitly checks for CSRF-related error messages
  - Retries only once with `_csrf_retry_attempted` flag
  - Clears token on failure
- **Go**: 
  - Only checks status code (403) and absence of token
  - Recursive retry through `doRequest()`
  - Keeps token after successful fetch

### 4. **Fallback Behavior**
- **Python**: Logs warning but continues without token if fetch fails
- **Go**: Logs warning in verbose mode and continues without token

### 5. **Token Persistence**
- **Python**: Token stored in session but fetched fresh for each operation
- **Go**: Token stored in client struct and reused until failure

## Recommendations

1. **Token Freshness**: The Python approach of fetching fresh tokens for each operation is more robust for long-running sessions where tokens might expire

2. **Performance**: The Go approach of reusing tokens is more efficient for bulk operations but may fail if tokens expire

3. **Error Detection**: The Python implementation's explicit CSRF error detection is more comprehensive

4. **Best Practice**: Consider implementing a hybrid approach:
   - Reuse tokens for performance (like Go)
   - Check token age and refresh if stale
   - Use comprehensive error detection (like Python)
   - Implement exponential backoff for retries

## Test Coverage

Both implementations have CSRF-specific tests:
- Python: Implicitly tested through operation tests
- Go: Explicit CSRF test suite in `internal/test/csrf_test.go`

The Go test suite provides better visibility into CSRF behavior with dedicated test cases for token fetching, reuse, and error scenarios.