#!/bin/bash

echo "Testing single OData operation..."

# Test filter operation
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"filter_Products_for_svc_46","arguments":{"$top":3}}}' | ./odata-mcp --service https://services.odata.org/V2/OData/OData.svc/ 2>&1 | grep -A2 -B2 "result\|error"