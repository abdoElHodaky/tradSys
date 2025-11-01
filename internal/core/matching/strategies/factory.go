// Package strategies provides impact calculation strategy implementations
package strategies

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
)

// DefaultImpactCalculatorFactory implements the ImpactCalculatorFactory interface
type DefaultImpactCalculatorFactory struct {
	creators map[string]func(liquidityFactor float64) interfaces.ImpactCalculator
	mu       sync.RWMutex
}

// NewDefaultImpactCalculatorFactory creates a new factory with default strategies
func NewDefaultImpactCalculatorFactory() interfaces.ImpactCalculatorFactory {
	factory := &DefaultImpactCalculatorFactory{
		creators: make(map[string]func(liquidityFactor float64) interfaces.ImpactCalculator),
	}

	// Register default strategies
	factory.RegisterModel("linear", NewLinearImpactCalculator)
	factory.RegisterModel("sqrt", NewSqrtImpactCalculator)
	factory.RegisterModel("log", NewLogImpactCalculator)

	return factory
}

// CreateCalculator creates an impact calculator for the specified model
func (f *DefaultImpactCalculatorFactory) CreateCalculator(modelName string, liquidityFactor float64) (interfaces.ImpactCalculator, error) {
	f.mu.RLock()
	creator, exists := f.creators[modelName]
	f.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("unknown impact model: %s (available: %v)", modelName, f.GetAvailableModels())
	}

	return creator(liquidityFactor), nil
}

// GetAvailableModels returns a list of available impact models
func (f *DefaultImpactCalculatorFactory) GetAvailableModels() []string {
	f.mu.RLock()
	defer f.mu.RUnlock()

	models := make([]string, 0, len(f.creators))
	for model := range f.creators {
		models = append(models, model)
	}
	return models
}

// RegisterModel registers a new impact calculation model
func (f *DefaultImpactCalculatorFactory) RegisterModel(modelName string, creator func(liquidityFactor float64) interfaces.ImpactCalculator) error {
	if creator == nil {
		return fmt.Errorf("creator function cannot be nil")
	}

	f.mu.Lock()
	defer f.mu.Unlock()

	f.creators[modelName] = creator
	return nil
}
