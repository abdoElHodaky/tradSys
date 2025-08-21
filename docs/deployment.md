# TradSys Deployment Guide

This guide explains how to deploy the TradSys trading system in various environments.

## Prerequisites

Before deploying TradSys, ensure you have the following:

- Go 1.18 or later
- Git
- Docker (for containerized deployment)
- Kubernetes (for orchestrated deployment)
- Access to required external services (databases, message brokers, etc.)

## Configuration

TradSys uses a flexible configuration system that supports:

- Configuration files (YAML, JSON, TOML)
- Environment variables
- Command-line flags

### Configuration File

Create a configuration file (e.g., `config.yaml`) with the following structure:

```yaml
# General configuration
general:
  environment: production  # development, staging, production
  log_level: info  # debug, info, warn, error
  metrics_enabled: true

# Database configuration
database:
  driver: postgres
  host: localhost
  port: 5432
  username: tradsys
  password: ${DB_PASSWORD}  # Use environment variable
  database: tradsys
  ssl_mode: require
  max_open_conns: 20
  max_idle_conns: 5
  conn_max_lifetime: 1h

# Message broker configuration
message_broker:
  type: nats  # nats, kafka, rabbitmq
  url: nats://localhost:4222
  username: ${NATS_USERNAME}
  password: ${NATS_PASSWORD}
  cluster_id: tradsys-cluster
  client_id: tradsys-client
  max_reconnects: 10
  reconnect_wait: 5s

# Market data configuration
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

# Trading configuration
trading:
  engine:
    matching_algorithm: price_time
    max_orders_per_second: 1000
    max_trades_per_second: 500
  risk:
    position_limit_enabled: true
    max_position_value: 100000
    max_drawdown: 0.1
    circuit_breaker_enabled: true
    circuit_breaker_threshold: 0.05

# Strategy configuration
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
    enabled: false
    symbols:
      - ETH-USD
    parameters:
      fast_period: 10
      slow_period: 30
      signal_period: 9

# API configuration
api:
  http:
    enabled: true
    host: 0.0.0.0
    port: 8080
    cors_enabled: true
    cors_allowed_origins:
      - http://localhost:3000
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

# Monitoring configuration
monitoring:
  prometheus:
    enabled: true
    host: 0.0.0.0
    port: 9090
    path: /metrics
  jaeger:
    enabled: true
    agent_host: localhost
    agent_port: 6831
    service_name: tradsys
```

### Environment Variables

Environment variables can be used to override configuration values. The naming convention is:

```
TRADSYS_SECTION_KEY=value
```

For example:

```
TRADSYS_DATABASE_HOST=db.example.com
TRADSYS_DATABASE_PASSWORD=secret
TRADSYS_TRADING_ENGINE_MAX_ORDERS_PER_SECOND=2000
```

### Command-Line Flags

Command-line flags can be used to override configuration values. The naming convention is:

```
--section.key=value
```

For example:

```
--database.host=db.example.com
--database.password=secret
--trading.engine.max_orders_per_second=2000
```

## Deployment Options

TradSys can be deployed in various ways:

1. **Single Binary Deployment**: Deploy as a single binary on a server
2. **Containerized Deployment**: Deploy as a Docker container
3. **Orchestrated Deployment**: Deploy on Kubernetes or other orchestration platforms
4. **Microservices Deployment**: Deploy as multiple microservices

### Single Binary Deployment

#### Build the Binary

```bash
# Clone the repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Build the binary
go build -o tradsys cmd/tradsys/main.go
```

#### Run the Binary

```bash
# Run with a configuration file
./tradsys --config=config.yaml

# Run with environment variables
TRADSYS_DATABASE_PASSWORD=secret ./tradsys --config=config.yaml

# Run with command-line flags
./tradsys --config=config.yaml --database.password=secret
```

### Containerized Deployment

#### Build the Docker Image

Create a `Dockerfile`:

```dockerfile
# Build stage
FROM golang:1.18-alpine AS build

WORKDIR /app

# Copy go.mod and go.sum
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -o tradsys cmd/tradsys/main.go

# Final stage
FROM alpine:3.15

WORKDIR /app

# Copy the binary from the build stage
COPY --from=build /app/tradsys .

# Copy configuration
COPY config.yaml .

# Expose ports
EXPOSE 8080 8081 9090

# Set environment variables
ENV TRADSYS_GENERAL_ENVIRONMENT=production

# Run the binary
ENTRYPOINT ["./tradsys", "--config=config.yaml"]
```

Build the Docker image:

```bash
docker build -t tradsys:latest .
```

#### Run the Docker Container

```bash
docker run -d \
  --name tradsys \
  -p 8080:8080 \
  -p 8081:8081 \
  -p 9090:9090 \
  -e TRADSYS_DATABASE_PASSWORD=secret \
  tradsys:latest
```

### Orchestrated Deployment

#### Kubernetes Deployment

Create a `deployment.yaml` file:

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tradsys
  labels:
    app: tradsys
spec:
  replicas: 1
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
          name: http
        - containerPort: 8081
          name: websocket
        - containerPort: 9090
          name: metrics
        env:
        - name: TRADSYS_GENERAL_ENVIRONMENT
          value: production
        - name: TRADSYS_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tradsys-secrets
              key: db-password
        volumeMounts:
        - name: config
          mountPath: /app/config.yaml
          subPath: config.yaml
      volumes:
      - name: config
        configMap:
          name: tradsys-config
```

Create a `service.yaml` file:

```yaml
apiVersion: v1
kind: Service
metadata:
  name: tradsys
  labels:
    app: tradsys
spec:
  selector:
    app: tradsys
  ports:
  - port: 8080
    targetPort: 8080
    name: http
  - port: 8081
    targetPort: 8081
    name: websocket
  - port: 9090
    targetPort: 9090
    name: metrics
  type: ClusterIP
```

Create a `configmap.yaml` file:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tradsys-config
data:
  config.yaml: |
    # Configuration content here
```

Create a `secret.yaml` file:

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: tradsys-secrets
type: Opaque
data:
  db-password: c2VjcmV0  # base64 encoded "secret"
```

Apply the Kubernetes resources:

```bash
kubectl apply -f configmap.yaml
kubectl apply -f secret.yaml
kubectl apply -f deployment.yaml
kubectl apply -f service.yaml
```

### Microservices Deployment

TradSys can be deployed as multiple microservices:

1. **Market Data Service**: Handles market data ingestion and processing
2. **Order Matching Service**: Handles order matching and trade generation
3. **Strategy Service**: Handles strategy execution
4. **Risk Management Service**: Handles risk management
5. **API Service**: Handles API requests

Each service can be deployed separately using the methods described above.

## Monitoring and Logging

### Prometheus Metrics

TradSys exposes Prometheus metrics at the `/metrics` endpoint. You can configure Prometheus to scrape these metrics.

Example Prometheus configuration:

```yaml
scrape_configs:
  - job_name: 'tradsys'
    scrape_interval: 15s
    static_configs:
      - targets: ['tradsys:9090']
```

### Jaeger Tracing

TradSys supports distributed tracing with Jaeger. You can configure Jaeger to collect traces.

Example Jaeger configuration:

```yaml
JAEGER_AGENT_HOST: jaeger-agent
JAEGER_AGENT_PORT: 6831
JAEGER_SERVICE_NAME: tradsys
```

### Logging

TradSys uses structured logging with zap. Logs are output to stdout/stderr by default.

Example log output:

```json
{"level":"info","ts":"2023-06-01T12:34:56.789Z","caller":"tradsys/main.go:42","msg":"Starting TradSys","version":"1.0.0","environment":"production"}
```

## Scaling

### Horizontal Scaling

TradSys can be horizontally scaled by deploying multiple instances behind a load balancer. This works well for the API service and other stateless components.

### Vertical Scaling

TradSys can be vertically scaled by increasing the resources (CPU, memory) allocated to each instance. This works well for the order matching engine and other stateful components.

### Database Scaling

The database can be scaled using:

- Read replicas for read-heavy workloads
- Sharding for write-heavy workloads
- Connection pooling for better resource utilization

### Message Broker Scaling

The message broker can be scaled using:

- Clustering for high availability
- Partitioning for parallel processing
- Consumer groups for load balancing

## Security

### Network Security

- Use TLS for all external communications
- Use a firewall to restrict access to only necessary ports
- Use a VPN for administrative access

### Authentication and Authorization

- Use JWT for API authentication
- Use role-based access control (RBAC) for authorization
- Use API keys for service-to-service communication

### Secrets Management

- Use Kubernetes Secrets or a dedicated secrets management solution (e.g., HashiCorp Vault)
- Rotate secrets regularly
- Use environment variables for sensitive configuration

## Backup and Recovery

### Database Backup

- Set up regular database backups
- Test database restoration procedures
- Store backups in a secure, off-site location

### Configuration Backup

- Version control your configuration files
- Document configuration changes
- Use infrastructure as code (IaC) tools for configuration management

### Disaster Recovery

- Document disaster recovery procedures
- Test disaster recovery procedures regularly
- Have a backup deployment environment ready

## Troubleshooting

### Common Issues

#### Database Connection Issues

```
Error: failed to connect to database: dial tcp: lookup db.example.com: no such host
```

Solutions:
- Check database hostname and DNS resolution
- Check database credentials
- Check database firewall rules
- Check database availability

#### Message Broker Connection Issues

```
Error: failed to connect to message broker: dial tcp: lookup nats.example.com: no such host
```

Solutions:
- Check message broker hostname and DNS resolution
- Check message broker credentials
- Check message broker firewall rules
- Check message broker availability

#### Order Matching Engine Issues

```
Error: failed to match order: order book not found for symbol BTC-USD
```

Solutions:
- Check if the symbol is configured correctly
- Check if the market data service is running
- Check if the order matching engine is initialized

#### Strategy Execution Issues

```
Error: failed to execute strategy: strategy not found: mean_reversion_btc
```

Solutions:
- Check if the strategy is configured correctly
- Check if the strategy is enabled
- Check if the strategy is registered with the strategy factory

### Logs and Metrics

- Check logs for error messages
- Check metrics for anomalies
- Use distributed tracing to identify bottlenecks

### Support

If you encounter issues that you cannot resolve, contact support:

- Email: support@tradsys.example.com
- Slack: #tradsys-support
- GitHub: https://github.com/abdoElHodaky/tradSys/issues

## Conclusion

This guide has covered the deployment of TradSys in various environments. By following these guidelines, you can deploy TradSys in a way that meets your requirements for performance, scalability, and reliability.

