# OData MCP Bridge (Go)

A Go implementation of the OData to Model Context Protocol (MCP) bridge, providing universal access to OData services through MCP tools.

This is a Go port of the Python OData-MCP bridge implementation, designed to be easier to run on different operating systems with better performance and simpler deployment. It supports both OData v2 and v4 services.

## 🆕 What's New (v1.6.0)

### Universal Tool Mode — One Tool to Rule Them All

The biggest addition is **Universal Tool Mode** (`--universal`), a game-changer for large OData services:

```bash
# Before: 485 tools, ~37,000 tokens, Claude says "no API available"
./odata-mcp https://large-sap-service.com/odata/

# After: 1 tool, ~900 tokens, works perfectly
./odata-mcp --universal https://large-sap-service.com/odata/
```

| Metric | Standard Mode | Universal Mode | Reduction |
|--------|---------------|----------------|-----------|
| Tools (Northwind) | 157 | 1 | 99.4% |
| Tools (SAP BP) | 485 | 1 | 99.8% |
| Token usage | ~37,000 | ~900 | 97.6% |

**Why is it opt-in?** Universal mode changes how you interact with OData:
- Standard mode: Each entity gets dedicated tools (`filter_Products`, `get_Orders`, etc.)
- Universal mode: One `odata` tool with `action`, `target`, and `params`

We made it opt-in because:
1. **Backward compatibility** — existing configs and workflows continue working
2. **Discoverability** — per-entity tools are self-documenting; LLMs can see exactly what's available
3. **Simplicity for small services** — if you have 20 tools, per-entity mode works great
4. **Explicit choice** — users should consciously choose the trade-off

**When to use `--universal`:**
- Service has 50+ entity sets
- Running multiple OData services simultaneously
- Experiencing "no API available" or tool selection failures
- Want minimal token footprint

See [Universal Tool Architecture](docs/008-issue-14-universal-tool-architecture.md) for the full story.

### MCP Header Forwarding

New `--forward-mcp-headers` flag enables passing HTTP headers from MCP clients to OData services:

```bash
./odata-mcp --transport streamable-http --forward-mcp-headers https://secured-service.com/odata/
```

This enables:
- **Dynamic authentication** — pass credentials per-request instead of at startup
- **Multi-tenant scenarios** — different users with different tokens
- **Custom headers** — `X-*` headers flow through to OData

### Issue Fixes Bonanza

This release fixes 10 open issues:

| Issue | Problem | Fix |
|-------|---------|-----|
| #12 | SAP OData shows no tools | Fixed XML namespace parsing (`sap:creatable` etc.) |
| #13 | `--max-items 99999` crashes | Added validation (max 10,000) |
| #14 | Multiple services = Claude stuck | Universal tool mode |
| #16 | GUID formatting wrong | Auto-detect SAP, add `guid'...'` prefix |
| #17 | Timeout instead of error | Immediate error response |
| #18 | Wildcard search fails | Parse `SearchRestrictions` annotation |
| #19 | Timeout hides SAP error | Return actual error message |
| #22 | BaseType not exposed | Added to EntityType model |
| #23 | Header handling | `--forward-mcp-headers` flag |
| #25 | Windows build no .exe | Fixed Makefile for Windows |

### Previous Releases

- **AI Foundry Compatibility** (v1.5.1): `--protocol-version` flag for AI Foundry's `2025-06-18` protocol
- **SAP GUID Filtering** (v1.5.0): Automatic `guid'...'` formatting for SAP services
- **Streamable HTTP Transport** (v1.5.0): Modern MCP protocol with `--transport streamable-http`

## Features

- **Universal OData Support**: Works with both OData v2 and v4 services
- **Dynamic Tool Generation**: Automatically creates MCP tools based on OData metadata
- **Multiple Authentication Methods**: Basic auth, cookie auth, and anonymous access
- **SAP OData Extensions**: Full support for SAP-specific OData features including CSRF tokens
- **Comprehensive CRUD Operations**: Generated tools for create, read, update, delete operations
- **Advanced Query Support**: OData query options ($filter, $select, $expand, $orderby, etc.)
- **Function Import Support**: Call OData function imports as MCP tools
- **Flexible Tool Naming**: Configurable tool naming with prefix/postfix options
- **Entity Filtering**: Selective tool generation with wildcard support
- **Cross-Platform**: Native Go binary for easy deployment on any OS
- **Read-Only Modes**: Restrict operations with `--read-only` or `--read-only-but-functions`
- **MCP Protocol Debugging**: Built-in trace logging with `--trace-mcp` for troubleshooting
- **Service-Specific Hints**: Flexible hint system with pattern matching for known service issues
- **Full MCP Compliance**: Complete protocol implementation for all MCP clients
- **Multiple Transports**: Support for stdio (default), HTTP/SSE, and Streamable HTTP
- **AI Foundry Compatible**: Configurable protocol version for AI Foundry and other MCP clients

## Installation

### Download Binary

Download the appropriate binary for your platform from the [releases page](https://github.com/oisee/odata_mcp_go/releases).

Pre-built binaries are available for:
- Linux (amd64)
- Windows (amd64)
- macOS (Intel and Apple Silicon)

### Build from Source

#### Quick Build (Go required)
```bash
git clone https://github.com/oisee/odata_mcp_go.git
cd odata_mcp_go
go build -o odata-mcp cmd/odata-mcp/main.go
```

#### Using Makefile (Recommended)
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build for all platforms with WSL integration (copies to /mnt/c/bin)
make build-all-wsl

# Build and test
make dev

# Check current version
make version

# See all options
make help
```

#### Using Build Script
```bash
# Build for current platform
./build.sh

# Build for all platforms
./build.sh all

# See all options
./build.sh help
```

#### Cross-Compilation Examples
```bash
# Using Make
make build-linux     # Linux (amd64)
make build-windows   # Windows (amd64)
make build-macos     # macOS (Intel + Apple Silicon)

# WSL-specific builds (copies Windows binary to /mnt/c/bin)
make build-windows-wsl  # Build Windows + WSL integration
make build-all-wsl      # Build all platforms + WSL integration

# Using build script
./build.sh linux     # Linux (amd64)
./build.sh windows   # Windows (amd64)
./build.sh macos     # macOS (Intel + Apple Silicon)

# Manual Go build
GOOS=linux GOARCH=amd64 go build -o odata-mcp-linux cmd/odata-mcp/main.go
GOOS=windows GOARCH=amd64 go build -o odata-mcp.exe cmd/odata-mcp/main.go
```

#### Docker Build
```bash
# Build Docker image
make docker

# Or manually
docker build -t odata-mcp .

# Run in container
docker run --rm -it odata-mcp --help
```

#### Building in WSL (Windows Subsystem for Linux)

When building in WSL, you can use special targets that automatically copy the Windows binary to your Windows file system:

```bash
# Build all platforms and copy Windows binary to C:\bin
make build-all-wsl

# Build only Windows and copy to C:\bin
make build-windows-wsl
```

Note: These commands will check if `/mnt/c/bin` exists and skip the copy if not found, so they're safe to use on any system.

## Usage

### Claude Desktop Configuration

Claude Desktop uses the stdio transport by default. Here are example configurations:

#### Finding Your Configuration File

The Claude Desktop configuration file location varies by platform:

- **Windows**: `%APPDATA%\Claude\claude_desktop_config.json`
- **macOS**: `~/Library/Application Support/Claude/claude_desktop_config.json`
- **Linux**: `~/.config/Claude/claude_desktop_config.json`

#### Basic Configuration

```json
{
    "mcpServers": {
        "northwind-v2": {
            "command": "C:/bin/odata-mcp.exe",
            "args": [
                "--service",
                "https://services.odata.org/V2/Northwind/Northwind.svc/",
                "--tool-shrink"
            ]
        },
        "northwind-v4": {
            "command": "C:/bin/odata-mcp.exe",
            "args": [
                "--service",
                "https://services.odata.org/V4/Northwind/Northwind.svc/",
                "--tool-shrink"
            ]
        }
    }
}
```

#### With Authentication

```json
{
    "mcpServers": {
        "my-sap-service": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://my-sap-system.com/sap/opu/odata/sap/MY_SERVICE/",
                "--user",
                "myusername",
                "--password",
                "mypassword",
                "--tool-shrink",
                "--entities",
                "Products,Orders,Customers"
            ]
        }
    }
}
```

#### Using Environment Variables (More Secure)

```json
{
    "mcpServers": {
        "my-secure-service": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://my-service.com/odata/",
                "--tool-shrink"
            ],
            "env": {
                "ODATA_USERNAME": "myusername",
                "ODATA_PASSWORD": "mypassword"
            }
        }
    }
}
```

**Note:** Claude Desktop currently doesn't support reading environment variables from your system. The `env` field in the configuration sets environment variables specifically for that MCP server process.

#### Security Best Practices for Claude Desktop

1. **Use environment variables** in the `env` field rather than hardcoding credentials in `args`
2. **Limit entity access** using the `--entities` flag to only expose necessary data
3. **Use read-only accounts** when possible for OData services
4. **Store configuration file securely** with appropriate file permissions

**Note:** Claude Desktop does not currently support API key authentication for MCP servers. All MCP servers run locally with the same permissions as Claude Desktop itself.

#### Read-Only Configuration Examples

```json
{
    "mcpServers": {
        "production-readonly": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://production.company.com/odata/",
                "--read-only",
                "--tool-shrink"
            ],
            "env": {
                "ODATA_USERNAME": "readonly_user",
                "ODATA_PASSWORD": "readonly_pass"
            }
        },
        "dev-with-functions": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service", 
                "https://dev.company.com/odata/",
                "--read-only-but-functions",
                "--trace-mcp"  // Enable debugging
            ]
        }
    }
}
```

#### Claude Code CLI Compatibility

The Claude Code CLI has stricter property name validation than other Claude tools. If you encounter errors like:

```
API Error: 400 {"type":"error","error":{"type":"invalid_request_error","message":"tools.17.custom.input_schema.properties: Property keys should match pattern ^[a-zA-Z0-9_.-]{1,64}$"}}
```

Use the `--claude-code-friendly` flag to remove the `$` prefix from OData parameter names:

```json
{
    "mcpServers": {
        "northwind-claude-code": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://services.odata.org/V2/Northwind/Northwind.svc/",
                "--claude-code-friendly",  // Removes $ from parameter names
                "--tool-shrink"
            ]
        }
    }
}
```

This transforms parameter names:
- `$filter` → `filter`
- `$select` → `select`
- `$expand` → `expand`
- `$orderby` → `orderby`
- `$top` → `top`
- `$skip` → `skip`
- `$count` → `count`

The server internally maps these friendly names back to their OData equivalents when making requests.

#### Operation Filtering Configuration Examples

```json
{
    "mcpServers": {
        "large-service-readonly": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://large-erp.company.com/odata/",
                "--disable", "cud",  // Disable create, update, delete
                "--tool-shrink",
                "--entities", "Orders,Products,Customers"
            ],
            "env": {
                "ODATA_USERNAME": "readonly_user",
                "ODATA_PASSWORD": "readonly_pass"
            }
        },
        "minimal-tools": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://api.company.com/odata/",
                "--enable", "gf",  // Only get and filter operations
                "--tool-shrink"
            ]
        },
        "no-actions": {
            "command": "/usr/local/bin/odata-mcp", 
            "args": [
                "--service",
                "https://api.company.com/odata/",
                "--disable", "a",  // Disable all function imports/actions
                "--verbose"
            ]
        }
    }
}

### AI Foundry Configuration

For AI Foundry integration, use the `--protocol-version` flag to specify the `2025-06-18` protocol:

```json
{
    "mcpServers": {
        "odata-for-ai-foundry": {
            "command": "/usr/local/bin/odata-mcp",
            "args": [
                "--service",
                "https://your-odata-service.com/",
                "--protocol-version", "2025-06-18",
                "--user", "your-username",
                "--password", "your-password"
            ]
        }
    }
}
```

Or via command line:
```bash
./odata-mcp --service https://your-service.com/odata --protocol-version "2025-06-18"
```

See the [AI Foundry Compatibility Guide](AI_FOUNDRY_COMPATIBILITY.md) for detailed setup instructions.

### Transport Options

The OData MCP bridge supports two transport mechanisms:

1. **STDIO (default)** - Standard input/output communication, used by Claude Desktop
2. **HTTP/SSE** - HTTP server with Server-Sent Events for web-based clients

> 🔒 **SECURITY MODEL**: HTTP transport uses a strict security model.
>
> **Security Requirements:**
> - **Localhost**: Token required (`--mcp-token`)
> - **Non-localhost**: Token + TLS required, no exceptions
> - **All interfaces (0.0.0.0/::)**: Requires `--allow-all-interfaces` + token + TLS
>
> Token can be any string - for dev, `--mcp-token dev` works fine.

#### Using Streamable HTTP Transport (Modern MCP Protocol)

**New in v1.5.0**: Support for Streamable HTTP transport (protocol version 2024-11-05)

```bash
# Start server with Streamable HTTP (recommended for modern clients)
./odata-mcp --transport streamable-http https://services.odata.org/V2/Northwind/Northwind.svc/

# Use custom localhost port
./odata-mcp --transport streamable-http --http-addr localhost:3000 https://services.odata.org/V2/Northwind/Northwind.svc/
```

Streamable HTTP endpoints:
- `POST /mcp` - Main MCP endpoint (supports automatic SSE upgrade)
- `GET /health` - Health check endpoint
- `POST /sse` - Legacy SSE endpoint (for backward compatibility)

#### Using HTTP/SSE Transport (Legacy)

```bash
# Start server on localhost (default: localhost:8080)
./odata-mcp --transport http --mcp-token "dev" https://services.odata.org/V2/Northwind/Northwind.svc/

# Use custom localhost port
./odata-mcp --transport http --http-addr localhost:3000 --mcp-token "dev" https://services.odata.org/V2/Northwind/Northwind.svc/

# Non-localhost requires token + TLS
./odata-mcp --transport http --http-addr 192.168.1.100:8080 \
  --mcp-token "my-secret-token" --tls --tls-cert cert.pem --tls-key key.pem \
  https://services.odata.org/V2/Northwind/Northwind.svc/

# All interfaces requires explicit flag + token + TLS
./odata-mcp --transport http --http-addr 0.0.0.0:8080 \
  --allow-all-interfaces --mcp-token "my-secret-token" \
  --tls --tls-cert cert.pem --tls-key key.pem \
  https://services.odata.org/V2/Northwind/Northwind.svc/
```

Legacy HTTP/SSE endpoints:
- `GET /health` - Health check endpoint
- `GET /sse` - Server-Sent Events endpoint for real-time communication
- `POST /rpc` - JSON-RPC endpoint for request/response communication

#### Testing HTTP/SSE Transport

1. **Using the provided HTML client:**
   ```bash
   # Start the server
   ./odata-mcp --transport http https://services.odata.org/V2/Northwind/Northwind.svc/
   
   # Open examples/sse_client.html in a web browser
   ```

2. **Using curl:**
   ```bash
   # Test SSE endpoint
   curl -N -H 'Accept: text/event-stream' http://localhost:8080/sse
   
   # Test RPC endpoint
   curl -X POST http://localhost:8080/rpc \
     -H "Content-Type: application/json" \
     -d '{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}'
   ```

3. **Using the test scripts:**
   ```bash
   # Test SSE with interactive script
   ./test_sse.sh
   
   # Test HTTP/RPC communication
   ./test_http_rpc.sh
   ```


### Basic Usage

```bash
# Using positional argument
./odata-mcp https://services.odata.org/V2/Northwind/Northwind.svc/

# Using --service flag
./odata-mcp --service https://services.odata.org/V2/Northwind/Northwind.svc/

# Using environment variable
export ODATA_SERVICE_URL=https://services.odata.org/V2/Northwind/Northwind.svc/
./odata-mcp
```

### Authentication

```bash
# Basic authentication
./odata-mcp --user admin --password secret https://my-service.com/odata/

# Cookie file authentication
./odata-mcp --cookie-file cookies.txt https://my-service.com/odata/

# Cookie string authentication  
./odata-mcp --cookie-string "session=abc123; token=xyz789" https://my-service.com/odata/

# Environment variables
export ODATA_USERNAME=admin
export ODATA_PASSWORD=secret
./odata-mcp https://my-service.com/odata/
```

### Tool Naming Options

```bash
# Use custom prefix instead of postfix
./odata-mcp --no-postfix --tool-prefix "myservice" https://my-service.com/odata/

# Use custom postfix
./odata-mcp --tool-postfix "northwind" https://my-service.com/odata/

# Use shortened tool names
./odata-mcp --tool-shrink https://my-service.com/odata/
```

### Entity and Function Filtering

```bash
# Filter to specific entities (supports wildcards)
./odata-mcp --entities "Products,Categories,Order*" https://my-service.com/odata/

# Filter to specific functions (supports wildcards)  
./odata-mcp --functions "Get*,Create*" https://my-service.com/odata/
```

### Read-Only Modes

```bash
# Hide all modifying operations (create, update, delete, and functions)
./odata-mcp --read-only https://my-service.com/odata/
./odata-mcp -ro https://my-service.com/odata/  # Short form

# Hide create/update/delete but allow function imports
./odata-mcp --read-only-but-functions https://my-service.com/odata/
./odata-mcp -robf https://my-service.com/odata/  # Short form
```

### Operation Type Filtering

Fine-grained control over which operation types are available. Operation types are:
- `C` - Create operations
- `S` - Search operations
- `F` - Filter/list operations
- `G` - Get (single entity) operations
- `U` - Update operations
- `D` - Delete operations
- `A` - Actions/function imports
- `R` - Read operations (expands to S, F, G)

```bash
# Enable only read operations (search, filter, get)
./odata-mcp --enable "r" https://my-service.com/odata/
./odata-mcp --enable "sfg" https://my-service.com/odata/  # Same as above

# Disable all modifying operations
./odata-mcp --disable "cud" https://my-service.com/odata/

# Enable only get and filter operations
./odata-mcp --enable "gf" https://my-service.com/odata/

# Disable actions/function imports
./odata-mcp --disable "a" https://my-service.com/odata/

# Case-insensitive
./odata-mcp --disable "CUD" https://my-service.com/odata/
```

Note: `--enable` and `--disable` cannot be used together.

### Universal Tool Mode

For large OData services with many entities, the standard per-entity tool generation can create hundreds of tools, causing:
- **Context rot**: LLMs struggle to reason when tool count exceeds ~128
- **High token usage**: Tool schemas can consume 15,000-40,000 tokens
- **Tool selection failures**: LLMs may report "no API available"

Universal mode solves this by generating a single tool that handles all operations:

```bash
# Enable universal tool mode
./odata-mcp --universal https://my-service.com/odata/

# Compare tool counts
./odata-mcp --trace https://my-service.com/odata/           # Standard: many tools
./odata-mcp --universal --trace https://my-service.com/odata/  # Universal: 1 tool
```

**When to use universal mode:**
- Service has more than ~50 entity sets
- Using multiple OData services simultaneously
- Experiencing "no API available" errors with large services

**Universal tool usage:**
```json
{"action": "list", "target": "Products", "params": {"filter": "Price gt 100", "top": 10}}
{"action": "get", "target": "Products", "params": {"key": {"ProductID": 1}}}
{"action": "create", "target": "Orders", "params": {"data": {"CustomerID": "C001"}}}
{"action": "call", "target": "ReleaseOrder", "params": {"OrderID": "O001"}}
```

### Debugging and Inspection

```bash
# Enable verbose output
./odata-mcp --verbose https://my-service.com/odata/

# Trace mode - show all tools without starting server
./odata-mcp --trace https://my-service.com/odata/

# Enable MCP protocol trace logging (saves to temp directory)
./odata-mcp --trace-mcp https://my-service.com/odata/
# Linux/WSL: /tmp/mcp_trace_*.log
# Windows: %TEMP%\mcp_trace_*.log
```

### Service Hints

The OData MCP bridge includes a flexible hint system to provide guidance for services with known issues or special requirements:

```bash
# Use default hints.json from binary directory
./odata-mcp https://my-service.com/odata/

# Use custom hints file
./odata-mcp --hints-file /path/to/custom-hints.json https://my-service.com/odata/

# Inject hint directly from command line
./odata-mcp --hint "Remember to use \$expand for complex queries" https://my-service.com/odata/

# Combine file and CLI hints (CLI has higher priority)
./odata-mcp --hints-file custom.json --hint '{"notes":["Override note"]}' https://my-service.com/odata/
```

## Configuration

### Command Line Flags

| Flag | Description | Default |
|------|-------------|---------|
| `--service` | OData service URL | |
| `-u, --user` | Username for basic auth | |
| `-p, --password` | Password for basic auth | |
| `--cookie-file` | Path to cookie file (Netscape format) | |
| `--cookie-string` | Cookie string (key1=val1; key2=val2) | |
| `--tool-prefix` | Custom prefix for tool names | |
| `--tool-postfix` | Custom postfix for tool names | |
| `--no-postfix` | Use prefix instead of postfix | `false` |
| `--tool-shrink` | Use shortened tool names | `false` |
| `--entities` | Comma-separated entity filter (supports wildcards) | |
| `--functions` | Comma-separated function filter (supports wildcards) | |
| `--sort-tools` | Sort tools alphabetically | `true` |
| `-v, --verbose` | Enable verbose output | `false` |
| `--debug` | Alias for --verbose | `false` |
| `--trace` | Show tools and exit (debug mode) | `false` |
| `--trace-mcp` | Enable MCP protocol trace logging | `false` |
| `--read-only, -ro` | Hide all modifying operations | `false` |
| `--read-only-but-functions, -robf` | Hide create/update/delete but allow functions | `false` |
| `--enable` | Enable only specified operation types (C,S,F,G,U,D,A,R) | |
| `--disable` | Disable specified operation types (C,S,F,G,U,D,A,R) | |
| `--hints-file` | Path to hints JSON file | `hints.json` in binary dir |
| `--hint` | Direct hint JSON or text from CLI | |
| `--transport` | Transport type: 'stdio', 'http' (SSE), or 'streamable-http' | `stdio` |
| `--http-addr` | HTTP server address (with --transport http/streamable-http) | `localhost:8080` |
| `--mcp-token` | Authentication token for HTTP transport (required) | |
| `--mcp-token-file` | Path to file containing authentication token | |
| `--tls` | Enable TLS for HTTP transport | `false` |
| `--tls-cert` | Path to TLS certificate file | |
| `--tls-key` | Path to TLS key file | |
| `--allow-all-interfaces` | Allow binding to 0.0.0.0/:: (requires --mcp-token and --tls) | `false` |
| `--legacy-dates` | Enable legacy date format conversion | `true` |
| `--no-legacy-dates` | Disable legacy date format conversion | `false` |
| `--convert-dates-from-sap` | Convert SAP date formats in responses | `false` |
| `--response-metadata` | Include __metadata blocks in responses | `false` |
| `--pagination-hints` | Add pagination information to responses | `false` |
| `--max-response-size` | Maximum response size in bytes | `5MB` |
| `--max-items` | Maximum number of items in response | `100` |
| `--verbose-errors` | Provide detailed error context | `false` |
| `--claude-code-friendly, -c` | Remove $ prefix from OData parameters for Claude Code CLI compatibility | `false` |
| `--protocol-version` | Override MCP protocol version (e.g., '2025-06-18' for AI Foundry) | `2024-11-05` |
| `--forward-mcp-headers` | Forward HTTP headers from MCP connection to OData service (Streamable HTTP only) | `false` |
| `--universal` | Use single universal OData tool instead of per-entity tools (reduces context for large services) | `false` |

### Environment Variables

| Variable | Description |
|----------|-------------|
| `ODATA_SERVICE_URL` or `ODATA_URL` | OData service URL |
| `ODATA_USERNAME` or `ODATA_USER` | Username for basic auth |
| `ODATA_PASSWORD` or `ODATA_PASS` | Password for basic auth |
| `ODATA_COOKIE_FILE` | Path to cookie file |
| `ODATA_COOKIE_STRING` | Cookie string |

### .env File Support

Create a `.env` file in the working directory:

```env
ODATA_SERVICE_URL=https://my-service.com/odata/
ODATA_USERNAME=admin
ODATA_PASSWORD=secret
```

## Generated Tools

The bridge automatically generates MCP tools based on the OData service metadata:

### Entity Set Tools

For each entity set, the following tools are generated (if the entity set supports the operation):

- `filter_{EntitySet}` - List/filter entities with OData query options
- `count_{EntitySet}` - Get count of entities with optional filter
- `search_{EntitySet}` - Full-text search (if supported by the service)
- `get_{EntitySet}` - Get a single entity by key
- `create_{EntitySet}` - Create a new entity (if allowed)
- `update_{EntitySet}` - Update an existing entity (if allowed)  
- `delete_{EntitySet}` - Delete an entity (if allowed)

### Function Import Tools

Each function import is mapped to an individual tool with the function name.

### Service Information Tool

- `odata_service_info` - Get metadata and capabilities of the OData service

## Examples

### Northwind Service (v2)

```bash
# Connect to the public Northwind OData v2 service
./odata-mcp --trace https://services.odata.org/V2/Northwind/Northwind.svc/

# This will show generated tools like:
# - filter_Products_for_northwind
# - get_Products_for_northwind  
# - filter_Categories_for_northwind
# - get_Orders_for_northwind
# - etc.
```

### Northwind Service (v4)

```bash
# Connect to the public Northwind OData v4 service
./odata-mcp --trace https://services.odata.org/V4/Northwind/Northwind.svc/

# OData v4 is automatically detected and handled appropriately
# Supports v4 specific features like:
# - $count parameter instead of $inlinecount
# - contains() filter function
# - New data types (Edm.Date, Edm.TimeOfDay, etc.)
```

### SAP OData Service

```bash
# Connect to SAP service with CSRF token support
./odata-mcp --user admin --password secret \
  https://my-sap-system.com/sap/opu/odata/sap/SERVICE_NAME/
```

## Differences from Python Version

While maintaining the same CLI interface and functionality, this Go implementation offers:

- **Better Performance**: Native compiled binary with lower memory usage
- **Easier Deployment**: Single binary with no runtime dependencies
- **Cross-Platform**: Native binaries for Windows, macOS, and Linux
- **Type Safety**: Go's type system provides better reliability
- **Simpler Installation**: No need for Python runtime or package management

## Versioning

This project uses automatic versioning based on git tags and commit history:

- **Tagged releases**: Uses git tags (e.g., `v1.0.0`)
- **Development builds**: Uses `0.1.<commit-count>` format
- **Uncommitted changes**: Appends `-dirty` suffix

```bash
# Check current version
make version

# Create a release
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

See [VERSIONING.md](VERSIONING.md) for detailed versioning guide.

## Releasing

This project uses automated GitHub Actions for releases. See [RELEASING.md](RELEASING.md) for the release process.

## Troubleshooting

### MCP Client Issues

If you're experiencing issues with MCP clients (Claude Desktop, RooCode, GitHub Copilot):

1. **Enable trace logging** to diagnose protocol issues:
   ```bash
   ./odata-mcp --trace-mcp https://my-service.com/odata/
   ```
   Then check the trace file in your temp directory.

2. **Common issues and solutions**:
   - **Tools not appearing**: Ensure the service URL is correct and accessible
   - **Validation errors**: Update to the latest version which includes MCP compliance fixes
   - **Connection failures**: Check authentication credentials and network connectivity

3. **Service-specific hints**: The `odata_service_info` tool now includes automatic hints for known problematic services

See [TROUBLESHOOTING.md](TROUBLESHOOTING.md) for detailed troubleshooting guide.

### Service Hints System

The OData MCP bridge includes a sophisticated hint system that helps users work around known service issues and provides implementation guidance.

#### Hint File Format

Create a `hints.json` file with the following structure:

```json
{
  "version": "1.0",
  "hints": [
    {
      "pattern": "*/sap/opu/odata/*",
      "priority": 10,
      "service_type": "SAP OData Service",
      "known_issues": ["List of known issues"],
      "workarounds": ["List of workarounds"],
      "field_hints": {
        "FieldName": {
          "type": "Edm.String",
          "format": "Expected format",
          "example": "12345",
          "description": "Field description"
        }
      },
      "examples": [
        {
          "description": "Example description",
          "query": "filter_EntitySet with $filter=...",
          "note": "Additional note"
        }
      ]
    }
  ]
}
```

#### Pattern Matching

The hint system supports wildcard patterns:
- `*` matches any sequence of characters
- `?` matches a single character
- Multiple patterns can match the same service (hints are merged by priority)

#### Default Hints

The bridge includes default hints for common services:

- **SAP OData Services** (`*/sap/opu/odata/*`): General SAP OData guidance including the critical workaround for HTTP 501 errors using `$expand`
- **SAP PO Tracking** (`*SRA020_PO_TRACKING_SRV*`): Specific hints for purchase order tracking including field formatting
- **Northwind Demo** (`*Northwind*`): Identifies the public demo service

#### Using Hints

Hints appear in the `odata_service_info` tool response under `implementation_hints`:

```bash
# View hints for your service
./odata-mcp https://my-service.com/odata/
# Then call the odata_service_info tool in your MCP client

# The response includes:
{
  "implementation_hints": {
    "service_type": "SAP OData Service",
    "known_issues": [...],
    "workarounds": [...],
    "field_hints": {...},
    "examples": [...],
    "hint_source": "Hints file: hints.json"
  }
}
```

## Security

This project includes comprehensive security measures to prevent credential leaks. See [SECURITY.md](SECURITY.md) for details.

**Important**: Never commit `.zmcp.json` or any files containing real credentials.

## Documentation

- [QUICK_REFERENCE.md](QUICK_REFERENCE.md) - Quick command reference
- [HINTS.md](HINTS.md) - Complete guide to the service hints system
- [TROUBLESHOOTING.md](TROUBLESHOOTING.md) - Common issues and solutions
- [SECURITY.md](SECURITY.md) - Security considerations
- [CHANGELOG.md](CHANGELOG.md) - Version history and changes

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

For questions and community discussion, visit our [GitHub Discussions](https://github.com/oisee/odata_mcp_go/discussions/11).

### Development

For development setup and testing:

```bash
# Run tests
make test

# Run with verbose output for debugging
./odata-mcp --verbose --trace-mcp https://my-service.com/odata/

# Check MCP compliance
./simple_compliance_test.sh
```

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.