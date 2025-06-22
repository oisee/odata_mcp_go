# OData MCP Server Test Suite

This directory contains comprehensive tests for the OData MCP server, with a focus on CSRF token tracking and MCP protocol compliance.

## Test Files

### csrf_test.go
Tests CSRF token handling for all modifying operations:
- **Token Fetching**: Verifies automatic CSRF token fetching for CREATE, UPDATE, DELETE operations
- **Token Reuse**: Ensures tokens are cached and reused across multiple requests
- **Retry Logic**: Tests automatic retry when receiving 403 Forbidden responses
- **Header Variations**: Validates support for different case variations of X-CSRF-Token header
- **Integration Tests**: Real-world testing against actual OData services

### mcp_protocol_test.go
Tests MCP (Model Context Protocol) compliance:
- **Protocol Initialization**: Validates proper handshake and capability negotiation
- **Tool Discovery**: Tests tools/list endpoint functionality
- **Tool Execution**: Validates tools/call with proper argument validation
- **Error Handling**: Ensures proper JSON-RPC error codes and messages
- **Concurrent Operations**: Tests thread-safety and concurrent request handling

### mcp_audit_test.go
Comprehensive MCP protocol audit and validation:
- **Request Validation**: Ensures all requests follow JSON-RPC 2.0 specification
- **Schema Validation**: Validates tool arguments against defined schemas
- **Response Consistency**: Verifies all responses have correct structure
- **Error Propagation**: Tests OData error to MCP error mapping
- **Notification Handling**: Validates proper handling of one-way notifications

## Running Tests

### Basic Test Execution
```bash
# Run all tests
go test -v ./internal/test/...

# Run specific test suite
go test -v -run TestCSRFTestSuite ./internal/test/...

# Run with coverage
go test -v -cover -coverprofile=coverage.out ./internal/test/...
```

### Using Environment Variables
```bash
# Set OData service credentials for integration tests
export ODATA_URL="https://your-odata-service.com/path"
export ODATA_USER="your-username"
export ODATA_PASS="your-password"

# Run integration tests
go test -v -run TestCSRFIntegration ./internal/test/...
```

### Using the Test Script
```bash
# Run comprehensive test suite
./test_csrf_operations.sh

# The script will:
# - Build the server
# - Run all CSRF tests
# - Run all MCP protocol tests
# - Generate coverage reports
# - Run performance benchmarks
```

## Test Categories

### 1. CSRF Token Tests
- **Proactive Fetching**: Tests that tokens are fetched before modifying operations
- **Token Caching**: Verifies tokens are reused to minimize network calls
- **Error Recovery**: Tests automatic retry with token after 403 responses
- **Compatibility**: Ensures support for various header formats

### 2. MCP Protocol Tests
- **Lifecycle**: Initialize → tools/list → tools/call → shutdown
- **Validation**: Request/response format, required fields, error codes
- **Concurrency**: Multiple simultaneous requests
- **Performance**: Benchmarks for common operations

### 3. Integration Tests
- **Real Services**: Tests against actual OData endpoints
- **End-to-End**: Full workflow from MCP client to OData service
- **Error Scenarios**: Network issues, authentication failures, etc.

## Mock Server

Tests use an embedded HTTP test server that simulates OData responses:
- Configurable CSRF token requirements
- Simulated authentication
- Standard OData responses
- Error injection for edge case testing

## Best Practices

1. **Test Isolation**: Each test creates its own server instance
2. **Cleanup**: Proper teardown of resources after each test
3. **Assertions**: Clear, descriptive assertions with helpful messages
4. **Coverage**: Aim for >80% code coverage
5. **Performance**: Include benchmarks for critical paths

## Extending Tests

To add new tests:
1. Create test functions following Go conventions
2. Use testify suite for complex test scenarios
3. Add integration tests for new OData features
4. Update this README with new test descriptions

## Troubleshooting

Common issues:
- **Environment not set**: Tests will use mock server if ODATA_* vars are missing
- **Build failures**: Ensure all dependencies are installed with `go mod download`
- **Timeout errors**: Increase timeout values for slow networks
- **CSRF failures**: Check if target service actually requires CSRF tokens