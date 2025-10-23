package eventsourcing

import (
	"time"
)

// Event represents an event in the event sourcing pattern
type Event struct {
	ID            string                 `json:"id"`
	AggregateID   string                 `json:"aggregate_id"`
	AggregateType string                 `json:"aggregate_type"`
	EventType     string                 `json:"event_type"`
	Version       int                    `json:"version"`
	Timestamp     time.Time              `json:"timestamp"`
	Payload       map[string]interface{} `json:"payload"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// EventHandler represents a handler for events
type EventHandler interface {
	// HandleEvent handles an event
	HandleEvent(event *Event) error
}

// EventHandlerFunc is a function that implements the EventHandler interface
type EventHandlerFunc func(event *Event) error

// HandleEvent handles an event
func (f EventHandlerFunc) HandleEvent(event *Event) error {
	return f(event)
}

// EventPublisher represents a publisher for events
type EventPublisher interface {
	// PublishEvent publishes an event
	PublishEvent(event *Event) error

	// PublishEvents publishes multiple events
	PublishEvents(events []*Event) error

	// Subscribe subscribes to events
	Subscribe(handler EventHandler) error

	// SubscribeToType subscribes to events of a specific type
	SubscribeToType(eventType string, handler EventHandler) error
}

// EventMetadata contains common metadata keys
const (
	MetadataUserID        = "user_id"
	MetadataTimestamp     = "timestamp"
	MetadataCorrelationID = "correlation_id"
	MetadataCausationID   = "causation_id"
	MetadataSource        = "source"
	MetadataIP            = "ip"
	MetadataUserAgent     = "user_agent"
)

// Common event types
const (
	EventTypeCreated     = "created"
	EventTypeUpdated     = "updated"
	EventTypeDeleted     = "deleted"
	EventTypeActivated   = "activated"
	EventTypeDeactivated = "deactivated"
	EventTypeSubmitted   = "submitted"
	EventTypeApproved    = "approved"
	EventTypeRejected    = "rejected"
	EventTypeCancelled   = "cancelled"
	EventTypeExecuted    = "executed"
	EventTypeExpired     = "expired"
)

// NewEvent creates a new event
func NewEvent(
	aggregateID string,
	aggregateType string,
	eventType string,
	version int,
	payload map[string]interface{},
	metadata map[string]interface{},
) *Event {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	// Set timestamp if not provided
	if _, ok := metadata[MetadataTimestamp]; !ok {
		metadata[MetadataTimestamp] = time.Now().UnixNano()
	}

	return &Event{
		AggregateID:   aggregateID,
		AggregateType: aggregateType,
		EventType:     eventType,
		Version:       version,
		Timestamp:     time.Now(),
		Payload:       payload,
		Metadata:      metadata,
	}
}
