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
