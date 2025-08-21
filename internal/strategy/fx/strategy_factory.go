package fx

import (
	"context"
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"github.com/abdoElHodaky/tradSys/internal/strategy/optimized"
	"go.uber.org/zap"
)

// StrategyFactory creates trading strategies
type StrategyFactory struct {
	logger                *zap.Logger
	workerPoolFactory     *workerpool.WorkerPoolFactory
	circuitBreakerFactory *resilience.CircuitBreakerFactory
	registry              *StrategyRegistry
	strategyCreators      map[string]StrategyCreator
	mu                    sync.RWMutex
}

// StrategyCreator is a function that creates a strategy
type StrategyCreator func(config strategy.StrategyConfig) (strategy.Strategy, error)

// NewStrategyFactory creates a new StrategyFactory
func NewStrategyFactory(
	params StrategyFactoryParams,
	registry *StrategyRegistry,
) *StrategyFactory {
	factory := &StrategyFactory{
		logger:                params.Logger,
		workerPoolFactory:     params.WorkerPoolFactory,
		circuitBreakerFactory: params.CircuitBreakerFactory,
		registry:              registry,
		strategyCreators:      make(map[string]StrategyCreator),
	}

	// Register built-in strategy types
	factory.registerBuiltInStrategies()

	return factory
}

// registerBuiltInStrategies registers built-in strategy types
func (f *StrategyFactory) registerBuiltInStrategies() {
	// Register mean reversion strategy
	f.RegisterStrategyType("mean_reversion", func(config strategy.StrategyConfig) (strategy.Strategy, error) {
		return f.createMeanReversionStrategy(config)
	})

	// Additional built-in strategies can be registered here
}

// RegisterStrategyType registers a strategy type
func (f *StrategyFactory) RegisterStrategyType(strategyType string, creator StrategyCreator) {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.strategyCreators[strategyType] = creator
	f.logger.Info("Registered strategy type", zap.String("type", strategyType))
}

// CreateStrategy creates a strategy from a configuration
func (f *StrategyFactory) CreateStrategy(config strategy.StrategyConfig) (strategy.Strategy, error) {
	f.mu.RLock()
	creator, exists := f.strategyCreators[config.Type]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown strategy type: %s", config.Type)
	}

	// Create the strategy
	s, err := creator(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create strategy: %w", err)
	}

	// Register the strategy
	f.registry.RegisterStrategy(s)

	return s, nil
}

// GetAvailableStrategyTypes returns the available strategy types
func (f *StrategyFactory) GetAvailableStrategyTypes() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]string, 0, len(f.strategyCreators))
	for t := range f.strategyCreators {
		types = append(types, t)
	}

	return types
}

// createMeanReversionStrategy creates a mean reversion strategy
func (f *StrategyFactory) createMeanReversionStrategy(config strategy.StrategyConfig) (strategy.Strategy, error) {
	// Extract parameters
	params := config.Parameters

	// Get required parameters with defaults
	lookbackPeriod := getIntParam(params, "lookback_period", 20)
	updateInterval := getIntParam(params, "update_interval", 5)
	stdDevPeriod := getIntParam(params, "std_dev_period", 20)
	entryThreshold := getFloatParam(params, "entry_threshold", 2.0)
	exitThreshold := getFloatParam(params, "exit_threshold", 0.5)

	// Create the strategy
	s := optimized.NewMeanReversionStrategy(
		config.Name,
		config.Symbols,
		lookbackPeriod,
		updateInterval,
		stdDevPeriod,
		entryThreshold,
		exitThreshold,
		f.workerPoolFactory,
		f.circuitBreakerFactory,
		f.logger,
	)

	return s, nil
}

// Helper functions for parameter extraction
func getIntParam(params map[string]interface{}, key string, defaultValue int) int {
	if val, ok := params[key]; ok {
		if intVal, ok := val.(int); ok {
			return intVal
		}
		if floatVal, ok := val.(float64); ok {
			return int(floatVal)
		}
	}
	return defaultValue
}

func getFloatParam(params map[string]interface{}, key string, defaultValue float64) float64 {
	if val, ok := params[key]; ok {
		if floatVal, ok := val.(float64); ok {
			return floatVal
		}
		if intVal, ok := val.(int); ok {
			return float64(intVal)
		}
	}
	return defaultValue
}

func getStringParam(params map[string]interface{}, key string, defaultValue string) string {
	if val, ok := params[key]; ok {
		if strVal, ok := val.(string); ok {
			return strVal
		}
	}
	return defaultValue
}

func getBoolParam(params map[string]interface{}, key string, defaultValue bool) bool {
	if val, ok := params[key]; ok {
		if boolVal, ok := val.(bool); ok {
			return boolVal
		}
	}
	return defaultValue
}

