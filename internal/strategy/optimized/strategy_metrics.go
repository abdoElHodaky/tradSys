package optimized

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// StrategyMetrics collects metrics for strategy operations
type StrategyMetrics struct {
	logger *zap.Logger
	mu     sync.RWMutex
	
	// Market data processing metrics
	marketDataLatencies map[string][]time.Duration
	
	// Order processing metrics
	orderLatencies map[string][]time.Duration
	
	// Strategy execution metrics
	strategyExecutionLatencies map[string]map[string][]time.Duration
	
	// Strategy operation metrics (start, stop, etc.)
	strategyOperationLatencies map[string]map[string][]time.Duration
}

// NewStrategyMetrics creates a new strategy metrics collector
func NewStrategyMetrics(logger *zap.Logger) *StrategyMetrics {
	return &StrategyMetrics{
		logger:                    logger,
		marketDataLatencies:       make(map[string][]time.Duration),
		orderLatencies:            make(map[string][]time.Duration),
		strategyExecutionLatencies: make(map[string]map[string][]time.Duration),
		strategyOperationLatencies: make(map[string]map[string][]time.Duration),
	}
}

// RecordMarketDataProcessing records the latency of market data processing
func (m *StrategyMetrics) RecordMarketDataProcessing(symbol string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.marketDataLatencies[symbol]; !ok {
		m.marketDataLatencies[symbol] = make([]time.Duration, 0, 100)
	}
	
	m.marketDataLatencies[symbol] = append(m.marketDataLatencies[symbol], duration)
	
	// Keep only the last 100 latencies
	if len(m.marketDataLatencies[symbol]) > 100 {
		m.marketDataLatencies[symbol] = m.marketDataLatencies[symbol][1:]
	}
	
	m.logger.Debug("Market data processing",
		zap.String("symbol", symbol),
		zap.Duration("duration", duration))
}

// RecordOrderProcessing records the latency of order processing
func (m *StrategyMetrics) RecordOrderProcessing(orderID string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.orderLatencies[orderID]; !ok {
		m.orderLatencies[orderID] = make([]time.Duration, 0, 10)
	}
	
	m.orderLatencies[orderID] = append(m.orderLatencies[orderID], duration)
	
	// Keep only the last 10 latencies
	if len(m.orderLatencies[orderID]) > 10 {
		m.orderLatencies[orderID] = m.orderLatencies[orderID][1:]
	}
	
	m.logger.Debug("Order processing",
		zap.String("order_id", orderID),
		zap.Duration("duration", duration))
}

// RecordStrategyExecution records the latency of strategy execution
func (m *StrategyMetrics) RecordStrategyExecution(strategyName, operationType string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.strategyExecutionLatencies[strategyName]; !ok {
		m.strategyExecutionLatencies[strategyName] = make(map[string][]time.Duration)
	}
	
	if _, ok := m.strategyExecutionLatencies[strategyName][operationType]; !ok {
		m.strategyExecutionLatencies[strategyName][operationType] = make([]time.Duration, 0, 100)
	}
	
	m.strategyExecutionLatencies[strategyName][operationType] = append(
		m.strategyExecutionLatencies[strategyName][operationType],
		duration,
	)
	
	// Keep only the last 100 latencies
	if len(m.strategyExecutionLatencies[strategyName][operationType]) > 100 {
		m.strategyExecutionLatencies[strategyName][operationType] = m.strategyExecutionLatencies[strategyName][operationType][1:]
	}
	
	m.logger.Debug("Strategy execution",
		zap.String("strategy", strategyName),
		zap.String("operation", operationType),
		zap.Duration("duration", duration))
}

// RecordStrategyOperation records the latency of strategy operations
func (m *StrategyMetrics) RecordStrategyOperation(strategyName, operationType string, duration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, ok := m.strategyOperationLatencies[strategyName]; !ok {
		m.strategyOperationLatencies[strategyName] = make(map[string][]time.Duration)
	}
	
	if _, ok := m.strategyOperationLatencies[strategyName][operationType]; !ok {
		m.strategyOperationLatencies[strategyName][operationType] = make([]time.Duration, 0, 10)
	}
	
	m.strategyOperationLatencies[strategyName][operationType] = append(
		m.strategyOperationLatencies[strategyName][operationType],
		duration,
	)
	
	// Keep only the last 10 latencies
	if len(m.strategyOperationLatencies[strategyName][operationType]) > 10 {
		m.strategyOperationLatencies[strategyName][operationType] = m.strategyOperationLatencies[strategyName][operationType][1:]
	}
	
	m.logger.Debug("Strategy operation",
		zap.String("strategy", strategyName),
		zap.String("operation", operationType),
		zap.Duration("duration", duration))
}

// GetMarketDataLatencies returns the latencies for market data processing
func (m *StrategyMetrics) GetMarketDataLatencies(symbol string) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if latencies, ok := m.marketDataLatencies[symbol]; ok {
		result := make([]time.Duration, len(latencies))
		copy(result, latencies)
		return result
	}
	
	return nil
}

// GetOrderLatencies returns the latencies for order processing
func (m *StrategyMetrics) GetOrderLatencies(orderID string) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if latencies, ok := m.orderLatencies[orderID]; ok {
		result := make([]time.Duration, len(latencies))
		copy(result, latencies)
		return result
	}
	
	return nil
}

// GetStrategyExecutionLatencies returns the latencies for strategy execution
func (m *StrategyMetrics) GetStrategyExecutionLatencies(strategyName, operationType string) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if strategyLatencies, ok := m.strategyExecutionLatencies[strategyName]; ok {
		if latencies, ok := strategyLatencies[operationType]; ok {
			result := make([]time.Duration, len(latencies))
			copy(result, latencies)
			return result
		}
	}
	
	return nil
}

// GetStrategyOperationLatencies returns the latencies for strategy operations
func (m *StrategyMetrics) GetStrategyOperationLatencies(strategyName, operationType string) []time.Duration {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if strategyLatencies, ok := m.strategyOperationLatencies[strategyName]; ok {
		if latencies, ok := strategyLatencies[operationType]; ok {
			result := make([]time.Duration, len(latencies))
			copy(result, latencies)
			return result
		}
	}
	
	return nil
}

// GetAverageMarketDataLatency returns the average latency for market data processing
func (m *StrategyMetrics) GetAverageMarketDataLatency(symbol string) time.Duration {
	latencies := m.GetMarketDataLatencies(symbol)
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, latency := range latencies {
		sum += latency
	}
	
	return sum / time.Duration(len(latencies))
}

// GetAverageOrderLatency returns the average latency for order processing
func (m *StrategyMetrics) GetAverageOrderLatency(orderID string) time.Duration {
	latencies := m.GetOrderLatencies(orderID)
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, latency := range latencies {
		sum += latency
	}
	
	return sum / time.Duration(len(latencies))
}

// GetAverageStrategyExecutionLatency returns the average latency for strategy execution
func (m *StrategyMetrics) GetAverageStrategyExecutionLatency(strategyName, operationType string) time.Duration {
	latencies := m.GetStrategyExecutionLatencies(strategyName, operationType)
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, latency := range latencies {
		sum += latency
	}
	
	return sum / time.Duration(len(latencies))
}

// GetAverageStrategyOperationLatency returns the average latency for strategy operations
func (m *StrategyMetrics) GetAverageStrategyOperationLatency(strategyName, operationType string) time.Duration {
	latencies := m.GetStrategyOperationLatencies(strategyName, operationType)
	if len(latencies) == 0 {
		return 0
	}
	
	var sum time.Duration
	for _, latency := range latencies {
		sum += latency
	}
	
	return sum / time.Duration(len(latencies))
}

