package price_levels

import (
	"container/heap"
	"sync"
	"time"
)

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64
	Quantity float64
	Orders   int
	Time     time.Time
}

// OrderBook represents a complete order book for a symbol
type OrderBook struct {
	Symbol    string
	Bids      *PriceLevelHeap
	Asks      *PriceLevelHeap
	LastPrice float64
	Volume    float64
	mutex     sync.RWMutex
}

// PriceLevelHeap implements heap.Interface for price levels
type PriceLevelHeap struct {
	levels []PriceLevel
	isBid  bool // true for bids (max heap), false for asks (min heap)
}

func (h PriceLevelHeap) Len() int { return len(h.levels) }

func (h PriceLevelHeap) Less(i, j int) bool {
	if h.isBid {
		return h.levels[i].Price > h.levels[j].Price // Max heap for bids
	}
	return h.levels[i].Price < h.levels[j].Price // Min heap for asks
}

func (h PriceLevelHeap) Swap(i, j int) {
	h.levels[i], h.levels[j] = h.levels[j], h.levels[i]
}

func (h *PriceLevelHeap) Push(x interface{}) {
	h.levels = append(h.levels, x.(PriceLevel))
}

func (h *PriceLevelHeap) Pop() interface{} {
	old := h.levels
	n := len(old)
	item := old[n-1]
	h.levels = old[0 : n-1]
	return item
}

// PriceLevelManager manages price levels and order books
type PriceLevelManager struct {
	orderBooks map[string]*OrderBook
	mutex      sync.RWMutex
	metrics    map[string]interface{}
}

// NewPriceLevelManager creates a new price level manager
func NewPriceLevelManager() *PriceLevelManager {
	return &PriceLevelManager{
		orderBooks: make(map[string]*OrderBook),
		metrics:    make(map[string]interface{}),
	}
}

// GetOrderBook returns the order book for a symbol
func (plm *PriceLevelManager) GetOrderBook(symbol string) *OrderBook {
	plm.mutex.RLock()
	defer plm.mutex.RUnlock()

	if book, exists := plm.orderBooks[symbol]; exists {
		return book
	}

	// Create new order book if it doesn't exist
	plm.mutex.RUnlock()
	plm.mutex.Lock()
	defer plm.mutex.Unlock()
	defer plm.mutex.RLock()

	// Double-check after acquiring write lock
	if book, exists := plm.orderBooks[symbol]; exists {
		return book
	}

	book := &OrderBook{
		Symbol: symbol,
		Bids:   &PriceLevelHeap{isBid: true},
		Asks:   &PriceLevelHeap{isBid: false},
	}
	heap.Init(book.Bids)
	heap.Init(book.Asks)
	plm.orderBooks[symbol] = book

	return book
}

// UpdatePriceLevel updates a price level in the order book
func (plm *PriceLevelManager) UpdatePriceLevel(symbol string, side string, price, quantity float64) {
	book := plm.GetOrderBook(symbol)
	book.mutex.Lock()
	defer book.mutex.Unlock()

	level := PriceLevel{
		Price:    price,
		Quantity: quantity,
		Orders:   1,
		Time:     time.Now(),
	}

	if side == "buy" {
		heap.Push(book.Bids, level)
	} else {
		heap.Push(book.Asks, level)
	}

	// Update metrics
	plm.mutex.Lock()
	plm.metrics["symbols_tracked"] = len(plm.orderBooks)
	plm.metrics["last_update"] = time.Now()
	plm.mutex.Unlock()
}

// GetBestBidAsk returns the best bid and ask prices
func (plm *PriceLevelManager) GetBestBidAsk(symbol string) (float64, float64) {
	book := plm.GetOrderBook(symbol)
	book.mutex.RLock()
	defer book.mutex.RUnlock()

	var bestBid, bestAsk float64

	if book.Bids.Len() > 0 {
		bestBid = book.Bids.levels[0].Price
	}

	if book.Asks.Len() > 0 {
		bestAsk = book.Asks.levels[0].Price
	}

	return bestBid, bestAsk
}

// GetSpread returns the bid-ask spread
func (plm *PriceLevelManager) GetSpread(symbol string) float64 {
	bestBid, bestAsk := plm.GetBestBidAsk(symbol)
	if bestBid > 0 && bestAsk > 0 {
		return bestAsk - bestBid
	}
	return 0
}

// GetMarketDepth returns market depth for a symbol
func (plm *PriceLevelManager) GetMarketDepth(symbol string, levels int) ([]PriceLevel, []PriceLevel) {
	book := plm.GetOrderBook(symbol)
	book.mutex.RLock()
	defer book.mutex.RUnlock()

	var bids, asks []PriceLevel

	// Get top bid levels
	bidCount := book.Bids.Len()
	if bidCount > levels {
		bidCount = levels
	}
	for i := 0; i < bidCount; i++ {
		bids = append(bids, book.Bids.levels[i])
	}

	// Get top ask levels
	askCount := book.Asks.Len()
	if askCount > levels {
		askCount = levels
	}
	for i := 0; i < askCount; i++ {
		asks = append(asks, book.Asks.levels[i])
	}

	return bids, asks
}

// GetPerformanceMetrics returns performance metrics
func (plm *PriceLevelManager) GetPerformanceMetrics() map[string]interface{} {
	plm.mutex.RLock()
	defer plm.mutex.RUnlock()

	metrics := make(map[string]interface{})
	for k, v := range plm.metrics {
		metrics[k] = v
	}

	return metrics
}

// RemovePriceLevel removes a price level from the order book
func (plm *PriceLevelManager) RemovePriceLevel(symbol string, side string, price float64) {
	book := plm.GetOrderBook(symbol)
	book.mutex.Lock()
	defer book.mutex.Unlock()

	var targetHeap *PriceLevelHeap
	if side == "buy" {
		targetHeap = book.Bids
	} else {
		targetHeap = book.Asks
	}

	// Find and remove the price level
	for i, level := range targetHeap.levels {
		if level.Price == price {
			// Remove element at index i
			targetHeap.levels = append(targetHeap.levels[:i], targetHeap.levels[i+1:]...)
			heap.Init(targetHeap) // Re-heapify
			break
		}
	}
}

// ClearOrderBook clears all price levels for a symbol
func (plm *PriceLevelManager) ClearOrderBook(symbol string) {
	plm.mutex.Lock()
	defer plm.mutex.Unlock()

	if book, exists := plm.orderBooks[symbol]; exists {
		book.mutex.Lock()
		book.Bids.levels = book.Bids.levels[:0]
		book.Asks.levels = book.Asks.levels[:0]
		book.mutex.Unlock()
	}
}
