#!/bin/bash

echo "Testing all OData operations..."
echo "================================"

# Test URL
SERVICE_URL="https://services.odata.org/V2/OData/OData.svc/"

# Test common MCP messages
echo "Test 1: Initialize and get tools list"
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/list",
  "params": {}
}' | ./odata-mcp --service "$SERVICE_URL" | grep -E '"result":|"name":|"description":' | head -20

echo ""
echo "Test 2: Filter Products with \$top=2"
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "filter_Products_for_svc_46",
    "arguments": {"$top": 2}
  }
}' | ./odata-mcp --service "$SERVICE_URL" | grep -A5 -B5 '"result":'

echo ""
echo "Test 3: Get single Product by ID"
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "get_Products_for_svc_46",
    "arguments": {"ID": 0}
  }
}' | ./odata-mcp --service "$SERVICE_URL" | grep -A3 -B3 '"Name":'

echo ""
echo "Test 4: Count Products"
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "count_Products_for_svc_46",
    "arguments": {}
  }
}' | ./odata-mcp --service "$SERVICE_URL" | grep -A2 -B2 '"count":'

echo ""
echo "Test 5: Service Info"
echo '{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "initialize",
  "params": {
    "protocolVersion": "2024-11-05",
    "capabilities": {"tools": {}},
    "clientInfo": {"name": "test", "version": "1.0"}
  }
}
{
  "jsonrpc": "2.0",
  "method": "initialized",
  "params": {}
}
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "tools/call",
  "params": {
    "name": "odata_service_info_for_svc_46",
    "arguments": {}
  }
}' | ./odata-mcp --service "$SERVICE_URL" | grep -A5 -B5 '"entity_sets":'

echo ""
echo "All tests completed!"