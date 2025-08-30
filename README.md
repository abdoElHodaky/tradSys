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


## Requirements

- Go 1.19 or higher
- PostgreSQL 13 or higher
- Redis 6 or higher

- NATS or Kafka for messaging

## Getting Started



### Prerequisites

- Go 1.19 or later
- PostgreSQL 14 or later
- Docker (optional, for containerized deployment)



### Installation

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys
 

# Install dependencies
go mod download

# Build the application
go build -o tradsys cmd/tradsys/main.go


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


## Decision Support System API


The Decision Support System (DSS) API provides advanced analytics and decision-making capabilities for trading strategies. It integrates with the core trading platform to provide real-time insights and recommendations.

### Integration Patterns

The DSS API supports multiple integration patterns:

## Development


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


- `POST /api/v1/dss/alerts`: Configure real-time alerts
- `GET /api/v1/dss/alerts`: List configured alerts


### Authentication and Security

The Decision Support System (DSS) API is designed to be flexible and extensible, allowing integration with various external systems. The API follows RESTful principles and uses JSON for data exchange, with additional support for gRPC and WebSocket protocols for high-performance use cases.


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

1. **Analysis Endpoint**
   - `POST /api/decision-support/analyze`
   - Submits market data, portfolio information, and other parameters for analysis
   - Returns analysis results including recommendations and insights
   - Supports both synchronous and asynchronous processing modes

2. **Recommendation Endpoint**
   - `GET /api/decision-support/recommendations`
   - Retrieves trading recommendations based on current market conditions
   - Supports filtering by instrument, strategy, and confidence level
   - Provides pagination and sorting options

3. **Scenario Analysis**
   - `POST /api/decision-support/scenarios`
   - Runs what-if scenarios with different market conditions
   - Returns potential outcomes and risk assessments
   - Supports batch processing of multiple scenarios

4. **Backtesting**
   - `POST /api/decision-support/backtest`
   - Tests strategies against historical data
   - Returns performance metrics and optimization suggestions
   - Supports long-running jobs with status tracking

5. **Portfolio Optimization**
   - `GET /api/decision-support/portfolio/optimize`
   - Provides portfolio optimization recommendations
   - Supports different optimization objectives (risk, return, Sharpe ratio)
   - Allows constraints specification (sector exposure, max position size)

6. **Alerts Configuration**
   - `POST /api/decision-support/alerts/configure`
   - Configures alerts based on market conditions or analysis results
   - Supports different notification channels (webhook, email, SMS)
   - Allows complex condition definitions using a rule engine

7. **Model Management**
   - `POST /api/decision-support/models`
   - Registers custom decision models for use in analysis
   - Supports model versioning and A/B testing
   - Provides model performance metrics

8. **Real-time Insights**
   - `GET /api/decision-support/insights/{symbol}`
   - Provides real-time market insights for specific symbols
   - Supports WebSocket connections for streaming updates
   - Includes sentiment analysis and news impact assessment


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

1. **Synchronous Request-Response**
   - Direct API calls for immediate analysis and recommendations
   - Suitable for user-initiated actions
   - Implements circuit breakers and timeouts for resilience

2. **Asynchronous Processing**
   - Submit analysis jobs that run in the background
   - Poll job status or receive webhook notifications when complete
   - Supports job cancellation and priority settings
   - Ideal for complex, time-consuming analysis

3. **Event-Driven Integration**
   - Subscribe to events and receive updates when conditions change
   - Supports WebHooks for push notifications
   - Implements the publish-subscribe pattern for real-time updates
   - Provides event filtering and transformation capabilities

4. **Streaming Data**
   - WebSocket API for continuous stream of recommendations and insights
   - Supports server-sent events (SSE) for one-way streaming
   - Provides connection management with automatic reconnection
   - Implements backpressure handling for high-volume data

5. **Batch Processing**
   - Bulk API endpoints for processing large datasets
   - Supports CSV and JSON data formats
   - Provides pagination and cursor-based result retrieval
   - Implements rate limiting and quota management

#### API Versioning and Compatibility

The DSS API implements versioning to ensure backward compatibility:

- API versions are specified in the URL path (e.g., `/api/v1/decision-support/analyze`)
- Changes to request/response formats are documented in the API changelog
- Deprecated endpoints are marked and maintained for a transition period
- New features are added in a backward-compatible manner when possible


We follow standard Go code style guidelines. Run the following before committing:


```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run
```

## Contributing

The DSS API uses OAuth 2.0 for authentication and supports role-based access control:

- JWT tokens for authentication with configurable expiration
- Fine-grained permissions for different API operations
- Rate limiting based on client identity
- All API requests are encrypted using TLS
- Audit logging for security monitoring

#### Performance Considerations

The API is designed for high performance and scalability:

- Horizontal scaling of API endpoints
- Response caching for frequently accessed data
- Compression for large payloads
- Connection pooling for database and external service connections
- Asynchronous processing for compute-intensive operations

#### Error Handling

The API implements consistent error handling:

- Standard error response format with error codes and messages
- Detailed error information for debugging (configurable)
- Validation errors with field-specific details
- Retry suggestions for transient errors
- Rate limit information in response headers

#### SDK and Client Libraries

To facilitate integration, TradSys provides:

- Official client SDKs for popular languages (Go, Python, JavaScript)
- OpenAPI/Swagger documentation for API exploration
- Code samples for common integration scenarios
- Postman collection for testing API endpoints


1. Fork the repository
2. Create a feature branch: `git checkout -b feature/my-feature`
3. Commit your changes: `git commit -am 'Add my feature'`
4. Push to the branch: `git push origin feature/my-feature`
5. Submit a pull request

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contact

For questions or support, please contact [abdo.arh38@yahoo.com](mailto:abdo.arh38@yahoo.com).

