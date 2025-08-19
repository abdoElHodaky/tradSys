package eventbus

import (
	"context"
	"encoding/json"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/google/uuid"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// WatermillEventBus provides an implementation of the EventBus interface using Watermill
type WatermillEventBus struct {
	publisher     message.Publisher
	subscriber    message.Subscriber
	router        *message.Router
	eventStore    store.EventStore
	handlers      []eventsourcing.EventHandler
	typeHandlers  map[string][]eventsourcing.EventHandler
	aggHandlers   map[string][]eventsourcing.EventHandler
	logger        *zap.Logger
	mu            sync.RWMutex
	topicPrefix   string
}

// WatermillEventBusConfig contains configuration for the WatermillEventBus
type WatermillEventBusConfig struct {
	// TopicPrefix is the prefix for all topics
	TopicPrefix string
	
	// BufferSize is the size of the buffer for the gochannel publisher/subscriber
	BufferSize int
	
	// Persistent determines if messages should be persisted in the gochannel
	Persistent bool
}

// DefaultWatermillEventBusConfig returns the default configuration
func DefaultWatermillEventBusConfig() WatermillEventBusConfig {
	return WatermillEventBusConfig{
		TopicPrefix: "events.",
		BufferSize:  1000,
		Persistent:  true,
	}
}

// NewWatermillEventBus creates a new WatermillEventBus
func NewWatermillEventBus(eventStore store.EventStore, logger *zap.Logger, config WatermillEventBusConfig) (*WatermillEventBus, error) {
	// Create a watermill logger
	watermillLogger := watermill.NewStdLoggerWithOut(logger.Sugar().Out(), false, false)
	
	// Create a gochannel publisher/subscriber
	pubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: config.BufferSize,
			Persistent:          config.Persistent,
		},
		watermillLogger,
	)
	
	// Create a router
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}
	
	return &WatermillEventBus{
		publisher:     pubSub,
		subscriber:    pubSub,
		router:        router,
		eventStore:    eventStore,
		handlers:      make([]eventsourcing.EventHandler, 0),
		typeHandlers:  make(map[string][]eventsourcing.EventHandler),
		aggHandlers:   make(map[string][]eventsourcing.EventHandler),
		logger:        logger,
		topicPrefix:   config.TopicPrefix,
	}, nil
}

// Start starts the event bus
func (b *WatermillEventBus) Start() error {
	// Start the router in a separate goroutine
	go func() {
		err := b.router.Run(context.Background())
		if err != nil {
			b.logger.Error("Failed to start router", zap.Error(err))
		}
	}()
	
	// Add handlers for all event types
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Subscribe to all events
	_, err := b.subscriber.Subscribe(context.Background(), b.topicPrefix+"*", b.handleMessage)
	if err != nil {
		return err
	}
	
	return nil
}

// Stop stops the event bus
func (b *WatermillEventBus) Stop() error {
	// Close the router
	return b.router.Close()
}

// PublishEvent publishes an event
func (b *WatermillEventBus) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := b.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Convert the event to a message
	msg, err := b.eventToMessage(event)
	if err != nil {
		return err
	}
	
	// Publish the message
	topic := b.topicPrefix + event.AggregateType
	return b.publisher.Publish(topic, msg)
}

// PublishEvents publishes multiple events
func (b *WatermillEventBus) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
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
		// Convert the event to a message
		msg, err := b.eventToMessage(event)
		if err != nil {
			return err
		}
		
		// Publish the message
		topic := b.topicPrefix + event.AggregateType
		err = b.publisher.Publish(topic, msg)
		if err != nil {
			return err
		}
	}
	
	return nil
}

// Subscribe subscribes to all events
func (b *WatermillEventBus) Subscribe(handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.handlers = append(b.handlers, handler)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (b *WatermillEventBus) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.typeHandlers[eventType]; !ok {
		b.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.typeHandlers[eventType] = append(b.typeHandlers[eventType], handler)
	
	return nil
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (b *WatermillEventBus) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	if _, ok := b.aggHandlers[aggregateType]; !ok {
		b.aggHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	
	b.aggHandlers[aggregateType] = append(b.aggHandlers[aggregateType], handler)
	
	return nil
}

// handleMessage handles a message from the subscriber
func (b *WatermillEventBus) handleMessage(msg *message.Message) ([]*message.Message, error) {
	// Convert the message to an event
	event, err := b.messageToEvent(msg)
	if err != nil {
		b.logger.Error("Failed to convert message to event", zap.Error(err))
		return nil, err
	}
	
	// Handle the event
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Notify all handlers
	for _, handler := range b.handlers {
		err := handler.HandleEvent(event)
		if err != nil {
			b.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.String("aggregate_type", event.AggregateType),
				zap.Error(err))
		}
	}
	
	// Notify type handlers
	if handlers, ok := b.typeHandlers[event.EventType]; ok {
		for _, handler := range handlers {
			err := handler.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}
	}
	
	// Notify aggregate handlers
	if handlers, ok := b.aggHandlers[event.AggregateType]; ok {
		for _, handler := range handlers {
			err := handler.HandleEvent(event)
			if err != nil {
				b.logger.Error("Failed to handle event",
					zap.String("event_type", event.EventType),
					zap.String("aggregate_id", event.AggregateID),
					zap.String("aggregate_type", event.AggregateType),
					zap.Error(err))
			}
		}
	}
	
	// Acknowledge the message
	return nil, nil
}

// eventToMessage converts an event to a message
func (b *WatermillEventBus) eventToMessage(event *eventsourcing.Event) (*message.Message, error) {
	// Marshal the event to JSON
	payload, err := json.Marshal(event)
	if err != nil {
		return nil, err
	}
	
	// Create a new message
	msg := message.NewMessage(uuid.New().String(), payload)
	
	// Add metadata
	msg.Metadata.Set("aggregate_id", event.AggregateID)
	msg.Metadata.Set("aggregate_type", event.AggregateType)
	msg.Metadata.Set("event_type", event.EventType)
	msg.Metadata.Set("version", string(event.Version))
	
	return msg, nil
}

// messageToEvent converts a message to an event
func (b *WatermillEventBus) messageToEvent(msg *message.Message) (*eventsourcing.Event, error) {
	// Unmarshal the message payload to an event
	var event eventsourcing.Event
	err := json.Unmarshal(msg.Payload, &event)
	if err != nil {
		return nil, err
	}
	
	return &event, nil
}

