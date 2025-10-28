package performance

import (
	"sync"
	"time"
)

// MarketDataPool provides high-performance pooling for market data objects
type MarketDataPool struct {
	pool sync.Pool
}

// NewMarketDataPool creates a new high-performance market data pool
func NewMarketDataPool() *MarketDataPool {
	return &MarketDataPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &MarketData{}
			},
		},
	}
}

// Get retrieves a market data object from the pool
func (p *MarketDataPool) Get() *MarketData {
	return p.pool.Get().(*MarketData)
}

// Put returns a market data object to the pool
func (p *MarketDataPool) Put(md *MarketData) {
	md.Reset()
	p.pool.Put(md)
}

// OrderPool provides high-performance pooling for order objects
type OrderPool struct {
	pool sync.Pool
}

// NewOrderPool creates a new high-performance order pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Order{}
			},
		},
	}
}

// Get retrieves an order object from the pool
func (p *OrderPool) Get() *Order {
	return p.pool.Get().(*Order)
}

// Put returns an order object to the pool
func (p *OrderPool) Put(order *Order) {
	order.Reset()
	p.pool.Put(order)
}

// MarketData represents high-performance market data
type MarketData struct {
	Symbol      string
	Price       float64
	Volume      float64
	Bid         float64
	Ask         float64
	BidSize     float64
	AskSize     float64
	Timestamp   time.Time
	SequenceNum uint64
}

// Reset resets the market data object for reuse
func (md *MarketData) Reset() {
	md.Symbol = ""
	md.Price = 0
	md.Volume = 0
	md.Bid = 0
	md.Ask = 0
	md.BidSize = 0
	md.AskSize = 0
	md.Timestamp = time.Time{}
	md.SequenceNum = 0
}

// Order represents a high-performance order
type Order struct {
	ID          string
	Symbol      string
	Side        string
	OrderType   string
	Price       float64
	Quantity    float64
	FilledQty   float64
	Status      string
	Timestamp   time.Time
	SequenceNum uint64
}

// Reset resets the order object for reuse
func (o *Order) Reset() {
	o.ID = ""
	o.Symbol = ""
	o.Side = ""
	o.OrderType = ""
	o.Price = 0
	o.Quantity = 0
	o.FilledQty = 0
	o.Status = ""
	o.Timestamp = time.Time{}
	o.SequenceNum = 0
}

// FastOrderPool provides ultra-high-performance pooling for order objects
type FastOrderPool struct {
	pool sync.Pool
}

// NewFastOrderPool creates a new fast order pool
func NewFastOrderPool() *FastOrderPool {
	return &FastOrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Order{}
			},
		},
	}
}

// Get retrieves an order object from the pool
func (p *FastOrderPool) Get() *Order {
	return p.pool.Get().(*Order)
}

// Put returns an order object to the pool
func (p *FastOrderPool) Put(order *Order) {
	order.Reset()
	p.pool.Put(order)
}

// TradePool provides high-performance pooling for trade objects
type TradePool struct {
	pool sync.Pool
}

// NewTradePool creates a new trade pool
func NewTradePool() *TradePool {
	return &TradePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Trade{}
			},
		},
	}
}

// Get retrieves a trade object from the pool
func (p *TradePool) Get() *Trade {
	return p.pool.Get().(*Trade)
}

// Put returns a trade object to the pool
func (p *TradePool) Put(trade *Trade) {
	trade.Reset()
	p.pool.Put(trade)
}

// Trade represents a high-performance trade
type Trade struct {
	ID          string
	Symbol      string
	Price       float64
	Quantity    float64
	BuyOrderID  string
	SellOrderID string
	Timestamp   time.Time
	SequenceNum uint64
}

// Reset resets the trade object for reuse
func (t *Trade) Reset() {
	t.ID = ""
	t.Symbol = ""
	t.Price = 0
	t.Quantity = 0
	t.BuyOrderID = ""
	t.SellOrderID = ""
	t.Timestamp = time.Time{}
	t.SequenceNum = 0
}

// WebSocketMessage represents a high-performance WebSocket message
type WebSocketMessage struct {
	Type      int
	Symbol    string
	Data      []byte
	Timestamp time.Time
}

// Reset resets the WebSocket message object for reuse
func (w *WebSocketMessage) Reset() {
	w.Type = 0
	w.Symbol = ""
	w.Data = w.Data[:0] // Keep capacity but reset length
	w.Timestamp = time.Time{}
}

// WebSocketMessagePool provides high-performance pooling for WebSocket messages
type WebSocketMessagePool struct {
	pool sync.Pool
}

// NewWebSocketMessagePool creates a new WebSocket message pool
func NewWebSocketMessagePool() *WebSocketMessagePool {
	return &WebSocketMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &WebSocketMessage{
					Data: make([]byte, 0, 1024), // Pre-allocate with capacity
				}
			},
		},
	}
}

// Get retrieves a WebSocket message object from the pool
func (p *WebSocketMessagePool) Get() *WebSocketMessage {
	return p.pool.Get().(*WebSocketMessage)
}

// Put returns a WebSocket message object to the pool
func (p *WebSocketMessagePool) Put(msg *WebSocketMessage) {
	msg.Reset()
	p.pool.Put(msg)
}

// PriceMessage represents a high-performance price message
type PriceMessage struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
}

// Reset resets the price message object for reuse
func (p *PriceMessage) Reset() {
	p.Symbol = ""
	p.Price = 0
	p.Volume = 0
	p.Timestamp = time.Time{}
}

// PriceMessagePool provides high-performance pooling for price messages
type PriceMessagePool struct {
	pool sync.Pool
}

// NewPriceMessagePool creates a new price message pool
func NewPriceMessagePool() *PriceMessagePool {
	return &PriceMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &PriceMessage{}
			},
		},
	}
}

// Get retrieves a price message object from the pool
func (p *PriceMessagePool) Get() *PriceMessage {
	return p.pool.Get().(*PriceMessage)
}

// Put returns a price message object to the pool
func (p *PriceMessagePool) Put(msg *PriceMessage) {
	msg.Reset()
	p.pool.Put(msg)
}

// OrderUpdateMessage represents a high-performance order update message
type OrderUpdateMessage struct {
	OrderID   string
	Symbol    string
	Status    string
	Price     float64
	Quantity  float64
	FilledQty float64
	Timestamp time.Time
}

// Reset resets the order update message object for reuse
func (o *OrderUpdateMessage) Reset() {
	o.OrderID = ""
	o.Symbol = ""
	o.Status = ""
	o.Price = 0
	o.Quantity = 0
	o.FilledQty = 0
	o.Timestamp = time.Time{}
}

// OrderUpdateMessagePool provides high-performance pooling for order update messages
type OrderUpdateMessagePool struct {
	pool sync.Pool
}

// NewOrderUpdateMessagePool creates a new order update message pool
func NewOrderUpdateMessagePool() *OrderUpdateMessagePool {
	return &OrderUpdateMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderUpdateMessage{}
			},
		},
	}
}

// Get retrieves an order update message object from the pool
func (p *OrderUpdateMessagePool) Get() *OrderUpdateMessage {
	return p.pool.Get().(*OrderUpdateMessage)
}

// Put returns an order update message object to the pool
func (p *OrderUpdateMessagePool) Put(msg *OrderUpdateMessage) {
	msg.Reset()
	p.pool.Put(msg)
}

// BufferPool provides high-performance pooling for byte buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf []byte) {
	buf = buf[:0] // Reset length but keep capacity
	p.pool.Put(buf)
}

// OrderRequest represents a high-performance order request
type OrderRequest struct {
	Symbol   string
	Side     string
	Type     string
	Price    float64
	Quantity float64
	UserID   string
}

// Reset resets the order request object for reuse
func (o *OrderRequest) Reset() {
	o.Symbol = ""
	o.Side = ""
	o.Type = ""
	o.Price = 0
	o.Quantity = 0
	o.UserID = ""
}

// OrderResponse represents a high-performance order response
type OrderResponse struct {
	OrderID   string
	Status    string
	Message   string
	Timestamp time.Time
}

// Reset resets the order response object for reuse
func (o *OrderResponse) Reset() {
	o.OrderID = ""
	o.Status = ""
	o.Message = ""
	o.Timestamp = time.Time{}
}

// OrderRequestPool provides high-performance pooling for order requests
type OrderRequestPool struct {
	pool sync.Pool
}

// NewOrderRequestPool creates a new order request pool
func NewOrderRequestPool() *OrderRequestPool {
	return &OrderRequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderRequest{}
			},
		},
	}
}

// Get retrieves an order request object from the pool
func (p *OrderRequestPool) Get() *OrderRequest {
	return p.pool.Get().(*OrderRequest)
}

// Put returns an order request object to the pool
func (p *OrderRequestPool) Put(req *OrderRequest) {
	req.Reset()
	p.pool.Put(req)
}

// OrderResponsePool provides high-performance pooling for order responses
type OrderResponsePool struct {
	pool sync.Pool
}

// NewOrderResponsePool creates a new order response pool
func NewOrderResponsePool() *OrderResponsePool {
	return &OrderResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderResponse{}
			},
		},
	}
}

// Get retrieves an order response object from the pool
func (p *OrderResponsePool) Get() *OrderResponse {
	return p.pool.Get().(*OrderResponse)
}

// Put returns an order response object to the pool
func (p *OrderResponsePool) Put(resp *OrderResponse) {
	resp.Reset()
	p.pool.Put(resp)
}

// Global pool instances for convenience
var (
	orderRequestPool  = NewOrderRequestPool()
	orderResponsePool = NewOrderResponsePool()
	orderPool         = NewOrderPool()
)

// Convenience functions for global pools
func GetOrderRequestFromPool() *OrderRequest {
	return orderRequestPool.Get()
}

func PutOrderRequestToPool(req *OrderRequest) {
	orderRequestPool.Put(req)
}

func GetOrderResponseFromPool() *OrderResponse {
	return orderResponsePool.Get()
}

func PutOrderResponseToPool(resp *OrderResponse) {
	orderResponsePool.Put(resp)
}

func GetOrderFromPool() *Order {
	return orderPool.Get()
}

func PutOrderToPool(order *Order) {
	orderPool.Put(order)
}
