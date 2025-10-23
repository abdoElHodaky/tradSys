package core

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/handlers"
	"go.uber.org/zap"
)

// EventSourcedHandler represents a command handler that uses event sourcing
type EventSourcedHandler interface {
	// Handle handles a command and returns events
	Handle(ctx context.Context, command Command) ([]*eventsourcing.Event, error)
}

// EventSourcedHandlerFunc is a function that implements the EventSourcedHandler interface
type EventSourcedHandlerFunc func(ctx context.Context, command Command) ([]*eventsourcing.Event, error)

// Handle handles a command and returns events
func (f EventSourcedHandlerFunc) Handle(ctx context.Context, command Command) ([]*eventsourcing.Event, error) {
	return f(ctx, command)
}

// EventSourcedCommandBus represents a command bus that uses event sourcing
type EventSourcedCommandBus struct {
	handlers      map[string]EventSourcedHandler
	eventBus      eventbus.EventBus
	aggregateRepo aggregate.Repository
	logger        *zap.Logger
	mu            sync.RWMutex
}

// NewEventSourcedCommandBus creates a new event-sourced command bus
func NewEventSourcedCommandBus(eventBus eventbus.EventBus, aggregateRepo aggregate.Repository, logger *zap.Logger) *EventSourcedCommandBus {
	return &EventSourcedCommandBus{
		handlers:      make(map[string]EventSourcedHandler),
		eventBus:      eventBus,
		aggregateRepo: aggregateRepo,
		logger:        logger,
	}
}

// Register registers a handler for a command
func (b *EventSourcedCommandBus) Register(commandType reflect.Type, handler EventSourcedHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create a zero value of the command type
	command, ok := reflect.New(commandType).Elem().Interface().(Command)
	if !ok {
		return fmt.Errorf("command type %s does not implement Command interface", commandType.Name())
	}

	// Get the command name
	commandName := command.CommandName()

	// Check if a handler is already registered for the command
	if _, exists := b.handlers[commandName]; exists {
		return fmt.Errorf("handler already registered for command %s", commandName)
	}

	// Register the handler
	b.handlers[commandName] = handler

	if b.logger != nil {
		b.logger.Info("Registered event-sourced command handler",
			zap.String("command", commandName),
			zap.String("handler", reflect.TypeOf(handler).String()))
	}

	return nil
}

// RegisterFunc registers a handler function for a command
func (b *EventSourcedCommandBus) RegisterFunc(commandType reflect.Type, handler func(ctx context.Context, command Command) ([]*eventsourcing.Event, error)) error {
	return b.Register(commandType, EventSourcedHandlerFunc(handler))
}

// Dispatch dispatches a command to its handler
func (b *EventSourcedCommandBus) Dispatch(ctx context.Context, command Command) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Get the command name
	commandName := command.CommandName()

	// Get the handler for the command
	handler, exists := b.handlers[commandName]
	if !exists {
		return fmt.Errorf("no handler registered for command %s", commandName)
	}

	// Handle the command and get events
	events, err := handler.Handle(ctx, command)
	if err != nil {
		return err
	}

	// Publish the events
	if len(events) > 0 {
		err = b.eventBus.PublishEvents(ctx, events)
		if err != nil {
			return err
		}
	}

	return nil
}

// AggregateCommandHandler represents a command handler that operates on an aggregate
type AggregateCommandHandler struct {
	aggregateType string
	aggregateRepo aggregate.Repository
	logger        *zap.Logger
}

// NewAggregateCommandHandler creates a new aggregate command handler
func NewAggregateCommandHandler(aggregateType string, aggregateRepo aggregate.Repository, logger *zap.Logger) *AggregateCommandHandler {
	return &AggregateCommandHandler{
		aggregateType: aggregateType,
		aggregateRepo: aggregateRepo,
		logger:        logger,
	}
}

// HandleCreate handles a create command
func (h *AggregateCommandHandler) HandleCreate(ctx context.Context, command Command, createAggregate func(command Command) (aggregate.Aggregate, error)) ([]*eventsourcing.Event, error) {
	// Create the aggregate
	agg, err := createAggregate(command)
	if err != nil {
		return nil, err
	}

	// Save the aggregate
	err = h.aggregateRepo.Save(ctx, agg)
	if err != nil {
		return nil, err
	}

	// Return the events
	return agg.GetUncommittedEvents(), nil
}

// HandleUpdate handles an update command
func (h *AggregateCommandHandler) HandleUpdate(ctx context.Context, command Command, aggregateID string, updateAggregate func(agg aggregate.Aggregate, command Command) error) ([]*eventsourcing.Event, error) {
	// Create a new aggregate instance
	agg, err := h.createEmptyAggregate(aggregateID)
	if err != nil {
		return nil, err
	}

	// Load the aggregate
	err = h.aggregateRepo.Load(ctx, aggregateID, agg)
	if err != nil {
		return nil, err
	}

	// Update the aggregate
	err = updateAggregate(agg, command)
	if err != nil {
		return nil, err
	}

	// Save the aggregate
	err = h.aggregateRepo.Save(ctx, agg)
	if err != nil {
		return nil, err
	}

	// Return the events
	return agg.GetUncommittedEvents(), nil
}

// createEmptyAggregate creates an empty aggregate
func (h *AggregateCommandHandler) createEmptyAggregate(aggregateID string) (aggregate.Aggregate, error) {
	// Check if the aggregate repository supports creating aggregates
	if creator, ok := h.aggregateRepo.(interface {
		CreateAggregate(aggregateType string, aggregateID string) (aggregate.Aggregate, error)
	}); ok {
		return creator.CreateAggregate(h.aggregateType, aggregateID)
	}

	return nil, fmt.Errorf("aggregate repository does not support creating aggregates")
}
