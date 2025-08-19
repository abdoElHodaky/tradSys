package pools

import (
	"sync"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
)

// MarketDataPool provides a pool of MarketDataResponse objects
// to reduce garbage collection pressure in high-frequency scenarios
type MarketDataPool struct {
	pool sync.Pool
}

// NewMarketDataPool creates a new market data object pool
func NewMarketDataPool() *MarketDataPool {
	return &MarketDataPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &marketdata.MarketDataResponse{}
			},
		},
	}
}

// Get retrieves a MarketDataResponse from the pool
func (p *MarketDataPool) Get() *marketdata.MarketDataResponse {
	return p.pool.Get().(*marketdata.MarketDataResponse)
}

// Put returns a MarketDataResponse to the pool after resetting its fields
func (p *MarketDataPool) Put(data *marketdata.MarketDataResponse) {
	// Reset fields to zero values to prevent data leakage
	data.Symbol = ""
	data.Price = 0
	data.Volume = 0
	data.Timestamp = 0
	data.Exchange = ""
	data.Bid = 0
	data.Ask = 0
	data.BidSize = 0
	data.AskSize = 0
	// Add any other fields that need to be reset

	p.pool.Put(data)
}

