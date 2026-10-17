---
name: architecture-analysis
description: Comprehensive fx module architecture analysis and recommendations
metadata:
  type: reference
---

# Fx Module Architecture Analysis for TradSys v3

## Current State

### Architectural Pattern Mismatch

The codebase exhibits two distinct dependency injection patterns:

| Pattern | Location | Implementation |
|---------|----------|----------------|
| **fx-based DI** | `internal/` packages | Uber fx framework with `fx.Provide`, `fx.Lifecycle` |
| **Manual DI** | `services/` packages | Direct struct initialization |

### Code Examples

**fx-based (internal/orders/module.go):**
```go
var OrderManagementModule = fx.Options(
    fx.Provide(NewService),
)

func NewFxService(
    lifecycle fx.Lifecycle,
    logger *zap.Logger,
    orderEngine *order_matching.Engine,
) *Service {
    service := NewService(orderEngine, logger)
    lifecycle.Append(fx.Hook{...})
    return service
}
```

**Manual (services/exchanges/adx_service.go):**
```go
func NewADXService() *ADXService {
    return &ADXService{
        exchangeID: "ADX",
        islamicCompliance: NewIslamicCompliance(),
        uaeCompliance: NewUAECompliance(),
        connector: NewADXConnector(),
        // All dependencies created inline
    }
}
```

## Fx Module Design Patterns

### Recommended Module Structure

```
services/
├── exchanges/
│   ├── module.go           # fx module for exchange services
│   ├── adx_service.go      # ADX-specific implementation
│   ├── egx_service.go      # EGX-specific implementation
│   ├── exchange_types.go   # Shared exchange types
│   └── islamic_types.go    # Islamic finance types
├── licensing/
│   └── module.go
├── islamic/
│   └── module.go
└── ... (other services)
```

### Module Template

```go
// services/exchanges/module.go
package exchanges

import (
    "context"
    "go.uber.org/fx"
    "go.uber.org/zap"
)

// ExchangeModule provides exchange service management
var ExchangeModule = fx.Options(
    fx.Provide(
        NewExchangeService,
        NewADXService,
        NewEGXService,
        NewIslamicCompliance,
        NewUAECompliance,
    ),
    fx.Invoke(RegisterExchangeHandlers),
)

// ExchangeService orchestrates all exchange services
type ExchangeService struct {
    fx.Shutdowner
    ADX  *ADXService
    EGX  *EGXService
    mu   sync.RWMutex
}

// NewExchangeService creates the orchestrator
func NewExchangeService(
    logger *zap.Logger,
    adx *ADXService,
    egx *EGXService,
) *ExchangeService {
    return &ExchangeService{
        ADX: adx,
        EGX: egx,
    }
}

// RegisterExchangeHandlers registers exchange handlers
func RegisterExchangeHandlers(lc fx.Lifecycle) {
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            // Initialize exchanges
            return nil
        },
        OnStop: func(ctx context.Context) error {
            // Cleanup
            return nil
        },
    })
}
```

## Stable Architecture Recommendations

### 1. Service Layer Modules

Create fx modules for each service boundary:

```go
// services/licensing/module.go
var LicensingModule = fx.Options(
    fx.Provide(
        NewLicenseValidator,
        NewQuotaManager,
        NewBillingEngine,
        fx.Annotate(NewService, fx.ResultFields),
    ),
)

// services/exchanges/module.go  
var ExchangesModule = fx.Options(
    fx.Provide(NewExchangeService),
    fx.Provide(
        fx.Annotated{Name: "adx", Target: NewADXService},
        fx.Annotated{Name: "egx", Target: NewEGXService},
    ),
)
```

### 2. Interface-Based Dependencies

Pass interfaces instead of concrete types:

```go
// Before (tightly coupled)
orderService orders.OrderService

// After (loosely coupled)
orderService OrderServiceInterface

type OrderServiceInterface interface {
    SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error)
    ValidateOrder(order *Order) error
}
```

### 3. Configuration Injection

Pass configuration through fx:

```go
type ExchangeConfig struct {
    APIEndpoint    string
    APIKey         string
    EnableIslamic  bool
    RateLimitRPS   int
}

func NewADXService(
    logger *zap.Logger,
    config ExchangeConfig,
) *ADXService {
    // Use config values
}
```

### 4. Lifecycle Management

Use fx.Lifecycle for proper startup/shutdown:

```go
func NewADXService(
    lc fx.Lifecycle,
    logger *zap.Logger,
    config ExchangeConfig,
) *ADXService {
    service := &ADXService{...}
    
    lc.Append(fx.Hook{
        OnStart: func(ctx context.Context) error {
            return service.Connect()
        },
        OnStop: func(ctx context.Context) error {
            return service.Shutdown(ctx)
        },
    })
    
    return service
}
```

## Migration Path

### Phase 1: Create Module Files (Low Risk)
- Create `module.go` files for each service package
- Define fx.Provide constructors without changing existing code
- Test that modules can be instantiated

### Phase 2: Refactor Constructors
- Update constructors to accept dependencies as parameters
- Replace inline initialization with injected dependencies

### Phase 3: Integrate with Main App
- Replace manual service assembly with fx modules
- Verify all services start/stop correctly

## Benefits of Fx-Based DI

1. **Testability**: Easy to mock dependencies
2. **Startup/Shutdown**: Automatic lifecycle management
3. **Error Handling**: Fail-fast on missing dependencies
4. **Documentation**: Built-in dependency graph
5. **Observability**: Integration with fx-cobolics for metrics

## Current File Analysis

Based on the analysis, here are the next refactoring targets:

| File | Lines | Priority | Action |
|------|-------|----------|--------|
| `services/exchanges/adx_service.go` | 887 | High | Already cleaned, add fx module |
| `services/assets/unified_asset_system.go` | 914 | High | Extract types, add fx module |
| `services/optimization/performance_optimizer.go` | 711 | Medium | Extract types to optimization_types.go |
| `services/websocket/websocket_gateway.go` | 707 | Medium | Already has components file |
| `services/assets/handler_registry.go` | 680 | Medium | Extract handler types |

## Conclusion

The codebase should migrate all service packages to use fx-based DI for consistency with internal packages. This will require:

1. Creating `module.go` files in each service package
2. Refactoring constructors to use constructor injection
3. Extracting types from monolithic files
4. Adding lifecycle hooks for proper resource management