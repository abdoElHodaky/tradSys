# Decision Support System Integration API

This document outlines the API design for integrating TradSys with external decision support systems.

## Overview

The Decision Support API provides a comprehensive interface for connecting TradSys with external decision support systems. It enables bidirectional data flow, allowing trading systems to send market data and receive trading recommendations, risk assessments, and portfolio optimization suggestions.

## Authentication

All API endpoints require JWT authentication. Tokens can be obtained through the standard authentication endpoints.

```
Authorization: Bearer <jwt_token>
```

## API Endpoints

### Data Analysis

#### Submit Data for Analysis

```
POST /api/decision-support/analyze
```

Submit trading data for analysis by the decision support system.

**Request Body:**
```json
{
  "data_type": "market_data|order_flow|portfolio|custom",
  "time_range": {
    "start": "2023-01-01T00:00:00Z",
    "end": "2023-01-31T23:59:59Z"
  },
  "symbols": ["AAPL", "MSFT", "GOOGL"],
  "additional_parameters": {
    "key1": "value1",
    "key2": "value2"
  },
  "analysis_type": "technical|fundamental|sentiment|ml_prediction"
}
```

**Response:**
```json
{
  "analysis_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "processing|completed|failed",
  "estimated_completion_time": "2023-02-01T01:30:00Z",
  "result_endpoint": "/api/decision-support/analysis/a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

#### Get Analysis Results

```
GET /api/decision-support/analysis/{analysis_id}
```

Retrieve the results of a previously submitted analysis request.

**Response:**
```json
{
  "analysis_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
  "status": "completed",
  "created_at": "2023-02-01T01:00:00Z",
  "completed_at": "2023-02-01T01:15:00Z",
  "results": {
    "summary": "Market shows bullish trend for technology sector",
    "details": {
      "technical_indicators": { ... },
      "sentiment_analysis": { ... },
      "predictions": { ... }
    },
    "confidence_score": 0.85
  }
}
```

### Trading Recommendations

#### Get Trading Recommendations

```
GET /api/decision-support/recommendations
```

Get trading recommendations based on current market conditions and portfolio.

**Query Parameters:**
- `symbols` (optional): Comma-separated list of symbols
- `strategy` (optional): Strategy type (e.g., "conservative", "aggressive")
- `time_horizon` (optional): Time horizon for recommendations (e.g., "short_term", "long_term")

**Response:**
```json
{
  "recommendations": [
    {
      "symbol": "AAPL",
      "action": "buy|sell|hold",
      "price_target": 150.00,
      "confidence": 0.78,
      "time_horizon": "short_term",
      "reasoning": "Strong earnings report and positive technical indicators",
      "risk_assessment": "low|medium|high"
    },
    {
      "symbol": "MSFT",
      "action": "hold",
      "confidence": 0.65,
      "time_horizon": "medium_term",
      "reasoning": "Current price aligned with fair value estimate",
      "risk_assessment": "low"
    }
  ],
  "generated_at": "2023-02-01T02:30:00Z",
  "valid_until": "2023-02-01T14:30:00Z"
}
```

### Scenario Analysis

#### Get Scenario Analysis

```
GET /api/decision-support/scenarios
```

Get scenario analysis for different market conditions.

**Query Parameters:**
- `portfolio_id` (required): ID of the portfolio to analyze
- `scenario_type` (optional): Type of scenario (e.g., "market_crash", "interest_rate_hike", "custom")

**Response:**
```json
{
  "scenarios": [
    {
      "name": "Market Correction (-10%)",
      "probability": 0.25,
      "impact": {
        "portfolio_value_change": -7.5,
        "var_95": -12.3,
        "max_drawdown": 15.2
      },
      "recommended_actions": [
        {
          "action": "increase_hedge",
          "details": "Increase hedging positions by 5%"
        },
        {
          "action": "rebalance",
          "details": "Reduce technology exposure by 3%"
        }
      ]
    },
    {
      "name": "Strong Bull Market (+15%)",
      "probability": 0.35,
      "impact": {
        "portfolio_value_change": 12.8,
        "var_95": -5.1,
        "max_drawdown": 7.5
      },
      "recommended_actions": [
        {
          "action": "increase_leverage",
          "details": "Increase position sizes in high-beta stocks"
        }
      ]
    }
  ],
  "analysis_date": "2023-02-01T03:00:00Z"
}
```

### Backtesting

#### Run Backtest

```
POST /api/decision-support/backtest
```

Run a backtest with specified parameters.

**Request Body:**
```json
{
  "strategy": {
    "name": "Moving Average Crossover",
    "parameters": {
      "short_period": 10,
      "long_period": 50,
      "position_sizing": "fixed|percent|kelly"
    }
  },
  "symbols": ["AAPL", "MSFT", "GOOGL"],
  "time_range": {
    "start": "2022-01-01T00:00:00Z",
    "end": "2022-12-31T23:59:59Z"
  },
  "initial_capital": 100000,
  "commission_model": {
    "type": "fixed|percentage",
    "value": 0.001
  }
}
```

**Response:**
```json
{
  "backtest_id": "b1c2d3e4-f5g6-7890-abcd-ef1234567890",
  "status": "processing|completed|failed",
  "estimated_completion_time": "2023-02-01T04:00:00Z",
  "result_endpoint": "/api/decision-support/backtest/b1c2d3e4-f5g6-7890-abcd-ef1234567890"
}
```

#### Get Backtest Results

```
GET /api/decision-support/backtest/{backtest_id}
```

Get the results of a previously run backtest.

**Response:**
```json
{
  "backtest_id": "b1c2d3e4-f5g6-7890-abcd-ef1234567890",
  "status": "completed",
  "strategy": {
    "name": "Moving Average Crossover",
    "parameters": {
      "short_period": 10,
      "long_period": 50
    }
  },
  "performance": {
    "total_return": 15.7,
    "annualized_return": 12.3,
    "sharpe_ratio": 1.2,
    "max_drawdown": 8.5,
    "win_rate": 0.65,
    "profit_factor": 1.8
  },
  "trades": [
    {
      "symbol": "AAPL",
      "entry_date": "2022-02-15T10:30:00Z",
      "entry_price": 145.50,
      "exit_date": "2022-03-10T15:45:00Z",
      "exit_price": 158.75,
      "profit_loss": 9.11,
      "profit_loss_percent": 9.11
    },
    // More trades...
  ],
  "equity_curve": [
    {
      "date": "2022-01-01T00:00:00Z",
      "equity": 100000
    },
    // More equity points...
  ]
}
```

### Market Insights

#### Get Market Insights

```
GET /api/decision-support/insights/{symbol}
```

Get detailed market insights for a specific symbol.

**Query Parameters:**
- `time_range` (optional): Time range for analysis (e.g., "1d", "1w", "1m", "3m", "1y")
- `insight_types` (optional): Types of insights to include (e.g., "technical,fundamental,sentiment")

**Response:**
```json
{
  "symbol": "AAPL",
  "last_price": 145.86,
  "change_percent": 1.25,
  "insights": {
    "technical": {
      "trend": "bullish|bearish|neutral",
      "support_levels": [140.50, 138.20],
      "resistance_levels": [148.75, 152.00],
      "indicators": {
        "rsi": 58.5,
        "macd": {
          "value": 2.35,
          "signal": 1.85,
          "histogram": 0.50
        },
        "moving_averages": {
          "sma_50": 142.30,
          "sma_200": 135.75,
          "ema_20": 144.50
        }
      }
    },
    "fundamental": {
      "pe_ratio": 24.5,
      "eps": 5.95,
      "market_cap": "2.45T",
      "dividend_yield": 0.65,
      "revenue_growth": 8.5,
      "fair_value_estimate": 155.00
    },
    "sentiment": {
      "overall": "positive|negative|neutral",
      "score": 0.72,
      "news_sentiment": 0.65,
      "social_media_sentiment": 0.78,
      "analyst_recommendations": {
        "buy": 25,
        "hold": 8,
        "sell": 2
      }
    },
    "events": [
      {
        "type": "earnings",
        "date": "2023-04-15T00:00:00Z",
        "description": "Q2 2023 Earnings Release"
      },
      {
        "type": "dividend",
        "date": "2023-03-10T00:00:00Z",
        "description": "Quarterly dividend payment"
      }
    ]
  },
  "generated_at": "2023-02-01T05:00:00Z"
}
```

### Portfolio Optimization

#### Get Portfolio Optimization

```
GET /api/decision-support/portfolio/optimize
```

Get portfolio optimization recommendations.

**Query Parameters:**
- `portfolio_id` (required): ID of the portfolio to optimize
- `objective` (optional): Optimization objective (e.g., "max_return", "min_risk", "max_sharpe")
- `constraints` (optional): JSON-encoded constraints for optimization

**Response:**
```json
{
  "current_portfolio": {
    "expected_return": 8.5,
    "volatility": 12.3,
    "sharpe_ratio": 0.69,
    "allocations": {
      "AAPL": 0.15,
      "MSFT": 0.12,
      "GOOGL": 0.10,
      "AMZN": 0.08,
      "BRK.B": 0.05,
      "other": 0.50
    }
  },
  "optimized_portfolio": {
    "expected_return": 9.8,
    "volatility": 11.5,
    "sharpe_ratio": 0.85,
    "allocations": {
      "AAPL": 0.18,
      "MSFT": 0.15,
      "GOOGL": 0.12,
      "AMZN": 0.10,
      "BRK.B": 0.08,
      "other": 0.37
    }
  },
  "rebalancing_actions": [
    {
      "symbol": "AAPL",
      "current_allocation": 0.15,
      "target_allocation": 0.18,
      "action": "buy",
      "amount_percent": 0.03
    },
    {
      "symbol": "XYZ",
      "current_allocation": 0.05,
      "target_allocation": 0.02,
      "action": "sell",
      "amount_percent": 0.03
    }
  ],
  "optimization_date": "2023-02-01T06:00:00Z"
}
```

### Alerts Configuration

#### Configure Decision Support Alerts

```
POST /api/decision-support/alerts/configure
```

Configure alerts for decision support events.

**Request Body:**
```json
{
  "alerts": [
    {
      "type": "recommendation",
      "symbols": ["AAPL", "MSFT", "GOOGL"],
      "actions": ["buy", "sell"],
      "min_confidence": 0.7,
      "notification_channels": ["email", "push", "sms"]
    },
    {
      "type": "market_insight",
      "symbols": ["SPY", "QQQ"],
      "conditions": [
        {
          "indicator": "rsi",
          "operator": "above",
          "value": 70
        },
        {
          "indicator": "price",
          "operator": "below",
          "value": "sma_200"
        }
      ],
      "notification_channels": ["email", "push"]
    },
    {
      "type": "risk_warning",
      "threshold": "high",
      "notification_channels": ["email", "sms", "push"]
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "message": "Alerts configured successfully",
  "alert_ids": [
    "alert-123",
    "alert-124",
    "alert-125"
  ]
}
```

#### Get Decision Support Alerts

```
GET /api/decision-support/alerts
```

Get currently configured decision support alerts.

**Response:**
```json
{
  "alerts": [
    {
      "id": "alert-123",
      "type": "recommendation",
      "symbols": ["AAPL", "MSFT", "GOOGL"],
      "actions": ["buy", "sell"],
      "min_confidence": 0.7,
      "notification_channels": ["email", "push", "sms"],
      "created_at": "2023-01-15T10:30:00Z",
      "last_triggered": "2023-01-28T14:45:00Z"
    },
    {
      "id": "alert-124",
      "type": "market_insight",
      "symbols": ["SPY", "QQQ"],
      "conditions": [
        {
          "indicator": "rsi",
          "operator": "above",
          "value": 70
        },
        {
          "indicator": "price",
          "operator": "below",
          "value": "sma_200"
        }
      ],
      "notification_channels": ["email", "push"],
      "created_at": "2023-01-20T09:15:00Z",
      "last_triggered": null
    }
  ]
}
```

## WebSocket API

In addition to the REST API, the Decision Support Service provides a WebSocket interface for real-time data streaming and notifications.

### Connection

```
WebSocket: wss://api.tradsys.com/ws/decision-support
```

Authentication is performed by passing the JWT token as a query parameter:

```
wss://api.tradsys.com/ws/decision-support?token=<jwt_token>
```

### Message Types

#### Subscribe to Recommendations

```json
{
  "action": "subscribe",
  "channel": "recommendations",
  "symbols": ["AAPL", "MSFT", "GOOGL"],
  "min_confidence": 0.7
}
```

#### Recommendation Update

```json
{
  "type": "recommendation",
  "timestamp": "2023-02-01T10:15:30Z",
  "symbol": "AAPL",
  "action": "buy",
  "price_target": 150.00,
  "confidence": 0.85,
  "reasoning": "Strong technical breakout with increasing volume"
}
```

#### Subscribe to Market Insights

```json
{
  "action": "subscribe",
  "channel": "market_insights",
  "symbols": ["AAPL", "MSFT", "SPY"],
  "insight_types": ["technical", "sentiment"]
}
```

#### Market Insight Update

```json
{
  "type": "market_insight",
  "timestamp": "2023-02-01T10:20:45Z",
  "symbol": "AAPL",
  "insight_type": "technical",
  "data": {
    "rsi": 72.5,
    "macd_histogram": 1.25,
    "trend_change": "bullish"
  }
}
```

## Integration Patterns

### Push Model

External decision support systems can push recommendations and insights to TradSys using the following endpoint:

```
POST /api/decision-support/external/push
```

**Request Body:**
```json
{
  "api_key": "external_system_api_key",
  "source": "external_system_name",
  "timestamp": "2023-02-01T11:00:00Z",
  "data_type": "recommendation|insight|alert",
  "data": {
    // Data structure depends on data_type
  }
}
```

### Pull Model

External systems can register webhooks to be notified when new data is available:

```
POST /api/decision-support/external/webhook/register
```

**Request Body:**
```json
{
  "callback_url": "https://external-system.com/webhook/callback",
  "events": ["new_market_data", "order_executed", "portfolio_updated"],
  "secret": "webhook_secret_for_signature_verification"
}
```

## Error Handling

All API endpoints follow a consistent error response format:

```json
{
  "error": true,
  "code": "error_code",
  "message": "Human-readable error message",
  "details": {
    // Additional error details if available
  }
}
```

Common error codes:
- `authentication_error`: Invalid or expired authentication token
- `authorization_error`: Insufficient permissions for the requested operation
- `validation_error`: Invalid request parameters
- `resource_not_found`: Requested resource not found
- `service_unavailable`: Decision support service temporarily unavailable
- `rate_limit_exceeded`: API rate limit exceeded

## Rate Limiting

API endpoints are subject to rate limiting to ensure fair usage. Rate limit information is included in response headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1612180800
```

## Versioning

The API is versioned to ensure backward compatibility. The current version is specified in the URL path:

```
/api/v1/decision-support/...
```

## Implementation Considerations

When implementing the Decision Support API, consider the following:

1. **Security**: Ensure proper authentication and authorization for all endpoints
2. **Performance**: Optimize for low latency, especially for real-time recommendations
3. **Scalability**: Design for high throughput during market hours
4. **Reliability**: Implement proper error handling and retry mechanisms
5. **Monitoring**: Add comprehensive logging and monitoring
6. **Documentation**: Keep API documentation up-to-date
7. **Testing**: Create thorough test suites for all endpoints

