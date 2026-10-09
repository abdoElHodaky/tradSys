// Package licensing provides pricing types for TradSys v3
package licensing

import (
	"sync"
	"time"
)

// Pricing-related types extracted from unified_asset_system.go

// PricingInfo contains pricing and market data information
type PricingInfo struct {
	CurrentPrice     float64
	PreviousClose    float64
	DayChange        float64
	DayChangePercent float64
	Volume           int64
	MarketCap        float64
	PE               float64
	DividendYield    float64
	LastUpdated      time.Time
}

// PricingModel defines the interface for pricing models
type PricingModel interface {
	Name() string
	Calculate(price float64, volume float64) float64
}

// DataAggregator aggregates market data
type DataAggregator struct {
	history map[string][]float64
	mu      sync.RWMutex
}

// NewDataAggregator creates a new data aggregator
func NewDataAggregator() *DataAggregator {
	return &DataAggregator{
		history: make(map[string][]float64),
	}
}

// PriceCache caches price data
type PriceCache struct {
	cache     map[string]float64
	lastPrice map[string]time.Time
	mu        sync.RWMutex
}

// NewPriceCache creates a new price cache
func NewPriceCache() *PriceCache {
	return &PriceCache{
		cache:     make(map[string]float64),
		lastPrice: make(map[string]time.Time),
	}
}

// Set stores a price in the cache
func (pc *PriceCache) Set(symbol string, price float64) {
	pc.mu.Lock()
	defer pc.mu.Unlock()
	pc.cache[symbol] = price
	pc.lastPrice[symbol] = time.Now()
}

// Get retrieves a price from the cache
func (pc *PriceCache) Get(symbol string) (float64, bool) {
	pc.mu.RLock()
	defer pc.mu.RUnlock()
	price, ok := pc.cache[symbol]
	return price, ok
}

// RealTimeFeed represents a real-time market data feed
type RealTimeFeed struct {
	symbol   string
	ticker   chan float64
	isActive bool
	mu       sync.RWMutex
}

// NewRealTimeFeed creates a new real-time feed
func NewRealTimeFeed(symbol string) *RealTimeFeed {
	return &RealTimeFeed{
		symbol: symbol,
		ticker: make(chan float64, 100),
	}
}
