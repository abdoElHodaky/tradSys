package pool

import (
	"sync"
)

// ObjectPool provides a generic object pool
type ObjectPool struct {
	pool sync.Pool
}

// NewObjectPool creates a new object pool
func NewObjectPool(newFunc func() interface{}) *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: newFunc,
		},
	}
}

// Get retrieves an object from the pool
func (p *ObjectPool) Get() interface{} {
	return p.pool.Get()
}

// Put returns an object to the pool
func (p *ObjectPool) Put(obj interface{}) {
	p.pool.Put(obj)
}

// FastOrderPool provides a pool for fast order objects to reduce allocations
type FastOrderPool struct {
	pool sync.Pool
}

// NewFastOrderPool creates a new fast order pool
func NewFastOrderPool() *FastOrderPool {
	return &FastOrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &FastOrder{
					Order: &Order{},
				}
			},
		},
	}
}

// Get retrieves a fast order from the pool
func (p *FastOrderPool) Get() *FastOrder {
	return p.pool.Get().(*FastOrder)
}

// Put returns a fast order to the pool
func (p *FastOrderPool) Put(order *FastOrder) {
	// Reset the order before putting it back
	order.Reset()
	p.pool.Put(order)
}

// TradePool provides a pool for trade objects to reduce allocations
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

// Get retrieves a trade from the pool
func (p *TradePool) Get() *Trade {
	return p.pool.Get().(*Trade)
}

// Put returns a trade to the pool
func (p *TradePool) Put(trade *Trade) {
	// Reset the trade before putting it back
	trade.Reset()
	p.pool.Put(trade)
}

// Order represents a trading order
type Order struct {
	ID             string
	Symbol         string
	Side           string
	Price          float64
	Quantity       float64
	FilledQuantity float64
	Type           string
}

// Reset resets the order fields
func (o *Order) Reset() {
	o.ID = ""
	o.Symbol = ""
	o.Side = ""
	o.Price = 0
	o.Quantity = 0
	o.FilledQuantity = 0
	o.Type = ""
}

// FastOrder represents a high-performance order with additional fields
type FastOrder struct {
	Order           *Order
	PriceInt64      int64
	QuantityInt64   int64
	CreatedAtNano   int64
	UpdatedAtNano   int64
}

// Reset resets the fast order fields
func (f *FastOrder) Reset() {
	if f.Order != nil {
		f.Order.Reset()
	}
	f.PriceInt64 = 0
	f.QuantityInt64 = 0
	f.CreatedAtNano = 0
	f.UpdatedAtNano = 0
}

// Trade represents a trade execution
type Trade struct {
	ID           string
	Symbol       string
	Price        float64
	Quantity     float64
	BuyOrderID   string
	SellOrderID  string
	Timestamp    int64
}

// Reset resets the trade fields
func (t *Trade) Reset() {
	t.ID = ""
	t.Symbol = ""
	t.Price = 0
	t.Quantity = 0
	t.BuyOrderID = ""
	t.SellOrderID = ""
	t.Timestamp = 0
}

// WebSocketMessage represents a websocket message
type WebSocketMessage struct {
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Symbol  string      `json:"symbol,omitempty"`
	Channel string      `json:"channel,omitempty"`
}

// Reset resets the websocket message fields
func (w *WebSocketMessage) Reset() {
	w.Type = ""
	w.Data = nil
	w.Symbol = ""
	w.Channel = ""
}

// WebSocketMessagePool provides a pool for websocket messages
type WebSocketMessagePool struct {
	pool sync.Pool
}

// NewWebSocketMessagePool creates a new websocket message pool
func NewWebSocketMessagePool() *WebSocketMessagePool {
	return &WebSocketMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &WebSocketMessage{}
			},
		},
	}
}

// Get retrieves a websocket message from the pool
func (p *WebSocketMessagePool) Get() *WebSocketMessage {
	return p.pool.Get().(*WebSocketMessage)
}

// Put returns a websocket message to the pool
func (p *WebSocketMessagePool) Put(msg *WebSocketMessage) {
	msg.Reset()
	p.pool.Put(msg)
}

// PriceMessage represents a price update message
type PriceMessage struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
}

// Reset resets the price message fields
func (p *PriceMessage) Reset() {
	p.Symbol = ""
	p.Price = 0
	p.Volume = 0
	p.Timestamp = 0
}

// PriceMessagePool provides a pool for price messages
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

// Get retrieves a price message from the pool
func (p *PriceMessagePool) Get() *PriceMessage {
	return p.pool.Get().(*PriceMessage)
}

// Put returns a price message to the pool
func (p *PriceMessagePool) Put(msg *PriceMessage) {
	msg.Reset()
	p.pool.Put(msg)
}

// OrderUpdateMessage represents an order update message
type OrderUpdateMessage struct {
	OrderID   string  `json:"order_id"`
	Symbol    string  `json:"symbol"`
	Status    string  `json:"status"`
	Price     float64 `json:"price"`
	Quantity  float64 `json:"quantity"`
	Timestamp int64   `json:"timestamp"`
}

// Reset resets the order update message fields
func (o *OrderUpdateMessage) Reset() {
	o.OrderID = ""
	o.Symbol = ""
	o.Status = ""
	o.Price = 0
	o.Quantity = 0
	o.Timestamp = 0
}

// OrderUpdateMessagePool provides a pool for order update messages
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

// Get retrieves an order update message from the pool
func (p *OrderUpdateMessagePool) Get() *OrderUpdateMessage {
	return p.pool.Get().(*OrderUpdateMessage)
}

// Put returns an order update message to the pool
func (p *OrderUpdateMessagePool) Put(msg *OrderUpdateMessage) {
	msg.Reset()
	p.pool.Put(msg)
}



// MatchResult represents the result of a matching operation
type MatchResult struct {
	Trades    []*Trade
	Timestamp int64
	Success   bool
	Error     error
}

// Reset resets the match result fields
func (m *MatchResult) Reset() {
	m.Trades = m.Trades[:0] // Keep underlying array, reset length
	m.Timestamp = 0
	m.Success = false
	m.Error = nil
}
