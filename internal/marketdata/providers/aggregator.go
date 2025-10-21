package marketdata

import (
	"context"
	"math"
	"sync"
	"time"
)

// MarketDataPoint represents a single market data point
type MarketDataPoint struct {
	Symbol      string    `json:"symbol"`
	Exchange    string    `json:"exchange"`
	BidPrice    float64   `json:"bid_price"`
	AskPrice    float64   `json:"ask_price"`
	BidSize     float64   `json:"bid_size"`
	AskSize     float64   `json:"ask_size"`
	LastPrice   float64   `json:"last_price"`
	Volume      float64   `json:"volume"`
	Timestamp   time.Time `json:"timestamp"`
	Confidence  float64   `json:"confidence"` // 0.0 to 1.0
	Latency     time.Duration `json:"latency"`
}

// AggregatedMarketData represents aggregated market data from multiple sources
type AggregatedMarketData struct {
	Symbol           string                 `json:"symbol"`
	BestBid          float64               `json:"best_bid"`
	BestAsk          float64               `json:"best_ask"`
	BestBidExchange  string                `json:"best_bid_exchange"`
	BestAskExchange  string                `json:"best_ask_exchange"`
	WeightedBid      float64               `json:"weighted_bid"`
	WeightedAsk      float64               `json:"weighted_ask"`
	Spread           float64               `json:"spread"`
	TotalBidSize     float64               `json:"total_bid_size"`
	TotalAskSize     float64               `json:"total_ask_size"`
	LastPrice        float64               `json:"last_price"`
	Volume           float64               `json:"volume"`
	SourceCount      int                   `json:"source_count"`
	Confidence       float64               `json:"confidence"`
	Timestamp        time.Time             `json:"timestamp"`
	Sources          []*MarketDataPoint    `json:"sources"`
}

// DataSource represents a market data source
type DataSource struct {
	Name       string    `json:"name"`
	Priority   int       `json:"priority"`   // Higher is better
	Weight     float64   `json:"weight"`     // 0.0 to 1.0
	Latency    time.Duration `json:"latency"`
	LastUpdate time.Time `json:"last_update"`
	IsActive   bool      `json:"is_active"`
}

// MarketDataAggregator aggregates market data from multiple exchanges
type MarketDataAggregator struct {
	sources         map[string]*DataSource
	marketData      map[string]map[string]*MarketDataPoint // symbol -> exchange -> data
	aggregatedData  map[string]*AggregatedMarketData       // symbol -> aggregated data
	subscribers     map[string][]chan *AggregatedMarketData // symbol -> subscribers
	mutex           sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	updateInterval  time.Duration
	staleThreshold  time.Duration
	running         bool
}

// NewMarketDataAggregator creates a new market data aggregator
func NewMarketDataAggregator(updateInterval time.Duration) *MarketDataAggregator {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &MarketDataAggregator{
		sources:         make(map[string]*DataSource),
		marketData:      make(map[string]map[string]*MarketDataPoint),
		aggregatedData:  make(map[string]*AggregatedMarketData),
		subscribers:     make(map[string][]chan *AggregatedMarketData),
		ctx:             ctx,
		cancel:          cancel,
		updateInterval:  updateInterval,
		staleThreshold:  5 * time.Second,
	}
}

// AddDataSource adds a new data source
func (mda *MarketDataAggregator) AddDataSource(name string, priority int, weight float64) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	mda.sources[name] = &DataSource{
		Name:       name,
		Priority:   priority,
		Weight:     weight,
		IsActive:   true,
		LastUpdate: time.Now(),
	}
}

// RemoveDataSource removes a data source
func (mda *MarketDataAggregator) RemoveDataSource(name string) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	delete(mda.sources, name)
	
	// Remove data from this source
	for symbol := range mda.marketData {
		delete(mda.marketData[symbol], name)
	}
}

// UpdateMarketData updates market data from a specific source
func (mda *MarketDataAggregator) UpdateMarketData(data *MarketDataPoint) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	// Initialize symbol map if needed
	if mda.marketData[data.Symbol] == nil {
		mda.marketData[data.Symbol] = make(map[string]*MarketDataPoint)
	}
	
	// Store the data point
	mda.marketData[data.Symbol][data.Exchange] = data
	
	// Update source last update time
	if source, exists := mda.sources[data.Exchange]; exists {
		source.LastUpdate = time.Now()
		source.Latency = data.Latency
	}
	
	// Trigger aggregation for this symbol
	go mda.aggregateSymbol(data.Symbol)
}

// aggregateSymbol aggregates market data for a specific symbol
func (mda *MarketDataAggregator) aggregateSymbol(symbol string) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	symbolData, exists := mda.marketData[symbol]
	if !exists || len(symbolData) == 0 {
		return
	}
	
	// Filter out stale data
	now := time.Now()
	var validData []*MarketDataPoint
	
	for exchange, data := range symbolData {
		if now.Sub(data.Timestamp) <= mda.staleThreshold {
			// Check if source is active
			if source, exists := mda.sources[exchange]; exists && source.IsActive {
				validData = append(validData, data)
			}
		}
	}
	
	if len(validData) == 0 {
		return
	}
	
	// Aggregate the data
	aggregated := mda.performAggregation(symbol, validData)
	mda.aggregatedData[symbol] = aggregated
	
	// Notify subscribers
	if subscribers, exists := mda.subscribers[symbol]; exists {
		for _, subscriber := range subscribers {
			select {
			case subscriber <- aggregated:
			default:
				// Subscriber channel is full, skip
			}
		}
	}
}

// performAggregation performs the actual aggregation logic
func (mda *MarketDataAggregator) performAggregation(symbol string, data []*MarketDataPoint) *AggregatedMarketData {
	if len(data) == 0 {
		return nil
	}
	
	aggregated := &AggregatedMarketData{
		Symbol:      symbol,
		Sources:     data,
		SourceCount: len(data),
		Timestamp:   time.Now(),
	}
	
	// Find best bid and ask
	var bestBid, bestAsk float64 = 0, math.MaxFloat64
	var bestBidExchange, bestAskExchange string
	
	// Calculate weighted prices and totals
	var weightedBidSum, weightedAskSum, totalBidWeight, totalAskWeight float64
	var totalBidSize, totalAskSize, totalVolume float64
	var confidenceSum float64
	
	for _, point := range data {
		// Update best bid
		if point.BidPrice > bestBid {
			bestBid = point.BidPrice
			bestBidExchange = point.Exchange
		}
		
		// Update best ask
		if point.AskPrice < bestAsk && point.AskPrice > 0 {
			bestAsk = point.AskPrice
			bestAskExchange = point.Exchange
		}
		
		// Get source weight
		weight := 1.0
		if source, exists := mda.sources[point.Exchange]; exists {
			weight = source.Weight
		}
		
		// Calculate weighted prices
		if point.BidPrice > 0 {
			weightedBidSum += point.BidPrice * weight
			totalBidWeight += weight
		}
		
		if point.AskPrice > 0 {
			weightedAskSum += point.AskPrice * weight
			totalAskWeight += weight
		}
		
		// Accumulate sizes and volume
		totalBidSize += point.BidSize
		totalAskSize += point.AskSize
		totalVolume += point.Volume
		confidenceSum += point.Confidence
		
		// Update last price (use highest priority source)
		if source, exists := mda.sources[point.Exchange]; exists {
			if aggregated.LastPrice == 0 || source.Priority > mda.getSourcePriority(aggregated.LastPrice, data) {
				aggregated.LastPrice = point.LastPrice
			}
		}
	}
	
	// Calculate weighted averages
	if totalBidWeight > 0 {
		aggregated.WeightedBid = weightedBidSum / totalBidWeight
	}
	
	if totalAskWeight > 0 {
		aggregated.WeightedAsk = weightedAskSum / totalAskWeight
	}
	
	// Set best prices
	aggregated.BestBid = bestBid
	aggregated.BestAsk = bestAsk
	aggregated.BestBidExchange = bestBidExchange
	aggregated.BestAskExchange = bestAskExchange
	
	// Calculate spread
	if bestBid > 0 && bestAsk < math.MaxFloat64 {
		aggregated.Spread = bestAsk - bestBid
	}
	
	// Set totals
	aggregated.TotalBidSize = totalBidSize
	aggregated.TotalAskSize = totalAskSize
	aggregated.Volume = totalVolume
	
	// Calculate overall confidence
	if len(data) > 0 {
		aggregated.Confidence = confidenceSum / float64(len(data))
	}
	
	return aggregated
}

// getSourcePriority gets the priority of a source that provided a price
func (mda *MarketDataAggregator) getSourcePriority(price float64, data []*MarketDataPoint) int {
	for _, point := range data {
		if point.LastPrice == price {
			if source, exists := mda.sources[point.Exchange]; exists {
				return source.Priority
			}
		}
	}
	return 0
}

// GetAggregatedData returns aggregated market data for a symbol
func (mda *MarketDataAggregator) GetAggregatedData(symbol string) (*AggregatedMarketData, bool) {
	mda.mutex.RLock()
	defer mda.mutex.RUnlock()
	
	data, exists := mda.aggregatedData[symbol]
	if exists {
		// Return a copy to avoid race conditions
		dataCopy := *data
		return &dataCopy, true
	}
	
	return nil, false
}

// Subscribe subscribes to aggregated market data updates for a symbol
func (mda *MarketDataAggregator) Subscribe(symbol string) <-chan *AggregatedMarketData {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	subscriber := make(chan *AggregatedMarketData, 100)
	mda.subscribers[symbol] = append(mda.subscribers[symbol], subscriber)
	
	return subscriber
}

// Unsubscribe unsubscribes from market data updates
func (mda *MarketDataAggregator) Unsubscribe(symbol string, subscriber <-chan *AggregatedMarketData) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	if subscribers, exists := mda.subscribers[symbol]; exists {
		for i, sub := range subscribers {
			if sub == subscriber {
				// Remove subscriber from slice
				mda.subscribers[symbol] = append(subscribers[:i], subscribers[i+1:]...)
				close(sub)
				break
			}
		}
	}
}

// Start starts the aggregator
func (mda *MarketDataAggregator) Start() {
	mda.mutex.Lock()
	if mda.running {
		mda.mutex.Unlock()
		return
	}
	mda.running = true
	mda.mutex.Unlock()
	
	// Start periodic aggregation
	go mda.periodicAggregation()
}

// Stop stops the aggregator
func (mda *MarketDataAggregator) Stop() {
	mda.mutex.Lock()
	if !mda.running {
		mda.mutex.Unlock()
		return
	}
	mda.running = false
	mda.mutex.Unlock()
	
	mda.cancel()
	
	// Close all subscriber channels
	mda.mutex.Lock()
	for symbol, subscribers := range mda.subscribers {
		for _, subscriber := range subscribers {
			close(subscriber)
		}
		delete(mda.subscribers, symbol)
	}
	mda.mutex.Unlock()
}

// periodicAggregation performs periodic aggregation of all symbols
func (mda *MarketDataAggregator) periodicAggregation() {
	ticker := time.NewTicker(mda.updateInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-mda.ctx.Done():
			return
		case <-ticker.C:
			mda.aggregateAllSymbols()
		}
	}
}

// aggregateAllSymbols aggregates data for all symbols
func (mda *MarketDataAggregator) aggregateAllSymbols() {
	mda.mutex.RLock()
	symbols := make([]string, 0, len(mda.marketData))
	for symbol := range mda.marketData {
		symbols = append(symbols, symbol)
	}
	mda.mutex.RUnlock()
	
	for _, symbol := range symbols {
		mda.aggregateSymbol(symbol)
	}
}

// GetDataSources returns all configured data sources
func (mda *MarketDataAggregator) GetDataSources() map[string]*DataSource {
	mda.mutex.RLock()
	defer mda.mutex.RUnlock()
	
	sources := make(map[string]*DataSource)
	for name, source := range mda.sources {
		sourceCopy := *source
		sources[name] = &sourceCopy
	}
	
	return sources
}

// SetSourceActive sets the active status of a data source
func (mda *MarketDataAggregator) SetSourceActive(name string, active bool) {
	mda.mutex.Lock()
	defer mda.mutex.Unlock()
	
	if source, exists := mda.sources[name]; exists {
		source.IsActive = active
	}
}

// GetStats returns aggregator statistics
func (mda *MarketDataAggregator) GetStats() map[string]interface{} {
	mda.mutex.RLock()
	defer mda.mutex.RUnlock()
	
	stats := make(map[string]interface{})
	stats["total_sources"] = len(mda.sources)
	stats["tracked_symbols"] = len(mda.marketData)
	stats["aggregated_symbols"] = len(mda.aggregatedData)
	stats["total_subscribers"] = mda.getTotalSubscribers()
	stats["running"] = mda.running
	stats["update_interval"] = mda.updateInterval
	stats["stale_threshold"] = mda.staleThreshold
	
	// Source statistics
	activeSources := 0
	for _, source := range mda.sources {
		if source.IsActive {
			activeSources++
		}
	}
	stats["active_sources"] = activeSources
	
	return stats
}

// getTotalSubscribers returns the total number of subscribers
func (mda *MarketDataAggregator) getTotalSubscribers() int {
	total := 0
	for _, subscribers := range mda.subscribers {
		total += len(subscribers)
	}
	return total
}
