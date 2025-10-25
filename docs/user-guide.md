# TradSys User Guide & Getting Started

## 🚀 Quick Start

TradSys is a high-performance trading system optimized for Middle East exchanges (EGX, ADX) with comprehensive compliance and Islamic finance support.

### Prerequisites

- Go 1.24+ 
- PostgreSQL 13+
- Redis 6+
- Docker & Docker Compose

### Installation

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Install dependencies
go mod download

# Set up environment
cp .env.example .env
# Edit .env with your configuration

# Start services
docker-compose up -d

# Run migrations
go run cmd/migrate/main.go

# Start the trading system
go run cmd/server/main.go
```

## 🏗️ System Architecture Overview

TradSys follows a microservices architecture with the following core components:

### Core Services
- **Matching Engine** (`pkg/matching/`) - High-frequency order matching
- **Order Management** (`internal/orders/`) - Order lifecycle management
- **Risk Management** (`internal/risk/`) - Real-time risk assessment
- **Compliance Engine** (`internal/compliance/`) - Multi-regulatory validation
- **WebSocket Gateway** (`services/websocket/`) - Real-time communication
- **Exchange Integration** (`services/exchange/`) - EGX/ADX connectivity

### Key Features
- **High-Frequency Trading (HFT)** - Sub-millisecond latency
- **Multi-Exchange Support** - EGX, ADX with unified interface
- **Islamic Finance Compliance** - Sharia-compliant trading
- **Real-time Risk Management** - VaR, Greeks, position limits
- **Comprehensive Monitoring** - Prometheus metrics and alerting

## 📊 Trading Operations

### Order Management

#### Placing Orders
```go
orderReq := &orders.OrderRequest{
    UserID:      "user123",
    Symbol:      "AAPL",
    Side:        orders.OrderSideBuy,
    Type:        orders.OrderTypeLimit,
    Quantity:    100,
    Price:       150.50,
    TimeInForce: orders.TimeInForceGTC,
}

order, err := orderService.CreateOrder(ctx, orderReq)
```

#### Order Types Supported
- **Market Orders** - Execute immediately at best available price
- **Limit Orders** - Execute at specified price or better
- **Stop Orders** - Trigger when price reaches stop level
- **Stop-Limit Orders** - Combine stop and limit functionality

#### Order Status Lifecycle
1. `new` - Order created but not yet submitted
2. `pending` - Order submitted to exchange
3. `partially_filled` - Order partially executed
4. `filled` - Order completely executed
5. `cancelled` - Order cancelled by user
6. `rejected` - Order rejected by system/exchange
7. `expired` - Order expired based on time-in-force

### Risk Management

#### Pre-Trade Risk Checks
- Position size limits
- Order value limits
- Concentration risk assessment
- Regulatory compliance validation
- Islamic finance screening (if enabled)

#### Real-Time Monitoring
- Value at Risk (VaR) calculation
- Greeks calculation for options
- Portfolio concentration analysis
- Margin requirement monitoring

### Compliance Features

#### Supported Regulatory Frameworks
- **SEC** (US Securities and Exchange Commission)
- **MiFID** (EU Markets in Financial Instruments Directive)
- **SCA** (UAE Securities and Commodities Authority)
- **ADGM** (Abu Dhabi Global Market)
- **DIFC** (Dubai International Financial Centre)
- **Sharia** (Islamic finance compliance)
- **FATCA** (Foreign Account Tax Compliance Act)
- **EMIR** (European Market Infrastructure Regulation)

#### Islamic Finance Support
- Sharia-compliant instrument screening
- Prohibited sector filtering
- Islamic fund identification
- Sukuk (Islamic bond) support

## 🔌 Exchange Integration

### Supported Exchanges
- **EGX** (Egyptian Exchange)
- **ADX** (Abu Dhabi Securities Exchange)
- **Unified Interface** for future exchange additions

### Connection Management
```go
// Initialize exchange client
client := exchange.NewClient(exchange.Config{
    Exchange: "ADX",
    APIKey:   "your-api-key",
    Secret:   "your-secret",
})

// Connect to exchange
err := client.Connect(ctx)
```

## 🌐 WebSocket Real-Time Data

### Connecting to WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

// Subscribe to market data
ws.send(JSON.stringify({
    type: 'subscribe',
    channel: 'market_data',
    symbol: 'AAPL'
}));

// Handle incoming data
ws.onmessage = function(event) {
    const data = JSON.parse(event.data);
    console.log('Market data:', data);
};
```

### Available Channels
- `market_data` - Real-time price updates
- `order_book` - Order book depth data
- `trades` - Trade execution data
- `order_updates` - Order status changes
- `portfolio` - Portfolio updates
- `alerts` - System alerts and notifications

## 🛡️ Security & Authentication

### API Authentication
```bash
# JWT Token authentication
curl -H "Authorization: Bearer <jwt-token>" \
     -X GET http://localhost:8080/api/v1/orders
```

### User Management
- JWT-based authentication
- Role-based access control (RBAC)
- Multi-factor authentication support
- Session management

## 📈 Performance Optimization

### High-Frequency Trading Features
- Sub-millisecond order matching
- Optimized memory allocation
- Lock-free data structures
- NUMA-aware processing

### Monitoring & Metrics
- Prometheus metrics export
- Grafana dashboards
- Real-time performance monitoring
- Latency tracking and alerting

## 🔧 Configuration

### Environment Variables
```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=tradsys
DB_USER=postgres
DB_PASSWORD=password

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# Exchange APIs
EGX_API_KEY=your-egx-key
ADX_API_KEY=your-adx-key

# WebSocket
WS_PORT=8080
WS_MAX_CONNECTIONS=10000

# Risk Management
RISK_VAR_CONFIDENCE=0.95
RISK_MAX_POSITION_SIZE=1000000

# Compliance
ENABLE_SHARIA_COMPLIANCE=true
ENABLE_SEC_COMPLIANCE=true
```

### Trading Configuration
```yaml
# config/trading.yaml
matching_engine:
  latency_target_ns: 100000  # 100 microseconds
  max_orders_per_second: 100000
  
risk_management:
  enable_pre_trade_checks: true
  enable_real_time_monitoring: true
  var_calculation_interval: 1s
  
compliance:
  enabled_regulations: ["sec", "mifid", "sca", "sharia"]
  auto_reporting: true
  audit_trail: true
```

## 🚨 Troubleshooting

### Common Issues

#### Connection Issues
```bash
# Check service status
docker-compose ps

# View logs
docker-compose logs -f tradsys

# Test database connection
go run cmd/healthcheck/main.go
```

#### Performance Issues
```bash
# Monitor system metrics
curl http://localhost:8080/metrics

# Check WebSocket connections
curl http://localhost:8080/api/v1/websocket/stats

# View order matching performance
curl http://localhost:8080/api/v1/matching/stats
```

#### Compliance Issues
```bash
# Check compliance status
curl http://localhost:8080/api/v1/compliance/status

# View recent violations
curl http://localhost:8080/api/v1/compliance/violations

# Test Sharia compliance
curl -X POST http://localhost:8080/api/v1/compliance/validate \
     -d '{"symbol": "AAPL", "type": "sharia"}'
```

## 📞 Support & Resources

### Documentation
- [API Reference](./api-reference.md)
- [Deployment Guide](./deployment-guide.md)
- [Architecture Guide](./architecture.md)
- [Troubleshooting Guide](./troubleshooting.md)

### Community
- GitHub Issues: Report bugs and feature requests
- Discussions: Community support and questions
- Wiki: Additional documentation and examples

### Professional Support
- Enterprise support available
- Custom exchange integrations
- Performance optimization consulting
- Compliance advisory services

---

**Ready to start trading? Follow the Quick Start guide above and you'll be up and running in minutes!** 🚀
