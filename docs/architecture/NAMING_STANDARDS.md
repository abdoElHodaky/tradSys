# 📋 TradSys v3 Naming Standards & Structure Guidelines

## 🎯 **NAMING CONSISTENCY OBJECTIVES**

**Phase 19 Goal**: Achieve 100% naming consistency across the entire codebase

---

## 📁 **DIRECTORY STRUCTURE STANDARDS**

### **Core Structure**
```
internal/
├── api/                    # API layer
│   ├── handlers/          # HTTP handlers
│   ├── middleware/        # HTTP middleware
│   └── routes/           # Route definitions
├── core/                  # Core business logic
│   ├── matching/         # Order matching engine
│   ├── settlement/       # Trade settlement
│   └── pricing/          # Pricing engine
├── services/             # Business services
│   ├── orders/          # Order management
│   ├── risk/            # Risk management
│   ├── market/          # Market data
│   └── auth/            # Authentication
├── adapters/             # External adapters
│   ├── database/        # Database adapters
│   ├── messaging/       # Message queue adapters
│   └── external/        # External API adapters
└── shared/               # Shared utilities
    ├── types/           # Common types
    ├── errors/          # Error definitions
    └── utils/           # Utility functions
```

---

## 🏷️ **FILE NAMING CONVENTIONS**

### **Service Files**
- **Pattern**: `{domain}_service.go`
- **Examples**: 
  - `order_service.go`
  - `risk_service.go`
  - `market_service.go`

### **Handler Files**
- **Pattern**: `{domain}_handler.go`
- **Examples**:
  - `order_handler.go`
  - `risk_handler.go`
  - `market_handler.go`

### **Type Files**
- **Pattern**: `{domain}_types.go`
- **Examples**:
  - `order_types.go`
  - `risk_types.go`
  - `trade_types.go`

### **Interface Files**
- **Pattern**: `{domain}_interface.go`
- **Examples**:
  - `order_interface.go`
  - `risk_interface.go`
  - `market_interface.go`

### **Error Files**
- **Pattern**: `{domain}_errors.go`
- **Examples**:
  - `order_errors.go`
  - `risk_errors.go`
  - `market_errors.go`

---

## 🔧 **INTERFACE NAMING STANDARDS**

### **Service Interfaces**
- **Pattern**: `{Domain}Service`
- **Examples**:
  ```go
  type OrderService interface {}
  type RiskService interface {}
  type MarketService interface {}
  ```

### **Repository Interfaces**
- **Pattern**: `{Domain}Repository`
- **Examples**:
  ```go
  type OrderRepository interface {}
  type RiskRepository interface {}
  type MarketRepository interface {}
  ```

### **Handler Interfaces**
- **Pattern**: `{Domain}Handler`
- **Examples**:
  ```go
  type OrderHandler interface {}
  type RiskHandler interface {}
  type MarketHandler interface {}
  ```

---

## 📦 **PACKAGE NAMING STANDARDS**

### **Internal Packages**
- **Pattern**: Single word, lowercase
- **Examples**: `orders`, `risk`, `market`, `auth`

### **Service Packages**
- **Pattern**: `{domain}svc` (if disambiguation needed)
- **Examples**: `ordersvc`, `risksvc`, `marketsvc`

### **Handler Packages**
- **Pattern**: `handlers` (grouped by domain)
- **Structure**:
  ```
  internal/api/handlers/
  ├── order_handler.go
  ├── risk_handler.go
  └── market_handler.go
  ```

---

## 🏗️ **STRUCT NAMING STANDARDS**

### **Service Structs**
- **Pattern**: `{Domain}Service`
- **Examples**:
  ```go
  type OrderService struct {}
  type RiskService struct {}
  type MarketService struct {}
  ```

### **Handler Structs**
- **Pattern**: `{Domain}Handler`
- **Examples**:
  ```go
  type OrderHandler struct {}
  type RiskHandler struct {}
  type MarketHandler struct {}
  ```

### **Repository Structs**
- **Pattern**: `{Domain}Repository`
- **Examples**:
  ```go
  type OrderRepository struct {}
  type RiskRepository struct {}
  type MarketRepository struct {}
  ```

---

## 🔄 **METHOD NAMING STANDARDS**

### **CRUD Operations**
- **Create**: `Create{Entity}()`
- **Read**: `Get{Entity}()`, `List{Entities}()`, `Find{Entity}()`
- **Update**: `Update{Entity}()`
- **Delete**: `Delete{Entity}()`

### **Business Operations**
- **Pattern**: Verb + Noun
- **Examples**:
  ```go
  PlaceOrder()
  CancelOrder()
  CalculateRisk()
  ValidateOrder()
  ProcessTrade()
  ```

### **Query Operations**
- **Pattern**: `Get` + Description
- **Examples**:
  ```go
  GetOrderByID()
  GetOrdersByUser()
  GetActiveOrders()
  GetRiskMetrics()
  ```

---

## 📊 **CONSTANT NAMING STANDARDS**

### **Error Codes**
- **Pattern**: `Err{Domain}{Description}`
- **Examples**:
  ```go
  ErrOrderNotFound
  ErrOrderInvalid
  ErrRiskLimitExceeded
  ErrMarketClosed
  ```

### **Status Constants**
- **Pattern**: `{Domain}Status{Value}`
- **Examples**:
  ```go
  OrderStatusPending
  OrderStatusFilled
  RiskStatusNormal
  RiskStatusWarning
  ```

### **Type Constants**
- **Pattern**: `{Domain}Type{Value}`
- **Examples**:
  ```go
  OrderTypeLimit
  OrderTypeMarket
  RiskTypePosition
  RiskTypeExposure
  ```

---

## 🔗 **DEPENDENCY INJECTION STANDARDS**

### **Constructor Functions**
- **Pattern**: `New{Service}()`
- **Examples**:
  ```go
  func NewOrderService() *OrderService
  func NewRiskService() *RiskService
  func NewMarketService() *MarketService
  ```

### **Factory Functions**
- **Pattern**: `Create{Entity}()`
- **Examples**:
  ```go
  func CreateOrder() *Order
  func CreateTrade() *Trade
  func CreateRiskLimit() *RiskLimit
  ```

---

## 📝 **LOGGING STANDARDS**

### **Logger Fields**
- **Pattern**: snake_case
- **Examples**:
  ```go
  logger.Info("Order placed",
    zap.String("order_id", orderID),
    zap.String("user_id", userID),
    zap.Float64("quantity", quantity))
  ```

### **Log Messages**
- **Pattern**: Action + Context
- **Examples**:
  - "Order placed successfully"
  - "Risk limit exceeded"
  - "Market data updated"

---

## 🧪 **TEST NAMING STANDARDS**

### **Test Files**
- **Pattern**: `{source_file}_test.go`
- **Examples**:
  - `order_service_test.go`
  - `risk_handler_test.go`
  - `market_types_test.go`

### **Test Functions**
- **Pattern**: `Test{Function}_{Scenario}`
- **Examples**:
  ```go
  func TestPlaceOrder_Success()
  func TestPlaceOrder_InvalidQuantity()
  func TestCalculateRisk_ExceedsLimit()
  ```

### **Benchmark Functions**
- **Pattern**: `Benchmark{Function}`
- **Examples**:
  ```go
  func BenchmarkPlaceOrder()
  func BenchmarkCalculateRisk()
  func BenchmarkMatchOrders()
  ```

---

## 🔄 **MIGRATION PLAN**

### **Phase 19 Implementation Steps**

1. **File Renaming** (Batch 1)
   - Standardize service files
   - Standardize handler files
   - Update import statements

2. **Interface Standardization** (Batch 2)
   - Rename interfaces to standard pattern
   - Update implementations
   - Update dependency injection

3. **Method Standardization** (Batch 3)
   - Rename methods to standard pattern
   - Update all callers
   - Update tests

4. **Constant Standardization** (Batch 4)
   - Rename constants to standard pattern
   - Update all references
   - Update documentation

5. **Package Reorganization** (Batch 5)
   - Move files to standard locations
   - Update import paths
   - Update build configurations

---

## ✅ **VALIDATION CHECKLIST**

### **File Level**
- [ ] File names follow standard pattern
- [ ] Package names are consistent
- [ ] Import statements are organized

### **Code Level**
- [ ] Interface names follow standard
- [ ] Struct names follow standard
- [ ] Method names follow standard
- [ ] Constant names follow standard

### **Documentation Level**
- [ ] Comments follow standard format
- [ ] API documentation is consistent
- [ ] Error messages are standardized

### **Test Level**
- [ ] Test files follow naming pattern
- [ ] Test functions follow naming pattern
- [ ] Benchmark functions follow naming pattern

---

## 🎯 **SUCCESS METRICS**

- **100% File Naming Consistency**
- **100% Interface Naming Consistency**
- **100% Method Naming Consistency**
- **100% Constant Naming Consistency**
- **Zero Naming Convention Violations**

---

*Naming Standards - TradSys v3 | Phase 19: Structure & Naming Finalization*
