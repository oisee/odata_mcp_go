#!/bin/bash

# Simple build script for OData MCP Bridge
# Alternative to Makefile for those who prefer shell scripts

set -e

BINARY_NAME="odata-mcp"
MAIN_PATH="cmd/odata-mcp/main.go"
VERSION=${VERSION:-"1.0.0"}
BUILD_DIR="build"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

log_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

log_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

log_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Print usage
usage() {
    echo "OData MCP Bridge - Build Script"
    echo "==============================="
    echo ""
    echo "Usage: $0 [command]"
    echo ""
    echo "Commands:"
    echo "  build         Build for current platform (default)"
    echo "  linux         Build for Linux (amd64)"
    echo "  windows       Build for Windows (amd64)"
    echo "  macos         Build for macOS (amd64 and arm64)"
    echo "  all           Build for all platforms"
    echo "  clean         Clean build artifacts"
    echo "  test          Run tests"
    echo "  run           Build and run with demo service"
    echo "  help          Show this help message"
    echo ""
    echo "Environment variables:"
    echo "  VERSION       Version string (default: $VERSION)"
    echo ""
    echo "Examples:"
    echo "  $0                    # Build for current platform"
    echo "  $0 all                # Build for all platforms"
    echo "  VERSION=2.0.0 $0 all  # Build all with custom version"
}

# Check if Go is installed
check_go() {
    if ! command -v go &> /dev/null; then
        log_error "Go is not installed or not in PATH"
        echo "Please install Go from https://golang.org/dl/"
        exit 1
    fi
    log_info "Using $(go version)"
}

# Download dependencies
deps() {
    log_info "Downloading dependencies..."
    go mod download
    go mod tidy
}

# Build for current platform
build_current() {
    log_info "Building $BINARY_NAME for current platform..."
    
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    go build \
        -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -w -s" \
        -o $BINARY_NAME \
        $MAIN_PATH
    
    log_success "Build complete: $BINARY_NAME"
}

# Build for Linux
build_linux() {
    log_info "Building $BINARY_NAME for Linux (amd64)..."
    mkdir -p $BUILD_DIR
    
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    GOOS=linux GOARCH=amd64 go build \
        -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -w -s" \
        -o $BUILD_DIR/$BINARY_NAME-linux-amd64 \
        $MAIN_PATH
    
    log_success "Linux build complete: $BUILD_DIR/$BINARY_NAME-linux-amd64"
}

# Build for Windows
build_windows() {
    log_info "Building $BINARY_NAME for Windows (amd64)..."
    mkdir -p $BUILD_DIR
    
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    GOOS=windows GOARCH=amd64 go build \
        -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -w -s" \
        -o $BUILD_DIR/$BINARY_NAME-windows-amd64.exe \
        $MAIN_PATH
    
    log_success "Windows build complete: $BUILD_DIR/$BINARY_NAME-windows-amd64.exe"
}

# Build for macOS
build_macos() {
    log_info "Building $BINARY_NAME for macOS (amd64 and arm64)..."
    mkdir -p $BUILD_DIR
    
    COMMIT=$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
    BUILD_TIME=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    
    # Intel Mac
    GOOS=darwin GOARCH=amd64 go build \
        -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -w -s" \
        -o $BUILD_DIR/$BINARY_NAME-darwin-amd64 \
        $MAIN_PATH
    
    # Apple Silicon Mac
    GOOS=darwin GOARCH=arm64 go build \
        -ldflags "-X main.Version=$VERSION -X main.Commit=$COMMIT -X main.BuildTime=$BUILD_TIME -w -s" \
        -o $BUILD_DIR/$BINARY_NAME-darwin-arm64 \
        $MAIN_PATH
    
    log_success "macOS builds complete:"
    echo "   $BUILD_DIR/$BINARY_NAME-darwin-amd64"
    echo "   $BUILD_DIR/$BINARY_NAME-darwin-arm64"
}

# Build for all platforms
build_all() {
    build_linux
    build_windows
    build_macos
    
    log_success "All platform builds complete!"
    echo ""
    echo "Build artifacts:"
    ls -la $BUILD_DIR/
}

# Clean build artifacts
clean() {
    log_info "Cleaning build artifacts..."
    rm -rf $BUILD_DIR
    rm -rf dist
    go clean
    log_success "Clean complete!"
}

# Run tests
test() {
    log_info "Running tests..."
    go test -v ./...
    log_success "Tests complete!"
}

# Build and run with demo service
run() {
    build_current
    log_info "Running $BINARY_NAME with OData demo service..."
    ./$BINARY_NAME --trace --service https://services.odata.org/V2/OData/OData.svc/
}

# Main script logic
main() {
    check_go
    
    case "${1:-build}" in
        "build"|"")
            deps
            build_current
            ;;
        "linux")
            deps
            build_linux
            ;;
        "windows")
            deps
            build_windows
            ;;
        "macos")
            deps
            build_macos
            ;;
        "all")
            deps
            build_all
            ;;
        "clean")
            clean
            ;;
        "test")
            test
            ;;
        "run")
            deps
            run
            ;;
        "help"|"-h"|"--help")
            usage
            ;;
        *)
            log_error "Unknown command: $1"
            echo ""
            usage
            exit 1
            ;;
    esac
}

# Run main function with all arguments
main "$@"