# 🔄 TradSys Migration Plan - Detailed Implementation Guide

## Overview
This document provides a step-by-step migration plan for implementing the TradSys resimplification and unification. Each step includes specific commands, validation procedures, and rollback strategies.

## Pre-Migration Checklist

### ✅ Backup and Safety
- [ ] Create backup branch: `git checkout -b v2-backup`
- [ ] Verify all tests pass: `go test ./...`
- [ ] Document current performance baselines
- [ ] Ensure deployment scripts work
- [ ] Backup production configurations

### ✅ Environment Setup
- [ ] Go 1.21+ installed
- [ ] Docker and kubectl available
- [ ] All dependencies downloaded: `go mod download`
- [ ] Clean working directory: `git status`

## Phase 1: Naming Unification (COMPLETED ✅)

### Step 1.1: Binary and Command Naming ✅
```bash
# Rename main command directory
mv cmd/server cmd/tradsys

# Update Dockerfile
sed -i 's|./cmd/server|./cmd/tradsys|g' Dockerfile

# Update README build instructions
sed -i 's|./cmd/server|./cmd/tradsys|g' README.md
```

### Step 1.2: Configuration Consolidation ✅
```bash
# Consolidate config directories
mv configs/hft-config.yaml config/tradsys-config.yaml
rmdir configs

# Update configuration references
find . -name "*.go" -exec sed -i 's|configs/hft-config.yaml|config/tradsys-config.yaml|g' {} \;
find . -name "*.yaml" -exec sed -i 's|configs/hft-config.yaml|config/tradsys-config.yaml|g' {} \;
```

### Step 1.3: Deployment Manifest Updates ✅
```bash
# Update Kubernetes manifests
sed -i 's|hft-server|tradsys|g' deployments/kubernetes/*.yaml
sed -i 's|hft-trading-system|tradsys|g' deployments/kubernetes/*.yaml

# Update Docker Compose
sed -i 's|hft-server|tradsys|g' docker-compose.yml
```

## Phase 2: Structure Simplification (IN PROGRESS 🔄)

### Step 2.1: Command Consolidation
```bash
# Create unified command structure
mkdir -p cmd/tradsys-unified

# Merge functionality from multiple commands
cat > cmd/tradsys-unified/main.go << 'EOF'
package main

import (
    "flag"
    "fmt"
    "os"
)

func main() {
    if len(os.Args) < 2 {
        printUsage()
        os.Exit(1)
    }

    switch os.Args[1] {
    case "server":
        runServer()
    case "gateway":
        runGateway()
    case "orders":
        runOrderService()
    case "risk":
        runRiskService()
    case "marketdata":
        runMarketDataService()
    case "ws":
        runWebSocketService()
    default:
        printUsage()
        os.Exit(1)
    }
}

func printUsage() {
    fmt.Println("Usage: tradsys <command>")
    fmt.Println("Commands:")
    fmt.Println("  server     - Run unified trading server")
    fmt.Println("  gateway    - Run API gateway")
    fmt.Println("  orders     - Run order service")
    fmt.Println("  risk       - Run risk service")
    fmt.Println("  marketdata - Run market data service")
    fmt.Println("  ws         - Run WebSocket service")
}
EOF
```

### Step 2.2: Internal Directory Reorganization
```bash
# Create new simplified structure
mkdir -p internal/core/{matching,risk,settlement}
mkdir -p internal/{connectivity,compliance,strategies}
mkdir -p internal/{api,monitoring,config}

# Move existing components
mv internal/hft/matching/* internal/core/matching/ 2>/dev/null || true
mv internal/hft/risk/* internal/core/risk/ 2>/dev/null || true
mv internal/hft/settlement/* internal/core/settlement/ 2>/dev/null || true

# Move trading components
mv internal/trading/connectivity/* internal/connectivity/ 2>/dev/null || true
mv internal/trading/compliance/* internal/compliance/ 2>/dev/null || true
mv internal/trading/strategies/* internal/strategies/ 2>/dev/null || true
```

### Step 2.3: Update Import Statements
```bash
# Create script to update imports
cat > scripts/update-imports.sh << 'EOF'
#!/bin/bash

# Update all Go files to use new import paths
find . -name "*.go" -type f -exec sed -i \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/matching|github.com/abdoElHodaky/tradSys/internal/core/matching|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/risk|github.com/abdoElHodaky/tradSys/internal/core/risk|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/hft/settlement|github.com/abdoElHodaky/tradSys/internal/core/settlement|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/connectivity|github.com/abdoElHodaky/tradSys/internal/connectivity|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/compliance|github.com/abdoElHodaky/tradSys/internal/compliance|g' \
    -e 's|github.com/abdoElHodaky/tradSys/internal/trading/strategies|github.com/abdoElHodaky/tradSys/internal/strategies|g' \
    {} \;

echo "Import paths updated"
EOF

chmod +x scripts/update-imports.sh
./scripts/update-imports.sh
```

## Phase 3: Interface Standardization

### Step 3.1: Error Handling Standardization
```bash
# Create unified error types
cat > internal/common/errors.go << 'EOF'
package common

import "fmt"

// TradSysError represents a unified error type
type TradSysError struct {
    Code    string
    Message string
    Cause   error
}

func (e *TradSysError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("[%s] %s: %v", e.Code, e.Message, e.Cause)
    }
    return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Common error constructors
func NewValidationError(msg string, cause error) *TradSysError {
    return &TradSysError{Code: "VALIDATION_ERROR", Message: msg, Cause: cause}
}

func NewSystemError(msg string, cause error) *TradSysError {
    return &TradSysError{Code: "SYSTEM_ERROR", Message: msg, Cause: cause}
}

func NewBusinessError(msg string, cause error) *TradSysError {
    return &TradSysError{Code: "BUSINESS_ERROR", Message: msg, Cause: cause}
}
EOF
```

### Step 3.2: Configuration Management Unification
```bash
# Create unified configuration loader
cat > internal/config/loader.go << 'EOF'
package config

import (
    "fmt"
    "os"
    "path/filepath"
    
    "gopkg.in/yaml.v3"
)

// Config represents the unified configuration
type Config struct {
    Server       ServerConfig       `yaml:"server"`
    Core         CoreConfig         `yaml:"core"`
    Connectivity ConnectivityConfig `yaml:"connectivity"`
    Compliance   ComplianceConfig   `yaml:"compliance"`
    Strategies   StrategiesConfig   `yaml:"strategies"`
    Database     DatabaseConfig     `yaml:"database"`
    Redis        RedisConfig        `yaml:"redis"`
    Logging      LoggingConfig      `yaml:"logging"`
    Metrics      MetricsConfig      `yaml:"metrics"`
}

// Load loads configuration from file and environment
func Load() (*Config, error) {
    configPath := os.Getenv("TRADSYS_CONFIG_PATH")
    if configPath == "" {
        configPath = "config/tradsys-config.yaml"
    }

    data, err := os.ReadFile(configPath)
    if err != nil {
        return nil, fmt.Errorf("failed to read config file: %w", err)
    }

    var config Config
    if err := yaml.Unmarshal(data, &config); err != nil {
        return nil, fmt.Errorf("failed to parse config: %w", err)
    }

    // Override with environment variables
    overrideWithEnv(&config)

    return &config, nil
}

func overrideWithEnv(config *Config) {
    if port := os.Getenv("TRADSYS_PORT"); port != "" {
        // Parse and set port
    }
    if logLevel := os.Getenv("TRADSYS_LOG_LEVEL"); logLevel != "" {
        config.Logging.Level = logLevel
    }
    // Add more environment overrides as needed
}
EOF
```

### Step 3.3: Logging Interface Standardization
```bash
# Create unified logging interface
cat > internal/common/logger.go << 'EOF'
package common

import (
    "log"
    "os"
)

// Logger interface for unified logging
type Logger interface {
    Debug(msg string, fields ...interface{})
    Info(msg string, fields ...interface{})
    Warn(msg string, fields ...interface{})
    Error(msg string, fields ...interface{})
    Fatal(msg string, fields ...interface{})
}

// DefaultLogger implements Logger using standard log
type DefaultLogger struct {
    logger *log.Logger
}

func NewDefaultLogger() *DefaultLogger {
    return &DefaultLogger{
        logger: log.New(os.Stdout, "[TradSys] ", log.LstdFlags|log.Lshortfile),
    }
}

func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
    l.logger.Printf("[DEBUG] "+msg, fields...)
}

func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
    l.logger.Printf("[INFO] "+msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
    l.logger.Printf("[WARN] "+msg, fields...)
}

func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
    l.logger.Printf("[ERROR] "+msg, fields...)
}

func (l *DefaultLogger) Fatal(msg string, fields ...interface{}) {
    l.logger.Fatalf("[FATAL] "+msg, fields...)
}
EOF
```

## Phase 4: Documentation Alignment

### Step 4.1: Consolidate Architecture Documentation
```bash
# Merge architecture documents
cat ARCHITECTURE.md COMPONENT_ANALYSIS.md > docs/UNIFIED_ARCHITECTURE.md

# Update references
sed -i 's|ARCHITECTURE.md|docs/UNIFIED_ARCHITECTURE.md|g' README.md
```

### Step 4.2: Update API Documentation
```bash
# Generate API documentation from code
go install github.com/swaggo/swag/cmd/swag@latest
swag init -g cmd/tradsys/main.go -o docs/api/

# Update README with API docs link
echo "- 📖 API Documentation: [docs/api/](docs/api/)" >> README.md
```

## Validation and Testing

### Performance Validation
```bash
# Run performance benchmarks
go test -bench=. ./internal/core/matching/
go test -bench=. ./internal/core/risk/
go test -bench=. ./internal/core/settlement/

# Validate latency targets
./scripts/benchmark.sh
```

### Functional Testing
```bash
# Run all unit tests
go test ./...

# Run integration tests
go test -tags=integration ./tests/integration/

# Run end-to-end tests
./scripts/e2e-test.sh
```

### Deployment Validation
```bash
# Build and test Docker image
docker build -t tradsys:test .
docker run --rm tradsys:test --version

# Test Kubernetes deployment
kubectl apply -f deployments/kubernetes/ --dry-run=client
```

## Rollback Procedures

### Quick Rollback
```bash
# Rollback to backup branch
git checkout v2-backup
git checkout -b v2-rollback
git push origin v2-rollback
```

### Selective Rollback
```bash
# Rollback specific files
git checkout v2-backup -- cmd/
git checkout v2-backup -- internal/
git commit -m "Rollback to stable state"
```

## Success Criteria Checklist

### Functional Requirements
- [ ] All unit tests pass: `go test ./...`
- [ ] Integration tests pass: `go test -tags=integration ./tests/integration/`
- [ ] Performance benchmarks meet targets
- [ ] Docker build succeeds: `docker build -t tradsys .`
- [ ] Kubernetes deployment validates: `kubectl apply --dry-run=client`

### Quality Improvements
- [ ] Consistent naming throughout codebase
- [ ] Simplified directory structure
- [ ] Unified configuration management
- [ ] Consolidated documentation
- [ ] Standardized error handling
- [ ] Unified logging interface

### Performance Targets
- [ ] Order processing latency < 100μs
- [ ] Risk check latency < 10μs
- [ ] Settlement processing < 1ms
- [ ] Memory usage optimized
- [ ] CPU utilization efficient

## Timeline and Milestones

### Week 1: Phase 1 Complete ✅
- [x] Naming unification
- [x] Configuration consolidation
- [x] Documentation updates

### Week 2: Phase 2 (Current)
- [ ] Command consolidation
- [ ] Directory reorganization
- [ ] Import path updates

### Week 3: Phase 3
- [ ] Error handling standardization
- [ ] Configuration management unification
- [ ] Logging interface standardization

### Week 4: Phase 4
- [ ] Documentation consolidation
- [ ] Final testing and validation
- [ ] Production deployment preparation

## Conclusion

This migration plan provides a systematic approach to transforming TradSys into a simplified, unified, and maintainable system while preserving all functionality and performance characteristics. Each phase builds upon the previous one, ensuring a smooth transition with minimal risk.
