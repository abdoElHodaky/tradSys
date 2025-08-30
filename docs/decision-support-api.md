# Decision Support System API Documentation

This document provides detailed information about the Decision Support System (DSS) API for TradSys. The DSS API enables integration with external decision support systems and provides analytical capabilities for trading decisions.

## API Endpoints

### Analysis

#### Analyze Market Data

```
POST /api/decision-support/analyze
```

Analyzes market data and provides trading recommendations.

**Request Body:**

```json
{
  "symbol": "AAPL",
  "timeframe": "1h",
  "start_time": "2023-01-01T00:00:00Z",
  "end_time": "2023-01-31T23:59:59Z",
  "indicators": ["rsi", "macd", "bollinger"],
  "parameters": {
    "rsi_period": 14,
    "macd_fast": 12,
    "macd_slow": 26,
    "macd_signal": 9,
    "bollinger_period": 20,
    "bollinger_std": 2
  }
}
```

**Response:**

```json
[
  {
    "symbol": "AAPL",
    "action": "buy",
    "price": 150.25,
    "quantity": 10,
    "confidence": 75.5,
    "rationale": "RSI indicates oversold condition, MACD shows bullish crossover",
    "timestamp": "2023-01-31T15:30:00Z",
    "expires_at": "2023-02-01T15:30:00Z",
    "indicators": {
      "rsi": 32.5,
      "macd": 0.75,
      "trend": 1.0
    }
  }
]
```

### Recommendations

#### Get Recommendations

```
GET /api/decision-support/recommendations?symbol=AAPL&limit=5
```

Retrieves trading recommendations for a specific symbol or all symbols.

**Query Parameters:**

- `symbol` (optional): Trading symbol to get recommendations for
- `limit` (optional): Maximum number of recommendations to return (default: 10)

**Response:**

```json
[
  {
    "symbol": "AAPL",
    "action": "buy",
    "price": 150.25,
    "quantity": 10,
    "confidence": 75.5,
    "rationale": "RSI indicates oversold condition, MACD shows bullish crossover",
    "timestamp": "2023-01-31T15:30:00Z",
    "expires_at": "2023-02-01T15:30:00Z",
    "indicators": {
      "rsi": 32.5,
      "macd": 0.75,
      "trend": 1.0
    }
  },
  {
    "symbol": "MSFT",
    "action": "sell",
    "price": 250.75,
    "quantity": 5,
    "confidence": 65.0,
    "rationale": "Approaching resistance level with bearish divergence",
    "timestamp": "2023-01-31T14:45:00Z",
    "expires_at": "2023-02-01T14:45:00Z",
    "indicators": {
      "rsi": 72.5,
      "macd": -0.5,
      "trend": -0.5
    }
  }
]
```

### Scenario Analysis

#### Analyze Scenarios

```
POST /api/decision-support/scenarios
```

Analyzes different market scenarios and their potential impact.

**Request Body:**

```json
{
  "base_symbol": "SPY",
  "scenarios": [
    {
      "name": "bullish",
      "description": "Market continues upward trend",
      "price_change": 0.05,
      "volatility_change": -0.1,
      "probability": 0.6,
      "additional_factors": {
        "interest_rate_change": 0.0025,
        "economic_growth": "strong"
      }
    },
    {
      "name": "bearish",
      "description": "Market reverses into downtrend",
      "price_change": -0.08,
      "volatility_change": 0.3,
      "probability": 0.3,
      "additional_factors": {
        "interest_rate_change": 0.005,
        "economic_growth": "weak"
      }
    }
  ],
  "portfolio": {
    "positions": [
      {
        "symbol": "AAPL",
        "quantity": 100,
        "entry_price": 145.75,
        "current_price": 150.25,
        "unrealized_pl": 450.0
      },
      {
        "symbol": "MSFT",
        "quantity": 50,
        "entry_price": 240.50,
        "current_price": 250.75,
        "unrealized_pl": 512.5
      }
    ],
    "cash": 10000.0,
    "total_value": 35000.0
  },
  "risk_parameters": {
    "max_drawdown": 0.1,
    "var_confidence": 0.95,
    "risk_free_rate": 0.03
  }
}
```

**Response:**

```json
{
  "bullish": {
    "expected_return": 1750.0,
    "risk": 0.05,
    "probability": 0.6,
    "recommended_actions": ["increase_tech_exposure", "reduce_cash"]
  },
  "bearish": {
    "expected_return": -2800.0,
    "risk": 0.15,
    "probability": 0.3,
    "recommended_actions": ["hedge_with_puts", "increase_defensive_positions"]
  },
  "overall_assessment": {
    "expected_return": 350.0,
    "risk_level": "moderate",
    "confidence": 0.7
  }
}
```

### Backtesting

#### Run Backtest

```
POST /api/decision-support/backtest
```

Runs a backtest of a trading strategy against historical data.

**Request Body:**

```json
{
  "strategy": "momentum",
  "symbols": ["AAPL", "MSFT", "AMZN"],
  "start_time": "2022-01-01T00:00:00Z",
  "end_time": "2022-12-31T23:59:59Z",
  "initial_capital": 100000.0,
  "parameters": {
    "lookback_period": 20,
    "momentum_threshold": 0.05,
    "position_size": 0.1,
    "stop_loss": 0.05,
    "take_profit": 0.15
  }
}
```

**Response:**

```json
{
  "strategy": "momentum",
  "start_time": "2022-01-01T00:00:00Z",
  "end_time": "2022-12-31T23:59:59Z",
  "initial_capital": 100000.0,
  "final_capital": 125000.0,
  "total_return": 0.25,
  "annualized_return": 0.25,
  "sharpe_ratio": 1.2,
  "max_drawdown": 0.12,
  "win_rate": 0.65,
  "trades": [
    {
      "symbol": "AAPL",
      "entry_time": "2022-01-15T10:30:00Z",
      "entry_price": 170.25,
      "exit_time": "2022-02-10T15:45:00Z",
      "exit_price": 175.50,
      "quantity": 100,
      "profit_loss": 525.0,
      "side": "buy"
    },
    {
      "symbol": "MSFT",
      "entry_time": "2022-02-01T09:30:00Z",
      "entry_price": 310.75,
      "exit_time": "2022-02-15T16:00:00Z",
      "exit_price": 300.25,
      "quantity": 50,
      "profit_loss": -525.0,
      "side": "buy"
    }
  ],
  "equity_curve": {
    "2022-01-01": 100000.0,
    "2022-01-15": 100525.0,
    "2022-02-01": 101050.0,
    "2022-02-15": 100525.0,
    "2022-12-31": 125000.0
  }
}
```

### Market Insights

#### Get Market Insights

```
GET /api/decision-support/insights/AAPL
```

Retrieves market insights for a specific symbol.

**Response:**

```json
{
  "trend": {
    "short_term": "bullish",
    "medium_term": "bullish",
    "long_term": "neutral",
    "strength": 0.75
  },
  "support_resistance": {
    "support_levels": [145.0, 142.5, 140.0],
    "resistance_levels": [155.0, 157.5, 160.0]
  },
  "volatility": {
    "current": 0.15,
    "historical": 0.12,
    "forecast": 0.14,
    "percentile": 75,
    "trend": "increasing"
  },
  "sentiment": {
    "overall": "positive",
    "social_media": "very_positive",
    "news": "neutral",
    "analyst": "positive"
  },
  "correlations": {
    "sp500": 0.75,
    "sector": 0.85,
    "vix": -0.6
  },
  "events": [
    {
      "type": "earnings",
      "date": "2023-04-15",
      "importance": "high",
      "description": "Q2 Earnings Report"
    },
    {
      "type": "dividend",
      "date": "2023-05-01",
      "importance": "medium",
      "description": "Quarterly Dividend Payment"
    }
  ]
}
```

### Portfolio Optimization

#### Optimize Portfolio

```
GET /api/decision-support/portfolio/optimize
```

Optimizes a portfolio based on the specified objective.

**Request Body:**

```json
{
  "positions": [
    {
      "symbol": "AAPL",
      "quantity": 100,
      "entry_price": 145.75,
      "current_price": 150.25,
      "unrealized_pl": 450.0
    },
    {
      "symbol": "MSFT",
      "quantity": 50,
      "entry_price": 240.50,
      "current_price": 250.75,
      "unrealized_pl": 512.5
    }
  ],
  "cash": 10000.0,
  "total_value": 35000.0
}
```

**Query Parameters:**

- `objective`: Optimization objective (options: "risk", "return", "sharpe", default: "sharpe")

**Response:**

```json
{
  "positions": [
    {
      "symbol": "AAPL",
      "quantity": 120,
      "entry_price": 145.75,
      "current_price": 150.25,
      "unrealized_pl": 540.0
    },
    {
      "symbol": "MSFT",
      "quantity": 40,
      "entry_price": 240.50,
      "current_price": 250.75,
      "unrealized_pl": 410.0
    },
    {
      "symbol": "AMZN",
      "quantity": 10,
      "entry_price": 3200.0,
      "current_price": 3200.0,
      "unrealized_pl": 0.0
    }
  ],
  "cash": 5000.0,
  "total_value": 36750.0
}
```

### Alerts

#### Configure Alert

```
POST /api/decision-support/alerts/configure
```

Configures an alert based on market conditions or analysis results.

**Request Body:**

```json
{
  "name": "AAPL RSI Alert",
  "description": "Alert when RSI crosses below 30 or above 70",
  "symbol": "AAPL",
  "condition": "rsi < 30 || rsi > 70",
  "threshold": 30.0,
  "notification_channels": ["email", "sms", "app"],
  "enabled": true
}
```

**Response:**

```json
{
  "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
}
```

#### Get Alerts

```
GET /api/decision-support/alerts?acknowledged=false
```

Retrieves current alerts.

**Query Parameters:**

- `acknowledged`: Filter by acknowledgement status (true/false)

**Response:**

```json
[
  {
    "id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
    "configuration_id": "c1d2e3f4-5678-90ab-cdef-123456789012",
    "timestamp": "2023-01-31T14:30:00Z",
    "symbol": "AAPL",
    "message": "RSI crossed below 30, indicating oversold conditions",
    "value": 28.5,
    "threshold": 30.0,
    "acknowledged": false
  },
  {
    "id": "b2c3d4e5-f6a7-8901-bcde-f23456789012",
    "configuration_id": "d2e3f4a5-6789-01bc-defg-234567890123",
    "timestamp": "2023-01-31T10:15:00Z",
    "symbol": "MSFT",
    "message": "Price crossed below 200-day moving average",
    "value": 245.75,
    "threshold": 248.50,
    "acknowledged": false
  }
]
```

#### Acknowledge Alert

```
POST /api/decision-support/alerts/a1b2c3d4-e5f6-7890-abcd-ef1234567890/acknowledge
```

Acknowledges an alert.

**Response:**

```
HTTP/1.1 200 OK
```

## Error Handling

The API uses standard HTTP status codes to indicate the success or failure of a request:

- `200 OK`: The request was successful
- `400 Bad Request`: The request was invalid or malformed
- `401 Unauthorized`: Authentication failed
- `403 Forbidden`: The authenticated user does not have permission to access the resource
- `404 Not Found`: The requested resource was not found
- `500 Internal Server Error`: An error occurred on the server

Error responses include a JSON body with details about the error:

```json
{
  "error": "Invalid request body",
  "message": "The request body is missing required fields",
  "details": {
    "missing_fields": ["symbol", "timeframe"]
  }
}
```

## Authentication

The API uses OAuth 2.0 for authentication. To access the API, you need to include an Authorization header with a valid access token:

```
Authorization: Bearer <access_token>
```

## Rate Limiting

The API implements rate limiting to ensure fair usage and system stability. Rate limits are applied on a per-user basis and are reset hourly.

The following headers are included in API responses to provide information about rate limiting:

- `X-RateLimit-Limit`: The maximum number of requests allowed per hour
- `X-RateLimit-Remaining`: The number of requests remaining in the current rate limit window
- `X-RateLimit-Reset`: The time at which the current rate limit window resets (Unix timestamp)

When a rate limit is exceeded, the API returns a `429 Too Many Requests` status code.

## Versioning

The API is versioned to ensure backward compatibility. The current version is v1.

## Webhooks

The API supports webhooks for real-time notifications of events such as new recommendations, alerts, and analysis results. To configure webhooks, use the webhook management endpoints:

```
POST /api/webhooks/configure
GET /api/webhooks
DELETE /api/webhooks/{id}
```

## Integration Examples

### Python Example

```python
import requests
import json

API_URL = "https://api.tradsys.com"
API_KEY = "your_api_key"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

# Get recommendations
response = requests.get(
    f"{API_URL}/api/decision-support/recommendations?symbol=AAPL&limit=5",
    headers=headers
)

if response.status_code == 200:
    recommendations = response.json()
    for rec in recommendations:
        print(f"Symbol: {rec['symbol']}, Action: {rec['action']}, Price: {rec['price']}")
else:
    print(f"Error: {response.status_code}, {response.text}")

# Run a backtest
backtest_request = {
    "strategy": "momentum",
    "symbols": ["AAPL", "MSFT", "AMZN"],
    "start_time": "2022-01-01T00:00:00Z",
    "end_time": "2022-12-31T23:59:59Z",
    "initial_capital": 100000.0,
    "parameters": {
        "lookback_period": 20,
        "momentum_threshold": 0.05,
        "position_size": 0.1
    }
}

response = requests.post(
    f"{API_URL}/api/decision-support/backtest",
    headers=headers,
    data=json.dumps(backtest_request)
)

if response.status_code == 200:
    result = response.json()
    print(f"Backtest result: {result['total_return']:.2%} return, {result['sharpe_ratio']:.2f} Sharpe ratio")
else:
    print(f"Error: {response.status_code}, {response.text}")
```

### JavaScript Example

```javascript
const API_URL = "https://api.tradsys.com";
const API_KEY = "your_api_key";

const headers = {
  "Authorization": `Bearer ${API_KEY}`,
  "Content-Type": "application/json"
};

// Get market insights
fetch(`${API_URL}/api/decision-support/insights/AAPL`, { headers })
  .then(response => {
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    return response.json();
  })
  .then(insights => {
    console.log("Market Insights:", insights);
    console.log(`Short-term trend: ${insights.trend.short_term}`);
    console.log(`Support levels: ${insights.support_resistance.support_levels.join(", ")}`);
  })
  .catch(error => {
    console.error("Error fetching insights:", error);
  });

// Configure an alert
const alertConfig = {
  "name": "AAPL RSI Alert",
  "description": "Alert when RSI crosses below 30 or above 70",
  "symbol": "AAPL",
  "condition": "rsi < 30 || rsi > 70",
  "threshold": 30.0,
  "notification_channels": ["email", "app"],
  "enabled": true
};

fetch(`${API_URL}/api/decision-support/alerts/configure`, {
  method: "POST",
  headers,
  body: JSON.stringify(alertConfig)
})
  .then(response => {
    if (!response.ok) {
      throw new Error(`HTTP error! Status: ${response.status}`);
    }
    return response.json();
  })
  .then(result => {
    console.log(`Alert configured with ID: ${result.id}`);
  })
  .catch(error => {
    console.error("Error configuring alert:", error);
  });
```

## Best Practices

1. **Use Pagination**: When retrieving large datasets, use pagination parameters to limit the amount of data returned in a single request.

2. **Handle Rate Limits**: Implement exponential backoff and retry logic to handle rate limiting.

3. **Webhook Reliability**: Implement proper error handling and retry logic for webhook deliveries.

4. **Cache Results**: Cache API responses when appropriate to reduce the number of API calls.

5. **Use Compression**: Enable gzip compression for API requests and responses to reduce bandwidth usage.

6. **Validate Inputs**: Always validate user inputs before sending them to the API.

7. **Handle Errors Gracefully**: Implement proper error handling to provide a good user experience.

8. **Monitor API Usage**: Monitor your API usage to avoid hitting rate limits and to identify potential issues.

## Support

For API support, please contact api-support@tradsys.com or visit our developer portal at https://developers.tradsys.com.

