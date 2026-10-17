---
name: kiss-modernization-plan
description: KISS principle application for large Go files
metadata:
  type: reference
---

# KISS (Keep It Simple, Stupid) Modernization Plan

## Large File Analysis

Files exceeding 500 lines need refactoring to follow KISS - separating concerns and reducing complexity.

### Top Priority Files

| File | Lines | Primary Issue | Action |
|------|-------|---------------|--------|
| `internal/orders/service.go` | 1084 | Types + logic mixed | Split to `types.go` |
| `services/assets/unified_asset_system.go` | 914 | Asset types in service | Split to `asset_types.go` |
| `services/exchanges/adx_service.go` | 886 | Has types bundled | Already has interfaces |
| `services/optimization/performance_optimizer.go` | 711 | Optimizer types | Extract to `optimizer_types.go` |

## KISS Principles Applied

### 1. Single Responsibility Principle
Each file should have ONE reason to change.

**Before:** `service.go` contains types, constants, AND service logic.

**After:** 
- `types.go` - Only type definitions
- `service.go` - Only business logic
- `constants.go` - Only constants
- `errors.go` - Only error definitions

### 2. Separation of Concerns

```
package orders

// types.go
type Order struct {...}
type Trade struct {...}
type OrderStatus string

// constants.go  
const (
    OrderStatusNew OrderStatus = "new"
    OrderStatusPending OrderStatus = "pending"
    // ...
)

// service.go
type Service struct {...}
func (s *Service) SubmitOrder(...) {...}
func (s *Service) GetOrder(...) {...}
```

### 3. KISS Implementation Steps

#### Step 1: Identify Type Definitions
Look for `type X struct {}`, `type X string`, `const (` blocks.

#### Step 2: Create `*_types.go` Files
Move all type definitions, exceptions:
- Errors/constants (different file)
- Interface definitions
- Test data

#### Step 3: Clean Imports
Remove unused imports from each file.
Use standard library types where possible (`any` instead of `interface{}`).

#### Step 4: Update References
Ensure all files compile together:
```bash
go build ./...
```

## File-Specific Plans

### internal/orders/service.go → Split Into:

1. **`internal/orders/types.go`** (estimated ~300 lines)
   - Order, Trade, OrderFilter, OrderRequest, OrderCancelRequest, OrderUpdateRequest
   - OrderStatus, OrderType, OrderSide, TimeInForce types and constants

2. **`internal/orders/service.go`** (reduced to ~500 lines)
   - OrderService struct
   - NewService function
   - SubmitOrder, GetOrder, CancelOrder, etc.

3. **`internal/orders/errors.go`** (new file)
   - ErrOrderNotFound, ErrInvalidOrder, etc.

4. **`internal/orders/module.go`** (existing, keep as is)
   - Fx module for orders

### services/assets/unified_asset_system.go → Split Into:

1. **`services/assets/asset_types.go`**
   - AssetType, AssetInfo, MarketData, etc.
   - ComplianceInfo, IslamicInfo types

2. **`services/assets/unified_asset_system.go`**
   - UnifiedAssetService
   - All service logic

3. **`services/assets/config.go`**
   - AssetServiceConfig
   - Configuration types

### services/optimization/performance_optimizer.go → Split Into:

1. **`services/optimization/optimizer_types.go`**
   - CacheOptimizer, DatabaseOptimizer, NetworkOptimizer
   - RegionalOptimizer, AutoScaler types
   - Metrics types: CacheMetrics, DatabaseMetrics, etc.

2. **`services/optimization/performance_optimizer.go`**
   - PerformanceOptimizer struct
   - Optimization logic

### services/exchanges/adx_service.go

**Status:** Partial improvement needed
- Had types, moved many to islamic_types.go
- Created interfaces.go for shared interfaces
- Still have ADX-specific types that could be extracted

**Next Steps:**
1. Create `adx_types.go` for ADX-specific types
2. Create `adx_service.go` for service logic only

## Quick Wins (Apply Immediately)

### Use Type Aliases for Standards

```go
// Instead of:
type OrderStatus int

// Use (if compatible):
type OrderStatus = string  // Type alias
```

### Extract Functions

Long functions (>50 lines) should be broken down:

```go
// Instead of one long function:
func (s *Service) ComplexOperation() {
    // 100 lines of code
}

// Break into:
func (s *Service) ComplexOperation() {
    s.step1()
    s.step2()
    s.step3()
}

func (s *Service) step1() { /* 30 lines */ }
func (s *Service) step2() { /* 40 lines */ }
func (s *Service) step3() { /* 20 lines */ }
```

### Use Go 1.21+ Patterns

```go
// Use any instead of interface{}
map[string]any

// Use slices.Contains instead of manual loops
slices.Contains(slice, value)

// Use fmt.Errorf with %w for errors
return fmt.Errorf("operation failed: %w", err)
```

## Verification Checklist

After each file split:

- [ ] `go build ./...` passes
- [ ] `go vet ./...` passes  
- [ ] All types accessible from same package
- [ ] No circular imports
- [ ] Functions return early (reduce nesting)
- [ ] Comments are minimal (Go is self-documenting)

## Implementation Order

1. **`internal/orders/service.go`** - Most critical (1084 lines)
2. **`services/assets/unified_asset_system.go`** - Asset management
3. **`services/optimization/performance_optimizer.go`** - Optimizer
4. **Review `services/exchanges/adx_service.go`** - Final cleanup

## Benefits of KISS Refactoring

| Metric | Before | After |
|--------|--------|-------|
| File size | 1084 lines | ~300-500 lines |
| Complexity | High (many responsibilities) | Low (single purpose) |
| Testability | Hard (tightly coupled) | Easier (separated concerns) |
| Merge conflicts | Frequent (large files) | Reduced (smaller files) |
| Code review | Slow (many lines) | Faster (focused changes) |

## How to Extend

For each new feature:
1. Create types in `*_types.go`
2. Put logic in service files
3. Use interfaces for dependencies
4. Keep functions under 50 lines
5. Package related types together