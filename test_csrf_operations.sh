#!/bin/bash

# Comprehensive test script for CSRF token tracking in OData MCP server
# This script tests all modifying operations and verifies CSRF token handling

set -e

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check for required environment variables
if [ -z "$ODATA_URL" ] || [ -z "$ODATA_USER" ] || [ -z "$ODATA_PASS" ]; then
    echo -e "${YELLOW}Warning: ODATA_URL, ODATA_USER, or ODATA_PASS not set${NC}"
    echo "Using mock server for testing"
    export ODATA_URL="http://localhost:8080/mock"
    export ODATA_USER="testuser"
    export ODATA_PASS="testpass"
fi

echo -e "${BLUE}=== CSRF Token Tracking Test Suite ===${NC}"
echo -e "Service URL: $ODATA_URL"
echo ""

# Function to run Go tests with nice output
run_test() {
    local test_name=$1
    local test_pattern=$2
    
    echo -e "${BLUE}Running: $test_name${NC}"
    
    if go test -v -run "$test_pattern" ./internal/test/... -count=1; then
        echo -e "${GREEN}✓ $test_name passed${NC}"
    else
        echo -e "${RED}✗ $test_name failed${NC}"
        return 1
    fi
    echo ""
}

# Build the server first
echo -e "${BLUE}Building OData MCP server...${NC}"
go build -o ./odata-mcp ./cmd/odata-mcp
echo -e "${GREEN}✓ Build successful${NC}"
echo ""

# Run CSRF-specific tests
echo -e "${YELLOW}=== CSRF Token Tests ===${NC}"
run_test "CSRF Token Fetch on Create" "TestCSRFTestSuite/TestCSRFTokenFetchOnCreate"
run_test "CSRF Token Fetch on Update" "TestCSRFTestSuite/TestCSRFTokenFetchOnUpdate"
run_test "CSRF Token Fetch on Delete" "TestCSRFTestSuite/TestCSRFTokenFetchOnDelete"
run_test "CSRF Token Reuse" "TestCSRFTestSuite/TestCSRFTokenReuseAcrossRequests"
run_test "CSRF Not Required for Read" "TestCSRFTestSuite/TestCSRFTokenNotRequiredForRead"
run_test "CSRF Retry on 403" "TestCSRFTestSuite/TestCSRFTokenRetryOn403"
run_test "CSRF Function Calls" "TestCSRFTestSuite/TestCSRFTokenFunctionCall"
run_test "CSRF Header Variations" "TestCSRFTestSuite/TestCSRFTokenHeaderVariations"

# Run MCP Protocol tests
echo -e "${YELLOW}=== MCP Protocol Tests ===${NC}"
run_test "MCP Initialize Protocol" "TestMCPProtocolTestSuite/TestInitializeProtocol"
run_test "MCP List Tools" "TestMCPProtocolTestSuite/TestListTools"
run_test "MCP Tool Call with CSRF" "TestMCPProtocolTestSuite/TestCallToolWithCSRF"
run_test "MCP Invalid Requests" "TestMCPProtocolTestSuite/TestInvalidRequest"
run_test "MCP Missing Parameters" "TestMCPProtocolTestSuite/TestMissingRequiredParams"
run_test "MCP Concurrent Requests" "TestMCPProtocolTestSuite/TestConcurrentRequests"

# Run MCP Audit tests
echo -e "${YELLOW}=== MCP Audit Tests ===${NC}"
run_test "MCP Request Validation" "TestMCPRequestValidation"
run_test "MCP Tool Schema Validation" "TestMCPToolSchemaValidation"
run_test "MCP Response Consistency" "TestMCPResponseConsistency"
run_test "MCP Notification Handling" "TestMCPNotificationHandling"
run_test "MCP Error Propagation" "TestMCPErrorPropagation"

# Run integration tests if environment is set
if [ "$ODATA_URL" != "http://localhost:8080/mock" ]; then
    echo -e "${YELLOW}=== Integration Tests ===${NC}"
    run_test "CSRF Integration Test" "TestCSRFIntegration"
    run_test "MCP Full Audit" "TestMCPFullAudit"
fi

# Run all tests together for coverage
echo -e "${YELLOW}=== Running All Tests with Coverage ===${NC}"
go test -v -cover -coverprofile=coverage.out ./internal/test/...
echo ""

# Generate coverage report
echo -e "${BLUE}Generating coverage report...${NC}"
go tool cover -html=coverage.out -o coverage.html
echo -e "${GREEN}✓ Coverage report generated: coverage.html${NC}"

# Run benchmarks
echo -e "${YELLOW}=== Running Benchmarks ===${NC}"
go test -bench=. -benchmem ./internal/test/... | grep -E "Benchmark|ns/op"

echo ""
echo -e "${GREEN}=== Test Suite Complete ===${NC}"

# Summary
echo -e "${BLUE}Test Summary:${NC}"
echo "- CSRF token tracking for all modifying operations (CREATE, UPDATE, DELETE)"
echo "- MCP protocol compliance and consistency"
echo "- Error handling and edge cases"
echo "- Performance benchmarks"
echo ""
echo -e "${BLUE}Key Features Tested:${NC}"
echo "- Automatic CSRF token fetching"
echo "- Token reuse across requests"
echo "- Retry mechanism on 403 responses"
echo "- Header case variation support"
echo "- MCP protocol validation"
echo "- Concurrent request handling"