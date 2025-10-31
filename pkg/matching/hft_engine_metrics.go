package matching

import (
	"sync/atomic"
	"time"

	"go.uber.org/zap"
)

// GetStats returns a snapshot of engine statistics
func (e *HFTEngine) GetStats() EngineStats {
	return EngineStats{
		OrdersProcessed:   atomic.LoadUint64(&e.ordersProcessed),
		TradesExecuted:    atomic.LoadUint64(&e.tradesExecuted),
		AvgLatencyNs:      atomic.LoadUint64(&e.avgLatency),
		MaxLatencyNs:      atomic.LoadUint64(&e.stats.MaxLatencyNs),
		MinLatencyNs:      atomic.LoadUint64(&e.stats.MinLatencyNs),
		TotalVolumeTraded: atomic.LoadUint64(&e.stats.TotalVolumeTraded),
		ActiveOrders:      atomic.LoadUint64(&e.stats.ActiveOrders),
		CancelledOrders:   atomic.LoadUint64(&e.stats.CancelledOrders),
		RejectedOrders:    atomic.LoadUint64(&e.stats.RejectedOrders),
		LastUpdateTime:    e.stats.LastUpdateTime,
	}
}

// ResetStats resets all engine statistics
func (e *HFTEngine) ResetStats() {
	atomic.StoreUint64(&e.ordersProcessed, 0)
	atomic.StoreUint64(&e.tradesExecuted, 0)
	atomic.StoreUint64(&e.avgLatency, 0)
	atomic.StoreUint64(&e.stats.MaxLatencyNs, 0)
	atomic.StoreUint64(&e.stats.MinLatencyNs, 0)
	atomic.StoreUint64(&e.stats.TotalVolumeTraded, 0)
	atomic.StoreUint64(&e.stats.ActiveOrders, 0)
	atomic.StoreUint64(&e.stats.CancelledOrders, 0)
	atomic.StoreUint64(&e.stats.RejectedOrders, 0)
	e.stats.LastUpdateTime = time.Now()
	
	e.logger.Info("HFT Engine statistics reset")
}

// UpdateLatencyStats updates latency statistics with a new measurement
func (e *HFTEngine) UpdateLatencyStats(latencyNs uint64) {
	// Update average latency (simple moving average)
	currentAvg := atomic.LoadUint64(&e.avgLatency)
	ordersProcessed := atomic.LoadUint64(&e.ordersProcessed)
	
	if ordersProcessed > 0 {
		newAvg := (currentAvg*(ordersProcessed-1) + latencyNs) / ordersProcessed
		atomic.StoreUint64(&e.avgLatency, newAvg)
	}
	
	// Update max latency
	for {
		currentMax := atomic.LoadUint64(&e.stats.MaxLatencyNs)
		if latencyNs <= currentMax {
			break
		}
		if atomic.CompareAndSwapUint64(&e.stats.MaxLatencyNs, currentMax, latencyNs) {
			break
		}
	}
	
	// Update min latency
	for {
		currentMin := atomic.LoadUint64(&e.stats.MinLatencyNs)
		if currentMin == 0 || latencyNs < currentMin {
			if atomic.CompareAndSwapUint64(&e.stats.MinLatencyNs, currentMin, latencyNs) {
				break
			}
		} else {
			break
		}
	}
}

// GetOrderBookStats returns statistics for a specific order book
func (e *HFTEngine) GetOrderBookStats(symbol string) *OrderBookStats {
	orderBook := e.getOrderBook(symbol)
	if orderBook == nil {
		return nil
	}
	
	return &OrderBookStats{
		Symbol:        orderBook.Symbol,
		TotalOrders:   atomic.LoadUint64(&orderBook.totalOrders),
		TotalTrades:   atomic.LoadUint64(&orderBook.totalTrades),
		TotalVolume:   atomic.LoadUint64(&orderBook.totalVolume),
		BestBid:       atomic.LoadUint64(&orderBook.bestBid),
		BestAsk:       atomic.LoadUint64(&orderBook.bestAsk),
		Spread:        atomic.LoadUint64(&orderBook.spread),
		LastTradeTime: orderBook.lastTradeTime,
	}
}

// OrderBookStats represents statistics for a specific order book
type OrderBookStats struct {
	Symbol        string
	TotalOrders   uint64
	TotalTrades   uint64
	TotalVolume   uint64
	BestBid       uint64
	BestAsk       uint64
	Spread        uint64
	LastTradeTime time.Time
}

// GetAllOrderBookStats returns statistics for all order books
func (e *HFTEngine) GetAllOrderBookStats() map[string]*OrderBookStats {
	orderBooks := (*map[string]*HFTOrderBook)(atomic.LoadPointer(&e.orderBooks))
	stats := make(map[string]*OrderBookStats)
	
	for symbol := range *orderBooks {
		if bookStats := e.GetOrderBookStats(symbol); bookStats != nil {
			stats[symbol] = bookStats
		}
	}
	
	return stats
}

// performanceMonitor monitors engine performance
func (e *HFTEngine) performanceMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			stats := e.GetStats()
			e.logger.Debug("HFT Engine Performance",
				zap.Uint64("orders_processed", stats.OrdersProcessed),
				zap.Uint64("trades_executed", stats.TradesExecuted),
				zap.Uint64("avg_latency_ns", stats.AvgLatencyNs),
				zap.Uint64("max_latency_ns", stats.MaxLatencyNs),
				zap.Uint64("min_latency_ns", stats.MinLatencyNs),
				zap.Uint64("active_orders", stats.ActiveOrders),
				zap.Uint64("total_volume", stats.TotalVolumeTraded))
		}
	}
}

// GetThroughputMetrics calculates throughput metrics
func (e *HFTEngine) GetThroughputMetrics() ThroughputMetrics {
	stats := e.GetStats()
	
	// Calculate orders per second (simple approximation)
	ordersPerSecond := float64(stats.OrdersProcessed)
	tradesPerSecond := float64(stats.TradesExecuted)
	
	// In a real implementation, you'd track time windows for accurate rates
	return ThroughputMetrics{
		OrdersPerSecond:    ordersPerSecond,
		TradesPerSecond:    tradesPerSecond,
		AvgLatencyMs:       float64(stats.AvgLatencyNs) / 1_000_000,
		MaxLatencyMs:       float64(stats.MaxLatencyNs) / 1_000_000,
		MinLatencyMs:       float64(stats.MinLatencyNs) / 1_000_000,
		TotalVolumeTraded:  stats.TotalVolumeTraded,
		ActiveOrders:       stats.ActiveOrders,
	}
}

// ThroughputMetrics represents throughput and performance metrics
type ThroughputMetrics struct {
	OrdersPerSecond    float64
	TradesPerSecond    float64
	AvgLatencyMs       float64
	MaxLatencyMs       float64
	MinLatencyMs       float64
	TotalVolumeTraded  uint64
	ActiveOrders       uint64
}

// LogPerformanceMetrics logs comprehensive performance metrics
func (e *HFTEngine) LogPerformanceMetrics() {
	stats := e.GetStats()
	throughput := e.GetThroughputMetrics()
	orderBookStats := e.GetAllOrderBookStats()
	
	e.logger.Info("HFT Engine Performance Summary",
		zap.Uint64("total_orders_processed", stats.OrdersProcessed),
		zap.Uint64("total_trades_executed", stats.TradesExecuted),
		zap.Float64("avg_latency_ms", throughput.AvgLatencyMs),
		zap.Float64("max_latency_ms", throughput.MaxLatencyMs),
		zap.Float64("min_latency_ms", throughput.MinLatencyMs),
		zap.Float64("orders_per_second", throughput.OrdersPerSecond),
		zap.Float64("trades_per_second", throughput.TradesPerSecond),
		zap.Uint64("active_orders", stats.ActiveOrders),
		zap.Uint64("cancelled_orders", stats.CancelledOrders),
		zap.Uint64("rejected_orders", stats.RejectedOrders),
		zap.Uint64("total_volume_traded", stats.TotalVolumeTraded),
		zap.Int("active_order_books", len(orderBookStats)))
}

// GetHealthMetrics returns health-related metrics for monitoring
func (e *HFTEngine) GetHealthMetrics() HealthMetrics {
	stats := e.GetStats()
	
	// Calculate health indicators
	errorRate := float64(stats.RejectedOrders) / float64(stats.OrdersProcessed) * 100
	if stats.OrdersProcessed == 0 {
		errorRate = 0
	}
	
	cancellationRate := float64(stats.CancelledOrders) / float64(stats.OrdersProcessed) * 100
	if stats.OrdersProcessed == 0 {
		cancellationRate = 0
	}
	
	return HealthMetrics{
		IsHealthy:           stats.AvgLatencyNs < 1_000_000, // Less than 1ms average
		ErrorRate:           errorRate,
		CancellationRate:    cancellationRate,
		AvgLatencyMs:        float64(stats.AvgLatencyNs) / 1_000_000,
		ActiveOrders:        stats.ActiveOrders,
		TotalOrdersProcessed: stats.OrdersProcessed,
		LastUpdateTime:      stats.LastUpdateTime,
	}
}

// HealthMetrics represents health-related metrics
type HealthMetrics struct {
	IsHealthy            bool
	ErrorRate            float64
	CancellationRate     float64
	AvgLatencyMs         float64
	ActiveOrders         uint64
	TotalOrdersProcessed uint64
	LastUpdateTime       time.Time
}
