# TradSys Naming Standards

This document defines the naming conventions and standards for the TradSys high-frequency trading system.

## Table of Contents

1. [General Principles](#general-principles)
2. [Package Naming](#package-naming)
3. [Type Naming](#type-naming)
4. [Function and Method Naming](#function-and-method-naming)
5. [Variable Naming](#variable-naming)
6. [Constant Naming](#constant-naming)
7. [Interface Naming](#interface-naming)
8. [File Naming](#file-naming)
9. [Directory Structure](#directory-structure)
10. [Examples](#examples)

## General Principles

1. **Clarity over brevity**: Names should be descriptive and self-documenting
2. **Consistency**: Use consistent patterns throughout the codebase
3. **Go conventions**: Follow standard Go naming conventions
4. **Domain-specific**: Use trading domain terminology appropriately
5. **Avoid abbreviations**: Use full words unless the abbreviation is widely understood

## Package Naming

### Rules
- Use lowercase, single words when possible
- Use underscores only when necessary for clarity
- Avoid generic names like `util`, `common`, `helper`
- Use domain-specific names

### Examples
```go
// Good
package matching
package orders
package marketdata
package risk

// Avoid
package utils
package helpers
package common
```

### Package Structure
```
internal/
├── auth/           # Authentication and authorization
├── config/         # Configuration management
├── matching/       # Order matching engines
├── orders/         # Order management
├── marketdata/     # Market data handling
├── risk/           # Risk management
├── positions/      # Position tracking
├── trades/         # Trade execution and tracking
├── websocket/      # WebSocket connections
└── grpc/           # gRPC services
```

## Type Naming

### Services
Use the suffix `Service` for business logic services:
```go
type OrderService struct {}
type RiskService struct {}
type MarketDataService struct {}
```

### Handlers
Use the suffix `Handler` for HTTP/API handlers:
```go
type OrderHandler struct {}
type TradeHandler struct {}
type WebSocketHandler struct {}
```

### Repositories
Use the suffix `Repository` for data access layers:
```go
type OrderRepository struct {}
type TradeRepository struct {}
type UserRepository struct {}
```

### Managers
Use the suffix `Manager` for coordination/orchestration:
```go
type ConnectionManager struct {}
type SessionManager struct {}
type PoolManager struct {}
```

### Engines
Use the suffix `Engine` for processing engines:
```go
type MatchingEngine struct {}
type RiskEngine struct {}
type PricingEngine struct {}
```

### Clients
Use the suffix `Client` for external service clients:
```go
type ExchangeClient struct {}
type DatabaseClient struct {}
type RedisClient struct {}
```

### Configs
Use the suffix `Config` for configuration structs:
```go
type DatabaseConfig struct {}
type ServerConfig struct {}
type MatchingConfig struct {}
```

## Function and Method Naming

### Constructors
Use `New` prefix for constructors:
```go
func NewOrderService(repo OrderRepository) *OrderService
func NewMatchingEngine(config *MatchingConfig) *MatchingEngine
```

### Getters
Use `Get` prefix for retrieving single items:
```go
func (s *OrderService) GetOrder(id string) (*Order, error)
func (s *OrderService) GetOrderByClientID(clientID string) (*Order, error)
```

### Listers
Use `List` prefix for retrieving multiple items:
```go
func (s *OrderService) ListOrders(userID string) ([]*Order, error)
func (s *OrderService) ListActiveOrders() ([]*Order, error)
```

### Creators
Use `Create` prefix for creating new entities:
```go
func (s *OrderService) CreateOrder(order *Order) error
func (s *OrderService) CreateBulkOrders(orders []*Order) error
```

### Updaters
Use `Update` prefix for modifying existing entities:
```go
func (s *OrderService) UpdateOrder(order *Order) error
func (s *OrderService) UpdateOrderStatus(id string, status OrderStatus) error
```

### Deleters
Use `Delete` prefix for removing entities:
```go
func (s *OrderService) DeleteOrder(id string) error
func (s *OrderService) DeleteExpiredOrders() error
```

### Validators
Use `Validate` prefix for validation functions:
```go
func (s *OrderService) ValidateOrder(order *Order) error
func ValidatePrice(price float64) error
```

### Processors
Use `Process` prefix for processing functions:
```go
func (e *MatchingEngine) ProcessOrder(order *Order) ([]*Trade, error)
func (e *RiskEngine) ProcessRiskCheck(order *Order) error
```

## Variable Naming

### Local Variables
Use short, descriptive names for local variables:
```go
// Good
order := &Order{}
trades := make([]*Trade, 0)
userID := "user123"

// Avoid
o := &Order{}
t := make([]*Trade, 0)
uid := "user123"
```

### Receiver Names
Use short, consistent receiver names:
```go
// Good
func (s *OrderService) CreateOrder(order *Order) error
func (e *MatchingEngine) ProcessOrder(order *Order) error
func (c *OrderClient) GetOrder(id string) error

// Avoid
func (orderService *OrderService) CreateOrder(order *Order) error
func (this *MatchingEngine) ProcessOrder(order *Order) error
```

### Common Abbreviations
These abbreviations are acceptable:
- `id` for identifier
- `url` for URL
- `http` for HTTP
- `grpc` for gRPC
- `db` for database
- `ctx` for context

## Constant Naming

Use `UPPER_SNAKE_CASE` for constants:
```go
const (
    DefaultTimeout = 30 * time.Second
    MaxOrderSize   = 1000000
    MinPrice       = 0.01
)

// Enums use PascalCase with type prefix
type OrderStatus string

const (
    OrderStatusPending   OrderStatus = "pending"
    OrderStatusFilled    OrderStatus = "filled"
    OrderStatusCanceled  OrderStatus = "canceled"
    OrderStatusRejected  OrderStatus = "rejected"
)
```

## Interface Naming

### Single Method Interfaces
Use `-er` suffix for single method interfaces:
```go
type Matcher interface {
    Match(order *Order) ([]*Trade, error)
}

type Validator interface {
    Validate(order *Order) error
}

type Notifier interface {
    Notify(event *Event) error
}
```

### Multi-Method Interfaces
Use descriptive names without `-er` suffix:
```go
type OrderRepository interface {
    GetOrder(id string) (*Order, error)
    CreateOrder(order *Order) error
    UpdateOrder(order *Order) error
    DeleteOrder(id string) error
}

type MatchingEngine interface {
    ProcessOrder(order *Order) ([]*Trade, error)
    CancelOrder(id string) error
    GetOrderBook(symbol string) (*OrderBook, error)
}
```

## File Naming

### Rules
- Use lowercase with underscores
- Group related functionality in files
- Use descriptive names

### Examples
```
order_service.go        # OrderService implementation
order_repository.go     # OrderRepository implementation
matching_engine.go      # MatchingEngine implementation
websocket_handler.go    # WebSocket handlers
grpc_server.go         # gRPC server implementation
```

### Test Files
Use `_test.go` suffix:
```
order_service_test.go
matching_engine_test.go
integration_test.go
```

## Directory Structure

### Recommended Structure
```
internal/
├── auth/
│   ├── service.go
│   ├── handler.go
│   ├── repository.go
│   └── jwt.go
├── orders/
│   ├── service.go
│   ├── handler.go
│   ├── repository.go
│   └── types.go
├── matching/
│   ├── engine.go
│   ├── hft_engine.go
│   ├── order_book.go
│   └── interfaces.go
└── common/
    ├── errors/
    ├── pool/
    └── metrics/
```

## Examples

### Good Naming Examples
```go
// Service with clear responsibility
type OrderManagementService struct {
    repository OrderRepository
    validator  OrderValidator
    notifier   TradeNotifier
}

// Constructor with clear parameters
func NewOrderManagementService(
    repo OrderRepository,
    validator OrderValidator,
    notifier TradeNotifier,
) *OrderManagementService {
    return &OrderManagementService{
        repository: repo,
        validator:  validator,
        notifier:   notifier,
    }
}

// Method with clear intent
func (s *OrderManagementService) ProcessLimitOrder(
    ctx context.Context,
    order *Order,
) (*ProcessingResult, error) {
    if err := s.validator.ValidateOrder(order); err != nil {
        return nil, errors.Wrap(err, errors.ErrInvalidOrder, "order validation failed")
    }
    
    // Processing logic...
    return result, nil
}
```

### Avoid These Patterns
```go
// Avoid generic names
type Manager struct {} // Too generic
type Handler struct {} // Too generic
type Utils struct {}   // Avoid utils

// Avoid abbreviations
type OrdSvc struct {}  // Use OrderService
type MktData struct {} // Use MarketData
type UsrMgr struct {}  // Use UserManager

// Avoid unclear method names
func (s *Service) Do(data interface{}) error        // Unclear
func (s *Service) Handle(req *Request) *Response    // Generic
func (s *Service) Process(input []byte) []byte      // Vague
```

## Migration Guidelines

When refactoring existing code to follow these standards:

1. **Start with interfaces**: Define clear interfaces first
2. **Rename incrementally**: Don't rename everything at once
3. **Update tests**: Ensure tests reflect new naming
4. **Update documentation**: Keep docs in sync with code
5. **Use aliases temporarily**: For backward compatibility during migration

## Enforcement

These standards should be enforced through:

1. **Code reviews**: All PRs should follow these conventions
2. **Linting tools**: Configure golangci-lint with naming rules
3. **Documentation**: Keep this document updated
4. **Team training**: Ensure all team members understand these standards

## References

- [Effective Go](https://golang.org/doc/effective_go.html)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Uber Go Style Guide](https://github.com/uber-go/guide/blob/master/style.md)
