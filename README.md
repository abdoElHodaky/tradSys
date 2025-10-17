# TradSys - High-Frequency Trading Platform

A high-performance, microservices-based trading platform built with Go, featuring real-time market data streaming, low-latency order execution, and advanced risk management.

## 🏗️ Architecture Overview

TradSys follows a modern microservices architecture designed for high-frequency trading requirements:

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │  Mobile Client  │    │  Trading Bot    │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          └──────────────────────┼──────────────────────┘
                                 │
                    ┌─────────────▼───────────────┐
                    │       API Gateway           │
                    │  • Authentication           │
                    │  • Rate Limiting            │
                    │  • Request Routing          │
                    │  • Circuit Breaker          │
                    └─────────────┬───────────────┘
                                  │
        ┌─────────────────────────┼─────────────────────────┐
        │                         │                         │
┌───────▼────────┐    ┌───────────▼────────┐    ┌───────────▼────────┐
│ Market Data    │    │   Order Service    │    │   Risk Service     │
│ Service        │    │                    │    │                    │
│ • Real-time    │    │ • Order Creation   │    │ • Position Limits  │
│ • Historical   │    │ • Execution        │    │ • Risk Validation  │
│ • Symbols      │    │ • Management       │    │ • Circuit Breakers │
└───────┬────────┘    └───────────┬────────┘    └───────────┬────────┘
        │                         │                         │
        └─────────────────────────┼─────────────────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │    WebSocket Service        │
                    │  • Real-time Streaming      │
                    │  • Market Data Push         │
                    │  • Order Updates            │
                    └─────────────┬───────────────┘
                                  │
                    ┌─────────────▼───────────────┐
                    │      Data Layer             │
                    │  • PostgreSQL (GORM)       │
                    │  • In-memory Cache          │
                    │  • NATS Messaging           │
                    └─────────────────────────────┘
```

### Core Services

1. **🌐 API Gateway** (`cmd/gateway/`)
   - Entry point for all client requests
   - Authentication & authorization
   - Rate limiting & circuit breaker
   - Service discovery & load balancing

2. **📊 Market Data Service** (`cmd/marketdata/`)
   - Real-time market data streaming
   - Historical data retrieval
   - Symbol management
   - OHLCV data processing

3. **📋 Order Service** (`cmd/orders/`)
   - Order lifecycle management
   - Trading strategy execution
   - Order validation & routing
   - Execution reporting

4. **⚠️ Risk Service** (`cmd/risk/`)
   - Real-time risk monitoring
   - Position limit enforcement
   - Pre-trade risk checks
   - Circuit breaker management

5. **🔌 WebSocket Service** (`cmd/ws/`)
   - Real-time data streaming
   - Client connection management
   - Market data subscriptions
   - Order status updates

## 🛠️ Technology Stack

| Component | Technology | Purpose |
|-----------|------------|---------|
| **Backend Framework** | Go + Gin | High-performance HTTP server |
| **Communication** | gRPC + WebSockets | Internal services & real-time client communication |
| **Service Mesh** | go-micro | Service discovery, resilience, load balancing |
| **Event Streaming** | NATS | Asynchronous messaging & event sourcing |
| **Database** | PostgreSQL + GORM | Persistent storage with ORM |
| **Caching** | go-cache | In-memory caching for performance |
| **Observability** | Jaeger + Prometheus | Distributed tracing & metrics |
| **Dependency Injection** | Uber FX | Clean dependency management |
| **Configuration** | Viper | Environment-based configuration |

## 📁 Project Structure

```
tradSys/
├── cmd/                          # Service entry points
│   ├── gateway/                  # API Gateway service
│   ├── marketdata/               # Market Data service
│   ├── orders/                   # Order Management service
│   ├── risk/                     # Risk Management service
│   └── ws/                       # WebSocket service
├── internal/                     # Internal packages
│   ├── api/                      # API handlers & middleware
│   ├── auth/                     # Authentication & authorization
│   ├── common/                   # Shared utilities & patterns
│   ├── config/                   # Configuration management
│   ├── db/                       # Database models & repositories
│   ├── gateway/                  # Gateway-specific logic
│   ├── marketdata/               # Market data processing
│   ├── micro/                    # Microservice utilities
│   ├── orders/                   # Order management logic
│   ├── risk/                     # Risk management logic
│   ├── statistics/               # Statistical analysis
│   ├── strategy/                 # Trading strategies
│   ├── transport/                # Transport layer (WebSocket, etc.)
│   └── ws/                       # WebSocket handlers
├── proto/                        # Protocol Buffer definitions
├── tests/                        # Test files
├── config/                       # Configuration files
└── docs/                         # Documentation
```

## 🚀 Recent Improvements

### Codebase Modernization (2025-10-17)

We've recently completed a comprehensive codebase improvement initiative:

#### ✅ **Phase 1-2: Repository Unification**
- Consolidated duplicate market data repositories
- Standardized to GORM for consistent database access
- Implemented camelCase naming conventions
- Unified error handling patterns

#### ✅ **Phase 3: Service Registration Simplification**
- Created common service registration utilities
- Standardized fx.Module patterns across services
- Implemented consistent lifecycle management
- Added unified error handling for service startup

#### ✅ **Phase 4: Service Forwarding Implementation**
- Replaced placeholder service forwarding with actual proxy implementation
- Integrated service discovery with load balancing
- Added circuit breaker patterns for resilience
- Implemented health checking for downstream services

#### ✅ **Phase 5: Configuration Management**
- Unified configuration structures across services
- Standardized environment variable naming
- Added configuration validation
- Resolved merge conflicts and duplications

#### ✅ **Phase 6: TODO Cleanup**
- Completed WebSocket functionality implementation
- Added missing imports and dependencies
- Prepared market data subscription handlers
- Enhanced order management via WebSocket

#### ✅ **Phase 7: Handler Pattern Optimization**
- Created common handler utilities (`HandlerUtils`)
- Implemented standardized API response formats
- Added unified request validation middleware
- Created generic pagination and error handling patterns

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
