package pools

import (
	"sync"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
)

// OrderPool provides object pooling for Order structs to reduce allocations
type OrderPool struct {
	pool sync.Pool
}

// NewOrderPool creates a new OrderPool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &types.Order{}
			},
		},
	}
}

// Get retrieves an Order from the pool
func (p *OrderPool) Get() *types.Order {
	return p.pool.Get().(*types.Order)
}

// Put returns an Order to the pool after resetting it
func (p *OrderPool) Put(order *types.Order) {
	// Reset the order to avoid memory leaks
	order.Reset()
	p.pool.Put(order)
}

// Reset method for Order struct (to be added to order_matching.Order)
// This should be implemented in the Order struct itself
