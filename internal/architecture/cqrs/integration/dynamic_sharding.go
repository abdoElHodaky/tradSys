package integration

import (
	"context"
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/integration/strategy"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// DynamicShardingConfig contains configuration for dynamic event sharding
type DynamicShardingConfig struct {
	// StrategyName is the name of the sharding strategy to use
	StrategyName string
	
	// ShardCount is the number of shards
	ShardCount int
	
	// PluginDir is the directory containing sharding strategy plugins
	PluginDir string
}

// DefaultDynamicShardingConfig returns the default dynamic sharding configuration
func DefaultDynamicShardingConfig() DynamicShardingConfig {
	return DynamicShardingConfig{
		StrategyName: "aggregate",
		ShardCount:   10,
		PluginDir:    "/etc/tradsys/cqrs/plugins/sharding",
	}
}

// DynamicEventShardingManager manages event sharding with dynamic strategies
type DynamicEventShardingManager struct {
	logger *zap.Logger
	
	// Configuration
	config DynamicShardingConfig
	
	// Strategy factory
	factory *strategy.ShardingStrategyFactory
	
	// Current strategy
	currentStrategy strategy.ShardingStrategy
	
	// NATS components
	conn   *nats.Conn
	js     nats.JetStreamContext
	
	// Streams
	streams []string
	
	// Synchronization
	mu     sync.RWMutex
}

// NewDynamicEventShardingManager creates a new dynamic event sharding manager
func NewDynamicEventShardingManager(
	logger *zap.Logger,
	config DynamicShardingConfig,
	conn *nats.Conn,
	js nats.JetStreamContext,
) *DynamicEventShardingManager {
	// Create the strategy factory
	factory := strategy.NewShardingStrategyFactory(logger)
	
	// Get the initial strategy
	currentStrategy, ok := factory.GetStrategy(config.StrategyName)
	if !ok {
		// Use the default strategy if the requested one is not found
		currentStrategy = factory.GetDefaultStrategy()
		logger.Warn("Requested sharding strategy not found, using default",
			zap.String("requested", config.StrategyName),
			zap.String("using", currentStrategy.GetName()),
		)
	}
	
	return &DynamicEventShardingManager{
		logger:          logger,
		config:          config,
		factory:         factory,
		currentStrategy: currentStrategy,
		conn:            conn,
		js:              js,
		streams:         make([]string, 0),
	}
}

// Initialize initializes the dynamic event sharding manager
func (m *DynamicEventShardingManager) Initialize(ctx context.Context) error {
	// Check if JetStream is available
	if m.js == nil {
		return fmt.Errorf("JetStream is required for event sharding")
	}
	
	// Load strategy plugins
	if m.config.PluginDir != "" {
		if err := m.loadStrategyPlugins(); err != nil {
			m.logger.Warn("Failed to load sharding strategy plugins", zap.Error(err))
		}
	}
	
	// Create streams for each shard
	for i := 0; i < m.config.ShardCount; i++ {
		streamName := fmt.Sprintf("events_shard_%d", i)
		
		// Create the stream
		_, err := m.js.StreamInfo(streamName)
		if err != nil {
			// Stream doesn't exist, create it
			streamConfig := &nats.StreamConfig{
				Name:      streamName,
				Subjects:  []string{fmt.Sprintf("events.shard.%d.>", i)},
				Retention: nats.LimitsPolicy,
				MaxAge:    24 * 60 * 60 * 1000 * 1000 * 1000, // 24 hours in nanoseconds
				MaxBytes:  1024 * 1024 * 1024, // 1GB
				Storage:   nats.FileStorage,
				Replicas:  1,
			}
			
			_, err = m.js.AddStream(streamConfig)
			if err != nil {
				return fmt.Errorf("failed to create stream %s: %w", streamName, err)
			}
		}
		
		// Add the stream to the list
		m.streams = append(m.streams, streamName)
		
		m.logger.Info("Created event shard stream", zap.String("stream", streamName))
	}
	
	return nil
}

// loadStrategyPlugins loads sharding strategy plugins
func (m *DynamicEventShardingManager) loadStrategyPlugins() error {
	// Check if the plugin directory exists
	if _, err := os.Stat(m.config.PluginDir); os.IsNotExist(err) {
		m.logger.Warn("Plugin directory does not exist", zap.String("directory", m.config.PluginDir))
		return nil
	}
	
	// Find all .so files in the plugin directory
	files, err := filepath.Glob(filepath.Join(m.config.PluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}
	
	for _, file := range files {
		if err := m.factory.LoadStrategyPlugin(file); err != nil {
			m.logger.Error("Failed to load sharding strategy plugin",
				zap.String("file", file),
				zap.Error(err))
			continue
		}
	}
	
	return nil
}

// GetShardForEvent gets the shard for an event
func (m *DynamicEventShardingManager) GetShardForEvent(event *eventsourcing.Event) int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.currentStrategy.GetShardForEvent(event, m.config.ShardCount)
}

// GetSubjectForEvent gets the subject for an event
func (m *DynamicEventShardingManager) GetSubjectForEvent(event *eventsourcing.Event) string {
	// Get the shard for the event
	shard := m.GetShardForEvent(event)
	
	// Create the subject
	return fmt.Sprintf("events.shard.%d.%s", shard, event.EventType)
}

// SetStrategy sets the current sharding strategy
func (m *DynamicEventShardingManager) SetStrategy(strategyName string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// Get the strategy
	strategy, ok := m.factory.GetStrategy(strategyName)
	if !ok {
		return fmt.Errorf("sharding strategy not found: %s", strategyName)
	}
	
	// Set the current strategy
	m.currentStrategy = strategy
	
	m.logger.Info("Set sharding strategy",
		zap.String("strategy", strategy.GetName()),
		zap.String("description", strategy.GetDescription()),
	)
	
	return nil
}

// GetCurrentStrategy gets the current sharding strategy
func (m *DynamicEventShardingManager) GetCurrentStrategy() strategy.ShardingStrategy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return m.currentStrategy
}

// RegisterStrategy registers a sharding strategy
func (m *DynamicEventShardingManager) RegisterStrategy(strategy strategy.ShardingStrategy) {
	m.factory.RegisterStrategy(strategy)
}

// DynamicShardingEventBusDecorator decorates an event bus with dynamic sharding
type DynamicShardingEventBusDecorator struct {
	eventBus eventbus.EventBus
	manager  *DynamicEventShardingManager
	logger   *zap.Logger
}

// NewDynamicShardingEventBusDecorator creates a new dynamic sharding event bus decorator
func NewDynamicShardingEventBusDecorator(
	eventBus eventbus.EventBus,
	manager *DynamicEventShardingManager,
	logger *zap.Logger,
) *DynamicShardingEventBusDecorator {
	return &DynamicShardingEventBusDecorator{
		eventBus: eventBus,
		manager:  manager,
		logger:   logger,
	}
}

// PublishEvent publishes an event with dynamic sharding
func (d *DynamicShardingEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Add the shard to the event metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	
	shard := d.manager.GetShardForEvent(event)
	event.Metadata["shard"] = fmt.Sprintf("%d", shard)
	
	// Publish the event
	return d.eventBus.PublishEvent(ctx, event)
}

// PublishEvents publishes multiple events with dynamic sharding
func (d *DynamicShardingEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	// Add the shard to each event's metadata
	for _, event := range events {
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		
		shard := d.manager.GetShardForEvent(event)
		event.Metadata["shard"] = fmt.Sprintf("%d", shard)
	}
	
	// Publish the events
	return d.eventBus.PublishEvents(ctx, events)
}

// Subscribe subscribes to all events
func (d *DynamicShardingEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	return d.eventBus.Subscribe(handler)
}

// SubscribeToType subscribes to events of a specific type
func (d *DynamicShardingEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToType(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *DynamicShardingEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToAggregate(aggregateType, handler)
}

