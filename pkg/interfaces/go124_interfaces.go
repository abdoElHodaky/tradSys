package interfaces

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Go 1.24 Enhanced Interfaces using generic type aliases and improved patterns



// MarketDataEvent represents a market data event
type MarketDataEvent interface {
	GetSymbol() string
	GetEventType() string
	GetTimestamp() time.Time
}

// GenericMatchingEngine defines the core interface for order matching engines
// Enhanced with Go 1.24 generic patterns for better type safety
type GenericMatchingEngine[O any, T any] interface {
	// AddOrder adds an order to the matching engine
	AddOrder(ctx context.Context, order O) (types.Result[T], error)

	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error

	// GetOrderBook returns the current order book state
	GetOrderBook(ctx context.Context, symbol string) (types.OrderBook, error)

	// GetTrades returns recent trades for a symbol
	GetTrades(ctx context.Context, symbol string, limit int) ([]T, error)

	// GetMetrics returns engine performance metrics
	GetMetrics(ctx context.Context) types.Metadata

	// Start starts the matching engine
	Start(ctx context.Context) error

	// Stop stops the matching engine gracefully
	Stop(ctx context.Context) error

	// Health returns the engine health status
	Health() types.HealthStatus
}

// OrderMatchingEngine is a specialized matching engine for orders and trades
type OrderMatchingEngine = GenericMatchingEngine[types.Order, types.Trade]

// ServiceManager defines a generic service management interface
type ServiceManager[T any] interface {
	// Register registers a service
	Register(name string, service T) error

	// Unregister removes a service
	Unregister(name string) error

	// Get retrieves a service by name
	Get(name string) (types.Option[T], error)

	// List returns all registered services
	List() map[string]T

	// Start starts all services
	Start(ctx context.Context) error

	// Stop stops all services
	Stop(ctx context.Context) error

	// Health returns the health status of all services
	Health() map[string]types.HealthStatus
}

// TradingServiceManager manages trading-related services
// Note: TradingService interface should be defined elsewhere
type TradingServiceManager = ServiceManager[interface{}]

// Go124EventStore defines a generic event store interface for Go 1.24
type Go124EventStore[E any] interface {
	// Store stores an event
	Store(ctx context.Context, event E) error

	// GetEvents retrieves events by criteria
	GetEvents(ctx context.Context, criteria EventCriteria) ([]E, error)

	// GetEventsByType retrieves events by type
	GetEventsByType(ctx context.Context, eventType string) ([]E, error)

	// GetEventsByAggregateID retrieves events for a specific aggregate
	GetEventsByAggregateID(ctx context.Context, aggregateID string) ([]E, error)

	// Subscribe subscribes to events
	Subscribe(ctx context.Context, handler types.EventHandler[E]) error

	// Unsubscribe unsubscribes from events
	Unsubscribe(ctx context.Context) error
}

// TradingEventStore is a specialized event store for trading events
type TradingEventStore = Go124EventStore[TradingEvent]

// CacheManager defines a generic cache management interface
type CacheManager[K comparable, V any] interface {
	// GetCache returns a cache instance
	GetCache(name string) (types.Cache[K, V], error)

	// CreateCache creates a new cache
	CreateCache(name string, config CacheConfig) (types.Cache[K, V], error)

	// DeleteCache removes a cache
	DeleteCache(name string) error

	// ListCaches returns all cache names
	ListCaches() []string

	// GetStats returns cache statistics
	GetStats() map[string]CacheStats

	// Clear clears all caches
	Clear() error
}

// OrderCacheManager manages order-related caches
type OrderCacheManager = CacheManager[string, types.Order]

// PriceCacheManager manages price-related caches
type PriceCacheManager = CacheManager[string, float64]

// RiskManager defines the interface for risk management
type RiskManager interface {
	// ValidateOrder validates an order against risk rules
	ValidateOrder(ctx context.Context, order types.Order) types.Result[bool]

	// CheckPosition validates a position against risk limits
	CheckPosition(ctx context.Context, position types.Position) types.Result[bool]

	// UpdateLimits updates risk limits for a user
	UpdateLimits(ctx context.Context, userID string, limits RiskLimits) error

	// GetLimits retrieves risk limits for a user
	GetLimits(ctx context.Context, userID string) (types.Option[RiskLimits], error)

	// GetMetrics returns risk management metrics
	GetMetrics(ctx context.Context) types.Metadata
}

// MarketDataProvider defines the interface for market data provision
type MarketDataProvider interface {
	// Subscribe subscribes to market data for symbols
	Subscribe(ctx context.Context, symbols types.SymbolSet, handler MarketDataHandler) error

	// Unsubscribe unsubscribes from market data
	Unsubscribe(ctx context.Context, symbols types.SymbolSet) error

	// GetSnapshot returns current market data snapshot
	GetSnapshot(ctx context.Context, symbol string) (types.Option[MarketDataSnapshot], error)

	// GetHistoricalData returns historical market data
	GetHistoricalData(ctx context.Context, symbol string, from, to time.Time) ([]MarketDataPoint, error)

	// IsConnected returns connection status
	IsConnected() bool

	// GetMetrics returns provider metrics
	GetMetrics() types.Metadata
}

// WebSocketManager defines the interface for WebSocket connection management
type WebSocketManager interface {
	// AddConnection adds a new WebSocket connection
	AddConnection(ctx context.Context, conn WebSocketConnection) error

	// RemoveConnection removes a WebSocket connection
	RemoveConnection(ctx context.Context, connectionID string) error

	// Broadcast broadcasts a message to all connections
	Broadcast(ctx context.Context, message WebSocketMessage) error

	// SendToUser sends a message to a specific user
	SendToUser(ctx context.Context, userID string, message WebSocketMessage) error

	// SendToConnection sends a message to a specific connection
	SendToConnection(ctx context.Context, connectionID string, message WebSocketMessage) error

	// GetConnections returns all active connections
	GetConnections() map[string]WebSocketConnection

	// GetUserConnections returns connections for a specific user
	GetUserConnections(userID string) []WebSocketConnection

	// GetMetrics returns WebSocket metrics
	GetMetrics() types.Metadata
}

// Supporting types for the interfaces

// EventCriteria defines criteria for event queries
type EventCriteria struct {
	AggregateID string
	EventType   string
	From        time.Time
	To          time.Time
	Limit       int
	Offset      int
}

// CacheConfig defines cache configuration
type CacheConfig struct {
	MaxSize        int
	TTL            time.Duration
	EvictionPolicy string
}

// CacheStats defines cache statistics
type CacheStats struct {
	Size      int
	HitRate   float64
	MissRate  float64
	Evictions int64
}

// RiskLimits defines risk limits for trading
type RiskLimits struct {
	MaxOrderSize      float64
	MaxPositionSize   float64
	MaxDailyLoss      float64
	MaxDrawdown       float64
	AllowedSymbols    types.SymbolSet
	RestrictedSymbols types.SymbolSet
}

// TradingEvent represents a trading-related event
type TradingEvent struct {
	ID          string
	Type        string
	AggregateID string
	Data        types.Metadata
	Timestamp   time.Time
	Version     int
}

// MarketDataHandler handles market data updates
type MarketDataHandler = types.EventHandler[MarketDataSnapshot]

// MarketDataSnapshot represents a market data snapshot
type MarketDataSnapshot struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
	Bid       float64
	Ask       float64
	BidSize   float64
	AskSize   float64
}

// MarketDataPoint represents a historical market data point
type MarketDataPoint struct {
	Symbol    string
	Timestamp time.Time
	Open      float64
	High      float64
	Low       float64
	Close     float64
	Volume    float64
}

// WebSocketConnection represents a WebSocket connection
type WebSocketConnection interface {
	// GetID returns the connection ID
	GetID() string

	// GetUserID returns the user ID associated with the connection
	GetUserID() string

	// Send sends a message to the connection
	Send(ctx context.Context, message WebSocketMessage) error

	// Close closes the connection
	Close() error

	// IsActive returns whether the connection is active
	IsActive() bool

	// GetMetadata returns connection metadata
	GetMetadata() types.Metadata
}

// WebSocketMessage represents a WebSocket message
type WebSocketMessage struct {
	Type      string
	Topic     string
	Data      interface{}
	Timestamp time.Time
}

// Note: HealthStatus is now defined in types package for consistency
