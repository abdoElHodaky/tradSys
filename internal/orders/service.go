package orders

// Note: This file has been split into focused components for better maintainability:
// - service_types.go: Type definitions, constants, and data structures (299 lines)
// - service_core.go: Main service struct, constructor, and core API methods (399 lines)
// - service_processors.go: Batch processing logic, validation, and helper functions (387 lines)
//
// This split maintains the same functionality while improving code organization
// and adhering to the code splitting standards. The order service provides
// comprehensive order lifecycle management with batch processing optimization.
//
// Key Features Preserved:
// - Complete order lifecycle management (create, update, cancel, expire)
// - Batch processing for high-throughput operations
// - Comprehensive caching with configurable expiration
// - User and symbol-based order indexing for fast lookups
// - Order validation and limit enforcement
// - Integration with matching engine for order execution
// - Background expiry checking and automatic cancellation
//
// All order processing logic has been moved to service_processors.go
// while maintaining the exact same API and functionality.

