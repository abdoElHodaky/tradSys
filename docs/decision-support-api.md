# Decision Support System API Documentation

This document provides detailed information about the TradSys Decision Support System (DSS) API, which enables integration with external decision support systems and analytical tools.

## API Overview

The DSS API is designed to be flexible, extensible, and high-performance, supporting multiple integration patterns and protocols to accommodate various use cases.

### Base URL

```
https://api.tradsys.com/api/v1/decision-support
```

### Supported Protocols

- **REST API**: Standard HTTP-based API for most operations
- **gRPC**: High-performance binary protocol for latency-sensitive operations
- **WebSocket**: Real-time streaming for continuous data updates

## Authentication

All API requests require authentication using OAuth 2.0 with JWT tokens.

### Obtaining Access Tokens

```
POST /auth/token
```

**Request Body:**
```json
{
  "client_id": "your-client-id",
  "client_secret": "your-client-secret",
  "grant_type": "client_credentials",
  "scope": "decision-support:read decision-support:write"
}
```

**Response:**
```json
{
  "access_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "token_type": "bearer",
  "expires_in": 3600,
  "scope": "decision-support:read decision-support:write"
}
```

### Using Access Tokens

Include the access token in the `Authorization` header of all API requests:

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

## API Endpoints

### 1. Analysis

#### Submit Analysis Request

```
POST /analyze
```

Submits market data, portfolio information, and other parameters for analysis.

**Request Body:**
```json
{
  "portfolio": {
    "positions": [
      {
        "symbol": "AAPL",
        "quantity": 100,
        "entry_price": 150.25
      },
      {
        "symbol": "MSFT",
        "quantity": 50,
        "entry_price": 280.75
      }
    ]
  },
  "market_data": {
    "symbols": ["AAPL", "MSFT", "GOOGL"],
    "timeframe": "1d",
    "start_date": "2023-01-01",
    "end_date": "2023-12-31"
  },
  "analysis_parameters": {
    "risk_tolerance": "medium",
    "investment_horizon": "long_term",
    "strategy_types": ["value", "momentum"],
    "constraints": {
      "max_position_size": 0.1,
      "sector_exposure": {
        "technology": 0.5
      }
    }
  },
  "processing_mode": "async"
}
```

**Response (Synchronous Mode):**
```json
{
  "analysis_id": "an-123456",
  "status": "completed",
  "results": {
    "recommendations": [
      {
        "symbol": "AAPL",
        "action": "hold",
        "confidence": 0.85,
        "target_price": 175.50,
        "time_horizon": "3_months",
        "rationale": "Strong fundamentals with potential growth catalysts"
      },
      {
        "symbol": "MSFT",
        "action": "buy",
        "confidence": 0.92,
        "target_price": 310.25,
        "time_horizon": "6_months",
        "rationale": "Undervalued based on cloud growth projections"
      }
    ],
    "portfolio_metrics": {
      "expected_return": 0.12,
      "volatility": 0.18,
      "sharpe_ratio": 0.67,
      "max_drawdown": 0.15
    },
    "risk_assessment": {
      "overall_risk": "medium",
      "concentration_risk": "high",
      "market_risk": "medium",
      "liquidity_risk": "low"
    }
  }
}
```

**Response (Asynchronous Mode):**
```json
{
  "analysis_id": "an-123456",
  "status": "processing",
  "estimated_completion_time": "2023-01-15T14:30:00Z",
  "status_url": "/analysis/an-123456/status"
}
```

#### Get Analysis Status

```
GET /analyze/{analysis_id}/status
```

Retrieves the status of an asynchronous analysis request.

**Response:**
```json
{
  "analysis_id": "an-123456",
  "status": "completed",
  "completion_time": "2023-01-15T14:28:30Z",
  "results_url": "/analyze/an-123456/results"
}
```

#### Get Analysis Results

```
GET /analyze/{analysis_id}/results
```

Retrieves the results of a completed analysis.

**Response:** Same as the synchronous response for the analysis request.

### 2. Recommendations

#### Get Recommendations

```
GET /recommendations
```

Retrieves trading recommendations based on current market conditions.

**Query Parameters:**
- `symbols` (optional): Comma-separated list of symbols to get recommendations for
- `strategy_types` (optional): Comma-separated list of strategy types
- `min_confidence` (optional): Minimum confidence level (0.0-1.0)
- `time_horizon` (optional): Time horizon for recommendations (short_term, medium_term, long_term)
- `limit` (optional): Maximum number of recommendations to return (default: 10)
- `offset` (optional): Offset for pagination (default: 0)

**Response:**
```json
{
  "recommendations": [
    {
      "symbol": "AAPL",
      "action": "buy",
      "confidence": 0.87,
      "target_price": 175.50,
      "time_horizon": "medium_term",
      "strategy_type": "value",
      "rationale": "Strong fundamentals with potential growth catalysts",
      "generated_at": "2023-01-15T12:30:00Z"
    },
    {
      "symbol": "MSFT",
      "action": "hold",
      "confidence": 0.75,
      "target_price": 290.25,
      "time_horizon": "short_term",
      "strategy_type": "momentum",
      "rationale": "Recent price action suggests consolidation",
      "generated_at": "2023-01-15T12:30:00Z"
    }
  ],
  "pagination": {
    "total": 42,
    "limit": 10,
    "offset": 0,
    "next_offset": 10
  }
}
```

### 3. Scenario Analysis

#### Submit Scenario Analysis

```
POST /scenarios
```

Runs what-if scenarios with different market conditions.

**Request Body:**
```json
{
  "portfolio": {
    "positions": [
      {
        "symbol": "AAPL",
        "quantity": 100,
        "entry_price": 150.25
      },
      {
        "symbol": "MSFT",
        "quantity": 50,
        "entry_price": 280.75
      }
    ]
  },
  "scenarios": [
    {
      "name": "Market Crash",
      "description": "Simulate a 20% market decline",
      "market_conditions": {
        "index_change": {
          "SPX": -0.20,
          "NDX": -0.25
        },
        "volatility_change": 0.50,
        "sector_changes": {
          "technology": -0.22,
          "healthcare": -0.15
        }
      }
    },
    {
      "name": "Interest Rate Hike",
      "description": "Simulate a 75 basis point rate increase",
      "market_conditions": {
        "interest_rate_change": 0.0075,
        "index_change": {
          "SPX": -0.05,
          "NDX": -0.08
        },
        "sector_changes": {
          "financials": 0.03,
          "utilities": -0.07
        }
      }
    }
  ]
}
```

**Response:**
```json
{
  "scenario_analysis_id": "sa-789012",
  "status": "completed",
  "results": [
    {
      "scenario_name": "Market Crash",
      "portfolio_impact": {
        "total_return": -0.215,
        "dollar_change": -5375.00,
        "positions": [
          {
            "symbol": "AAPL",
            "price_change": -0.23,
            "dollar_change": -3450.00
          },
          {
            "symbol": "MSFT",
            "price_change": -0.19,
            "dollar_change": -1925.00
          }
        ]
      },
      "risk_metrics": {
        "var_95": 6200.00,
        "expected_shortfall": 7100.00,
        "max_drawdown": 0.215
      },
      "recommendations": [
        {
          "action": "hedge",
          "instrument": "SPY",
          "strategy": "put_options",
          "rationale": "Protect against further market decline"
        }
      ]
    },
    {
      "scenario_name": "Interest Rate Hike",
      "portfolio_impact": {
        "total_return": -0.062,
        "dollar_change": -1550.00,
        "positions": [
          {
            "symbol": "AAPL",
            "price_change": -0.07,
            "dollar_change": -1050.00
          },
          {
            "symbol": "MSFT",
            "price_change": -0.05,
            "dollar_change": -500.00
          }
        ]
      },
      "risk_metrics": {
        "var_95": 1800.00,
        "expected_shortfall": 2200.00,
        "max_drawdown": 0.062
      },
      "recommendations": [
        {
          "action": "rebalance",
          "rationale": "Increase allocation to financial sector"
        }
      ]
    }
  ]
}
```

### 4. Backtesting

#### Submit Backtest

```
POST /backtest
```

Tests strategies against historical data.

**Request Body:**
```json
{
  "strategy": {
    "name": "Moving Average Crossover",
    "parameters": {
      "fast_period": 10,
      "slow_period": 50,
      "signal_period": 9
    }
  },
  "instruments": ["AAPL", "MSFT", "GOOGL"],
  "timeframe": "1d",
  "start_date": "2022-01-01",
  "end_date": "2022-12-31",
  "initial_capital": 100000,
  "position_sizing": {
    "method": "percent_of_capital",
    "value": 0.1
  },
  "execution_settings": {
    "slippage": 0.001,
    "commission": 0.0005
  }
}
```

**Response:**
```json
{
  "backtest_id": "bt-345678",
  "status": "processing",
  "estimated_completion_time": "2023-01-15T15:30:00Z",
  "status_url": "/backtest/bt-345678/status"
}
```

#### Get Backtest Status

```
GET /backtest/{backtest_id}/status
```

Retrieves the status of a backtest.

**Response:**
```json
{
  "backtest_id": "bt-345678",
  "status": "completed",
  "completion_time": "2023-01-15T15:28:30Z",
  "results_url": "/backtest/bt-345678/results"
}
```

#### Get Backtest Results

```
GET /backtest/{backtest_id}/results
```

Retrieves the results of a completed backtest.

**Response:**
```json
{
  "backtest_id": "bt-345678",
  "strategy": {
    "name": "Moving Average Crossover",
    "parameters": {
      "fast_period": 10,
      "slow_period": 50,
      "signal_period": 9
    }
  },
  "performance_metrics": {
    "total_return": 0.187,
    "annualized_return": 0.187,
    "sharpe_ratio": 1.25,
    "sortino_ratio": 1.87,
    "max_drawdown": 0.12,
    "win_rate": 0.58,
    "profit_factor": 1.65
  },
  "equity_curve": {
    "timestamps": ["2022-01-01", "2022-01-02", "..."],
    "equity": [100000, 100500, "..."]
  },
  "trades": [
    {
      "instrument": "AAPL",
      "entry_date": "2022-01-15",
      "entry_price": 172.50,
      "exit_date": "2022-02-10",
      "exit_price": 185.75,
      "quantity": 58,
      "pnl": 767.50,
      "return": 0.077
    }
  ],
  "optimization_suggestions": [
    {
      "parameter": "fast_period",
      "current_value": 10,
      "suggested_value": 12,
      "expected_improvement": 0.015
    }
  ]
}
```

### 5. Portfolio Optimization

#### Get Portfolio Optimization

```
GET /portfolio/optimize
```

Provides portfolio optimization recommendations.

**Query Parameters:**
- `objective` (required): Optimization objective (max_return, min_risk, max_sharpe)
- `risk_tolerance` (optional): Risk tolerance level (low, medium, high)
- `investment_horizon` (optional): Investment horizon (short_term, medium_term, long_term)
- `constraints` (optional): JSON-encoded constraints

**Response:**
```json
{
  "optimization_id": "opt-901234",
  "objective": "max_sharpe",
  "current_portfolio": {
    "expected_return": 0.08,
    "volatility": 0.15,
    "sharpe_ratio": 0.53
  },
  "optimized_portfolio": {
    "expected_return": 0.11,
    "volatility": 0.14,
    "sharpe_ratio": 0.79
  },
  "allocation": [
    {
      "symbol": "AAPL",
      "current_weight": 0.25,
      "optimized_weight": 0.18,
      "change": -0.07
    },
    {
      "symbol": "MSFT",
      "current_weight": 0.15,
      "optimized_weight": 0.22,
      "change": 0.07
    },
    {
      "symbol": "GOOGL",
      "current_weight": 0.10,
      "optimized_weight": 0.15,
      "change": 0.05
    }
  ],
  "rebalancing_recommendations": [
    {
      "symbol": "AAPL",
      "action": "sell",
      "quantity": 28,
      "estimated_value": 4900.00
    },
    {
      "symbol": "MSFT",
      "action": "buy",
      "quantity": 12,
      "estimated_value": 3360.00
    },
    {
      "symbol": "GOOGL",
      "action": "buy",
      "quantity": 5,
      "estimated_value": 1450.00
    }
  ],
  "efficient_frontier": {
    "returns": [0.05, 0.06, 0.07, 0.08, 0.09, 0.10, 0.11, 0.12],
    "volatilities": [0.08, 0.09, 0.11, 0.13, 0.15, 0.18, 0.22, 0.28],
    "sharpe_ratios": [0.63, 0.67, 0.64, 0.62, 0.60, 0.56, 0.50, 0.43]
  }
}
```

### 6. Alerts Configuration

#### Configure Alerts

```
POST /alerts/configure
```

Configures alerts based on market conditions or analysis results.

**Request Body:**
```json
{
  "alerts": [
    {
      "name": "Price Alert",
      "description": "Alert when AAPL crosses above $180",
      "conditions": {
        "type": "price_cross",
        "symbol": "AAPL",
        "direction": "above",
        "value": 180.00
      },
      "notification": {
        "channels": ["email", "webhook"],
        "email": "user@example.com",
        "webhook_url": "https://example.com/webhook"
      },
      "expiration": "2023-06-30T23:59:59Z"
    },
    {
      "name": "Technical Indicator Alert",
      "description": "Alert when MSFT RSI crosses below 30",
      "conditions": {
        "type": "indicator_cross",
        "symbol": "MSFT",
        "indicator": "rsi",
        "parameters": {
          "period": 14
        },
        "direction": "below",
        "value": 30
      },
      "notification": {
        "channels": ["email"],
        "email": "user@example.com"
      }
    }
  ]
}
```

**Response:**
```json
{
  "status": "success",
  "alerts": [
    {
      "id": "alert-123456",
      "name": "Price Alert",
      "status": "active",
      "created_at": "2023-01-15T13:45:00Z"
    },
    {
      "id": "alert-123457",
      "name": "Technical Indicator Alert",
      "status": "active",
      "created_at": "2023-01-15T13:45:00Z"
    }
  ]
}
```

#### Get Alerts

```
GET /alerts
```

Retrieves configured alerts.

**Query Parameters:**
- `status` (optional): Filter by status (active, triggered, expired)
- `type` (optional): Filter by alert type
- `limit` (optional): Maximum number of alerts to return (default: 10)
- `offset` (optional): Offset for pagination (default: 0)

**Response:**
```json
{
  "alerts": [
    {
      "id": "alert-123456",
      "name": "Price Alert",
      "description": "Alert when AAPL crosses above $180",
      "conditions": {
        "type": "price_cross",
        "symbol": "AAPL",
        "direction": "above",
        "value": 180.00
      },
      "notification": {
        "channels": ["email", "webhook"]
      },
      "status": "active",
      "created_at": "2023-01-15T13:45:00Z",
      "expiration": "2023-06-30T23:59:59Z"
    }
  ],
  "pagination": {
    "total": 5,
    "limit": 10,
    "offset": 0
  }
}
```

### 7. Model Management

#### Register Model

```
POST /models
```

Registers a custom decision model for use in analysis.

**Request Body:**
```json
{
  "name": "Custom LSTM Model",
  "description": "Long Short-Term Memory model for price prediction",
  "version": "1.0.0",
  "model_type": "time_series_prediction",
  "supported_instruments": ["stocks", "forex"],
  "input_features": ["price", "volume", "volatility"],
  "output_features": ["price_prediction", "confidence"],
  "parameters": {
    "lookback_period": 30,
    "prediction_horizon": 5,
    "training_frequency": "daily"
  },
  "endpoint": {
    "type": "rest",
    "url": "https://example.com/model/predict",
    "authentication": {
      "type": "api_key",
      "header_name": "X-API-Key"
    }
  }
}
```

**Response:**
```json
{
  "model_id": "model-567890",
  "status": "registered",
  "created_at": "2023-01-15T14:00:00Z",
  "api_key": "sk_model_567890_abcdefghijklmnopqrstuvwxyz"
}
```

#### Get Models

```
GET /models
```

Retrieves registered models.

**Query Parameters:**
- `model_type` (optional): Filter by model type
- `status` (optional): Filter by status (active, inactive)
- `limit` (optional): Maximum number of models to return (default: 10)
- `offset` (optional): Offset for pagination (default: 0)

**Response:**
```json
{
  "models": [
    {
      "model_id": "model-567890",
      "name": "Custom LSTM Model",
      "description": "Long Short-Term Memory model for price prediction",
      "version": "1.0.0",
      "model_type": "time_series_prediction",
      "status": "active",
      "created_at": "2023-01-15T14:00:00Z",
      "last_used": "2023-01-16T09:30:00Z",
      "performance_metrics": {
        "accuracy": 0.82,
        "mse": 0.0045,
        "mae": 0.052
      }
    }
  ],
  "pagination": {
    "total": 3,
    "limit": 10,
    "offset": 0
  }
}
```

### 8. Real-time Insights

#### Get Market Insights

```
GET /insights/{symbol}
```

Provides real-time market insights for a specific symbol.

**Response:**
```json
{
  "symbol": "AAPL",
  "timestamp": "2023-01-15T14:15:00Z",
  "price": {
    "current": 172.50,
    "change": 2.75,
    "change_percent": 0.0162
  },
  "technical_analysis": {
    "trend": "bullish",
    "strength": "moderate",
    "support_levels": [168.50, 165.25],
    "resistance_levels": [175.00, 180.50],
    "indicators": {
      "rsi": {
        "value": 62.5,
        "interpretation": "neutral"
      },
      "macd": {
        "value": 1.25,
        "signal": 0.75,
        "histogram": 0.50,
        "interpretation": "bullish"
      }
    }
  },
  "fundamental_analysis": {
    "pe_ratio": 28.5,
    "eps": 6.05,
    "market_cap": "2.85T",
    "dividend_yield": 0.0052,
    "interpretation": "fairly_valued"
  },
  "sentiment_analysis": {
    "overall": "positive",
    "score": 0.72,
    "news_sentiment": 0.65,
    "social_media_sentiment": 0.78,
    "recent_news": [
      {
        "title": "Apple Reports Record Q1 Earnings",
        "source": "Financial Times",
        "url": "https://ft.com/article/123",
        "sentiment": "positive",
        "published_at": "2023-01-14T18:30:00Z"
      }
    ]
  },
  "recommendations": {
    "consensus": "buy",
    "price_target": {
      "average": 185.50,
      "low": 165.00,
      "high": 210.00
    },
    "analyst_ratings": {
      "buy": 25,
      "hold": 8,
      "sell": 2
    }
  }
}
```

#### WebSocket Connection for Real-time Updates

To establish a WebSocket connection for real-time updates:

```
WebSocket: wss://api.tradsys.com/api/v1/decision-support/insights/stream
```

**Connection Parameters:**
- `symbols`: Comma-separated list of symbols to stream insights for
- `access_token`: OAuth access token

**Example Message:**
```json
{
  "type": "insight_update",
  "symbol": "AAPL",
  "timestamp": "2023-01-15T14:16:00Z",
  "price": {
    "current": 172.75,
    "change": 3.00,
    "change_percent": 0.0177
  },
  "technical_analysis": {
    "trend": "bullish",
    "strength": "moderate"
  },
  "sentiment_analysis": {
    "overall": "positive",
    "score": 0.73
  }
}
```

## Error Handling

All API endpoints use consistent error responses:

**Example Error Response:**
```json
{
  "error": {
    "code": "invalid_request",
    "message": "Invalid request parameters",
    "details": [
      {
        "field": "portfolio.positions[0].quantity",
        "message": "Quantity must be a positive number"
      }
    ],
    "request_id": "req-abcdef123456",
    "documentation_url": "https://docs.tradsys.com/api/errors#invalid_request"
  }
}
```

### Common Error Codes

- `authentication_error`: Authentication failed
- `authorization_error`: Insufficient permissions
- `invalid_request`: Invalid request parameters
- `resource_not_found`: Requested resource not found
- `rate_limit_exceeded`: Rate limit exceeded
- `internal_error`: Internal server error
- `service_unavailable`: Service temporarily unavailable

## Rate Limiting

The API implements rate limiting to ensure fair usage. Rate limits are specified in the response headers:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1673805600
```

When a rate limit is exceeded, the API returns a `429 Too Many Requests` status code with a `rate_limit_exceeded` error.

## Webhooks

The DSS API can send webhook notifications for various events:

### Webhook Events

- `analysis.completed`: Analysis has completed
- `backtest.completed`: Backtest has completed
- `alert.triggered`: Alert condition has been triggered
- `recommendation.generated`: New recommendation has been generated
- `model.performance_update`: Model performance metrics have been updated

### Webhook Payload

```json
{
  "event": "analysis.completed",
  "timestamp": "2023-01-15T14:30:00Z",
  "data": {
    "analysis_id": "an-123456",
    "status": "completed",
    "results_url": "/analyze/an-123456/results"
  }
}
```

### Webhook Security

Webhooks are secured using HMAC signatures. The signature is included in the `X-TradSys-Signature` header:

```
X-TradSys-Signature: t=1673805600,v1=5257a869e7ecebeda32affa62cdca3fa51cad7e77a0e56ff536d0ce8e108d8bd
```

To verify the signature:
1. Extract the timestamp (`t`) and signature (`v1`) from the header
2. Compute the HMAC using your webhook secret: `HMAC-SHA256(webhook_secret, timestamp + '.' + request_body)`
3. Compare the computed signature with the received signature

## SDK and Client Libraries

TradSys provides official client SDKs for popular languages:

- [Go SDK](https://github.com/tradsys/tradsys-go)
- [Python SDK](https://github.com/tradsys/tradsys-python)
- [JavaScript SDK](https://github.com/tradsys/tradsys-js)

### Example SDK Usage (Python)

```python
from tradsys import DecisionSupportClient

# Initialize client
client = DecisionSupportClient(api_key="your-api-key")

# Get recommendations
recommendations = client.get_recommendations(
    symbols=["AAPL", "MSFT"],
    min_confidence=0.7,
    time_horizon="medium_term"
)

# Submit analysis request
analysis = client.analyze(
    portfolio={
        "positions": [
            {"symbol": "AAPL", "quantity": 100, "entry_price": 150.25},
            {"symbol": "MSFT", "quantity": 50, "entry_price": 280.75}
        ]
    },
    market_data={
        "symbols": ["AAPL", "MSFT", "GOOGL"],
        "timeframe": "1d",
        "start_date": "2023-01-01",
        "end_date": "2023-12-31"
    },
    analysis_parameters={
        "risk_tolerance": "medium",
        "investment_horizon": "long_term"
    }
)

# Stream real-time insights
for insight in client.stream_insights(symbols=["AAPL", "MSFT"]):
    print(f"New insight for {insight['symbol']}: {insight['sentiment_analysis']['overall']}")
```

## Changelog

### v1.0.0 (2023-01-01)

- Initial release of the Decision Support System API

### v1.1.0 (2023-03-15)

- Added portfolio optimization endpoint
- Added WebSocket support for real-time insights
- Improved error handling and documentation

### v1.2.0 (2023-06-30)

- Added model management endpoints
- Enhanced backtesting capabilities
- Added support for custom alert conditions
- Improved performance and scalability

## Support

For API support, please contact:

- Email: api-support@tradsys.com
- Support Portal: https://support.tradsys.com
- API Status: https://status.tradsys.com

