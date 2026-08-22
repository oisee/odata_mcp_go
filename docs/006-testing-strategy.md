# OData MCP Bridge: Testing Strategy

*For Contributors and Maintainers*

---

## Overview

The OData MCP Bridge uses a **three-tier testing strategy** to balance fast iteration with comprehensive real-world validation:

```
┌─────────────────────────────────────────────────────────────┐
│                    Testing Pyramid                          │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│                    ┌───────────┐                            │
│                    │    SAP    │  ← Real system tests       │
│                    │  (Tier 3) │    (contributors w/ SAP)   │
│                    └───────────┘                            │
│                                                             │
│               ┌─────────────────────┐                       │
│               │      Northwind      │  ← Public OData       │
│               │       (Tier 2)      │    (optional CI)      │
│               └─────────────────────┘                       │
│                                                             │
│         ┌─────────────────────────────────┐                 │
│         │       Mock Server (Tier 1)      │  ← Default      │
│         │    Embedded httptest.Server     │    (always)     │
│         └─────────────────────────────────┘                 │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

---

## Tier 1: Mock Server (Default)

All tests run against an **embedded mock OData server** by default. No external dependencies required.

### How It Works

```go
// SetupSuite creates embedded mock server if no env vars set
func (suite *MCPProtocolTestSuite) SetupSuite() {
    suite.serviceURL = os.Getenv("ODATA_URL")
    if suite.serviceURL == "" {
        suite.mockServer = httptest.NewServer(createMockODataHandler(suite.T()))
        suite.serviceURL = suite.mockServer.URL
        suite.T().Cleanup(func() {
            if suite.mockServer != nil {
                suite.mockServer.Close()
            }
        })
    }
}
```

### Run Mock Tests

```bash
go test ./...
```

### What Mock Tests Cover

- MCP protocol compliance (JSON-RPC, error codes)
- Tool generation from metadata
- Request/response formatting
- CSRF token handling simulation
- Concurrent request handling

---

## Tier 2: Northwind (Public OData)

The [Northwind OData service](https://services.odata.org/) provides real OData V2/V4 endpoints for testing.

### Run with Northwind

```bash
# V2 (EDMX XML metadata, $inlinecount)
ODATA_URL="https://services.odata.org/V2/Northwind/Northwind.svc" \
  go test -v ./internal/test/...

# V4 (CSDL metadata, $count)
ODATA_URL="https://services.odata.org/V4/Northwind/Northwind.svc" \
  go test -v ./internal/test/...
```

### What Northwind Tests

- Real HTTP communication
- OData V2 vs V4 differences:
  - V2: EDMX XML metadata, `$inlinecount=allpages`
  - V4: CSDL metadata, `$count=true`
- Metadata parsing from live service
- Standard OData query operations

### Limitations

- No SAP-specific annotations
- No CSRF token requirements
- No authentication
- No GUID formatting quirks

---

## Tier 3: SAP Integration (Contributor Access)

Full integration testing against real SAP OData services. Requires SAP system access.

### Environment Variables

| Variable | Description | Example |
|----------|-------------|---------|
| `ODATA_URL` | Full OData service URL | `http://host:50000/sap/opu/odata/sap/SERVICE_SRV` |
| `ODATA_USERNAME` | SAP username | `DEVELOPER` |
| `ODATA_PASSWORD` | SAP password | `secret123` |

### Run with SAP

```bash
ODATA_URL="http://vhcala4hci:50000/sap/opu/odata/sap/EPM_REF_APPS_PROD_MAN_SRV" \
ODATA_USERNAME="DEVELOPER" \
ODATA_PASSWORD="MyPassword" \
  go test -v ./internal/test/...
```

### Recommended SAP Services for Testing

| Service | Description | Entity Sets |
|---------|-------------|-------------|
| `EPM_REF_APPS_PROD_MAN_SRV` | EPM Product Management | Products, Suppliers, Categories |
| `SEPMRA_PROD_MAN` | Smart Template Product Mgmt | Products with drafts |
| `GWSAMPLE_BASIC` | Gateway Sample (if activated) | Business Partners, Products, Orders |

### What SAP Tests Cover

- SAP-specific CSRF token handling
- GUID property auto-formatting (`guid'...'`)
- SAP annotations (`sap:creatable`, `sap:updatable`, etc.)
- Multi-schema metadata parsing
- Legacy date format (`/Date(...)`)
- Real authentication flows

---

## Test Architecture Details

### Dynamic Tool Name Assertions

Tests check for **tool name patterns** rather than hardcoded names, since actual tools are generated dynamically:

```go
// Good: Check prefixes
expectedPrefixes := []string{
    "odata_service_info_",
    "filter_",
    "get_",
    "create_",
    "update_",
    "delete_",
}

// Bad: Hardcoded names (don't use)
// expectedTools := []string{"create_entity", "update_entity"}  // ❌
```

### JSON-RPC ID Type Handling

JSON numbers unmarshal as `float64` in Go. The test client converts IDs for map lookup:

```go
// Convert ID to int for map lookup (JSON numbers unmarshal as float64)
var lookupID interface{} = resp.ID
if f, ok := resp.ID.(float64); ok {
    lookupID = int(f)
}
```

### Scanner Buffer Size

The default `bufio.Scanner` has a 64KB buffer, but `tools/list` responses can be large (Northwind: 80KB, SAP: 33KB). Use a 1MB buffer:

```go
scanner := bufio.NewScanner(stdout)
scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer
```

### Startup Wait for Remote Services

Remote services (Northwind over internet, SAP over network) need time to fetch metadata during startup. The test client uses extended wait times:

```go
startupWait := 500 * time.Millisecond
if strings.Contains(serviceURL, "odata.org") || strings.Contains(serviceURL, "sap") {
    startupWait = 5 * time.Second
}
```

### Cleanup with t.Cleanup()

We use `t.Cleanup()` instead of `TearDownSuite` for guaranteed cleanup ([testify #1123](https://github.com/stretchr/testify/issues/1123)):

```go
suite.T().Cleanup(func() {
    if suite.mockServer != nil {
        suite.mockServer.Close()
    }
})
```

---

## Adding New Tests

### For Mock-Only Tests

1. Add test to existing suite in `internal/test/`
2. Use dynamic assertions (prefixes, patterns)
3. Run: `go test ./internal/test/...`

### For SAP-Specific Behavior

1. Document in test what SAP behavior is being tested
2. Skip if `ODATA_URL` doesn't contain SAP patterns:
   ```go
   if !strings.Contains(os.Getenv("ODATA_URL"), "sap") {
       t.Skip("Skipping SAP-specific test")
   }
   ```
3. Use build tags for future tier separation (planned)

---

## CI/CD Recommendations

### GitHub Actions Example

```yaml
jobs:
  test-mock:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: '1.22'
      - run: go test ./...

  test-northwind:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: |
          ODATA_URL="https://services.odata.org/V2/Northwind/Northwind.svc" \
            go test -v ./internal/test/...

  # SAP tests require self-hosted runner with SAP access
  test-sap:
    runs-on: [self-hosted, sap-access]
    if: github.event_name == 'push' && github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - run: go test -v ./internal/test/...
    env:
      ODATA_URL: ${{ secrets.SAP_ODATA_URL }}
      ODATA_USERNAME: ${{ secrets.SAP_USERNAME }}
      ODATA_PASSWORD: ${{ secrets.SAP_PASSWORD }}
```

---

## Troubleshooting

### Tests Timeout

**Symptom:** Tests hang for 5+ seconds then fail with "timeout waiting for response"

**Cause:** Usually a JSON-RPC ID type mismatch (fixed in Jan 2026)

**Solution:** Ensure IDs are converted to `int` before map lookup

### Mock Server Not Starting

**Symptom:** "connection refused" errors

**Cause:** Mock server URL not being passed to binary

**Solution:** Check `suite.serviceURL` is set before starting odata-mcp

### SAP Service Returns 403

**Symptom:** "No service found for namespace..." error

**Cause:** Service exists but not published on Gateway hub

**Solution:** Activate service in `/IWFND/MAINT_SERVICE` or use different service

---

---

## Future: Test Infrastructure Refactoring

### Current State

Test files have duplicated mock infrastructure:

| File | Lines | Mock Pattern |
|------|-------|--------------|
| `mcp_protocol_test.go` | 680 | MCPClient + mock handler |
| `csrf_test.go` | 500+ | Inline mock with CSRF |
| `odata_v4_test.go` | 300+ | Inline V4 mock |
| `query_translation_test.go` | 260+ | Inline V2/V4 mock |

**Duplicated accidentally:**
- V2/V4 metadata XML strings (copy-pasted)
- CSRF token handling logic
- Basic entity response structures

### Proposed Refactoring

Extract shared test utilities to `internal/test/testutil/`:

```
internal/test/
├── testutil/
│   ├── mcp_client.go        # MCPClient for MCP protocol tests
│   ├── mock_metadata.go     # V2/V4 metadata XML constants
│   ├── mock_handlers.go     # Configurable OData mock
│   └── helpers.go           # Common assertions, setup
├── mcp_protocol_test.go
├── csrf_test.go
└── ...
```

### Configurable Mock Builder

```go
// internal/test/testutil/mock_odata.go
package testutil

type MockODataConfig struct {
    Version     string   // "2.0" or "4.0"
    EnableCSRF  bool
    EntitySets  []string
    CSRFToken   string
    TrackCalls  bool
}

func NewMockOData(cfg MockODataConfig) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        // Handle /$metadata
        if strings.HasSuffix(r.URL.Path, "/$metadata") {
            if cfg.Version == "4.0" {
                w.Write([]byte(V4MetadataTemplate(cfg.EntitySets)))
            } else {
                w.Write([]byte(V2MetadataTemplate(cfg.EntitySets)))
            }
            return
        }

        // Handle CSRF
        if cfg.EnableCSRF {
            if r.Header.Get("X-CSRF-Token") == "Fetch" {
                w.Header().Set("X-CSRF-Token", cfg.CSRFToken)
                return
            }
        }
        // ...
    })
}
```

### Usage After Refactoring

```go
// Before: 50+ lines of inline mock
server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
    // ... lots of duplicated code
}))

// After: 3 lines
server := httptest.NewServer(testutil.NewMockOData(testutil.MockODataConfig{
    Version:    "4.0",
    EnableCSRF: true,
    EntitySets: []string{"Products", "Categories"},
}))
```

### Benefits

1. **DRY**: Single source for V2/V4 metadata
2. **Consistency**: Same CSRF handling everywhere
3. **Maintainability**: Fix bugs in one place
4. **Easier testing**: Less boilerplate for new tests

### Priority

Medium - Current tests work, but refactoring would reduce ~500 lines of duplication and make adding new tests easier.

---

## References

- [Go httptest documentation](https://pkg.go.dev/net/http/httptest)
- [Testify suite patterns](https://pkg.go.dev/github.com/stretchr/testify/suite)
- [OData.org test services](https://www.odata.org/odata-services/)
- [SAP Gateway demo services](https://help.sap.com/docs/SAP_NETWEAVER_AS_ABAP_FOR_SOH_740/0ce52cdd5b1f41f5b01c3bbe9f7df1e2/59fbcd80aa3a4c1cb0a90e81e00c3d8d.html)
