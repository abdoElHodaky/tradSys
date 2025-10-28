package pool

import (
	"sync"
)

// ObjectPool provides a generic object pool implementation
type ObjectPool struct {
	pool sync.Pool
	new  func() interface{}
}

// NewObjectPool creates a new object pool
func NewObjectPool(newFunc func() interface{}) *ObjectPool {
	return &ObjectPool{
		pool: sync.Pool{
			New: newFunc,
		},
		new: newFunc,
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

// MarketDataPool provides pooling for market data objects
type MarketDataPool struct {
	pool *ObjectPool
}

// NewMarketDataPool creates a new market data pool
func NewMarketDataPool() *MarketDataPool {
	return &MarketDataPool{
		pool: NewObjectPool(func() interface{} {
			return &MarketData{}
		}),
	}
}

// Get retrieves a market data object from the pool
func (p *MarketDataPool) Get() *MarketData {
	return p.pool.Get().(*MarketData)
}

// Put returns a market data object to the pool
func (p *MarketDataPool) Put(md *MarketData) {
	// Reset the object before returning to pool
	md.Reset()
	p.pool.Put(md)
}

// OrderPool provides pooling for order objects
type OrderPool struct {
	pool *ObjectPool
}

// NewOrderPool creates a new order pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: NewObjectPool(func() interface{} {
			return &Order{}
		}),
	}
}

// Get retrieves an order object from the pool
func (p *OrderPool) Get() *Order {
	return p.pool.Get().(*Order)
}

// Put returns an order object to the pool
func (p *OrderPool) Put(order *Order) {
	// Reset the object before returning to pool
	order.Reset()
	p.pool.Put(order)
}

// MarketData represents market data for pooling
type MarketData struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp int64
}

// Reset resets the market data object
func (md *MarketData) Reset() {
	md.Symbol = ""
	md.Price = 0
	md.Volume = 0
	md.Timestamp = 0
}

// Order represents an order for pooling
type Order struct {
	ID        string
	Symbol    string
	Side      string
	Price     float64
	Quantity  float64
	Timestamp int64
}

// Reset resets the order object
func (o *Order) Reset() {
	o.ID = ""
	o.Symbol = ""
	o.Side = ""
	o.Price = 0
	o.Quantity = 0
	o.Timestamp = 0
}

