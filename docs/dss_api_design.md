# Decision Support System API Design

This document outlines the comprehensive API design for integrating external Decision Support Systems (DSS) with the TradSys platform.

## Overview

The Decision Support System API enables third-party analytics and decision-making systems to integrate with TradSys. This API facilitates the exchange of market data, trading signals, recommendations, and execution capabilities between TradSys and external DSS platforms.

## Design Principles

1. **Consistency**: Follow RESTful design principles with consistent patterns
2. **Flexibility**: Support multiple integration patterns to accommodate different use cases
3. **Security**: Implement robust authentication and authorization
4. **Performance**: Optimize for low-latency trading operations
5. **Scalability**: Design for high throughput and horizontal scaling
6. **Observability**: Include comprehensive logging, metrics, and tracing

## API Versioning

All API endpoints are versioned to ensure backward compatibility:

```
/api/v1/dss/...
```

Future versions will be released as:

```
/api/v2/dss/...
```

## Authentication Methods

The API supports multiple authentication methods:

1. **OAuth 2.0 with JWT**:
   ```
   Authorization: Bearer <jwt_token>
   ```

2. **API Keys**:
   ```
   X-API-Key: <api_key>
   ```

3. **Client Certificates**: For high-security enterprise integrations

## Core API Resources

### 1. Market Data

#### Endpoints

- `GET /api/v1/dss/market-data/{symbol}`: Get latest market data for a symbol
- `GET /api/v1/dss/market-data/{symbol}/candles`: Get OHLCV candles for a symbol
- `GET /api/v1/dss/market-data/{symbol}/depth`: Get order book depth for a symbol
- `GET /api/v1/dss/market-data/{symbol}/trades`: Get recent trades for a symbol

#### WebSocket Stream

- `GET /api/v1/dss/stream/market-data`: WebSocket endpoint for real-time market data

#### Example Request/Response

```json
// GET /api/v1/dss/market-data/BTC-USD/candles?timeframe=1h&limit=10

// Response
{
  "symbol": "BTC-USD",
  "timeframe": "1h",
  "candles": [
    {
      "timestamp": "2023-06-15T14:00:00Z",
      "open": 43250.50,
      "high": 43500.00,
      "low": 43100.25,
      "close": 43350.75,
      "volume": 1250.5
    },
    // Additional candles...
  ]
}
```

### 2. Analysis

#### Endpoints

- `POST /api/v1/dss/analyze`: Analyze market data with specified indicators
- `GET /api/v1/dss/analyze/{analysis_id}`: Get results of a previous analysis
- `GET /api/v1/dss/analyze/indicators`: List available technical indicators
- `POST /api/v1/dss/analyze/custom`: Run custom analysis with provided algorithm

#### Example Request/Response

```json
// POST /api/v1/dss/analyze

// Request
{
  "symbol": "BTC-USD",
  "timeframe": "1h",
  "indicators": [
    {"name": "rsi", "parameters": {"period": 14}},
    {"name": "macd", "parameters": {"fast_period": 12, "slow_period": 26, "signal_period": 9}}
  ],
  "start_time": "2023-06-01T00:00:00Z",
  "end_time": "2023-06-15T23:59:59Z"
}

// Response
{
  "analysis_id": "an_12345",
  "status": "completed",
  "symbol": "BTC-USD",
  "timeframe": "1h",
  "results": {
    "rsi": {
      "data": [
        {"timestamp": "2023-06-15T14:00:00Z", "value": 65.75},
        // Additional data points...
      ],
      "summary": {
        "current": 65.75,
        "min": 30.2,
        "max": 75.8,
        "avg": 52.4
      }
    },
    "macd": {
      "data": [
        {
          "timestamp": "2023-06-15T14:00:00Z", 
          "line": 0.0025, 
          "signal": 0.0015, 
          "histogram": 0.001
        },
        // Additional data points...
      ],
      "signals": [
        {"timestamp": "2023-06-10T09:00:00Z", "type": "crossover", "direction": "bullish"},
        {"timestamp": "2023-06-05T16:00:00Z", "type": "crossover", "direction": "bearish"}
      ]
    }
  }
}
```

### 3. Recommendations

#### Endpoints

- `POST /api/v1/dss/recommend`: Generate trading recommendations
- `GET /api/v1/dss/recommend/{recommendation_id}`: Get a specific recommendation
- `GET /api/v1/dss/recommend/history`: Get historical recommendations
- `POST /api/v1/dss/recommend/execute`: Execute a recommendation as a trade

#### Example Request/Response

```json
// POST /api/v1/dss/recommend

// Request
{
  "symbol": "BTC-USD",
  "strategy": "momentum",
  "risk_profile": "moderate",
  "position_size": "auto",
  "context": {
    "portfolio_value": 100000,
    "current_exposure": 0.25,
    "max_exposure": 0.5
  }
}

// Response
{
  "recommendation_id": "rec_67890",
  "timestamp": "2023-06-15T14:35:22.123Z",
  "symbol": "BTC-USD",
  "action": "buy",
  "confidence": 0.85,
  "price_target": 45000.00,
  "stop_loss": 42500.00,
  "time_horizon": "medium",
  "position_size": {
    "type": "percentage",
    "value": 0.1,
    "estimated_usd": 10000
  },
  "reasoning": [
    "RSI showing bullish divergence",
    "MACD crossover detected",
    "Volume increasing on recent price action"
  ],
  "expiration": "2023-06-15T15:35:22.123Z"
}
```

### 4. Models

#### Endpoints

- `GET /api/v1/dss/models`: List available analysis models
- `POST /api/v1/dss/models`: Create a new analysis model
- `GET /api/v1/dss/models/{id}`: Get details of a specific model
- `PUT /api/v1/dss/models/{id}`: Update a model
- `DELETE /api/v1/dss/models/{id}`: Delete a model
- `POST /api/v1/dss/models/{id}/backtest`: Backtest a model

#### Example Request/Response

```json
// POST /api/v1/dss/models

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
  },
  "risk_management": {
    "stop_loss_percentage": 2.5,
    "take_profit_percentage": 5.0,
    "max_position_size": 0.1
  }
}

// Response
{
  "model_id": "mdl_12345",
  "name": "Custom RSI Strategy",
  "created_at": "2023-06-15T10:30:00Z",
  "status": "active",
  "version": 1
}
```

### 5. Backtesting

#### Endpoints

- `POST /api/v1/dss/backtest`: Run backtesting on historical data
- `GET /api/v1/dss/backtest/{backtest_id}`: Get results of a backtest
- `GET /api/v1/dss/backtest/{backtest_id}/trades`: Get trades from a backtest
- `GET /api/v1/dss/backtest/{backtest_id}/metrics`: Get performance metrics from a backtest

#### Example Request/Response

```json
// POST /api/v1/dss/backtest

// Request
{
  "model_id": "mdl_12345",
  "symbols": ["BTC-USD", "ETH-USD"],
  "timeframe": "1h",
  "start_time": "2022-01-01T00:00:00Z",
  "end_time": "2022-12-31T23:59:59Z",
  "initial_capital": 100000,
  "parameters": {
    "rsi_period": 14,
    "overbought_threshold": 70,
    "oversold_threshold": 30
  }
}

// Response
{
  "backtest_id": "bt_67890",
  "status": "processing",
  "estimated_completion_time": "2023-06-15T15:00:00Z",
  "progress": 0.0
}
```

### 6. Alerts

#### Endpoints

- `POST /api/v1/dss/alerts`: Configure real-time alerts
- `GET /api/v1/dss/alerts`: List configured alerts
- `GET /api/v1/dss/alerts/{alert_id}`: Get a specific alert
- `PUT /api/v1/dss/alerts/{alert_id}`: Update an alert
- `DELETE /api/v1/dss/alerts/{alert_id}`: Delete an alert
- `GET /api/v1/dss/alerts/history`: Get alert history

#### Example Request/Response

```json
// POST /api/v1/dss/alerts

// Request
{
  "name": "BTC RSI Alert",
  "description": "Alert when BTC RSI crosses below 30",
  "symbol": "BTC-USD",
  "conditions": [
    {
      "indicator": "rsi",
      "parameters": {"period": 14},
      "operator": "less_than",
      "value": 30
    }
  ],
  "notification_channels": ["email", "webhook"],
  "webhook_url": "https://your-system.com/webhooks/alerts",
  "cooldown_period": "1h"
}

// Response
{
  "alert_id": "alrt_12345",
  "name": "BTC RSI Alert",
  "status": "active",
  "created_at": "2023-06-15T10:30:00Z"
}
```

### 7. Webhooks

#### Endpoints

- `POST /api/v1/dss/webhooks`: Register a webhook
- `GET /api/v1/dss/webhooks`: List registered webhooks
- `GET /api/v1/dss/webhooks/{webhook_id}`: Get a specific webhook
- `PUT /api/v1/dss/webhooks/{webhook_id}`: Update a webhook
- `DELETE /api/v1/dss/webhooks/{webhook_id}`: Delete a webhook
- `POST /api/v1/dss/webhooks/{webhook_id}/test`: Test a webhook

#### Example Request/Response

```json
// POST /api/v1/dss/webhooks

// Request
{
  "url": "https://your-system.com/webhooks/tradsys",
  "description": "Main integration webhook",
  "events": [
    "recommendation.new",
    "alert.triggered",
    "analysis.completed",
    "trade.executed"
  ],
  "secret": "your_webhook_secret",
  "headers": {
    "X-Custom-Header": "custom-value"
  },
  "status": "active"
}

// Response
{
  "webhook_id": "wh_12345",
  "url": "https://your-system.com/webhooks/tradsys",
  "status": "active",
  "created_at": "2023-06-15T10:30:00Z"
}
```

## WebSocket API

The WebSocket API provides real-time data streams for various types of information:

### Connection

```
ws://api.tradsys.com/api/v1/dss/stream?token=<jwt_token>
```

### Subscription Message

```json
{
  "action": "subscribe",
  "channels": ["recommendations", "alerts", "market_insights", "market_data"],
  "symbols": ["BTC-USD", "ETH-USD"]
}
```

### Message Types

1. **Market Data Updates**:
   ```json
   {
     "type": "market_data",
     "timestamp": "2023-06-15T14:35:22.123Z",
     "symbol": "BTC-USD",
     "data": {
       "price": 43350.75,
       "volume_24h": 12500.5,
       "change_24h_percent": 2.5
     }
   }
   ```

2. **Recommendations**:
   ```json
   {
     "type": "recommendation",
     "timestamp": "2023-06-15T14:35:22.123Z",
     "symbol": "BTC-USD",
     "data": {
       "recommendation_id": "rec_67890",
       "action": "buy",
       "confidence": 0.85,
       "price_target": 45000.00,
       "reasoning": "RSI oversold condition with increasing volume"
     }
   }
   ```

3. **Alerts**:
   ```json
   {
     "type": "alert",
     "timestamp": "2023-06-15T14:35:22.123Z",
     "alert_id": "alrt_12345",
     "symbol": "BTC-USD",
     "data": {
       "condition": "rsi_below_30",
       "value": 28.5,
       "message": "BTC RSI is now in oversold territory"
     }
   }
   ```

4. **Market Insights**:
   ```json
   {
     "type": "market_insight",
     "timestamp": "2023-06-15T14:35:22.123Z",
     "symbols": ["BTC-USD", "global"],
     "data": {
       "insight_type": "volatility_alert",
       "message": "Increased market volatility detected",
       "details": {
         "volatility_index": 28.5,
         "change_percent": 15.3,
         "affected_markets": ["crypto", "forex"]
       }
     }
   }
   ```

## Error Handling

All API endpoints use standard HTTP status codes and return detailed error messages:

```json
{
  "error": {
    "code": "invalid_parameters",
    "message": "Invalid parameters provided",
    "details": {
      "field": "timeframe",
      "issue": "must be one of: 1m, 5m, 15m, 1h, 1d"
    },
    "request_id": "req_abcdef123456",
    "documentation_url": "https://docs.tradsys.com/api/errors#invalid_parameters"
  }
}
```

### Common Error Codes

- `invalid_parameters`: Request parameters are invalid
- `authentication_required`: Authentication is required
- `insufficient_permissions`: User lacks required permissions
- `resource_not_found`: Requested resource does not exist
- `rate_limit_exceeded`: API rate limit has been exceeded
- `internal_error`: Internal server error occurred

## Rate Limiting

API endpoints are rate-limited to ensure fair usage:

- 100 requests per minute for standard users
- 1000 requests per minute for premium users
- 5000 requests per minute for enterprise users

Rate limit information is included in response headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1623766522
```

## Pagination

List endpoints support pagination using cursor-based pagination:

```
GET /api/v1/dss/recommendations?limit=10&cursor=rec_67890
```

Response includes pagination metadata:

```json
{
  "data": [
    // Items...
  ],
  "pagination": {
    "next_cursor": "rec_67891",
    "has_more": true
  }
}
```

## Implementation Guidelines

### Service Architecture

The DSS API should be implemented as a dedicated microservice within the TradSys architecture:

1. **API Gateway**: Routes requests to the DSS service
2. **DSS Service**: Handles DSS-specific logic and integrations
3. **Authentication Service**: Validates tokens and permissions
4. **Market Data Service**: Provides market data to the DSS service
5. **Trading Service**: Executes trades based on DSS recommendations

### Data Flow

1. **Inbound Data Flow**:
   - External DSS systems send recommendations and analysis
   - TradSys validates and processes these inputs
   - Recommendations can be automatically executed or presented to users

2. **Outbound Data Flow**:
   - TradSys sends market data and events to external DSS systems
   - Real-time updates via WebSockets
   - Batch data via REST endpoints

### Security Considerations

1. **Authentication**: Implement OAuth 2.0 with JWT tokens
2. **Authorization**: Define granular permissions for different API operations
3. **Rate Limiting**: Prevent abuse with tiered rate limits
4. **Input Validation**: Validate all input parameters
5. **Encryption**: Use TLS for all communications
6. **Audit Logging**: Log all API access and changes

### Performance Optimization

1. **Caching**: Implement Redis caching for frequently accessed data
2. **Connection Pooling**: Use connection pools for database access
3. **Asynchronous Processing**: Use message queues for time-consuming operations
4. **Horizontal Scaling**: Design services to scale horizontally
5. **Data Pagination**: Implement cursor-based pagination for large datasets

## SDK Support

To facilitate integration, TradSys should provide client SDKs in multiple languages:

1. **Go SDK**: Native SDK for Go applications
2. **Python SDK**: For data science and ML-focused DSS systems
3. **JavaScript/TypeScript SDK**: For web-based trading platforms
4. **Java SDK**: For enterprise DSS systems

## Documentation

Comprehensive documentation should be provided:

1. **API Reference**: OpenAPI/Swagger documentation
2. **Integration Guides**: Step-by-step integration tutorials
3. **Code Examples**: Sample code in multiple languages
4. **Webhooks Guide**: Detailed webhook implementation guide
5. **Authentication Guide**: Security implementation details

## Conclusion

This API design provides a comprehensive framework for integrating external Decision Support Systems with the TradSys platform. By following these guidelines, TradSys can offer a flexible, secure, and high-performance integration experience for DSS providers and consumers.

