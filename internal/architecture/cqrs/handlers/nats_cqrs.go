package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/nats-io/nats.go"
	"go.uber.org/zap"
)

// NatsCQRSAdapter provides an adapter for NATS-based CQRS
type NatsCQRSAdapter struct {
	logger *zap.Logger

	// NATS components
	conn *nats.Conn
	js   nats.JetStreamContext

	// Our components
	eventStore    store.EventStore
	aggregateRepo aggregate.Repository

	// Command handlers
	commandHandlers map[string]command.EventSourcedHandler

	// Event handlers
	eventHandlers map[string][]eventsourcing.EventHandler

	// Subscriptions
	subs []*nats.Subscription

	// Context for managing subscriptions
	ctx    context.Context
	cancel context.CancelFunc

	// Synchronization
	mu sync.RWMutex
}

// NatsCQRSConfig contains configuration for the NatsCQRSAdapter
type NatsCQRSConfig struct {
	// URLs is a list of NATS server URLs
	URLs []string

	// ConnectionTimeout is the timeout for connecting to NATS
	ConnectionTimeout time.Duration

	// MaxReconnects is the maximum number of reconnect attempts
	MaxReconnects int

	// ReconnectWait is the time to wait between reconnect attempts
	ReconnectWait time.Duration

	// UseJetStream determines if JetStream should be used
	UseJetStream bool

	// CommandStreamConfig is the configuration for the command stream
	CommandStreamConfig *nats.StreamConfig

	// EventStreamConfig is the configuration for the event stream
	EventStreamConfig *nats.StreamConfig
}

// DefaultNatsCQRSConfig returns the default configuration
func DefaultNatsCQRSConfig() NatsCQRSConfig {
	return NatsCQRSConfig{
		URLs:              []string{nats.DefaultURL},
		ConnectionTimeout: 5 * time.Second,
		MaxReconnects:     10,
		ReconnectWait:     1 * time.Second,
		UseJetStream:      true,
		CommandStreamConfig: &nats.StreamConfig{
			Name:      "commands",
			Subjects:  []string{"commands.>"},
			Retention: nats.WorkQueuePolicy,
			MaxAge:    24 * time.Hour,
			MaxBytes:  1024 * 1024 * 1024, // 1GB
			Storage:   nats.FileStorage,
			Replicas:  1,
		},
		EventStreamConfig: &nats.StreamConfig{
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

// NewNatsCQRSAdapter creates a new NatsCQRSAdapter
func NewNatsCQRSAdapter(
	eventStore store.EventStore,
	aggregateRepo aggregate.Repository,
	logger *zap.Logger,
	config NatsCQRSConfig,
) (*NatsCQRSAdapter, error) {
	// Create a context for managing subscriptions
	ctx, cancel := context.WithCancel(context.Background())

	// Create NATS connection options
	opts := []nats.Option{
		nats.Name("tradSys-cqrs"),
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

	// Create the adapter
	adapter := &NatsCQRSAdapter{
		logger:          logger,
		conn:            nc,
		eventStore:      eventStore,
		aggregateRepo:   aggregateRepo,
		commandHandlers: make(map[string]command.EventSourcedHandler),
		eventHandlers:   make(map[string][]eventsourcing.EventHandler),
		subs:            make([]*nats.Subscription, 0),
		ctx:             ctx,
		cancel:          cancel,
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

		// Create the command stream if it doesn't exist
		_, err = js.StreamInfo(config.CommandStreamConfig.Name)
		if err != nil {
			// Stream doesn't exist, create it
			_, err = js.AddStream(config.CommandStreamConfig)
			if err != nil {
				nc.Close()
				cancel()
				return nil, fmt.Errorf("failed to create command stream: %w", err)
			}
		}

		// Create the event stream if it doesn't exist
		_, err = js.StreamInfo(config.EventStreamConfig.Name)
		if err != nil {
			// Stream doesn't exist, create it
			_, err = js.AddStream(config.EventStreamConfig)
			if err != nil {
				nc.Close()
				cancel()
				return nil, fmt.Errorf("failed to create event stream: %w", err)
			}
		}

		adapter.js = js
	}

	return adapter, nil
}

// Start starts the adapter
func (a *NatsCQRSAdapter) Start() error {
	// Nothing to do here as NATS subscriptions are started when registered
	return nil
}

// Stop stops the adapter
func (a *NatsCQRSAdapter) Stop() error {
	// Cancel the context to stop all subscriptions
	a.cancel()

	// Drain and close all subscriptions
	for _, sub := range a.subs {
		err := sub.Drain()
		if err != nil {
			a.logger.Error("Failed to drain subscription", zap.Error(err))
		}
	}

	// Close the connection
	a.conn.Close()

	return nil
}

// RegisterCommandHandler registers a command handler
func (a *NatsCQRSAdapter) RegisterCommandHandler(
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

	a.mu.Lock()
	defer a.mu.Unlock()

	// Check if the handler is already registered
	if _, exists := a.commandHandlers[commandName]; exists {
		return fmt.Errorf("handler already registered for command %s", commandName)
	}

	// Register the handler
	a.commandHandlers[commandName] = handler

	// Subscribe to the command
	subject := "commands." + commandName

	// Create a message handler
	msgHandler := func(msg *nats.Msg) {
		// Unmarshal the command
		var cmd command.Command
		err := json.Unmarshal(msg.Data, &cmd)
		if err != nil {
			a.logger.Error("Failed to unmarshal command", zap.Error(err))
			return
		}

		// Handle the command
		events, err := handler.Handle(context.Background(), cmd)
		if err != nil {
			a.logger.Error("Failed to handle command",
				zap.String("command", cmd.CommandName()),
				zap.String("aggregate_id", cmd.AggregateID()),
				zap.Error(err))
			return
		}

		// Save and publish events
		if len(events) > 0 {
			// Save events to the event store
			err = a.eventStore.SaveEvents(context.Background(), events)
			if err != nil {
				a.logger.Error("Failed to save events",
					zap.String("command", cmd.CommandName()),
					zap.String("aggregate_id", cmd.AggregateID()),
					zap.Error(err))
				return
			}

			// Publish events
			for _, event := range events {
				// Marshal the event
				payload, err := json.Marshal(event)
				if err != nil {
					a.logger.Error("Failed to marshal event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.Error(err))
					continue
				}

				// Create the subject
				eventSubject := "events." + event.EventType

				// Publish the event
				if a.js != nil {
					// Publish with JetStream
					_, err = a.js.Publish(eventSubject, payload)
				} else {
					// Publish with standard NATS
					err = a.conn.Publish(eventSubject, payload)
				}

				if err != nil {
					a.logger.Error("Failed to publish event",
						zap.String("event_type", event.EventType),
						zap.String("aggregate_id", event.AggregateID),
						zap.Error(err))
				}
			}
		}

		// Acknowledge the message if using JetStream
		if a.js != nil && msg.Reply != "" {
			err = msg.Ack()
			if err != nil {
				a.logger.Error("Failed to acknowledge message", zap.Error(err))
			}
		}
	}

	// Subscribe to the subject
	var sub *nats.Subscription
	var err error

	if a.js != nil {
		// Subscribe with JetStream
		sub, err = a.js.QueueSubscribe(subject, "tradSys-command-handlers", msgHandler)
	} else {
		// Subscribe with standard NATS
		sub, err = a.conn.QueueSubscribe(subject, "tradSys-command-handlers", msgHandler)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to command: %w", err)
	}

	// Add the subscription to the list
	a.subs = append(a.subs, sub)

	return nil
}

// RegisterEventHandler registers an event handler
func (a *NatsCQRSAdapter) RegisterEventHandler(
	eventType string,
	handler eventsourcing.EventHandler,
) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	// Add the handler to the map
	if _, ok := a.eventHandlers[eventType]; !ok {
		a.eventHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	a.eventHandlers[eventType] = append(a.eventHandlers[eventType], handler)

	// Subscribe to the event
	subject := "events." + eventType

	// Create a message handler
	msgHandler := func(msg *nats.Msg) {
		// Unmarshal the event
		var event eventsourcing.Event
		err := json.Unmarshal(msg.Data, &event)
		if err != nil {
			a.logger.Error("Failed to unmarshal event", zap.Error(err))
			return
		}

		// Handle the event
		err = handler.HandleEvent(&event)
		if err != nil {
			a.logger.Error("Failed to handle event",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.Error(err))
		}

		// Acknowledge the message if using JetStream
		if a.js != nil && msg.Reply != "" {
			err = msg.Ack()
			if err != nil {
				a.logger.Error("Failed to acknowledge message", zap.Error(err))
			}
		}
	}

	// Subscribe to the subject
	var sub *nats.Subscription
	var err error

	if a.js != nil {
		// Subscribe with JetStream
		sub, err = a.js.QueueSubscribe(subject, "tradSys-event-handlers", msgHandler)
	} else {
		// Subscribe with standard NATS
		sub, err = a.conn.QueueSubscribe(subject, "tradSys-event-handlers", msgHandler)
	}

	if err != nil {
		return fmt.Errorf("failed to subscribe to event: %w", err)
	}

	// Add the subscription to the list
	a.subs = append(a.subs, sub)

	return nil
}

// DispatchCommand dispatches a command
func (a *NatsCQRSAdapter) DispatchCommand(ctx context.Context, cmd command.Command) error {
	// Marshal the command
	payload, err := json.Marshal(cmd)
	if err != nil {
		return err
	}

	// Create the subject
	subject := "commands." + cmd.CommandName()

	// Publish the command
	if a.js != nil {
		// Publish with JetStream
		_, err = a.js.Publish(subject, payload)
	} else {
		// Publish with standard NATS
		err = a.conn.Publish(subject, payload)
	}

	if err != nil {
		return fmt.Errorf("failed to publish command: %w", err)
	}

	return nil
}

// CreateEventBusAdapter creates an EventBus adapter that uses NATS
func (a *NatsCQRSAdapter) CreateEventBusAdapter() eventbus.EventBus {
	// Create a NATS event bus configuration
	config := eventbus.DefaultNatsEventBusConfig()

	// Create a NATS event bus
	bus, err := eventbus.NewNatsEventBus(a.eventStore, a.logger, config)
	if err != nil {
		a.logger.Error("Failed to create NATS event bus", zap.Error(err))
		return nil
	}

	return bus
}
