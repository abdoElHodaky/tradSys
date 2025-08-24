package strategy

import (
	"hash/fnv"
	"plugin"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// ShardingStrategy defines the interface for event sharding strategies
type ShardingStrategy interface {
	// GetShardForEvent returns the shard for an event
	GetShardForEvent(event *eventsourcing.Event, shardCount int) int
	
	// GetName returns the name of the strategy
	GetName() string
	
	// GetDescription returns the description of the strategy
	GetDescription() string
}

// AggregateShardingStrategy shards events by aggregate ID
type AggregateShardingStrategy struct{}

// GetShardForEvent returns the shard for an event based on aggregate ID
func (s *AggregateShardingStrategy) GetShardForEvent(event *eventsourcing.Event, shardCount int) int {
	return hashString(event.AggregateID) % shardCount
}

// GetName returns the name of the strategy
func (s *AggregateShardingStrategy) GetName() string {
	return "aggregate"
}

// GetDescription returns the description of the strategy
func (s *AggregateShardingStrategy) GetDescription() string {
	return "Shards events by aggregate ID"
}

// TypeShardingStrategy shards events by event type
type TypeShardingStrategy struct{}

// GetShardForEvent returns the shard for an event based on event type
func (s *TypeShardingStrategy) GetShardForEvent(event *eventsourcing.Event, shardCount int) int {
	return hashString(event.EventType) % shardCount
}

// GetName returns the name of the strategy
func (s *TypeShardingStrategy) GetName() string {
	return "type"
}

// GetDescription returns the description of the strategy
func (s *TypeShardingStrategy) GetDescription() string {
	return "Shards events by event type"
}

// CustomShardingStrategy uses a custom function to shard events
type CustomShardingStrategy struct {
	name        string
	description string
	shardFunc   func(event *eventsourcing.Event, shardCount int) int
}

// NewCustomShardingStrategy creates a new custom sharding strategy
func NewCustomShardingStrategy(
	name string,
	description string,
	shardFunc func(event *eventsourcing.Event, shardCount int) int,
) *CustomShardingStrategy {
	return &CustomShardingStrategy{
		name:        name,
		description: description,
		shardFunc:   shardFunc,
	}
}

// GetShardForEvent returns the shard for an event using the custom function
func (s *CustomShardingStrategy) GetShardForEvent(event *eventsourcing.Event, shardCount int) int {
	return s.shardFunc(event, shardCount)
}

// GetName returns the name of the strategy
func (s *CustomShardingStrategy) GetName() string {
	return s.name
}

// GetDescription returns the description of the strategy
func (s *CustomShardingStrategy) GetDescription() string {
	return s.description
}

// hashString hashes a string to an integer
func hashString(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

// ShardingStrategyFactory creates sharding strategies
type ShardingStrategyFactory struct {
	strategies map[string]ShardingStrategy
	logger     *zap.Logger
}

// NewShardingStrategyFactory creates a new sharding strategy factory
func NewShardingStrategyFactory(logger *zap.Logger) *ShardingStrategyFactory {
	factory := &ShardingStrategyFactory{
		strategies: make(map[string]ShardingStrategy),
		logger:     logger,
	}
	
	// Register built-in strategies
	factory.RegisterStrategy(&AggregateShardingStrategy{})
	factory.RegisterStrategy(&TypeShardingStrategy{})
	
	return factory
}

// RegisterStrategy registers a sharding strategy
func (f *ShardingStrategyFactory) RegisterStrategy(strategy ShardingStrategy) {
	f.strategies[strategy.GetName()] = strategy
	f.logger.Info("Registered sharding strategy",
		zap.String("name", strategy.GetName()),
		zap.String("description", strategy.GetDescription()),
	)
}

// GetStrategy returns a sharding strategy by name
func (f *ShardingStrategyFactory) GetStrategy(name string) (ShardingStrategy, bool) {
	strategy, ok := f.strategies[name]
	return strategy, ok
}

// GetDefaultStrategy returns the default sharding strategy
func (f *ShardingStrategyFactory) GetDefaultStrategy() ShardingStrategy {
	return &AggregateShardingStrategy{}
}

// LoadStrategyPlugin loads a sharding strategy from a plugin
func (f *ShardingStrategyFactory) LoadStrategyPlugin(path string) error {
	// Open the plugin
	p, err := plugin.Open(path)
	if err != nil {
		return err
	}
	
	// Look up the CreateStrategy symbol
	createSymbol, err := p.Lookup("CreateStrategy")
	if err != nil {
		return err
	}
	
	// Check the symbol type
	createFunc, ok := createSymbol.(func() ShardingStrategy)
	if !ok {
		return fmt.Errorf("CreateStrategy has wrong signature")
	}
	
	// Create the strategy
	strategy := createFunc()
	
	// Register the strategy
	f.RegisterStrategy(strategy)
	
	return nil
}

