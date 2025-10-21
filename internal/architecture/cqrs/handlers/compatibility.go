package handlers

import (
	"context"
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/core"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/core"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/handlers"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing/core"
	"go.uber.org/zap"
)

// CompatibilityLayer provides a compatibility layer between the new CQRS implementation
// and the existing event sourcing system. It ensures that events flow correctly between
// both systems and maintains consistency.
type CompatibilityLayer struct {
	// Logger
	logger *zap.Logger

	// CQRS components
	cqrsAdapter *WatermillCQRSAdapter
	
	// Event sourcing components
	eventStore    store.EventStore
	aggregateRepo aggregate.Repository
	eventBus      eventbus.EventBus
	
	// Synchronization
	mu sync.RWMutex
	
	// Mapping of event types to handlers
	eventHandlers map[string][]eventsourcing.EventHandler
	
	// Feature flags for gradual integration
	useNewCommandHandling bool
	useNewEventHandling   bool
	useNewQueryHandling   bool
}

// NewCompatibilityLayer creates a new compatibility layer
func NewCompatibilityLayer(
	cqrsAdapter *WatermillCQRSAdapter,
	eventStore store.EventStore,
	aggregateRepo aggregate.Repository,
	eventBus eventbus.EventBus,
	logger *zap.Logger,
) *CompatibilityLayer {
	return &CompatibilityLayer{
		logger:        logger,
		cqrsAdapter:   cqrsAdapter,
		eventStore:    eventStore,
		aggregateRepo: aggregateRepo,
		eventBus:      eventBus,
		eventHandlers: make(map[string][]eventsourcing.EventHandler),
		
		// Default to using existing implementations
		useNewCommandHandling: false,
		useNewEventHandling:   false,
		useNewQueryHandling:   false,
	}
}

// EnableNewCommandHandling enables the new CQRS command handling
func (c *CompatibilityLayer) EnableNewCommandHandling() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.useNewCommandHandling = true
	c.logger.Info("Enabled new CQRS command handling")
}

// EnableNewEventHandling enables the new CQRS event handling
func (c *CompatibilityLayer) EnableNewEventHandling() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.useNewEventHandling = true
	c.logger.Info("Enabled new CQRS event handling")
}

// EnableNewQueryHandling enables the new CQRS query handling
func (c *CompatibilityLayer) EnableNewQueryHandling() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.useNewQueryHandling = true
	c.logger.Info("Enabled new CQRS query handling")
}

// RegisterEventHandler registers an event handler with both systems
func (c *CompatibilityLayer) RegisterEventHandler(
	eventType string,
	handler eventsourcing.EventHandler,
) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	// Register with the existing event bus
	err := c.eventBus.SubscribeToType(eventType, handler)
	if err != nil {
		return fmt.Errorf("failed to register with existing event bus: %w", err)
	}
	
	// Register with the new CQRS adapter
	err = c.cqrsAdapter.RegisterEventHandler(eventType, handler)
	if err != nil {
		return fmt.Errorf("failed to register with new CQRS adapter: %w", err)
	}
	
	// Store the handler for reference
	if _, ok := c.eventHandlers[eventType]; !ok {
		c.eventHandlers[eventType] = make([]eventsourcing.EventHandler, 0)
	}
	c.eventHandlers[eventType] = append(c.eventHandlers[eventType], handler)
	
	return nil
}

// DispatchCommand dispatches a command to the appropriate system
func (c *CompatibilityLayer) DispatchCommand(ctx context.Context, cmd command.Command) error {
	c.mu.RLock()
	useNew := c.useNewCommandHandling
	c.mu.RUnlock()
	
	if useNew {
		// Use the new CQRS command handling
		return c.cqrsAdapter.DispatchCommand(ctx, cmd)
	}
	
	// Use the existing command handling
	// This would typically call your existing command bus or handler
	// For now, we'll just log that we're using the old system
	c.logger.Info("Using existing command handling",
		zap.String("command", cmd.CommandName()),
		zap.String("aggregate_id", cmd.AggregateID()),
	)
	
	// This is a placeholder - you would implement this to call your existing command handling
	return fmt.Errorf("existing command handling not implemented")
}

// PublishEvent publishes an event to both systems to ensure consistency
func (c *CompatibilityLayer) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	c.mu.RLock()
	useNew := c.useNewEventHandling
	c.mu.RUnlock()
	
	// Always save to the event store for consistency
	err := c.eventStore.SaveEvents(ctx, []*eventsourcing.Event{event})
	if err != nil {
		return fmt.Errorf("failed to save event to store: %w", err)
	}
	
	if useNew {
		// Use the new CQRS event handling
		// Create an event bus adapter from the CQRS adapter
		eventBus := c.cqrsAdapter.CreateEventBusAdapter()
		
		// Publish the event using the new system
		return eventBus.PublishEvent(ctx, event)
	}
	
	// Use the existing event handling
	return c.eventBus.PublishEvent(ctx, event)
}

// SyncEventStores ensures that both event stores are in sync
// This can be used for migration or recovery
func (c *CompatibilityLayer) SyncEventStores(ctx context.Context, aggregateID string, aggregateType string) error {
	// Get events from the existing event store
	events, err := c.eventStore.GetEvents(ctx, aggregateID, aggregateType)
	if err != nil {
		return fmt.Errorf("failed to get events from existing store: %w", err)
	}
	
	// Log the sync operation
	c.logger.Info("Syncing event stores",
		zap.String("aggregate_id", aggregateID),
		zap.String("aggregate_type", aggregateType),
		zap.Int("event_count", len(events)),
	)
	
	// Process each event through both systems
	for _, event := range events {
		// Create an event bus adapter from the CQRS adapter
		eventBus := c.cqrsAdapter.CreateEventBusAdapter()
		
		// Publish to the new system without saving to the store again
		// We're bypassing the store since we already have the events
		err = eventBus.PublishEvent(ctx, event)
		if err != nil {
			return fmt.Errorf("failed to publish event to new system: %w", err)
		}
	}
	
	return nil
}

// GetAggregateFromBothSystems gets an aggregate from both systems and compares them
// This is useful for validation during migration
func (c *CompatibilityLayer) GetAggregateFromBothSystems(
	ctx context.Context,
	aggregateID string,
	aggregateType string,
) (aggregate.Aggregate, aggregate.Aggregate, error) {
	// Get from existing system
	existingAggregate, err := c.aggregateRepo.Load(ctx, aggregateType, aggregateID)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to load from existing system: %w", err)
	}
	
	// Get from new system (this would be implemented based on your new CQRS system)
	// For now, we'll just return the same aggregate
	newAggregate := existingAggregate
	
	return existingAggregate, newAggregate, nil
}

// ValidateConsistency validates that both systems are consistent
func (c *CompatibilityLayer) ValidateConsistency(
	ctx context.Context,
	aggregateIDs []string,
	aggregateType string,
) (bool, []string, error) {
	inconsistencies := make([]string, 0)
	
	for _, aggregateID := range aggregateIDs {
		existingAggregate, newAggregate, err := c.GetAggregateFromBothSystems(ctx, aggregateID, aggregateType)
		if err != nil {
			return false, nil, err
		}
		
		// Compare the aggregates
		// This is a simple version - you would implement a more thorough comparison
		if existingAggregate.Version() != newAggregate.Version() {
			inconsistencies = append(inconsistencies, fmt.Sprintf(
				"Aggregate %s has different versions: existing=%d, new=%d",
				aggregateID,
				existingAggregate.Version(),
				newAggregate.Version(),
			))
		}
	}
	
	return len(inconsistencies) == 0, inconsistencies, nil
}

