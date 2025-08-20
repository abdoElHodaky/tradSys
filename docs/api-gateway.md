# API Gateway

The API Gateway serves as the entry point for all client requests to the high-frequency trading platform. It provides a unified interface for clients to interact with the various microservices that make up the platform.

## Architecture

The API Gateway is implemented using the following components:

1. **Gin Web Framework**: Provides the HTTP server and routing capabilities
2. **Fx Dependency Injection**: Manages dependencies and lifecycle of components
3. **Service Proxy**: Forwards requests to the appropriate microservices
4. **Circuit Breaker**: Prevents cascading failures by stopping requests to failing services
5. **Rate Limiting**: Protects services from excessive requests
6. **Authentication**: Verifies user identity and permissions
7. **Monitoring**: Collects metrics and traces for observability

## Features

### 1. Request Routing

The API Gateway routes requests to the appropriate microservices based on the request path:

- `/api/market-data/*` → Market Data Service
- `/api/orders/*` → Order Service
- `/api/risk/*` → Risk Service
- `/api/pairs/*` → Market Data Service (Pairs API)
- `/api/strategies/*` → Order Service (Strategies API)
- `/api/users/*` → User Service

### 2. Authentication and Authorization

The API Gateway handles authentication and authorization for all requests:

- JWT-based authentication
- Role-based access control
- Token validation and refresh
- Secure cookie management

### 3. Rate Limiting

The API Gateway implements rate limiting to protect services from excessive requests:

- IP-based rate limiting
- Path-based rate limiting
- User-based rate limiting
- Configurable limits and burst allowances

### 4. Circuit Breaking

The API Gateway implements circuit breaking to prevent cascading failures:

- Service-level circuit breakers
- Configurable failure thresholds
- Automatic recovery with half-open state
- Fallback responses for open circuits

### 5. Request/Response Transformation

The API Gateway can transform requests and responses:

- Header manipulation
- Response aggregation
- Protocol translation (e.g., HTTP to gRPC)
- Content type conversion

### 6. Monitoring and Observability

The API Gateway provides monitoring and observability features:

- Request logging
- Prometheus metrics
- Distributed tracing with Jaeger
- Health checks

### 7. Security

The API Gateway implements security features:

- TLS termination
- CORS configuration
- Security headers
- Input validation
- Content security policy

## Configuration

The API Gateway is configured using environment variables or a configuration file:

```yaml
service:
  name: api-gateway
  version: 1.0.0
  address: :8000
  environment: development

gateway:
  readTimeout: 5000
  writeTimeout: 10000
  maxHeaderBytes: 1048576
  rateLimitRequests: 100
  rateLimitBurst: 200
  circuitBreakerThreshold: 5
  circuitBreakerTimeout: 30

registry:
  type: mdns
  addresses: []

broker:
  type: nats
  addresses: [nats:4222]

tracing:
  enabled: true
  type: jaeger
  address: jaeger:6831

metrics:
  enabled: true
  address: :8000

resilience:
  circuitBreakerEnabled: true
  rateLimitingEnabled: true

auth:
  jwtSecret: your-secret-key
  tokenExpiry: 3600
  refreshExpiry: 86400
```

## Deployment

The API Gateway can be deployed using Docker or Kubernetes:

### Docker

```bash
docker-compose up -d api-gateway
```

### Kubernetes

```bash
kubectl apply -f deployments/kubernetes/gateway.yaml
```

## Development

To run the API Gateway locally:

```bash
go run cmd/gateway/main.go
```

## API Documentation

The API Gateway provides a Swagger UI for API documentation at `/swagger/index.html`.

## Metrics

The API Gateway exposes Prometheus metrics at `/metrics`.

## Health Check

The API Gateway provides a health check endpoint at `/health`.

## Logging

The API Gateway logs requests and errors using structured logging with Zap.

## Dependencies

- Gin: HTTP server and routing
- Fx: Dependency injection
- Zap: Structured logging
- Prometheus: Metrics collection
- Jaeger: Distributed tracing
- NATS: Event broker
- JWT: Authentication

