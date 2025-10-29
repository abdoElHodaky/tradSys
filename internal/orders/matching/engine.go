package order_matching

// Note: This file has been split into focused components for better maintainability:
// - engine_types.go: Type definitions, constants, and data structures (237 lines)
// - engine_core.go: Main service struct, constructor, and core API methods (382 lines)
// - engine_processors.go: Order matching algorithms and processing logic (400 lines)
//
// This split maintains the same functionality while improving code organization
// and adhering to the code splitting standards. The standard engine provides
// reliable order matching with heap-based priority queues.
//
// Key Features Preserved:
// - Heap-based order book management for efficient price-time priority
// - Support for market, limit, stop-limit, and stop-market orders
// - Comprehensive stop order triggering and execution
// - Thread-safe operations with proper mutex handling
// - Detailed trade execution logging and monitoring
//
// All order matching algorithms have been moved to engine_processors.go
// while maintaining the exact same matching behavior and performance characteristics.

