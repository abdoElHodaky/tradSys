# 🛣️ TradSys v3 Routing Guide

## 📋 **ROUTING OVERVIEW**

TradSys v3 implements a comprehensive routing system with organized endpoints for different functional areas.

### **Base Structure**
```
/                           # Root endpoints
├── /health                 # System health check
├── /ready                  # Readiness probe
├── /metrics                # System metrics
├── /api-docs               # API documentation index
├── /swagger                # Swagger YAML redirect
├── /docs/                  # Static documentation files
└── /api/v1/                # Main API endpoints
```

---

## 🏥 **SYSTEM ENDPOINTS**

### **Health Check**
```http
GET /health
```
**Description**: System health status and basic metrics  
**Response**: System status, version, timestamp

### **Readiness Check**
```http
GET /ready
```
**Description**: Component readiness status  
**Response**: Status of core, connectivity, compliance, strategies, websocket, risk

### **System Metrics**
```http
GET /metrics
```
**Description**: Comprehensive system performance metrics  
**Response**: Service info, version, timestamp, performance metrics

### **API Documentation**
```http
GET /api-docs
```
**Description**: API documentation index with endpoint listing  
**Response**: Documentation metadata and endpoint directory

---

## 📊 **API v1 ENDPOINTS**

### **Order Management** (`/api/v1/orders`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/orders` | Create new trading order |
| `GET` | `/orders` | List orders with filtering |
| `GET` | `/orders/{id}` | Get specific order by ID |
| `DELETE` | `/orders/{id}` | Cancel existing order |

**Query Parameters**:
- `limit`: Maximum number of results (1-1000, default: 50)
- `symbol`: Filter by trading symbol
- `status`: Filter by order status (pending, filled, cancelled, rejected)

### **Trade Management** (`/api/v1/trades`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/trades` | List executed trades |
| `GET` | `/trades/{symbol}` | Get trades for specific symbol |

**Query Parameters**:
- `limit`: Maximum number of results (1-1000, default: 50)
- `symbol`: Filter by trading symbol

### **Position Management** (`/api/v1/positions`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/positions` | List current positions |
| `GET` | `/positions/{symbol}` | Get position for specific symbol |

### **Market Data** (`/api/v1/market`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/market/orderbook/{symbol}` | Get order book for symbol |
| `GET` | `/market/ticker/{symbol}` | Get ticker data for symbol |
| `GET` | `/market/symbols` | List available trading symbols |

**Query Parameters**:
- `depth`: Order book depth (1-100, default: 20)

### **Performance Metrics** (`/api/v1/metrics`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `GET` | `/metrics` | System performance metrics |
| `GET` | `/metrics/performance` | Detailed performance data |

---

## 🏦 **BOND ENDPOINTS** (`/api/v1/bonds`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/bonds` | Create new bond |
| `GET` | `/bonds/{symbol}/metrics` | Get bond metrics |
| `GET` | `/bonds/yield-curve` | Get yield curve data |
| `GET` | `/bonds/{symbol}/credit-risk` | Assess credit risk |
| `GET` | `/bonds/{symbol}/duration` | Get duration and convexity |
| `GET` | `/bonds/{symbol}/cash-flows` | Project cash flows |
| `POST` | `/bonds/{symbol}/rating-change` | Update credit rating |
| `GET` | `/bonds/calculate-ytm` | Calculate yield to maturity |
| `GET` | `/bonds/calculate-duration` | Calculate duration |
| `GET` | `/bonds/by-rating` | Get bonds by rating |
| `GET` | `/bonds/maturity-schedule` | Get maturity schedule |

---

## 📈 **ETF ENDPOINTS** (`/api/v1/etfs`)

| Method | Endpoint | Description |
|--------|----------|-------------|
| `POST` | `/etfs` | Create new ETF |
| `GET` | `/etfs/{symbol}/metrics` | Get ETF metrics |
| `POST` | `/etfs/{symbol}/metrics` | Update ETF metrics |
| `GET` | `/etfs/{symbol}/tracking-error` | Get tracking error |
| `POST` | `/etfs/{symbol}/creation-redemption` | Process creation/redemption |
| `GET` | `/etfs/{symbol}/holdings` | Get ETF holdings |
| `GET` | `/etfs/validate-order` | Validate ETF order |
| `GET` | `/etfs/{symbol}/liquidity` | Get liquidity metrics |
| `POST` | `/etfs/{symbol}/rebalance` | Trigger rebalance |

---

## 🔌 **WEBSOCKET ENDPOINTS**

### **WebSocket Gateway**
```
ws://localhost:8080/ws
```

**Supported Message Types**:
- Market data subscriptions
- Order updates
- Trade notifications
- Risk alerts
- System notifications

**Connection Flow**:
1. Establish WebSocket connection
2. Send authentication message
3. Subscribe to desired channels
4. Receive real-time updates

---

## 🔐 **AUTHENTICATION FLOW**

### **JWT Authentication**
```http
Authorization: Bearer <jwt_token>
```

### **API Key Authentication**
```http
X-API-Key: <api_key>
```

### **Authentication Endpoints**
```http
POST /api/v1/auth/login     # User login
POST /api/v1/auth/refresh   # Token refresh
POST /api/v1/auth/logout    # User logout
```

---

## 📝 **REQUEST/RESPONSE FORMATS**

### **Standard Request Headers**
```http
Content-Type: application/json
Authorization: Bearer <token>
X-API-Key: <api_key>
Accept: application/json
```

### **Standard Response Format**
```json
{
  "status": "success|error",
  "data": { ... },
  "message": "Optional message",
  "timestamp": 1640995200,
  "request_id": "req_123456"
}
```

### **Error Response Format**
```json
{
  "error": "Error message",
  "code": "ERROR_CODE",
  "timestamp": 1640995200,
  "request_id": "req_123456"
}
```

---

## 🚀 **PERFORMANCE CONSIDERATIONS**

### **Rate Limiting**
- **Standard endpoints**: 1000 requests/minute
- **Market data**: 5000 requests/minute  
- **Order operations**: 10000 requests/minute
- **WebSocket**: No rate limiting

### **Caching**
- Market data: 100ms cache
- Order book: Real-time
- Positions: 1s cache
- Metrics: 5s cache

### **Pagination**
- Default page size: 50
- Maximum page size: 1000
- Use `limit` and `offset` parameters

---

## 🔧 **DEVELOPMENT TOOLS**

### **Testing Endpoints**
```bash
# Health check
curl http://localhost:8080/health

# API documentation
curl http://localhost:8080/api-docs

# Create order
curl -X POST http://localhost:8080/api/v1/orders \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{"symbol":"BTC-USD","side":"buy","type":"limit","quantity":1.0,"price":50000}'
```

### **WebSocket Testing**
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');
ws.onopen = () => {
  ws.send(JSON.stringify({
    type: 'auth',
    token: 'your-jwt-token'
  }));
};
```

---

## 📊 **MONITORING & OBSERVABILITY**

### **Health Monitoring**
- `/health` - Basic health check
- `/ready` - Kubernetes readiness probe
- `/metrics` - Prometheus metrics

### **Logging**
- Request/response logging
- Error tracking
- Performance monitoring
- Security audit logs

### **Tracing**
- Distributed tracing support
- Request correlation IDs
- Performance profiling
- Dependency tracking

---

*Routing Guide - TradSys v3 | Architecture Phase 16-17 Complete | 90% Optimization Achieved*
