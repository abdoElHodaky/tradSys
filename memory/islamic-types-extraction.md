---
name: islamic-types-extraction
description: Documentation for Islamic types extracted from monolithic files
metadata:
  type: reference
---

# Islamic Types Extraction

## Overview

Islamic finance types and compliance infrastructure were extracted from monolithic files to improve organization and maintainability.

## Files Modified

### services/exchanges/islamic_types.go (NEW)

Contains all Islamic finance related types:
- `ComplianceLevel` - enum for Sharia compliance levels
- `ShariaBoard` - Sharia supervisory board
- `ShariaScholar` - Sharia scholar
- `ShariaRule` - Islamic finance rule
- `IslamicCompliance` - Sharia compliance manager
- `UAECompliance` - UAE regulatory compliance
- `SCACompliance` - Saudi compliance
- `ZakatCalculator` - Zakat calculator for Islamic finance
- `IslamicPortfolio` - Islamic investment portfolio
- And related types...

### services/exchanges/adx_service.go

Contains UAE Exchange specific service:
- `ADXService` - main ADX service
- `ADXConnector` - ADX connection handler
- `ADXMarketData` - market data handling
- `SukukService` - Sukuk trading
- `IslamicFundService` - Islamic funds
- etc.

## Extraction Logic

Since both files are in the same `exchanges` package, there's no import cycle. The types in `islamic_types.go` provide the shared infrastructure, while `adx_service.go` provides the UAE Exchange-specific implementations.

## Why Important

- Separates domain types from service implementations
- Enables reuse of Islamic compliance types across different exchange services
- Follows Go best practices for code organization

## How to Extend

When adding new Islamic finance types:
1. Add to `islamic_types.go` for shared types
2. Add to specific exchange service files for exchange-specific types
3. Use type aliases if needed for backward compatibility