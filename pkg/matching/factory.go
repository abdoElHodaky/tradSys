// Package matching provides a centralized factory for creating matching engines.
// This package serves as the public API for engine creation and management.
package matching

import (
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
	"go.uber.org/zap"
)

// Type aliases for backward compatibility with tests
type (
	// Core types
	Engine = types.Engine
	Order  = types.Order
	Trade  = types.Trade
	
	// Configuration
	EngineConfig = types.EngineConfig
	EngineStats  = types.EngineStats
	EngineError  = types.EngineError
)

// Constants for backward compatibility
const (
	// Order sides
	SideBuy  = types.OrderSideBuy
	SideSell = types.OrderSideSell
	
	// Order types  
	TypeMarket = types.OrderTypeMarket
	TypeLimit  = types.OrderTypeLimit
	TypeStop   = types.OrderTypeStop
	
	// Time in force
	TimeInForceGTC = types.TimeInForceGTC
	TimeInForceIOC = types.TimeInForceIOC
	TimeInForceFOK = types.TimeInForceFOK
	
	// Order status
	StatusPending   = types.OrderStatusPending
	StatusPartial   = types.OrderStatusPartial
	StatusFilled    = types.OrderStatusFilled
	StatusCancelled = types.OrderStatusCancelled
)

// Helper functions for backward compatibility
func NewEngineError(code, message, details, severity string) *EngineError {
	return types.NewEngineError(code, message, details, severity)
}

func DefaultEngineConfig() *EngineConfig {
	return types.DefaultEngineConfig()
}

func NewEngineStats() *EngineStats {
	return types.NewEngineStats()
}

// EngineType represents the type of matching engine to create
type EngineType string

const (
	// EngineTypeBasic creates a basic matching engine suitable for standard trading
	EngineTypeBasic EngineType = "basic"
	
	// EngineTypeAdvanced creates an advanced matching engine with price improvement
	EngineTypeAdvanced EngineType = "advanced"
	
	// EngineTypeHFT creates a high-frequency trading optimized engine
	EngineTypeHFT EngineType = "hft"
	
	// EngineTypeOptimized creates an optimized engine for high throughput
	EngineTypeOptimized EngineType = "optimized"
	
	// EngineTypeCompliance creates a compliance-aware matching engine
	EngineTypeCompliance EngineType = "compliance"
)

// EngineFactory provides centralized engine creation
type EngineFactory struct {
	logger *zap.Logger
}

// NewEngineFactory creates a new engine factory
func NewEngineFactory(logger *zap.Logger) *EngineFactory {
	return &EngineFactory{
		logger: logger,
	}
}

// CreateEngine creates a new matching engine of the specified type
func (f *EngineFactory) CreateEngine(engineType EngineType, config *types.EngineConfig) (types.Engine, error) {
	if config == nil {
		config = types.DefaultEngineConfig()
	}
	
	f.logger.Info("Creating matching engine",
		zap.String("type", string(engineType)),
		zap.String("symbol", config.Symbol),
	)
	
	switch engineType {
	case EngineTypeBasic:
		return f.createBasicEngine(config)
	case EngineTypeAdvanced:
		return f.createAdvancedEngine(config)
	case EngineTypeHFT:
		return f.createHFTEngine(config)
	case EngineTypeOptimized:
		return f.createOptimizedEngine(config)
	case EngineTypeCompliance:
		return f.createComplianceEngine(config)
	default:
		return nil, fmt.Errorf("unknown engine type: %s", engineType)
	}
}

// NewEngine is a convenience function for creating engines with default factory
func NewEngine(engineType EngineType, config *types.EngineConfig, logger *zap.Logger) (types.Engine, error) {
	factory := NewEngineFactory(logger)
	return factory.CreateEngine(engineType, config)
}

// GetSupportedEngineTypes returns all supported engine types
func GetSupportedEngineTypes() []EngineType {
	return []EngineType{
		EngineTypeBasic,
		EngineTypeAdvanced,
		EngineTypeHFT,
		EngineTypeOptimized,
		EngineTypeCompliance,
	}
}

// ValidateEngineType checks if the engine type is supported
func ValidateEngineType(engineType EngineType) error {
	for _, supported := range GetSupportedEngineTypes() {
		if engineType == supported {
			return nil
		}
	}
	return fmt.Errorf("unsupported engine type: %s", engineType)
}

// GetEngineTypeDescription returns a description of the engine type
func GetEngineTypeDescription(engineType EngineType) string {
	switch engineType {
	case EngineTypeBasic:
		return "Basic matching engine suitable for standard trading operations"
	case EngineTypeAdvanced:
		return "Advanced matching engine with price improvement and sophisticated algorithms"
	case EngineTypeHFT:
		return "High-frequency trading optimized engine with ultra-low latency"
	case EngineTypeOptimized:
		return "Optimized engine for high throughput and performance"
	case EngineTypeCompliance:
		return "Compliance-aware matching engine with regulatory checks"
	default:
		return "Unknown engine type"
	}
}

// EngineCapabilities describes the capabilities of each engine type
type EngineCapabilities struct {
	SupportsHFT           bool
	SupportsPriceImprovement bool
	SupportsCompliance    bool
	MaxThroughput         int    // orders per second
	TypicalLatency        string // typical latency range
	RecommendedUseCase    string
}

// GetEngineCapabilities returns the capabilities of the specified engine type
func GetEngineCapabilities(engineType EngineType) *EngineCapabilities {
	switch engineType {
	case EngineTypeBasic:
		return &EngineCapabilities{
			SupportsHFT:           false,
			SupportsPriceImprovement: false,
			SupportsCompliance:    false,
			MaxThroughput:         10000,
			TypicalLatency:        "1-10ms",
			RecommendedUseCase:    "Standard retail trading, low to medium volume",
		}
	case EngineTypeAdvanced:
		return &EngineCapabilities{
			SupportsHFT:           false,
			SupportsPriceImprovement: true,
			SupportsCompliance:    true,
			MaxThroughput:         50000,
			TypicalLatency:        "500μs-5ms",
			RecommendedUseCase:    "Institutional trading, medium to high volume",
		}
	case EngineTypeHFT:
		return &EngineCapabilities{
			SupportsHFT:           true,
			SupportsPriceImprovement: true,
			SupportsCompliance:    true,
			MaxThroughput:         1000000,
			TypicalLatency:        "1-100μs",
			RecommendedUseCase:    "High-frequency trading, ultra-low latency requirements",
		}
	case EngineTypeOptimized:
		return &EngineCapabilities{
			SupportsHFT:           true,
			SupportsPriceImprovement: true,
			SupportsCompliance:    false,
			MaxThroughput:         500000,
			TypicalLatency:        "10-500μs",
			RecommendedUseCase:    "High-volume trading, optimized for throughput",
		}
	case EngineTypeCompliance:
		return &EngineCapabilities{
			SupportsHFT:           false,
			SupportsPriceImprovement: true,
			SupportsCompliance:    true,
			MaxThroughput:         25000,
			TypicalLatency:        "1-5ms",
			RecommendedUseCase:    "Regulated markets, compliance-first trading",
		}
	default:
		return nil
	}
}

// createBasicEngine creates a basic matching engine
func (f *EngineFactory) createBasicEngine(config *types.EngineConfig) (types.Engine, error) {
	// Create a basic engine implementation
	engine := &BasicEngine{
		BaseEngine: types.NewBaseEngine(config, f.logger),
	}
	
	f.logger.Info("Created basic matching engine",
		zap.String("symbol", config.Symbol),
		zap.Int("max_depth", config.MaxOrderBookDepth),
	)
	
	return engine, nil
}

// createAdvancedEngine creates an advanced matching engine
func (f *EngineFactory) createAdvancedEngine(config *types.EngineConfig) (types.Engine, error) {
	// Create an advanced engine implementation
	engine := &AdvancedEngine{
		BaseEngine: types.NewBaseEngine(config, f.logger),
		priceImprovementEnabled: true,
	}
	
	f.logger.Info("Created advanced matching engine",
		zap.String("symbol", config.Symbol),
		zap.Bool("price_improvement", true),
	)
	
	return engine, nil
}

// createHFTEngine creates a high-frequency trading engine
func (f *EngineFactory) createHFTEngine(config *types.EngineConfig) (types.Engine, error) {
	// Optimize config for HFT
	hftConfig := *config
	hftConfig.EnableOptimizations = true
	hftConfig.BatchSize = 1 // Process orders immediately
	
	engine := &HFTEngine{
		BaseEngine: types.NewBaseEngine(&hftConfig, f.logger),
		ultraLowLatency: true,
	}
	
	f.logger.Info("Created HFT matching engine",
		zap.String("symbol", config.Symbol),
		zap.Duration("min_latency_target", config.MinLatencyTarget),
	)
	
	return engine, nil
}

// createOptimizedEngine creates an optimized matching engine
func (f *EngineFactory) createOptimizedEngine(config *types.EngineConfig) (types.Engine, error) {
	// Optimize config for throughput
	optimizedConfig := *config
	optimizedConfig.EnableOptimizations = true
	optimizedConfig.BatchSize = 1000 // Process in larger batches
	
	engine := &OptimizedEngine{
		BaseEngine: types.NewBaseEngine(&optimizedConfig, f.logger),
		highThroughput: true,
	}
	
	f.logger.Info("Created optimized matching engine",
		zap.String("symbol", config.Symbol),
		zap.Int("batch_size", optimizedConfig.BatchSize),
	)
	
	return engine, nil
}

// createComplianceEngine creates a compliance-aware matching engine
func (f *EngineFactory) createComplianceEngine(config *types.EngineConfig) (types.Engine, error) {
	engine := &ComplianceEngine{
		BaseEngine: types.NewBaseEngine(config, f.logger),
		complianceEnabled: true,
	}
	
	f.logger.Info("Created compliance matching engine",
		zap.String("symbol", config.Symbol),
		zap.Bool("compliance_enabled", true),
	)
	
	return engine, nil
}

// RecommendEngineType recommends an engine type based on requirements
func RecommendEngineType(requirements *EngineRequirements) EngineType {
	if requirements.RequiresCompliance && requirements.MaxLatency.Milliseconds() > 1 {
		return EngineTypeCompliance
	}
	
	if requirements.MaxLatency.Microseconds() < 100 {
		return EngineTypeHFT
	}
	
	if requirements.ExpectedThroughput > 100000 {
		return EngineTypeOptimized
	}
	
	if requirements.RequiresPriceImprovement {
		return EngineTypeAdvanced
	}
	
	return EngineTypeBasic
}

// EngineRequirements describes the requirements for engine selection
type EngineRequirements struct {
	ExpectedThroughput       int           // orders per second
	MaxLatency              time.Duration // maximum acceptable latency
	RequiresCompliance      bool          // requires compliance checks
	RequiresPriceImprovement bool          // requires price improvement
	RequiresHFT             bool          // requires HFT capabilities
}
