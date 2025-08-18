# TradSys - High-Frequency Trading Platform

TradSys is a high-performance trading platform designed for high-frequency trading (HFT) applications. It provides a robust architecture for market data processing, order management, risk control, and strategy execution.

## Architecture

The platform is built with a layered architecture:

```
┌─────────────────────────────────────────────────────────────────┐
│                      Client Applications                         │
└───────────────┬─────────────────────────────┬───────────────────┘
                │                             │
                ▼                             ▼
┌───────────────────────────┐   ┌───────────────────────────┐
│    REST API (Gin)         │   │    WebSocket Server       │
│  (Admin/Configuration)    │   │  (Market Data Streaming)  │
└───────────────┬───────────┘   └───────────────┬───────────┘
                │                               │
                ▼                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                        API Gateway Layer                         │
└───────────────────────────────┬───────────────────────────────┘
                                │
                                ▼
┌─────────────────────────────────────────────────────────────────┐
│                      gRPC Service Mesh                           │
│  (Internal Service-to-Service Communication with Protocol Buffers)│
└───────────┬───────────────┬────────────────┬──────────────────┘
            │               │                │
            ▼               ▼                ▼
┌───────────────────┐ ┌─────────────────┐ ┌────────────────────┐
│  Market Data      │ │ Order Management │ │ Risk Management    │
│  Service          │ │ System           │ │ System             │
└─────────┬─────────┘ └────────┬────────┘ └──────────┬─────────┘
          │                    │                     │
          │                    ▼                     │
          │           ┌─────────────────┐           │
          └──────────►│ Strategy Engine │◄──────────┘
                      └────────┬────────┘
                               │
                               ▼
┌─────────────────────────────────────────────────────────────────┐
│                      Persistence Layer                           │
│   (SQLite3 with Query Builder and Optimization)                  │
└─────────────────────────────────────────────────────────────────┘
```

## Key Features

- **High Performance**: Optimized for low-latency trading operations
- **Scalable Architecture**: Microservices design with gRPC communication
- **Real-time Data**: WebSocket streaming for market data and order updates
- **Binary Protocol**: Protocol Buffers for efficient binary serialization
- **P2P Communication**: PeerJS integration for peer-to-peer trading signals
- **Risk Management**: Pre-trade risk checks and circuit breakers
- **Strategy Framework**: Pluggable trading strategies with backtesting
- **Database Optimization**: Query builder with performance optimizations
- **Monitoring & Alerting**: Prometheus metrics and real-time alerting system

## Technology Stack

- **Go**: Core programming language
- **Gin**: HTTP framework for REST API
- **gRPC**: High-performance RPC framework
- **Protocol Buffers**: Efficient binary serialization
- **WebSockets**: Real-time data streaming
- **SQLite**: Embedded database with optimizations
- **GORM**: ORM for database operations
- **Zap**: High-performance logging
- **Prometheus**: Metrics and monitoring

## Getting Started

### Prerequisites

- Go 1.21 or higher
- Protocol Buffers compiler (protoc)
- Git

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/abdoElHodaky/tradSys.git
   cd tradSys
   ```

2. Install dependencies:
   ```bash
   go mod download
   ```

3. Generate Protocol Buffer code:
   ```bash
   # Install protoc-gen-go and protoc-gen-go-grpc if not already installed
   go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
   go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

   # Generate code
   protoc --go_out=. --go-grpc_out=. proto/marketdata/marketdata.proto
   protoc --go_out=. --go-grpc_out=. proto/orders/orders.proto
   protoc --go_out=. --go-grpc_out=. proto/risk/risk.proto
   protoc --go_out=. --go-grpc_out=. proto/ws/message.proto
   ```

4. Build the application:
   ```bash
   go build -o tradesys cmd/server/main.go
   ```

5. Run the server:
   ```bash
   ./tradesys
   ```

## Usage

### REST API

The REST API is available at `http://localhost:8080` and provides endpoints for:

- System health check: `GET /health`
- Metrics: `GET /metrics`
- API documentation: `GET /swagger/index.html` (when implemented)

### WebSocket API

The platform provides multiple WebSocket endpoints:

- Legacy WebSocket: `ws://localhost:8080/ws` - JSON-based messaging
- Enhanced WebSocket: `ws://localhost:8080/ws/v2` - Binary Protocol Buffers messaging with compression
- PeerJS Signaling: `ws://localhost:8080/peerjs/ws` - WebRTC signaling for peer-to-peer communication

### PeerJS API

The PeerJS API is available at:

- Signaling server: `ws://localhost:8080/peerjs/ws`
- Stats endpoint: `http://localhost:8080/peerjs/stats`

### gRPC Services

The gRPC server is available at `localhost:50051` and provides the following services:

- MarketDataService: For market data operations
- OrderService: For order management
- RiskService: For risk management
- StrategyService: For strategy management and backtesting

### Strategy Framework

The platform includes a strategy framework for implementing trading strategies:

```go
// Create a new market making strategy
strategy := strategy.NewMarketMakingStrategy(
    "BTC-USD-MarketMaker",
    logger,
    "BTC-USD",
    10.0,  // 10 basis points spread
    1.0,   // 1 BTC quantity
    5.0,   // 5 BTC max position
    time.Second*30,
    orderService,
)

// Register with strategy manager
strategyManager.RegisterStrategy(strategy)

// Start the strategy
strategyManager.StartStrategy(context.Background(), strategy.GetName())
```

### Backtesting

The platform includes a backtesting engine for testing strategies:

```go
// Create a backtest engine
backtestEngine := strategy.NewBacktestEngine(logger)

// Register strategy
backtestEngine.RegisterStrategy(strategy)

// Load market data
backtestEngine.LoadMarketData(marketData)

// Run backtest
result, err := backtestEngine.RunBacktest(
    context.Background(),
    strategy.GetName(),
    10000.0, // Initial capital
)
```

## Development

### Project Structure

```
├── api/                  # REST API handlers
├── cmd/                  # Application entry points
│   └── server/           # Main server
├── docs/                 # Documentation
│   └── architecture.md   # Architecture documentation
├── internal/             # Internal packages
│   ├── api/              # API handlers
│   │   └── handlers/     # API endpoint handlers
│   ├── db/               # Database layer
│   │   ├── models/       # Database models
│   │   ├── query/        # Query builder and optimizer
│   │   └── repositories/ # Data access layer
│   ├── marketdata/       # Market data service
│   ├── monitoring/       # Monitoring and alerting
│   │   ├── metrics.go    # Prometheus metrics collection
│   │   └── alerts.go     # Alerting system
│   ├── orders/           # Order management service
│   ├── peerjs/           # PeerJS integration
│   │   ├── server.go     # PeerJS signaling server
│   │   └── client.go     # PeerJS client implementation
│   ├── risk/             # Risk management service
│   ├── strategy/         # Strategy engine
│   │   ├── framework.go  # Strategy framework
│   │   ├── market_making.go # Market making strategy
│   │   └── backtest.go   # Backtesting engine
│   └── ws/               # WebSocket server
│       ├── server.go     # Legacy WebSocket server
│       ├── enhanced_server.go # Enhanced WebSocket server
│       ├── connection_pool.go # Connection pooling
│       └── binary_message.go # Binary message handling
└── proto/                # Protocol Buffer definitions
    ├── marketdata/       # Market data service definitions
    ├── orders/           # Order service definitions
    ├── risk/             # Risk service definitions
    └── ws/               # WebSocket message definitions
```

## Performance Considerations

TradSys is optimized for high-frequency trading with:

1. **Memory Management**
   - Object pooling to minimize GC pressure
   - Zero-allocation techniques for hot paths

2. **Database Optimization**
   - Query optimization with execution plan analysis
   - Index management for common query patterns
   - Asynchronous persistence for critical paths

3. **Network Optimization**
   - Binary protocols for all internal communication
   - Message batching and compression

4. **Concurrency Model**
   - Go's goroutines and channels for parallel processing
   - Lock-free data structures where possible

## Implementation Phases

The platform has been implemented in four phases:

### Phase 1: Core Infrastructure
- Enhanced WebSocket server with binary messaging
- Connection pooling and optimization
- Market data service
- Order management system
- Risk management system

### Phase 2: P2P Communication
- PeerJS signaling server
- WebRTC-based peer-to-peer communication
- Client-side PeerJS implementation

### Phase 3: Strategy & Monitoring
- Strategy framework with pluggable strategies
- Market making strategy implementation
- Backtesting engine for strategy testing
- Prometheus metrics collection
- Real-time alerting system

### Phase 3.1: Bug Fixes and Improvements
- Fixed missing package imports
- Added Protocol Buffers code generation script
- Resolved dependency version conflicts
- Improved error handling
- Enhanced documentation
- Improved build process reliability

### Phase 3.2: Syntax Fixes and Missing Components
- Added missing Message struct in PeerJS client
- Implemented comprehensive configuration management
- Added JWT-based authentication and authorization
- Added unit tests for authentication
- Improved error handling
- Enhanced documentation

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.
