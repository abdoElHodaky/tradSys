package order_matching

// Note: This file has been split into focused components for ultra-high performance:
// - hft_types.go: Type definitions, constants, and data structures (270 lines)
// - hft_core.go: Main service struct, constructor, and core API methods (331 lines)
// - hft_processors.go: Ultra-critical matching algorithms and processing logic (444 lines)
//
// This split maintains the same functionality while improving maintainability
// and adhering to the code splitting standards. The HFT engine maintains its
// <100μs latency requirements through zero-overhead abstractions.
//
// Performance Requirements Preserved:
// - <100μs order processing latency
// - Zero-allocation matching algorithms
// - Lock-free data structures and atomic operations
// - Object pooling for memory management
//
// All ultra-critical matching algorithms have been moved to hft_processors.go
// while maintaining the exact same performance characteristics.
