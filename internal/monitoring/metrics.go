package monitoring

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"go.uber.org/zap"
)

// MetricsCollector collects metrics for the trading system
type MetricsCollector struct {
	logger *zap.Logger
	
	// System metrics
	systemStartTime time.Time
	
	// Market data metrics
	marketDataReceived *prometheus.CounterVec
	marketDataLatency  *prometheus.HistogramVec
	
	// Order metrics
	ordersCreated     *prometheus.CounterVec
	ordersFilled      *prometheus.CounterVec
	ordersCancelled   *prometheus.CounterVec
	ordersRejected    *prometheus.CounterVec
	orderLatency      *prometheus.HistogramVec
	
	// WebSocket metrics
	wsConnections     *prometheus.GaugeVec
	wsMessagesReceived *prometheus.CounterVec
	wsMessagesSent    *prometheus.CounterVec
	
	// Strategy metrics
	strategyPnL       *prometheus.GaugeVec
	strategyTrades    *prometheus.CounterVec
	
	// Database metrics
	dbQueries         *prometheus.CounterVec
	dbQueryLatency    *prometheus.HistogramVec
	
	// Mutex for thread safety
	mu                sync.RWMutex
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(logger *zap.Logger) *MetricsCollector {
	collector := &MetricsCollector{
		logger:          logger,
		systemStartTime: time.Now(),
	}
	
	// Initialize Prometheus metrics
	collector.initializeMetrics()
	
	return collector
}

// initializeMetrics initializes Prometheus metrics
func (c *MetricsCollector) initializeMetrics() {
	// Market data metrics
	c.marketDataReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "market_data_received_total",
			Help: "Total number of market data messages received",
		},
		[]string{"symbol", "exchange"},
	)
	
	c.marketDataLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "market_data_latency_seconds",
			Help:    "Latency of market data processing in seconds",
			Buckets: prometheus.ExponentialBuckets(0.0001, 2, 10), // 100µs to ~100ms
		},
		[]string{"symbol", "exchange"},
	)
	
	// Order metrics
	c.ordersCreated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_created_total",
			Help: "Total number of orders created",
		},
		[]string{"symbol", "side", "type"},
	)
	
	c.ordersFilled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_filled_total",
			Help: "Total number of orders filled",
		},
		[]string{"symbol", "side", "type"},
	)
	
	c.ordersCancelled = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_cancelled_total",
			Help: "Total number of orders cancelled",
		},
		[]string{"symbol", "side", "type"},
	)
	
	c.ordersRejected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "orders_rejected_total",
			Help: "Total number of orders rejected",
		},
		[]string{"symbol", "side", "type", "reason"},
	)
	
	c.orderLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "order_latency_seconds",
			Help:    "Latency of order processing in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"symbol", "side", "type"},
	)
	
	// WebSocket metrics
	c.wsConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "ws_connections",
			Help: "Number of active WebSocket connections",
		},
		[]string{"type"},
	)
	
	c.wsMessagesReceived = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_messages_received_total",
			Help: "Total number of WebSocket messages received",
		},
		[]string{"type", "channel"},
	)
	
	c.wsMessagesSent = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "ws_messages_sent_total",
			Help: "Total number of WebSocket messages sent",
		},
		[]string{"type", "channel"},
	)
	
	// Strategy metrics
	c.strategyPnL = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "strategy_pnl",
			Help: "Current P&L of trading strategies",
		},
		[]string{"strategy", "symbol"},
	)
	
	c.strategyTrades = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "strategy_trades_total",
			Help: "Total number of trades executed by strategies",
		},
		[]string{"strategy", "symbol", "side"},
	)
	
	// Database metrics
	c.dbQueries = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "db_queries_total",
			Help: "Total number of database queries",
		},
		[]string{"operation", "table"},
	)
	
	c.dbQueryLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "db_query_latency_seconds",
			Help:    "Latency of database queries in seconds",
			Buckets: prometheus.ExponentialBuckets(0.001, 2, 10), // 1ms to ~1s
		},
		[]string{"operation", "table"},
	)
}

// RecordMarketData records market data metrics
func (c *MetricsCollector) RecordMarketData(symbol, exchange string, latency time.Duration) {
	c.marketDataReceived.WithLabelValues(symbol, exchange).Inc()
	c.marketDataLatency.WithLabelValues(symbol, exchange).Observe(latency.Seconds())
}

// RecordOrderCreated records order creation metrics
func (c *MetricsCollector) RecordOrderCreated(symbol, side, orderType string, latency time.Duration) {
	c.ordersCreated.WithLabelValues(symbol, side, orderType).Inc()
	c.orderLatency.WithLabelValues(symbol, side, orderType).Observe(latency.Seconds())
}

// RecordOrderFilled records order fill metrics
func (c *MetricsCollector) RecordOrderFilled(symbol, side, orderType string) {
	c.ordersFilled.WithLabelValues(symbol, side, orderType).Inc()
}

// RecordOrderCancelled records order cancellation metrics
func (c *MetricsCollector) RecordOrderCancelled(symbol, side, orderType string) {
	c.ordersCancelled.WithLabelValues(symbol, side, orderType).Inc()
}

// RecordOrderRejected records order rejection metrics
func (c *MetricsCollector) RecordOrderRejected(symbol, side, orderType, reason string) {
	c.ordersRejected.WithLabelValues(symbol, side, orderType, reason).Inc()
}

// RecordWebSocketConnection records WebSocket connection metrics
func (c *MetricsCollector) RecordWebSocketConnection(connectionType string, count int) {
	c.wsConnections.WithLabelValues(connectionType).Set(float64(count))
}

// RecordWebSocketMessageReceived records WebSocket message received metrics
func (c *MetricsCollector) RecordWebSocketMessageReceived(messageType, channel string) {
	c.wsMessagesReceived.WithLabelValues(messageType, channel).Inc()
}

// RecordWebSocketMessageSent records WebSocket message sent metrics
func (c *MetricsCollector) RecordWebSocketMessageSent(messageType, channel string) {
	c.wsMessagesSent.WithLabelValues(messageType, channel).Inc()
}

// RecordStrategyPnL records strategy P&L metrics
func (c *MetricsCollector) RecordStrategyPnL(strategy, symbol string, pnl float64) {
	c.strategyPnL.WithLabelValues(strategy, symbol).Set(pnl)
}

// RecordStrategyTrade records strategy trade metrics
func (c *MetricsCollector) RecordStrategyTrade(strategy, symbol, side string) {
	c.strategyTrades.WithLabelValues(strategy, symbol, side).Inc()
}

// RecordDatabaseQuery records database query metrics
func (c *MetricsCollector) RecordDatabaseQuery(operation, table string, latency time.Duration) {
	c.dbQueries.WithLabelValues(operation, table).Inc()
	c.dbQueryLatency.WithLabelValues(operation, table).Observe(latency.Seconds())
}

// GetUptime returns the system uptime
func (c *MetricsCollector) GetUptime() time.Duration {
	return time.Since(c.systemStartTime)
}

// GetMetrics returns a snapshot of the current metrics
func (c *MetricsCollector) GetMetrics() map[string]interface{} {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	metrics := make(map[string]interface{})
	
	// System metrics
	metrics["uptime"] = c.GetUptime().String()
	
	// Other metrics would be fetched from Prometheus
	// This is just a placeholder for direct metrics access
	
	return metrics
}
