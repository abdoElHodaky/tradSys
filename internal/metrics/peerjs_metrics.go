package metrics

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// PeerJSMetrics collects metrics for PeerJS connections
type PeerJSMetrics struct {
	// Peer metrics
	activePeers           prometheus.Gauge
	peerTotal             prometheus.Counter
	peerErrors            prometheus.Counter
	peerDurations         prometheus.Histogram
	
	// Message metrics
	messagesReceived      prometheus.Counter
	messagesSent          prometheus.Counter
	messageErrors         prometheus.Counter
	messageSize           prometheus.Histogram
	messageLatency        prometheus.Histogram
	
	// Signal metrics
	signalsReceived       prometheus.Counter
	signalsSent           prometheus.Counter
	signalErrors          prometheus.Counter
	signalLatency         prometheus.Histogram
	
	// Connection metrics
	connectionAttempts    prometheus.Counter
	connectionSuccesses   prometheus.Counter
	connectionFailures    prometheus.Counter
	
	// Plugin metrics
	pluginsLoaded         prometheus.Gauge
	pluginErrors          prometheus.Counter
	pluginMessageHandled  prometheus.Counter
	
	// Peer tracking for duration calculation
	peerStartTimes        map[string]time.Time
	peerMu                sync.RWMutex
	
	// Logger
	logger                *zap.Logger
}

// NewPeerJSMetrics creates a new PeerJSMetrics
func NewPeerJSMetrics(registry prometheus.Registerer, logger *zap.Logger) *PeerJSMetrics {
	m := &PeerJSMetrics{
		activePeers: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "peerjs_active_peers",
			Help: "Number of active PeerJS peers",
		}),
		peerTotal: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_peer_total",
			Help: "Total number of PeerJS peers",
		}),
		peerErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_peer_errors",
			Help: "Number of PeerJS peer errors",
		}),
		peerDurations: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "peerjs_peer_duration_seconds",
			Help:    "Duration of PeerJS peer connections in seconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1s to ~17m
		}),
		messagesReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_messages_received_total",
			Help: "Total number of PeerJS messages received",
		}),
		messagesSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_messages_sent_total",
			Help: "Total number of PeerJS messages sent",
		}),
		messageErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_message_errors_total",
			Help: "Total number of PeerJS message errors",
		}),
		messageSize: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "peerjs_message_size_bytes",
			Help:    "Size of PeerJS messages in bytes",
			Buckets: prometheus.ExponentialBuckets(64, 2, 10), // 64B to ~32KB
		}),
		messageLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "peerjs_message_latency_seconds",
			Help:    "Latency of PeerJS messages in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}),
		signalsReceived: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_signals_received_total",
			Help: "Total number of PeerJS signals received",
		}),
		signalsSent: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_signals_sent_total",
			Help: "Total number of PeerJS signals sent",
		}),
		signalErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_signal_errors_total",
			Help: "Total number of PeerJS signal errors",
		}),
		signalLatency: prometheus.NewHistogram(prometheus.HistogramOpts{
			Name:    "peerjs_signal_latency_seconds",
			Help:    "Latency of PeerJS signals in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		}),
		connectionAttempts: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_connection_attempts_total",
			Help: "Total number of PeerJS connection attempts",
		}),
		connectionSuccesses: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_connection_successes_total",
			Help: "Total number of successful PeerJS connections",
		}),
		connectionFailures: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_connection_failures_total",
			Help: "Total number of failed PeerJS connections",
		}),
		pluginsLoaded: prometheus.NewGauge(prometheus.GaugeOpts{
			Name: "peerjs_plugins_loaded",
			Help: "Number of PeerJS plugins loaded",
		}),
		pluginErrors: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_plugin_errors_total",
			Help: "Total number of PeerJS plugin errors",
		}),
		pluginMessageHandled: prometheus.NewCounter(prometheus.CounterOpts{
			Name: "peerjs_plugin_message_handled_total",
			Help: "Total number of PeerJS messages handled by plugins",
		}),
		peerStartTimes: make(map[string]time.Time),
		logger:         logger,
	}

	// Register metrics with Prometheus
	registry.MustRegister(
		m.activePeers,
		m.peerTotal,
		m.peerErrors,
		m.peerDurations,
		m.messagesReceived,
		m.messagesSent,
		m.messageErrors,
		m.messageSize,
		m.messageLatency,
		m.signalsReceived,
		m.signalsSent,
		m.signalErrors,
		m.signalLatency,
		m.connectionAttempts,
		m.connectionSuccesses,
		m.connectionFailures,
		m.pluginsLoaded,
		m.pluginErrors,
		m.pluginMessageHandled,
	)

	return m
}

// RecordPeerConnected records a peer connecting
func (m *PeerJSMetrics) RecordPeerConnected(peerID string) {
	m.activePeers.Inc()
	m.peerTotal.Inc()
	
	m.peerMu.Lock()
	m.peerStartTimes[peerID] = time.Now()
	m.peerMu.Unlock()
}

// RecordPeerDisconnected records a peer disconnecting
func (m *PeerJSMetrics) RecordPeerDisconnected(peerID string) {
	m.activePeers.Dec()
	
	m.peerMu.Lock()
	startTime, ok := m.peerStartTimes[peerID]
	if ok {
		duration := time.Since(startTime).Seconds()
		m.peerDurations.Observe(duration)
		delete(m.peerStartTimes, peerID)
	}
	m.peerMu.Unlock()
}

// RecordPeerError records a peer error
func (m *PeerJSMetrics) RecordPeerError() {
	m.peerErrors.Inc()
}

// RecordMessageReceived records a message being received
func (m *PeerJSMetrics) RecordMessageReceived(size int) {
	m.messagesReceived.Inc()
	m.messageSize.Observe(float64(size))
}

// RecordMessageSent records a message being sent
func (m *PeerJSMetrics) RecordMessageSent(size int) {
	m.messagesSent.Inc()
	m.messageSize.Observe(float64(size))
}

// RecordMessageError records a message error
func (m *PeerJSMetrics) RecordMessageError() {
	m.messageErrors.Inc()
}

// RecordMessageLatency records the latency of a message
func (m *PeerJSMetrics) RecordMessageLatency(latency time.Duration) {
	m.messageLatency.Observe(latency.Seconds())
}

// RecordSignalReceived records a signal being received
func (m *PeerJSMetrics) RecordSignalReceived() {
	m.signalsReceived.Inc()
}

// RecordSignalSent records a signal being sent
func (m *PeerJSMetrics) RecordSignalSent() {
	m.signalsSent.Inc()
}

// RecordSignalError records a signal error
func (m *PeerJSMetrics) RecordSignalError() {
	m.signalErrors.Inc()
}

// RecordSignalLatency records the latency of a signal
func (m *PeerJSMetrics) RecordSignalLatency(latency time.Duration) {
	m.signalLatency.Observe(latency.Seconds())
}

// RecordConnectionAttempt records a connection attempt
func (m *PeerJSMetrics) RecordConnectionAttempt() {
	m.connectionAttempts.Inc()
}

// RecordConnectionSuccess records a successful connection
func (m *PeerJSMetrics) RecordConnectionSuccess() {
	m.connectionSuccesses.Inc()
}

// RecordConnectionFailure records a failed connection
func (m *PeerJSMetrics) RecordConnectionFailure() {
	m.connectionFailures.Inc()
}

// RecordPluginLoaded records a plugin being loaded
func (m *PeerJSMetrics) RecordPluginLoaded() {
	m.pluginsLoaded.Inc()
}

// RecordPluginUnloaded records a plugin being unloaded
func (m *PeerJSMetrics) RecordPluginUnloaded() {
	m.pluginsLoaded.Dec()
}

// RecordPluginError records a plugin error
func (m *PeerJSMetrics) RecordPluginError() {
	m.pluginErrors.Inc()
}

// RecordPluginMessageHandled records a message being handled by a plugin
func (m *PeerJSMetrics) RecordPluginMessageHandled() {
	m.pluginMessageHandled.Inc()
}

// GetActivePeers returns the number of active peers
func (m *PeerJSMetrics) GetActivePeers() float64 {
	return getGaugeValue(m.activePeers)
}

// GetPluginsLoaded returns the number of plugins loaded
func (m *PeerJSMetrics) GetPluginsLoaded() float64 {
	return getGaugeValue(m.pluginsLoaded)
}

