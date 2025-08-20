FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git protoc protobuf-dev

# Set working directory
WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Generate Protocol Buffer code
RUN mkdir -p proto/marketdata proto/orders proto/risk proto/ws
RUN go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
RUN go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
RUN protoc --go_out=. --go-grpc_out=. proto/marketdata/marketdata.proto
RUN protoc --go_out=. --go-grpc_out=. proto/orders/orders.proto
RUN protoc --go_out=. --go-grpc_out=. proto/risk/risk.proto
RUN protoc --go_out=. --go-grpc_out=. proto/ws/message.proto

# Build the API Gateway
RUN CGO_ENABLED=0 GOOS=linux go build -a -o gateway cmd/gateway/main.go

# Create final image
FROM alpine:latest

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy binary from builder
COPY --from=builder /app/gateway .

# Copy configuration
COPY config/config.yaml ./config/

# Create data directory
RUN mkdir -p /app/data

# Set environment variables
ENV GIN_MODE=release

# Expose ports
EXPOSE 8000

# Run the application
CMD ["./gateway"]

