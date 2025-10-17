package eventbus

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	nats "github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// NatsEventBus provides an implementation of the EventBus interface using NATS
type NatsEventBus struct {
	conn         *nats.Conn
	js           nats.JetStreamContext
	eventStore   store.EventStore
	handlers     []eventsourcing.EventHandler
	typeHandlers map[string][]eventsourcing.EventHandler
	aggHandlers  map[string][]eventsourcing.EventHandler
	logger       *zap.Logger
	mu           sync.RWMutex
	topicPrefix  string
	
	// Subscriptions
	subs         []*nats.Subscription
	
	// Context for managing subscriptions
	ctx          context.Context
	cancel       context.CancelFunc
}

// NatsEventBusConfig contains configuration for the NatsEventBus
type NatsEventBusConfig struct {
	// URLs is a list of NATS server URLs
	URLs []string
	
	// TopicPrefix is the prefix for all topics
	TopicPrefix string
	
	// ConnectionTimeout is the timeout for connecting to NATS
	ConnectionTimeout time.Duration
	
	// MaxReconnects is the maximum number of reconnect attempts
	MaxReconnects int
	
	// ReconnectWait is the time to wait between reconnect attempts
	ReconnectWait time.Duration
	
	// UseJetStream determines if JetStream should be used
	UseJetStream bool
	
	// StreamConfig is the configuration for the JetStream stream
	StreamConfig *nats.StreamConfig
}

// DefaultNatsEventBusConfig returns the default configuration
func DefaultNatsEventBusConfig() NatsEventBusConfig {
	return NatsEventBusConfig{
		URLs:              []string{nats.DefaultURL},
		TopicPrefix:       "events.",
		ConnectionTimeout: 5 * time.Second,
		MaxReconnects:     10,
		ReconnectWait:     1 * time.Second,
		UseJetStream:      true,
		StreamConfig: &nats.StreamConfig{
			Name:      "events",
			Subjects:  []string{"events.>"},
			Retention: nats.LimitsPolicy,
			MaxAge:    24 * time.Hour,
			MaxBytes:  1024 * 1024 * 1024, // 1GB
			Storage:   nats.FileStorage,
			Replicas:  1,
		},
	}
}

// NewNatsEventBus creates a new NatsEventBus
func NewNatsEventBus(eventStore store.EventStore, logger *zap.Logger, config NatsEventBusConfig) (*NatsEventBus, error) {
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
	bus := &NatsEventBus{
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
		
		// Create the stream if it doesn't exist
		_, err = js.StreamInfo(config.StreamConfig.Name)
		if err != nil {
			// Stream doesn't exist, create it
			_, err = js.AddStream(config.StreamConfig)
			if err != nil {
				nc.Close()
				cancel()
				return nil, fmt.Errorf("failed to create JetStream stream: %w", err)
			}
		}
		
		bus.js = js
	}
	
	return bus, nil
}

// Start starts the event bus
func (b *NatsEventBus) Start() error {
	// Nothing to do here as NATS subscriptions are started when registered
	return nil
}

// Stop stops the event bus
func (b *NatsEventBus) Stop() error {
	// Cancel the context to stop all subscriptions
	b.cancel()
	
	// Drain and close all subscriptions
	for _, sub := range b.subs {
		err := sub.Drain()
		if err != nil {
			b.logger.Error("Failed to drain subscription", zap.Error(err))
		}
	}
	
	// Close the connection
	b.conn.Close()
	
	return nil
}

// PublishEvent publishes an event
func (b *NatsEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
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
	
	// Create the subject
	subject := b.topicPrefix + event.EventType
	
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

// PublishEvents publishes multiple events
func (b *NatsEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	// Save the events to the store
	err := b.eventStore.SaveEvents(ctx, events)
	if err != nil {
		return err
	}
	
	// Publish each event
	for _, event := range events {
		// Marshal the event to JSON
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}
		
		// Create the subject
		subject := b.topicPrefix + event.EventType
		
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
	}
	
	return nil
}

// Subscribe subscribes to all events
func (b *NatsEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Add the handler to the list
	b.handlers = append(b.handlers, handler)
	
	// Subscribe to all events
	subject := b.topicPrefix + ">"
	
	// Create a message handler
	msgHandler := func(msg *nats.Msg) {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			b.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}
		
		// Handle the event
		err = handler.HandleEvent(&event)
		if err != nil {
			b.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(err))
		}
	}
	
	// Subscribe to the subject
	var sub *nats.Subscription
	var err error
	
	if b.js != nil {
		// Subscribe with JetStream
		sub, err = b.js.Subscribe(subject, msgHandler)
	} else {
		// Subscribe with standard NATS
		sub, err = b.conn.Subscribe(subject, msgHandler)
	}
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to events: %w", err)
	}
	
	// Add the subscription to the list
	b.subs = append(b.subs, sub)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *NatsEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Add the handler to the map
	if _, ok := b.typeHandlers[eventType]; !ok {
		b.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	
	// Subscribe to the specific event type
	subject := b.topicPrefix + eventType
	
	// Create a message handler
	msgHandler := func(msg *nats.Msg) {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			b.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}
		
		// Handle the event
		err = handler.HandleEvent(&event)
		if err != nil {
			b.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(err))
		}
	}
	
	// Subscribe to the subject
	var sub *nats.Subscription
	var err error
	
	if b.js != nil {
		// Subscribe with JetStream
		sub, err = b.js.Subscribe(subject, msgHandler)
	} else {
		// Subscribe with standard NATS
		sub, err = b.conn.Subscribe(subject, msgHandler)
	}
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to event type: %w", err)
	}
	
	// Add the subscription to the list
	b.subs = append(b.subs, sub)
	
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *NatsEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Add the handler to the map
	if _, ok := b.aggHandlers[aggregateType]; !ok {
		b.aggHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	b.aggHandlers[aggregateType] = append(b.aggHandlers[aggregateType], handler)
	
	// Subscribe to all events
	subject := b.topicPrefix + ">"
	
	// Create a message handler
	msgHandler := func(msg *nats.Msg) {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			b.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}
		
		// Check if the event is for the specified aggregate type
		if event.AggregateType != aggregateType {
			return
		}
		
		// Handle the event
		err = handler.HandleEvent(&event)
		if err != nil {
			b.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(err))
		}
	}
	
	// Subscribe to the subject
	var sub *nats.Subscription
	var err error
	
	if b.js != nil {
		// Subscribe with JetStream
		sub, err = b.js.Subscribe(subject, msgHandler)
	} else {
		// Subscribe with standard NATS
		sub, err = b.conn.Subscribe(subject, msgHandler)
	}
	
	if err != nil {
		return fmt.Errorf("failed to subscribe to aggregate type: %w", err)
	}
	
	// Add the subscription to the list
	b.subs = append(b.subs, sub)
	
	return nil
}
