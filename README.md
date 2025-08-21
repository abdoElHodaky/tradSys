# TradSys - High-Frequency Trading Platform

A high-performance trading platform built with Go, Gin, and fx dependency injection for real-time market data and order execution.

## Architecture

The platform follows a microservices architecture with the following components:

1. **API Gateway**: Entry point for all client requests, handles authentication, rate limiting, and request routing
2. **Market Data Service**: Provides real-time and historical market data
3. **Order Service**: Handles order creation, execution, and management
4. **Risk Service**: Monitors positions and validates orders against risk parameters
5. **WebSocket Service**: Streams real-time data to clients

## Technology Stack

- **Backend Framework**: Go with Gin
- **Dependency Injection**: Uber's fx framework
- **Communication**: gRPC for internal services, WebSockets for client communication
- **Service Mesh**: go-micro for service discovery and resilience
- **Event Streaming**: NATS for asynchronous messaging
- **Database**: PostgreSQL with GORM for persistent storage
- **Caching**: In-memory caching with go-cache
- **Observability**: Distributed tracing with Jaeger, metrics with Prometheus
- **Deployment**: Kubernetes for orchestration

## Key Components

### Authentication API

The Authentication API provides secure user authentication and authorization:

- JWT-based authentication with token refresh
- Role-based access control
- Secure password hashing
- Context propagation for user identification
- API endpoints:
  - POST /api/auth/login - Authenticate a user
  - POST /api/auth/refresh - Refresh an access token
  - POST /api/auth/register - Register a new user

### Order Management API

The Order Management API handles all order-related operations:

- Full CRUD operations for orders
- Database integration with GORM
- Order validation and execution
- Transaction support
- API endpoints:
  - GET /api/orders - List orders with filtering
  - POST /api/orders - Create a new order
  - GET /api/orders/:id - Get a specific order
  - DELETE /api/orders/:id - Cancel an order

### Risk Management API

The Risk Management API provides risk control and position management:

- Position tracking and management
- Risk limit creation and enforcement
- Order validation against risk parameters
- Circuit breaker functionality
- API endpoints:
  - GET /api/risk/positions - Get user positions
  - GET /api/risk/limits - Get user risk limits
  - POST /api/risk/limits - Create or update risk limits
  - DELETE /api/risk/limits/:id - Delete a risk limit
  - POST /api/risk/validate - Validate an order against risk limits

## Dependency Injection with fx

The platform uses Uber's fx framework for dependency injection, providing:

- Modular code organization
- Simplified testing
- Automatic dependency resolution
- Lifecycle management
- Clean separation of concerns

Example of a module definition:

```go
// Module provides the risk service module for fx
var Module = fx.Options(
    // Provide the risk repository
    fx.Provide(func(db *gorm.DB, logger *zap.Logger) *repositories.RiskRepository {
        return repositories.NewRiskRepository(db, logger)
    }),
    
    // Provide the risk service
    fx.Provide(NewService),
)
```

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

2. Run the application:
   ```bash
   go run cmd/main.go
   ```

3. Run tests:
   ```bash
   go test ./...
   ```

## API Documentation

The API documentation is available at http://localhost:8000/docs/swagger-ui/index.html when running the application.

## Project Structure

```
tradSys/
├── cmd/                    # Application entry points
│   └── main.go             # Main application entry point
├── internal/               # Internal packages
│   ├── api/                # API handlers and routes
│   │   ├── handlers/       # HTTP handlers
│   │   └── module.go       # API module definition
│   ├── auth/               # Authentication
│   │   ├── middleware.go   # Authentication middleware
│   │   └── module.go       # Auth module definition
│   ├── config/             # Configuration
│   ├── db/                 # Database
│   │   ├── models.go       # Database models
│   │   ├── module.go       # Database module definition
│   │   └── repositories/   # Database repositories
│   ├── gateway/            # API Gateway
│   ├── orders/             # Order management
│   │   ├── interface.go    # Service interface
│   │   ├── module.go       # Order module definition
│   │   └── service.go      # Order service implementation
│   └── risk/               # Risk management
│       ├── interface.go    # Service interface
│       ├── module.go       # Risk module definition
│       └── service.go      # Risk service implementation
├── proto/                  # Protocol Buffers definitions
│   ├── auth/               # Auth service protos
│   ├── orders/             # Order service protos
│   └── risk/               # Risk service protos
├── docs/                   # Documentation
│   ├── swagger-ui/         # Swagger UI
│   └── swagger.yaml        # OpenAPI specification
├── scripts/                # Utility scripts
├── docker-compose.yml      # Docker Compose configuration
└── README.md               # Project documentation
```

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

