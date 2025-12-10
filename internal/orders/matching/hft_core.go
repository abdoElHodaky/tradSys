// 🎯 **HFT Engine Core Service**
// Generated using TradSys Code Splitting Standards
//
// This file contains the main service struct, constructor, and core API methods
// for the High-Frequency Trading Engine component. It follows the established patterns for
// service initialization, lifecycle management, and primary business operations.
//
// Performance Requirements: <100μs latency, zero-allocation order processing
// File size limit: 350 lines

package order_matching

import (
	"context"
	"runtime"
	"sync/atomic"
	"time"
	"unsafe"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"go.uber.org/zap"
)

// NewHFTEngine creates a new HFT-optimized order matching engine
func NewHFTEngine(logger *zap.Logger) *HFTEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize order books map
	orderBooksMap := make(map[string]*HFTOrderBook)

	engine := &HFTEngine{
		TradeChannel:  make(chan *Trade, DefaultTradeChannelBuffer), // Large buffer for high throughput
		fastOrderPool: pool.NewFastOrderPool(),                      // Fast order pool for zero-allocation processing
		tradePool:     pool.NewTradePool(DefaultTradePoolSize),      // Pre-allocate trades
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		workerPool:    make(chan struct{}, runtime.NumCPU()*2), // 2x CPU cores
		stats: &EngineStats{
			MinLatencyNanos: ^uint64(0), // Max uint64 value
		},
	}

	// Store the map atomically
	atomic.StorePointer(&engine.orderBooks, unsafe.Pointer(&orderBooksMap))

	// Start performance monitoring
	go engine.monitorPerformance()

	// Start trade processor
	go engine.processTradesAsync()

	return engine
}

// NewHFTEngineWithConfig creates a new HFT engine with custom configuration
func NewHFTEngineWithConfig(config *HFTEngineConfig, logger *zap.Logger) *HFTEngine {
	ctx, cancel := context.WithCancel(context.Background())

	// Initialize order books map
	orderBooksMap := make(map[string]*HFTOrderBook)

	engine := &HFTEngine{
		TradeChannel:  make(chan *Trade, config.TradeChannelBuffer),
		fastOrderPool: pool.NewFastOrderPool(),
		tradePool:     pool.NewTradePool(config.TradePoolSize),
		logger:        logger,
		ctx:           ctx,
		cancel:        cancel,
		workerPool:    make(chan struct{}, config.WorkerPoolSize),
		stats: &EngineStats{
			MinLatencyNanos: ^uint64(0), // Max uint64 value
		},
	}

	// Store the map atomically
	atomic.StorePointer(&engine.orderBooks, unsafe.Pointer(&orderBooksMap))

	// Start background processes based on config
	if config.EnableMetrics {
		go engine.monitorPerformance()
	}

	go engine.processTradesAsync()

	return engine
}

// PlaceOrderFast places an order with HFT optimizations
func (e *HFTEngine) PlaceOrderFast(order *Order) ([]*Trade, error) {
	startTime := time.Now()

	// Get or create order book
	orderBook := e.getOrCreateOrderBook(order.Symbol)

	// Convert to fast order for better performance
	fastOrder := e.fastOrderPool.Get()
	defer e.fastOrderPool.Put(fastOrder) // Return to pool when done

	fastOrder.Order = *order
	fastOrder.PriceInt64 = int64(order.Price * 100000000) // 8 decimal places precision
	fastOrder.QuantityInt64 = int64(order.Quantity * 100000000)
	fastOrder.CreatedAtNano = startTime.UnixNano()
	fastOrder.UpdatedAtNano = startTime.UnixNano()

	// Process order
	trades, err := orderBook.processOrderFast(fastOrder)

	// Update performance metrics
	latency := uint64(time.Since(startTime).Nanoseconds())
	e.updateLatencyStats(latency)
	atomic.AddUint64(&e.ordersProcessed, 1)

	// Send trades to channel asynchronously
	if len(trades) > 0 {
		go func() {
			for _, trade := range trades {
				select {
				case e.TradeChannel <- trade:
					atomic.AddUint64(&e.tradesExecuted, 1)
				default:
					e.logger.Warn("Trade channel full, dropping trade",
						zap.String("trade_id", trade.ID),
						zap.String("symbol", trade.Symbol))
				}
			}
		}()
	}

	return trades, err
}

// PlaceOrder places an order (standard interface)
func (e *HFTEngine) PlaceOrder(order *Order) ([]*Trade, error) {
	return e.PlaceOrderFast(order)
}

// CancelOrder cancels an existing order
func (e *HFTEngine) CancelOrder(orderID, symbol string) error {
	orderBook := e.getOrderBook(symbol)
	if orderBook == nil {
		return ErrOrderBookNotFound
	}

	return orderBook.cancelOrder(orderID)
}

// GetOrderBook returns the order book for a symbol
func (e *HFTEngine) GetOrderBook(symbol string) *HFTOrderBook {
	return e.getOrderBook(symbol)
}

// GetOrderBookSnapshot returns a snapshot of the order book
func (e *HFTEngine) GetOrderBookSnapshot(symbol string, depth int) (*HFTOrderBookSnapshot, error) {
	orderBook := e.getOrderBook(symbol)
	if orderBook == nil {
		return nil, ErrOrderBookNotFound
	}

	return orderBook.getSnapshot(depth), nil
}

// GetStats returns current engine statistics
func (e *HFTEngine) GetStats() *EngineStats {
	return &EngineStats{
		OrdersProcessed:   atomic.LoadUint64(&e.ordersProcessed),
		TradesExecuted:    atomic.LoadUint64(&e.tradesExecuted),
		AvgLatencyNanos:   atomic.LoadUint64(&e.avgLatency),
		MaxLatencyNanos:   atomic.LoadUint64(&e.stats.MaxLatencyNanos),
		MinLatencyNanos:   atomic.LoadUint64(&e.stats.MinLatencyNanos),
		TotalLatencyNanos: atomic.LoadUint64(&e.stats.TotalLatencyNanos),
		LastUpdateTime:    atomic.LoadInt64(&e.stats.LastUpdateTime),
	}
}

// GetPerformanceMetrics returns detailed performance metrics
func (e *HFTEngine) GetPerformanceMetrics() *HFTPerformanceMetrics {
	stats := e.GetStats()

	metrics := &HFTPerformanceMetrics{
		OrdersPerSecond: float64(stats.OrdersProcessed) / time.Since(time.Unix(0, stats.LastUpdateTime)).Seconds(),
		TradesPerSecond: float64(stats.TradesExecuted) / time.Since(time.Unix(0, stats.LastUpdateTime)).Seconds(),
		LastUpdateTime:  time.Now().UnixNano(),
	}

	metrics.OrderProcessingLatency.Min = stats.MinLatencyNanos
	metrics.OrderProcessingLatency.Max = stats.MaxLatencyNanos
	metrics.OrderProcessingLatency.Avg = stats.AvgLatencyNanos

	return metrics
}

// GetEngineState returns the current state of the engine
func (e *HFTEngine) GetEngineState() *HFTEngineState {
	orderBooksPtr := atomic.LoadPointer(&e.orderBooks)
	orderBooksMap := (*map[string]*HFTOrderBook)(orderBooksPtr)

	state := &HFTEngineState{
		IsRunning:      e.ctx.Err() == nil,
		ActiveSymbols:  len(*orderBooksMap),
		TotalOrders:    atomic.LoadUint64(&e.ordersProcessed),
		TotalTrades:    atomic.LoadUint64(&e.tradesExecuted),
		CurrentLatency: atomic.LoadUint64(&e.avgLatency),
		HealthStatus:   e.getHealthStatus(),
	}

	return state
}

// Shutdown gracefully shuts down the engine
func (e *HFTEngine) Shutdown() error {
	e.cancel()

	// Wait for goroutines to finish (with timeout)
	done := make(chan struct{})
	go func() {
		// Wait for trade channel to drain
		for len(e.TradeChannel) > 0 {
			time.Sleep(time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		e.logger.Info("HFT engine shutdown completed")
	case <-time.After(5 * time.Second):
		e.logger.Warn("HFT engine shutdown timeout, forcing close")
	}

	close(e.TradeChannel)
	return nil
}

// getOrCreateOrderBook gets or creates an order book for a symbol
func (e *HFTEngine) getOrCreateOrderBook(symbol string) *HFTOrderBook {
	// Load current map
	orderBooksPtr := atomic.LoadPointer(&e.orderBooks)
	orderBooksMap := (*map[string]*HFTOrderBook)(orderBooksPtr)

	// Check if order book exists
	if orderBook, exists := (*orderBooksMap)[symbol]; exists {
		return orderBook
	}

	// Create new order book
	newOrderBook := &HFTOrderBook{
		Symbol:      symbol,
		logger:      e.logger,
		lastUpdated: time.Now().UnixNano(),
	}

	// Initialize price level trees
	bidsTree := &PriceLevelTree{side: OrderSideBuy}
	asksTree := &PriceLevelTree{side: OrderSideSell}
	atomic.StorePointer(&newOrderBook.bids, unsafe.Pointer(bidsTree))
	atomic.StorePointer(&newOrderBook.asks, unsafe.Pointer(asksTree))

	// Create new map with the new order book
	newOrderBooksMap := make(map[string]*HFTOrderBook)
	for k, v := range *orderBooksMap {
		newOrderBooksMap[k] = v
	}
	newOrderBooksMap[symbol] = newOrderBook

	// Atomically update the map
	atomic.StorePointer(&e.orderBooks, unsafe.Pointer(&newOrderBooksMap))

	return newOrderBook
}

// getOrderBook gets an existing order book for a symbol
func (e *HFTEngine) getOrderBook(symbol string) *HFTOrderBook {
	orderBooksPtr := atomic.LoadPointer(&e.orderBooks)
	orderBooksMap := (*map[string]*HFTOrderBook)(orderBooksPtr)

	if orderBook, exists := (*orderBooksMap)[symbol]; exists {
		return orderBook
	}

	return nil
}

// updateLatencyStats updates latency statistics atomically
func (e *HFTEngine) updateLatencyStats(latency uint64) {
	// Update average latency
	currentAvg := atomic.LoadUint64(&e.avgLatency)
	newAvg := (currentAvg + latency) / 2
	atomic.StoreUint64(&e.avgLatency, newAvg)

	// Update max latency
	for {
		currentMax := atomic.LoadUint64(&e.stats.MaxLatencyNanos)
		if latency <= currentMax {
			break
		}
		if atomic.CompareAndSwapUint64(&e.stats.MaxLatencyNanos, currentMax, latency) {
			break
		}
	}

	// Update min latency
	for {
		currentMin := atomic.LoadUint64(&e.stats.MinLatencyNanos)
		if latency >= currentMin {
			break
		}
		if atomic.CompareAndSwapUint64(&e.stats.MinLatencyNanos, currentMin, latency) {
			break
		}
	}

	// Update total latency
	atomic.AddUint64(&e.stats.TotalLatencyNanos, latency)
	atomic.StoreInt64(&e.stats.LastUpdateTime, time.Now().UnixNano())
}

// getHealthStatus determines the current health status
func (e *HFTEngine) getHealthStatus() string {
	currentLatency := atomic.LoadUint64(&e.avgLatency)

	if currentLatency < MaxTargetLatencyNanos/2 {
		return HealthStatusHealthy
	} else if currentLatency < MaxTargetLatencyNanos {
		return HealthStatusDegraded
	} else if currentLatency < MaxTargetLatencyNanos*2 {
		return HealthStatusUnhealthy
	} else {
		return HealthStatusCritical
	}
}
