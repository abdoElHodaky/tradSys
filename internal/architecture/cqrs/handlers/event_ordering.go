package handlers

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// EventOrderingGuarantee defines the level of ordering guarantee
type EventOrderingGuarantee int

const (
	// NoOrdering indicates no ordering guarantees
	NoOrdering EventOrderingGuarantee = iota

	// AggregateOrdering guarantees ordering within a single aggregate
	AggregateOrdering

	// TypeOrdering guarantees ordering within an event type
	TypeOrdering

	// GlobalOrdering guarantees global ordering of all events
	GlobalOrdering
)

// EventOrderingValidator validates event ordering across different event bus implementations
type EventOrderingValidator struct {
	logger *zap.Logger

	// Sequence tracking
	aggregateSequences map[string]int64 // aggregateID -> sequence
	typeSequences      map[string]int64 // eventType -> sequence
	globalSequence     int64

	// Correlation tracking
	correlations map[string]time.Time // correlationID -> timestamp

	// Configuration
	requiredGuarantee EventOrderingGuarantee

	// Statistics
	violations int64
	processed  int64

	// Synchronization
	mu sync.RWMutex
}

// NewEventOrderingValidator creates a new event ordering validator
func NewEventOrderingValidator(
	logger *zap.Logger,
	requiredGuarantee EventOrderingGuarantee,
) *EventOrderingValidator {
	return &EventOrderingValidator{
		logger:             logger,
		aggregateSequences: make(map[string]int64),
		typeSequences:      make(map[string]int64),
		globalSequence:     0,
		correlations:       make(map[string]time.Time),
		requiredGuarantee:  requiredGuarantee,
		violations:         0,
		processed:          0,
	}
}

// ValidateEvent validates the ordering of an event
func (v *EventOrderingValidator) ValidateEvent(event *eventsourcing.Event) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.processed++

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

	// Validate based on the required guarantee
	switch v.requiredGuarantee {
	case GlobalOrdering:
		// Check global sequence
		if event.Version <= v.globalSequence {
			v.violations++
			return fmt.Errorf("global ordering violation: event version %d <= global sequence %d",
				event.Version, v.globalSequence)
		}
		v.globalSequence = event.Version

		// Fall through to also check type and aggregate ordering
		fallthrough

	case TypeOrdering:
		// Check type sequence
		if seq, ok := v.typeSequences[event.EventType]; ok {
			if event.Version <= seq {
				v.violations++
				return fmt.Errorf("type ordering violation: event version %d <= type sequence %d for type %s",
					event.Version, seq, event.EventType)
			}
		}
		v.typeSequences[event.EventType] = event.Version

		// Fall through to also check aggregate ordering
		fallthrough

	case AggregateOrdering:
		// Check aggregate sequence
		aggregateKey := event.AggregateID + ":" + event.AggregateType
		if seq, ok := v.aggregateSequences[aggregateKey]; ok {
			if event.Version <= seq {
				v.violations++
				return fmt.Errorf("aggregate ordering violation: event version %d <= aggregate sequence %d for aggregate %s",
					event.Version, seq, aggregateKey)
			}
		}
		v.aggregateSequences[aggregateKey] = event.Version
	}

	return nil
}

// GetStatistics returns statistics about the validator
func (v *EventOrderingValidator) GetStatistics() (processed int64, violations int64) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	return v.processed, v.violations
}

// LogStatistics logs statistics about the validator
func (v *EventOrderingValidator) LogStatistics() {
	processed, violations := v.GetStatistics()

	v.logger.Info("Event ordering statistics",
		zap.Int64("processed", processed),
		zap.Int64("violations", violations),
		zap.Float64("violation_rate", float64(violations)/float64(processed)*100),
	)
}

// OrderingEventHandler is an event handler that validates event ordering
type OrderingEventHandler struct {
	validator *EventOrderingValidator
	logger    *zap.Logger
}

// NewOrderingEventHandler creates a new ordering event handler
func NewOrderingEventHandler(validator *EventOrderingValidator, logger *zap.Logger) *OrderingEventHandler {
	return &OrderingEventHandler{
		validator: validator,
		logger:    logger,
	}
}

// HandleEvent handles an event and validates its ordering
func (h *OrderingEventHandler) HandleEvent(event *eventsourcing.Event) error {
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

// OrderingEventBusDecorator decorates an event bus with ordering validation
type OrderingEventBusDecorator struct {
	eventBus   eventbus.EventBus
	validator  *EventOrderingValidator
	logger     *zap.Logger
	addHandler bool
}

// NewOrderingEventBusDecorator creates a new ordering event bus decorator
func NewOrderingEventBusDecorator(
	eventBus eventbus.EventBus,
	validator *EventOrderingValidator,
	logger *zap.Logger,
	addHandler bool,
) *OrderingEventBusDecorator {
	decorator := &OrderingEventBusDecorator{
		eventBus:   eventBus,
		validator:  validator,
		logger:     logger,
		addHandler: addHandler,
	}

	// Add a handler to validate all events if requested
	if addHandler {
		handler := NewOrderingEventHandler(validator, logger)
		err := eventBus.Subscribe(handler)
		if err != nil {
			logger.Error("Failed to subscribe ordering handler", zap.Error(err))
		}
	}

	return decorator
}

// PublishEvent publishes an event with ordering validation
func (d *OrderingEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
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

// PublishEvents publishes multiple events with ordering validation
func (d *OrderingEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
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
func (d *OrderingEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	return d.eventBus.Subscribe(handler)
}

// SubscribeToType subscribes to events of a specific type
func (d *OrderingEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToType(eventType, handler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *OrderingEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	return d.eventBus.SubscribeToAggregate(aggregateType, handler)
}
