# OData MCP Bridge (Go)

A Go implementation of the OData v2 to Model Context Protocol (MCP) bridge, providing universal access to OData services through MCP tools.

This is a Go port of the Python OData-MCP bridge implementation, designed to be easier to run on different operating systems with better performance and simpler deployment.

## Features

- **Universal OData v2 Support**: Works with any OData v2 service
- **Dynamic Tool Generation**: Automatically creates MCP tools based on OData metadata
- **Multiple Authentication Methods**: Basic auth, cookie auth, and anonymous access
- **SAP OData Extensions**: Full support for SAP-specific OData features including CSRF tokens
- **Comprehensive CRUD Operations**: Generated tools for create, read, update, delete operations
- **Advanced Query Support**: OData query options ($filter, $select, $expand, $orderby, etc.)
- **Function Import Support**: Call OData function imports as MCP tools
- **Flexible Tool Naming**: Configurable tool naming with prefix/postfix options
- **Entity Filtering**: Selective tool generation with wildcard support
- **Cross-Platform**: Native Go binary for easy deployment on any OS

## Installation

### Download Binary

Download the appropriate binary for your platform from the releases page.

### Build from Source

#### Quick Build (Go required)
```bash
git clone <repository-url>
cd odata_mcp_go
go build -o odata-mcp cmd/odata-mcp/main.go
```

#### Using Makefile (Recommended)
```bash
# Build for current platform
make build

# Build for all platforms
make build-all

# Build and test
make dev

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

## Usage

### Claude Desktop config:
```json
{
    "mcpServers": {
        "northwind-go": {
            "args": [
                "--service",
                "https://services.odata.org/V2/Northwind/Northwind.svc/",
                "--tool-shrink"
            ],
            "command": "C:/bin/odata-mcp.exe"
        }
    }
}
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

### Debugging and Inspection

```bash
# Enable verbose output
./odata-mcp --verbose https://my-service.com/odata/

# Trace mode - show all tools without starting server
./odata-mcp --trace https://my-service.com/odata/
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

### Northwind Service

```bash
# Connect to the public Northwind OData service
./odata-mcp --trace https://services.odata.org/V2/Northwind/Northwind.svc/

# This will show generated tools like:
# - filter_Products_for_northwind
# - get_Products_for_northwind  
# - filter_Categories_for_northwind
# - get_Orders_for_northwind
# - etc.
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

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the same terms as the original Python implementation.