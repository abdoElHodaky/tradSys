# Multi-stage build for TradSys HFT Trading System
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git gcc musl-dev sqlite-dev ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the unified trading system with optimizations for HFT
RUN CGO_ENABLED=1 GOOS=linux go build \
    -ldflags="-w -s -extldflags '-static'" \
    -a -installsuffix cgo \
    -o tradsys \
    ./cmd/server

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
COPY --from=builder /app/tradsys .

# Copy configuration files
COPY --from=builder /app/config ./config

# Create necessary directories
RUN mkdir -p /app/data /app/logs /app/reports && \
    chown -R hft:hft /app

# Switch to non-root user
USER hft

# Expose ports for different services
EXPOSE 8080 8081 8082 9090 9091

# Health check for unified system
HEALTHCHECK --interval=15s --timeout=5s --start-period=10s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Set environment variables for production
ENV TRADSYS_ENVIRONMENT=production
ENV TRADSYS_CONFIG_PATH=/app/config/production.json
ENV TRADSYS_LOG_LEVEL=info
ENV TRADSYS_METRICS_ENABLED=true
ENV GIN_MODE=release

# Run the unified trading system
CMD ["./tradsys"]
