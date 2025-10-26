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

// FastOrderPool provides a pool for order objects to reduce allocations
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

// Get retrieves an order from the pool
func (p *FastOrderPool) Get() *Order {
	return p.pool.Get().(*Order)
}

// Put returns an order to the pool
func (p *FastOrderPool) Put(order *Order) {
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
	ID       string
	Symbol   string
	Side     string
	Price    float64
	Quantity float64
	Type     string
}

// Reset resets the order fields
func (o *Order) Reset() {
	o.ID = ""
	o.Symbol = ""
	o.Side = ""
	o.Price = 0
	o.Quantity = 0
	o.Type = ""
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
