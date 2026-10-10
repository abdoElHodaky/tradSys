---
name: large-file-extraction
description: Plan for extracting types from large Go files for better organization
metadata:
  type: project
---

# Large File Extraction Plan

## Files Identified for Extraction

### 1. services/exchanges/adx_service.go (979 lines)

**Types to Extract:**
- `ComplianceLevel`, `ComplianceRule`, `ComplianceSeverity` (shared with EGX)
- `IslamicCompliance`, `NewIslamicCompliance`
- `ZakatCalculator`, `NewZakatCalculator`
- `UAECompliance`, `NewUAECompliance`, `SCACompliance`
- `SukukService`, `SukukType`, `SukukRiskEngine`, `RiskModel`
- `IslamicYieldCalculator`, `SukukPricingEngine`
- `IslamicFundService`, `IslamicFundType`, `IslamicNAVCalculator`
- `FundScreeningEngine`, `IslamicPerformanceCalculator`, `PerformanceData`
- Data structures: `SukukData`, `IslamicFundData`, `FundPerformance`, `IslamicHolding`, `IslamicPortfolio`, `ZakatCalculation`, `ShariaComplianceReport`, `ScreeningResult`
- `AuditEntry`, `IslamicAuditTrail`, `NewIslamicAuditTrail`
- `KYCRequirements`, `ReportingRequirements`, `LicensingRequirements`
- `ComplianceDatabase`, `ComplianceDataStore`
- `ShariaBoard`, `ShariaScholar`, `ShariaRule`

**Approach:**
- Create `services/exchanges/islamic_types.go` for Islamic-specific types
- Extract EGX compliance types to `services/exchanges/egyptian_compliance.go`

### 2. services/assets/handler_registry.go (680 lines)

**Types to Extract:**
- **Handler Types:** `HandlerRegistry`, `AssetHandler` (interface)
- **Settlement Types:** `Settlement`
- **Trading Types:** `TradingHours`
- **Risk Types:** `RiskParameters`
- **Fee Types:** `FeeStructure`, `FeeCalculation`
- **Handler Structs:** `BaseAssetHandler`, `StockHandler`, `SukukHandler`

**Approach:**
- Create `services/assets/handler_types.go` for type definitions
- Keep handler implementations in `handler_registry.go`
- Consider moving handlers to separate files per asset type

### 3. services/optimization/performance_optimizer.go (711 lines)

**Types to Extract:**
- `CacheOptimizer`, `DatabaseOptimizer`, `NetworkOptimizer`
- `RegionalOptimizer`, `RegionConfig`, `EdgeNode`
- `AutoScaler`, `ScalingPolicy`
- `ComprehensiveMonitoring`, `MetricsCollector`
- Metrics types: `CacheMetrics`, `DatabaseMetrics`, `NetworkMetrics`, `RegionalMetrics`, `SecurityMetrics`, `SystemMetrics`
- `PerformanceAlert`, `PerformanceIssue`
- `SecurityOptimizer` (and sub-types)

**Approach:**
- Create `services/optimization/optimizer_types.go` for all types
- Keep implementation methods in the main file

### 4. services/websocket/websocket_gateway.go (707 lines)

**Status:** Already partially separated
- `websocket_components.go` (594 lines) has components
- `websocket_gateway.go` has core gateway and connection types

**Types Remaining:**
- `WebSocketGateway`, `WebSocketConnection`, `WebSocketConnectionContext`
- `WebSocketMessage`, `MessageType`, `ExchangeType`, `LicenseTier`
- `Subscription`, `ExchangeChannel`

**Approach:**
- Consider moving message/connection types to `websocket_types.go`

## DI Injection Mapping

### Current Manual Injection Pattern

```go
// services/exchanges/adx_service.go
func NewADXService() *ADXService {
    return &ADXService{
        islamicCompliance:  NewIslamicCompliance(),
        uaeCompliance:      NewUAECompliance(),
        connector:          NewADXConnector(),
        marketData:         NewADXMarketData(),
        orderManager:       NewADXOrderManager(),
        riskEngine:         NewADXRiskEngine(),
        sukukService:       NewSukukService(),
        islamicFundService: NewIslamicFundService(),
    }
}
```

### Target fx-Based Injection Pattern

```go
// services/exchanges/module.go
var ADXModule = fx.Options(
    fx.Provide(NewADXService),
)

func NewADXService(
    logger *zap.Logger,
    cfg *Config,
    compliance *IslamicCompliance,
    connector *ADXConnector,
) *ADXService {
    return &ADXService{
        logger:           logger,
        config:           cfg,
        compliance:       compliance,
        connector:        connector,
        // ...
    }
}
```

## Implementation Order

1. **First Phase:** Create type extraction files for types that are shared or reusable
2. **Second Phase:** Remove duplicate type definitions from original files
3. **Third Phase:** Create fx modules for DI
4. **Fourth Phase:** Update tests and documentation

## Files Fixed After This Work

- [x] `services/assets/unified_asset_system.go` - Types extracted to licensing package
- [x] `services/assets/position_types.go` - Created
- [x] `services/assets/config_types.go` - Created
- [x] `services/assets/search_types.go` - Created

## Next Steps

1. Create `services/exchanges/islamic_types.go` for Islamic finance types
2. Create `services/assets/handler_types.go` for handler types
3. Create `services/optimization/types.go` for optimization types
4. Create `services/websocket/types.go` for websocket types
5. Consider fx module creation for services packages