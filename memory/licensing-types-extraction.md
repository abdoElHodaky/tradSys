---
name: licensing-types-extraction
description: Extracted license-related types from unified_asset_system.go to licensing service
metadata:
  type: reference
---

# Licensing Types Extraction

## Types Extracted from unified_asset_system.go

The following types were extracted from `/services/assets/unified_asset_system.go` to the licensing service:

### License-Related Types
- `LicenseValidator` - validates licenses for asset access
- `QuotaManager` - manages usage quotas for licenses
- `BillingEngine` - handles billing calculations
- `UsageTracker` - tracks usage for billing

### Compliance-Related Types
- `EgyptianComplianceInfo` - Egyptian market compliance information
- `UAEComplianceInfo` - UAE market compliance information
- `IslamicComplianceInfo` - Islamic finance compliance information
- `GlobalComplianceInfo` - Global market compliance information
- `UnifiedComplianceInfo` - Compliance information across jurisdictions
- `ComplianceRuleSet` - Compliance rules definition
- `UnifiedAuditTrail` - Audit event tracking
- `ComplianceReportingEngine` - Compliance report generation
- `ComplianceAlertManager` - Compliance alerts management

### Pricing-Related Types
- `PricingInfo` - Pricing and market data information
- `PricingModel` - Interface for pricing models
- `DataAggregator` - Market data aggregation
- `PriceCache` - Price data caching
- `RealTimeFeed` - Real-time market data feed

## Files Created
- `/services/licensing/implementation.go` - License validator, quota manager, billing engine, usage tracker implementations
- `/services/licensing/compliance_types.go` - Compliance-related types
- `/services/licensing/pricing_types.go` - Pricing-related types

## Notes
- The websocket service has its own `LicenseValidator` implementation which may need to be updated to use the licensing service types
- The `UnifiedLicensingManager` from unified_asset_system.go wraps these components
- Future work may involve creating a compliance service for compliance management

**Why:** These types were scattered in a large unified_asset_system.go file, making the codebase harder to maintain and understand. Extracting them to their respective services improves code organization and reusability.

**How to apply:** Use the types from the licensing service package (`github.com/abdoElHodaky/tradSys/services/licensing`) instead of the types defined in unified_asset_system.go.