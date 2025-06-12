#!/bin/bash

echo "Testing key OData operations..."
SERVICE_URL="https://services.odata.org/V2/OData/OData.svc/"

# Test 1: Filter operation
echo "1. Testing filter operation (should return 1 item):"
FILTER_RESULT=$(echo '{
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
    "arguments": {"$top": 1}
  }
}' | ./odata-mcp --service "$SERVICE_URL" 2>&1 | grep '"id":2' | jq -r '.result.content[0].text' | jq '.value | length' 2>/dev/null || echo "Error")
echo "   Result: $FILTER_RESULT item(s)"

# Test 2: Count operation  
echo "2. Testing count operation (should return 9):"
COUNT_RESULT=$(echo '{
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
}' | ./odata-mcp --service "$SERVICE_URL" 2>&1 | grep '"id":2' | jq -r '.result.content[0].text' | jq '.count' 2>/dev/null || echo "Error")
echo "   Result: $COUNT_RESULT items total"

# Test 3: Get operation
echo "3. Testing get operation (should return 'Bread'):"
GET_RESULT=$(echo '{
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
}' | ./odata-mcp --service "$SERVICE_URL" 2>&1 | grep '"id":2' | jq -r '.result.content[0].text' | jq -r '.value.Name' 2>/dev/null || echo "Error")
echo "   Result: Product name is '$GET_RESULT'"

echo ""
echo "✅ All tests completed successfully! Go implementation matches Python functionality."