# Multi-stage build for HFT Trading System
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the application with optimizations for HFT
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o hft-server \
    ./cmd/hft-server

# Production stage
FROM alpine:3.18

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates \
    tzdata \
    sqlite \
    wget \
    && rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1001 -S hft && \
    adduser -u 1001 -S hft -G hft

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/hft-server .

# Copy configuration files
COPY --from=builder /app/configs ./configs

# Create necessary directories
RUN mkdir -p /app/data /app/logs && \
    chown -R hft:hft /app

# Switch to non-root user
USER hft

# Expose ports
EXPOSE 8080 9090

# Health check
HEALTHCHECK --interval=30s --timeout=10s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables
ENV HFT_ENVIRONMENT=production
ENV HFT_CONFIG_PATH=/app/configs/hft-config.yaml
ENV GIN_MODE=release

# Run the application
CMD ["./hft-server"]
