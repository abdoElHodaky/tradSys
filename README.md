# High-Frequency Trading Platform

A high-performance trading platform built with Go, Gin, and WebSockets for real-time market data and order execution.

## Architecture

The platform follows a microservices architecture with the following components:

1. **API Gateway**: Entry point for all client requests, handles authentication, rate limiting, and request routing
2. **Market Data Service**: Provides real-time and historical market data
3. **Order Service**: Handles order creation, execution, and management
4. **Risk Service**: Monitors positions and validates orders against risk parameters
5. **WebSocket Service**: Streams real-time data to clients

## Technology Stack

- **Backend Framework**: Go with Gin
- **Communication**: gRPC for internal services, WebSockets for client communication
- **Service Mesh**: go-micro for service discovery and resilience
- **Event Streaming**: NATS for asynchronous messaging
- **Database**: PostgreSQL for persistent storage
- **Caching**: In-memory caching with go-cache
- **Observability**: Distributed tracing with Jaeger, metrics with Prometheus
- **Deployment**: Kubernetes for orchestration

## Features

- Real-time market data streaming via WebSockets
- Low-latency order execution
- Advanced trading strategies (market making, statistical arbitrage)
- Risk management with position limits and circuit breakers
- Authentication and authorization
- Performance optimization with object pooling
- Statistical analysis (cointegration, correlation)
- High-precision latency tracking

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Docker and Docker Compose
- Protocol Buffers compiler
- PostgreSQL (optional for local development)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/abdoElHodaky/tradSys.git
   cd tradSys
   ```

2. Generate Protocol Buffer code:
   ```bash
   ./scripts/generate_proto.sh
   ```

3. Start the services with Docker Compose:
   ```bash
   docker-compose up -d
   ```

4. Access the API Gateway at http://localhost:8000

### Development

1. Install dependencies:
   ```bash
   go mod download
   ```

2. Run a specific service:
   ```bash
   go run cmd/gateway/main.go
   go run cmd/marketdata/main.go
   go run cmd/orders/main.go
   go run cmd/risk/main.go
   go run cmd/ws/main.go
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

## API Documentation

The API documentation is available at http://localhost:8000/swagger/index.html when running the API Gateway.

## Monitoring

- Prometheus metrics: http://localhost:9090
- Grafana dashboards: http://localhost:3000
- Jaeger tracing: http://localhost:16686

## Deployment

The platform can be deployed to Kubernetes using the manifests in the `deployments/kubernetes` directory:

```bash
kubectl apply -f deployments/kubernetes/
```

## Performance Considerations

The platform is optimized for high-frequency trading with the following features:

- Object pooling for market data and orders
- Efficient goroutine management
- Connection pooling for databases and WebSockets
- Buffer pools for market data
- Incremental statistics calculation
- Query optimization and caching

## License

This project is licensed under the MIT License - see the LICENSE file for details.

