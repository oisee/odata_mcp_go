# MCP Protocol Handling Comparison: Go vs Python

## Original Issue Analysis

### Go Server Response (Before Fix)
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "capabilities": {
      "tools": {
        "listChanged": true
      }
    },
    "protocolVersion": "2024-11-05",
    "serverInfo": {
      "name": "odata-mcp-bridge",
      "version": "1.0.0"
    }
  }
}
```

### Python Server Response (Reference)
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "experimental": {},
      "prompts": {
        "listChanged": false
      },
      "resources": {
        "subscribe": false,
        "listChanged": false
      },
      "tools": {
        "listChanged": true
      }
    },
    "serverInfo": {
      "name": "odata-mcp",
      "version": "1.9.2"
    }
  }
}
```

## Key Differences Found

### 1. Protocol Version
- **Go (before)**: `"2024-11-05"` (older version)
- **Python**: `"2025-03-26"` (latest version)

### 2. Capabilities Structure
**Go (before)** - minimal:
- Only includes `tools` capability
- Single `listChanged: true` property

**Python** - comprehensive:
- Includes `experimental`, `prompts`, `resources`, and `tools` capabilities  
- Each capability has detailed properties like `listChanged` and `subscribe`

### 3. Field Order
**Go (before)**: `capabilities`, `protocolVersion`, `serverInfo`
**Python**: `protocolVersion`, `capabilities`, `serverInfo`

## Root Cause Analysis

The Go server was likely not being recognized by Claude Code because:

1. **Outdated Protocol Version**: Using `2024-11-05` instead of the latest `2025-03-26`
2. **Missing Standard Capabilities**: Missing `experimental`, `prompts`, and `resources` declarations
3. **Incomplete Capability Declarations**: Only declaring tools, making the server appear limited

## 'Initialized' Notification Handling

Both implementations handle this correctly:

### Go Implementation
```go
// handleInitialized handles the initialized notification
func (s *Server) handleInitialized(req *Request) error {
	s.mu.Lock()
	s.initialized = true
	s.mu.Unlock()
	return nil
}
```

### Python Implementation
Uses FastMCP library which handles this through the standard MCP SDK automatically.

## Fixes Applied

### 1. Updated Protocol Version
**File**: `internal/constants/constants.go`
```go
// Changed from:
MCPProtocolVersion = "2024-11-05"
// To:
MCPProtocolVersion = "2025-03-26"
```

### 2. Enhanced Capabilities Declaration
**File**: `internal/mcp/server.go`
```go
// Added comprehensive capabilities matching Python implementation:
"capabilities": map[string]interface{}{
    "experimental": map[string]interface{}{},
    "prompts": map[string]interface{}{
        "listChanged": false,
    },
    "resources": map[string]interface{}{
        "subscribe":   false,
        "listChanged": false,
    },
    "tools": map[string]interface{}{
        "listChanged": true,
    },
},
```

### 3. Fixed Field Order
Reordered response fields to match Python implementation:
1. `protocolVersion`
2. `capabilities` 
3. `serverInfo`

## Go Server Response (After Fix)
```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "result": {
    "protocolVersion": "2025-03-26",
    "capabilities": {
      "experimental": {},
      "prompts": {
        "listChanged": false
      },
      "resources": {
        "subscribe": false,
        "listChanged": false
      },
      "tools": {
        "listChanged": true
      }
    },
    "serverInfo": {
      "name": "odata-mcp-bridge",
      "version": "1.0.0"
    }
  }
}
```

## Verification

✅ **Protocol Version**: Now matches Python (`2025-03-26`)
✅ **Capabilities**: Now includes all standard MCP capabilities  
✅ **Field Order**: Now matches Python implementation
✅ **Tool Discovery**: Go server now reports 157 tools (same as before, confirming no regression)
✅ **MCP Flow**: Complete initialize → initialized → tools/list flow works correctly

## Impact Assessment

These changes should resolve Claude Code's tool recognition issues because:

1. **Modern Protocol Support**: Now using the latest MCP protocol version
2. **Complete Capability Declaration**: Properly declares all standard MCP capabilities
3. **Standard Compliance**: Response structure now matches the reference Python implementation
4. **Backward Compatibility**: `2024-11-05` is still supported by MCP SDK, so no breaking changes