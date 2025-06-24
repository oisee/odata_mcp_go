#!/bin/bash

echo "=== Testing SAP Date Handling ==="
echo ""

# Test with legacy dates enabled (default)
echo "1. Testing with legacy dates ENABLED (default for SAP):"
echo "   Running: ./odata-mcp --trace https://services.odata.org/V2/Northwind/Northwind.svc/"
./odata-mcp --trace https://services.odata.org/V2/Northwind/Northwind.svc/ 2>&1 | grep -A2 -B2 "Legacy date"

echo ""
echo "2. Testing with legacy dates DISABLED:"
echo "   Running: ./odata-mcp --no-legacy-dates --trace https://services.odata.org/V2/Northwind/Northwind.svc/"
./odata-mcp --no-legacy-dates --trace https://services.odata.org/V2/Northwind/Northwind.svc/ 2>&1 | grep -A2 -B2 "Legacy date"

echo ""
echo "3. Testing date conversion in a simple query:"
echo ""

# Create a test script that sends commands via MCP
cat > test_date_conversion.py << 'EOF'
import json
import subprocess
import sys

def send_request(proc, request):
    """Send a request and get response"""
    proc.stdin.write(json.dumps(request) + '\n')
    proc.stdin.flush()
    response = proc.stdout.readline()
    return json.loads(response) if response else None

# Start MCP server with legacy dates
proc = subprocess.Popen(
    ['./odata-mcp', 'https://services.odata.org/V2/Northwind/Northwind.svc/'],
    stdin=subprocess.PIPE,
    stdout=subprocess.PIPE,
    stderr=subprocess.PIPE,
    text=True
)

# Initialize
init_req = {
    "jsonrpc": "2.0",
    "id": 1,
    "method": "initialize",
    "params": {"protocolVersion": "0.1.0", "capabilities": {}}
}
print("Sending initialize...")
resp = send_request(proc, init_req)
print(f"Initialize response: {resp.get('result', {}).get('serverInfo', {})}")

# Send initialized
proc.stdin.write(json.dumps({"jsonrpc": "2.0", "method": "initialized"}) + '\n')
proc.stdin.flush()

# Get an order to see date handling
order_req = {
    "jsonrpc": "2.0",
    "id": 2,
    "method": "tools/call",
    "params": {
        "name": "get_Orders_for_NorthSvc",
        "arguments": {"OrderID": 10248}
    }
}
print("\nFetching Order 10248 to check date fields...")
resp = send_request(proc, order_req)

if resp and 'result' in resp:
    result = resp['result']
    if 'content' in result and len(result['content']) > 0:
        content = result['content'][0].get('text', '{}')
        order_data = json.loads(content)
        
        print("\nDate fields in response:")
        for key, value in order_data.items():
            if isinstance(value, str) and ('Date' in key or 'date' in key):
                print(f"  {key}: {value}")
else:
    print("Error getting order data")

proc.terminate()
EOF

python3 test_date_conversion.py

# Clean up
rm -f test_date_conversion.py