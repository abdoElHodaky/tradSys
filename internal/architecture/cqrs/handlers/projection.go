package handlers

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// ProjectionQueryHandler represents a query handler that uses projections
type ProjectionQueryHandler[T any] struct {
	projectionName    string
	projectionManager projection.ProjectionManager
	queryFunc         func(ctx context.Context, projection interface{}, query Query) (T, error)
	logger            *zap.Logger
}

// NewProjectionQueryHandler creates a new projection query handler
func NewProjectionQueryHandler[T any](
	projectionName string,
	projectionManager projection.ProjectionManager,
	queryFunc func(ctx context.Context, projection interface{}, query Query) (T, error),
	logger *zap.Logger,
) *ProjectionQueryHandler[T] {
	return &ProjectionQueryHandler[T]{
		projectionName:    projectionName,
		projectionManager: projectionManager,
		queryFunc:         queryFunc,
		logger:            logger,
	}
}

// Handle handles a query and returns a result
func (h *ProjectionQueryHandler[T]) Handle(ctx context.Context, query Query) (T, error) {
	// Get the projection
	proj, err := h.getProjection(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	// Execute the query function
	return h.queryFunc(ctx, proj, query)
}

// getProjection gets the projection
func (h *ProjectionQueryHandler[T]) getProjection(ctx context.Context) (interface{}, error) {
	// Check if the projection manager implements the GetProjection method
	if getter, ok := h.projectionManager.(interface {
		GetProjection(ctx context.Context, projectionName string) (interface{}, error)
	}); ok {
		return getter.GetProjection(ctx, h.projectionName)
	}

	return nil, ErrProjectionNotFound
}

// CachedProjectionQueryHandler represents a query handler that uses cached projections
type CachedProjectionQueryHandler[T any] struct {
	projectionName    string
	projectionManager projection.ProjectionManager
	queryFunc         func(ctx context.Context, projection interface{}, query Query) (T, error)
	logger            *zap.Logger
	cache             map[string]interface{}
	cacheTTL          time.Duration
	cacheTime         time.Time
	mu                sync.RWMutex
}

// NewCachedProjectionQueryHandler creates a new cached projection query handler
func NewCachedProjectionQueryHandler[T any](
	projectionName string,
	projectionManager projection.ProjectionManager,
	queryFunc func(ctx context.Context, projection interface{}, query Query) (T, error),
	logger *zap.Logger,
	cacheTTL time.Duration,
) *CachedProjectionQueryHandler[T] {
	return &CachedProjectionQueryHandler[T]{
		projectionName:    projectionName,
		projectionManager: projectionManager,
		queryFunc:         queryFunc,
		logger:            logger,
		cache:             make(map[string]interface{}),
		cacheTTL:          cacheTTL,
	}
}

// Handle handles a query and returns a result
func (h *CachedProjectionQueryHandler[T]) Handle(ctx context.Context, query Query) (T, error) {
	// Get the projection
	proj, err := h.getProjection(ctx)
	if err != nil {
		var zero T
		return zero, err
	}

	// Execute the query function
	return h.queryFunc(ctx, proj, query)
}

// getProjection gets the projection
func (h *CachedProjectionQueryHandler[T]) getProjection(ctx context.Context) (interface{}, error) {
	h.mu.RLock()

	// Check if the cache is valid
	if time.Since(h.cacheTime) < h.cacheTTL {
		// Get the projection from the cache
		proj, ok := h.cache[h.projectionName]
		h.mu.RUnlock()

		if ok {
			return proj, nil
		}
	} else {
		h.mu.RUnlock()
	}

	// Cache is invalid or projection not found, get the projection from the manager
	h.mu.Lock()
	defer h.mu.Unlock()

	// Check if the projection manager implements the GetProjection method
	if getter, ok := h.projectionManager.(interface {
		GetProjection(ctx context.Context, projectionName string) (interface{}, error)
	}); ok {
		proj, err := getter.GetProjection(ctx, h.projectionName)
		if err != nil {
			return nil, err
		}

		// Update the cache
		h.cache[h.projectionName] = proj
		h.cacheTime = time.Now()

		return proj, nil
	}

	return nil, ErrProjectionNotFound
}

// ProjectionEventHandler represents an event handler that updates a projection
type ProjectionEventHandler struct {
	projection projection.Projection
	logger     *zap.Logger
}

// NewProjectionEventHandler creates a new projection event handler
func NewProjectionEventHandler(projection projection.Projection, logger *zap.Logger) *ProjectionEventHandler {
	return &ProjectionEventHandler{
		projection: projection,
		logger:     logger,
	}
}

// HandleEvent handles an event
func (h *ProjectionEventHandler) HandleEvent(event *eventsourcing.Event) error {
	// Handle the event
	err := h.projection.HandleEvent(context.Background(), event)
	if err != nil {
		h.logger.Error("Failed to handle event",
			zap.String("event_type", event.EventType),
			zap.String("aggregate_id", event.AggregateID),
			zap.String("aggregate_type", event.AggregateType),
			zap.String("projection", h.projection.GetName()),
			zap.Error(err))
		return err
	}

	return nil
}

// Common errors
var (
	ErrProjectionNotFound = projection.ErrProjectionNotFound
)
