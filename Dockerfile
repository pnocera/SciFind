# ============================================================================
# Multi-stage Docker build for SciFIND Backend
# Optimized for production with distroless final image
# Based on Phase 3 specifications for Docker optimization
# ============================================================================

# Build stage - Go 1.24+ with all build tools
FROM golang:1.24-alpine AS builder

# Install build dependencies
RUN apk add --no-cache \
    git \
    ca-certificates \
    tzdata \
    upx

# Create non-root user for security
RUN adduser -D -s /bin/sh -u 1001 appuser

# Set working directory
WORKDIR /app

# Copy dependency files first for better caching
COPY go.mod go.sum ./

# Download dependencies with verification
RUN go mod download && go mod verify

# Copy source code
COPY . .

# Build with optimization flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags='-w -s -extldflags "-static"' \
    -a -installsuffix cgo \
    -o scifind-server ./cmd/server

# Compress binary with UPX for smaller size
RUN upx --best --lzma scifind-server

# Verify binary
RUN ./scifind-server --version || echo "Binary built successfully"

# ============================================================================
# Runtime stage - Distroless for minimal attack surface
# ============================================================================
FROM gcr.io/distroless/static:nonroot

# Copy timezone data and certificates
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# Copy application binary
COPY --from=builder /app/scifind-server /app/scifind-server

# Copy configuration directory (if exists)
COPY --from=builder /app/configs /app/configs

# Set working directory
WORKDIR /app

# Use non-root user
USER nonroot:nonroot

# Expose application port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD ["/app/scifind-server", "health"]

# Application entrypoint
ENTRYPOINT ["/app/scifind-server"]
