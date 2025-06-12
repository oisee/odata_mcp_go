# Multi-stage build for minimal production image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /build

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -X main.Version=docker -X main.BuildTime=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
    -o odata-mcp \
    cmd/odata-mcp/main.go

# Final stage - minimal runtime image
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S appgroup && \
    adduser -u 1001 -S appuser -G appgroup

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /build/odata-mcp .

# Copy documentation
COPY README.md .

# Change ownership to non-root user
RUN chown -R appuser:appgroup /app

# Switch to non-root user
USER appuser

# Expose default port (not really needed for stdin/stdout MCP)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ./odata-mcp --help > /dev/null || exit 1

# Default command
ENTRYPOINT ["./odata-mcp"]

# Default arguments (can be overridden)
CMD ["--help"]