package integration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/eventbus"
	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/integration/strategy"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// DynamicOrderingConfig contains configuration for dynamic event ordering
type DynamicOrderingConfig struct {
	// StrategyName is the name of the ordering strategy to use
	StrategyName string
	
	// PluginDir is the directory containing ordering strategy plugins
	PluginDir string
}

// DefaultDynamicOrderingConfig returns the default dynamic ordering configuration
func DefaultDynamicOrderingConfig() DynamicOrderingConfig {
	return DynamicOrderingConfig{
		StrategyName: "aggregate",
		PluginDir:    "/etc/tradsys/cqrs/plugins/ordering",
	}
}

// DynamicEventOrderingValidator validates event ordering with dynamic strategies
type DynamicEventOrderingValidator struct {
	logger *zap.Logger
	
	// Configuration
	config DynamicOrderingConfig
	
	// Strategy factory
	factory *strategy.OrderingStrategyFactory
	
	// Current strategy
	currentStrategy strategy.OrderingStrategy
	
	// Correlation tracking
	correlations map[string]time.Time // correlationID -> timestamp
	
	// Synchronization
	mu sync.RWMutex
}

// NewDynamicEventOrderingValidator creates a new dynamic event ordering validator
func NewDynamicEventOrderingValidator(
	logger *zap.Logger,
	config DynamicOrderingConfig,
) *DynamicEventOrderingValidator {
	// Create the strategy factory
	factory := strategy.NewOrderingStrategyFactory(logger)
	
	// Get the initial strategy
	currentStrategy, ok := factory.GetStrategy(config.StrategyName)
	if !ok {
		// Use the default strategy if the requested one is not found
		currentStrategy = factory.GetDefaultStrategy()
		logger.Warn("Requested ordering strategy not found, using default",
			zap.String("requested", config.StrategyName),
			zap.String("using", currentStrategy.GetName()),
		)
	}
	
	return &DynamicEventOrderingValidator{
		logger:          logger,
		config:          config,
		factory:         factory,
		currentStrategy: currentStrategy,
		correlations:    make(map[string]time.Time),
	}
}

// Initialize initializes the dynamic event ordering validator
func (v *DynamicEventOrderingValidator) Initialize() error {
	// Load strategy plugins
	if v.config.PluginDir != "" {
		if err := v.loadStrategyPlugins(); err != nil {
			v.logger.Warn("Failed to load ordering strategy plugins", zap.Error(err))
		}
	}
	
	return nil
}

// loadStrategyPlugins loads ordering strategy plugins
func (v *DynamicEventOrderingValidator) loadStrategyPlugins() error {
	// Check if the plugin directory exists
	if _, err := os.Stat(v.config.PluginDir); os.IsNotExist(err) {
		v.logger.Warn("Plugin directory does not exist", zap.String("directory", v.config.PluginDir))
		return nil
	}
	
	// Find all .so files in the plugin directory
	files, err := filepath.Glob(filepath.Join(v.config.PluginDir, "*.so"))
	if err != nil {
		return fmt.Errorf("failed to list plugin files: %w", err)
	}
	
	for _, file := range files {
		if err := v.factory.LoadStrategyPlugin(file); err != nil {
			v.logger.Error("Failed to load ordering strategy plugin",
				zap.String("file", file),
				zap.Error(err))
			continue
		}
	}
	
	return nil
}

// ValidateEvent validates the ordering of an event
func (v *DynamicEventOrderingValidator) ValidateEvent(event *eventsourcing.Event) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Check correlation ID
	if correlationID, ok := event.Metadata["correlation_id"]; ok {
		if timestamp, exists := v.correlations[correlationID]; exists {
			// Log the correlation
			v.logger.Debug("Correlated event",
				zap.String("correlation_id", correlationID),
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.Duration("time_since_first", time.Since(timestamp)),
			)
		} else {
			// Record the first occurrence
			v.correlations[correlationID] = time.Now()
		}
	}
	
	// Validate using the current strategy
	return v.currentStrategy.ValidateEvent(event)
}

// SetStrategy sets the current ordering strategy
func (v *DynamicEventOrderingValidator) SetStrategy(strategyName string) error {
	v.mu.Lock()
	defer v.mu.Unlock()
	
	// Get the strategy
	strategy, ok := v.factory.GetStrategy(strategyName)
	if !ok {
		return fmt.Errorf("ordering strategy not found: %s", strategyName)
	}
	
	// Set the current strategy
	v.currentStrategy = strategy
	
	v.logger.Info("Set ordering strategy",
		zap.String("strategy", strategy.GetName()),
		zap.String("description", strategy.GetDescription()),
	)
	
	return nil
}

// GetCurrentStrategy gets the current ordering strategy
func (v *DynamicEventOrderingValidator) GetCurrentStrategy() strategy.OrderingStrategy {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	return v.currentStrategy
}

// RegisterStrategy registers an ordering strategy
func (v *DynamicEventOrderingValidator) RegisterStrategy(strategy strategy.OrderingStrategy) {
	v.factory.RegisterStrategy(strategy)
}

// GetStatistics returns statistics about the validator
func (v *DynamicEventOrderingValidator) GetStatistics() (processed int64, violations int64) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	
	return v.currentStrategy.GetStatistics()
}

// LogStatistics logs statistics about the validator
func (v *DynamicEventOrderingValidator) LogStatistics() {
	processed, violations := v.GetStatistics()
	
	v.logger.Info("Event ordering statistics",
		zap.Int64("processed", processed),
		zap.Int64("violations", violations),
		zap.Float64("violation_rate", float64(violations)/float64(processed)*100),
		zap.String("strategy", v.GetCurrentStrategy().GetName()),
	)
}

// DynamicOrderingEventHandler is an event handler that validates event ordering
type DynamicOrderingEventHandler struct {
	validator *DynamicEventOrderingValidator
	logger    *zap.Logger
}

// NewDynamicOrderingEventHandler creates a new dynamic ordering event handler
func NewDynamicOrderingEventHandler(validator *DynamicEventOrderingValidator, logger *zap.Logger) *DynamicOrderingEventHandler {
	return &DynamicOrderingEventHandler{
		validator: validator,
		logger:    logger,
	}
}

// HandleEvent handles an event and validates its ordering
func (h *DynamicOrderingEventHandler) HandleEvent(event *eventsourcing.Event) error {
	err := h.validator.ValidateEvent(event)
	if err != nil {
		h.logger.Warn("Event ordering violation",
			zap.String("event_type", event.EventType),
			zap.String("aggregate_id", event.AggregateID),
			zap.String("aggregate_type", event.AggregateType),
			zap.Int64("version", event.Version),
			zap.Error(err),
		)
	}
	
	return nil
}

// DynamicOrderingEventBusDecorator decorates an event bus with dynamic ordering validation
type DynamicOrderingEventBusDecorator struct {
	eventBus   eventbus.EventBus
	validator  *DynamicEventOrderingValidator
	logger     *zap.Logger
	addHandler bool
}

// NewDynamicOrderingEventBusDecorator creates a new dynamic ordering event bus decorator
func NewDynamicOrderingEventBusDecorator(
	eventBus eventbus.EventBus,
	validator *DynamicEventOrderingValidator,
	logger *zap.Logger,
	addHandler bool,
) *DynamicOrderingEventBusDecorator {
	decorator := &DynamicOrderingEventBusDecorator{
		eventBus:   eventBus,
		validator:  validator,
		logger:     logger,
		addHandler: addHandler,
	}
	
	// Add a handler to validate all events if requested
	if addHandler {
		handler := NewDynamicOrderingEventHandler(validator, logger)
		err := eventBus.Subscribe(handler)
		if err != nil {
			logger.Error("Failed to subscribe ordering handler", zap.Error(err))
		}
	}
	
	return decorator
}

// PublishEvent publishes an event with dynamic ordering validation
func (d *DynamicOrderingEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Add a correlation ID if not present
	if _, ok := event.Metadata["correlation_id"]; !ok {
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		event.Metadata["correlation_id"] = uuid.New().String()
	}
	
	// Validate the event ordering
	err := d.validator.ValidateEvent(event)
	if err != nil {
		d.logger.Warn("Event ordering violation during publish",
			zap.String("event_type", event.EventType),
			zap.String("aggregate_id", event.AggregateID),
			zap.String("aggregate_type", event.AggregateType),
			zap.Int64("version", event.Version),
			zap.Error(err),
		)
		// Continue publishing despite the violation
	}
	
	// Publish the event
	return d.eventBus.PublishEvent(ctx, event)
}

// PublishEvents publishes multiple events with dynamic ordering validation
func (d *DynamicOrderingEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	// Add correlation IDs and validate ordering for each event
	for _, event := range events {
		// Add a correlation ID if not present
		if _, ok := event.Metadata["correlation_id"]; !ok {
			if event.Metadata == nil {
				event.Metadata = make(map[string]string)
			}
			event.Metadata["correlation_id"] = uuid.New().String()
		}
		
		// Validate the event ordering
		err := d.validator.ValidateEvent(event)
		if err != nil {
			d.logger.Warn("Event ordering violation during batch publish",
				zap.String("event_type", event.EventType),
				zap.String("aggregate_id", event.AggregateID),
				zap.String("aggregate_type", event.AggregateType),
				zap.Int64("version", event.Version),
				zap.Error(err),
			)
			// Continue publishing despite the violation
		}
	}
	
	// Publish the events
	return d.eventBus.PublishEvents(ctx, events)
}

// Subscribe subscribes to all events
func (d *DynamicOrderingEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	return d.eventBus.Subscribe(handler)
}

// SubscribeToType subscribes to events of a specific type
func (d *DynamicOrderingEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToType(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *DynamicOrderingEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToAggregate(aggregateType, handler)
}

