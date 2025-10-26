# TradSys Troubleshooting Guide

## 🚨 Quick Diagnostics

### System Health Check
```bash
# Check all services status
curl http://localhost:8080/health

# Check individual components
curl http://localhost:8080/health/database
curl http://localhost:8080/health/redis
curl http://localhost:8080/health/matching-engine
curl http://localhost:8080/health/websocket

# Check system metrics
curl http://localhost:8080/metrics
```

### Log Analysis
```bash
# View application logs
docker-compose logs -f tradsys

# View specific service logs
docker-compose logs -f postgres
docker-compose logs -f redis

# Search for errors
docker-compose logs tradsys | grep -i error
docker-compose logs tradsys | grep -i panic
```

## 🔧 Common Issues & Solutions

### 1. Application Won't Start

#### Symptoms
- Service fails to start
- Exit code 1 or 2
- Connection refused errors

#### Diagnosis
```bash
# Check configuration
go run cmd/validate-config/main.go

# Test database connection
go run cmd/healthcheck/main.go

# Check port availability
netstat -tulpn | grep :8080
lsof -i :8080

# Check environment variables
env | grep -E "(DB_|REDIS_|API_)"
```

#### Solutions

**Database Connection Issues:**
```bash
# Check PostgreSQL status
docker-compose ps postgres
docker-compose logs postgres

# Test connection manually
psql -h localhost -U tradsys -d tradsys -c "SELECT 1;"

# Reset database
docker-compose down postgres
docker volume rm tradsys_postgres_data
docker-compose up -d postgres
```

**Redis Connection Issues:**
```bash
# Check Redis status
docker-compose ps redis
docker-compose logs redis

# Test connection
redis-cli -h localhost -p 6379 ping

# Clear Redis data
redis-cli -h localhost -p 6379 flushall
```

**Port Conflicts:**
```bash
# Find process using port
sudo lsof -i :8080
sudo kill -9 <PID>

# Change port in configuration
export SERVER_PORT=8081
```

### 2. High Latency Issues

#### Symptoms
- Slow API responses (>100ms)
- WebSocket message delays
- Order matching delays

#### Diagnosis
```bash
# Check system resources
htop
iostat -x 1
vmstat 1

# Check network latency
ping -c 10 localhost
traceroute exchange-api.com

# Check application metrics
curl http://localhost:8080/metrics | grep -E "(latency|duration)"

# Profile application
curl http://localhost:8080/debug/pprof/profile > cpu.prof
go tool pprof cpu.prof
```

#### Solutions

**CPU Bottlenecks:**
```bash
# Increase CPU allocation
docker-compose up --scale tradsys=3

# Optimize Go runtime
export GOMAXPROCS=16
export GOGC=50  # More aggressive GC
```

**Memory Issues:**
```bash
# Check memory usage
free -h
docker stats

# Increase memory limits
# In docker-compose.yml:
# mem_limit: 16g
# memswap_limit: 16g
```

**Database Performance:**
```sql
-- Check slow queries
SELECT query, mean_time, calls 
FROM pg_stat_statements 
ORDER BY mean_time DESC 
LIMIT 10;

-- Add indexes
CREATE INDEX CONCURRENTLY idx_orders_user_id_created_at 
ON orders(user_id, created_at);

-- Update statistics
ANALYZE;
```

**Network Optimization:**
```bash
# Tune network parameters
echo 'net.core.rmem_max = 134217728' >> /etc/sysctl.conf
echo 'net.core.wmem_max = 134217728' >> /etc/sysctl.conf
sysctl -p
```

### 3. WebSocket Connection Issues

#### Symptoms
- WebSocket connections dropping
- Message delivery failures
- Connection timeouts

#### Diagnosis
```bash
# Check WebSocket status
curl http://localhost:8081/ws/stats

# Test WebSocket connection
wscat -c ws://localhost:8081/ws

# Check connection limits
ulimit -n
cat /proc/sys/fs/file-max
```

#### Solutions

**Connection Limits:**
```bash
# Increase file descriptor limits
echo '* soft nofile 65536' >> /etc/security/limits.conf
echo '* hard nofile 65536' >> /etc/security/limits.conf

# For systemd services
echo 'LimitNOFILE=65536' >> /etc/systemd/system/tradsys.service
systemctl daemon-reload
```

**WebSocket Configuration:**
```yaml
# config/production.yaml
websocket:
  max_connections: 10000
  read_buffer_size: 4096
  write_buffer_size: 4096
  heartbeat_interval: 30s
  compression: true
  enable_ping_pong: true
```

**Load Balancer Configuration:**
```nginx
# nginx.conf
upstream websocket {
    ip_hash;  # Sticky sessions
    server 127.0.0.1:8081;
    server 127.0.0.1:8082;
}

location /ws {
    proxy_pass http://websocket;
    proxy_http_version 1.1;
    proxy_set_header Upgrade $http_upgrade;
    proxy_set_header Connection "upgrade";
    proxy_read_timeout 86400;
}
```

### 4. Database Performance Issues

#### Symptoms
- Slow query responses
- High database CPU usage
- Connection pool exhaustion

#### Diagnosis
```sql
-- Check active connections
SELECT count(*) FROM pg_stat_activity;

-- Check slow queries
SELECT query, mean_time, calls, total_time
FROM pg_stat_statements 
WHERE mean_time > 100  -- queries > 100ms
ORDER BY mean_time DESC;

-- Check locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Check table sizes
SELECT schemaname, tablename, 
       pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size
FROM pg_tables 
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;
```

#### Solutions

**Connection Pool Tuning:**
```yaml
# config/production.yaml
database:
  max_connections: 100
  max_idle_connections: 25
  connection_max_lifetime: 1h
  connection_max_idle_time: 10m
```

**Query Optimization:**
```sql
-- Add missing indexes
CREATE INDEX CONCURRENTLY idx_orders_symbol_created_at 
ON orders(symbol, created_at) 
WHERE status IN ('pending', 'partially_filled');

-- Partition large tables
CREATE TABLE orders_2024 PARTITION OF orders 
FOR VALUES FROM ('2024-01-01') TO ('2025-01-01');

-- Update table statistics
ANALYZE orders;
ANALYZE trades;
```

**PostgreSQL Tuning:**
```conf
# postgresql.conf
shared_buffers = 256MB
effective_cache_size = 1GB
maintenance_work_mem = 64MB
checkpoint_completion_target = 0.9
wal_buffers = 16MB
default_statistics_target = 100
random_page_cost = 1.1
effective_io_concurrency = 200
max_worker_processes = 8
max_parallel_workers_per_gather = 4
```

### 5. Memory Leaks & High Memory Usage

#### Symptoms
- Continuously increasing memory usage
- Out of memory errors
- Garbage collection pauses

#### Diagnosis
```bash
# Monitor memory usage
watch -n 1 'free -h && docker stats --no-stream'

# Go memory profiling
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Check for memory leaks
curl http://localhost:8080/debug/pprof/allocs > allocs.prof
go tool pprof allocs.prof

# GC statistics
curl http://localhost:8080/debug/vars | jq '.memstats'
```

#### Solutions

**Go Runtime Tuning:**
```bash
# Adjust garbage collection
export GOGC=50        # More aggressive GC
export GOMEMLIMIT=8GB # Memory limit

# Debug GC
export GODEBUG=gctrace=1
```

**Code Optimization:**
```go
// Use object pools for frequent allocations
var orderPool = sync.Pool{
    New: func() interface{} {
        return &Order{}
    },
}

// Reuse slices
orders := orders[:0]  // Reset slice but keep capacity

// Close resources properly
defer func() {
    if conn != nil {
        conn.Close()
    }
}()
```

**Container Limits:**
```yaml
# docker-compose.yml
services:
  tradsys:
    deploy:
      resources:
        limits:
          memory: 8G
        reservations:
          memory: 4G
```

### 6. Exchange Integration Issues

#### Symptoms
- Failed API calls to exchanges
- Authentication errors
- Rate limiting errors

#### Diagnosis
```bash
# Check exchange connectivity
curl -I https://api.egx.com.eg/health
curl -I https://api.adx.ae/health

# Test API credentials
curl -H "Authorization: Bearer $EGX_API_KEY" \
     https://api.egx.com.eg/v1/symbols

# Check rate limits
curl -v http://localhost:8080/api/v1/exchanges/EGX/rate-limits
```

#### Solutions

**API Credential Issues:**
```bash
# Verify credentials
echo $EGX_API_KEY | base64 -d
echo $ADX_API_SECRET | wc -c

# Rotate credentials
export EGX_API_KEY="new-api-key"
docker-compose restart tradsys
```

**Rate Limiting:**
```go
// Implement exponential backoff
func retryWithBackoff(fn func() error, maxRetries int) error {
    for i := 0; i < maxRetries; i++ {
        if err := fn(); err == nil {
            return nil
        }
        time.Sleep(time.Duration(1<<i) * time.Second)
    }
    return errors.New("max retries exceeded")
}
```

**Network Issues:**
```bash
# Check DNS resolution
nslookup api.egx.com.eg
nslookup api.adx.ae

# Test connectivity
telnet api.egx.com.eg 443
openssl s_client -connect api.egx.com.eg:443
```

### 7. Compliance Validation Failures

#### Symptoms
- Orders rejected by compliance engine
- Sharia compliance failures
- Regulatory violation alerts

#### Diagnosis
```bash
# Check compliance status
curl http://localhost:8080/api/v1/compliance/status

# View recent violations
curl http://localhost:8080/api/v1/compliance/violations?limit=10

# Test specific compliance rules
curl -X POST http://localhost:8080/api/v1/compliance/validate \
     -H "Content-Type: application/json" \
     -d '{"type": "order", "symbol": "AAPL", "user_id": "user123"}'
```

#### Solutions

**Rule Configuration:**
```yaml
# config/compliance.yaml
compliance:
  enabled_regulations: ["sec", "mifid", "sca", "sharia"]
  rules:
    position_limit:
      enabled: true
      max_position_size: 1000000
    concentration_risk:
      enabled: true
      max_concentration: 0.3
    sharia_screening:
      enabled: true
      prohibited_sectors: ["alcohol", "gambling", "tobacco"]
```

**Sharia Compliance:**
```bash
# Update Sharia screening database
curl -X POST http://localhost:8080/api/v1/compliance/sharia/update-database

# Check symbol compliance
curl http://localhost:8080/api/v1/compliance/sharia/check/AAPL
```

### 8. Monitoring & Alerting Issues

#### Symptoms
- Missing metrics in Prometheus
- Grafana dashboards not loading
- No alerts being triggered

#### Diagnosis
```bash
# Check Prometheus targets
curl http://localhost:9090/api/v1/targets

# Test metrics endpoint
curl http://localhost:8080/metrics

# Check Grafana connectivity
curl http://localhost:3000/api/health

# Verify alert rules
curl http://localhost:9090/api/v1/rules
```

#### Solutions

**Prometheus Configuration:**
```yaml
# prometheus.yml
scrape_configs:
  - job_name: 'tradsys'
    static_configs:
      - targets: ['tradsys:8080']
    scrape_interval: 5s
    metrics_path: /metrics
```

**Missing Metrics:**
```go
// Ensure metrics are registered
func init() {
    prometheus.MustRegister(ordersTotal)
    prometheus.MustRegister(matchingLatency)
    prometheus.MustRegister(errorRate)
}
```

**Alert Rules:**
```yaml
# alerts.yml
groups:
  - name: tradsys
    rules:
      - alert: HighLatency
        expr: histogram_quantile(0.95, tradsys_matching_latency_seconds_bucket) > 0.001
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "High matching engine latency"
```

## 🔍 Advanced Debugging

### Performance Profiling

```bash
# CPU profiling
curl http://localhost:8080/debug/pprof/profile?seconds=30 > cpu.prof
go tool pprof cpu.prof

# Memory profiling
curl http://localhost:8080/debug/pprof/heap > heap.prof
go tool pprof heap.prof

# Goroutine analysis
curl http://localhost:8080/debug/pprof/goroutine > goroutine.prof
go tool pprof goroutine.prof

# Mutex contention
curl http://localhost:8080/debug/pprof/mutex > mutex.prof
go tool pprof mutex.prof
```

### Database Debugging

```sql
-- Enable query logging
ALTER SYSTEM SET log_statement = 'all';
ALTER SYSTEM SET log_min_duration_statement = 100;
SELECT pg_reload_conf();

-- Check query plans
EXPLAIN (ANALYZE, BUFFERS) 
SELECT * FROM orders WHERE user_id = 'user123' AND status = 'pending';

-- Monitor real-time activity
SELECT pid, now() - pg_stat_activity.query_start AS duration, query 
FROM pg_stat_activity 
WHERE (now() - pg_stat_activity.query_start) > interval '5 minutes';
```

### Network Debugging

```bash
# Capture network traffic
sudo tcpdump -i any -w tradsys.pcap port 8080

# Analyze with Wireshark
wireshark tradsys.pcap

# Check connection states
ss -tuln | grep :8080
netstat -an | grep :8080

# Monitor bandwidth usage
iftop -i eth0
```

## 📞 Getting Help

### Log Collection Script

```bash
#!/bin/bash
# collect-logs.sh
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LOG_DIR="tradsys_logs_$TIMESTAMP"

mkdir -p $LOG_DIR

# Application logs
docker-compose logs --no-color tradsys > $LOG_DIR/tradsys.log
docker-compose logs --no-color postgres > $LOG_DIR/postgres.log
docker-compose logs --no-color redis > $LOG_DIR/redis.log

# System information
uname -a > $LOG_DIR/system_info.txt
free -h > $LOG_DIR/memory.txt
df -h > $LOG_DIR/disk.txt
docker stats --no-stream > $LOG_DIR/docker_stats.txt

# Configuration
cp .env $LOG_DIR/env.txt
cp docker-compose.yml $LOG_DIR/

# Metrics snapshot
curl -s http://localhost:8080/metrics > $LOG_DIR/metrics.txt
curl -s http://localhost:8080/health > $LOG_DIR/health.txt

# Create archive
tar -czf $LOG_DIR.tar.gz $LOG_DIR
echo "Logs collected in $LOG_DIR.tar.gz"
```

### Support Channels

1. **GitHub Issues**: Report bugs and feature requests
2. **Documentation**: Check the user guide and API reference
3. **Community Forum**: Ask questions and share solutions
4. **Professional Support**: Enterprise support available

### Before Contacting Support

- [ ] Check this troubleshooting guide
- [ ] Review application logs
- [ ] Verify configuration
- [ ] Test with minimal setup
- [ ] Collect diagnostic information
- [ ] Document steps to reproduce

---

**Still having issues? Use the log collection script above and reach out for help!** 🆘
