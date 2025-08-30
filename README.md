# TradSys - Trading System Platform

TradSys is a microservices-based trading system platform built with Go and modern architecture patterns. It provides a robust foundation for building scalable and resilient trading applications with advanced decision support capabilities.

## Features

- **Service Mesh Architecture**: Built with dependency injection and modular design
- **Distributed Trading**: Supports distributed order processing and matching
- **Risk Management**: Integrated risk management capabilities with circuit breakers and bulkheads
- **Market Data Processing**: Real-time and historical market data handling
- **Monitoring**: Built-in support for metrics and tracing
- **Decision Support Integration**: APIs for connecting with external decision support systems
- **Event-Driven Architecture**: CQRS pattern with event sourcing
- **Resilience Patterns**: Circuit breakers, retries, timeouts, and bulkheads

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

- Go 1.22 or later
- PostgreSQL 14 or later
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
- `GET /api/market-data/{symbol}/candles`: Get OHLCV candles for a symbol
- `GET /api/market-data/indicators/{indicator}/{symbol}`: Get technical indicator values

### Risk API

- `GET /api/risk/exposure`: Get current risk exposure
- `POST /api/risk/limits`: Set risk limits
- `GET /api/risk/circuit-breakers`: Get circuit breaker status
- `POST /api/risk/circuit-breakers/reset`: Reset circuit breakers

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
│   ├── architecture/     # Architecture components (fx, resilience)
│   ├── config/           # Configuration
│   ├── db/               # Database models and repositories
│   ├── marketdata/       # Market data processing
│   ├── plugin/           # Plugins (CQRS, event bus)
│   ├── trading/          # Trading logic and order management
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

### Decision Support System API Design

The Decision Support System (DSS) API is designed to be flexible and extensible, allowing integration with various external systems. The API follows RESTful principles and uses JSON for data exchange.

#### Key API Endpoints

1. **Analysis Endpoint**
   - `POST /api/decision-support/analyze`
   - Submits market data, portfolio information, and other parameters for analysis
   - Returns analysis results including recommendations and insights

2. **Recommendation Endpoint**
   - `GET /api/decision-support/recommendations`
   - Retrieves trading recommendations based on current market conditions
   - Supports filtering by instrument, strategy, and confidence level

3. **Scenario Analysis**
   - `POST /api/decision-support/scenarios`
   - Runs what-if scenarios with different market conditions
   - Returns potential outcomes and risk assessments

4. **Backtesting**
   - `POST /api/decision-support/backtest`
   - Tests strategies against historical data
   - Returns performance metrics and optimization suggestions

5. **Portfolio Optimization**
   - `GET /api/decision-support/portfolio/optimize`
   - Provides portfolio optimization recommendations
   - Supports different optimization objectives (risk, return, Sharpe ratio)

6. **Alerts Configuration**
   - `POST /api/decision-support/alerts/configure`
   - Configures alerts based on market conditions or analysis results
   - Supports different notification channels

#### Integration Patterns

The DSS API supports multiple integration patterns:

1. **Synchronous Request-Response**
   - Direct API calls for immediate analysis and recommendations
   - Suitable for user-initiated actions

2. **Asynchronous Processing**
   - Submit analysis jobs that run in the background
   - Receive notifications when analysis is complete
   - Suitable for complex, time-consuming analysis

3. **Event-Driven Integration**
   - Subscribe to events and receive updates when conditions change
   - Suitable for real-time monitoring and alerts

4. **Streaming Data**
   - Continuous stream of recommendations and insights
   - Suitable for algorithmic trading systems

#### Authentication and Security

The DSS API uses OAuth 2.0 for authentication and supports role-based access control. All API requests are encrypted using TLS.

#### Rate Limiting and Quotas

To ensure fair usage and system stability, the API implements rate limiting and usage quotas. These can be configured based on user tiers and subscription levels.

For detailed API documentation, see [Decision Support API](docs/decision-support-api.md).

## License

This project is licensed under the MIT License - see the LICENSE file for details.

