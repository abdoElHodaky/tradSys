package matching

import (
	"fmt"
	"sort"
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// GetStatus returns the current status of the matching engine
func (me *MatchingEngine) GetStatus() *EngineStatus {
	me.mu.RLock()
	defer me.mu.RUnlock()

	totalOrders := 0
	for _, orderBook := range me.OrderBooks {
		orderBook.mu.RLock()
		totalOrders += len(orderBook.Orders)
		orderBook.mu.RUnlock()
	}

	me.Metrics.mu.RLock()
	status := &EngineStatus{
		Running:         me.Running,
		TotalSymbols:    len(me.OrderBooks),
		TotalOrders:     totalOrders,
		TotalTrades:     atomic.LoadInt64(&me.Metrics.TotalTrades),
		LastTradeTime:   me.Metrics.LastTradeTime,
		OrdersPerSecond: me.Metrics.OrdersPerSecond,
		TradesPerSecond: me.Metrics.TradesPerSecond,
		AverageLatency:  me.Metrics.AverageLatency,
	}
	me.Metrics.mu.RUnlock()

	return status
}

// GetMetrics returns the current metrics
func (me *MatchingEngine) GetMetrics() *EngineMetrics {
	me.Metrics.mu.RLock()
	defer me.Metrics.mu.RUnlock()

	// Return a copy to avoid race conditions
	return &EngineMetrics{
		TotalTrades:     atomic.LoadInt64(&me.Metrics.TotalTrades),
		TotalOrders:     atomic.LoadInt64(&me.Metrics.TotalOrders),
		TotalVolume:     me.Metrics.TotalVolume,
		AverageLatency:  me.Metrics.AverageLatency,
		MaxLatency:      me.Metrics.MaxLatency,
		MinLatency:      me.Metrics.MinLatency,
		LastTradeTime:   me.Metrics.LastTradeTime,
		OrdersPerSecond: me.Metrics.OrdersPerSecond,
		TradesPerSecond: me.Metrics.TradesPerSecond,
	}
}

// UpdateMetrics updates the engine metrics
func (me *MatchingEngine) UpdateMetrics(latency time.Duration) {
	me.Metrics.mu.Lock()
	defer me.Metrics.mu.Unlock()

	// Update latency metrics
	if latency > me.Metrics.MaxLatency {
		me.Metrics.MaxLatency = latency
	}
	if latency < me.Metrics.MinLatency {
		me.Metrics.MinLatency = latency
	}

	// Calculate average latency (simple moving average)
	totalTrades := atomic.LoadInt64(&me.Metrics.TotalTrades)
	if totalTrades > 0 {
		me.Metrics.AverageLatency = time.Duration(
			(int64(me.Metrics.AverageLatency)*(totalTrades-1) + int64(latency)) / totalTrades,
		)
	}
}

// GetOrderBookSnapshot returns a snapshot of an order book
func (me *MatchingEngine) GetOrderBookSnapshot(symbol string) (*OrderBookSnapshot, error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	orderBook, exists := me.OrderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	return orderBook.GetSnapshot(), nil
}

// GetSnapshot returns a snapshot of the order book
func (ob *OrderBook) GetSnapshot() *OrderBookSnapshot {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	snapshot := &OrderBookSnapshot{
		Symbol:    ob.Symbol,
		LastPrice: ob.LastPrice,
		Timestamp: time.Now(),
	}

	// Copy bids
	snapshot.Bids = make([]*Order, len(ob.Bids.Orders))
	copy(snapshot.Bids, ob.Bids.Orders)

	// Copy asks
	snapshot.Asks = make([]*Order, len(ob.Asks.Orders))
	copy(snapshot.Asks, ob.Asks.Orders)

	// Calculate depths
	for _, bid := range snapshot.Bids {
		snapshot.BidDepth += bid.RemainingQuantity()
	}
	for _, ask := range snapshot.Asks {
		snapshot.AskDepth += ask.RemainingQuantity()
	}

	// Calculate spread
	if len(snapshot.Bids) > 0 && len(snapshot.Asks) > 0 {
		bestBid := snapshot.Bids[0].Price
		bestAsk := snapshot.Asks[0].Price
		snapshot.Spread = bestAsk - bestBid
	}

	return snapshot
}

// GetMarketData returns market data for a symbol
func (me *MatchingEngine) GetMarketData(symbol string) (*MarketData, error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	orderBook, exists := me.OrderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	return orderBook.GetMarketData(), nil
}

// GetMarketData returns market data for the order book
func (ob *OrderBook) GetMarketData() *MarketData {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	marketData := &MarketData{
		Symbol:    ob.Symbol,
		LastPrice: ob.LastPrice,
		Timestamp: time.Now(),
	}

	// Get best bid and ask
	if ob.Bids.Len() > 0 {
		bestBid := ob.Bids.Peek()
		marketData.BestBid = bestBid.Price
		marketData.BidSize = bestBid.RemainingQuantity()
	}

	if ob.Asks.Len() > 0 {
		bestAsk := ob.Asks.Peek()
		marketData.BestAsk = bestAsk.Price
		marketData.AskSize = bestAsk.RemainingQuantity()
	}

	// Calculate 24h statistics (simplified - would need historical data in real implementation)
	marketData.High = ob.LastPrice * 1.05 // Placeholder
	marketData.Low = ob.LastPrice * 0.95  // Placeholder
	marketData.Open = ob.LastPrice * 0.98 // Placeholder
	marketData.Close = ob.LastPrice
	marketData.Change = marketData.Close - marketData.Open
	if marketData.Open != 0 {
		marketData.ChangePercent = (marketData.Change / marketData.Open) * 100
	}

	return marketData
}

// GetOrderBookDepth returns the order book depth
func (me *MatchingEngine) GetOrderBookDepth(symbol string, levels int) (*OrderBookDepth, error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	orderBook, exists := me.OrderBooks[symbol]
	if !exists {
		return nil, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	return orderBook.GetDepth(levels), nil
}

// GetDepth returns the order book depth with specified levels
func (ob *OrderBook) GetDepth(levels int) *OrderBookDepth {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	depth := &OrderBookDepth{
		Symbol:    ob.Symbol,
		Timestamp: time.Now(),
	}

	// Aggregate bids by price level
	bidLevels := make(map[float64]*PriceLevel)
	for _, order := range ob.Bids.Orders {
		level, exists := bidLevels[order.Price]
		if !exists {
			level = &PriceLevel{
				Price:  order.Price,
				Orders: make([]*Order, 0),
			}
			bidLevels[order.Price] = level
		}
		level.Quantity += order.RemainingQuantity()
		level.OrderCount++
		level.Orders = append(level.Orders, order)
	}

	// Aggregate asks by price level
	askLevels := make(map[float64]*PriceLevel)
	for _, order := range ob.Asks.Orders {
		level, exists := askLevels[order.Price]
		if !exists {
			level = &PriceLevel{
				Price:  order.Price,
				Orders: make([]*Order, 0),
			}
			askLevels[order.Price] = level
		}
		level.Quantity += order.RemainingQuantity()
		level.OrderCount++
		level.Orders = append(level.Orders, order)
	}

	// Convert to sorted slices
	for _, level := range bidLevels {
		depth.Bids = append(depth.Bids, level)
	}
	for _, level := range askLevels {
		depth.Asks = append(depth.Asks, level)
	}

	// Sort bids (highest price first)
	sort.Slice(depth.Bids, func(i, j int) bool {
		return depth.Bids[i].Price > depth.Bids[j].Price
	})

	// Sort asks (lowest price first)
	sort.Slice(depth.Asks, func(i, j int) bool {
		return depth.Asks[i].Price < depth.Asks[j].Price
	})

	// Limit to requested levels
	if levels > 0 {
		if len(depth.Bids) > levels {
			depth.Bids = depth.Bids[:levels]
		}
		if len(depth.Asks) > levels {
			depth.Asks = depth.Asks[:levels]
		}
	}

	return depth
}

// GetTradeHistory returns trade history for a symbol
func (me *MatchingEngine) GetTradeHistory(symbol string, limit int) (*TradeHistory, error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	var symbolTrades []*Trade
	for _, trade := range me.Trades {
		if trade.Symbol == symbol {
			symbolTrades = append(symbolTrades, trade)
		}
	}

	// Sort by timestamp (most recent first)
	sort.Slice(symbolTrades, func(i, j int) bool {
		return symbolTrades[i].Timestamp.After(symbolTrades[j].Timestamp)
	})

	// Limit results
	if limit > 0 && len(symbolTrades) > limit {
		symbolTrades = symbolTrades[:limit]
	}

	if len(symbolTrades) == 0 {
		return &TradeHistory{
			Symbol: symbol,
			Trades: symbolTrades,
		}, nil
	}

	// Calculate statistics
	history := &TradeHistory{
		Symbol:      symbol,
		Trades:      symbolTrades,
		StartTime:   symbolTrades[len(symbolTrades)-1].Timestamp,
		EndTime:     symbolTrades[0].Timestamp,
		TotalTrades: len(symbolTrades),
	}

	var totalValue float64
	for _, trade := range symbolTrades {
		history.TotalVolume += trade.Quantity
		totalValue += trade.Quantity * trade.Price
	}

	// Calculate VWAP
	if history.TotalVolume > 0 {
		history.VWAP = totalValue / history.TotalVolume
	}

	return history, nil
}

// GetAllSymbols returns all symbols with active order books
func (me *MatchingEngine) GetAllSymbols() []string {
	me.mu.RLock()
	defer me.mu.RUnlock()

	symbols := make([]string, 0, len(me.OrderBooks))
	for symbol := range me.OrderBooks {
		symbols = append(symbols, symbol)
	}

	sort.Strings(symbols)
	return symbols
}

// GetTotalVolume returns the total volume traded
func (me *MatchingEngine) GetTotalVolume() float64 {
	me.Metrics.mu.RLock()
	defer me.Metrics.mu.RUnlock()
	return me.Metrics.TotalVolume
}

// GetTotalTrades returns the total number of trades
func (me *MatchingEngine) GetTotalTrades() int64 {
	return atomic.LoadInt64(&me.Metrics.TotalTrades)
}

// GetTotalOrders returns the total number of orders processed
func (me *MatchingEngine) GetTotalOrders() int64 {
	return atomic.LoadInt64(&me.Metrics.TotalOrders)
}

// LogStatus logs the current engine status
func (me *MatchingEngine) LogStatus() {
	status := me.GetStatus()
	me.logger.Info("Matching Engine Status",
		zap.Bool("running", status.Running),
		zap.Int("total_symbols", status.TotalSymbols),
		zap.Int("total_orders", status.TotalOrders),
		zap.Int64("total_trades", status.TotalTrades),
		zap.Float64("orders_per_second", status.OrdersPerSecond),
		zap.Float64("trades_per_second", status.TradesPerSecond),
		zap.Duration("average_latency", status.AverageLatency))
}

// LogOrderBookStatus logs the status of a specific order book
func (me *MatchingEngine) LogOrderBookStatus(symbol string) {
	me.mu.RLock()
	orderBook, exists := me.OrderBooks[symbol]
	me.mu.RUnlock()

	if !exists {
		me.logger.Warn("Order book not found", zap.String("symbol", symbol))
		return
	}

	orderBook.mu.RLock()
	defer orderBook.mu.RUnlock()

	me.logger.Info("Order Book Status",
		zap.String("symbol", symbol),
		zap.Int("total_orders", len(orderBook.Orders)),
		zap.Int("bid_orders", orderBook.Bids.Len()),
		zap.Int("ask_orders", orderBook.Asks.Len()),
		zap.Float64("last_price", orderBook.LastPrice))
}

// ResetMetrics resets all metrics
func (me *MatchingEngine) ResetMetrics() {
	me.Metrics.mu.Lock()
	defer me.Metrics.mu.Unlock()

	atomic.StoreInt64(&me.Metrics.TotalTrades, 0)
	atomic.StoreInt64(&me.Metrics.TotalOrders, 0)
	me.Metrics.TotalVolume = 0
	me.Metrics.AverageLatency = 0
	me.Metrics.MaxLatency = 0
	me.Metrics.MinLatency = time.Hour
	me.Metrics.LastTradeTime = time.Time{}
	me.Metrics.OrdersPerSecond = 0
	me.Metrics.TradesPerSecond = 0

	me.logger.Info("Metrics reset")
}

// CalculatePerformanceMetrics calculates performance metrics
func (me *MatchingEngine) CalculatePerformanceMetrics(duration time.Duration) {
	me.Metrics.mu.Lock()
	defer me.Metrics.mu.Unlock()

	if duration <= 0 {
		return
	}

	seconds := duration.Seconds()
	me.Metrics.OrdersPerSecond = float64(atomic.LoadInt64(&me.Metrics.TotalOrders)) / seconds
	me.Metrics.TradesPerSecond = float64(atomic.LoadInt64(&me.Metrics.TotalTrades)) / seconds
}

// GetBestBidAsk returns the best bid and ask for a symbol
func (me *MatchingEngine) GetBestBidAsk(symbol string) (bestBid, bestAsk float64, err error) {
	me.mu.RLock()
	defer me.mu.RUnlock()

	orderBook, exists := me.OrderBooks[symbol]
	if !exists {
		return 0, 0, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	orderBook.mu.RLock()
	defer orderBook.mu.RUnlock()

	if orderBook.Bids.Len() > 0 {
		bestBid = orderBook.Bids.Peek().Price
	}

	if orderBook.Asks.Len() > 0 {
		bestAsk = orderBook.Asks.Peek().Price
	}

	return bestBid, bestAsk, nil
}

// GetSpread returns the bid-ask spread for a symbol
func (me *MatchingEngine) GetSpread(symbol string) (float64, error) {
	bestBid, bestAsk, err := me.GetBestBidAsk(symbol)
	if err != nil {
		return 0, err
	}

	if bestBid > 0 && bestAsk > 0 {
		return bestAsk - bestBid, nil
	}

	return 0, nil
}

// GetMidPrice returns the mid price for a symbol
func (me *MatchingEngine) GetMidPrice(symbol string) (float64, error) {
	bestBid, bestAsk, err := me.GetBestBidAsk(symbol)
	if err != nil {
		return 0, err
	}

	if bestBid > 0 && bestAsk > 0 {
		return (bestBid + bestAsk) / 2, nil
	}

	return 0, nil
}

// ValidateOrderBook validates the integrity of an order book
func (ob *OrderBook) ValidateOrderBook() []string {
	ob.mu.RLock()
	defer ob.mu.RUnlock()

	var issues []string

	// Check if heaps are properly ordered
	for i := 1; i < len(ob.Bids.Orders); i++ {
		if ob.Bids.Orders[i-1].Price < ob.Bids.Orders[i].Price {
			issues = append(issues, fmt.Sprintf("Bid heap not properly ordered at index %d", i))
		}
	}

	for i := 1; i < len(ob.Asks.Orders); i++ {
		if ob.Asks.Orders[i-1].Price > ob.Asks.Orders[i].Price {
			issues = append(issues, fmt.Sprintf("Ask heap not properly ordered at index %d", i))
		}
	}

	// Check for negative quantities
	for _, order := range ob.Orders {
		if order.Quantity <= 0 {
			issues = append(issues, fmt.Sprintf("Order %s has non-positive quantity", order.ID))
		}
		if order.RemainingQuantity() < 0 {
			issues = append(issues, fmt.Sprintf("Order %s has negative remaining quantity", order.ID))
		}
	}

	return issues
}

// GetHealthStatus returns the health status of the matching engine
func (me *MatchingEngine) GetHealthStatus() map[string]interface{} {
	status := me.GetStatus()
	metrics := me.GetMetrics()

	health := map[string]interface{}{
		"status":            "healthy",
		"running":           status.Running,
		"total_symbols":     status.TotalSymbols,
		"total_orders":      status.TotalOrders,
		"total_trades":      status.TotalTrades,
		"orders_per_second": status.OrdersPerSecond,
		"trades_per_second": status.TradesPerSecond,
		"average_latency":   metrics.AverageLatency.String(),
		"max_latency":       metrics.MaxLatency.String(),
		"min_latency":       metrics.MinLatency.String(),
		"total_volume":      metrics.TotalVolume,
		"last_trade_time":   metrics.LastTradeTime,
	}

	// Check for potential issues
	issues := make([]string, 0)

	if !status.Running {
		health["status"] = "stopped"
		issues = append(issues, "Engine is not running")
	}

	if metrics.AverageLatency > time.Millisecond*100 {
		health["status"] = "degraded"
		issues = append(issues, "High average latency detected")
	}

	if len(issues) > 0 {
		health["issues"] = issues
	}

	return health
}
