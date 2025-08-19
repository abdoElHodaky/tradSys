package projection

import (
	"context"
	"errors"
	"reflect"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/event"
)

// Projection represents a read model projection in the CQRS pattern
type Projection interface {
	// Name returns the name of the projection
	Name() string
	
	// HandleEvent handles an event and updates the projection
	HandleEvent(ctx context.Context, event event.Event) error
	
	// Reset resets the projection to its initial state
	Reset(ctx context.Context) error
}

// ProjectionManager manages projections and their event handlers
type ProjectionManager struct {
	projections map[string]Projection
	handlers    map[string]map[string]bool // eventType -> projectionName -> bool
	mu          sync.RWMutex
}

// NewProjectionManager creates a new projection manager
func NewProjectionManager() *ProjectionManager {
	return &ProjectionManager{
		projections: make(map[string]Projection),
		handlers:    make(map[string]map[string]bool),
	}
}

// RegisterProjection registers a projection with the manager
func (pm *ProjectionManager) RegisterProjection(projection Projection) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	name := projection.Name()
	if _, exists := pm.projections[name]; exists {
		return errors.New("projection already registered: " + name)
	}
	
	pm.projections[name] = projection
	return nil
}

// RegisterEventHandler registers an event handler for a projection
func (pm *ProjectionManager) RegisterEventHandler(projectionName string, eventType string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	
	if _, exists := pm.projections[projectionName]; !exists {
		return errors.New("projection not registered: " + projectionName)
	}
	
	if _, exists := pm.handlers[eventType]; !exists {
		pm.handlers[eventType] = make(map[string]bool)
	}
	
	pm.handlers[eventType][projectionName] = true
	return nil
}

// HandleEvent handles an event and updates the relevant projections
func (pm *ProjectionManager) HandleEvent(ctx context.Context, event event.Event) error {
	pm.mu.RLock()
	eventType := event.EventName()
	projectionNames, exists := pm.handlers[eventType]
	if !exists {
		pm.mu.RUnlock()
		return nil // No projections registered for this event type
	}
	
	// Get the projections that handle this event type
	var projections []Projection
	for projectionName := range projectionNames {
		projection, exists := pm.projections[projectionName]
		if exists {
			projections = append(projections, projection)
		}
	}
	pm.mu.RUnlock()
	
	// Handle the event with each projection
	var errs []error
	for _, projection := range projections {
		if err := projection.HandleEvent(ctx, event); err != nil {
			errs = append(errs, err)
		}
	}
	
	if len(errs) > 0 {
		return errors.New("one or more projections failed to handle the event")
	}
	
	return nil
}

// GetProjection returns a projection by name
func (pm *ProjectionManager) GetProjection(name string) (Projection, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	
	projection, exists := pm.projections[name]
	if !exists {
		return nil, errors.New("projection not found: " + name)
	}
	
	return projection, nil
}

// ResetProjection resets a projection to its initial state
func (pm *ProjectionManager) ResetProjection(ctx context.Context, name string) error {
	projection, err := pm.GetProjection(name)
	if err != nil {
		return err
	}
	
	return projection.Reset(ctx)
}

// ResetAllProjections resets all projections to their initial state
func (pm *ProjectionManager) ResetAllProjections(ctx context.Context) error {
	pm.mu.RLock()
	projections := make([]Projection, 0, len(pm.projections))
	for _, projection := range pm.projections {
		projections = append(projections, projection)
	}
	pm.mu.RUnlock()
	
	var errs []error
	for _, projection := range projections {
		if err := projection.Reset(ctx); err != nil {
			errs = append(errs, err)
		}
	}
	
	if len(errs) > 0 {
		return errors.New("one or more projections failed to reset")
	}
	
	return nil
}

// RebuildProjection rebuilds a projection from the event store
func (pm *ProjectionManager) RebuildProjection(ctx context.Context, name string, eventStore event.EventStore) error {
	projection, err := pm.GetProjection(name)
	if err != nil {
		return err
	}
	
	// Reset the projection
	if err := projection.Reset(ctx); err != nil {
		return err
	}
	
	// Get all event types that this projection handles
	pm.mu.RLock()
	var eventTypes []string
	for eventType, projections := range pm.handlers {
		if _, exists := projections[name]; exists {
			eventTypes = append(eventTypes, eventType)
		}
	}
	pm.mu.RUnlock()
	
	// Process events for each event type
	for _, eventType := range eventTypes {
		events, err := eventStore.GetEventsByType(ctx, eventType)
		if err != nil {
			return err
		}
		
		for _, event := range events {
			if err := projection.HandleEvent(ctx, event); err != nil {
				return err
			}
		}
	}
	
	return nil
}

