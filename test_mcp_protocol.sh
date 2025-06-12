#!/bin/bash

echo "Testing MCP Protocol Compliance..."

SERVICE_URL="https://services.odata.org/V2/OData/OData.svc/"

echo "1. Testing initialize:"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}' | ./odata-mcp --service "$SERVICE_URL" | head -1

echo ""
echo "2. Testing initialized (notification):"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}' | ./odata-mcp --service "$SERVICE_URL" | head -2 | tail -1

echo ""
echo "3. Testing tools/list:"  
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}' | ./odata-mcp --service "$SERVICE_URL" | grep '"id":2' | jq '.result.tools | length'

echo ""
echo "4. Testing with malformed JSON:"
echo '{"jsonrpc":"2.0","id":3,"method":"invalid"}' | ./odata-mcp --service "$SERVICE_URL" | jq '.error.code'

echo ""
echo "Protocol compliance check complete."