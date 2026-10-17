---
name: vet-fixes
description: Fixes for go vet warnings in trading strategies
metadata:
  type: reference
---

# Go Vet Fixes

## Issues Fixed

### 1. optimized_framework.go:233 - Protobuf Mutex Copy

**Problem:**
```
optimized_framework.go:233:14: assignment copies lock value to *dataCopy:
github.com/abdoElHodaky/tradSys/proto/marketdata.MarketDataResponse contains 
google.golang.org/protobuf/runtime/protoimpl.MessageState contains sync.Mutex
```

**Root Cause:**
The code was copying a protobuf message by dereferencing:
```go
dataCopy := m.marketDataPool.Get()
*dataCopy = *data  // Unsafe copy - copies mutex inside protobuf
```

**Fix:**
```go
import "google.golang.org/protobuf/proto"

dataCopy := proto.Clone(data).(*marketdata.MarketDataResponse)  // Safe clone
```

**Why:**
Protobuf messages contain runtime state (mutexes for thread safety). Direct copying can cause data races or panics. `proto.Clone` properly handles protobuf internals.

### 2. optimized_statistical_arbitrage.go:68,88 - OrderService Mutex Copy

**Problem:**
```
optimized_statistical_arbitrage.go:68:15: NewOptimizedStatisticalArbitrageStrategy passes lock by value: 
github.com/abdoElHodaky/tradSys/internal/orders.OrderService contains sync.RWMutex
```

**Root Cause:**
The `OptimizedStatisticalArbitrageStrategy` struct had `orderService` as a value type:
```go
type OptimizedStatisticalArbitrageStrategy struct {
    orderService orders.OrderService  // Copy of mutex!
    ...
}

func NewOptimizedStatisticalArbitrageStrategy(..., orderService orders.OrderService, ...) {
    return &OptimizedStatisticalArbitrageStrategy{
        orderService: orderService,  // Copies the mutex
    }
}
```

**Fix:**
```go
type OptimizedStatisticalArbitrageStrategy struct {
    orderService *orders.OrderService  // Use pointer
    ...
}

func NewOptimizedStatisticalArbitrageStrategy(..., orderService *orders.OrderService, ...) {
    return &OptimizedStatisticalArbitrageStrategy{
        orderService: orderService,
    }
}
```

**Why:**
Go's copylock analysis warns when structs containing `sync.Mutex` or `sync.RWMutex` are copied by value. The mutex state would be duplicated incorrectly.

## Verification

```bash
$ go build ./...        # No errors
$ go vet ./...          # No warnings
```

## Template for Future Fixes

When encountering copylock warnings:

1. **Identify the type** with mutex field
2. **Check if pointer semantics make sense** (shared state, mutable)
3. **Change field from T to *T** if appropriate
4. **Update all usages** to pass pointers instead of values
5. **Verify with `go vet`** the fix is complete