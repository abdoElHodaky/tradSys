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
- **Risk Management**: Pre-trade risk checks and circuit breakers
- **Strategy Framework**: Pluggable trading strategies
- **Database Optimization**: Query builder with performance optimizations
- **Monitoring**: Prometheus metrics for system health

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

Connect to the WebSocket server at `ws://localhost:8080/ws` to receive real-time market data and order updates.

### gRPC Services

The gRPC server is available at `localhost:50051` and provides the following services:

- MarketDataService: For market data operations
- OrderService: For order management
- RiskService: For risk management

## Development

### Project Structure

```
├── api/                  # REST API handlers
├── cmd/                  # Application entry points
│   └── server/           # Main server
├── internal/             # Internal packages
│   ├── db/               # Database layer
│   │   ├── models/       # Database models
│   │   ├── query/        # Query builder and optimizer
│   │   └── repositories/ # Data access layer
│   ├── marketdata/       # Market data service
│   ├── orders/           # Order management service
│   ├── risk/             # Risk management service
│   ├── strategy/         # Strategy engine
│   └── ws/               # WebSocket server
└── proto/                # Protocol Buffer definitions
    ├── marketdata/       # Market data service definitions
    ├── orders/           # Order service definitions
    └── risk/             # Risk service definitions
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

## License

This project is licensed under the MIT License - see the LICENSE file for details.

## Contributing

Contributions are welcome! Please feel free to submit a Pull Request.

