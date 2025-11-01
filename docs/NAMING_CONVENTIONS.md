# TradSys Naming Conventions

## File Naming Standards

### Split Module Naming Pattern
All split modules follow the consistent pattern: `{component}_{suffix}.go`

#### Suffix Definitions:
- **`_types.go`** - Type definitions, interfaces, constants, enums
- **`_core.go`** - Core logic, constructors, main public API methods
- **`_helpers.go`** - Helper functions, utilities, general calculations
- **`_handlers.go`** - Event handlers, message handlers, callback functions
- **`_checks.go`** - Validation functions, checks, rules (risk management specific)
- **`_websocket.go`** - WebSocket-specific functionality and streaming
- **`_calculations.go`** - Mathematical/statistical calculations (specialized algorithms)

### Examples of Consistent Naming:

#### Phase 1: Service Layer Modularization
```
services/assets/handler_registry_types.go
services/assets/handler_registry_core.go
services/assets/handler_registry_handlers.go

services/assets/unified_asset_system_types.go
services/assets/unified_asset_system_core.go

services/islamic/sharia_service_types.go
services/islamic/sharia_service_core.go
services/islamic/sharia_service_helpers.go

internal/services/etf_service_types.go
internal/services/etf_service_core.go
internal/services/etf_service_helpers.go

services/websocket/ws_gateway_types.go
services/websocket/ws_gateway_core.go
services/websocket/ws_gateway_handlers.go

services/trading/risk_manager_types.go
services/trading/risk_manager_core.go
services/trading/risk_manager_checks.go

services/common/types_core.go
services/common/types_extended.go
```

#### Phase 2: External Integration Modularization
```
internal/marketdata/external/binance_types.go
internal/marketdata/external/binance_core.go
internal/marketdata/external/binance_websocket.go
```

#### Phase 3: Algorithm Component Separation
```
internal/trading/strategies/statistical_arbitrage_types.go
internal/trading/strategies/statistical_arbitrage_core.go
internal/trading/strategies/statistical_arbitrage_calculations.go
```

## Naming Rationale

### Clear Separation of Concerns
Each suffix represents a specific responsibility:
- **Types**: Data structures and interfaces
- **Core**: Main business logic and public APIs
- **Helpers**: Utility functions and general calculations
- **Handlers**: Event processing and callbacks
- **Checks**: Validation and rule enforcement
- **WebSocket**: Real-time communication
- **Calculations**: Complex mathematical operations

### Consistency Benefits
1. **Predictable Structure**: Developers can easily locate specific functionality
2. **Maintainability**: Clear boundaries between different types of code
3. **Scalability**: Easy to extend with additional modules following the same pattern
4. **Code Review**: Reviewers can quickly understand file purposes
5. **IDE Navigation**: Consistent naming improves IDE autocomplete and search

## File Organization Principles

### Size Limits
- Target: <500 lines per file
- Maximum acceptable: <530 lines for complex algorithms
- Achieved: 87% reduction in files over 500 lines

### Content Guidelines
- **Types files**: Only type definitions, no business logic
- **Core files**: Main public API, constructors, lifecycle methods
- **Helper files**: Private utility functions, calculations
- **Handler files**: Event processing, callbacks, message handling
- **Check files**: Validation logic, rule enforcement
- **WebSocket files**: Real-time communication, streaming
- **Calculation files**: Mathematical algorithms, statistical operations

## Migration History

### Applied Changes
- **Phase 1-3 Splits**: All new files follow consistent naming
- **Consistency Fix**: Renamed `sharia_service_constructors.go` → `sharia_service_helpers.go`
- **Documentation**: Created this naming convention guide

### Future Considerations
- All new split modules should follow this naming pattern
- Consider splitting any remaining files >500 lines using these conventions
- Maintain consistency when adding new functionality to existing modules
