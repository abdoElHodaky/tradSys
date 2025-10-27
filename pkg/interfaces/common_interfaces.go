package interfaces

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// HealthStatus represents the health status of a service
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

// Event type constants
const (
	OrderEventCreated  = "order.created"
	OrderEventCanceled = "order.canceled"
	OrderEventFilled   = "order.filled"
	TradeEventExecuted = "trade.executed"
)

// Repository defines a generic repository interface
type Repository interface {
	// Create creates a new entity
	Create(ctx context.Context, entity interface{}) error
	// GetByID retrieves an entity by ID
	GetByID(ctx context.Context, id string) (interface{}, error)
	// Update updates an existing entity
	Update(ctx context.Context, entity interface{}) error
	// Delete deletes an entity by ID
	Delete(ctx context.Context, id string) error
	// List lists entities with pagination
	List(ctx context.Context, offset, limit int) ([]interface{}, error)
}

// OrderRepository defines the interface for order data operations
type OrderRepository interface {
	Repository
	// GetOrdersByUser retrieves orders for a specific user
	GetOrdersByUser(ctx context.Context, userID string) ([]*types.Order, error)
	// GetOrdersBySymbol retrieves orders for a specific symbol
	GetOrdersBySymbol(ctx context.Context, symbol string) ([]*types.Order, error)
	// GetOrdersByStatus retrieves orders by status
	GetOrdersByStatus(ctx context.Context, status string) ([]*types.Order, error)
	// GetActiveOrders retrieves all active orders
	GetActiveOrders(ctx context.Context) ([]*types.Order, error)
}

// TradeRepository defines the interface for trade data operations
type TradeRepository interface {
	Repository
	// GetTradesByUser retrieves trades for a specific user
	GetTradesByUser(ctx context.Context, userID string) ([]*types.Trade, error)
	// GetTradesBySymbol retrieves trades for a specific symbol
	GetTradesBySymbol(ctx context.Context, symbol string) ([]*types.Trade, error)
	// GetTradesByDateRange retrieves trades within a date range
	GetTradesByDateRange(ctx context.Context, start, end time.Time) ([]*types.Trade, error)
	// GetTradesWithFilters retrieves trades with custom filters
	GetTradesWithFilters(ctx context.Context, filters *TradeFilters) ([]*types.Trade, error)
}

// TradeFilters defines filters for trade queries
type TradeFilters struct {
	UserID    string     `json:"user_id,omitempty"`
	Symbol    string     `json:"symbol,omitempty"`
	StartDate *time.Time `json:"start_date,omitempty"`
	EndDate   *time.Time `json:"end_date,omitempty"`
	MinAmount *float64   `json:"min_amount,omitempty"`
	MaxAmount *float64   `json:"max_amount,omitempty"`
	Status    string     `json:"status,omitempty"`
	Limit     int        `json:"limit,omitempty"`
	Offset    int        `json:"offset,omitempty"`
}

// EventStore defines an interface for event storage
type EventStore interface {
	// SaveEvent saves an event to the store
	SaveEvent(ctx context.Context, event Event) error
	// GetEvents retrieves events for an aggregate
	GetEvents(ctx context.Context, aggregateID string) ([]Event, error)
	// GetEventsSince retrieves events since a specific timestamp
	GetEventsSince(ctx context.Context, since time.Time) ([]Event, error)
	// Subscribe subscribes to events
	Subscribe(ctx context.Context, handler EventHandler) error
}

// OrderService defines the interface for order management
type OrderService interface {
	// CreateOrder creates a new order
	CreateOrder(ctx context.Context, order *types.Order) error
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) error
	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*types.Order, error)
	// GetOrdersByUser retrieves orders for a user
	GetOrdersByUser(ctx context.Context, userID string) ([]*types.Order, error)
}

// TradeService defines the interface for trade management
type TradeService interface {
	// ExecuteTrade executes a trade
	ExecuteTrade(ctx context.Context, trade *types.Trade) error
	// GetTrade retrieves a trade by ID
	GetTrade(ctx context.Context, tradeID string) (*types.Trade, error)
	// GetTradesByUser retrieves trades for a user
	GetTradesByUser(ctx context.Context, userID string) ([]*types.Trade, error)
}

// MarketDataService defines the interface for market data management
type MarketDataService interface {
	// GetMarketData retrieves market data for a symbol
	GetMarketData(ctx context.Context, symbol string) (*types.MarketData, error)
	// SubscribeToMarketData subscribes to market data updates
	SubscribeToMarketData(ctx context.Context, symbol string, handler func(*types.MarketData)) error
}

// PositionService defines the interface for position management
type PositionService interface {
	// GetPosition retrieves a position
	GetPosition(ctx context.Context, userID, symbol string) (*types.Position, error)
	// UpdatePosition updates a position
	UpdatePosition(ctx context.Context, position *types.Position) error
	// GetPositionsByUser retrieves all positions for a user
	GetPositionsByUser(ctx context.Context, userID string) ([]*types.Position, error)
}

// RiskService defines the interface for risk management
type RiskService interface {
	// CheckRisk checks if an order passes risk checks
	CheckRisk(ctx context.Context, order *types.Order) error
	// GetRiskMetrics retrieves risk metrics for a user
	GetRiskMetrics(ctx context.Context, userID string) (*types.RiskMetrics, error)
}

// MatchingEngine defines the interface for order matching
type MatchingEngine interface {
	// Start starts the matching engine
	Start(ctx context.Context) error
	// Stop stops the matching engine
	Stop(ctx context.Context) error
	// ProcessOrder processes an order
	ProcessOrder(ctx context.Context, order *types.Order) ([]*types.Trade, error)
	// GetOrderBook retrieves the order book for a symbol
	GetOrderBook(ctx context.Context, symbol string) (*types.OrderBook, error)
}

// Logger defines the interface for logging
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
}

// MetricsCollector defines the interface for metrics collection
type MetricsCollector interface {
	// Counter increments a counter metric
	Counter(name string, value float64, tags map[string]string)
	// Gauge sets a gauge metric
	Gauge(name string, value float64, tags map[string]string)
	// Histogram records a histogram metric
	Histogram(name string, value float64, tags map[string]string)
	// RecordTimer records a timer metric
	RecordTimer(name string, duration time.Duration, tags map[string]string)
	// IncrementCounter increments a counter
	IncrementCounter(name string, tags map[string]string)
	// RecordGauge records a gauge metric
	RecordGauge(name string, value float64, tags map[string]string)
}

// EventPublisher defines the interface for event publishing
type EventPublisher interface {
	// PublishOrderEvent publishes an order event
	PublishOrderEvent(ctx context.Context, event OrderEvent) error
	// PublishTradeEvent publishes a trade event
	PublishTradeEvent(ctx context.Context, event TradeEvent) error
	// PublishMarketDataEvent publishes a market data event
	PublishMarketDataEvent(ctx context.Context, event MarketDataEvent) error
}

// OrderEvent represents an order-related event
type OrderEvent struct {
	Type      string       `json:"type"`
	Order     *types.Order `json:"order"`
	Timestamp time.Time    `json:"timestamp"`
	UserID    string       `json:"user_id"`
}

// TradeEvent represents a trade-related event
type TradeEvent struct {
	Type      string       `json:"type"`
	Trade     *types.Trade `json:"trade"`
	Timestamp time.Time    `json:"timestamp"`
}

// MarketDataEvent represents a market data event
type MarketDataEvent struct {
	Type       string                 `json:"type"`
	Symbol     string                 `json:"symbol"`
	MarketData *types.MarketData      `json:"market_data"`
	Timestamp  time.Time              `json:"timestamp"`
	Metadata   map[string]interface{} `json:"metadata,omitempty"`
}

// Market data event types
const (
	MarketDataEventTick  = "market_data.tick"
	MarketDataEventOHLCV = "market_data.ohlcv"
	MarketDataEventDepth = "market_data.depth"
)

// Event represents a domain event
type Event interface {
	// GetID returns the event ID
	GetID() string
	// GetAggregateID returns the aggregate ID
	GetAggregateID() string
	// GetEventType returns the event type
	GetEventType() string
	// GetTimestamp returns the event timestamp
	GetTimestamp() time.Time
	// GetData returns the event data
	GetData() interface{}
	// GetMetadata returns the event metadata
	GetMetadata() map[string]interface{}
}

// EventHandler handles events
type EventHandler interface {
	// Handle handles an event
	Handle(ctx context.Context, event Event) error
	// GetEventTypes returns the event types this handler can process
	GetEventTypes() []string
}

// Aggregate represents a domain aggregate
type Aggregate interface {
	// GetID returns the aggregate ID
	GetID() string
	// GetVersion returns the aggregate version
	GetVersion() int
	// GetUncommittedEvents returns uncommitted events
	GetUncommittedEvents() []Event
	// MarkEventsAsCommitted marks events as committed
	MarkEventsAsCommitted()
	// LoadFromHistory loads the aggregate from event history
	LoadFromHistory(events []Event) error
}

// Projection defines an interface for event projections
type Projection interface {
	// GetName returns the projection name
	GetName() string
	// Handle handles an event for projection
	Handle(ctx context.Context, event Event) error
	// GetLastProcessedEventID returns the last processed event ID
	GetLastProcessedEventID() string
	// SetLastProcessedEventID sets the last processed event ID
	SetLastProcessedEventID(eventID string)
}

// Cache defines a generic cache interface
type Cache interface {
	// Get retrieves a value from cache
	Get(ctx context.Context, key string) (interface{}, error)
	// Set stores a value in cache
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	// Delete removes a value from cache
	Delete(ctx context.Context, key string) error
	// Exists checks if a key exists in cache
	Exists(ctx context.Context, key string) (bool, error)
	// Clear clears all cache entries
	Clear(ctx context.Context) error
}

// MessageQueue defines a message queue interface
type MessageQueue interface {
	// Publish publishes a message to a topic
	Publish(ctx context.Context, topic string, message interface{}) error
	// Subscribe subscribes to a topic
	Subscribe(ctx context.Context, topic string, handler MessageHandler) error
	// Unsubscribe unsubscribes from a topic
	Unsubscribe(ctx context.Context, topic string) error
	// Close closes the message queue connection
	Close() error
}

// MessageHandler handles messages from a queue
type MessageHandler interface {
	// Handle handles a message
	Handle(ctx context.Context, message Message) error
}

// Message represents a message in a queue
type Message interface {
	// GetID returns the message ID
	GetID() string
	// GetTopic returns the message topic
	GetTopic() string
	// GetPayload returns the message payload
	GetPayload() interface{}
	// GetTimestamp returns the message timestamp
	GetTimestamp() time.Time
	// GetMetadata returns the message metadata
	GetMetadata() map[string]interface{}
	// Ack acknowledges the message
	Ack() error
	// Nack negatively acknowledges the message
	Nack() error
}

// Validator defines a validation interface
type Validator interface {
	// Validate validates an object
	Validate(ctx context.Context, obj interface{}) error
	// ValidateField validates a specific field
	ValidateField(ctx context.Context, field string, value interface{}) error
}

// OrderValidator defines the interface for order validation
type OrderValidator interface {
	// ValidateOrder validates an order
	ValidateOrder(ctx context.Context, order *types.Order) error
	// ValidateOrderRequest validates an order request
	ValidateOrderRequest(ctx context.Context, req interface{}) error
	// ValidateOrderUpdate validates an order update
	ValidateOrderUpdate(ctx context.Context, order *types.Order, req interface{}) error
	// ValidateOrderCancellation validates an order cancellation
	ValidateOrderCancellation(ctx context.Context, order *types.Order, req interface{}) error
}

// Serializer defines a serialization interface
type Serializer interface {
	// Serialize serializes an object to bytes
	Serialize(obj interface{}) ([]byte, error)
	// Deserialize deserializes bytes to an object
	Deserialize(data []byte, obj interface{}) error
	// GetContentType returns the content type
	GetContentType() string
}



// Metrics defines a metrics interface
type Metrics interface {
	// Counter increments a counter metric
	Counter(name string, value float64, tags map[string]string)
	// Gauge sets a gauge metric
	Gauge(name string, value float64, tags map[string]string)
	// Histogram records a histogram metric
	Histogram(name string, value float64, tags map[string]string)
	// Timer records a timer metric
	Timer(name string, duration time.Duration, tags map[string]string)
}

// HealthChecker defines a health check interface
type HealthChecker interface {
	// Check performs a health check
	Check(ctx context.Context) error
	// GetName returns the health check name
	GetName() string
}

// RateLimiter defines a rate limiting interface
type RateLimiter interface {
	// Allow checks if an operation is allowed
	Allow(ctx context.Context, key string) (bool, error)
	// Reserve reserves capacity for an operation
	Reserve(ctx context.Context, key string, tokens int) error
	// GetLimit returns the current limit for a key
	GetLimit(ctx context.Context, key string) (int, error)
	// SetLimit sets the limit for a key
	SetLimit(ctx context.Context, key string, limit int) error
}

// CircuitBreaker defines a circuit breaker interface
type CircuitBreaker interface {
	// Execute executes a function with circuit breaker protection
	Execute(ctx context.Context, fn func() error) error
	// GetState returns the current circuit breaker state
	GetState() string
	// Reset resets the circuit breaker
	Reset()
}

// Tracer defines a distributed tracing interface
type Tracer interface {
	// StartSpan starts a new span
	StartSpan(ctx context.Context, operationName string) (context.Context, Span)
	// InjectHeaders injects trace headers into a map
	InjectHeaders(ctx context.Context, headers map[string]string) error
	// ExtractHeaders extracts trace context from headers
	ExtractHeaders(headers map[string]string) (context.Context, error)
}

// Span represents a trace span
type Span interface {
	// SetTag sets a tag on the span
	SetTag(key string, value interface{})
	// SetError sets an error on the span
	SetError(err error)
	// Finish finishes the span
	Finish()
}

// Database defines a generic database interface
type Database interface {
	// Connect connects to the database
	Connect(ctx context.Context) error
	// Disconnect disconnects from the database
	Disconnect(ctx context.Context) error
	// Ping pings the database
	Ping(ctx context.Context) error
	// BeginTx begins a transaction
	BeginTx(ctx context.Context) (Transaction, error)
	// Execute executes a query
	Execute(ctx context.Context, query string, args ...interface{}) error
	// Query queries the database
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	// QueryRow queries a single row
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
}

// Transaction represents a database transaction
type Transaction interface {
	// Commit commits the transaction
	Commit() error
	// Rollback rolls back the transaction
	Rollback() error
	// Execute executes a query in the transaction
	Execute(ctx context.Context, query string, args ...interface{}) error
	// Query queries the database in the transaction
	Query(ctx context.Context, query string, args ...interface{}) (Rows, error)
	// QueryRow queries a single row in the transaction
	QueryRow(ctx context.Context, query string, args ...interface{}) Row
}

// Rows represents database query results
type Rows interface {
	// Next advances to the next row
	Next() bool
	// Scan scans the current row into variables
	Scan(dest ...interface{}) error
	// Close closes the rows
	Close() error
	// Err returns any error encountered during iteration
	Err() error
}

// Row represents a single database row
type Row interface {
	// Scan scans the row into variables
	Scan(dest ...interface{}) error
}

// FileStorage defines a file storage interface
type FileStorage interface {
	// Upload uploads a file
	Upload(ctx context.Context, path string, data []byte) error
	// Download downloads a file
	Download(ctx context.Context, path string) ([]byte, error)
	// Delete deletes a file
	Delete(ctx context.Context, path string) error
	// Exists checks if a file exists
	Exists(ctx context.Context, path string) (bool, error)
	// List lists files in a directory
	List(ctx context.Context, prefix string) ([]string, error)
}

// Notifier defines a notification interface
type Notifier interface {
	// Send sends a notification
	Send(ctx context.Context, notification Notification) error
	// SendBatch sends multiple notifications
	SendBatch(ctx context.Context, notifications []Notification) error
}

// Notification represents a notification
type Notification interface {
	// GetID returns the notification ID
	GetID() string
	// GetType returns the notification type
	GetType() string
	// GetRecipient returns the notification recipient
	GetRecipient() string
	// GetSubject returns the notification subject
	GetSubject() string
	// GetBody returns the notification body
	GetBody() string
	// GetMetadata returns the notification metadata
	GetMetadata() map[string]interface{}
}



// EngineMetrics represents performance metrics for a matching engine
type EngineMetrics struct {
	OrdersProcessed  uint64        `json:"orders_processed"`
	TradesExecuted   uint64        `json:"trades_executed"`
	AverageLatency   time.Duration `json:"average_latency"`
	ThroughputPerSec float64       `json:"throughput_per_sec"`
	LastProcessedAt  time.Time     `json:"last_processed_at"`
	ActiveOrders     int           `json:"active_orders"`
	QueueDepth       int           `json:"queue_depth"`
}
