// Package exchanges implements ADX (Abu Dhabi Exchange) service for TradSys v3
// ADX Service provides UAE Exchange integration with Islamic finance focus

package exchanges

// Note: This file has been split into focused components for better maintainability:
// - adx_types.go: Type definitions, constants, and data structures (287 lines)
// - adx_core.go: Main service struct, constructor, and core API methods (361 lines)
// - adx_processors.go: Validation methods, helper functions, and processing logic (390 lines)
//
// This split maintains the same functionality while improving code organization
// and adhering to the code splitting standards. The ADX service provides
// comprehensive Abu Dhabi Exchange integration with Islamic finance focus.
//
// Key Features Preserved:
// - Islamic finance compliance with Sharia board integration
// - Sukuk (Islamic bonds) trading capabilities
// - Islamic mutual funds support
// - Zakat calculation for Islamic investments
// - UAE regulatory compliance (SCA, ADGM, DIFC)
// - Real-time market data with Islamic filtering
// - Multi-language support (English/Arabic)
// - Comprehensive order validation and risk management
// - Performance monitoring and health checks
//
// All validation and processing logic has been moved to adx_processors.go
// while maintaining the exact same API and functionality with enhanced
// Islamic finance capabilities for the UAE market.
