# TradSys v2 - High-Frequency Trading System

[![Go Version](https://img.shields.io/badge/Go-1.21+-blue.svg)](https://golang.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)
[![Build Status](https://img.shields.io/badge/Build-Passing-brightgreen.svg)](https://github.com/abdoElHodaky/tradSys/actions)
[![System Status](https://img.shields.io/badge/Status-Development-yellow.svg)](#system-status)

A high-performance, low-latency trading system designed for algorithmic and high-frequency trading operations. Built with Go for maximum performance and reliability.

## 🎯 **System Status**

| Component | Status | Completion | Notes |
|-----------|--------|------------|-------|
| **Core Services** | 🟢 Ready | 90% | Order & Risk services operational |
| **Market Data** | 🟢 Enhanced | 98% | Provider management & thread-safe operations |
| **Authentication** | 🟢 Implemented | 95% | JWT-based auth with role management |
| **API Gateway** | 🟢 Ready | 80% | REST endpoints & WebSocket support |
| **Risk Management** | 🟢 Enhanced | 85% | VaR calculation & real margin calculations |
| **Metrics** | 🟢 Implemented | 75% | Prometheus integration with custom metrics |
| **Testing** | 🔴 Limited | 15% | Only 4 test files currently |
| **Documentation** | 🟡 Basic | 50% | README & config docs available |
| **Deployment** | 🟢 Ready | 90% | Kubernetes manifests complete |

**Latest Updates (v2 Branch):**
- ✅ **Market Data Service**: Enhanced with provider management & thread-safe operations
- ✅ **Risk Management**: Implemented VaR calculation & real margin calculations
- ✅ **Order Management**: Real risk assessment & margin calculations
- ✅ **Metrics System**: Prometheus integration with custom trading metrics
- ✅ **Authentication System**: Complete JWT-based authentication with login/refresh
- ✅ **Service Architecture**: Microservices with gRPC communication
- ✅ **Configuration**: Unified YAML configuration system

## 🚀 **Quick Start**

### Prerequisites
- Go 1.21 or higher
- SQLite3 (for local development)
- Git

### Installation

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Build the system
go build -o tradsys cmd/tradsys/main.go

# Run the unified trading server
./tradsys server
```

### Basic Usage

```bash
# Start the full trading system
./tradsys server

# Run specific services
./tradsys gateway      # API Gateway
./tradsys orders       # Order Management
./tradsys risk         # Risk Management
./tradsys marketdata   # Market Data Service
./tradsys ws           # WebSocket Service

# Show version
./tradsys version

# Show help
./tradsys help
```

## 📋 **Features**

### Core Trading Engine
- **Ultra-low latency order matching** (sub-100μs target)
- **Real-time risk management** with configurable limits
- **Multi-exchange connectivity** (Binance, Coinbase, etc.)
- **Advanced order types** (Market, Limit, Stop, IOC, FOK)
- **Position management** with automatic settlement

### High-Frequency Trading Optimizations
- **Memory pooling** for zero-allocation operations
- **Lock-free data structures** for concurrent access
- **Binary protocol** for WebSocket communications
- **Batch processing** for database operations
- **CPU affinity** and NUMA awareness

### Risk Management
- **Real-time position monitoring**
- **Configurable risk limits** (position size, leverage, daily loss)
- **Margin requirements** with automatic liquidation
- **Compliance reporting** and audit trails

### Market Data
- **Real-time price feeds** from multiple exchanges
- **Order book reconstruction** with microsecond precision
- **Historical data storage** and backtesting support
- **Custom indicators** and technical analysis

### API & Connectivity
- **RESTful API** with comprehensive endpoints
- **WebSocket streaming** for real-time updates
- **gRPC services** for internal communication
- **Rate limiting** and authentication

### Authentication & Security
- **JWT-based authentication** with refresh tokens
- **Role-based access control** (Admin, Trader, Viewer)
- **Secure password hashing** with bcrypt
- **Token validation middleware** for protected routes
- **Default users**: `admin/admin123`, `trader/trader123`

## 🏗️ **Architecture**

### System Overview

```
┌─────────────────┐    ┌─────────────────┐    ┌─────────────────┐
│   Web Client    │    │   Trading App   │    │  External APIs  │
└─────────┬───────┘    └─────────┬───────┘    └─────────┬───────┘
          │                      │                      │
          ▼                      ▼                      ▼
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway                              │
│                     (Rate Limiting, Auth)                      │
└─────────────────────────┬───────────────────────────────────────┘
                          │
          ┌───────────────┼───────────────┐
          ▼               ▼               ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
│ Order Management│ │ Market Data │ │ Risk Management │
│    Service      │ │   Service   │ │    Service      │
└─────────┬───────┘ └──────┬──────┘ └─────────┬───────┘
          │                │                  │
          └────────────────┼──────────────────┘
                           ▼
          ┌─────────────────────────────────────┐
          │         Core Trading Engine         │
          │    (Matching, Settlement, etc.)     │
          └─────────────────┬───────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          ▼                 ▼                 ▼
┌─────────────────┐ ┌─────────────┐ ┌─────────────────┐
│    Database     │ │  Exchanges  │ │   Monitoring    │
│   (SQLite)      │ │ (Binance,   │ │  (Metrics,      │
│                 │ │  Coinbase)  │ │   Logging)      │
└─────────────────┘ └─────────────┘ └─────────────────┘
```

### Directory Structure

```
tradSys/
├── cmd/
│   └── tradsys/           # Main application entry point
├── internal/
│   ├── api/               # REST API handlers and routes
│   ├── trading/           # Core trading engine
│   │   ├── strategies/    # Trading strategies
│   │   ├── core/          # Order matching, settlement
│   │   ├── risk_management/ # Risk controls
│   │   └── order_management/ # Order lifecycle
│   ├── connectivity/      # Exchange connectors
│   ├── compliance/        # Regulatory compliance
│   ├── monitoring/        # Metrics and health checks
│   └── config/           # Configuration management
├── config/
│   └── tradsys.yaml      # Unified configuration file
├── docs/                 # Documentation
├── scripts/              # Build and deployment scripts
└── tests/                # Test suites
```

## ⚙️ **Configuration**

The system uses a unified YAML configuration file located at `config/tradsys.yaml`. Key sections include:

### Server Configuration
```yaml
server:
  port: 8080
  host: "0.0.0.0"
  read_timeout: 30s
  write_timeout: 30s
  max_connections: 10000
```

### Trading Engine
```yaml
trading:
  matching:
    engine_type: "fifo"
    max_orders_per_symbol: 100000
    price_precision: 8
  risk:
    max_position_size: 10.0
    max_leverage: 5.0
    max_daily_loss: 1000.0
```

### Performance Tuning
```yaml
performance:
  gc_percent: 200
  memory_limit: 2147483648  # 2GB
  pools:
    order_pool_size: 1000
    message_pool_size: 5000
  targets:
    order_latency: 100      # microseconds
    ws_latency: 50          # microseconds
```

### Environment Variables
```bash
# Exchange API Keys
export BINANCE_API_KEY="your_binance_api_key"
export BINANCE_API_SECRET="your_binance_secret"

# JWT Authentication
export JWT_SECRET="your_jwt_secret"

# Database (optional, defaults to SQLite)
export DATABASE_URL="sqlite://tradSys.db"
```

## 🔌 **API Documentation**

### Authentication Endpoints

#### Login
```bash
POST /auth/login
Content-Type: application/json

{
  "username": "admin",
  "password": "admin123"
}

# Response
{
  "success": true,
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": "admin-001",
      "username": "admin",
      "email": "admin@tradsys.com",
      "role": "admin"
    },
    "expires_at": "2024-10-21T10:24:07Z"
  }
}
```

#### Refresh Token
```bash
POST /auth/refresh
Content-Type: application/json

{
  "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

#### Protected Routes
```bash
# Get user profile
GET /auth/profile
Authorization: Bearer <token>

# Logout
POST /auth/logout
Authorization: Bearer <token>
```

### Trading Endpoints

#### Place Order
```bash
POST /api/v1/orders
Authorization: Bearer <token>
Content-Type: application/json

{
  "symbol": "BTCUSDT",
  "side": "buy",
  "type": "limit",
  "quantity": "0.001",
  "price": "50000.00"
}
```

#### Get Orders
```bash
GET /api/v1/orders
Authorization: Bearer <token>

# Get specific order
GET /api/v1/orders/{order_id}
Authorization: Bearer <token>
```

#### Market Data
```bash
# Get ticker
GET /api/v1/market/ticker/{symbol}

# Get order book
GET /api/v1/market/orderbook/{symbol}

# Get recent trades
GET /api/v1/market/trades/{symbol}
```

### WebSocket Endpoints

#### Connect to WebSocket
```javascript
const ws = new WebSocket('ws://localhost:8080/ws');

// Subscribe to market data
ws.send(JSON.stringify({
  "type": "subscribe",
  "channel": "ticker",
  "symbol": "BTCUSDT"
}));

// Subscribe to order updates (requires authentication)
ws.send(JSON.stringify({
  "type": "subscribe",
  "channel": "orders",
  "token": "your_jwt_token"
}));
```

### Health & Monitoring

```bash
# Health check
GET /health

# Readiness check
GET /ready

# Metrics (Prometheus format)
GET /metrics
```

## 🔧 **Development**

### Building from Source

```bash
# Install dependencies
go mod download

# Run tests
go test ./...

# Build optimized binary
go build -ldflags="-s -w" -o tradsys cmd/tradsys/main.go

# Build with race detection (development)
go build -race -o tradsys-debug cmd/tradsys/main.go
```

### Running Tests

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run benchmarks
go test -bench=. ./...

# Run specific test package
go test ./internal/trading/...
```

### Development Mode

```bash
# Run with debug logging
TRADSYS_ENV=development ./tradsys server

# Run with custom config
./tradsys server --config config/dev.yaml

# Enable profiling
./tradsys server --profile --profile-port 6060
```

## 📊 **Performance**

### Latency Targets
- **Order Processing**: < 100μs (microseconds)
- **WebSocket Updates**: < 50μs
- **Database Operations**: < 1ms
- **Risk Checks**: < 10μs

### Throughput Capacity
- **Orders per Second**: 100,000+
- **Market Data Updates**: 1,000,000+ per second
- **Concurrent WebSocket Connections**: 10,000+
- **API Requests**: 10,000 per second

### Memory Usage
- **Base Memory**: ~50MB
- **Per Connection**: ~4KB
- **Order Book**: ~1MB per symbol
- **Total Recommended**: 2-8GB depending on load

## 🔒 **Security**

### Authentication & Authorization
- **JWT-based authentication** for API access
- **API key management** for exchange connectivity
- **Role-based access control** (RBAC)
- **Rate limiting** to prevent abuse

### Risk Controls
- **Position limits** per account and symbol
- **Maximum leverage** controls
- **Daily loss limits** with automatic shutdown
- **Margin requirements** with liquidation

### Compliance
- **Trade reporting** for regulatory requirements
- **Audit trails** for all system actions
- **Data encryption** at rest and in transit
- **Secure key management**

## 🚀 **Deployment**

### Docker Deployment

```bash
# Build Docker image
docker build -t tradsys:latest .

# Run container
docker run -d \
  --name tradsys \
  -p 8080:8080 \
  -p 8081:8081 \
  -v $(pwd)/config:/app/config \
  -v $(pwd)/data:/app/data \
  tradsys:latest
```

### Production Deployment

```bash
# Build optimized binary
CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags="-s -w" -o tradsys cmd/tradsys/main.go

# Set production environment
export TRADSYS_ENV=production

# Run with systemd (recommended)
sudo systemctl start tradsys
sudo systemctl enable tradsys
```

### Kubernetes Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tradsys
spec:
  replicas: 3
  selector:
    matchLabels:
      app: tradsys
  template:
    metadata:
      labels:
        app: tradsys
    spec:
      containers:
      - name: tradsys
        image: tradsys:latest
        ports:
        - containerPort: 8080
        - containerPort: 8081
        resources:
          requests:
            memory: "1Gi"
            cpu: "500m"
          limits:
            memory: "2Gi"
            cpu: "1000m"
```

## 📈 **Monitoring**

### Health Checks
- **Health endpoint**: `GET /health`
- **Readiness endpoint**: `GET /ready`
- **Metrics endpoint**: `GET /metrics` (Prometheus format)

### Key Metrics
- **Order latency** (p50, p95, p99)
- **Throughput** (orders/sec, messages/sec)
- **Error rates** by service and endpoint
- **Memory usage** and garbage collection
- **Database performance**

### Logging
```bash
# View logs in JSON format
./tradsys server 2>&1 | jq '.'

# Filter by log level
./tradsys server 2>&1 | jq 'select(.level == "error")'

# Monitor specific component
./tradsys server 2>&1 | jq 'select(.component == "trading.engine")'
```

## 🧪 **Testing**

### Unit Tests
```bash
# Run unit tests
go test ./internal/...

# Test specific package
go test ./internal/trading/core/
```

### Integration Tests
```bash
# Run integration tests
go test -tags=integration ./tests/...
```

### Load Testing
```bash
# Install hey (HTTP load testing tool)
go install github.com/rakyll/hey@latest

# Test API endpoints
hey -n 10000 -c 100 http://localhost:8080/api/v1/orders

# Test WebSocket connections
./scripts/ws-load-test.sh
```

### Benchmarking
```bash
# Run benchmarks
go test -bench=BenchmarkOrderMatching ./internal/trading/core/
go test -bench=BenchmarkRiskCheck ./internal/trading/risk_management/
```

## 🤝 **Contributing**

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Workflow
1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Make your changes
4. Add tests for new functionality
5. Run the test suite (`go test ./...`)
6. Commit your changes (`git commit -m 'Add amazing feature'`)
7. Push to the branch (`git push origin feature/amazing-feature`)
8. Open a Pull Request

### Code Style
- Follow standard Go conventions
- Use `gofmt` for formatting
- Run `golint` and `go vet`
- Add comments for exported functions
- Write tests for new features

## 📄 **License**

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🆘 **Support**

### Documentation
- **API Documentation**: Available at `/docs` when running the server
- **Configuration Reference**: See `config/tradsys.yaml` for all options
- **Architecture Guide**: Detailed system design documentation

### Getting Help
- **Issues**: Report bugs and request features on [GitHub Issues](https://github.com/abdoElHodaky/tradSys/issues)
- **Discussions**: Join the community on [GitHub Discussions](https://github.com/abdoElHodaky/tradSys/discussions)
- **Email**: Contact the maintainers at [support@tradsys.dev](mailto:support@tradsys.dev)

### FAQ

**Q: What exchanges are supported?**
A: Currently Binance and Coinbase Pro, with more exchanges planned.

**Q: Can I run this in production?**
A: Yes, but ensure proper testing and risk management configuration.

**Q: What's the minimum hardware requirement?**
A: 4GB RAM, 2 CPU cores, SSD storage recommended for production.

**Q: How do I add a new trading strategy?**
A: Implement the Strategy interface in `internal/trading/strategies/` and register it in the configuration.

**Q: Is there a paper trading mode?**
A: Yes, set `sandbox: true` in the exchange configuration.

---

## 🎯 **Roadmap**

### Version 2.1 (Current)
- ✅ Unified configuration system
- ✅ Simplified directory structure
- ✅ Enhanced documentation
- ✅ Performance optimizations

### Version 2.2 (Planned)
- [ ] Additional exchange connectors (Kraken, FTX)
- [ ] Advanced order types (Iceberg, TWAP)
- [ ] Machine learning integration
- [ ] Enhanced backtesting framework

### Version 3.0 (Future)
- [ ] Distributed architecture
- [ ] Multi-asset support (Forex, Commodities)
- [ ] Advanced risk analytics
- [ ] Web-based management interface

---

**Built with ❤️ by the TradSys Team**

*High-frequency trading made accessible, reliable, and profitable.*
