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
- **Decision Support Integration**: API for integrating with external decision support systems

## Requirements

- Go 1.20 or higher
- PostgreSQL 14 or higher
- Redis 7 or higher
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
messaging:
  provider: nats
  url: nats://localhost:4222
  cluster_id: tradsys
  client_id: tradsys-server
  max_reconnects: 10
  reconnect_wait: 5s
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
- **Decision Support Service**: Provides analytics and decision-making capabilities

## Decision Support System Integration

The Decision Support System (DSS) integration allows external systems to provide advanced analytics and decision-making capabilities for trading strategies. TradSys offers multiple integration patterns to accommodate different use cases.

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
  ```json
  // Request
  {
    "symbol": "BTC-USD",
    "timeframe": "1h",
    "indicators": ["rsi", "macd", "bollinger"],
    "start_time": "2023-01-01T00:00:00Z",
    "end_time": "2023-01-31T23:59:59Z"
  }
  
  // Response
  {
    "analysis_id": "an_12345",
    "symbol": "BTC-USD",
    "timeframe": "1h",
    "results": {
      "rsi": {
        "current": 65.75,
        "trend": "bullish",
        "signals": [
          {"time": "2023-01-15T14:00:00Z", "value": 30.2, "signal": "oversold"}
        ]
      },
      "macd": {
        "current": {"line": 0.0025, "signal": 0.0015, "histogram": 0.001},
        "trend": "bullish",
        "signals": [
          {"time": "2023-01-20T09:00:00Z", "type": "crossover", "direction": "bullish"}
        ]
      }
    }
  }
  ```

- `POST /api/v1/dss/recommend`: Generate trading recommendations
  ```json
  // Request
  {
    "symbol": "BTC-USD",
    "strategy": "momentum",
    "risk_profile": "moderate",
    "position_size": "auto"
  }
  
  // Response
  {
    "recommendation_id": "rec_67890",
    "symbol": "BTC-USD",
    "action": "buy",
    "confidence": 0.85,
    "price_target": 45000.00,
    "stop_loss": 42500.00,
    "time_horizon": "medium",
    "reasoning": [
      "RSI showing bullish divergence",
      "MACD crossover detected",
      "Volume increasing on recent price action"
    ]
  }
  ```

- `POST /api/v1/dss/backtest`: Run backtesting on historical data
- `POST /api/v1/dss/scenario`: Perform scenario analysis

#### Configuration Endpoints

- `GET /api/v1/dss/models`: List available analysis models
- `POST /api/v1/dss/models`: Create a new analysis model
  ```json
  // Request
  {
    "name": "Custom RSI Strategy",
    "description": "RSI-based strategy with custom parameters",
    "type": "technical",
    "parameters": {
      "rsi_period": 14,
      "overbought_threshold": 70,
      "oversold_threshold": 30,
      "signal_confirmation": true
    },
    "signals": {
      "buy": ["rsi_oversold", "price_above_ma"],
      "sell": ["rsi_overbought", "price_below_ma"]
    }
  }
  
  // Response
  {
    "model_id": "mdl_12345",
    "name": "Custom RSI Strategy",
    "created_at": "2023-06-15T10:30:00Z",
    "status": "active"
  }
  ```

- `GET /api/v1/dss/models/{id}`: Get details of a specific model
- `PUT /api/v1/dss/models/{id}`: Update a model
- `DELETE /api/v1/dss/models/{id}`: Delete a model

#### Real-time Endpoints

- `GET /api/v1/dss/stream`: WebSocket endpoint for real-time insights
  ```
  // Connection
  ws://api.tradsys.com/api/v1/dss/stream?token=<jwt_token>&symbols=BTC-USD,ETH-USD
  
  // Subscription message
  {
    "action": "subscribe",
    "channels": ["recommendations", "alerts", "market_insights"],
    "symbols": ["BTC-USD", "ETH-USD"]
  }
  
  // Sample message
  {
    "type": "recommendation",
    "timestamp": "2023-06-15T14:35:22.123Z",
    "symbol": "BTC-USD",
    "data": {
      "action": "buy",
      "confidence": 0.78,
      "price_target": 44500.00,
      "reasoning": "RSI oversold condition with increasing volume"
    }
  }
  ```

- `POST /api/v1/dss/alerts`: Configure real-time alerts
- `GET /api/v1/dss/alerts`: List configured alerts

### Authentication and Security

The DSS API uses OAuth 2.0 for authentication with JWT tokens. All endpoints require authentication except for public documentation endpoints.

```
Authorization: Bearer <jwt_token>
```

API keys with specific scopes can also be used for machine-to-machine integration:

```
X-API-Key: <api_key>
```

### Rate Limiting

API endpoints are rate-limited to ensure fair usage:

- 100 requests per minute for standard users
- 1000 requests per minute for premium users
- 5000 requests per minute for enterprise users

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
    },
    "request_id": "req_abcdef123456"
  }
}
```

### Webhooks

The DSS API supports webhooks for asynchronous notifications:

```json
// Webhook configuration
{
  "url": "https://your-system.com/webhooks/tradsys",
  "events": ["recommendation.new", "alert.triggered", "analysis.completed"],
  "secret": "your_webhook_secret"
}

// Sample webhook payload
{
  "event": "recommendation.new",
  "timestamp": "2023-06-15T14:35:22.123Z",
  "data": {
    "recommendation_id": "rec_67890",
    "symbol": "BTC-USD",
    "action": "buy",
    "confidence": 0.85
  },
  "signature": "sha256=..."
}
```

## Development

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
swag init -g cmd/tradsys/main.go

# View documentation
# Open http://localhost:8080/swagger/index.html after starting the server
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

