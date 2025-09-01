# TradSys - High-Performance Trading System Platform

TradSys is a comprehensive, high-performance trading system platform designed for professional trading operations. Built with a microservices architecture and event-driven design, it provides a robust foundation for building scalable trading applications with support for various asset classes, real-time market data integration, and advanced trading strategies.

## 🚀 Features

- **High-Performance Architecture**
  - Event-driven design for real-time processing
  - Microservices architecture for scalability and resilience
  - Optimized for low-latency trading operations
  - Horizontal scaling capabilities for handling increased load

- **Advanced Trading Capabilities**
  - Multi-asset class support (cryptocurrencies, forex, equities, etc.)
  - Real-time market data processing
  - Customizable trading strategies
  - Backtesting and strategy optimization
  - Risk management and position tracking

- **Resilient Infrastructure**
  - Circuit breakers for fault tolerance
  - Rate limiting to prevent system overload
  - Retry mechanisms for transient failures
  - Comprehensive monitoring and alerting

- **Extensible Design**
  - Plugin system for custom strategies and integrations
  - Lazy loading for efficient resource utilization
  - Modular components that can be deployed independently
  - Comprehensive API for external integrations

- **Decision Support System**
  - Advanced analytics for trading decisions
  - Real-time insights and recommendations
  - Multiple integration patterns (synchronous, asynchronous, streaming)
  - Customizable models and alerts

## 📋 System Requirements

- **Go 1.23** or higher
- **PostgreSQL 14** or higher
- **Redis 7** or higher
- **NATS** or **Kafka** for messaging
- **Docker** (optional, for containerized deployment)

## 🏗️ Architecture Overview

TradSys is built on a modern microservices architecture with the following core components:

### Core Services

- **API Gateway**: Entry point for external requests, handles routing and authentication
- **Order Service**: Manages order lifecycle, execution, and tracking
- **Market Data Service**: Processes and distributes real-time market data
- **Strategy Service**: Executes trading strategies and generates signals
- **Risk Management Service**: Enforces risk controls and limits
- **Position Service**: Tracks positions and exposures across accounts
- **Authentication Service**: Manages user authentication and authorization
- **Decision Support Service**: Provides analytics and decision-making capabilities

### Key Architectural Patterns

- **CQRS (Command Query Responsibility Segregation)**: Separates read and write operations
- **Event Sourcing**: Stores state changes as a sequence of events
- **Circuit Breaker**: Prevents cascading failures in distributed systems
- **Bulkhead**: Isolates components to contain failures
- **Rate Limiting**: Controls resource usage and prevents overload
- **Lazy Loading**: Loads components on-demand to optimize resource usage

## 🚦 Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Install dependencies
go mod download

# Build the application
go build -o tradsys ./cmd/main.go
```

### Configuration

Create a configuration file `config.yaml` in the root directory:

```yaml
environment: development
server:
  port: 8080
  readTimeout: 5s
  writeTimeout: 10s
  idleTimeout: 120s
database:
  driver: postgres
  host: localhost
  port: 5432
  username: postgres
  password: postgres
  database: tradsys
  sslMode: disable
  maxConns: 10
  maxIdle: 5
messaging:
  provider: nats
  url: nats://localhost:4222
  cluster_id: tradsys
  client_id: tradsys-server
  max_reconnects: 10
  reconnect_wait: 5s
```

### Running the Application

```bash
# Run the application
./tradsys --config config.yaml
```

## 📊 Decision Support System (DSS)

The Decision Support System provides advanced analytics and decision-making capabilities for trading strategies. It integrates with the core trading platform to provide real-time insights and recommendations.

### Integration Patterns

The DSS API supports multiple integration patterns:

1. **Synchronous Request-Response**: For immediate analysis and recommendations
2. **Asynchronous Processing**: For complex, time-consuming operations
3. **Event-Driven Integration**: For real-time updates based on market events
4. **Streaming Data**: Via WebSockets for continuous updates
5. **Batch Processing**: For large datasets and historical analysis

### Key API Endpoints

#### Analysis and Recommendations

- `POST /api/v1/dss/analyze`: Analyze market data and provide insights
- `POST /api/v1/dss/recommend`: Generate trading recommendations
- `POST /api/v1/dss/backtest`: Run backtesting on historical data
- `POST /api/v1/dss/scenario`: Perform scenario analysis
- `GET /api/v1/dss/portfolio/optimize`: Optimize portfolio allocation

#### Model Management

- `GET /api/v1/dss/models`: List available analysis models
- `POST /api/v1/dss/models`: Create a new analysis model
- `GET /api/v1/dss/models/{id}`: Get details of a specific model
- `PUT /api/v1/dss/models/{id}`: Update a model
- `DELETE /api/v1/dss/models/{id}`: Delete a model

#### Real-time Insights

- `GET /api/v1/dss/stream`: WebSocket endpoint for real-time insights
- `POST /api/v1/dss/alerts`: Configure real-time alerts
- `GET /api/v1/dss/alerts`: List configured alerts

## 🧩 Plugin System

TradSys features a powerful plugin system that allows extending the platform with custom components:

### Available Plugin Types

- **Strategy Plugins**: Implement custom trading strategies
- **Indicator Plugins**: Add technical indicators for market analysis
- **Risk Validator Plugins**: Create custom risk validation rules
- **Exchange Connector Plugins**: Connect to additional exchanges
- **Matching Algorithm Plugins**: Implement custom order matching algorithms

### Plugin Development

Plugins are implemented as Go plugins or as gRPC services. The platform provides interfaces and SDKs for developing plugins in various languages.

## 🔧 Development

### Testing

```bash
# Run all tests
go test ./...

# Run specific tests
go test ./internal/trading/...

# Run integration tests
go test -tags=integration ./...
```

### Code Style

We follow standard Go code style guidelines. Run the following before committing:

```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run
```

### API Documentation

API documentation is available using Swagger/OpenAPI:

```bash
# Generate API documentation
swag init -g cmd/main.go

# View documentation
# Open http://localhost:8080/swagger/index.html after starting the server
```

## 🔒 Security and Performance Enhancements

Recent updates have significantly improved the platform's security, reliability, and performance:

### Memory Management

- **Resource Manager**: Implemented a dedicated resource manager for automatic cleanup of resources
- **Memory Leak Prevention**: Fixed memory leaks in order heap operations and added periodic cleanup
- **Proper Resource Lifecycle**: Enhanced resource lifecycle management with context support

### Concurrency and Thread Safety

- **Lock Manager**: Implemented a sophisticated lock manager with deadlock detection
- **Atomic Operations**: Added atomic operations for thread-safe counters and statistics
- **Mutex Usage**: Improved mutex usage for proper synchronization of shared resources
- **Race Condition Prevention**: Fixed race conditions in critical components

### Performance Monitoring

- **Profiler**: Enhanced profiling capabilities with CPU, memory, block, and mutex profiling
- **Snapshot System**: Implemented concurrent snapshot creation with compression support
- **Message Batching**: Added priority-based message batching with compression

### Testing Infrastructure

- **Integration Tests**: Enhanced integration tests for core components
- **Unit Tests**: Added comprehensive unit tests for resource management, profiling, and message handling
- **Test Coverage**: Improved test coverage for critical components

## 🤝 Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add my feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## 📄 License

This project is licensed under the MIT License - see the LICENSE file for details.

## 📞 Contact

For questions or support, please contact [abdo.arh38@yahoo.com](mailto:abdo.arh38@yahoo.com)

