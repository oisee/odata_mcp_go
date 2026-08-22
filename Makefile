# OData MCP Bridge - Go Implementation
# Makefile for building, testing, and distributing

# Variables
BINARY_NAME=odata-mcp
MAIN_PATH=cmd/odata-mcp/main.go
BUILD_DIR=build
DIST_DIR=dist

# Add .exe extension on Windows
ifeq ($(OS),Windows_NT)
BINARY_EXT=.exe
else
BINARY_EXT=
endif

# Version detection - uses git tags if available, otherwise generates from commit count
GIT_TAG=$(shell git describe --tags --exact-match 2>/dev/null)
GIT_COMMIT_COUNT=$(shell git rev-list --count HEAD 2>/dev/null || echo "0")
GIT_COMMIT_SHORT=$(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
GIT_DIRTY=$(shell test -n "`git status --porcelain`" && echo "-dirty" || echo "")

# If we have a tag, use it. Otherwise, generate version from commit count
ifeq ($(GIT_TAG),)
VERSION?=0.1.$(GIT_COMMIT_COUNT)$(GIT_DIRTY)
else
VERSION?=$(GIT_TAG)$(GIT_DIRTY)
endif

COMMIT?=$(GIT_COMMIT_SHORT)
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
	@echo "  test          - Run Go unit tests"
	@echo "  test-regression - Check if binary exists and works (catches ENOENT)"
	@echo "  test-e2e      - Run end-to-end tests"
	@echo "  test-all      - Run all tests including regression"
	@echo "  clean         - Clean build artifacts"
	@echo "  install       - Install binary to GOPATH/bin"
	@echo "  run           - Build and run with sample service"
	@echo "  deps          - Download dependencies"
	@echo "  fmt           - Format code"
	@echo "  lint          - Run linter (requires golangci-lint)"
	@echo "  dist          - Create distribution packages"
	@echo "  docker        - Build Docker image"
	@echo "  docker-run    - Run in Docker container"
	@echo "  version       - Display current version information"
	@echo ""
	@echo "Release targets:"
	@echo "  release       - Create a GitHub release (TAG=v1.x.x)"
	@echo "  release-local - Create release archives locally"
	@echo ""
	@echo "Cross-compilation targets:"
	@echo "  build-linux      - Build for Linux (amd64)"
	@echo "  build-windows    - Build for Windows (amd64)"
	@echo "  build-macos      - Build for macOS (amd64 and arm64)"
	@echo ""
	@echo "WSL-specific targets:"
	@echo "  build-all-wsl    - Build all platforms + copy Windows binary to /mnt/c/bin"
	@echo "  build-windows-wsl- Build Windows + copy to /mnt/c/bin"
	@echo ""
	@echo "Version information:"
	@echo "  Current version: $(VERSION)"
	@echo "  Commit: $(COMMIT)"
	@echo ""
	@echo "Versioning strategy:"
	@echo "  - If git tag exists on current commit, use the tag"
	@echo "  - Otherwise, use 0.1.<commit-count>"
	@echo "  - Append '-dirty' if there are uncommitted changes"

# Build for current platform
.PHONY: build
build: deps
	@echo "Building $(BINARY_NAME) for current platform..."
	go build $(LDFLAGS) $(GCFLAGS) $(ASMFLAGS) -o $(BINARY_NAME)$(BINARY_EXT) $(MAIN_PATH)
	@echo "✅ Build complete: $(BINARY_NAME)$(BINARY_EXT)"

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

# Build for all platforms with WSL integration (copies Windows binary to /mnt/c/bin)
.PHONY: build-all-wsl
build-all-wsl: build-all
	@echo "Copying Windows binary to WSL mount..."
	@if [ -d "/mnt/c/bin" ]; then \
		rm -f "/mnt/c/bin/$(BINARY_NAME).exe" 2>/dev/null || true; \
		cp "$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe" "/mnt/c/bin/$(BINARY_NAME).exe"; \
		echo "✅ Windows binary copied to /mnt/c/bin/$(BINARY_NAME).exe"; \
	else \
		echo "⚠️  /mnt/c/bin not found - skipping WSL copy"; \
	fi

# Build Windows binary and copy to WSL mount
.PHONY: build-windows-wsl
build-windows-wsl: build-windows
	@echo "Copying Windows binary to WSL mount..."
	@if [ -d "/mnt/c/bin" ]; then \
		rm -f "/mnt/c/bin/$(BINARY_NAME).exe" 2>/dev/null || true; \
		cp "$(BUILD_DIR)/$(BINARY_NAME)-windows-amd64.exe" "/mnt/c/bin/$(BINARY_NAME).exe"; \
		echo "✅ Windows binary copied to /mnt/c/bin/$(BINARY_NAME).exe"; \
	else \
		echo "⚠️  /mnt/c/bin not found - skipping WSL copy"; \
	fi

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

# Run regression test to ensure binary exists and works
.PHONY: test-regression
test-regression:
	@echo "Running regression tests (binary must exist)..."
	@if [ ! -f "$(BINARY_NAME)$(BINARY_EXT)" ]; then \
		echo "❌ Binary not found! Run 'make build' first."; \
		exit 1; \
	fi
	./test_regression_binary_exists.sh

# Run E2E tests (requires binary)
.PHONY: test-e2e
test-e2e: test-regression
	@echo "Running E2E tests..."
	./test_streamable_http.sh

# Run all tests including regression
.PHONY: test-all
test-all: test build test-regression test-e2e
	@echo "✅ All tests passed including regression checks!"

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
	rm -f $(BINARY_NAME) $(BINARY_NAME).exe
	rm -rf $(BUILD_DIR)
	rm -rf $(DIST_DIR)
	go clean
	@echo "✅ Clean complete!"

# Run with sample service
.PHONY: run
run: build
	@echo "Running $(BINARY_NAME) with OData demo service..."
	./$(BINARY_NAME)$(BINARY_EXT) --trace --service https://services.odata.org/V2/OData/OData.svc/

# Run with Northwind service
.PHONY: run-northwind
run-northwind: build
	@echo "Running $(BINARY_NAME) with Northwind service..."
	./$(BINARY_NAME)$(BINARY_EXT) --trace --service https://services.odata.org/V2/Northwind/Northwind.svc/

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

# Display version information
.PHONY: version
version:
	@echo "$(VERSION)"

# Check dependencies
.PHONY: check
check:
	@echo "Checking dependencies..."
	go mod verify
	go vet ./...
	@echo "✅ Dependencies verified!"


# Quick development iteration
.PHONY: quick
quick:
	go build -o $(BINARY_NAME)$(BINARY_EXT) $(MAIN_PATH) && ./$(BINARY_NAME)$(BINARY_EXT) --help

# Create a new release (requires gh CLI)
.PHONY: release
release:
	@if [ -z "$(TAG)" ]; then \
		echo "❌ Please specify TAG=v1.x.x"; \
		exit 1; \
	fi
	@if ! command -v gh >/dev/null 2>&1; then \
		echo "❌ GitHub CLI (gh) is required. Install from: https://cli.github.com"; \
		exit 1; \
	fi
	@echo "Creating release $(TAG)..."
	git tag -a $(TAG) -m "Release $(TAG)"
	git push origin $(TAG)
	@echo "✅ Tag pushed. GitHub Actions will create the release."
	@echo "Check progress at: https://github.com/$(shell git remote get-url origin | sed 's/.*github.com[:/]\(.*\)\.git/\1/')/actions"

# Create release archives locally
.PHONY: release-local
release-local: build-all
	@echo "Creating release archives..."
	@mkdir -p $(DIST_DIR)
	cd $(BUILD_DIR) && \
		tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-linux-amd64.tar.gz $(BINARY_NAME)-linux-amd64 && \
		zip ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-windows-amd64.zip $(BINARY_NAME)-windows-amd64.exe && \
		tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-amd64.tar.gz $(BINARY_NAME)-darwin-amd64 && \
		tar -czf ../$(DIST_DIR)/$(BINARY_NAME)-$(VERSION)-darwin-arm64.tar.gz $(BINARY_NAME)-darwin-arm64
	cd $(DIST_DIR) && sha256sum *.tar.gz *.zip > checksums.txt
	@echo "✅ Release archives created in $(DIST_DIR)/"
	@ls -la $(DIST_DIR)/
