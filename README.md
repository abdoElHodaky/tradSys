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

#### ✅ **Phase 8: Error Handling and Logging Consistency**
- Implemented correlation ID middleware for request tracing
- Added distributed logging with correlation tracking
- Completed WebSocket functionality implementations
- Added comprehensive health check endpoints

## ✨ Features

### 🚀 **Core Trading Features**
- **Real-time Market Data**: WebSocket streaming with symbol subscriptions
- **Low-latency Order Execution**: High-performance order processing
- **Advanced Trading Strategies**: Market making, statistical arbitrage, pairs trading
- **Risk Management**: Position limits, circuit breakers, pre-trade validation
- **Statistical Analysis**: Cointegration testing, correlation analysis, spread calculation

### 🔒 **Security & Authentication**
- **JWT Authentication**: Secure token-based authentication
- **Role-based Authorization**: Admin, trader, and viewer roles
- **Rate Limiting**: IP and path-based request throttling
- **Security Headers**: CORS, CSP, and other security middleware
- **Input Validation**: Comprehensive request validation and sanitization

### 📊 **Observability & Monitoring**
- **Request Tracing**: Correlation ID tracking across all services
- **Structured Logging**: Consistent logging with correlation context
- **Health Checks**: Liveness, readiness, and dependency health monitoring
- **Metrics Collection**: Prometheus-compatible metrics (ready for integration)
- **Distributed Tracing**: Jaeger integration for request flow tracking

### 🏗️ **Architecture & Performance**
- **Microservices Architecture**: Clean separation of concerns
- **Service Discovery**: Automatic service registration and discovery
- **Circuit Breakers**: Resilience patterns for external dependencies
- **Connection Pooling**: Optimized database connections
- **Caching Strategy**: In-memory caching for performance
- **Load Balancing**: Request distribution across service instances

## 🔌 API Endpoints

### **Health & Monitoring**
```
GET /health          # Overall service health
GET /health/live     # Liveness probe (K8s)
GET /health/ready    # Readiness probe (K8s)
```

### **Authentication**
```
POST /auth/login     # User authentication
POST /auth/refresh   # Token refresh
POST /auth/logout    # User logout
```

### **Market Data**
```
GET  /api/v1/pairs                    # List all trading pairs
GET  /api/v1/pairs/{id}               # Get specific pair
POST /api/v1/pairs                    # Create new pair
PUT  /api/v1/pairs/{id}               # Update pair
DELETE /api/v1/pairs/{id}             # Delete pair
GET  /api/v1/pairs/{id}/statistics    # Get pair statistics
GET  /api/v1/pairs/{id}/positions     # Get position history
POST /api/v1/pairs/{id}/analyze       # Analyze pair correlation
```

### **WebSocket Endpoints**
```
WS /ws                               # WebSocket connection
  ├── marketdata.subscribe           # Subscribe to market data
  ├── marketdata.unsubscribe         # Unsubscribe from market data
  ├── order.submit                   # Submit trading order
  └── order.cancel                   # Cancel trading order
```

### **Request/Response Format**
```json
{
  "success": true,
  "data": { ... },
  "message": "Optional message",
  "pagination": {
    "page": 1,
    "page_size": 20,
    "total": 100,
    "total_pages": 5
  }
}
```

## 🏗️ Enhanced Architecture Diagrams

### **Request Flow with Correlation Tracking**
```
┌─────────────┐    ┌─────────────────────────────────────┐
│   Client    │───▶│         API Gateway                 │
│             │    │  ┌─────────────────────────────────┐ │
└─────────────┘    │  │ 1. Generate Correlation ID      │ │
                   │  │ 2. Add Security Headers         │ │
                   │  │ 3. Rate Limiting Check          │ │
                   │  │ 4. JWT Validation               │ │
                   │  │ 5. Route to Service             │ │
                   │  └─────────────────────────────────┘ │
                   └─────────────┬───────────────────────┘
                                 │ X-Correlation-ID: abc-123
                                 ▼
                   ┌─────────────────────────────────────┐
                   │        Microservice                 │
                   │  ┌─────────────────────────────────┐ │
                   │  │ 1. Extract Correlation ID       │ │
                   │  │ 2. Add to Logging Context       │ │
                   │  │ 3. Process Business Logic       │ │
                   │  │ 4. Database Operations          │ │
                   │  │ 5. Return Response              │ │
                   │  └─────────────────────────────────┘ │
                   └─────────────┬───────────────────────┘
                                 │ X-Correlation-ID: abc-123
                                 ▼
                   ┌─────────────────────────────────────┐
                   │            Response                 │
                   │  • Same Correlation ID              │
                   │  • Structured JSON                  │
                   │  • Consistent Error Format          │
                   └─────────────────────────────────────┘
```

### **WebSocket Real-time Data Flow**
```
┌─────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Client    │    │  WebSocket Hub  │    │ Market Data     │
│             │    │                 │    │ Service         │
│             │───▶│ 1. Connect      │    │                 │
│             │    │ 2. Authenticate │    │                 │
│             │    │ 3. Subscribe    │───▶│ 4. Add Client   │
│             │    │                 │    │    to Symbol    │
│             │    │                 │    │                 │
│             │    │                 │◀───│ 5. Price Update │
│             │◀───│ 6. Broadcast    │    │                 │
│             │    │    to Clients   │    │                 │
└─────────────┘    └─────────────────┘    └─────────────────┘

Message Types:
• marketdata.subscribe    → Subscribe to symbol
• marketdata.unsubscribe  → Unsubscribe from symbol  
• order.submit           → Submit trading order
• order.cancel           → Cancel existing order
• price.update           → Real-time price data
• order.status           → Order status updates
```

## 🧪 Testing & Quality Assurance

### **Current Test Coverage**
- **JWT Authentication**: Unit tests for token generation and validation
- **Gateway Integration**: End-to-end API gateway testing
- **Health Checks**: Liveness and readiness probe testing

### **Running Tests**
```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./internal/auth/...

# Run integration tests
go test ./tests/integration/...
```

### **Test Structure**
```
tests/
├── integration/           # Integration tests
│   ├── gateway/          # API gateway tests
│   ├── websocket/        # WebSocket tests
│   └── database/         # Database integration tests
├── unit/                 # Unit tests
│   ├── handlers/         # Handler unit tests
│   ├── services/         # Service unit tests
│   └── repositories/     # Repository unit tests
└── fixtures/             # Test data and fixtures
```

### **Quality Metrics**
- **Code Coverage**: Target 80%+ coverage
- **Linting**: golangci-lint with strict rules
- **Security**: gosec security scanning
- **Performance**: Benchmark tests for critical paths

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

## 🚀 Deployment & Operations

### **Kubernetes Deployment**
```bash
# Deploy infrastructure components
kubectl apply -f deployments/kubernetes/infrastructure.yaml

# Deploy services
kubectl apply -f deployments/kubernetes/gateway.yaml
kubectl apply -f deployments/kubernetes/marketdata.yaml
kubectl apply -f deployments/kubernetes/orders.yaml
kubectl apply -f deployments/kubernetes/risk.yaml
kubectl apply -f deployments/kubernetes/ws.yaml
```

### **Docker Compose (Development)**
```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down
```

### **Environment Variables**
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=tradingsystem
DB_USER=postgres
DB_PASSWORD=password

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRY=24h

# Services
GATEWAY_PORT=8080
MARKETDATA_PORT=8081
ORDERS_PORT=8082
RISK_PORT=8083
WS_PORT=8084

# Monitoring
PROMETHEUS_URL=http://localhost:9090
JAEGER_ENDPOINT=http://localhost:14268/api/traces
```

## 📊 Monitoring & Observability

### **Health Checks**
```bash
# Check service health
curl http://localhost:8080/health

# Kubernetes probes
curl http://localhost:8080/health/live    # Liveness
curl http://localhost:8080/health/ready   # Readiness
```

### **Metrics Collection**
- **Prometheus**: Metrics scraping and storage
- **Grafana**: Dashboards and visualization
- **Jaeger**: Distributed tracing
- **ELK Stack**: Log aggregation and analysis

### **Key Metrics**
- Request latency (p50, p95, p99)
- Request rate and error rate
- Database connection pool usage
- WebSocket connection count
- Order processing latency
- Memory and CPU utilization

### **Alerting Rules**
- High error rate (>5%)
- High latency (>500ms p95)
- Database connection failures
- Service unavailability
- Memory/CPU threshold breaches

## 🔧 Development & Maintenance

### **Code Quality**
```bash
# Linting
golangci-lint run

# Security scanning
gosec ./...

# Dependency check
go mod tidy
go mod verify

# Format code
gofmt -w .
```

### **Performance Testing**
```bash
# Load testing with hey
hey -n 10000 -c 100 http://localhost:8080/health

# WebSocket load testing
# Use custom WebSocket load testing tools
```

### **Database Migrations**
```bash
# Run migrations
go run cmd/migrate/main.go up

# Rollback migrations
go run cmd/migrate/main.go down

# Create new migration
go run cmd/migrate/main.go create add_new_table
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
