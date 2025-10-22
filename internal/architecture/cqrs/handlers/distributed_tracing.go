package handlers

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/core"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// TracingConfig contains configuration for distributed tracing
type TracingConfig struct {
	// Enabled determines if tracing is enabled
	Enabled bool
	
	// SamplingRate is the rate at which traces are sampled (1 in N)
	SamplingRate int
	
	// ExportEndpoint is the endpoint to export traces to
	ExportEndpoint string
	
	// ServiceName is the name of the service
	ServiceName string
}

// DefaultTracingConfig returns the default tracing configuration
func DefaultTracingConfig() TracingConfig {
	return TracingConfig{
		Enabled:        true,
		SamplingRate:   10,
		ExportEndpoint: "http://localhost:14268/api/traces",
		ServiceName:    "tradSys",
	}
}

// SpanContext represents the context of a span
type SpanContext struct {
	// TraceID is the ID of the trace
	TraceID string
	
	// SpanID is the ID of the span
	SpanID string
	
	// ParentSpanID is the ID of the parent span
	ParentSpanID string
	
	// StartTime is the start time of the span
	StartTime time.Time
	
	// EndTime is the end time of the span
	EndTime time.Time
	
	// Operation is the operation being traced
	Operation string
	
	// Tags are key-value pairs that provide additional context
	Tags map[string]string
}

// DistributedTracer provides distributed tracing functionality
type DistributedTracer struct {
	logger *zap.Logger
	
	// Configuration
	config TracingConfig
	
	// Sampling
	sampleCounter int
}

// NewDistributedTracer creates a new distributed tracer
func NewDistributedTracer(logger *zap.Logger, config TracingConfig) *DistributedTracer {
	return &DistributedTracer{
		logger:        logger,
		config:        config,
		sampleCounter: 0,
	}
}

// StartSpan starts a new span
func (t *DistributedTracer) StartSpan(ctx context.Context, operation string) (context.Context, *SpanContext) {
	// Check if tracing is enabled
	if !t.config.Enabled {
		return ctx, nil
	}
	
	// Check if we should sample this trace
	t.sampleCounter++
	if t.sampleCounter % t.config.SamplingRate != 0 {
		return ctx, nil
	}
	
	// Reset the sample counter if it gets too large
	if t.sampleCounter >= 1000000 {
		t.sampleCounter = 0
	}
	
	// Get the parent span from the context
	var parentSpanID string
	if parentSpan, ok := ctx.Value("span").(*SpanContext); ok {
		parentSpanID = parentSpan.SpanID
	}
	
	// Create a new span
	span := &SpanContext{
		TraceID:      uuid.New().String(),
		SpanID:       uuid.New().String(),
		ParentSpanID: parentSpanID,
		StartTime:    time.Now(),
		Operation:    operation,
		Tags:         make(map[string]string),
	}
	
	// Add the span to the context
	newCtx := context.WithValue(ctx, "span", span)
	
	// Log the span start
	t.logger.Debug("Started span",
		zap.String("trace_id", span.TraceID),
		zap.String("span_id", span.SpanID),
		zap.String("parent_span_id", span.ParentSpanID),
		zap.String("operation", span.Operation),
	)
	
	return newCtx, span
}

// EndSpan ends a span
func (t *DistributedTracer) EndSpan(span *SpanContext) {
	// Check if tracing is enabled
	if !t.config.Enabled || span == nil {
		return
	}
	
	// Set the end time
	span.EndTime = time.Now()
	
	// Calculate the duration
	duration := span.EndTime.Sub(span.StartTime)
	
	// Log the span end
	t.logger.Debug("Ended span",
		zap.String("trace_id", span.TraceID),
		zap.String("span_id", span.SpanID),
		zap.String("parent_span_id", span.ParentSpanID),
		zap.String("operation", span.Operation),
		zap.Duration("duration", duration),
		zap.Any("tags", span.Tags),
	)
	
	// Export the span
	t.exportSpan(span)
}

// AddTag adds a tag to a span
func (t *DistributedTracer) AddTag(span *SpanContext, key string, value string) {
	// Check if tracing is enabled
	if !t.config.Enabled || span == nil {
		return
	}
	
	// Add the tag
	span.Tags[key] = value
}

// exportSpan exports a span to the tracing system
func (t *DistributedTracer) exportSpan(span *SpanContext) {
	// This is a placeholder for actual span export logic
	// In a real implementation, this would send the span to a tracing system like Jaeger or Zipkin
	
	// For now, we'll just log that we would export the span
	t.logger.Debug("Would export span",
		zap.String("trace_id", span.TraceID),
		zap.String("span_id", span.SpanID),
		zap.String("parent_span_id", span.ParentSpanID),
		zap.String("operation", span.Operation),
		zap.Duration("duration", span.EndTime.Sub(span.StartTime)),
		zap.Any("tags", span.Tags),
	)
}

// TracingEventBusDecorator decorates an event bus with distributed tracing
type TracingEventBusDecorator struct {
	eventBus eventbus.EventBus
	tracer   *DistributedTracer
	logger   *zap.Logger
}

// NewTracingEventBusDecorator creates a new tracing event bus decorator
func NewTracingEventBusDecorator(
	eventBus eventbus.EventBus,
	tracer *DistributedTracer,
	logger *zap.Logger,
) *TracingEventBusDecorator {
	return &TracingEventBusDecorator{
		eventBus: eventBus,
		tracer:   tracer,
		logger:   logger,
	}
}

// PublishEvent publishes an event with distributed tracing
func (d *TracingEventBusDecorator) PublishEvent(ctx context.Context, event *eventsourcing.Event) error {
	// Start a span for the publish operation
	operation := fmt.Sprintf("publish_event.%s", event.EventType)
	ctx, span := d.tracer.StartSpan(ctx, operation)
	
	// Add tags to the span
	if span != nil {
		d.tracer.AddTag(span, "event_type", event.EventType)
		d.tracer.AddTag(span, "aggregate_id", event.AggregateID)
		d.tracer.AddTag(span, "aggregate_type", event.AggregateType)
		d.tracer.AddTag(span, "version", fmt.Sprintf("%d", event.Version))
		
		// Add the trace ID to the event metadata
		if event.Metadata == nil {
			event.Metadata = make(map[string]string)
		}
		event.Metadata["trace_id"] = span.TraceID
		event.Metadata["span_id"] = span.SpanID
	}
	
	// Publish the event
	err := d.eventBus.PublishEvent(ctx, event)
	
	// End the span
	if span != nil {
		if err != nil {
			d.tracer.AddTag(span, "error", err.Error())
		}
		d.tracer.EndSpan(span)
	}
	
	return err
}

// PublishEvents publishes multiple events with distributed tracing
func (d *TracingEventBusDecorator) PublishEvents(ctx context.Context, events []*eventsourcing.Event) error {
	// Start a span for the publish operation
	operation := "publish_events"
	ctx, span := d.tracer.StartSpan(ctx, operation)
	
	// Add tags to the span
	if span != nil {
		d.tracer.AddTag(span, "event_count", fmt.Sprintf("%d", len(events)))
		
		// Add event types to the span
		for i, event := range events {
			d.tracer.AddTag(span, fmt.Sprintf("event_%d_type", i), event.EventType)
			
			// Add the trace ID to the event metadata
			if event.Metadata == nil {
				event.Metadata = make(map[string]string)
			}
			event.Metadata["trace_id"] = span.TraceID
			event.Metadata["span_id"] = span.SpanID
		}
	}
	
	// Publish the events
	err := d.eventBus.PublishEvents(ctx, events)
	
	// End the span
	if span != nil {
		if err != nil {
			d.tracer.AddTag(span, "error", err.Error())
		}
		d.tracer.EndSpan(span)
	}
	
	return err
}

// Subscribe subscribes to all events
func (d *TracingEventBusDecorator) Subscribe(handler eventsourcing.EventHandler) error {
	// Create a tracing event handler
	tracingHandler := &TracingEventHandler{
		handler: handler,
		tracer:  d.tracer,
		logger:  d.logger,
	}
	
	return d.eventBus.Subscribe(tracingHandler)
}

// SubscribeToType subscribes to events of a specific type
func (d *TracingEventBusDecorator) SubscribeToType(eventType string, handler eventsourcing.EventHandler) error {
	// Create a tracing event handler
	tracingHandler := &TracingEventHandler{
		handler: handler,
		tracer:  d.tracer,
		logger:  d.logger,
	}
	
	return d.eventBus.SubscribeToType(eventType, tracingHandler)
}

// SubscribeToAggregate subscribes to events of a specific aggregate type
func (d *TracingEventBusDecorator) SubscribeToAggregate(aggregateType string, handler eventsourcing.EventHandler) error {
	// Create a tracing event handler
	tracingHandler := &TracingEventHandler{
		handler: handler,
		tracer:  d.tracer,
		logger:  d.logger,
	}
	
	return d.eventBus.SubscribeToAggregate(aggregateType, tracingHandler)
}

// TracingEventHandler is an event handler that adds distributed tracing
type TracingEventHandler struct {
	handler eventsourcing.EventHandler
	tracer  *DistributedTracer
	logger  *zap.Logger
}

// HandleEvent handles an event with distributed tracing
func (h *TracingEventHandler) HandleEvent(event *eventsourcing.Event) error {
	// Create a context
	ctx := context.Background()
	
	// Check if there's a trace ID in the event metadata
	var parentSpanID string
	if event.Metadata != nil {
		if traceID, ok := event.Metadata["trace_id"]; ok {
			// Create a span context with the trace ID
			ctx = context.WithValue(ctx, "trace_id", traceID)
		}
		if spanID, ok := event.Metadata["span_id"]; ok {
			parentSpanID = spanID
		}
	}
	
	// Start a span for the handle operation
	operation := fmt.Sprintf("handle_event.%s", event.EventType)
	ctx, span := h.tracer.StartSpan(ctx, operation)
	
	// Add tags to the span
	if span != nil {
		h.tracer.AddTag(span, "event_type", event.EventType)
		h.tracer.AddTag(span, "aggregate_id", event.AggregateID)
		h.tracer.AddTag(span, "aggregate_type", event.AggregateType)
		h.tracer.AddTag(span, "version", fmt.Sprintf("%d", event.Version))
		h.tracer.AddTag(span, "handler", fmt.Sprintf("%T", h.handler))
		
		// Set the parent span ID if available
		if parentSpanID != "" {
			span.ParentSpanID = parentSpanID
		}
	}
	
	// Handle the event
	err := h.handler.HandleEvent(event)
	
	// End the span
	if span != nil {
		if err != nil {
			h.tracer.AddTag(span, "error", err.Error())
		}
		h.tracer.EndSpan(span)
	}
	
	return err
}

