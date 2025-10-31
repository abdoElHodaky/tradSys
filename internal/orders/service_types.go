// 🎯 **Order Service Types**
// Generated using TradSys Code Splitting Standards
//
// This file contains type definitions, constants, and data structures
// for the Order Management Service component. All types follow the established
// naming conventions and include comprehensive documentation for order lifecycle management.
//
// Performance Requirements: Standard latency, comprehensive order management
// File size limit: 300 lines

package orders

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	// OrderStatusNew represents a new order
	OrderStatusNew OrderStatus = "new"
	// OrderStatusPending represents a pending order
	OrderStatusPending OrderStatus = "pending"
	// OrderStatusPartiallyFilled represents a partially filled order
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	// OrderStatusFilled represents a filled order
	OrderStatusFilled OrderStatus = "filled"
	// OrderStatusCancelled represents a cancelled order
	OrderStatusCancelled OrderStatus = "cancelled"
	// OrderStatusRejected represents a rejected order
	OrderStatusRejected OrderStatus = "rejected"
	// OrderStatusExpired represents an expired order
	OrderStatusExpired OrderStatus = "expired"
)

// OrderType represents the type of order
type OrderType string

const (
	// OrderTypeLimit represents a limit order
	OrderTypeLimit OrderType = "limit"
	// OrderTypeMarket represents a market order
	OrderTypeMarket OrderType = "market"
	// OrderTypeStopLimit represents a stop limit order
	OrderTypeStopLimit OrderType = "stop_limit"
	// OrderTypeStopMarket represents a stop market order
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	// OrderSideBuy represents a buy order
	OrderSideBuy OrderSide = "buy"
	// OrderSideSell represents a sell order
	OrderSideSell OrderSide = "sell"
)

// TimeInForce represents the time in force of an order
type TimeInForce string

const (
	// TimeInForceGTC represents a good-till-cancelled order
	TimeInForceGTC TimeInForce = "GTC"
	// TimeInForceIOC represents an immediate-or-cancel order
	TimeInForceIOC TimeInForce = "IOC"
	// TimeInForceFOK represents a fill-or-kill order
	TimeInForceFOK TimeInForce = "FOK"
	// TimeInForceDay represents a day order
	TimeInForceDay TimeInForce = "DAY"
)

// Order represents an order
type Order struct {
	// ID is the unique identifier for the order
	ID string `json:"id"`
	// UserID is the user ID
	UserID string `json:"user_id"`
	// ClientOrderID is the client order ID
	ClientOrderID string `json:"client_order_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Side is the side of the order (buy or sell)
	Side OrderSide `json:"side"`
	// Type is the type of the order
	Type OrderType `json:"type"`
	// Price is the price of the order
	Price float64 `json:"price"`
	// StopPrice is the stop price for stop orders
	StopPrice float64 `json:"stop_price"`
	// Quantity is the quantity of the order
	Quantity float64 `json:"quantity"`
	// FilledQuantity is the filled quantity of the order
	FilledQuantity float64 `json:"filled_quantity"`
	// Status is the status of the order
	Status OrderStatus `json:"status"`
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce `json:"time_in_force"`
	// CreatedAt is the time the order was created
	CreatedAt time.Time `json:"created_at"`
	// UpdatedAt is the time the order was last updated
	UpdatedAt time.Time `json:"updated_at"`
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time `json:"expires_at"`
	// Trades is the trades associated with the order
	Trades []*Trade `json:"trades"`
	// Metadata is additional metadata for the order
	Metadata map[string]interface{} `json:"metadata"`
}

// Trade represents a trade
type Trade struct {
	// ID is the unique identifier for the trade
	ID string `json:"id"`
	// OrderID is the order ID
	OrderID string `json:"order_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Side is the side of the trade (buy or sell)
	Side OrderSide `json:"side"`
	// Price is the price of the trade
	Price float64 `json:"price"`
	// Quantity is the quantity of the trade
	Quantity float64 `json:"quantity"`
	// ExecutedAt is the time the trade was executed
	ExecutedAt time.Time `json:"executed_at"`
	// Fee is the fee for the trade
	Fee float64 `json:"fee"`
	// FeeCurrency is the currency of the fee
	FeeCurrency string `json:"fee_currency"`
	// CounterPartyOrderID is the counter party order ID
	CounterPartyOrderID string `json:"counter_party_order_id"`
	// Metadata is additional metadata for the trade
	Metadata map[string]interface{} `json:"metadata"`
}

// OrderFilter represents a filter for orders
type OrderFilter struct {
	// UserID is the user ID
	UserID string `json:"user_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Side is the side of the order (buy or sell)
	Side OrderSide `json:"side"`
	// Type is the type of the order
	Type OrderType `json:"type"`
	// Status is the status of the order
	Status OrderStatus `json:"status"`
	// StartTime is the start time for the filter
	StartTime time.Time `json:"start_time"`
	// EndTime is the end time for the filter
	EndTime time.Time `json:"end_time"`
}

// OrderRequest represents an order request
type OrderRequest struct {
	// UserID is the user ID
	UserID string `json:"user_id"`
	// ClientOrderID is the client order ID
	ClientOrderID string `json:"client_order_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Side is the side of the order (buy or sell)
	Side OrderSide `json:"side"`
	// Type is the type of the order
	Type OrderType `json:"type"`
	// Price is the price of the order
	Price float64 `json:"price"`
	// StopPrice is the stop price for stop orders
	StopPrice float64 `json:"stop_price"`
	// Quantity is the quantity of the order
	Quantity float64 `json:"quantity"`
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce `json:"time_in_force"`
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time `json:"expires_at"`
	// Metadata is additional metadata for the order
	Metadata map[string]interface{} `json:"metadata"`
}

// OrderCancelRequest represents an order cancel request
type OrderCancelRequest struct {
	// UserID is the user ID
	UserID string `json:"user_id"`
	// OrderID is the order ID
	OrderID string `json:"order_id"`
	// ClientOrderID is the client order ID
	ClientOrderID string `json:"client_order_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
}

// OrderUpdateRequest represents an order update request
type OrderUpdateRequest struct {
	// UserID is the user ID
	UserID string `json:"user_id"`
	// OrderID is the order ID
	OrderID string `json:"order_id"`
	// ClientOrderID is the client order ID
	ClientOrderID string `json:"client_order_id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Price is the price of the order
	Price float64 `json:"price"`
	// StopPrice is the stop price for stop orders
	StopPrice float64 `json:"stop_price"`
	// Quantity is the quantity of the order
	Quantity float64 `json:"quantity"`
	// TimeInForce is the time in force of the order
	TimeInForce TimeInForce `json:"time_in_force"`
	// ExpiresAt is the time the order expires
	ExpiresAt time.Time `json:"expires_at"`
}

// Service represents an order management service
type Service struct {
	// Engine is the order matching engine
	Engine *order_matching.Engine
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// UserOrders is a map of user ID to order IDs
	UserOrders map[string]map[string]bool
	// SymbolOrders is a map of symbol to order IDs
	SymbolOrders map[string]map[string]bool
	// ClientOrderIDs is a map of client order ID to order ID
	ClientOrderIDs map[string]string
	// OrderCache is a cache for frequently accessed orders
	OrderCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Batch processing channel for order operations
	orderBatchChan chan orderOperation
}

// orderOperation represents a batch operation on orders
type orderOperation struct {
	opType    string
	order     *Order
	requestID string
	resultCh  chan orderOperationResult
}

// orderOperationResult represents the result of a batch operation
type orderOperationResult struct {
	order *Order
	err   error
}

// ServiceConfig contains configuration for the order service
type ServiceConfig struct {
	// CacheExpiration is the cache expiration time
	CacheExpiration time.Duration `json:"cache_expiration"`
	// CacheCleanupInterval is the cache cleanup interval
	CacheCleanupInterval time.Duration `json:"cache_cleanup_interval"`
	// BatchChannelSize is the size of the batch processing channel
	BatchChannelSize int `json:"batch_channel_size"`
	// MaxOrdersPerUser limits orders per user
	MaxOrdersPerUser int `json:"max_orders_per_user"`
	// EnableBatchProcessing enables batch processing
	EnableBatchProcessing bool `json:"enable_batch_processing"`
}

// Constants for order service operation
const (
	// Default configuration values
	DefaultCacheExpiration     = 5 * time.Minute
	DefaultCacheCleanupInterval = 10 * time.Minute
	DefaultBatchChannelSize    = 1000
	DefaultMaxOrdersPerUser    = 1000
)

// Error definitions (non-validation errors)
var (
	ErrOrderAlreadyExists   = errors.New("order already exists")
	ErrOrderNotCancellable  = errors.New("order not cancellable")
	ErrOrderExpired         = errors.New("order expired")
	ErrInsufficientBalance  = errors.New("insufficient balance")
	ErrInvalidPrice         = errors.New("invalid price")
	ErrMaxOrdersExceeded    = errors.New("maximum orders per user exceeded")
)

// Stop stops the service and cancels all background operations
func (s *Service) Stop() {
	if s.cancel != nil {
		s.cancel()
	}
}
