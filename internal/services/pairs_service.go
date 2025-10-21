package services

import (
	"context"
	"fmt"
	"time"
)

// PairsServiceImpl implements the PairsService interface
type PairsServiceImpl struct {
	pairs map[string]*TradingPair
	tickers map[string]*Ticker
}

// NewPairsService creates a new pairs service instance
func NewPairsService() PairsService {
	service := &PairsServiceImpl{
		pairs:   make(map[string]*TradingPair),
		tickers: make(map[string]*Ticker),
	}
	
	// Initialize with some default pairs
	service.initializeDefaultPairs()
	
	return service
}

// GetPair retrieves a trading pair by symbol
func (s *PairsServiceImpl) GetPair(ctx context.Context, symbol string) (*TradingPair, error) {
	pair, exists := s.pairs[symbol]
	if !exists {
		return nil, fmt.Errorf("trading pair not found: %s", symbol)
	}
	
	return pair, nil
}

// ListPairs retrieves trading pairs based on filter criteria
func (s *PairsServiceImpl) ListPairs(ctx context.Context, filter *PairFilter) ([]*TradingPair, error) {
	var result []*TradingPair
	
	for _, pair := range s.pairs {
		if s.matchesPairFilter(pair, filter) {
			result = append(result, pair)
		}
	}
	
	// Apply pagination
	if filter != nil {
		start := filter.Offset
		if start > len(result) {
			start = len(result)
		}
		
		end := start + filter.Limit
		if filter.Limit == 0 || end > len(result) {
			end = len(result)
		}
		
		if start < end {
			result = result[start:end]
		} else {
			result = []*TradingPair{}
		}
	}
	
	return result, nil
}

// GetPairInfo retrieves detailed information about a trading pair
func (s *PairsServiceImpl) GetPairInfo(ctx context.Context, symbol string) (*PairInfo, error) {
	_, exists := s.pairs[symbol]
	if !exists {
		return nil, fmt.Errorf("trading pair not found: %s", symbol)
	}
	
	// Simulate market data
	info := &PairInfo{
		Symbol:             symbol,
		Volume24h:          1000000.0 + float64(time.Now().Unix()%100000),
		PriceChange:        -0.5 + float64(time.Now().Unix()%100)/100.0,
		PriceChangePercent: -0.5 + float64(time.Now().Unix()%100)/100.0,
		HighPrice:          50000.0 + float64(time.Now().Unix()%1000),
		LowPrice:           49000.0 + float64(time.Now().Unix()%1000),
		LastPrice:          49500.0 + float64(time.Now().Unix()%1000),
	}
	
	return info, nil
}

// GetTicker retrieves current ticker data for a symbol
func (s *PairsServiceImpl) GetTicker(ctx context.Context, symbol string) (*Ticker, error) {
	// Check if pair exists
	if _, exists := s.pairs[symbol]; !exists {
		return nil, fmt.Errorf("trading pair not found: %s", symbol)
	}
	
	// Generate or retrieve ticker data
	ticker := s.generateTicker(symbol)
	s.tickers[symbol] = ticker
	
	return ticker, nil
}

// GetOrderBook retrieves order book data for a symbol
func (s *PairsServiceImpl) GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error) {
	// Check if pair exists
	if _, exists := s.pairs[symbol]; !exists {
		return nil, fmt.Errorf("trading pair not found: %s", symbol)
	}
	
	if depth <= 0 {
		depth = 10 // Default depth
	}
	
	orderBook := s.generateOrderBook(symbol, depth)
	return orderBook, nil
}

// GetTrades retrieves recent trades for a symbol
func (s *PairsServiceImpl) GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error) {
	// Check if pair exists
	if _, exists := s.pairs[symbol]; !exists {
		return nil, fmt.Errorf("trading pair not found: %s", symbol)
	}
	
	if limit <= 0 {
		limit = 50 // Default limit
	}
	
	trades := s.generateRecentTrades(symbol, limit)
	return trades, nil
}

// initializeDefaultPairs sets up some default trading pairs
func (s *PairsServiceImpl) initializeDefaultPairs() {
	defaultPairs := []*TradingPair{
		{
			Symbol:      "BTCUSDT",
			BaseAsset:   "BTC",
			QuoteAsset:  "USDT",
			Status:      "TRADING",
			MinQuantity: 0.00001,
			MaxQuantity: 9000.0,
			StepSize:    0.00001,
			MinPrice:    0.01,
			MaxPrice:    1000000.0,
			TickSize:    0.01,
		},
		{
			Symbol:      "ETHUSDT",
			BaseAsset:   "ETH",
			QuoteAsset:  "USDT",
			Status:      "TRADING",
			MinQuantity: 0.0001,
			MaxQuantity: 90000.0,
			StepSize:    0.0001,
			MinPrice:    0.01,
			MaxPrice:    100000.0,
			TickSize:    0.01,
		},
		{
			Symbol:      "ADAUSDT",
			BaseAsset:   "ADA",
			QuoteAsset:  "USDT",
			Status:      "TRADING",
			MinQuantity: 0.1,
			MaxQuantity: 900000.0,
			StepSize:    0.1,
			MinPrice:    0.0001,
			MaxPrice:    1000.0,
			TickSize:    0.0001,
		},
	}
	
	for _, pair := range defaultPairs {
		s.pairs[pair.Symbol] = pair
	}
}

// generateTicker creates simulated ticker data
func (s *PairsServiceImpl) generateTicker(symbol string) *Ticker {
	basePrice := 50000.0
	if symbol == "ETHUSDT" {
		basePrice = 3000.0
	} else if symbol == "ADAUSDT" {
		basePrice = 0.5
	}
	
	// Add some randomness based on current time
	variation := float64(time.Now().Unix()%1000) / 1000.0 * 0.1 // ±10% variation
	price := basePrice * (0.95 + variation)
	
	return &Ticker{
		Symbol:        symbol,
		Price:         price,
		Bid:           price * 0.999,
		Ask:           price * 1.001,
		Volume:        100000.0 + float64(time.Now().Unix()%50000),
		Change:        price * 0.02 * (variation - 0.05), // ±1% change
		ChangePercent: 2.0 * (variation - 0.05),          // ±10% change
		Timestamp:     time.Now(),
	}
}

// generateOrderBook creates simulated order book data
func (s *PairsServiceImpl) generateOrderBook(symbol string, depth int) *OrderBook {
	ticker := s.generateTicker(symbol)
	
	var bids, asks []OrderBookEntry
	
	// Generate bids (buy orders) - prices below current price
	for i := 0; i < depth; i++ {
		price := ticker.Price * (1.0 - float64(i+1)*0.001) // Decreasing prices
		quantity := 1.0 + float64(i)*0.5                   // Increasing quantities
		bids = append(bids, OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}
	
	// Generate asks (sell orders) - prices above current price
	for i := 0; i < depth; i++ {
		price := ticker.Price * (1.0 + float64(i+1)*0.001) // Increasing prices
		quantity := 1.0 + float64(i)*0.5                   // Increasing quantities
		asks = append(asks, OrderBookEntry{
			Price:    price,
			Quantity: quantity,
		})
	}
	
	return &OrderBook{
		Symbol:    symbol,
		Bids:      bids,
		Asks:      asks,
		Timestamp: time.Now(),
	}
}

// generateRecentTrades creates simulated recent trade data
func (s *PairsServiceImpl) generateRecentTrades(symbol string, limit int) []*Trade {
	ticker := s.generateTicker(symbol)
	var trades []*Trade
	
	for i := 0; i < limit; i++ {
		// Generate trade data with some variation
		variation := float64(i%10) / 1000.0 // Small price variations
		price := ticker.Price * (0.999 + variation)
		quantity := 0.1 + float64(i%5)*0.2
		
		side := "buy"
		if i%2 == 0 {
			side = "sell"
		}
		
		trade := &Trade{
			ID:         fmt.Sprintf("trade_%s_%d", symbol, i),
			Symbol:     symbol,
			Side:       side,
			Quantity:   quantity,
			Price:      price,
			Commission: price * quantity * 0.001, // 0.1% commission
			Timestamp:  time.Now().Add(-time.Duration(i) * time.Minute),
		}
		
		trades = append(trades, trade)
	}
	
	return trades
}

// matchesPairFilter checks if a trading pair matches the given filter
func (s *PairsServiceImpl) matchesPairFilter(pair *TradingPair, filter *PairFilter) bool {
	if filter == nil {
		return true
	}
	
	if filter.BaseAsset != nil && pair.BaseAsset != *filter.BaseAsset {
		return false
	}
	if filter.QuoteAsset != nil && pair.QuoteAsset != *filter.QuoteAsset {
		return false
	}
	if filter.Status != nil && pair.Status != *filter.Status {
		return false
	}
	
	return true
}
