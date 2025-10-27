# Go 1.24 Migration and Feature Guide

## Overview

This guide explains how to leverage the new Go 1.24 features implemented in the tradSys codebase, including generic type aliases, enhanced JSON handling, and improved package organization.

## New Package Structure

### Before (Duplicated Structure)
```
internal/common/     # Internal utilities
pkg/common/          # Public utilities (overlapping)
internal/matching/   # Internal matching engines
pkg/matching/        # Public matching engines (overlapping)
```

### After (Optimized Structure)
```
pkg/
├── types/           # Core types and generic aliases
├── interfaces/      # Public API interfaces
├── matching/        # Public matching engine APIs
├── common/          # Reusable utilities and base classes
├── config/          # Configuration utilities
└── errors/          # Public error types

internal/
├── api/             # HTTP handlers and routes
├── core/            # Core business logic implementation
├── db/              # Database models and repositories
├── services/        # Internal service implementations
├── websocket/       # WebSocket implementation
└── grpc/            # gRPC service implementations
```

## Go 1.24 Generic Type Aliases

### Basic Usage

```go
import "github.com/abdoElHodaky/tradSys/pkg/types"

// Use generic type aliases for cleaner code
var attributes types.StringAttributes = make(types.StringAttributes)
attributes["priority"] = "high"
attributes["client_type"] = "institutional"

// Use sets for collections
var symbols types.SymbolSet = make(types.SymbolSet)
symbols["AAPL"] = struct{}{}
symbols["GOOGL"] = struct{}{}
```

### Enhanced Order Types

```go
order := types.Order{
    ID:       "order-123",
    Symbol:   "AAPL",
    Side:     types.OrderSideBuy,
    Type:     types.OrderTypeLimit,
    Price:    150.0,
    Quantity: 100.0,
    Status:   types.OrderStatusPending,
    
    // New Go 1.24 enhanced fields
    Attributes: types.OrderAttributes{
        "priority":     "high",
        "algo_params":  map[string]interface{}{"twap": true},
        "client_info":  "institutional",
    },
    Metadata: types.Metadata{
        "source":      "api",
        "version":     "v2",
        "created_by":  "user-456",
    },
}
```

## Result and Option Types

### Result Type for Error Handling

```go
import "github.com/abdoElHodaky/tradSys/pkg/types"

// Function that returns a Result
func ProcessOrder(order types.Order) types.Result[types.Trade] {
    if order.Quantity <= 0 {
        return types.NewResultWithError[types.Trade](
            types.NewError("invalid_quantity", "Order quantity must be positive"),
        )
    }
    
    trade := types.Trade{
        ID:       "trade-456",
        Symbol:   order.Symbol,
        Price:    order.Price,
        Quantity: order.Quantity,
    }
    
    return types.NewResult(trade)
}

// Usage
result := ProcessOrder(order)
if result.IsSuccess() {
    trade := result.Unwrap()
    fmt.Printf("Trade executed: %+v\n", trade)
} else {
    fmt.Printf("Error: %v\n", result.Error)
}
```

### Option Type for Nullable Values

```go
// Function that returns an Option
func FindOrder(orderID string) types.Option[types.Order] {
    // Simulate database lookup
    if order, exists := orderCache[orderID]; exists {
        return types.Some(order)
    }
    return types.None[types.Order]()
}

// Usage
orderOption := FindOrder("order-123")
if orderOption.IsSome() {
    order := orderOption.Unwrap()
    fmt.Printf("Found order: %+v\n", order)
} else {
    fmt.Println("Order not found")
}

// Or use UnwrapOr for default values
order := orderOption.UnwrapOr(types.Order{
    Status: types.OrderStatusCanceled,
})
```

## Generic Interfaces

### Using the New Matching Engine Interface

```go
import (
    "github.com/abdoElHodaky/tradSys/pkg/interfaces"
    "github.com/abdoElHodaky/tradSys/pkg/matching"
    "github.com/abdoElHodaky/tradSys/pkg/types"
)

// Create a new Go 1.24 optimized matching engine
var cache types.OrderCache = NewInMemoryCache()
var eventBus types.OrderEventBus = NewEventBus()
var riskManager interfaces.RiskManager = NewRiskManager()

engine := matching.NewGo124MatchingEngine(cache, eventBus, riskManager)

// Start the engine
ctx := context.Background()
if err := engine.Start(ctx); err != nil {
    log.Fatal("Failed to start matching engine:", err)
}

// Add an order
order := types.Order{
    ID:       "order-123",
    Symbol:   "AAPL",
    Side:     types.OrderSideBuy,
    Type:     types.OrderTypeLimit,
    Price:    150.0,
    Quantity: 100.0,
}

result, err := engine.AddOrder(ctx, order)
if err != nil {
    log.Printf("Error adding order: %v", err)
} else if result.IsSuccess() {
    trade := result.Unwrap()
    log.Printf("Trade executed: %+v", trade)
}
```

### Service Management with Generics

```go
import "github.com/abdoElHodaky/tradSys/pkg/common"

// Create a service registry for trading services
logger := zap.NewProduction()
registry := common.NewServiceRegistry[TradingService](logger)

// Register services
matchingService := NewMatchingService()
riskService := NewRiskService()

registry.Register("matching", matchingService)
registry.Register("risk", riskService)

// Retrieve services
matchingOption := registry.Get("matching")
if matchingOption.IsSome() {
    service := matchingOption.Unwrap()
    service.Start(context.Background())
}
```

## Enhanced Service Base Class

### Using the Go 1.24 Service Base

```go
import (
    "github.com/abdoElHodaky/tradSys/pkg/common"
    "github.com/abdoElHodaky/tradSys/pkg/types"
)

type MyTradingService struct {
    *common.Go124ServiceBase
    // Additional service-specific fields
}

func NewMyTradingService(logger *zap.Logger) *MyTradingService {
    base := common.NewGo124ServiceBase("my-trading-service", "1.0.0", logger)
    
    service := &MyTradingService{
        Go124ServiceBase: base,
    }
    
    // Set custom attributes
    service.SetAttribute("service_type", "trading")
    service.SetAttribute("max_connections", 1000)
    
    // Set metadata
    service.SetMetadata("deployment", "production")
    service.SetMetadata("region", "us-east-1")
    
    // Set custom health check
    service.SetHealthCheck(func(ctx context.Context, s *common.Go124ServiceBase) error {
        // Custom health check logic
        if !s.IsRunning() {
            return types.NewError("service_stopped", "Service is not running")
        }
        return nil
    })
    
    return service
}

// Override Start method for custom initialization
func (s *MyTradingService) Start(ctx context.Context) error {
    // Call base Start method
    if err := s.Go124ServiceBase.Start(ctx); err != nil {
        return err
    }
    
    // Custom initialization logic
    s.GetLogger().Info("Custom trading service started")
    
    return nil
}
```

## Error Handling with Enhanced Errors

### Creating and Using Enhanced Errors

```go
import "github.com/abdoElHodaky/tradSys/pkg/types"

// Create a basic error
err := types.NewError("validation_failed", "Order validation failed")

// Add details to the error
err = err.WithDetail("order_id", "12345")
err = err.WithDetail("symbol", "AAPL")
err = err.WithDetail("validation_rule", "insufficient_balance")

// Create error with cause
originalErr := errors.New("database connection failed")
wrappedErr := types.NewErrorWithCause(
    "order_save_failed", 
    "Failed to save order to database", 
    originalErr,
)

// Use in functions
func ValidateOrder(order types.Order) error {
    if order.Quantity <= 0 {
        return types.NewError("invalid_quantity", "Quantity must be positive").
            WithDetail("order_id", order.ID).
            WithDetail("quantity", order.Quantity)
    }
    
    if order.Price <= 0 {
        return types.NewError("invalid_price", "Price must be positive").
            WithDetail("order_id", order.ID).
            WithDetail("price", order.Price)
    }
    
    return nil
}
```

## Testing with Go 1.24 Features

### Unit Tests

```go
func TestOrderProcessing(t *testing.T) {
    // Test with enhanced order type
    order := types.Order{
        ID:       "test-order",
        Symbol:   "AAPL",
        Side:     types.OrderSideBuy,
        Type:     types.OrderTypeLimit,
        Price:    150.0,
        Quantity: 100.0,
        Attributes: types.OrderAttributes{
            "test_mode": true,
        },
    }
    
    // Test Result type
    result := ProcessOrder(order)
    assert.True(t, result.IsSuccess())
    
    trade := result.Unwrap()
    assert.Equal(t, "AAPL", trade.Symbol)
    assert.Equal(t, 150.0, trade.Price)
}

func TestServiceRegistry(t *testing.T) {
    logger := zap.NewNop()
    registry := common.NewServiceRegistry[TestService](logger)
    
    service := &TestService{Name: "test"}
    err := registry.Register("test", service)
    assert.NoError(t, err)
    
    retrievedOption := registry.Get("test")
    assert.True(t, retrievedOption.IsSome())
    
    retrieved := retrievedOption.Unwrap()
    assert.Equal(t, "test", retrieved.Name)
}
```

## Performance Considerations

### Benchmarking Generic Types

```go
func BenchmarkGenericVsInterface(b *testing.B) {
    b.Run("GenericCache", func(b *testing.B) {
        cache := NewGenericCache[string, types.Order]()
        order := types.Order{ID: "test"}
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            cache.Set("key", order, time.Hour)
            _, _ = cache.Get("key")
        }
    })
    
    b.Run("InterfaceCache", func(b *testing.B) {
        cache := NewInterfaceCache()
        order := types.Order{ID: "test"}
        
        b.ResetTimer()
        for i := 0; i < b.N; i++ {
            cache.Set("key", order, time.Hour)
            _, _ = cache.Get("key")
        }
    })
}
```

## Migration Checklist

### Phase 1: Update Imports
- [ ] Update imports to use `pkg/types` for core types
- [ ] Update imports to use `pkg/interfaces` for public interfaces
- [ ] Update imports to use `pkg/common` for utilities

### Phase 2: Adopt Generic Types
- [ ] Replace `map[string]interface{}` with `types.StringAttributes`
- [ ] Replace custom set implementations with `types.Set[T]`
- [ ] Use `types.Result[T]` for error-prone operations
- [ ] Use `types.Option[T]` for nullable values

### Phase 3: Update Service Implementations
- [ ] Extend `common.Go124ServiceBase` for new services
- [ ] Use generic service registry for service management
- [ ] Implement enhanced error handling with `types.TradingError`

### Phase 4: Testing and Validation
- [ ] Run comprehensive tests with new types
- [ ] Benchmark performance improvements
- [ ] Validate error handling improvements
- [ ] Test service lifecycle management

## Best Practices

1. **Use Generic Type Aliases**: Prefer `types.StringAttributes` over `map[string]interface{}`
2. **Leverage Result Types**: Use `types.Result[T]` for operations that can fail
3. **Embrace Option Types**: Use `types.Option[T]` instead of pointers for optional values
4. **Enhanced Error Handling**: Use `types.TradingError` with details for better debugging
5. **Service Management**: Use the generic service registry for type-safe service management
6. **Testing**: Write comprehensive tests for generic types and interfaces

## Troubleshooting

### Common Issues

1. **Type Inference Problems**: Explicitly specify generic type parameters when needed
2. **Interface Compatibility**: Ensure implementations satisfy the new generic interfaces
3. **Performance Regression**: Benchmark critical paths after migration
4. **Error Handling**: Update error handling to use the new enhanced error types

### Getting Help

- Check the test files in `pkg/types/` for usage examples
- Review the interface definitions in `pkg/interfaces/`
- Consult the architecture documentation in `docs/architecture/`

---

*This guide covers the major Go 1.24 features implemented in tradSys. For specific implementation details, refer to the source code and tests.*
