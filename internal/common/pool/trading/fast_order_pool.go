package pools

import (
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
)

// FastOrder represents an optimized order structure for HFT
type FastOrder struct {
	types.Order

	// Pre-allocated fields for performance
	PriceInt64    int64 // Price as int64 for faster comparison
	QuantityInt64 int64 // Quantity as int64 for faster arithmetic
	CreatedAtNano int64 // Created time as nanoseconds
	UpdatedAtNano int64 // Updated time as nanoseconds

	// Memory pool index for recycling
	PoolIndex int32
}

// Reset resets the FastOrder struct for object pooling
func (fo *FastOrder) Reset() {
	fo.Order.Reset()
	fo.PriceInt64 = 0
	fo.QuantityInt64 = 0
	fo.CreatedAtNano = 0
	fo.UpdatedAtNano = 0
	fo.PoolIndex = 0
}

// FastOrderPool provides object pooling for FastOrder structs to reduce allocations
type FastOrderPool struct {
	pool sync.Pool
}

// NewFastOrderPool creates a new FastOrderPool
func NewFastOrderPool() *FastOrderPool {
	return &FastOrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &FastOrder{}
			},
		},
	}
}

// Get retrieves a FastOrder from the pool
func (p *FastOrderPool) Get() *FastOrder {
	return p.pool.Get().(*FastOrder)
}

// Put returns a FastOrder to the pool after resetting it
func (p *FastOrderPool) Put(order *FastOrder) {
	// Reset the order to avoid memory leaks
	order.Reset()
	p.pool.Put(order)
}
