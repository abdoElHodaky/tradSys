---
name: di-analysis
description: Dependency Injection patterns in TradSys v3
metadata:
  type: reference
---

# Dependency Injection Analysis for TradSys v3

## Overview
The codebase has two distinct DI patterns:

### 1. Internal Packages (fx-based DI)

**Pattern:** Use Uber's fx framework for dependency injection

**Example:**
```go
// Package internal/orders/module.go
var OrderManagementModule = fx.Options(
    fx.Provide(NewService),
)

// NewFxService injects dependencies
func NewFxService(
    lifecycle fx.Lifecycle,
    logger *zap.Logger,
    orderEngine *order_matching.Engine,
) *Service {
    service := NewService(orderEngine, logger)
    // Add lifecycle hooks
    return service
}
```

**Files using fx:**
- `internal/gateway/module.go`
- `internal/risk/module.go`
- `internal/orders/module.go`
- `internal/marketdata/module.go`
- `internal/db/module.go`
- `internal/db/repositories/module.go`
- `pkg/matching/module.go`

**Key Features:**
- `fx.Provide()` registers constructors
- `fx.Lifecycle` for startup/shutdown hooks
- Dependencies automatically resolved by type

### 2. Services Packages (Manual DI)

**Pattern:** Direct struct initialization, no container

**Example:**
```go
// services/exchanges/adx_service.go
func NewADXService() *ADXService {
    return &ADXService{
        exchangeID:         "ADX",
        islamicCompliance:  NewIslamicCompliance(),
        uaeCompliance:      NewUAECompliance(),
        connector:          NewADXConnector(),
        marketData:         NewADXMarketData(),
        // ... all dependencies created inline
    }
}
```

**Files using manual DI:**
- All files under `services/` directory
- Config passed via params structs (HandlerParams, ServiceParams, etc.)

## Large Files Analysis

| File | Lines | Package | Issues |
|------|-------|---------|--------|
| `services/exchanges/adx_service.go` | 979 | exchanges | Many type+func bundles |
| `services/assets/unified_asset_system.go` | 914 | assets | Type aliases from licensing |
| `services/optimization/performance_optimizer.go` | 711 | optimization | Many optimizer types |
| `services/websocket/websocket_gateway.go` | 707 | websocket | Has separate components file |
| `services/assets/handler_registry.go` | 680 | assets | Handler types bundled |

## Recommendations

### For Services DI Migration:

1. **Create fx modules for each service package**
   ```go
   // services/exchanges/module.go
   var ADXModule = fx.Options(
       fx.Provide(NewADXService),
   )
   ```

2. **Use constructor injection pattern**
   ```go
   func NewADXService(
       logger *zap.Logger,
       config *Config,
   ) *ADXService {
       return &ADXService{...}
   }
   ```

3. **Standardize parameters struct**
   ```go
   type ADXParams struct {
       Logger  *zap.Logger
       Config  *Config
       // dependencies as fields
   }
   ```

### For File Organization:

1. **handler_registry.go** - Types could be in handler_types.go
   - HandlerRegistry, AssetHandler
   - Settlement, TradingHours
   - RiskParameters, FeeStructure, FeeCalculation
   - BaseAssetHandler, StockHandler, SukukHandler

2. **adx_service.go** - Consider adx_types.go for:
   - ComplianceLevel, ComplianceRule, ComplianceSeverity
   - IslamicCompliance, ZakatCalculator
   - UAECompliance, SCACompliance
   - SukukService, SukukType, SukukRiskEngine
   - IslamicFundService, IslamicNAVCalculator
   - Data structures: SukukData, IslamicFundData, etc.

3. **performance_optimizer.go** - Consider optimization_types.go for:
   - CacheOptimizer, DatabaseOptimizer, NetworkOptimizer
   - RegionalOptimizer, AutoScaler
   - Metrics types: CacheMetrics, DatabaseMetrics, etc.

## Why This Matters

**Current Issues:**
- Large files are hard to maintain
- No DI container for services (manual wiring)
- Types duplicated or defined but not reusable

**Benefits of Changes:**
- Better testability (dependency injection)
- Improved code organization
- Reusable types across services
- Consistent patterns between internal and services packages