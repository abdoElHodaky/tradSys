package integration

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/cqrs/command"
	"github.com/abdoElHodaky/tradSys/internal/eventsourcing"
	"go.uber.org/zap"
)

// PerformanceMetrics contains performance metrics for the CQRS system
type PerformanceMetrics struct {
	// Command metrics
	CommandCount        int64
	CommandLatencyAvg   int64 // in nanoseconds
	CommandLatencyMax   int64 // in nanoseconds
	CommandErrorCount   int64
	
	// Event metrics
	EventCount          int64
	EventLatencyAvg     int64 // in nanoseconds
	EventLatencyMax     int64 // in nanoseconds
	EventErrorCount     int64
	
	// Query metrics
	QueryCount          int64
	QueryLatencyAvg     int64 // in nanoseconds
	QueryLatencyMax     int64 // in nanoseconds
	QueryErrorCount     int64
	
	// System metrics
	EventQueueSize      int64
	CommandQueueSize    int64
}

// PerformanceMonitor monitors the performance of the CQRS system
type PerformanceMonitor struct {
	logger *zap.Logger
	
	// Metrics
	metrics PerformanceMetrics
	
	// Synchronization
	mu sync.RWMutex
	
	// Sampling
	sampleRate      int // 1 in N operations are sampled
	sampleThreshold int // Current sample counter
	
	// Latency tracking
	commandLatencySum int64
	eventLatencySum   int64
	queryLatencySum   int64
}

// NewPerformanceMonitor creates a new performance monitor
func NewPerformanceMonitor(logger *zap.Logger, sampleRate int) *PerformanceMonitor {
	return &PerformanceMonitor{
		logger:     logger,
		sampleRate: sampleRate,
		metrics:    PerformanceMetrics{},
	}
}

// TrackCommandExecution tracks the execution of a command
func (m *PerformanceMonitor) TrackCommandExecution(
	ctx context.Context,
	cmd command.Command,
	fn func(context.Context, command.Command) error,
) error {
	// Record start time
	startTime := time.Now()
	
	// Execute the command
	err := fn(ctx, cmd)
	
	// Record end time and calculate latency
	latency := time.Since(startTime).Nanoseconds()
	
	// Update metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.CommandCount++
	
	if err != nil {
		m.metrics.CommandErrorCount++
	}
	
	// Sample latency metrics to reduce overhead
	m.sampleThreshold++
	if m.sampleThreshold >= m.sampleRate {
		m.sampleThreshold = 0
		
		// Update latency metrics
		m.commandLatencySum += latency
		avgLatency := m.commandLatencySum / m.metrics.CommandCount
		m.metrics.CommandLatencyAvg = avgLatency
		
		if latency > m.metrics.CommandLatencyMax {
			m.metrics.CommandLatencyMax = latency
		}
		
		// Log performance metrics for sampled operations
		m.logger.Debug("Command performance",
			zap.String("command", cmd.CommandName()),
			zap.Int64("latency_ns", latency),
			zap.Int64("avg_latency_ns", avgLatency),
			zap.Int64("max_latency_ns", m.metrics.CommandLatencyMax),
			zap.Error(err),
		)
	}
	
	return err
}

// TrackEventProcessing tracks the processing of an event
func (m *PerformanceMonitor) TrackEventProcessing(
	ctx context.Context,
	event *eventsourcing.Event,
	fn func(context.Context, *eventsourcing.Event) error,
) error {
	// Record start time
	startTime := time.Now()
	
	// Process the event
	err := fn(ctx, event)
	
	// Record end time and calculate latency
	latency := time.Since(startTime).Nanoseconds()
	
	// Update metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.EventCount++
	
	if err != nil {
		m.metrics.EventErrorCount++
	}
	
	// Sample latency metrics to reduce overhead
	m.sampleThreshold++
	if m.sampleThreshold >= m.sampleRate {
		m.sampleThreshold = 0
		
		// Update latency metrics
		m.eventLatencySum += latency
		avgLatency := m.eventLatencySum / m.metrics.EventCount
		m.metrics.EventLatencyAvg = avgLatency
		
		if latency > m.metrics.EventLatencyMax {
			m.metrics.EventLatencyMax = latency
		}
		
		// Log performance metrics for sampled operations
		m.logger.Debug("Event performance",
			zap.String("event_type", event.EventType),
			zap.Int64("latency_ns", latency),
			zap.Int64("avg_latency_ns", avgLatency),
			zap.Int64("max_latency_ns", m.metrics.EventLatencyMax),
			zap.Error(err),
		)
	}
	
	return err
}

// TrackQueryExecution tracks the execution of a query
func (m *PerformanceMonitor) TrackQueryExecution(
	ctx context.Context,
	queryName string,
	fn func(context.Context) (interface{}, error),
) (interface{}, error) {
	// Record start time
	startTime := time.Now()
	
	// Execute the query
	result, err := fn(ctx)
	
	// Record end time and calculate latency
	latency := time.Since(startTime).Nanoseconds()
	
	// Update metrics
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.QueryCount++
	
	if err != nil {
		m.metrics.QueryErrorCount++
	}
	
	// Sample latency metrics to reduce overhead
	m.sampleThreshold++
	if m.sampleThreshold >= m.sampleRate {
		m.sampleThreshold = 0
		
		// Update latency metrics
		m.queryLatencySum += latency
		avgLatency := m.queryLatencySum / m.metrics.QueryCount
		m.metrics.QueryLatencyAvg = avgLatency
		
		if latency > m.metrics.QueryLatencyMax {
			m.metrics.QueryLatencyMax = latency
		}
		
		// Log performance metrics for sampled operations
		m.logger.Debug("Query performance",
			zap.String("query", queryName),
			zap.Int64("latency_ns", latency),
			zap.Int64("avg_latency_ns", avgLatency),
			zap.Int64("max_latency_ns", m.metrics.QueryLatencyMax),
			zap.Error(err),
		)
	}
	
	return result, err
}

// UpdateQueueSizes updates the queue sizes
func (m *PerformanceMonitor) UpdateQueueSizes(eventQueueSize, commandQueueSize int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.metrics.EventQueueSize = eventQueueSize
	m.metrics.CommandQueueSize = commandQueueSize
}

// GetMetrics returns a copy of the current metrics
func (m *PerformanceMonitor) GetMetrics() PerformanceMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return m.metrics
}

// LogMetrics logs the current metrics
func (m *PerformanceMonitor) LogMetrics() {
	metrics := m.GetMetrics()
	
	m.logger.Info("CQRS performance metrics",
		// Command metrics
		zap.Int64("command_count", metrics.CommandCount),
		zap.Int64("command_latency_avg_ns", metrics.CommandLatencyAvg),
		zap.Int64("command_latency_max_ns", metrics.CommandLatencyMax),
		zap.Int64("command_error_count", metrics.CommandErrorCount),
		
		// Event metrics
		zap.Int64("event_count", metrics.EventCount),
		zap.Int64("event_latency_avg_ns", metrics.EventLatencyAvg),
		zap.Int64("event_latency_max_ns", metrics.EventLatencyMax),
		zap.Int64("event_error_count", metrics.EventErrorCount),
		
		// Query metrics
		zap.Int64("query_count", metrics.QueryCount),
		zap.Int64("query_latency_avg_ns", metrics.QueryLatencyAvg),
		zap.Int64("query_latency_max_ns", metrics.QueryLatencyMax),
		zap.Int64("query_error_count", metrics.QueryErrorCount),
		
		// System metrics
		zap.Int64("event_queue_size", metrics.EventQueueSize),
		zap.Int64("command_queue_size", metrics.CommandQueueSize),
	)
}

// StartPeriodicLogging starts periodic logging of metrics
func (m *PerformanceMonitor) StartPeriodicLogging(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.LogMetrics()
		}
	}
}

