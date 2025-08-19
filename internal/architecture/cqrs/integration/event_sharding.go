package integration

import (
	"context"
	"fmt"
	"hash/fnv"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// ShardingStrategy determines how events are sharded
type ShardingStrategy int

const (
	// NoSharding indicates no sharding
	NoSharding ShardingStrategy = iota
	
	// AggregateSharding shards events by aggregate ID
	AggregateSharding
	
	// TypeSharding shards events by event type
	TypeSharding
	
	// CustomSharding uses a custom sharding function
	CustomSharding
)

// ShardingConfig contains configuration for event sharding
type ShardingConfig struct {
	// Strategy determines the sharding strategy
	Strategy ShardingStrategy
	
	// ShardCount is the number of shards
	ShardCount int
	
	// CustomShardingFunc is a custom sharding function
	CustomShardingFunc func(event *eventsourcing.Event) int
}

// DefaultShardingConfig returns the default sharding configuration
func DefaultShardingConfig() ShardingConfig {
	return ShardingConfig{
		Strategy:   AggregateSharding,
		ShardCount: 10,
		CustomShardingFunc: nil,
	}
}

// EventShardingManager manages event sharding
type EventShardingManager struct {
	logger *zap.Logger
	
	// Configuration
	config ShardingConfig
	
	// NATS components
	conn   *nats.Conn
	js     nats.JetStreamContext
	
	// Streams
	streams []string
	
	// Synchronization
	mu     sync.RWMutex
}

// NewEventShardingManager creates a new event sharding manager
func NewEventShardingManager(
	logger *zap.Logger,
	config ShardingConfig,
	conn *nats.Conn,
	js nats.JetStreamContext,
) *EventShardingManager {
	return &EventShardingManager{
		logger:  logger,
		config:  config,
		conn:    conn,
		js:      js,
		streams: make([]string, 0),
	}
}

// Initialize initializes the event sharding manager
func (m *EventShardingManager) Initialize(ctx context.Context) error {
	// Check if sharding is enabled
	if m.config.Strategy == NoSharding {
		m.logger.Info("Event sharding is disabled")
		return nil
	}
	
	// Check if JetStream is available
	if m.js == nil {
		return fmt.Errorf("JetStream is required for event sharding")
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

// GetShardForEvent gets the shard for an event
func (m *EventShardingManager) GetShardForEvent(event *eventsourcing.Event) int {
	switch m.config.Strategy {
	case AggregateSharding:
		// Shard by aggregate ID
		return m.hashString(event.AggregateID) % m.config.ShardCount
		
	case TypeSharding:
		// Shard by event type
		return m.hashString(event.EventType) % m.config.ShardCount
		
	case CustomSharding:
		// Use the custom sharding function
		if m.config.CustomShardingFunc != nil {
			return m.config.CustomShardingFunc(event) % m.config.ShardCount
		}
		
		// Fall back to aggregate sharding
		return m.hashString(event.AggregateID) % m.config.ShardCount
		
	default:
		// No sharding
		return 0
	}
}

// GetSubjectForEvent gets the subject for an event
func (m *EventShardingManager) GetSubjectForEvent(event *eventsourcing.Event) string {
	// Check if sharding is enabled
	if m.config.Strategy == NoSharding {
		return fmt.Sprintf("events.%s", event.EventType)
	}
	
	// Get the shard for the event
	shard := m.GetShardForEvent(event)
	
	// Create the subject
	return fmt.Sprintf("events.shard.%d.%s", shard, event.EventType)
}

// hashString hashes a string to an integer
func (m *EventShardingManager) hashString(s string) int {
	h := fnv.New32a()
	h.Write([]byte(s))
	return int(h.Sum32())
}

// ShardingEventBusDecorator decorates an event bus with sharding
type ShardingEventBusDecorator struct {
	eventBus eventbus.EventBus
	manager  *EventShardingManager
	logger   *zap.Logger
}

// NewShardingEventBusDecorator creates a new sharding event bus decorator
func NewShardingEventBusDecorator(
	eventBus eventbus.EventBus,
	manager *EventShardingManager,
	logger *zap.Logger,
) *ShardingEventBusDecorator {
	return &ShardingEventBusDecorator{
		eventBus: eventBus,
		manager:  manager,
		logger:   logger,
	}
}

// PublishEvent publishes an event with sharding
func (d *ShardingEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Add the shard to the event metadata
	if event.Metadata == nil {
		event.Metadata = make(map[string]string)
	}
	
	shard := d.manager.GetShardForEvent(event)
	event.Metadata["shard"] = fmt.Sprintf("%d", shard)
	
	// Publish the event
	return d.eventBus.PublishEvent(ctx, event)
}

// PublishEvents publishes multiple events with sharding
func (d *ShardingEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
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
func (d *ShardingEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	return d.eventBus.Subscribe(handler)
}

// SubscribeToType subscribes to events of a specific type
func (d *ShardingEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToType(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *ShardingEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToAggregate(aggregateType, handler)
}

// NatsShardingEventBus is a NATS implementation of the EventBus interface with sharding
type NatsShardingEventBus struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	eventStore   store.EventStore
	handlers     []eventsourcing.EventHandler
	typeHandlers map[string][]eventsourcing.EventHandler
	aggHandlers  map[string][]eventsourcing.EventHandler
	logger       *zap.Logger
	mu           sync.RWMutex
	topicPrefix  string
	subs         []*nats.Subscription
	ctx          context.Context
	cancel       context.CancelFunc
	manager      *EventShardingManager
}

// NewNatsShardingEventBus creates a new NATS event bus with sharding
func NewNatsShardingEventBus(
	eventStore store.EventStore,
	logger *zap.Logger,
	config eventbus.NatsEventBusConfig,
	manager *EventShardingManager,
) (*NatsShardingEventBus, error) {
	// Create a context for managing subscriptions
	ctx, cancel := context.WithCancel(context.Background())
	
	// Create NATS connection options
	opts := []nats.Option{
		nats.Name("tradSys-event-bus"),
		nats.Timeout(config.ConnectionTimeout),
		nats.MaxReconnects(config.MaxReconnects),
		nats.ReconnectWait(config.ReconnectWait),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			logger.Warn("NATS disconnected", zap.Error(err))
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger.Info("NATS reconnected", zap.String("url", nc.ConnectedUrl()))
		}),
		nats.ErrorHandler(func(nc *nats.Conn, sub *nats.Subscription, err error) {
			logger.Error("NATS error", zap.Error(err), zap.String("subject", sub.Subject))
		}),
	}
	
	// Connect to NATS
	nc, err := nats.Connect(config.URLs[0], opts...)
	if err != nil {
		cancel()
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}
	
	// Create the event bus
	bus := &NatsShardingEventBus{
		conn:         nc,
		eventStore:   eventStore,
		handlers:     make([]eventsourcing.EventHandler, 0),
		typeHandlers: make(map[string][]eventsourcing.EventHandler),
		aggHandlers:  make(map[string][]eventsourcing.EventHandler),
		logger:       logger,
		topicPrefix:  config.TopicPrefix,
		subs:         make([]*nats.Subscription, 0),
		ctx:          ctx,
		cancel:       cancel,
		manager:      manager,
	}
	
	// Setup JetStream if enabled
	if config.UseJetStream {
		// Create JetStream context
		js, err := nc.JetStream()
		if err != nil {
			nc.Close()
			cancel()
			return nil, fmt.Errorf("failed to create JetStream context: %w", err)
		}
		
		bus.js = js
		
		// Initialize the sharding manager
		err = manager.Initialize(ctx)
		if err != nil {
			nc.Close()
			cancel()
			return nil, fmt.Errorf("failed to initialize sharding manager: %w", err)
		}
	}
	
	return bus, nil
}

// PublishEvent publishes an event with sharding
func (b *NatsShardingEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Marshal the event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}
	
	// Get the subject for the event
	subject := b.manager.GetSubjectForEvent(event)
	
	// Publish the event
	if b.js != nil {
		// Publish with JetStream
		_, err = b.js.Publish(subject, payload)
	} else {
		// Publish with standard NATS
		err = b.conn.Publish(subject, payload)
	}
	
	if err != nil {
		return fmt.Errorf("failed to publish event: %w", err)
	}
	
	return nil
}

// Other methods would be similar to NatsEventBus but with sharding support

