package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	cqrscore "github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/core"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	escore "github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/handlers"
	"go.uber.org/zap"
)

// WatermillCQRSAdapter provides an adapter for Watermill's CQRS component
type WatermillCQRSAdapter struct {
	logger          *zap.Logger
	watermillLogger watermill.LoggerAdapter

	// Watermill components
	router *message.Router

	// Our components
	eventStore    escore.EventStore
	aggregateRepo handlers.Repository

	// Publishers and subscribers
	commandPublisher  message.Publisher
	commandSubscriber message.Subscriber
	eventPublisher    message.Publisher
	eventSubscriber   message.Subscriber
}

// WatermillCQRSConfig contains configuration for the WatermillCQRSAdapter
type WatermillCQRSConfig struct {
	CommandsChannelBuffer int
	EventsChannelBuffer   int
	Persistent            bool
}

// DefaultWatermillCQRSConfig returns the default configuration
func DefaultWatermillCQRSConfig() WatermillCQRSConfig {
	return WatermillCQRSConfig{
		CommandsChannelBuffer: 1000,
		EventsChannelBuffer:   1000,
		Persistent:            true,
	}
}

// NewWatermillCQRSAdapter creates a new WatermillCQRSAdapter
func NewWatermillCQRSAdapter(
	eventStore store.EventStore,
	aggregateRepo aggregate.Repository,
	logger *zap.Logger,
	config WatermillCQRSConfig,
) (*WatermillCQRSAdapter, error) {
	// Create a watermill logger
	watermillLogger := watermill.NewStdLoggerWithOut(logger.Sugar().Out(), false, false)

	// Create a router
	router, err := message.NewRouter(message.RouterConfig{}, watermillLogger)
	if err != nil {
		return nil, err
	}

	// Add recovery middleware
	router.AddMiddleware(func(h message.HandlerFunc) message.HandlerFunc {
		return func(msg *message.Message) ([]*message.Message, error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Recovered from panic in message handler",
						zap.Any("panic", r),
						zap.String("message_uuid", msg.UUID),
					)
				}
			}()
			return h(msg)
		}
	})

	// Create command publisher/subscriber
	commandPubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: config.CommandsChannelBuffer,
			Persistent:          config.Persistent,
		},
		watermillLogger,
	)

	// Create event publisher/subscriber
	eventPubSub := gochannel.NewGoChannel(
		gochannel.Config{
			OutputChannelBuffer: config.EventsChannelBuffer,
			Persistent:          config.Persistent,
		},
		watermillLogger,
	)

	return &WatermillCQRSAdapter{
		logger:            logger,
		watermillLogger:   watermillLogger,
		router:            router,
		eventStore:        eventStore,
		aggregateRepo:     aggregateRepo,
		commandPublisher:  commandPubSub,
		commandSubscriber: commandPubSub,
		eventPublisher:    eventPubSub,
		eventSubscriber:   eventPubSub,
	}, nil
}

// Start starts the adapter
func (a *WatermillCQRSAdapter) Start() error {
	// Start the router in a separate goroutine
	go func() {
		err := a.router.Run(context.Background())
		if err != nil {
			a.logger.Error("Failed to start router", zap.Error(err))
		}
	}()

	return nil
}

// Stop stops the adapter
func (a *WatermillCQRSAdapter) Stop() error {
	// Close the router
	return a.router.Close()
}

// RegisterCommandHandler registers a command handler
func (a *WatermillCQRSAdapter) RegisterCommandHandler(
	commandType reflect.Type,
	handler command.EventSourcedHandler,
) error {
	// Create a zero value of the command type
	cmd, ok := reflect.New(commandType).Elem().Interface().(command.Command)
	if !ok {
		return fmt.Errorf("command type %s does not implement Command interface", commandType.Name())
	}

	// Get the command name
	commandName := cmd.CommandName()

	// Create a handler function
	handlerFunc := func(msg *message.Message) ([]*message.Message, error) {
		// Unmarshal the command
		var cmd command.Command
		err := json.Unmarshal(msg.Payload, &cmd)
		if err != nil {
			return nil, err
		}

		// Handle the command
		events, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			return nil, err
		}

		// Save and publish events
		if len(events) > 0 {
			// Save events to the event store
			err = a.eventStore.SaveEvents(context.Background(), events)
			if err != nil {
				return nil, err
			}

			// Publish events
			var messages []*message.Message
			for _, event := range events {
				// Convert to message
				payload, err := json.Marshal(event)
				if err != nil {
					return nil, err
				}

				// Create a message
				msg := message.NewMessage(watermill.NewUUID(), payload)

				// Add metadata
				msg.Metadata.Set("aggregate_id", event.AggregateID)
				msg.Metadata.Set("aggregate_type", event.AggregateType)
				msg.Metadata.Set("event_type", event.EventType)

				messages = append(messages, msg)
			}

			return messages, nil
		}

		return nil, nil
	}

	// Register the handler
	a.router.AddHandler(
		"command_handler_"+commandName,
		"commands."+commandName,
		a.commandSubscriber,
		"events.*",
		a.eventPublisher,
		handlerFunc,
	)

	return nil
}

// RegisterEventHandler registers an event handler
func (a *WatermillCQRSAdapter) RegisterEventHandler(
	eventType string,
	handler eventsourcing.EventHandler,
) error {
	// Create a handler function
	handlerFunc := func(msg *message.Message) error {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Payload, &event)
		if err != nil {
			return err
		}

		// Handle the event
		return handler.HandleEvent(&event)
	}

	// Register the handler
	a.router.AddNoPublisherHandler(
		"event_handler_"+eventType,
		"events."+eventType,
		a.eventSubscriber,
		func(msg *message.Message) error {
			return handlerFunc(msg)
		},
	)

	return nil
}

// DispatchCommand dispatches a command
func (a *WatermillCQRSAdapter) DispatchCommand(ctx context.Context, cmd command.Command) error {
	// Marshal the command
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	// Create a message
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Publish the message
	return a.commandPublisher.Publish("commands."+cmd.CommandName(), msg)
}

// CreateEventBusAdapter creates an EventBus adapter that uses Watermill
func (a *WatermillCQRSAdapter) CreateEventBusAdapter() eventbus.EventBus {
	return &watermillEventBusAdapter{
		adapter:      a,
		logger:       a.logger,
		handlers:     make([]eventsourcing.EventHandler, 0),
		typeHandlers: make(map[string][]eventsourcing.EventHandler),
		aggHandlers:  make(map[string][]eventsourcing.EventHandler),
	}
}

// watermillEventBusAdapter adapts Watermill's event bus to our EventBus interface
type watermillEventBusAdapter struct {
	adapter *WatermillCQRSAdapter
	logger  *zap.Logger

	// Handlers
	handlers     []eventsourcing.EventHandler
	typeHandlers map[string][]eventsourcing.EventHandler
	aggHandlers  map[string][]eventsourcing.EventHandler
	mu           sync.RWMutex
}

// PublishEvent publishes an event
func (a *watermillEventBusAdapter) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := a.adapter.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}

	// Marshal the event
	payload, err := json.Marshal(event)
	if err != nil {
		return err
	}

	// Create a message
	msg := message.NewMessage(watermill.NewUUID(), payload)

	// Add metadata
	msg.Metadata.Set("aggregate_id", event.AggregateID)
	msg.Metadata.Set("aggregate_type", event.AggregateType)
	msg.Metadata.Set("event_type", event.EventType)

	// Publish the message
	return a.adapter.eventPublisher.Publish("events."+event.EventType, msg)
}

// PublishEvents publishes multiple events
func (a *watermillEventBusAdapter) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}

	// Save the events to the store
	err := a.adapter.eventStore.SaveEvents(ctx, events)
	if err != nil {
		return err
	}

	// Publish each event
	for _, event := range events {
		// Marshal the event
		payload, err := json.Marshal(event)
		if err != nil {
			return err
		}

		// Create a message
		msg := message.NewMessage(watermill.NewUUID(), payload)

		// Add metadata
		msg.Metadata.Set("aggregate_id", event.AggregateID)
		msg.Metadata.Set("aggregate_type", event.AggregateType)
		msg.Metadata.Set("event_type", event.EventType)

		// Publish the message
		err = a.adapter.eventPublisher.Publish("events."+event.EventType, msg)
		if err != nil {
			return err
		}
	}

	return nil
}

// Subscribe subscribes to all events
func (a *watermillEventBusAdapter) Subscribe(handler eventsourcing.EventHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.handlers = append(a.handlers, handler)

	// Register a handler for all events
	handlerFunc := func(msg *message.Message) error {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Payload, &event)
		if err != nil {
			return err
		}

		// Handle the event
		return handler.HandleEvent(&event)
	}

	// Register the handler
	a.adapter.router.AddNoPublisherHandler(
		"event_handler_all_"+watermill.NewUUID(),
		"events.*",
		a.adapter.eventSubscriber,
		func(msg *message.Message) error {
			return handlerFunc(msg)
		},
	)

	return nil
}

// SubscribeToType subscribes to events of a specific type
func (a *watermillEventBusAdapter) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.typeHandlers[eventType]; !ok {
		a.typeHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}

	a.typeHandlers[eventType] = append(a.typeHandlers[eventType], handler)

	// Register with watermill
	return a.adapter.RegisterEventHandler(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (a *watermillEventBusAdapter) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, ok := a.aggHandlers[aggregateType]; !ok {
		a.aggHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}

	a.aggHandlers[aggregateType] = append(a.aggHandlers[aggregateType], handler)

	// Register a handler for all events of this aggregate type
	handlerFunc := func(msg *message.Message) error {
		// Check if the message is for this aggregate type
		if msg.Metadata.Get("aggregate_type") != aggregateType {
			return nil
		}

		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Payload, &event)
		if err != nil {
			return err
		}

		// Handle the event
		return handler.HandleEvent(&event)
	}

	// Register the handler
	a.adapter.router.AddNoPublisherHandler(
		"event_handler_aggregate_"+aggregateType+"_"+watermill.NewUUID(),
		"events.*",
		a.adapter.eventSubscriber,
		func(msg *message.Message) error {
			return handlerFunc(msg)
		},
	)

	return nil
}
