package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/core"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// EventBusType represents the type of event bus
type EventBusType string

const (
	// InMemoryEventBusType represents an in-memory event bus
	InMemoryEventBusType EventBusType = "in-memory"
	
	// WatermillEventBusType represents a Watermill event bus
	WatermillEventBusType EventBusType = "watermill"
	
	// NatsEventBusType represents a NATS event bus
	NatsEventBusType EventBusType = "nats"
)

// EventRoutingStrategy determines how events are routed to different event buses
type EventRoutingStrategy int

const (
	// SingleBusStrategy routes all events to a single bus
	SingleBusStrategy EventRoutingStrategy = iota
	
	// TypeBasedStrategy routes events based on their type
	TypeBasedStrategy
	
	// AggregateBasedStrategy routes events based on their aggregate type
	AggregateBasedStrategy
	
	// PriorityBasedStrategy tries buses in order until one succeeds
	PriorityBasedStrategy
	
	// BroadcastStrategy sends events to all buses
	BroadcastStrategy
)

// EventBusRouter routes events to different event bus implementations
type EventBusRouter struct {
	logger   *zap.Logger
	strategy EventRoutingStrategy
	
	// Event buses
	buses    map[EventBusType]eventbus.EventBus
	
	// Default bus
	defaultBus EventBusType
	
	// Type routing
	typeRoutes map[string]EventBusType
	
	// Aggregate routing
	aggregateRoutes map[string]EventBusType
	
	// Priority order
	priorityOrder []EventBusType
	
	// Synchronization
	mu sync.RWMutex
}

// EventBusRouterConfig contains configuration for the EventBusRouter
type EventBusRouterConfig struct {
	Strategy        EventRoutingStrategy
	DefaultBus      EventBusType
	TypeRoutes      map[string]EventBusType
	AggregateRoutes map[string]EventBusType
	PriorityOrder   []EventBusType
}

// NewEventBusRouter creates a new event bus router
func NewEventBusRouter(
	logger *zap.Logger,
	config EventBusRouterConfig,
) *EventBusRouter {
	return &EventBusRouter{
		logger:          logger,
		strategy:        config.Strategy,
		buses:           make(map[EventBusType]eventbus.EventBus),
		defaultBus:      config.DefaultBus,
		typeRoutes:      config.TypeRoutes,
		aggregateRoutes: config.AggregateRoutes,
		priorityOrder:   config.PriorityOrder,
	}
}

// RegisterEventBus registers an event bus with the router
func (r *EventBusRouter) RegisterEventBus(busType EventBusType, bus eventbus.EventBus) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.buses[busType] = bus
	r.logger.Info("Registered event bus", zap.String("type", string(busType)))
}

// SetDefaultBus sets the default event bus
func (r *EventBusRouter) SetDefaultBus(busType EventBusType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, ok := r.buses[busType]; !ok {
		return fmt.Errorf("event bus %s not registered", busType)
	}
	
	r.defaultBus = busType
	r.logger.Info("Set default event bus", zap.String("type", string(busType)))
	
	return nil
}

// SetTypeRoute sets the event bus for a specific event type
func (r *EventBusRouter) SetTypeRoute(eventType string, busType EventBusType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, ok := r.buses[busType]; !ok {
		return fmt.Errorf("event bus %s not registered", busType)
	}
	
	if r.typeRoutes == nil {
		r.typeRoutes = make(map[string]EventBusType)
	}
	
	r.typeRoutes[eventType] = busType
	r.logger.Info("Set event type route",
		zap.String("event_type", eventType),
		zap.String("bus_type", string(busType)),
	)
	
	return nil
}

// SetAggregateRoute sets the event bus for a specific aggregate type
func (r *EventBusRouter) SetAggregateRoute(aggregateType string, busType EventBusType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, ok := r.buses[busType]; !ok {
		return fmt.Errorf("event bus %s not registered", busType)
	}
	
	if r.aggregateRoutes == nil {
		r.aggregateRoutes = make(map[string]EventBusType)
	}
	
	r.aggregateRoutes[aggregateType] = busType
	r.logger.Info("Set aggregate type route",
		zap.String("aggregate_type", aggregateType),
		zap.String("bus_type", string(busType)),
	)
	
	return nil
}

// SetPriorityOrder sets the priority order for event buses
func (r *EventBusRouter) SetPriorityOrder(order []EventBusType) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	for _, busType := range order {
		if _, ok := r.buses[busType]; !ok {
			return fmt.Errorf("event bus %s not registered", busType)
		}
	}
	
	r.priorityOrder = order
	r.logger.Info("Set priority order", zap.Any("order", order))
	
	return nil
}

// getEventBus gets the appropriate event bus for an event
func (r *EventBusRouter) getEventBus(event *eventsourcing.Event) (eventbus.EventBus, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	switch r.strategy {
	case SingleBusStrategy:
		// Use the default bus
		if bus, ok := r.buses[r.defaultBus]; ok {
			return bus, nil
		}
		
	case TypeBasedStrategy:
		// Check if there's a route for this event type
		if busType, ok := r.typeRoutes[event.EventType]; ok {
			if bus, ok := r.buses[busType]; ok {
				return bus, nil
			}
		}
		
		// Fall back to the default bus
		if bus, ok := r.buses[r.defaultBus]; ok {
			return bus, nil
		}
		
	case AggregateBasedStrategy:
		// Check if there's a route for this aggregate type
		if busType, ok := r.aggregateRoutes[event.AggregateType]; ok {
			if bus, ok := r.buses[busType]; ok {
				return bus, nil
			}
		}
		
		// Fall back to the default bus
		if bus, ok := r.buses[r.defaultBus]; ok {
			return bus, nil
		}
		
	case PriorityBasedStrategy:
		// Try buses in priority order
		for _, busType := range r.priorityOrder {
			if bus, ok := r.buses[busType]; ok {
				return bus, nil
			}
		}
		
		// Fall back to the default bus if not in priority order
		if bus, ok := r.buses[r.defaultBus]; ok {
			return bus, nil
		}
	}
	
	return nil, fmt.Errorf("no suitable event bus found for event %s", event.EventType)
}

// getAllEventBuses gets all registered event buses
func (r *EventBusRouter) getAllEventBuses() []eventbus.EventBus {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	buses := make([]eventbus.EventBus, 0, len(r.buses))
	for _, bus := range r.buses {
		buses = append(buses, bus)
	}
	
	return buses
}

// PublishEvent publishes an event
func (r *EventBusRouter) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	if r.strategy == BroadcastStrategy {
		// Send to all buses
		var lastErr error
		for _, bus := range r.getAllEventBuses() {
			err := bus.PublishEvent(ctx, event)
			if err != nil {
				r.logger.Error("Failed to publish event to bus",
					zap.String("event_type", event.EventType),
					zap.Error(err),
				)
				lastErr = err
			}
		}
		
		return lastErr
	}
	
	// Get the appropriate bus
	bus, err := r.getEventBus(event)
	if err != nil {
		return err
	}
	
	// Publish the event
	return bus.PublishEvent(ctx, event)
}

// PublishEvents publishes multiple events
func (r *EventBusRouter) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	if len(events) == 0 {
		return nil
	}
	
	if r.strategy == BroadcastStrategy {
		// Send to all buses
		var lastErr error
		for _, bus := range r.getAllEventBuses() {
			err := bus.PublishEvents(ctx, events)
			if err != nil {
				r.logger.Error("Failed to publish events to bus", zap.Error(err))
				lastErr = err
			}
		}
		
		return lastErr
	}
	
	// Group events by bus
	eventsByBus := make(map[eventbus.EventBus][]*eventsourcing.Event)
	
	for _, event := range events {
		bus, err := r.getEventBus(event)
		if err != nil {
			return err
		}
		
		eventsByBus[bus] = append(eventsByBus[bus], event)
	}
	
	// Publish events to each bus
	var lastErr error
	for bus, busEvents := range eventsByBus {
		err := bus.PublishEvents(ctx, busEvents)
		if err != nil {
			r.logger.Error("Failed to publish events to bus", zap.Error(err))
			lastErr = err
		}
	}
	
	return lastErr
}

// Subscribe subscribes to all events
func (r *EventBusRouter) Subscribe(handler eventsourcing.EventHandler) error {
	// Subscribe to all buses
	var lastErr error
	for _, bus := range r.getAllEventBuses() {
		err := bus.Subscribe(handler)
		if err != nil {
			r.logger.Error("Failed to subscribe to bus", zap.Error(err))
			lastErr = err
		}
	}
	
	return lastErr
}

// SubscribeToType subscribes to events of a specific type
func (r *EventBusRouter) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	r.mu.RLock()
	
	// Check if there's a specific bus for this event type
	if r.strategy == TypeBasedStrategy {
		if busType, ok := r.typeRoutes[eventType]; ok {
			if bus, ok := r.buses[busType]; ok {
				r.mu.RUnlock()
				return bus.SubscribeToType(eventType, handler)
			}
		}
	}
	
	r.mu.RUnlock()
	
	// Subscribe to all buses
	var lastErr error
	for _, bus := range r.getAllEventBuses() {
		err := bus.SubscribeToType(eventType, handler)
		if err != nil {
			r.logger.Error("Failed to subscribe to type on bus",
				zap.String("event_type", eventType),
				zap.Error(err),
			)
			lastErr = err
		}
	}
	
	return lastErr
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (r *EventBusRouter) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	r.mu.RLock()
	
	// Check if there's a specific bus for this aggregate type
	if r.strategy == AggregateBasedStrategy {
		if busType, ok := r.aggregateRoutes[aggregateType]; ok {
			if bus, ok := r.buses[busType]; ok {
				r.mu.RUnlock()
				return bus.SubscribeToAggregate(aggregateType, handler)
			}
		}
	}
	
	r.mu.RUnlock()
	
	// Subscribe to all buses
	var lastErr error
	for _, bus := range r.getAllEventBuses() {
		err := bus.SubscribeToAggregate(aggregateType, handler)
		if err != nil {
			r.logger.Error("Failed to subscribe to aggregate on bus",
				zap.String("aggregate_type", aggregateType),
				zap.Error(err),
			)
			lastErr = err
		}
	}
	
	return lastErr
}

