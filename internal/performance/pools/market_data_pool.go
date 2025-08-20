package pools

import (
	"sync"

	"github.com/abdoElHodaky/tradSys/proto/marketdata"
)

// MarketDataPool provides a pool of market data responses
type MarketDataPool struct {
	pool sync.Pool
}

// NewMarketDataPool creates a new market data pool
func NewMarketDataPool() *MarketDataPool {
	return &MarketDataPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &marketdata.MarketDataResponse{}
			},
		},
	}
}

// Get gets a market data response from the pool
func (p *MarketDataPool) Get() *marketdata.MarketDataResponse {
	return p.pool.Get().(*marketdata.MarketDataResponse)
}

// Put puts a market data response back into the pool
func (p *MarketDataPool) Put(response *marketdata.MarketDataResponse) {
	// Reset the response
	response.Symbol = ""
	response.Price = 0
	response.Volume = 0
	response.Bid = 0
	response.Ask = 0
	response.High = 0
	response.Low = 0
	response.Open = 0
	response.Close = 0
	response.Timestamp = 0
	response.Interval = ""

	// Put the response back into the pool
	p.pool.Put(response)
}

// NewMarketDataResponse creates a new market data response
func (p *MarketDataPool) NewMarketDataResponse(
	symbol string,
	price float64,
	volume float64,
	bid float64,
	ask float64,
	high float64,
	low float64,
	open float64,
	close float64,
	timestamp int64,
	interval string,
) *marketdata.MarketDataResponse {
	response := p.Get()
	response.Symbol = symbol
	response.Price = price
	response.Volume = volume
	response.Bid = bid
	response.Ask = ask
	response.High = high
	response.Low = low
	response.Open = open
	response.Close = close
	response.Timestamp = timestamp
	response.Interval = interval
	return response
}

