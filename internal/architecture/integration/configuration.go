package integration

import (
	"go.uber.org/zap"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/handlers"
)

// EventOrderingGuarantee defines the event ordering guarantee level
type EventOrderingGuarantee = handlers.EventOrderingGuarantee

const (
	// AggregateOrdering ensures events are ordered per aggregate
	AggregateOrdering EventOrderingGuarantee = handlers.AggregateOrdering
)

// EventRoutingStrategy defines the event routing strategy
type EventRoutingStrategy = handlers.EventRoutingStrategy

const (
	// SingleBusStrategy routes all events to a single bus
	SingleBusStrategy EventRoutingStrategy = handlers.SingleBusStrategy
)

// EventBusType defines the type of event bus
type EventBusType = handlers.EventBusType

const (
	// InMemoryEventBusType is an in-memory event bus
	InMemoryEventBusType EventBusType = handlers.InMemoryEventBusType
	// NatsEventBusType is a NATS event bus
	NatsEventBusType EventBusType = handlers.NatsEventBusType
	// WatermillEventBusType is a Watermill-based event bus
	WatermillEventBusType EventBusType = handlers.WatermillEventBusType
)

// NatsCQRSConfig contains configuration for NATS CQRS
type NatsCQRSConfig = handlers.NatsCQRSConfig

// DefaultNatsCQRSConfig returns the default NATS CQRS configuration
func DefaultNatsCQRSConfig() NatsCQRSConfig {
	return handlers.DefaultNatsCQRSConfig()
}

// WatermillCQRSConfig contains configuration for Watermill CQRS
type WatermillCQRSConfig = handlers.WatermillCQRSConfig

// DefaultWatermillCQRSConfig returns the default Watermill CQRS configuration
func DefaultWatermillCQRSConfig() WatermillCQRSConfig {
	return handlers.DefaultWatermillCQRSConfig()
}

// EventBusRouterConfig contains configuration for the event bus router
type EventBusRouterConfig = handlers.EventBusRouterConfig

// EventBusRouter routes events to appropriate event buses
type EventBusRouter = handlers.EventBusRouter

// NewEventBusRouter creates a new event bus router
func NewEventBusRouter(logger *zap.Logger, config EventBusRouterConfig) *EventBusRouter {
	return handlers.NewEventBusRouter(logger, config)
}

// CircuitBreakerConfig contains configuration for the circuit breaker
type CircuitBreakerConfig = handlers.CircuitBreakerConfig

// EventOrderingValidator validates event ordering
type EventOrderingValidator = handlers.EventOrderingValidator

// NewEventOrderingValidator creates a new event ordering validator
func NewEventOrderingValidator(logger *zap.Logger, ordering EventOrderingGuarantee) *EventOrderingValidator {
	return handlers.NewEventOrderingValidator(logger, ordering)
}

// CQRSSystem represents a CQRS system with multiple adapters
type CQRSSystem = handlers.CQRSSystem

// CQRSFactory creates CQRS systems
type CQRSFactory = handlers.CQRSFactory

// NewCQRSFactory creates a new CQRS factory
func NewCQRSFactory(
	logger *zap.Logger,
	useWatermill bool,
	useNats bool,
	useCompatLayer bool,
	useMonitoring bool,
) *CQRSFactory {
	return handlers.NewCQRSFactory(logger, useWatermill, useNats, useCompatLayer, useMonitoring)
}
