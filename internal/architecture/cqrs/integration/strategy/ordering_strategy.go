package strategy

import (
	"fmt"
	"plugin"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// OrderingStrategy defines the interface for event ordering strategies
type OrderingStrategy interface {
	// ValidateEvent validates the ordering of an event
	ValidateEvent(event *eventsourcing.Event) error
	
	// GetName returns the name of the strategy
	GetName() string
	
	// GetDescription returns the description of the strategy
	GetDescription() string
	
	// GetStatistics returns statistics about the strategy
	GetStatistics() (processed int64, violations int64)
}

// AggregateOrderingStrategy validates ordering within a single aggregate
type AggregateOrderingStrategy struct {
	logger             *zap.Logger
	aggregateSequences map[string]int64 // aggregateID:aggregateType -> sequence
	violations         int64
	processed          int64
	mu                 sync.RWMutex
}

// NewAggregateOrderingStrategy creates a new aggregate ordering strategy
func NewAggregateOrderingStrategy(logger *zap.Logger) *AggregateOrderingStrategy {
	return &AggregateOrderingStrategy{
		logger:             logger,
		aggregateSequences: make(map[string]int64),
		violations:         0,
		processed:          0,
	}
}

// ValidateEvent validates the ordering of an event within an aggregate
func (s *AggregateOrderingStrategy) ValidateEvent(event *eventsourcing.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.processed++
	
	// Check aggregate sequence
	aggregateKey := event.AggregateID + ":" + event.AggregateType
	if seq, ok := s.aggregateSequences[aggregateKey]; ok {
		if event.Version <= seq {
			s.violations++
			return fmt.Errorf("aggregate ordering violation: event version %d <= aggregate sequence %d for aggregate %s",
				event.Version, seq, aggregateKey)
		}
	}
	s.aggregateSequences[aggregateKey] = event.Version
	
	return nil
}

// GetName returns the name of the strategy
func (s *AggregateOrderingStrategy) GetName() string {
	return "aggregate"
}

// GetDescription returns the description of the strategy
func (s *AggregateOrderingStrategy) GetDescription() string {
	return "Validates ordering within a single aggregate"
}

// GetStatistics returns statistics about the strategy
func (s *AggregateOrderingStrategy) GetStatistics() (processed int64, violations int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.processed, s.violations
}

// TypeOrderingStrategy validates ordering within an event type
type TypeOrderingStrategy struct {
	logger         *zap.Logger
	typeSequences  map[string]int64 // eventType -> sequence
	violations     int64
	processed      int64
	mu             sync.RWMutex
}

// NewTypeOrderingStrategy creates a new type ordering strategy
func NewTypeOrderingStrategy(logger *zap.Logger) *TypeOrderingStrategy {
	return &TypeOrderingStrategy{
		logger:         logger,
		typeSequences:  make(map[string]int64),
		violations:     0,
		processed:      0,
	}
}

// ValidateEvent validates the ordering of an event within a type
func (s *TypeOrderingStrategy) ValidateEvent(event *eventsourcing.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.processed++
	
	// Check type sequence
	if seq, ok := s.typeSequences[event.EventType]; ok {
		if event.Version <= seq {
			s.violations++
			return fmt.Errorf("type ordering violation: event version %d <= type sequence %d for type %s",
				event.Version, seq, event.EventType)
		}
	}
	s.typeSequences[event.EventType] = event.Version
	
	return nil
}

// GetName returns the name of the strategy
func (s *TypeOrderingStrategy) GetName() string {
	return "type"
}

// GetDescription returns the description of the strategy
func (s *TypeOrderingStrategy) GetDescription() string {
	return "Validates ordering within an event type"
}

// GetStatistics returns statistics about the strategy
func (s *TypeOrderingStrategy) GetStatistics() (processed int64, violations int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.processed, s.violations
}

// GlobalOrderingStrategy validates global ordering of all events
type GlobalOrderingStrategy struct {
	logger         *zap.Logger
	globalSequence int64
	violations     int64
	processed      int64
	mu             sync.RWMutex
}

// NewGlobalOrderingStrategy creates a new global ordering strategy
func NewGlobalOrderingStrategy(logger *zap.Logger) *GlobalOrderingStrategy {
	return &GlobalOrderingStrategy{
		logger:         logger,
		globalSequence: 0,
		violations:     0,
		processed:      0,
	}
}

// ValidateEvent validates the global ordering of an event
func (s *GlobalOrderingStrategy) ValidateEvent(event *eventsourcing.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.processed++
	
	// Check global sequence
	if event.Version <= s.globalSequence {
		s.violations++
		return fmt.Errorf("global ordering violation: event version %d <= global sequence %d",
			event.Version, s.globalSequence)
	}
	s.globalSequence = event.Version
	
	return nil
}

// GetName returns the name of the strategy
func (s *GlobalOrderingStrategy) GetName() string {
	return "global"
}

// GetDescription returns the description of the strategy
func (s *GlobalOrderingStrategy) GetDescription() string {
	return "Validates global ordering of all events"
}

// GetStatistics returns statistics about the strategy
func (s *GlobalOrderingStrategy) GetStatistics() (processed int64, violations int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.processed, s.violations
}

// CustomOrderingStrategy uses a custom function to validate event ordering
type CustomOrderingStrategy struct {
	name        string
	description string
	validateFunc func(event *eventsourcing.Event) error
	violations   int64
	processed    int64
	mu           sync.RWMutex
}

// NewCustomOrderingStrategy creates a new custom ordering strategy
func NewCustomOrderingStrategy(
	name string,
	description string,
	validateFunc func(event *eventsourcing.Event) error,
) *CustomOrderingStrategy {
	return &CustomOrderingStrategy{
		name:         name,
		description:  description,
		validateFunc: validateFunc,
		violations:   0,
		processed:    0,
	}
}

// ValidateEvent validates the ordering of an event using the custom function
func (s *CustomOrderingStrategy) ValidateEvent(event *eventsourcing.Event) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.processed++
	
	err := s.validateFunc(event)
	if err != nil {
		s.violations++
	}
	
	return err
}

// GetName returns the name of the strategy
func (s *CustomOrderingStrategy) GetName() string {
	return s.name
}

// GetDescription returns the description of the strategy
func (s *CustomOrderingStrategy) GetDescription() string {
	return s.description
}

// GetStatistics returns statistics about the strategy
func (s *CustomOrderingStrategy) GetStatistics() (processed int64, violations int64) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.processed, s.violations
}

// OrderingStrategyFactory creates ordering strategies
type OrderingStrategyFactory struct {
	strategies map[string]OrderingStrategy
	logger     *zap.Logger
}

// NewOrderingStrategyFactory creates a new ordering strategy factory
func NewOrderingStrategyFactory(logger *zap.Logger) *OrderingStrategyFactory {
	factory := &OrderingStrategyFactory{
		strategies: make(map[string]OrderingStrategy),
		logger:     logger,
	}
	
	// Register built-in strategies
	factory.RegisterStrategy(NewAggregateOrderingStrategy(logger))
	factory.RegisterStrategy(NewTypeOrderingStrategy(logger))
	factory.RegisterStrategy(NewGlobalOrderingStrategy(logger))
	
	return factory
}

// RegisterStrategy registers an ordering strategy
func (f *OrderingStrategyFactory) RegisterStrategy(strategy OrderingStrategy) {
	f.strategies[strategy.GetName()] = strategy
	f.logger.Info("Registered ordering strategy",
		zap.String("name", strategy.GetName()),
		zap.String("description", strategy.GetDescription()),
	)
}

// GetStrategy returns an ordering strategy by name
func (f *OrderingStrategyFactory) GetStrategy(name string) (OrderingStrategy, bool) {
	strategy, ok := f.strategies[name]
	return strategy, ok
}

// GetDefaultStrategy returns the default ordering strategy
func (f *OrderingStrategyFactory) GetDefaultStrategy() OrderingStrategy {
	return NewAggregateOrderingStrategy(f.logger)
}

// LoadStrategyPlugin loads an ordering strategy from a plugin
func (f *OrderingStrategyFactory) LoadStrategyPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}
	
	// Look up the CreateStrategy symbol
	createSymbol, err := p.Lookup("CreateOrderingStrategy")
	if err != nil {
		return err
	}
	
	// Check the symbol type
	createFunc, ok := createSymbol.(func(*zap.Logger) OrderingStrategy)
	if !ok {
		return fmt.Errorf("CreateOrderingStrategy has wrong signature")
	}
	
	// Create the strategy
	strategy := createFunc(f.logger)
	
	// Register the strategy
	f.RegisterStrategy(strategy)
	
	return nil
}

