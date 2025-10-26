# TradSys v3 API Reference & Integration Guide

## 🔌 API Overview

TradSys v3 provides a comprehensive REST API and WebSocket interface for high-frequency trading operations, supporting multiple asset classes including bonds, ETFs, and traditional securities.

**Version**: 3.0.0  
**Architecture Phase**: 16-17 Complete (90% optimization)  
**Base URL**: `http://localhost:8080/api/v1`  
**WebSocket URL**: `ws://localhost:8080/ws`  
**Authentication**: JWT Bearer Token / API Key

## 📚 Documentation Access

- **Swagger YAML**: `/swagger` or `/docs/api/swagger.yaml`
- **API Documentation**: `/api-docs` 
- **Full Documentation**: `/docs`
- **Architecture Docs**: `/docs/architecture/`
- **Deployment Guide**: `/docs/deployment/`  

## 🔐 Authentication

### JWT Token Authentication
```bash
# Login to get JWT token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "your-username",
    "password": "your-password"
  }'

# Response
{
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
  "expires_at": "2024-01-01T12:00:00Z",
  "user_id": "user123"
}

# Use token in subsequent requests
curl -H "Authorization: Bearer <token>" \
     -X GET http://localhost:8080/api/v1/orders
```

### API Key Authentication (Alternative)
```bash
curl -H "X-API-Key: your-api-key" \
     -X GET http://localhost:8080/api/v1/orders
```

## 📊 Order Management API

### Create Order
```http
POST /api/v1/orders
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_id": "user123",
  "client_order_id": "client-order-001",
  "symbol": "AAPL",
  "side": "buy",
  "type": "limit",
  "quantity": 100,
  "price": 150.50,
  "time_in_force": "GTC",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

**Response:**
```json
{
  "id": "order-uuid-123",
  "user_id": "user123",
  "client_order_id": "client-order-001",
  "symbol": "AAPL",
  "side": "buy",
  "type": "limit",
  "quantity": 100,
  "price": 150.50,
  "filled_quantity": 0,
  "status": "new",
  "time_in_force": "GTC",
  "created_at": "2024-01-01T10:00:00Z",
  "updated_at": "2024-01-01T10:00:00Z",
  "expires_at": "2024-12-31T23:59:59Z"
}
```

### Get Order
```http
GET /api/v1/orders/{order_id}
Authorization: Bearer <token>
```

### List Orders
```http
GET /api/v1/orders?user_id=user123&symbol=AAPL&status=pending&limit=50&offset=0
Authorization: Bearer <token>
```

**Query Parameters:**
- `user_id` (string) - Filter by user ID
- `symbol` (string) - Filter by trading symbol
- `side` (string) - Filter by order side (buy/sell)
- `type` (string) - Filter by order type
- `status` (string) - Filter by order status
- `start_time` (string) - Filter orders after this time (ISO 8601)
- `end_time` (string) - Filter orders before this time (ISO 8601)
- `limit` (int) - Maximum number of results (default: 50, max: 1000)
- `offset` (int) - Number of results to skip (default: 0)

### Update Order
```http
PUT /api/v1/orders/{order_id}
Content-Type: application/json
Authorization: Bearer <token>

{
  "quantity": 150,
  "price": 149.75,
  "time_in_force": "IOC"
}
```

### Cancel Order
```http
DELETE /api/v1/orders/{order_id}
Authorization: Bearer <token>
```

### Bulk Cancel Orders
```http
POST /api/v1/orders/cancel-bulk
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_id": "user123",
  "symbol": "AAPL",
  "side": "buy"
}
```

## 🛡️ Risk Management API

### Get Risk Metrics
```http
GET /api/v1/risk/metrics/{user_id}
Authorization: Bearer <token>
```

**Response:**
```json
{
  "user_id": "user123",
  "total_unrealized_pnl": 1250.75,
  "total_market_value": 50000.00,
  "portfolio_var_95": 2500.00,
  "concentration_risk": 0.35,
  "risk_level": "medium",
  "positions": [
    {
      "symbol": "AAPL",
      "quantity": 100,
      "market_value": 15050.00,
      "unrealized_pnl": 550.00,
      "var_95": 750.00,
      "risk_level": "low"
    }
  ],
  "calculated_at": "2024-01-01T10:00:00Z"
}
```

### Calculate Order Risk
```http
POST /api/v1/risk/calculate-order
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_id": "user123",
  "symbol": "AAPL",
  "side": "buy",
  "quantity": 100,
  "price": 150.50,
  "order_type": "limit"
}
```

### Get Risk Limits
```http
GET /api/v1/risk/limits/{user_id}
Authorization: Bearer <token>
```

### Set Risk Limits
```http
POST /api/v1/risk/limits
Content-Type: application/json
Authorization: Bearer <token>

{
  "user_id": "user123",
  "limits": [
    {
      "type": "position",
      "symbol": "AAPL",
      "value": 10000.00
    },
    {
      "type": "var",
      "value": 5000.00
    }
  ]
}
```

## ✅ Compliance API

### Validate Compliance
```http
POST /api/v1/compliance/validate
Content-Type: application/json
Authorization: Bearer <token>

{
  "type": "order",
  "user_id": "user123",
  "order_data": {
    "symbol": "AAPL",
    "side": "buy",
    "quantity": 100,
    "price": 150.50,
    "order_type": "limit"
  }
}
```

**Response:**
```json
{
  "passed": true,
  "violations": [],
  "warnings": [
    {
      "rule_id": "concentration_warning",
      "rule_name": "Portfolio Concentration Warning",
      "description": "Position will exceed 30% of portfolio",
      "timestamp": "2024-01-01T10:00:00Z"
    }
  ],
  "score": 85.5,
  "validated_at": "2024-01-01T10:00:00Z"
}
```

### Get Compliance Rules
```http
GET /api/v1/compliance/rules?regulation=sharia&enabled=true
Authorization: Bearer <token>
```

### Sharia Compliance Check
```http
POST /api/v1/compliance/sharia/validate
Content-Type: application/json
Authorization: Bearer <token>

{
  "symbol": "AAPL",
  "user_id": "user123"
}
```

## 🔌 Exchange Integration API

### Get Exchange Info
```http
GET /api/v1/exchanges/{exchange_id}/info
Authorization: Bearer <token>
```

**Response:**
```json
{
  "id": "ADX",
  "name": "Abu Dhabi Securities Exchange",
  "country": "UAE",
  "timezone": "Asia/Dubai",
  "trading_hours": {
    "monday": {
      "is_open": true,
      "open_time": "10:00:00",
      "close_time": "15:00:00"
    }
  },
  "supported_assets": ["stock", "bond", "etf", "sukuk"],
  "islamic_compliant": true
}
```

### Get Market Data
```http
GET /api/v1/exchanges/{exchange_id}/market-data/{symbol}
Authorization: Bearer <token>
```

### Get Order Book
```http
GET /api/v1/exchanges/{exchange_id}/orderbook/{symbol}?depth=10
Authorization: Bearer <token>
```

### Get Symbols
```http
GET /api/v1/exchanges/{exchange_id}/symbols?asset_type=stock&islamic_compliant=true
Authorization: Bearer <token>
```

## 🌐 WebSocket API

### Connection
```javascript
const ws = new WebSocket('ws://localhost:8080/ws?token=<jwt-token>');

ws.onopen = function() {
    console.log('Connected to TradSys WebSocket');
};

ws.onmessage = function(event) {
    const message = JSON.parse(event.data);
    console.log('Received:', message);
};
```

### Message Format
All WebSocket messages follow this format:
```json
{
  "type": "message_type",
  "channel": "channel_name",
  "data": { /* message data */ },
  "timestamp": "2024-01-01T10:00:00Z",
  "message_id": "msg_123"
}
```

### Subscribe to Market Data
```javascript
ws.send(JSON.stringify({
    type: 'subscribe',
    data: {
        channel: 'market_data',
        symbol: 'AAPL',
        type: 'market_data'
    }
}));
```

### Subscribe to Order Updates
```javascript
ws.send(JSON.stringify({
    type: 'subscribe',
    data: {
        channel: 'order_updates',
        type: 'order_updates'
    }
}));
```

### Available Channels
- `market_data` - Real-time price updates
- `order_book` - Order book depth changes
- `trades` - Trade executions
- `order_updates` - Order status changes
- `portfolio` - Portfolio updates
- `alerts` - System alerts
- `compliance` - Compliance notifications

### Market Data Message
```json
{
  "type": "market_data",
  "channel": "market_data",
  "data": {
    "symbol": "AAPL",
    "price": 150.75,
    "volume": 1000,
    "bid_price": 150.50,
    "ask_price": 150.75,
    "change_24h": 2.25,
    "change_perc": 1.52
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

### Order Update Message
```json
{
  "type": "order_update",
  "channel": "order_updates",
  "data": {
    "order_id": "order-123",
    "status": "filled",
    "filled_quantity": 100,
    "average_price": 150.60,
    "trades": [
      {
        "trade_id": "trade-456",
        "quantity": 100,
        "price": 150.60,
        "timestamp": "2024-01-01T10:00:00Z"
      }
    ]
  },
  "timestamp": "2024-01-01T10:00:00Z"
}
```

## 📊 Monitoring & Metrics API

### System Health
```http
GET /api/v1/health
```

**Response:**
```json
{
  "status": "healthy",
  "timestamp": "2024-01-01T10:00:00Z",
  "services": {
    "database": "healthy",
    "redis": "healthy",
    "matching_engine": "healthy",
    "websocket": "healthy"
  },
  "uptime": "72h30m15s"
}
```

### System Metrics
```http
GET /api/v1/metrics
Authorization: Bearer <token>
```

### Trading Statistics
```http
GET /api/v1/stats/trading?period=24h
Authorization: Bearer <token>
```

**Response:**
```json
{
  "period": "24h",
  "total_orders": 15420,
  "total_trades": 8750,
  "total_volume": 125000000.50,
  "average_latency_ms": 2.5,
  "orders_per_second": 180.5,
  "trades_per_second": 101.2,
  "top_symbols": [
    {
      "symbol": "AAPL",
      "volume": 25000000.00,
      "trades": 1250
    }
  ]
}
```

## 🚨 Error Handling

### Error Response Format
```json
{
  "error": {
    "code": "INVALID_ORDER",
    "message": "Order quantity must be positive",
    "details": "Quantity value -100 is not allowed",
    "timestamp": "2024-01-01T10:00:00Z",
    "request_id": "req_123"
  }
}
```

### Common Error Codes
- `INVALID_REQUEST` - Malformed request
- `UNAUTHORIZED` - Authentication required
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `INVALID_ORDER` - Order validation failed
- `INSUFFICIENT_FUNDS` - Not enough balance
- `MARKET_CLOSED` - Market is closed
- `COMPLIANCE_FAILURE` - Compliance check failed
- `RATE_LIMIT_EXCEEDED` - Too many requests
- `INTERNAL_ERROR` - Server error

### HTTP Status Codes
- `200` - Success
- `201` - Created
- `400` - Bad Request
- `401` - Unauthorized
- `403` - Forbidden
- `404` - Not Found
- `409` - Conflict
- `422` - Unprocessable Entity
- `429` - Too Many Requests
- `500` - Internal Server Error
- `503` - Service Unavailable

## 🔧 SDK & Client Libraries

### Go SDK
```go
import "github.com/abdoElHodaky/tradSys/sdk/go"

client := tradsys.NewClient(&tradsys.Config{
    BaseURL: "http://localhost:8080",
    APIKey:  "your-api-key",
})

order, err := client.Orders.Create(ctx, &tradsys.OrderRequest{
    Symbol:   "AAPL",
    Side:     "buy",
    Quantity: 100,
    Price:    150.50,
})
```

### Python SDK
```python
from tradsys import TradSysClient

client = TradSysClient(
    base_url="http://localhost:8080",
    api_key="your-api-key"
)

order = client.orders.create(
    symbol="AAPL",
    side="buy",
    quantity=100,
    price=150.50
)
```

### JavaScript SDK
```javascript
import { TradSysClient } from '@tradsys/sdk';

const client = new TradSysClient({
    baseURL: 'http://localhost:8080',
    apiKey: 'your-api-key'
});

const order = await client.orders.create({
    symbol: 'AAPL',
    side: 'buy',
    quantity: 100,
    price: 150.50
});
```

## 📝 Integration Examples

### High-Frequency Trading Bot
```go
package main

import (
    "context"
    "log"
    "time"
    
    "github.com/abdoElHodaky/tradSys/sdk/go"
)

func main() {
    client := tradsys.NewClient(&tradsys.Config{
        BaseURL: "http://localhost:8080",
        APIKey:  "your-api-key",
    })
    
    // Subscribe to market data
    ws, err := client.WebSocket.Connect(ctx)
    if err != nil {
        log.Fatal(err)
    }
    
    ws.Subscribe("market_data", "AAPL")
    
    for msg := range ws.Messages() {
        if msg.Type == "market_data" {
            // Implement trading logic
            if shouldBuy(msg.Data) {
                order, err := client.Orders.Create(ctx, &tradsys.OrderRequest{
                    Symbol:   "AAPL",
                    Side:     "buy",
                    Type:     "market",
                    Quantity: 100,
                })
                if err != nil {
                    log.Printf("Order failed: %v", err)
                }
            }
        }
    }
}
```

### Risk Management Integration
```python
import asyncio
from tradsys import TradSysClient

async def monitor_risk():
    client = TradSysClient(api_key="your-api-key")
    
    while True:
        # Get current risk metrics
        risk = await client.risk.get_metrics("user123")
        
        if risk.risk_level == "high":
            # Close risky positions
            positions = await client.positions.list("user123")
            for position in positions:
                if position.unrealized_pnl < -1000:  # $1000 loss
                    await client.orders.create(
                        symbol=position.symbol,
                        side="sell" if position.quantity > 0 else "buy",
                        quantity=abs(position.quantity),
                        type="market"
                    )
        
        await asyncio.sleep(1)  # Check every second

asyncio.run(monitor_risk())
```

## 🔒 Rate Limiting

### Rate Limits
- **Orders API**: 1000 requests/minute per user
- **Market Data API**: 10000 requests/minute per user
- **WebSocket**: 10000 messages/minute per connection
- **Risk API**: 100 requests/minute per user

### Rate Limit Headers
```http
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 999
X-RateLimit-Reset: 1640995200
```

### Handling Rate Limits
```python
import time
from tradsys import TradSysClient, RateLimitError

client = TradSysClient(api_key="your-api-key")

try:
    order = client.orders.create(...)
except RateLimitError as e:
    # Wait until rate limit resets
    time.sleep(e.retry_after)
    order = client.orders.create(...)
```

---

**Ready to integrate? Use the examples above to get started with the TradSys API!** 🚀
