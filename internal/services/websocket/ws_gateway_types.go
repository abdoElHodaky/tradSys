package websocket

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MessageType defines WebSocket message types
type MessageType string

const (
	MessageTypeSubscribe   MessageType = "subscribe"
	MessageTypeUnsubscribe MessageType = "unsubscribe"
	MessageTypeMarketData  MessageType = "market_data"
	MessageTypeOrderUpdate MessageType = "order_update"
	MessageTypePortfolio   MessageType = "portfolio"
	MessageTypeAlert       MessageType = "alert"
	MessageTypeHeartbeat   MessageType = "heartbeat"
	MessageTypeError       MessageType = "error"
)

// SubscriptionType defines subscription types
type SubscriptionType string

const (
	SubTypeMarketData   SubscriptionType = "market_data"
	SubTypeOrderBook    SubscriptionType = "order_book"
	SubTypeTrades       SubscriptionType = "trades"
	SubTypeOrderUpdates SubscriptionType = "order_updates"
	SubTypePortfolio    SubscriptionType = "portfolio"
	SubTypeAlerts       SubscriptionType = "alerts"
)

// Gateway manages WebSocket connections and routing for high-performance trading
type Gateway struct {
	// Core components
	connectionManager *ConnectionManager
	messageHandler    *MessageHandler
	performanceOpt    *PerformanceOptimizer

	// Configuration
	config *GatewayConfig
	logger *zap.Logger

	// Connection tracking
	connections map[string]*Connection
	mu          sync.RWMutex

	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc

	// Performance metrics
	metrics *GatewayMetrics
}

// GatewayConfig contains gateway configuration
type GatewayConfig struct {
	MaxConnections     int           `json:"max_connections"`
	MaxMessageSize     int64         `json:"max_message_size"`
	WriteTimeout       time.Duration `json:"write_timeout"`
	ReadTimeout        time.Duration `json:"read_timeout"`
	PingInterval       time.Duration `json:"ping_interval"`
	PongTimeout        time.Duration `json:"pong_timeout"`
	EnableCompression  bool          `json:"enable_compression"`
	BufferSize         int           `json:"buffer_size"`
	MaxSubscriptions   int           `json:"max_subscriptions"`
	RateLimitPerSecond int           `json:"rate_limit_per_second"`
}

// Connection represents a WebSocket connection optimized for trading
type Connection struct {
	ID       string
	UserID   string
	Exchange string

	// WebSocket connection
	conn *websocket.Conn

	// Message channels
	send    chan []byte
	receive chan []byte

	// Subscriptions
	subscriptions map[string]*Subscription
	subMu         sync.RWMutex

	// Performance tracking
	lastActivity     time.Time
	messageCount     int64
	bytesTransferred int64
	latencySum       int64

	// Connection state
	isActive bool
	mu       sync.RWMutex

	// Context for cancellation
	ctx    context.Context
	cancel context.CancelFunc
}

// Subscription represents a channel subscription
type Subscription struct {
	ID       string
	Channel  string
	Symbol   string
	Type     SubscriptionType
	Filters  map[string]interface{}
	Created  time.Time
	LastData time.Time
}

// GatewayMetrics tracks gateway performance
type GatewayMetrics struct {
	TotalConnections  int64         `json:"total_connections"`
	ActiveConnections int64         `json:"active_connections"`
	MessagesPerSecond float64       `json:"messages_per_second"`
	AverageLatency    time.Duration `json:"average_latency"`
	BytesPerSecond    float64       `json:"bytes_per_second"`
	ErrorRate         float64       `json:"error_rate"`
	SubscriptionCount int64         `json:"subscription_count"`
	LastUpdated       time.Time     `json:"last_updated"`
}

// Message represents a WebSocket message
type Message struct {
	Type      MessageType            `json:"type"`
	Channel   string                 `json:"channel,omitempty"`
	Symbol    string                 `json:"symbol,omitempty"`
	Data      interface{}            `json:"data,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	ID        string                 `json:"id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// ConnectionManager manages WebSocket connections
type ConnectionManager struct {
	gateway *Gateway
	logger  *zap.Logger
}

// MessageHandler handles message processing and routing
type MessageHandler struct {
	gateway *Gateway
	logger  *zap.Logger
}

// PerformanceOptimizer optimizes gateway performance
type PerformanceOptimizer struct {
	gateway *Gateway
	logger  *zap.Logger
}

// ConnectionStats represents connection statistics
type ConnectionStats struct {
	ID                string        `json:"id"`
	UserID            string        `json:"user_id"`
	Exchange          string        `json:"exchange"`
	ConnectedAt       time.Time     `json:"connected_at"`
	LastActivity      time.Time     `json:"last_activity"`
	MessageCount      int64         `json:"message_count"`
	BytesTransferred  int64         `json:"bytes_transferred"`
	AverageLatency    time.Duration `json:"average_latency"`
	SubscriptionCount int           `json:"subscription_count"`
	IsActive          bool          `json:"is_active"`
}

// BroadcastOptions contains options for broadcasting messages
type BroadcastOptions struct {
	Channel     string                 `json:"channel"`
	Symbol      string                 `json:"symbol,omitempty"`
	UserFilter  func(string) bool      `json:"-"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Priority    int                    `json:"priority"`
	Compression bool                   `json:"compression"`
}

// RateLimiter manages connection rate limiting
type RateLimiter struct {
	requests    map[string][]time.Time
	maxRequests int
	window      time.Duration
	mu          sync.RWMutex
}

// HealthStatus represents gateway health status
type HealthStatus struct {
	Status            string            `json:"status"`
	ActiveConnections int64             `json:"active_connections"`
	TotalConnections  int64             `json:"total_connections"`
	MessagesPerSecond float64           `json:"messages_per_second"`
	AverageLatency    time.Duration     `json:"average_latency"`
	ErrorRate         float64           `json:"error_rate"`
	Uptime            time.Duration     `json:"uptime"`
	LastCheck         time.Time         `json:"last_check"`
	Components        map[string]string `json:"components"`
}
