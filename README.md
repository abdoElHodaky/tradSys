# TradSys - Trading System Platform

TradSys is a microservices-based trading system platform built with Go and go-micro.dev/v4. It provides a robust foundation for building scalable and resilient trading applications with advanced decision support capabilities.

## Features

- **Service Mesh Architecture**: Built on go-micro.dev/v4 for service discovery, load balancing, and resilience
- **Distributed Trading**: Supports distributed order processing and matching
- **Risk Management**: Integrated risk management capabilities
- **Market Data Processing**: Real-time market data handling
- **Monitoring**: Built-in support for metrics and tracing
- **Decision Support Integration**: APIs for connecting with external decision support systems

## Architecture

TradSys is built using a microservices architecture with the following components:

- **Order Service**: Handles order creation, validation, and lifecycle management
- **Matching Service**: Implements order matching algorithms
- **Risk Service**: Provides risk assessment and management
- **Market Data Service**: Processes and distributes market data
- **Monitoring Service**: Collects and exposes metrics
- **Decision Support Service**: Integrates with external decision support systems and provides analytical insights

## Getting Started

### Prerequisites

- Go 1.20 or later
- Docker (optional, for containerized deployment)

### Installation

1. Clone the repository:
   ```bash
   git clone https://github.com/abdoElHodaky/tradSys.git
   cd tradSys
   ```

2. Install dependencies:
   ```bash
   go mod tidy
   ```

3. Build the services:
   ```bash
   go build -o bin/ ./cmd/...
   ```

## Usage

### Running Services

To run a service:

```bash
./bin/[service-name] --config=config/[service-name].yaml
```

### Configuration

Configuration is handled through YAML files in the `config/` directory. Each service has its own configuration file.

## API Design

The system exposes APIs for various trading operations. Here's an overview of the main API endpoints:

### Order API

- `POST /api/orders`: Create a new order
- `GET /api/orders/{id}`: Get order details
- `PUT /api/orders/{id}`: Update an order
- `DELETE /api/orders/{id}`: Cancel an order

### Market Data API

- `GET /api/market-data/{symbol}`: Get latest market data for a symbol
- `GET /api/market-data/{symbol}/history`: Get historical market data

### Risk API

- `GET /api/risk/exposure`: Get current risk exposure
- `POST /api/risk/limits`: Set risk limits

### Decision Support API

- `POST /api/decision-support/analyze`: Submit data for analysis
- `GET /api/decision-support/recommendations`: Get trading recommendations
- `GET /api/decision-support/scenarios`: Get scenario analysis results
- `POST /api/decision-support/backtest`: Run backtest with specified parameters
- `GET /api/decision-support/insights/{symbol}`: Get market insights for a symbol
- `GET /api/decision-support/portfolio/optimize`: Get portfolio optimization recommendations
- `POST /api/decision-support/alerts/configure`: Configure decision support alerts
- `GET /api/decision-support/alerts`: Get current decision support alerts

## Development

### Project Structure

```
tradSys/
├── cmd/                  # Service entry points
│   ├── orders/           # Order service
│   ├── marketdata/       # Market data service
│   ├── risk/             # Risk service
│   └── decisionsupport/  # Decision support service
├── internal/             # Internal packages
│   ├── config/           # Configuration
│   ├── micro/            # Service mesh utilities
│   ├── trading/          # Trading logic
│   ├── risk/             # Risk management
│   └── decisionsupport/  # Decision support logic
├── proto/                # Protocol buffers
│   ├── orders/           # Order service definitions
│   ├── marketdata/       # Market data service definitions
│   ├── risk/             # Risk service definitions
│   └── decisionsupport/  # Decision support service definitions
└── examples/             # Example applications
```

### Adding a New Service

1. Create a new directory in `cmd/`
2. Define service interfaces in `proto/`
3. Implement the service in `internal/`
4. Update configuration in `config/`

## Decision Support System Integration

TradSys provides comprehensive integration with external decision support systems through a dedicated API. This integration allows for:

- Real-time trading recommendations
- Market insights and analysis
- Portfolio optimization
- Scenario analysis and backtesting
- Risk assessment and alerts

For detailed API documentation, see [Decision Support API](docs/decision-support-api.md).

## License

This project is licensed under the MIT License - see the LICENSE file for details.
