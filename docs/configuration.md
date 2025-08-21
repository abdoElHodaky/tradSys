# TradSys Configuration Guide

This guide explains how to configure the TradSys trading system.

## Configuration Overview

TradSys uses a flexible configuration system that supports:

- Configuration files (YAML, JSON, TOML)
- Environment variables
- Command-line flags

Configuration is loaded in the following order, with later sources overriding earlier ones:

1. Default values
2. Configuration file
3. Environment variables
4. Command-line flags

## Configuration File

TradSys supports the following configuration file formats:

- YAML (`.yaml`, `.yml`)
- JSON (`.json`)
- TOML (`.toml`)

The configuration file is specified using the `--config` flag:

```bash
./tradsys --config=config.yaml
```

If no configuration file is specified, TradSys looks for a file named `config.yaml` in the following locations:

1. The current directory
2. `$HOME/.tradsys/`
3. `/etc/tradsys/`

### Configuration File Structure

The configuration file is structured into sections, each containing related settings:

```yaml
# General configuration
general:
  environment: development
  log_level: info

# Database configuration
database:
  driver: postgres
  host: localhost
  port: 5432
  username: tradsys
  password: password
  database: tradsys

# ... other sections
```

## Environment Variables

Environment variables can be used to override configuration values. The naming convention is:

```
TRADSYS_SECTION_KEY=value
```

For example:

```bash
TRADSYS_GENERAL_ENVIRONMENT=production
TRADSYS_DATABASE_HOST=db.example.com
TRADSYS_DATABASE_PASSWORD=secret
```

For nested configuration values, use underscores to separate levels:

```bash
TRADSYS_TRADING_ENGINE_MAX_ORDERS_PER_SECOND=2000
```

## Command-Line Flags

Command-line flags can be used to override configuration values. The naming convention is:

```
--section.key=value
```

For example:

```bash
--general.environment=production
--database.host=db.example.com
--database.password=secret
```

For nested configuration values, use dots to separate levels:

```bash
--trading.engine.max_orders_per_second=2000
```

## Configuration Sections

### General Configuration

The `general` section contains general settings for the application:

```yaml
general:
  # Application environment (development, staging, production)
  environment: development
  
  # Log level (debug, info, warn, error)
  log_level: info
  
  # Enable metrics collection
  metrics_enabled: true
  
  # Enable tracing
  tracing_enabled: true
  
  # Application name
  app_name: tradsys
  
  # Application version
  app_version: 1.0.0
```

### Database Configuration

The `database` section contains settings for the database connection:

```yaml
database:
  # Database driver (postgres, mysql, sqlite)
  driver: postgres
  
  # Database host
  host: localhost
  
  # Database port
  port: 5432
  
  # Database username
  username: tradsys
  
  # Database password
  password: password
  
  # Database name
  database: tradsys
  
  # SSL mode (disable, require, verify-ca, verify-full)
  ssl_mode: disable
  
  # Maximum number of open connections
  max_open_conns: 20
  
  # Maximum number of idle connections
  max_idle_conns: 5
  
  # Connection maximum lifetime
  conn_max_lifetime: 1h
```

### Message Broker Configuration

The `message_broker` section contains settings for the message broker:

```yaml
message_broker:
  # Message broker type (nats, kafka, rabbitmq)
  type: nats
  
  # Message broker URL
  url: nats://localhost:4222
  
  # Message broker username
  username: tradsys
  
  # Message broker password
  password: password
  
  # Cluster ID (for NATS Streaming)
  cluster_id: tradsys-cluster
  
  # Client ID (for NATS Streaming)
  client_id: tradsys-client
  
  # Maximum number of reconnect attempts
  max_reconnects: 10
  
  # Reconnect wait time
  reconnect_wait: 5s
```

### Market Data Configuration

The `market_data` section contains settings for market data sources:

```yaml
market_data:
  # Market data sources
  sources:
    # Binance WebSocket
    - name: binance
      type: websocket
      url: wss://stream.binance.com:9443/ws
      symbols:
        - BTCUSDT
        - ETHUSDT
      channels:
        - trade
        - kline_1m
        - depth
      rate_limit: 20
    
    # Coinbase WebSocket
    - name: coinbase
      type: websocket
      url: wss://ws-feed.pro.coinbase.com
      symbols:
        - BTC-USD
        - ETH-USD
      channels:
        - matches
        - level2
      rate_limit: 10
  
  # Market data aggregation
  aggregation:
    # Enable OHLCV aggregation
    ohlcv_enabled: true
    
    # OHLCV timeframes
    ohlcv_timeframes:
      - 1m
      - 5m
      - 15m
      - 1h
      - 4h
      - 1d
    
    # Maximum number of historical candles to keep
    max_historical_candles: 1000
```

### Trading Configuration

The `trading` section contains settings for the trading engine:

```yaml
trading:
  # Order matching engine
  engine:
    # Matching algorithm (price_time, pro_rata)
    matching_algorithm: price_time
    
    # Maximum number of orders per second
    max_orders_per_second: 1000
    
    # Maximum number of trades per second
    max_trades_per_second: 500
  
  # Risk management
  risk:
    # Enable position limit
    position_limit_enabled: true
    
    # Maximum position value
    max_position_value: 100000
    
    # Maximum drawdown
    max_drawdown: 0.1
    
    # Enable circuit breaker
    circuit_breaker_enabled: true
    
    # Circuit breaker threshold
    circuit_breaker_threshold: 0.05
  
  # Order execution
  execution:
    # Enable order validation
    validation_enabled: true
    
    # Enable order logging
    logging_enabled: true
    
    # Enable order metrics
    metrics_enabled: true
```

### Strategy Configuration

The `strategies` section contains settings for trading strategies:

```yaml
strategies:
  # Mean reversion strategy for BTC
  - name: mean_reversion_btc
    type: mean_reversion
    enabled: true
    symbols:
      - BTC-USD
    parameters:
      lookback_period: 20
      update_interval: 5
      std_dev_period: 20
      entry_threshold: 2.0
      exit_threshold: 0.5
  
  # Trend following strategy for ETH
  - name: trend_following_eth
    type: trend_following
    enabled: false
    symbols:
      - ETH-USD
    parameters:
      fast_period: 10
      slow_period: 30
      signal_period: 9
```

### API Configuration

The `api` section contains settings for the API:

```yaml
api:
  # HTTP API
  http:
    # Enable HTTP API
    enabled: true
    
    # HTTP host
    host: 0.0.0.0
    
    # HTTP port
    port: 8080
    
    # Enable CORS
    cors_enabled: true
    
    # CORS allowed origins
    cors_allowed_origins:
      - http://localhost:3000
    
    # Enable rate limiting
    rate_limit_enabled: true
    
    # Rate limit (requests per second)
    rate_limit: 100
    
    # Request timeout
    timeout: 30s
  
  # WebSocket API
  websocket:
    # Enable WebSocket API
    enabled: true
    
    # WebSocket host
    host: 0.0.0.0
    
    # WebSocket port
    port: 8081
    
    # WebSocket path
    path: /ws
    
    # Maximum number of connections
    max_connections: 1000
    
    # Read buffer size
    read_buffer_size: 1024
    
    # Write buffer size
    write_buffer_size: 1024
    
    # Ping interval
    ping_interval: 30s
    
    # Pong wait time
    pong_wait: 60s
```

### Monitoring Configuration

The `monitoring` section contains settings for monitoring:

```yaml
monitoring:
  # Prometheus metrics
  prometheus:
    # Enable Prometheus metrics
    enabled: true
    
    # Prometheus host
    host: 0.0.0.0
    
    # Prometheus port
    port: 9090
    
    # Prometheus path
    path: /metrics
  
  # Jaeger tracing
  jaeger:
    # Enable Jaeger tracing
    enabled: true
    
    # Jaeger agent host
    agent_host: localhost
    
    # Jaeger agent port
    agent_port: 6831
    
    # Jaeger service name
    service_name: tradsys
```

## Configuration Validation

TradSys validates the configuration at startup and reports any errors. If the configuration is invalid, TradSys will not start.

Example validation error:

```
Error: invalid configuration: database.port must be a positive integer
```

## Configuration Reloading

TradSys supports configuration reloading without restarting the application. Send a SIGHUP signal to the process to reload the configuration:

```bash
kill -HUP $(pgrep tradsys)
```

Only certain configuration values can be reloaded without restarting the application. These include:

- Log level
- Rate limits
- Circuit breaker settings
- Strategy parameters

## Configuration Best Practices

### Use Environment Variables for Secrets

Use environment variables for sensitive information such as passwords and API keys:

```yaml
database:
  password: ${DB_PASSWORD}
```

### Use Different Configurations for Different Environments

Use different configuration files for different environments:

- `config.development.yaml`
- `config.staging.yaml`
- `config.production.yaml`

### Use Configuration Validation

Validate your configuration before deploying:

```bash
./tradsys --validate-config --config=config.yaml
```

### Document Your Configuration

Document your configuration choices and the reasoning behind them.

### Version Control Your Configuration

Store your configuration files in version control, but exclude files containing secrets.

## Configuration Examples

### Development Configuration

```yaml
general:
  environment: development
  log_level: debug

database:
  driver: sqlite
  database: tradsys.db

message_broker:
  type: nats
  url: nats://localhost:4222

market_data:
  sources:
    - name: binance
      type: websocket
      url: wss://stream.binance.com:9443/ws
      symbols:
        - BTCUSDT
      channels:
        - trade
      rate_limit: 20

strategies:
  - name: mean_reversion_btc
    type: mean_reversion
    enabled: true
    symbols:
      - BTCUSDT
    parameters:
      lookback_period: 20
      update_interval: 5
      std_dev_period: 20
      entry_threshold: 2.0
      exit_threshold: 0.5

api:
  http:
    enabled: true
    host: 0.0.0.0
    port: 8080
  websocket:
    enabled: true
    host: 0.0.0.0
    port: 8081

monitoring:
  prometheus:
    enabled: true
    host: 0.0.0.0
    port: 9090
  jaeger:
    enabled: false
```

### Production Configuration

```yaml
general:
  environment: production
  log_level: info

database:
  driver: postgres
  host: db.example.com
  port: 5432
  username: tradsys
  password: ${DB_PASSWORD}
  database: tradsys
  ssl_mode: require
  max_open_conns: 20
  max_idle_conns: 5
  conn_max_lifetime: 1h

message_broker:
  type: nats
  url: nats://nats.example.com:4222
  username: ${NATS_USERNAME}
  password: ${NATS_PASSWORD}
  cluster_id: tradsys-cluster
  client_id: tradsys-client
  max_reconnects: 10
  reconnect_wait: 5s

market_data:
  sources:
    - name: binance
      type: websocket
      url: wss://stream.binance.com:9443/ws
      symbols:
        - BTCUSDT
        - ETHUSDT
      channels:
        - trade
        - kline_1m
        - depth
      rate_limit: 20
    - name: coinbase
      type: websocket
      url: wss://ws-feed.pro.coinbase.com
      symbols:
        - BTC-USD
        - ETH-USD
      channels:
        - matches
        - level2
      rate_limit: 10

strategies:
  - name: mean_reversion_btc
    type: mean_reversion
    enabled: true
    symbols:
      - BTC-USD
    parameters:
      lookback_period: 20
      update_interval: 5
      std_dev_period: 20
      entry_threshold: 2.0
      exit_threshold: 0.5
  - name: trend_following_eth
    type: trend_following
    enabled: true
    symbols:
      - ETH-USD
    parameters:
      fast_period: 10
      slow_period: 30
      signal_period: 9

api:
  http:
    enabled: true
    host: 0.0.0.0
    port: 8080
    cors_enabled: true
    cors_allowed_origins:
      - https://app.example.com
    rate_limit_enabled: true
    rate_limit: 100
    timeout: 30s
  websocket:
    enabled: true
    host: 0.0.0.0
    port: 8081
    path: /ws
    max_connections: 1000
    read_buffer_size: 1024
    write_buffer_size: 1024
    ping_interval: 30s
    pong_wait: 60s

monitoring:
  prometheus:
    enabled: true
    host: 0.0.0.0
    port: 9090
    path: /metrics
  jaeger:
    enabled: true
    agent_host: jaeger.example.com
    agent_port: 6831
    service_name: tradsys
```

## Conclusion

This guide has covered the configuration of TradSys. By following these guidelines, you can configure TradSys to meet your specific requirements.

