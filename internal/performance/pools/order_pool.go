package pools

import (
	"sync"

	"github.com/abdoElHodaky/tradSys/proto/orders"
)

// OrderPool provides a pool of OrderResponse objects
// to reduce garbage collection pressure in high-frequency scenarios
type OrderPool struct {
	pool sync.Pool
}

// NewOrderPool creates a new order object pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &orders.OrderResponse{}
			},
		},
	}
}

// Get retrieves an OrderResponse from the pool
func (p *OrderPool) Get() *orders.OrderResponse {
	return p.pool.Get().(*orders.OrderResponse)
}

// Put returns an OrderResponse to the pool after resetting its fields
func (p *OrderPool) Put(order *orders.OrderResponse) {
	// Reset fields to zero values to prevent data leakage
	order.OrderId = ""
	order.Symbol = ""
	order.Side = ""
	order.Type = ""
	order.Quantity = 0
	order.Price = 0
	order.Status = ""
	order.Timestamp = 0
	order.FilledQuantity = 0
	order.AveragePrice = 0
	order.Strategy = ""
	// Add any other fields that need to be reset

	p.pool.Put(order)
}

