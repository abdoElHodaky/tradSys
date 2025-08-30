# TradSys - Trading System Platform

TradSys is a comprehensive trading system platform designed for high-performance, scalable trading operations. It provides a robust foundation for building trading applications with support for various asset classes, market data integration, and advanced trading strategies.

## Features

- **Event-Driven Architecture**: Built on a modern event-driven architecture for real-time processing
- **Microservices Design**: Modular components that can be deployed independently
- **High Performance**: Optimized for low-latency trading operations
- **Scalability**: Horizontal scaling capabilities for handling increased load
- **Resilience**: Circuit breakers, rate limiting, and retry mechanisms for robust operations
- **Extensibility**: Plugin system for custom strategies and integrations
- **Comprehensive Monitoring**: Built-in metrics and logging for operational visibility

## Requirements

- Go 1.19 or higher
- PostgreSQL 13 or higher
- Redis 6 or higher
- NATS or Kafka for messaging

## Getting Started

### Installation

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Install dependencies
go mod download

# Build the application
go build -o tradsys cmd/tradsys/main.go
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
```

### Running

```bash
# Run the application
./tradsys --config config.yaml
```

## Architecture

TradSys is built on a microservices architecture with the following components:

- **API Gateway**: Entry point for external requests
- **Order Service**: Handles order management and execution
- **Market Data Service**: Processes and distributes market data
- **Strategy Service**: Executes trading strategies
- **Risk Management Service**: Enforces risk controls
- **Position Service**: Tracks positions and exposures
- **Authentication Service**: Manages user authentication and authorization

## Decision Support System API

The Decision Support System (DSS) API provides advanced analytics and decision-making capabilities for trading strategies. It integrates with the core trading platform to provide real-time insights and recommendations.

### Integration Patterns

The DSS API supports multiple integration patterns:

1. **Synchronous Request-Response**: For immediate analysis and recommendations
2. **Asynchronous Processing**: For complex, time-consuming operations
3. **Event-Driven Integration**: For real-time updates based on market events
4. **Streaming Data**: Via WebSockets for continuous updates
5. **Batch Processing**: For large datasets and historical analysis

### API Endpoints

#### Analysis Endpoints

- `POST /api/v1/dss/analyze`: Analyze market data and provide insights
- `POST /api/v1/dss/recommend`: Generate trading recommendations
- `POST /api/v1/dss/backtest`: Run backtesting on historical data
- `POST /api/v1/dss/scenario`: Perform scenario analysis

#### Configuration Endpoints

- `GET /api/v1/dss/models`: List available analysis models
- `POST /api/v1/dss/models`: Create a new analysis model
- `GET /api/v1/dss/models/{id}`: Get details of a specific model
- `PUT /api/v1/dss/models/{id}`: Update a model
- `DELETE /api/v1/dss/models/{id}`: Delete a model

#### Real-time Endpoints

- `GET /api/v1/dss/stream`: WebSocket endpoint for real-time insights
- `POST /api/v1/dss/alerts`: Configure real-time alerts
- `GET /api/v1/dss/alerts`: List configured alerts

### Authentication and Security

The DSS API uses OAuth 2.0 for authentication with JWT tokens. All endpoints require authentication except for public documentation endpoints.

```
Authorization: Bearer <jwt_token>
```

### Rate Limiting

API endpoints are rate-limited to ensure fair usage:

- 100 requests per minute for standard users
- 1000 requests per minute for premium users

### Error Handling

The API uses standard HTTP status codes and returns detailed error messages:

```json
{
  "error": {
    "code": "invalid_parameters",
    "message": "Invalid parameters provided",
    "details": {
      "field": "timeframe",
      "issue": "must be one of: 1m, 5m, 15m, 1h, 1d"
    }
  }
}
```

## Development

### Testing

```bash
# Run all tests
go test ./...

# Run specific tests
go test ./internal/trading/...
```

### Code Style

We follow standard Go code style guidelines. Run the following before committing:

```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run
```

## Contributing

1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add my feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contact

For questions or support, please contact [abdo.arh38@yahoo.com](mailto:abdo.arh38@yahoo.com).

