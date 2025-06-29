# OData MCP Bridge - Go Implementation
# Makefile for building, testing, and distributing

# Variables
BINARY_NAME=odata-mcp
MAIN_PATH=cmd/odata-mcp/main.go
BUILD_DIR=build
DIST_DIR=dist
VERSION?=1.0.1
COMMIT?=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_TIME?=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ")

# Go build flags
LDFLAGS=-ldflags "-X main.Version=$(VERSION) -X main.Commit=$(COMMIT) -X main.BuildTime=$(BUILD_TIME) -w -s"
GCFLAGS=-gcflags="all=-trimpath=$(PWD)"
ASMFLAGS=-asmflags="all=-trimpath=$(PWD)"

# Default target
.PHONY: all
all: build

# Help target
.PHONY: help
help:
	@echo "OData MCP Bridge - Build System"
	@echo "================================"
	@echo ""
	@echo "Available targets:"
	@echo "  build         - Build binary for current platform"
	@echo "  build-all     - Build binaries for all platforms"
	@echo "  test          - Run tests"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  run           - Build and run with sample service"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  dist          - Create distribution packages"
	@echo "  docker        - Build Docker image"
	@echo "  docker-run    - Run in Docker container"
	@echo ""
	@echo "Cross-compilation targets:"
	@echo "  build-linux   - Build for Linux (amd64)"
	@echo "  build-windows - Build for Windows (amd64)"
	@echo "  build-macos   - Build for macOS (amd64 and arm64)"
	@echo ""
	@echo "Environment variables:"
	@echo "  VERSION       - Version string (default: $(VERSION))"
	@echo "  COMMIT        - Git commit hash (default: auto-detected)"
	@echo "  BUILD_TIME    - Build timestamp (default: current time)"

# Build for current platform
.PHONY: build
build: deps
	@echo "Building $(BINARY_NAME) for current platform..."
	go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BINARY_NAME) $(MAIN_PATH)
	@echo "✅ Build complete: $(BINARY_NAME)"

# Cross-compilation targets
.PHONY: build-linux
build-linux: deps
	@echo "Building $(BINARY_NAME) for Linux (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=linux GOARCH=amd64 go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(MAIN_PATH)
	@echo "✅ Linux build complete: $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64"

.PHONY: build-windows
build-windows: deps
	@echo "Building $(BINARY_NAME) for Windows (amd64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=windows GOARCH=amd64 go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(MAIN_PATH)
	@echo "✅ Windows build complete: $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe"
	rm -f "/mnt/c/bin/$(BINARY_NAME).exe" 2>/dev/null || true
	cp "$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe" "/mnt/c/bin/$(BINARY_NAME).exe"

.PHONY: build-macos
build-macos: deps
	@echo "Building $(BINARY_NAME) for macOS (amd64 and arm64)..."
	@mkdir -p $(BUILD_DIR)
	GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(MAIN_PATH)
	GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(MAIN_PATH)
	@echo "✅ macOS builds complete:"
	@echo "   $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64"
	@echo "   $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64"

# Build for all platforms
.PHONY: build-all
build-all: build-linux build-windows build-macos
	@echo "✅ All platform builds complete!"
	@ls -la $(BUILD_DIR)/

# Install binary
.PHONY: install
install: build
	@echo "Installing $(BINARY_NAME) to GOPATH/bin..."
	go install $(LDFLAGS) $(MAIN_PATH)
	@echo "✅ Installation complete!"

# Download dependencies
.PHONY: deps
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy

# Run tests
.PHONY: test
test:
	@echo "Running tests..."
	go test -v ./...
	@echo "✅ Tests complete!"

# Run with race detection
.PHONY: test-race
test-race:
	@echo "Running tests with race detection..."
	go test -v -race ./...

# Format code
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "✅ Code formatted!"

# Run linter (requires golangci-lint)
.PHONY: lint
lint:
	@if command -v golangci-lint >/dev/null 2>&1; then \
		echo "Running linter..."; \
		golangci-lint run; \
		echo "✅ Linting complete!"; \
	else \
		echo "⚠️ golangci-lint not found. Install with:"; \
		echo "   go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest"; \
	fi

# Clean build artifacts
.PHONY: clean
clean:
	@echo "Cleaning build artifacts..."
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	go clean
	@echo "✅ Clean complete!"

# Run with sample service
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) with OData demo service..."
	./$(BINARY_NAME) --trace --service https://services.odata.org/V2/OData/OData.svc/

# Run with Northwind service
.PHONY: run-northwind
run-northwind: build
	@echo "Running $(BINARY_NAME) with Northwind service..."
	./$(BINARY_NAME) --trace --service https://services.odata.org/V2/Northwind/Northwind.svc/

# Create distribution packages
.PHONY: dist
dist: build-all
	@echo "Creating distribution packages..."
	@mkdir -p $(DIST_DIR)
	
	# Linux
	@mkdir -p $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64
	cp $(BUILD_DIR)/$(BINARY_NAME)-linux-amd64 $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64/$(BINARY_NAME)
	cp README.md $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64/
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-$(VERSION)-linux-amd64/
	
	# Windows
	@mkdir -p $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64
	cp $(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64/$(BINARY_NAME).exe
	cp README.md $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64/
	cd $(DIST_DIR) && zip -r $(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-$(VERSION)-windows-amd64/
	
	# macOS Intel
	@mkdir -p $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64
	cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-amd64 $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64/$(BINARY_NAME)
	cp README.md $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64/
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-$(VERSION)-darwin-amd64/
	
	# macOS Apple Silicon
	@mkdir -p $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64
	cp $(BUILD_DIR)/$(BINARY_NAME)-darwin-arm64 $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64/$(BINARY_NAME)
	cp README.md $(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64/
	cd $(DIST_DIR) && tar -czf $(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-$(VERSION)-darwin-arm64/
	
	@echo "✅ Distribution packages created:"
	@ls -la $(DIST_DIR)/*.tar.gz $(DIST_DIR)/*.zip 2>/dev/null || true

# Docker targets
.PHONY: docker
docker:
	@echo "Building Docker image..."
	docker build -t $(BINARY_NAME):$(VERSION) -t $(BINARY_NAME):latest .
	@echo "✅ Docker image built: $(BINARY_NAME):$(VERSION)"

.PHONY: docker-run
docker-run: docker
	@echo "Running Docker container..."
	docker run --rm -it $(BINARY_NAME):latest --help

# Development helpers
.PHONY: dev
dev: fmt test build
	@echo "✅ Development build complete!"

.PHONY: watch
watch:
	@if command -v entr >/dev/null 2>&1; then \
		echo "Watching for changes... (requires entr)"; \
		find . -name "*.go" | entr -r make dev; \
	else \
		echo "⚠️ Watch requires 'entr'. Install with: brew install entr"; \
	fi

# Show build info
.PHONY: info
info:
	@echo "Build Information:"
	@echo "=================="
	@echo "Binary Name: $(BINARY_NAME)"
	@echo "Version:     $(VERSION)"
	@echo "Commit:      $(COMMIT)"
	@echo "Build Time:  $(BUILD_TIME)"
	@echo "Go Version:  $(shell go version)"
	@echo "GOOS:        $(shell go env GOOS)"
	@echo "GOARCH:      $(shell go env GOARCH)"

# Check dependencies
.PHONY: check
check:
	@echo "Checking dependencies..."
	go mod verify
	go vet ./...
	@echo "✅ Dependencies verified!"

# Generate build metadata
.PHONY: version
version:
	@echo "$(VERSION)"

# Quick development iteration
.PHONY: quick
quick:
	go build -o $(BINARY_NAME) $(MAIN_PATH) && ./$(BINARY_NAME) --help
