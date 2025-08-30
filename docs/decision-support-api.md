# Decision Support System API Documentation

## Overview

The Decision Support System (DSS) API provides advanced analytics and decision-making capabilities for trading strategies. It integrates with the TradSys platform to deliver real-time insights, recommendations, and risk assessments.

## Integration Patterns

The DSS API supports multiple integration patterns to accommodate different use cases:

### 1. Synchronous Request-Response

For immediate analysis and recommendations:

```
Client ---> HTTP Request ---> DSS API ---> Response ---> Client
```

Best for:
- Simple, quick analyses
- User-initiated requests
- Interactive dashboards

### 2. Asynchronous Processing

For complex, time-consuming operations:

```
Client ---> HTTP Request ---> DSS API ---> Task ID ---> Client
                                  |
                                  v
                              Processing
                                  |
                                  v
Client <--- Webhook/Poll <--- Results
```

Best for:
- Complex calculations
- Batch processing
- Resource-intensive operations

### 3. Event-Driven Integration

For real-time updates based on market events:

```
Market Event ---> Event Bus ---> DSS API ---> Analysis ---> Event Bus ---> Subscribers
```

Best for:
- Real-time market reactions
- Automated trading systems
- Continuous monitoring

### 4. Streaming Data

Via WebSockets for continuous updates:

```
Client <---> WebSocket Connection <---> DSS API
```

Best for:
- Real-time dashboards
- Continuous data visualization
- Live trading signals

### 5. Batch Processing

For large datasets and historical analysis:

```
Client ---> Upload Dataset ---> DSS API ---> Process Batch ---> Results ---> Client
```

Best for:
- Historical data analysis
- Large dataset processing
- Overnight processing jobs

## Authentication

The DSS API uses OAuth 2.0 for authentication with JWT tokens.

### Obtaining a Token

```
POST /api/v1/auth/token
```

Request:
```json
{
  "client_id": "your_client_id",
  "client_secret": "your_client_secret",
  "grant_type": "client_credentials"
}
```

Response:
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "Bearer",
  "expires_in": 3600
}
```

### Using the Token

Include the token in the Authorization header:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## API Endpoints

### Analysis Endpoints

#### Analyze Market Data

```
POST /api/v1/dss/analyze
```

Request:
```json
{
  "instrument": "AAPL",
  "timeframe": "1h",
  "start_time": "2023-01-01T00:00:00Z",
  "end_time": "2023-01-31T23:59:59Z",
  "indicators": ["sma", "rsi", "macd"],
  "parameters": {
    "sma": {"period": 20},
    "rsi": {"period": 14},
    "macd": {"fast_period": 12, "slow_period": 26, "signal_period": 9}
  }
}
```

Response:
```json
{
  "instrument": "AAPL",
  "timeframe": "1h",
  "analysis_id": "a1b2c3d4",
  "timestamp": "2023-02-01T10:15:30Z",
  "results": {
    "sma": [{"timestamp": "2023-01-01T01:00:00Z", "value": 142.35}, ...],
    "rsi": [{"timestamp": "2023-01-01T01:00:00Z", "value": 65.42}, ...],
    "macd": [{"timestamp": "2023-01-01T01:00:00Z", "macd": 0.35, "signal": 0.28, "histogram": 0.07}, ...]
  },
  "insights": [
    {"type": "trend", "message": "Upward trend detected", "confidence": 0.85},
    {"type": "reversal", "message": "Potential reversal point at 2023-01-15", "confidence": 0.65}
  ]
}
```

#### Generate Trading Recommendations

```
POST /api/v1/dss/recommend
```

Request:
```json
{
  "portfolio": [
    {"instrument": "AAPL", "quantity": 100, "entry_price": 145.75},
    {"instrument": "MSFT", "quantity": 50, "entry_price": 235.50}
  ],
  "risk_profile": "moderate",
  "time_horizon": "medium",
  "constraints": {
    "max_positions": 10,
    "max_allocation_per_position": 0.15,
    "sectors_to_avoid": ["energy"]
  }
}
```

Response:
```json
{
  "recommendation_id": "r5t6y7u8",
  "timestamp": "2023-02-01T10:20:45Z",
  "recommendations": [
    {
      "action": "buy",
      "instrument": "GOOGL",
      "quantity": 10,
      "price_range": {"min": 2150.00, "max": 2200.00},
      "rationale": "Diversification into tech sector with strong fundamentals",
      "confidence": 0.82
    },
    {
      "action": "sell",
      "instrument": "AAPL",
      "quantity": 25,
      "price_range": {"min": 150.00, "max": 155.00},
      "rationale": "Take partial profits after recent rally",
      "confidence": 0.75
    }
  ],
  "portfolio_impact": {
    "expected_return": 0.085,
    "risk_change": -0.02,
    "diversification_score": 0.72
  }
}
```

#### Run Backtesting

```
POST /api/v1/dss/backtest
```

Request:
```json
{
  "strategy": {
    "name": "Moving Average Crossover",
    "parameters": {
      "fast_period": 10,
      "slow_period": 50,
      "position_size": 0.1
    }
  },
  "instruments": ["AAPL", "MSFT", "AMZN"],
  "timeframe": "1d",
  "start_date": "2022-01-01",
  "end_date": "2022-12-31",
  "initial_capital": 100000,
  "commission": 0.001
}
```

Response:
```json
{
  "backtest_id": "b9i8u7y6",
  "status": "completed",
  "summary": {
    "total_return": 0.156,
    "annualized_return": 0.156,
    "max_drawdown": 0.12,
    "sharpe_ratio": 1.35,
    "sortino_ratio": 1.85,
    "win_rate": 0.65,
    "profit_factor": 1.75
  },
  "trades": [
    {
      "instrument": "AAPL",
      "entry_date": "2022-01-15",
      "entry_price": 172.50,
      "exit_date": "2022-02-10",
      "exit_price": 185.25,
      "quantity": 58,
      "pnl": 739.50,
      "return": 0.074
    },
    // More trades...
  ],
  "equity_curve": [
    {"date": "2022-01-01", "equity": 100000},
    {"date": "2022-01-02", "equity": 100120},
    // More points...
  ]
}
```

#### Perform Scenario Analysis

```
POST /api/v1/dss/scenario
```

Request:
```json
{
  "portfolio": [
    {"instrument": "AAPL", "quantity": 100, "entry_price": 145.75},
    {"instrument": "MSFT", "quantity": 50, "entry_price": 235.50},
    {"instrument": "AMZN", "quantity": 20, "entry_price": 3250.00}
  ],
  "scenarios": [
    {
      "name": "Market Crash",
      "description": "Sudden market downturn of 20%",
      "market_changes": [
        {"sector": "technology", "change": -0.25},
        {"sector": "financials", "change": -0.20},
        {"sector": "healthcare", "change": -0.15}
      ]
    },
    {
      "name": "Interest Rate Hike",
      "description": "Fed raises rates by 75 basis points",
      "market_changes": [
        {"sector": "technology", "change": -0.10},
        {"sector": "financials", "change": 0.05},
        {"sector": "utilities", "change": -0.08}
      ]
    }
  ]
}
```

Response:
```json
{
  "scenario_analysis_id": "s2d3f4g5",
  "timestamp": "2023-02-01T11:30:15Z",
  "portfolio_value": 123500.00,
  "results": [
    {
      "scenario": "Market Crash",
      "portfolio_impact": {
        "value_after": 95095.00,
        "change": -0.23,
        "var_95": 32500.00
      },
      "instrument_impacts": [
        {"instrument": "AAPL", "price_change": -0.25, "value_change": -3643.75},
        {"instrument": "MSFT", "price_change": -0.25, "value_change": -2943.75},
        {"instrument": "AMZN", "price_change": -0.25, "value_change": -16250.00}
      ]
    },
    {
      "scenario": "Interest Rate Hike",
      "portfolio_impact": {
        "value_after": 114220.00,
        "change": -0.075,
        "var_95": 12500.00
      },
      "instrument_impacts": [
        {"instrument": "AAPL", "price_change": -0.10, "value_change": -1457.50},
        {"instrument": "MSFT", "price_change": -0.10, "value_change": -1177.50},
        {"instrument": "AMZN", "price_change": -0.10, "value_change": -6500.00}
      ]
    }
  ],
  "recommendations": [
    {
      "type": "hedge",
      "description": "Consider purchasing put options on QQQ to hedge technology exposure",
      "estimated_cost": 2500.00,
      "protection_level": 0.60
    },
    {
      "type": "diversification",
      "description": "Increase allocation to defensive sectors like consumer staples",
      "target_allocation": 0.15
    }
  ]
}
```

### Configuration Endpoints

#### List Available Analysis Models

```
GET /api/v1/dss/models
```

Response:
```json
{
  "models": [
    {
      "id": "m1n2b3v4",
      "name": "Technical Analysis Suite",
      "description": "Comprehensive technical analysis with multiple indicators",
      "category": "technical",
      "version": "2.1.0",
      "parameters": [
        {"name": "timeframe", "type": "string", "required": true, "default": "1d", "options": ["1m", "5m", "15m", "1h", "4h", "1d"]},
        {"name": "indicators", "type": "array", "required": false, "default": ["sma", "ema", "rsi"]}
      ]
    },
    {
      "id": "m5n6b7v8",
      "name": "Fundamental Analysis Model",
      "description": "Analysis based on company fundamentals and financial metrics",
      "category": "fundamental",
      "version": "1.5.0",
      "parameters": [
        {"name": "metrics", "type": "array", "required": false, "default": ["pe_ratio", "eps_growth", "debt_to_equity"]},
        {"name": "comparison_period", "type": "string", "required": false, "default": "1y", "options": ["1q", "1y", "3y", "5y"]}
      ]
    }
  ],
  "total": 2,
  "page": 1,
  "page_size": 10
}
```

#### Create a New Analysis Model

```
POST /api/v1/dss/models
```

Request:
```json
{
  "name": "Custom Momentum Strategy",
  "description": "Momentum-based strategy with volatility adjustment",
  "category": "momentum",
  "parameters": [
    {"name": "lookback_period", "type": "integer", "required": true, "default": 20, "min": 5, "max": 100},
    {"name": "volatility_window", "type": "integer", "required": false, "default": 10, "min": 5, "max": 30},
    {"name": "threshold", "type": "number", "required": false, "default": 0.05, "min": 0.01, "max": 0.2}
  ],
  "code": "function analyze(data, params) { /* JavaScript code */ }",
  "is_public": false
}
```

Response:
```json
{
  "id": "m9n8b7v6",
  "name": "Custom Momentum Strategy",
  "description": "Momentum-based strategy with volatility adjustment",
  "category": "momentum",
  "version": "1.0.0",
  "created_at": "2023-02-01T12:45:30Z",
  "parameters": [
    {"name": "lookback_period", "type": "integer", "required": true, "default": 20, "min": 5, "max": 100},
    {"name": "volatility_window", "type": "integer", "required": false, "default": 10, "min": 5, "max": 30},
    {"name": "threshold", "type": "number", "required": false, "default": 0.05, "min": 0.01, "max": 0.2}
  ],
  "is_public": false
}
```

### Real-time Endpoints

#### WebSocket Connection

```
GET /api/v1/dss/stream
```

Connection Parameters:
- `token`: JWT authentication token
- `instruments`: Comma-separated list of instruments to monitor
- `events`: Comma-separated list of event types to subscribe to

Example:
```
wss://api.tradsys.com/api/v1/dss/stream?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...&instruments=AAPL,MSFT,AMZN&events=signal,alert,insight
```

Message Types:

1. Signal:
```json
{
  "type": "signal",
  "timestamp": "2023-02-01T13:15:45.123Z",
  "instrument": "AAPL",
  "signal": "buy",
  "strength": 0.85,
  "indicators": {
    "rsi": 32.5,
    "macd_histogram": 0.35
  },
  "price": 145.75
}
```

2. Alert:
```json
{
  "type": "alert",
  "timestamp": "2023-02-01T13:20:10.456Z",
  "alert_id": "a1s2d3f4",
  "instrument": "MSFT",
  "condition": "price_above",
  "threshold": 250.00,
  "current_value": 252.75,
  "message": "MSFT price crossed above $250.00"
}
```

3. Insight:
```json
{
  "type": "insight",
  "timestamp": "2023-02-01T13:25:30.789Z",
  "category": "pattern",
  "instrument": "AMZN",
  "message": "Potential double bottom pattern forming",
  "confidence": 0.72,
  "timeframe": "4h"
}
```

#### Configure Real-time Alerts

```
POST /api/v1/dss/alerts
```

Request:
```json
{
  "name": "AAPL Price Alert",
  "instrument": "AAPL",
  "condition": {
    "type": "price_crossing",
    "direction": "above",
    "value": 150.00
  },
  "actions": [
    {
      "type": "notification",
      "channels": ["email", "webhook"],
      "message": "AAPL crossed above $150.00"
    }
  ],
  "expiration": "2023-03-01T00:00:00Z"
}
```

Response:
```json
{
  "alert_id": "a5s6d7f8",
  "name": "AAPL Price Alert",
  "status": "active",
  "created_at": "2023-02-01T14:10:25Z",
  "instrument": "AAPL",
  "condition": {
    "type": "price_crossing",
    "direction": "above",
    "value": 150.00
  },
  "actions": [
    {
      "type": "notification",
      "channels": ["email", "webhook"],
      "message": "AAPL crossed above $150.00"
    }
  ],
  "expiration": "2023-03-01T00:00:00Z"
}
```

## Error Handling

The API uses standard HTTP status codes and returns detailed error messages:

### Error Response Format

```json
{
  "error": {
    "code": "error_code",
    "message": "Human-readable error message",
    "details": {
      "field": "specific_field",
      "issue": "description of the issue"
    },
    "request_id": "req_123456789"
  }
}
```

### Common Error Codes

- `invalid_parameters`: One or more parameters are invalid
- `authentication_failed`: Authentication failed
- `authorization_failed`: User is not authorized to perform the action
- `resource_not_found`: The requested resource was not found
- `rate_limit_exceeded`: API rate limit has been exceeded
- `internal_error`: An internal server error occurred

### HTTP Status Codes

- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication failed
- `403 Forbidden`: Authorization failed
- `404 Not Found`: Resource not found
- `429 Too Many Requests`: Rate limit exceeded
- `500 Internal Server Error`: Server error

## Rate Limiting

API endpoints are rate-limited to ensure fair usage:

- 100 requests per minute for standard users
- 1000 requests per minute for premium users

Rate limit headers are included in all responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1612345678
```

## Webhooks

The DSS API can send webhooks for various events:

### Configuring Webhooks

```
POST /api/v1/dss/webhooks
```

Request:
```json
{
  "url": "https://your-server.com/webhook",
  "events": ["analysis.completed", "recommendation.generated", "alert.triggered"],
  "secret": "your_webhook_secret"
}
```

Response:
```json
{
  "webhook_id": "w1e2r3t4",
  "url": "https://your-server.com/webhook",
  "events": ["analysis.completed", "recommendation.generated", "alert.triggered"],
  "created_at": "2023-02-01T15:30:45Z",
  "status": "active"
}
```

### Webhook Payload

```json
{
  "event": "analysis.completed",
  "timestamp": "2023-02-01T15:35:10Z",
  "webhook_id": "w1e2r3t4",
  "data": {
    "analysis_id": "a1b2c3d4",
    "status": "completed",
    "instrument": "AAPL",
    "result_url": "https://api.tradsys.com/api/v1/dss/analysis/a1b2c3d4"
  }
}
```

### Webhook Security

Webhooks include a signature header for verification:

```
X-DSS-Signature: sha256=5257a869e7bdf3ecbd7687cf1bfcc2e5a9cdca6dfe8b081cbfa54f5c2a0e2002
```

To verify the signature:

1. Get the raw request body
2. Create an HMAC-SHA256 hash using your webhook secret
3. Compare the hash with the signature in the header

## SDK Examples

### JavaScript/TypeScript SDK

```typescript
import { DSSClient } from 'tradsys-dss-sdk';

// Initialize client
const client = new DSSClient({
  apiKey: 'your_api_key',
  apiSecret: 'your_api_secret',
  baseUrl: 'https://api.tradsys.com'
});

// Analyze market data
async function analyzeMarket() {
  try {
    const result = await client.analyze({
      instrument: 'AAPL',
      timeframe: '1h',
      start_time: '2023-01-01T00:00:00Z',
      end_time: '2023-01-31T23:59:59Z',
      indicators: ['sma', 'rsi', 'macd'],
      parameters: {
        sma: { period: 20 },
        rsi: { period: 14 },
        macd: { fast_period: 12, slow_period: 26, signal_period: 9 }
      }
    });
    
    console.log('Analysis results:', result);
  } catch (error) {
    console.error('Analysis failed:', error);
  }
}

// Connect to WebSocket stream
function connectToStream() {
  const stream = client.connectStream({
    instruments: ['AAPL', 'MSFT', 'AMZN'],
    events: ['signal', 'alert', 'insight']
  });
  
  stream.on('signal', (signal) => {
    console.log('Received signal:', signal);
  });
  
  stream.on('alert', (alert) => {
    console.log('Received alert:', alert);
  });
  
  stream.on('insight', (insight) => {
    console.log('Received insight:', insight);
  });
  
  stream.on('error', (error) => {
    console.error('Stream error:', error);
  });
  
  return stream;
}
```

### Python SDK

```python
from tradsys_dss_sdk import DSSClient

# Initialize client
client = DSSClient(
    api_key='your_api_key',
    api_secret='your_api_secret',
    base_url='https://api.tradsys.com'
)

# Run backtesting
def run_backtest():
    try:
        result = client.backtest(
            strategy={
                'name': 'Moving Average Crossover',
                'parameters': {
                    'fast_period': 10,
                    'slow_period': 50,
                    'position_size': 0.1
                }
            },
            instruments=['AAPL', 'MSFT', 'AMZN'],
            timeframe='1d',
            start_date='2022-01-01',
            end_date='2022-12-31',
            initial_capital=100000,
            commission=0.001
        )
        
        print(f"Backtest results: {result['summary']}")
        print(f"Total return: {result['summary']['total_return']:.2%}")
        print(f"Sharpe ratio: {result['summary']['sharpe_ratio']:.2f}")
        
        return result
    except Exception as e:
        print(f"Backtest failed: {e}")
        return None

# Configure an alert
def create_alert():
    try:
        alert = client.create_alert(
            name='AAPL Price Alert',
            instrument='AAPL',
            condition={
                'type': 'price_crossing',
                'direction': 'above',
                'value': 150.00
            },
            actions=[
                {
                    'type': 'notification',
                    'channels': ['email', 'webhook'],
                    'message': 'AAPL crossed above $150.00'
                }
            ],
            expiration='2023-03-01T00:00:00Z'
        )
        
        print(f"Alert created: {alert['alert_id']}")
        return alert
    except Exception as e:
        print(f"Alert creation failed: {e}")
        return None
```

## Versioning

The DSS API uses semantic versioning (MAJOR.MINOR.PATCH) and is currently at version 1.0.0.

API endpoints are versioned in the URL path (e.g., `/api/v1/dss/analyze`).

## Support

For questions or support, please contact [abdo.arh38@yahoo.com](mailto:abdo.arh38@yahoo.com) or visit our [developer portal](https://developer.tradsys.com).

