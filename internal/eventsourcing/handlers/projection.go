package handlers

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"go.uber.org/zap"
)

// Projection represents a projection in the event sourcing pattern
type Projection interface {
	// GetName returns the name of the projection
	GetName() string

	// HandleEvent handles an event
	HandleEvent(ctx context.Context, event *eventsourcing.Event) error

	// Reset resets the projection
	Reset(ctx context.Context) error
}

// ProjectionManager manages projections
type ProjectionManager interface {
	// RegisterProjection registers a projection
	RegisterProjection(projection Projection) error

	// RebuildProjection rebuilds a projection
	RebuildProjection(ctx context.Context, projectionName string) error

	// RebuildAllProjections rebuilds all projections
	RebuildAllProjections(ctx context.Context) error

	// HandleEvent handles an event
	HandleEvent(ctx context.Context, event *eventsourcing.Event) error
}

// DefaultProjectionManager provides a default implementation of the ProjectionManager interface
type DefaultProjectionManager struct {
	eventStore  core.EventStore
	projections map[string]Projection
	logger      *zap.Logger
	mu          sync.RWMutex
}

// NewDefaultProjectionManager creates a new default projection manager
func NewDefaultProjectionManager(eventStore core.EventStore, logger *zap.Logger) *DefaultProjectionManager {
	return &DefaultProjectionManager{
		eventStore:  eventStore,
		projections: make(map[string]Projection),
		logger:      logger,
	}
}

// RegisterProjection registers a projection
func (m *DefaultProjectionManager) RegisterProjection(projection Projection) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if the projection is already registered
	if _, ok := m.projections[projection.GetName()]; ok {
		return ErrProjectionAlreadyRegistered
	}

	// Register the projection
	m.projections[projection.GetName()] = projection

	return nil
}

// RebuildProjection rebuilds a projection
func (m *DefaultProjectionManager) RebuildProjection(ctx context.Context, projectionName string) error {
	m.mu.RLock()
	projection, ok := m.projections[projectionName]
	m.mu.RUnlock()

	if !ok {
		return ErrProjectionNotFound
	}

	// Reset the projection
	err := projection.Reset(ctx)
	if err != nil {
		return err
	}

	// Get all events
	events, err := m.eventStore.GetAllEvents(ctx, time.Time{}, 0)
	if err != nil {
		return err
	}

	// Handle events
	for _, event := range events {
		err := projection.HandleEvent(ctx, event)
		if err != nil {
			return err
		}
	}

	return nil
}

// RebuildAllProjections rebuilds all projections
func (m *DefaultProjectionManager) RebuildAllProjections(ctx context.Context) error {
	m.mu.RLock()
	projections := make([]Projection, 0, len(m.projections))
	for _, projection := range m.projections {
		projections = append(projections, projection)
	}
	m.mu.RUnlock()

	// Reset all projections
	for _, projection := range projections {
		err := projection.Reset(ctx)
		if err != nil {
			return err
		}
	}

	// Get all events
	events, err := m.eventStore.GetAllEvents(ctx, time.Time{}, 0)
	if err != nil {
		return err
	}

	// Handle events for all projections
	for _, event := range events {
		for _, projection := range projections {
			err := projection.HandleEvent(ctx, event)
			if err != nil {
				return err
			}
		}
	}

	return nil
}

// HandleEvent handles an event
func (m *DefaultProjectionManager) HandleEvent(ctx context.Context, event *eventsourcing.Event) error {
	m.mu.RLock()
	projections := make([]Projection, 0, len(m.projections))
	for _, projection := range m.projections {
		projections = append(projections, projection)
	}
	m.mu.RUnlock()

	// Handle the event for all projections
	for _, projection := range projections {
		err := projection.HandleEvent(ctx, event)
		if err != nil {
			return err
		}
	}

	return nil
}

// BaseProjection provides a base implementation of the Projection interface
type BaseProjection struct {
	Name          string
	EventHandlers map[string]func(ctx context.Context, event *eventsourcing.Event) error
	logger        *zap.Logger
}

// NewBaseProjection creates a new base projection
func NewBaseProjection(name string, logger *zap.Logger) *BaseProjection {
	return &BaseProjection{
		Name:          name,
		EventHandlers: make(map[string]func(ctx context.Context, event *eventsourcing.Event) error),
		logger:        logger,
	}
}

// GetName returns the name of the projection
func (p *BaseProjection) GetName() string {
	return p.Name
}

// RegisterEventHandler registers an event handler
func (p *BaseProjection) RegisterEventHandler(eventType string, handler func(ctx context.Context, event *eventsourcing.Event) error) {
	p.EventHandlers[eventType] = handler
}

// HandleEvent handles an event
func (p *BaseProjection) HandleEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Check if there's a handler for this event type
	handler, ok := p.EventHandlers[event.EventType]
	if !ok {
		// No handler for this event type
		return nil
	}

	// Handle the event
	return handler(ctx, event)
}

// Reset resets the projection
func (p *BaseProjection) Reset(ctx context.Context) error {
	// This is a no-op in the base implementation
	return nil
}

// Common errors
var (
	ErrProjectionNotFound          = errors.New("projection not found")
	ErrProjectionAlreadyRegistered = errors.New("projection already registered")
)
