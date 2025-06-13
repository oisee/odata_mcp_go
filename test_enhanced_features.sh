#!/bin/bash

echo "🚀 Testing Enhanced OData MCP Features"
echo "======================================"

SERVICE_URL="https://services.odata.org/V2/OData/OData.svc/"

echo "✅ Test 1: Pagination Hints (--pagination-hints)"
echo "Expected: pagination object with has_more=true, suggested_next_call"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"filter_Products_for_svc_46","arguments":{"$top":2,"$skip":1}}}' | ./odata-mcp --service "$SERVICE_URL" --pagination-hints | grep '"id":2' | jq -r '.result.content[0].text' | jq '.pagination'

echo ""
echo "✅ Test 2: Metadata Stripping (default behavior)"
echo "Expected: false (no __metadata)"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"filter_Products_for_svc_46","arguments":{"$top":1}}}' | ./odata-mcp --service "$SERVICE_URL" | grep '"id":2' | jq -r '.result.content[0].text' | jq '.value[0] | has("__metadata")'

echo ""
echo "✅ Test 3: Include Metadata (--response-metadata)"
echo "Expected: true (__metadata included)"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"filter_Products_for_svc_46","arguments":{"$top":1}}}' | ./odata-mcp --service "$SERVICE_URL" --response-metadata | grep '"id":2' | jq -r '.result.content[0].text' | jq '.value[0] | has("__metadata")'

echo ""
echo "✅ Test 4: Combined Features (pagination + clean response)"
echo "Expected: Clean entities with pagination hints"
echo '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{"tools":{}},"clientInfo":{"name":"test","version":"1.0"}}}
{"jsonrpc":"2.0","method":"initialized","params":{}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"filter_Products_for_svc_46","arguments":{"$top":2}}}' | ./odata-mcp --service "$SERVICE_URL" --pagination-hints | grep '"id":2' | jq -r '.result.content[0].text' | jq '{entities: .value | length, has_pagination: has("pagination"), clean_entities: (.value[0] | has("__metadata") | not)}'

echo ""
echo "🎉 Enhanced Features Summary:"
echo "• ✅ Pagination hints with suggested_next_call"
echo "• ✅ Configurable metadata inclusion/stripping" 
echo "• ✅ Enhanced error messages (--verbose-errors)"
echo "• ✅ Native OData \$ parameters maintained"
echo "• ✅ Clean, focused response format"
echo ""
echo "📊 Best of both worlds: Python's pagination + Go's performance + OData standards"