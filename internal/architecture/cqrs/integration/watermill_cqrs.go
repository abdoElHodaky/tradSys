package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/ThreeDotsLabs/watermill"
	"github.com/ThreeDotsLabs/watermill/components/cqrs"
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/ThreeDotsLabs/watermill/message/router/middleware"
	"github.com/ThreeDotsLabs/watermill/pubsub/gochannel"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/command"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/query"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/aggregate"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/store"
	"go.uber.org/zap"
)

// WatermillCQRSAdapter provides an adapter for Watermill's CQRS component
type WatermillCQRSAdapter struct {
	logger          *zap.Logger
	watermillLogger watermill.LoggerAdapter
	
	// Watermill components
	router          *message.Router
	commandBus      *cqrs.CommandBus
	eventBus        *cqrs.EventBus
	
	// Our components
	eventStore      store.EventStore
	aggregateRepo   aggregate.Repository
	
	// Publishers and subscribers
	commandPublisher message.Publisher
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
	router.AddMiddleware(middleware.Recoverer)
	
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
	
	// Create CQRS facade
	cqrsFacade, err := cqrs.NewFacade(cqrs.FacadeConfig{
		GenerateCommandsTopic: func(commandName string) string {
			return "commands." + commandName
		},
		GenerateEventsTopic: func(eventName string) string {
			return "events." + eventName
		},
		CommandHandlers: func(cb *cqrs.CommandBus, eb *cqrs.EventBus) []cqrs.CommandHandler {
			// Command handlers will be registered later
			return []cqrs.CommandHandler{}
		},
		EventHandlers: func(cb *cqrs.CommandBus, eb *cqrs.EventBus) []cqrs.EventHandler {
			// Event handlers will be registered later
			return []cqrs.EventHandler{}
		},
		Router:                router,
		CommandsPublisher:     commandPubSub,
		CommandsSubscriberConstructor: func(handlerName string) (message.Subscriber, error) {
			return commandPubSub, nil
		},
		EventsPublisher:     eventPubSub,
		EventsSubscriberConstructor: func(handlerName string) (message.Subscriber, error) {
			return eventPubSub, nil
		},
		Logger:              watermillLogger,
		CommandEventMarshaler: cqrs.JSONMarshaler{},
	})
	if err != nil {
		return nil, err
	}
	
	return &WatermillCQRSAdapter{
		logger:            logger,
		watermillLogger:   watermillLogger,
		router:            router,
		commandBus:        cqrsFacade.CommandBus(),
		eventBus:          cqrsFacade.EventBus(),
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
	
	// Create a watermill command handler
	watermillHandler := func(ctx context.Context, cmd interface{}) error {
		// Convert to our command type
		ourCmd, ok := cmd.(command.Command)
		if !ok {
			return fmt.Errorf("command %T does not implement Command interface", cmd)
		}
		
		// Handle the command
		events, err := handler.Handle(ctx, ourCmd)
		if err != nil {
			return err
		}
		
		// Save and publish events
		if len(events) > 0 {
			// Save events to the event store
			err = a.eventStore.SaveEvents(ctx, events)
			if err != nil {
				return err
			}
			
			// Publish events
			for _, event := range events {
				// Convert to watermill event
				watermillEvent := struct {
					Event *eventsourcing.Event `json:"event"`
				}{
					Event: event,
				}
				
				// Publish the event
				err = a.eventBus.Publish(ctx, watermillEvent)
				if err != nil {
					return err
				}
			}
		}
		
		return nil
	}
	
	// Register the handler with watermill
	return a.commandBus.AddHandler(
		commandName,
		reflect.New(commandType).Interface(),
		watermillHandler,
	)
}

// RegisterEventHandler registers an event handler
func (a *WatermillCQRSAdapter) RegisterEventHandler(
	eventType string,
	handler eventsourcing.EventHandler,
) error {
	// Create a watermill event handler
	watermillHandler := func(ctx context.Context, event interface{}) error {
		// Convert to our event type
		wrapper, ok := event.(struct {
			Event *eventsourcing.Event `json:"event"`
		})
		if !ok {
			return fmt.Errorf("event %T is not a valid event wrapper", event)
		}
		
		// Handle the event
		return handler.HandleEvent(wrapper.Event)
	}
	
	// Create a dummy event for registration
	dummyEvent := struct {
		Event *eventsourcing.Event `json:"event"`
	}{
		Event: &eventsourcing.Event{
			EventType: eventType,
		},
	}
	
	// Register the handler with watermill
	return a.eventBus.AddHandler(
		"handler_"+eventType,
		dummyEvent,
		watermillHandler,
	)
}

// DispatchCommand dispatches a command
func (a *WatermillCQRSAdapter) DispatchCommand(ctx context.Context, cmd command.Command) error {
	return a.commandBus.Send(ctx, cmd)
}

// CreateEventBusAdapter creates an EventBus adapter that uses Watermill
func (a *WatermillCQRSAdapter) CreateEventBusAdapter() eventbus.EventBus {
	return &watermillEventBusAdapter{
		adapter: a,
		logger:  a.logger,
	}
}

// watermillEventBusAdapter adapts Watermill's event bus to our EventBus interface
type watermillEventBusAdapter struct {
	adapter *WatermillCQRSAdapter
	logger  *zap.Logger
	
	// Handlers
	handlers      []eventsourcing.EventHandler
	typeHandlers  map[string][]eventsourcing.EventHandler
	aggHandlers   map[string][]eventsourcing.EventHandler
	mu            sync.RWMutex
}

// PublishEvent publishes an event
func (a *watermillEventBusAdapter) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Save the event to the store
	err := a.adapter.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return err
	}
	
	// Convert to watermill event
	watermillEvent := struct {
		Event *eventsourcing.Event `json:"event"`
	}{
		Event: event,
	}
	
	// Publish the event
	return a.adapter.eventBus.Publish(ctx, watermillEvent)
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
		// Convert to watermill event
		watermillEvent := struct {
			Event *eventsourcing.Event `json:"event"`
		}{
			Event: event,
		}
		
		// Publish the event
		err = a.adapter.eventBus.Publish(ctx, watermillEvent)
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
	
	if a.handlers == nil {
		a.handlers = make([]eventsourcing.EventHandler, 0)
	}
	
	a.handlers = append(a.handlers, handler)
	
	return nil
}

// SubscribeToType subscribes to events of a specific type
func (a *watermillEventBusAdapter) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.typeHandlers == nil {
		a.typeHandlers = make(map[string][]eventsourcing.EventHandler)
	}
	
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
	
	if a.aggHandlers == nil {
		a.aggHandlers = make(map[string][]eventsourcing.EventHandler)
	}
	
	if _, ok := a.aggHandlers[aggregateType]; !ok {
		a.aggHandlers[aggregateType] = make([]eventsourcing.EventHandler, 0)
	}
	
	a.aggHandlers[aggregateType] = append(a.aggHandlers[aggregateType], handler)
	
	return nil
}

