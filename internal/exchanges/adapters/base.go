package adapters

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// OrderType represents different order types
type OrderType string

const (
	OrderTypeMarket    OrderType = "market"
	OrderTypeLimit     OrderType = "limit"
	OrderTypeStopLimit OrderType = "stop_limit"
	OrderTypeIceberg   OrderType = "iceberg"
)

// OrderSide represents order side
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderStatus represents order status
type OrderStatus string

const (
	OrderStatusNew       OrderStatus = "new"
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusFilled    OrderStatus = "filled"
	OrderStatusCancelled OrderStatus = "cancelled"
	OrderStatusRejected  OrderStatus = "rejected"
)

// ExchangeOrder represents an order on an exchange
type ExchangeOrder struct {
	ID            string      `json:"id"`
	ExchangeID    string      `json:"exchange_id"`
	Symbol        string      `json:"symbol"`
	Side          OrderSide   `json:"side"`
	Type          OrderType   `json:"type"`
	Quantity      float64     `json:"quantity"`
	Price         float64     `json:"price"`
	StopPrice     float64     `json:"stop_price,omitempty"`
	Status        OrderStatus `json:"status"`
	FilledQty     float64     `json:"filled_qty"`
	RemainingQty  float64     `json:"remaining_qty"`
	AvgFillPrice  float64     `json:"avg_fill_price"`
	CreatedAt     time.Time   `json:"created_at"`
	UpdatedAt     time.Time   `json:"updated_at"`
	TimeInForce   string      `json:"time_in_force"`
	ClientOrderID string      `json:"client_order_id"`
}

// MarketData represents market data from an exchange
type MarketData struct {
	Symbol    string    `json:"symbol"`
	BidPrice  float64   `json:"bid_price"`
	AskPrice  float64   `json:"ask_price"`
	BidSize   float64   `json:"bid_size"`
	AskSize   float64   `json:"ask_size"`
	LastPrice float64   `json:"last_price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Exchange  string    `json:"exchange"`
}

// Trade represents a trade execution
type Trade struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	Symbol    string    `json:"symbol"`
	Side      OrderSide `json:"side"`
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Fee       float64   `json:"fee"`
	Timestamp time.Time `json:"timestamp"`
	Exchange  string    `json:"exchange"`
}

// Balance represents account balance
type Balance struct {
	Asset    string  `json:"asset"`
	Free     float64 `json:"free"`
	Locked   float64 `json:"locked"`
	Total    float64 `json:"total"`
	Exchange string  `json:"exchange"`
}

// ExchangeInfo represents exchange information
type ExchangeInfo struct {
	Name           string             `json:"name"`
	Status         string             `json:"status"`
	TradingFees    map[string]float64 `json:"trading_fees"`
	WithdrawalFees map[string]float64 `json:"withdrawal_fees"`
	MinOrderSizes  map[string]float64 `json:"min_order_sizes"`
	MaxOrderSizes  map[string]float64 `json:"max_order_sizes"`
	SupportedPairs []string           `json:"supported_pairs"`
	RateLimits     map[string]int     `json:"rate_limits"`
	LastUpdate     time.Time          `json:"last_update"`
}

// ExchangeAdapter defines the interface for exchange adapters
type ExchangeAdapter interface {
	// Connection management
	Connect(ctx context.Context) error
	Disconnect() error
	IsConnected() bool
	GetExchangeInfo() *ExchangeInfo

	// Order management
	PlaceOrder(ctx context.Context, order *ExchangeOrder) (*ExchangeOrder, error)
	CancelOrder(ctx context.Context, orderID string, symbol string) error
	GetOrder(ctx context.Context, orderID string, symbol string) (*ExchangeOrder, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*ExchangeOrder, error)
	GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*ExchangeOrder, error)

	// Market data
	GetMarketData(ctx context.Context, symbol string) (*MarketData, error)
	SubscribeMarketData(ctx context.Context, symbols []string, callback func(*MarketData)) error
	UnsubscribeMarketData(ctx context.Context, symbols []string) error

	// Account management
	GetBalances(ctx context.Context) ([]*Balance, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)

	// WebSocket streams
	StartWebSocket(ctx context.Context) error
	StopWebSocket() error
}

// BaseAdapter provides common functionality for exchange adapters
type BaseAdapter struct {
	name         string
	connected    bool
	rateLimiter  *RateLimiter
	mutex        sync.RWMutex
	exchangeInfo *ExchangeInfo
	lastPing     time.Time
}

// NewBaseAdapter creates a new base adapter
func NewBaseAdapter(name string, rateLimit int) *BaseAdapter {
	return &BaseAdapter{
		name:        name,
		rateLimiter: NewRateLimiter(rateLimit),
		exchangeInfo: &ExchangeInfo{
			Name:           name,
			Status:         "disconnected",
			TradingFees:    make(map[string]float64),
			WithdrawalFees: make(map[string]float64),
			MinOrderSizes:  make(map[string]float64),
			MaxOrderSizes:  make(map[string]float64),
			RateLimits:     make(map[string]int),
			LastUpdate:     time.Now(),
		},
	}
}

// IsConnected returns connection status
func (ba *BaseAdapter) IsConnected() bool {
	ba.mutex.RLock()
	defer ba.mutex.RUnlock()
	return ba.connected
}

// SetConnected sets connection status
func (ba *BaseAdapter) SetConnected(connected bool) {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()
	ba.connected = connected
	if connected {
		ba.exchangeInfo.Status = "connected"
	} else {
		ba.exchangeInfo.Status = "disconnected"
	}
}

// GetExchangeInfo returns exchange information
func (ba *BaseAdapter) GetExchangeInfo() *ExchangeInfo {
	ba.mutex.RLock()
	defer ba.mutex.RUnlock()

	// Return a copy to avoid race conditions
	info := *ba.exchangeInfo
	return &info
}

// UpdateExchangeInfo updates exchange information
func (ba *BaseAdapter) UpdateExchangeInfo(info *ExchangeInfo) {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()
	ba.exchangeInfo = info
	ba.exchangeInfo.LastUpdate = time.Now()
}

// CheckRateLimit checks if request is within rate limits
func (ba *BaseAdapter) CheckRateLimit() error {
	return ba.rateLimiter.Wait()
}

// ValidateOrder validates an order before submission
func (ba *BaseAdapter) ValidateOrder(order *ExchangeOrder) error {
	if order.Symbol == "" {
		return fmt.Errorf("symbol is required")
	}

	if order.Quantity <= 0 {
		return fmt.Errorf("quantity must be positive")
	}

	if order.Type == OrderTypeLimit && order.Price <= 0 {
		return fmt.Errorf("price must be positive for limit orders")
	}

	if order.Type == OrderTypeStopLimit && (order.Price <= 0 || order.StopPrice <= 0) {
		return fmt.Errorf("price and stop price must be positive for stop-limit orders")
	}

	// Check minimum order size
	ba.mutex.RLock()
	minSize, exists := ba.exchangeInfo.MinOrderSizes[order.Symbol]
	ba.mutex.RUnlock()

	if exists && order.Quantity < minSize {
		return fmt.Errorf("order quantity %f is below minimum %f for %s",
			order.Quantity, minSize, order.Symbol)
	}

	// Check maximum order size
	ba.mutex.RLock()
	maxSize, exists := ba.exchangeInfo.MaxOrderSizes[order.Symbol]
	ba.mutex.RUnlock()

	if exists && order.Quantity > maxSize {
		return fmt.Errorf("order quantity %f exceeds maximum %f for %s",
			order.Quantity, maxSize, order.Symbol)
	}

	return nil
}

// NormalizeSymbol normalizes symbol format for the exchange
func (ba *BaseAdapter) NormalizeSymbol(symbol string) string {
	// Default implementation - can be overridden by specific adapters
	return symbol
}

// GetName returns the adapter name
func (ba *BaseAdapter) GetName() string {
	return ba.name
}

// UpdateLastPing updates the last ping timestamp
func (ba *BaseAdapter) UpdateLastPing() {
	ba.mutex.Lock()
	defer ba.mutex.Unlock()
	ba.lastPing = time.Now()
}

// GetLastPing returns the last ping timestamp
func (ba *BaseAdapter) GetLastPing() time.Time {
	ba.mutex.RLock()
	defer ba.mutex.RUnlock()
	return ba.lastPing
}

// RateLimiter implements a simple rate limiter
type RateLimiter struct {
	requests chan struct{}
	ticker   *time.Ticker
	done     chan struct{}
}

// NewRateLimiter creates a new rate limiter
func NewRateLimiter(requestsPerSecond int) *RateLimiter {
	rl := &RateLimiter{
		requests: make(chan struct{}, requestsPerSecond),
		ticker:   time.NewTicker(time.Second / time.Duration(requestsPerSecond)),
		done:     make(chan struct{}),
	}

	// Fill initial capacity
	for i := 0; i < requestsPerSecond; i++ {
		rl.requests <- struct{}{}
	}

	// Start refill goroutine
	go rl.refill()

	return rl
}

// Wait waits for rate limit availability
func (rl *RateLimiter) Wait() error {
	select {
	case <-rl.requests:
		return nil
	case <-time.After(time.Second):
		return fmt.Errorf("rate limit exceeded")
	}
}

// refill refills the rate limiter
func (rl *RateLimiter) refill() {
	for {
		select {
		case <-rl.ticker.C:
			select {
			case rl.requests <- struct{}{}:
			default:
				// Channel is full, skip
			}
		case <-rl.done:
			return
		}
	}
}

// Stop stops the rate limiter
func (rl *RateLimiter) Stop() {
	close(rl.done)
	rl.ticker.Stop()
}

// OrderBuilder helps build orders with validation
type OrderBuilder struct {
	order *ExchangeOrder
}

// NewOrderBuilder creates a new order builder
func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		order: &ExchangeOrder{
			CreatedAt:   time.Now(),
			Status:      OrderStatusNew,
			TimeInForce: "GTC", // Good Till Cancelled
		},
	}
}

// Symbol sets the symbol
func (ob *OrderBuilder) Symbol(symbol string) *OrderBuilder {
	ob.order.Symbol = symbol
	return ob
}

// Side sets the order side
func (ob *OrderBuilder) Side(side OrderSide) *OrderBuilder {
	ob.order.Side = side
	return ob
}

// Type sets the order type
func (ob *OrderBuilder) Type(orderType OrderType) *OrderBuilder {
	ob.order.Type = orderType
	return ob
}

// Quantity sets the quantity
func (ob *OrderBuilder) Quantity(quantity float64) *OrderBuilder {
	ob.order.Quantity = quantity
	ob.order.RemainingQty = quantity
	return ob
}

// Price sets the price
func (ob *OrderBuilder) Price(price float64) *OrderBuilder {
	ob.order.Price = price
	return ob
}

// StopPrice sets the stop price
func (ob *OrderBuilder) StopPrice(stopPrice float64) *OrderBuilder {
	ob.order.StopPrice = stopPrice
	return ob
}

// ClientOrderID sets the client order ID
func (ob *OrderBuilder) ClientOrderID(clientOrderID string) *OrderBuilder {
	ob.order.ClientOrderID = clientOrderID
	return ob
}

// TimeInForce sets the time in force
func (ob *OrderBuilder) TimeInForce(tif string) *OrderBuilder {
	ob.order.TimeInForce = tif
	return ob
}

// Build builds and returns the order
func (ob *OrderBuilder) Build() *ExchangeOrder {
	return ob.order
}
