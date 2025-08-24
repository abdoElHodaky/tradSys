package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// WebSocketMetrics collects metrics for WebSocket connections
type WebSocketMetrics struct {
	// Connection metrics
	activeConnections      prometheus.Gauge
	connectionTotal        prometheus.Counter
	connectionErrors       prometheus.Counter
	connectionDurations    prometheus.Histogram
	
	// Message metrics
	messagesReceived       prometheus.Counter
	messagesSent           prometheus.Counter
	messageErrors          prometheus.Counter
	messageSize            prometheus.Histogram
	messageLatency         prometheus.Histogram
	
	// Subscription metrics
	activeSubscriptions    prometheus.Gauge
	subscriptionTotal      prometheus.Counter
	subscriptionErrors     prometheus.Counter
	
	// Batching metrics
	batchSize              prometheus.Histogram
	batchLatency           prometheus.Histogram
	
	// Compression metrics
	compressionRatio       prometheus.Histogram
	compressionTime        prometheus.Histogram
	
	// Connection tracking for duration calculation
	connectionStartTimes   map[string]time.Time
	connectionMu           sync.RWMutex
	
	// Logger
	logger                 *zap.Logger
}

// NewWebSocketMetrics creates a new WebSocketMetrics
func NewWebSocketMetrics(registry prometheus.Registerer, logger *zap.Logger) *WebSocketMetrics {
	m := &WebSocketMetrics{
		activeConnections: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "websocket_active_connections",
			Help: "Number of active WebSocket connections",
		}),
		connectionTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_connection_total",
			Help: "Total number of WebSocket connections",
		}),
		connectionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_connection_errors",
			Help: "Number of WebSocket connection errors",
		}),
		connectionDurations: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_connection_duration_seconds",
			Help:    "Duration of WebSocket connections in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17m
		}),
		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_messages_received_total",
			Help: "Total number of WebSocket messages received",
		}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		}),
		messageErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_message_errors_total",
			Help: "Total number of WebSocket message errors",
		}),
		messageSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_message_size_bytes",
			Help:    "Size of WebSocket messages in bytes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to ~32KB
		}),
		messageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_message_latency_seconds",
			Help:    "Latency of WebSocket messages in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}),
		activeSubscriptions: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "websocket_active_subscriptions",
			Help: "Number of active WebSocket subscriptions",
		}),
		subscriptionTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_subscription_total",
			Help: "Total number of WebSocket subscriptions",
		}),
		subscriptionErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "websocket_subscription_errors",
			Help: "Number of WebSocket subscription errors",
		}),
		batchSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_batch_size",
			Help:    "Size of WebSocket message batches",
			Buckets: prometheus.LinearBuckets(1, 5, 10), // 1 to 46 messages
		}),
		batchLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_batch_latency_seconds",
			Help:    "Latency of WebSocket message batches in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}),
		compressionRatio: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_compression_ratio",
			Help:    "Compression ratio of WebSocket messages",
			Buckets: prometheus.LinearBuckets(1, 0.5, 10), // 1 to 5.5
		}),
		compressionTime: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "websocket_compression_time_seconds",
			Help:    "Time to compress WebSocket messages in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10), // 0.1ms to ~0.1s
		}),
		connectionStartTimes: make(map[string]time.Time),
		logger:               logger,
	}

	// Register metrics with Prometheus
	registry.MustRegister(
		m.activeConnections,
		m.connectionTotal,
		m.connectionErrors,
		m.connectionDurations,
		m.messagesReceived,
		m.messagesSent,
		m.messageErrors,
		m.messageSize,
		m.messageLatency,
		m.activeSubscriptions,
		m.subscriptionTotal,
		m.subscriptionErrors,
		m.batchSize,
		m.batchLatency,
		m.compressionRatio,
		m.compressionTime,
	)

	return m
}

// RecordConnectionOpen records a WebSocket connection opening
func (m *WebSocketMetrics) RecordConnectionOpen(connectionID string) {
	m.activeConnections.Inc()
	m.connectionTotal.Inc()
	
	m.connectionMu.Lock()
	m.connectionStartTimes[connectionID] = time.Now()
	m.connectionMu.Unlock()
}

// RecordConnectionClose records a WebSocket connection closing
func (m *WebSocketMetrics) RecordConnectionClose(connectionID string) {
	m.activeConnections.Dec()
	
	m.connectionMu.Lock()
	startTime, ok := m.connectionStartTimes[connectionID]
	if ok {
		duration := time.Since(startTime).Seconds()
		m.connectionDurations.Observe(duration)
		delete(m.connectionStartTimes, connectionID)
	}
	m.connectionMu.Unlock()
}

// RecordConnectionError records a WebSocket connection error
func (m *WebSocketMetrics) RecordConnectionError() {
	m.connectionErrors.Inc()
}

// RecordMessageReceived records a WebSocket message being received
func (m *WebSocketMetrics) RecordMessageReceived(size int) {
	m.messagesReceived.Inc()
	m.messageSize.Observe(float64(size))
}

// RecordMessageSent records a WebSocket message being sent
func (m *WebSocketMetrics) RecordMessageSent(size int) {
	m.messagesSent.Inc()
	m.messageSize.Observe(float64(size))
}

// RecordMessageError records a WebSocket message error
func (m *WebSocketMetrics) RecordMessageError() {
	m.messageErrors.Inc()
}

// RecordMessageLatency records the latency of a WebSocket message
func (m *WebSocketMetrics) RecordMessageLatency(latency time.Duration) {
	m.messageLatency.Observe(latency.Seconds())
}

// RecordSubscriptionAdd records a WebSocket subscription being added
func (m *WebSocketMetrics) RecordSubscriptionAdd() {
	m.activeSubscriptions.Inc()
	m.subscriptionTotal.Inc()
}

// RecordSubscriptionRemove records a WebSocket subscription being removed
func (m *WebSocketMetrics) RecordSubscriptionRemove() {
	m.activeSubscriptions.Dec()
}

// RecordSubscriptionError records a WebSocket subscription error
func (m *WebSocketMetrics) RecordSubscriptionError() {
	m.subscriptionErrors.Inc()
}

// RecordBatch records a WebSocket message batch
func (m *WebSocketMetrics) RecordBatch(size int, latency time.Duration) {
	m.batchSize.Observe(float64(size))
	m.batchLatency.Observe(latency.Seconds())
}

// RecordCompression records WebSocket message compression
func (m *WebSocketMetrics) RecordCompression(originalSize, compressedSize int, duration time.Duration) {
	if originalSize > 0 && compressedSize > 0 {
		ratio := float64(originalSize) / float64(compressedSize)
		m.compressionRatio.Observe(ratio)
	}
	m.compressionTime.Observe(duration.Seconds())
}

// GetActiveConnections returns the number of active connections
func (m *WebSocketMetrics) GetActiveConnections() float64 {
	return getGaugeValue(m.activeConnections)
}

// GetActiveSubscriptions returns the number of active subscriptions
func (m *WebSocketMetrics) GetActiveSubscriptions() float64 {
	return getGaugeValue(m.activeSubscriptions)
}

// Helper function to get gauge value
func getGaugeValue(gauge prometheus.Gauge) float64 {
	ch := make(chan prometheus.Metric, 1)
	gauge.Collect(ch)
	m := <-ch
	
	var dtoMetric dto.Metric
	m.Write(&dtoMetric)
	
	return *dtoMetric.Gauge.Value
}

