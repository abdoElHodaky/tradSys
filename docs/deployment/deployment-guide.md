# TradSys Deployment & Configuration Guide

## 🚀 Deployment Overview

TradSys supports multiple deployment strategies from development to enterprise production environments with high-availability and disaster recovery capabilities.

## 📋 Prerequisites

### System Requirements

#### Minimum Requirements (Development)
- **CPU**: 4 cores, 2.4GHz
- **RAM**: 8GB
- **Storage**: 50GB SSD
- **Network**: 100Mbps

#### Recommended Requirements (Production)
- **CPU**: 16+ cores, 3.0GHz+ (Intel Xeon or AMD EPYC)
- **RAM**: 64GB+ DDR4
- **Storage**: 500GB+ NVMe SSD (RAID 10)
- **Network**: 10Gbps+ with low latency

#### High-Frequency Trading Requirements
- **CPU**: 32+ cores, 3.5GHz+ with NUMA optimization
- **RAM**: 128GB+ DDR4-3200
- **Storage**: 1TB+ NVMe SSD with dedicated WAL storage
- **Network**: 25Gbps+ with kernel bypass (DPDK)
- **Latency**: Sub-100μs network latency to exchanges

### Software Dependencies

```bash
# Core Dependencies
Go 1.24+
PostgreSQL 13+ (or 15+ for better performance)
Redis 6+ (or 7+ for improved memory efficiency)
Docker 20.10+
Docker Compose 2.0+

# Optional (Production)
Kubernetes 1.24+
Prometheus 2.40+
Grafana 9.0+
Nginx 1.20+
```

## 🐳 Docker Deployment

### Quick Start with Docker Compose

```bash
# Clone repository
git clone https://github.com/abdoElHodaky/tradSys.git
cd tradSys

# Copy environment configuration
cp .env.example .env

# Edit configuration
nano .env

# Start all services
docker-compose up -d

# Check service status
docker-compose ps

# View logs
docker-compose logs -f tradsys
```

### Docker Compose Configuration

```yaml
# docker-compose.yml
version: '3.8'

services:
  tradsys:
    build: .
    ports:
      - "8080:8080"
      - "8081:8081"  # WebSocket
    environment:
      - DB_HOST=postgres
      - REDIS_HOST=redis
      - ENV=production
    depends_on:
      - postgres
      - redis
    volumes:
      - ./config:/app/config
      - ./logs:/app/logs
    restart: unless-stopped
    deploy:
      resources:
        limits:
          cpus: '4.0'
          memory: 8G
        reservations:
          cpus: '2.0'
          memory: 4G

  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: tradsys
      POSTGRES_USER: tradsys
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --appendonly yes --maxmemory 2gb --maxmemory-policy allkeys-lru
    volumes:
      - redis_data:/data
    ports:
      - "6379:6379"
    restart: unless-stopped

  prometheus:
    image: prom/prometheus:latest
    ports:
      - "9090:9090"
    volumes:
      - ./monitoring/prometheus.yml:/etc/prometheus/prometheus.yml
      - prometheus_data:/prometheus
    command:
      - '--config.file=/etc/prometheus/prometheus.yml'
      - '--storage.tsdb.path=/prometheus'
      - '--web.console.libraries=/etc/prometheus/console_libraries'
      - '--web.console.templates=/etc/prometheus/consoles'

  grafana:
    image: grafana/grafana:latest
    ports:
      - "3000:3000"
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=${GRAFANA_PASSWORD}
    volumes:
      - grafana_data:/var/lib/grafana
      - ./monitoring/grafana:/etc/grafana/provisioning

volumes:
  postgres_data:
  redis_data:
  prometheus_data:
  grafana_data:
```

### Production Docker Configuration

```dockerfile
# Dockerfile.production
FROM golang:1.24-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -ldflags '-extldflags "-static"' -o tradsys cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /root/

COPY --from=builder /app/tradsys .
COPY --from=builder /app/config ./config
COPY --from=builder /app/migrations ./migrations

EXPOSE 8080 8081
CMD ["./tradsys"]
```

## ☸️ Kubernetes Deployment

### Namespace and ConfigMap

```yaml
# k8s/namespace.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tradsys

---
# k8s/configmap.yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: tradsys-config
  namespace: tradsys
data:
  config.yaml: |
    server:
      port: 8080
      websocket_port: 8081
    database:
      host: postgres-service
      port: 5432
      name: tradsys
    redis:
      host: redis-service
      port: 6379
    matching_engine:
      latency_target_ns: 100000
      max_orders_per_second: 100000
    risk_management:
      enable_pre_trade_checks: true
      var_calculation_interval: 1s
```

### Deployment Configuration

```yaml
# k8s/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: tradsys
  namespace: tradsys
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
        env:
        - name: DB_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tradsys-secrets
              key: db-password
        - name: REDIS_PASSWORD
          valueFrom:
            secretKeyRef:
              name: tradsys-secrets
              key: redis-password
        volumeMounts:
        - name: config
          mountPath: /app/config
        resources:
          requests:
            memory: "4Gi"
            cpu: "2000m"
          limits:
            memory: "8Gi"
            cpu: "4000m"
        livenessProbe:
          httpGet:
            path: /health
            port: 8080
          initialDelaySeconds: 30
          periodSeconds: 10
        readinessProbe:
          httpGet:
            path: /ready
            port: 8080
          initialDelaySeconds: 5
          periodSeconds: 5
      volumes:
      - name: config
        configMap:
          name: tradsys-config

---
# k8s/service.yaml
apiVersion: v1
kind: Service
metadata:
  name: tradsys-service
  namespace: tradsys
spec:
  selector:
    app: tradsys
  ports:
  - name: http
    port: 80
    targetPort: 8080
  - name: websocket
    port: 8081
    targetPort: 8081
  type: LoadBalancer
```

### High Availability Setup

```yaml
# k8s/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: tradsys-hpa
  namespace: tradsys
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: tradsys
  minReplicas: 3
  maxReplicas: 20
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80

---
# k8s/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: tradsys-pdb
  namespace: tradsys
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: tradsys
```

## 🔧 Configuration Management

### Environment Variables

```bash
# .env
# Application
ENV=production
LOG_LEVEL=info
DEBUG=false

# Server
SERVER_PORT=8080
WEBSOCKET_PORT=8081
GRPC_PORT=9090

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=tradsys
DB_USER=tradsys
DB_PASSWORD=secure_password
DB_SSL_MODE=require
DB_MAX_CONNECTIONS=100
DB_MAX_IDLE_CONNECTIONS=10

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=redis_password
REDIS_DB=0
REDIS_MAX_CONNECTIONS=100

# Exchange APIs
EGX_API_URL=https://api.egx.com.eg
EGX_API_KEY=your_egx_api_key
EGX_API_SECRET=your_egx_secret

ADX_API_URL=https://api.adx.ae
ADX_API_KEY=your_adx_api_key
ADX_API_SECRET=your_adx_secret

# WebSocket
WS_MAX_CONNECTIONS=10000
WS_READ_BUFFER_SIZE=1024
WS_WRITE_BUFFER_SIZE=1024
WS_HEARTBEAT_INTERVAL=30s

# Risk Management
RISK_VAR_CONFIDENCE=0.95
RISK_MAX_POSITION_SIZE=1000000
RISK_CALCULATION_INTERVAL=1s

# Compliance
ENABLE_SHARIA_COMPLIANCE=true
ENABLE_SEC_COMPLIANCE=true
ENABLE_MIFID_COMPLIANCE=true
COMPLIANCE_AUDIT_ENABLED=true

# Monitoring
PROMETHEUS_ENABLED=true
PROMETHEUS_PORT=9090
METRICS_INTERVAL=10s

# Security
JWT_SECRET=your_jwt_secret_key
JWT_EXPIRY=24h
API_RATE_LIMIT=1000
CORS_ORIGINS=*

# Performance
GOMAXPROCS=0  # Use all available CPUs
GOGC=100      # Default garbage collection target
```

### Configuration Files

```yaml
# config/production.yaml
server:
  port: 8080
  websocket_port: 8081
  grpc_port: 9090
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 120s
  max_header_bytes: 1048576

database:
  host: ${DB_HOST}
  port: ${DB_PORT}
  name: ${DB_NAME}
  user: ${DB_USER}
  password: ${DB_PASSWORD}
  ssl_mode: require
  max_connections: 100
  max_idle_connections: 10
  connection_max_lifetime: 1h

redis:
  host: ${REDIS_HOST}
  port: ${REDIS_PORT}
  password: ${REDIS_PASSWORD}
  db: 0
  max_connections: 100
  dial_timeout: 5s
  read_timeout: 3s
  write_timeout: 3s

matching_engine:
  latency_target_ns: 100000  # 100 microseconds
  max_orders_per_second: 100000
  order_book_depth: 1000
  enable_hft_mode: true
  numa_optimization: true

risk_management:
  enable_pre_trade_checks: true
  enable_real_time_monitoring: true
  var_confidence: 0.95
  var_calculation_interval: 1s
  max_position_size: 1000000
  concentration_limit: 0.3

compliance:
  enabled_regulations: ["sec", "mifid", "sca", "sharia"]
  auto_reporting: true
  audit_trail: true
  sharia_screening: true

websocket:
  max_connections: 10000
  read_buffer_size: 1024
  write_buffer_size: 1024
  heartbeat_interval: 30s
  compression: true

monitoring:
  prometheus:
    enabled: true
    port: 9090
    path: /metrics
  logging:
    level: info
    format: json
    output: stdout
  tracing:
    enabled: true
    jaeger_endpoint: http://jaeger:14268/api/traces
```

## 🗄️ Database Setup

### PostgreSQL Configuration

```sql
-- Create database and user
CREATE DATABASE tradsys;
CREATE USER tradsys WITH ENCRYPTED PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE tradsys TO tradsys;

-- Connect to tradsys database
\c tradsys

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_stat_statements";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";

-- Performance tuning
ALTER SYSTEM SET shared_buffers = '256MB';
ALTER SYSTEM SET effective_cache_size = '1GB';
ALTER SYSTEM SET maintenance_work_mem = '64MB';
ALTER SYSTEM SET checkpoint_completion_target = 0.9;
ALTER SYSTEM SET wal_buffers = '16MB';
ALTER SYSTEM SET default_statistics_target = 100;
ALTER SYSTEM SET random_page_cost = 1.1;
ALTER SYSTEM SET effective_io_concurrency = 200;

-- Reload configuration
SELECT pg_reload_conf();
```

### Database Migrations

```bash
# Run migrations
go run cmd/migrate/main.go up

# Rollback migrations
go run cmd/migrate/main.go down

# Check migration status
go run cmd/migrate/main.go status
```

### Redis Configuration

```conf
# redis.conf
# Memory management
maxmemory 2gb
maxmemory-policy allkeys-lru

# Persistence
save 900 1
save 300 10
save 60 10000

# Network
tcp-keepalive 300
timeout 0

# Performance
tcp-backlog 511
databases 16
```

## 🔒 Security Configuration

### SSL/TLS Setup

```bash
# Generate SSL certificates
openssl req -x509 -newkey rsa:4096 -keyout key.pem -out cert.pem -days 365 -nodes

# Configure nginx for SSL termination
# /etc/nginx/sites-available/tradsys
server {
    listen 443 ssl http2;
    server_name tradsys.yourdomain.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers ECDHE-RSA-AES256-GCM-SHA512:DHE-RSA-AES256-GCM-SHA512;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /ws {
        proxy_pass http://localhost:8081;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection "upgrade";
        proxy_set_header Host $host;
    }
}
```

### Firewall Configuration

```bash
# UFW firewall rules
sudo ufw allow 22/tcp      # SSH
sudo ufw allow 80/tcp      # HTTP
sudo ufw allow 443/tcp     # HTTPS
sudo ufw allow 8080/tcp    # TradSys API
sudo ufw allow 8081/tcp    # WebSocket
sudo ufw deny 5432/tcp     # PostgreSQL (internal only)
sudo ufw deny 6379/tcp     # Redis (internal only)
sudo ufw enable
```

## 📊 Monitoring Setup

### Prometheus Configuration

```yaml
# monitoring/prometheus.yml
global:
  scrape_interval: 15s
  evaluation_interval: 15s

rule_files:
  - "rules/*.yml"

scrape_configs:
  - job_name: 'tradsys'
    static_configs:
      - targets: ['localhost:8080']
    metrics_path: /metrics
    scrape_interval: 5s

  - job_name: 'postgres'
    static_configs:
      - targets: ['localhost:9187']

  - job_name: 'redis'
    static_configs:
      - targets: ['localhost:9121']

alerting:
  alertmanagers:
    - static_configs:
        - targets:
          - alertmanager:9093
```

### Grafana Dashboards

```json
{
  "dashboard": {
    "title": "TradSys Trading Metrics",
    "panels": [
      {
        "title": "Orders Per Second",
        "type": "graph",
        "targets": [
          {
            "expr": "rate(tradsys_orders_total[1m])",
            "legendFormat": "Orders/sec"
          }
        ]
      },
      {
        "title": "Matching Engine Latency",
        "type": "graph",
        "targets": [
          {
            "expr": "histogram_quantile(0.95, tradsys_matching_latency_seconds_bucket)",
            "legendFormat": "95th percentile"
          }
        ]
      }
    ]
  }
}
```

## 🚀 Performance Optimization

### System Tuning

```bash
# /etc/sysctl.conf
# Network optimization
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728
net.core.netdev_max_backlog = 5000

# Memory optimization
vm.swappiness = 1
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5

# CPU optimization
kernel.sched_migration_cost_ns = 5000000
kernel.sched_autogroup_enabled = 0

# Apply changes
sudo sysctl -p
```

### Go Runtime Optimization

```bash
# Environment variables for Go runtime
export GOMAXPROCS=16        # Number of CPU cores
export GOGC=100            # GC target percentage
export GODEBUG=gctrace=1   # Enable GC tracing (development only)
```

## 🔄 Backup and Recovery

### Database Backup

```bash
#!/bin/bash
# backup.sh
DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR="/backups"
DB_NAME="tradsys"

# Create backup
pg_dump -h localhost -U tradsys -d $DB_NAME | gzip > $BACKUP_DIR/tradsys_$DATE.sql.gz

# Cleanup old backups (keep 30 days)
find $BACKUP_DIR -name "tradsys_*.sql.gz" -mtime +30 -delete

# Upload to S3 (optional)
aws s3 cp $BACKUP_DIR/tradsys_$DATE.sql.gz s3://your-backup-bucket/
```

### Disaster Recovery

```bash
#!/bin/bash
# restore.sh
BACKUP_FILE=$1

if [ -z "$BACKUP_FILE" ]; then
    echo "Usage: $0 <backup_file>"
    exit 1
fi

# Stop application
docker-compose stop tradsys

# Restore database
gunzip -c $BACKUP_FILE | psql -h localhost -U tradsys -d tradsys

# Start application
docker-compose start tradsys
```

## 📋 Deployment Checklist

### Pre-Deployment
- [ ] System requirements verified
- [ ] Dependencies installed
- [ ] Configuration files prepared
- [ ] SSL certificates configured
- [ ] Database migrations tested
- [ ] Backup procedures tested
- [ ] Monitoring configured
- [ ] Security hardening applied

### Deployment
- [ ] Application deployed
- [ ] Database migrations applied
- [ ] Services started and healthy
- [ ] Load balancer configured
- [ ] SSL/TLS working
- [ ] WebSocket connections working
- [ ] API endpoints responding
- [ ] Monitoring data flowing

### Post-Deployment
- [ ] Performance benchmarks run
- [ ] Integration tests passed
- [ ] Compliance checks verified
- [ ] Backup procedures verified
- [ ] Monitoring alerts configured
- [ ] Documentation updated
- [ ] Team notified

## 🆘 Troubleshooting

### Common Issues

#### Service Won't Start
```bash
# Check logs
docker-compose logs tradsys

# Check configuration
go run cmd/validate-config/main.go

# Test database connection
go run cmd/healthcheck/main.go
```

#### High Latency
```bash
# Check system resources
htop
iostat -x 1

# Check network latency
ping exchange-api.com
traceroute exchange-api.com

# Check application metrics
curl http://localhost:8080/metrics | grep latency
```

#### Memory Issues
```bash
# Check memory usage
free -h
docker stats

# Check Go heap
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

---

**Ready for production? Follow this guide to deploy TradSys with confidence!** 🚀
