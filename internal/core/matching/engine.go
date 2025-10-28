package order_matching

import (
	"errors"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrInvalidOrder = errors.New("invalid order")
	ErrOrderNotFound = errors.New("order not found")
	ErrInsufficientLiquidity = errors.New("insufficient liquidity")
)

// Engine interface defines the contract for order matching engines
type Engine interface {
	ProcessOrder(order *Order) ([]*Trade, error)
	CancelOrder(orderID string) error
	GetOrderBook(symbol string) (*OrderBookSnapshot, error)
	GetPerformanceMetrics() map[string]interface{}
}

// DefaultEngine is the default implementation of the Engine interface
type DefaultEngine struct {
	logger *zap.Logger
	optimizedEngine *OptimizedEngine
}

// NewEngine creates a new default matching engine
func NewEngine(logger *zap.Logger) *DefaultEngine {
	return &DefaultEngine{
		logger: logger,
		optimizedEngine: NewOptimizedEngine(logger),
	}
}

// ProcessOrder processes an order using the optimized engine
func (e *DefaultEngine) ProcessOrder(order *Order) ([]*Trade, error) {
	return e.optimizedEngine.ProcessOrderFast(order)
}

// CancelOrder cancels an order
func (e *DefaultEngine) CancelOrder(orderID string) error {
	// TODO: Implement order cancellation
	e.logger.Info("Order cancellation requested", zap.String("orderID", orderID))
	return nil
}

// GetOrderBook returns the order book snapshot for a symbol
func (e *DefaultEngine) GetOrderBook(symbol string) (*OrderBookSnapshot, error) {
	// TODO: Implement order book snapshot
	e.logger.Info("Order book snapshot requested", zap.String("symbol", symbol))
	return &OrderBookSnapshot{
		Symbol: symbol,
		Bids: []*OrderBookLevel{},
		Asks: []*OrderBookLevel{},
	}, nil
}

// GetPerformanceMetrics returns performance metrics
func (e *DefaultEngine) GetPerformanceMetrics() map[string]interface{} {
	return e.optimizedEngine.GetPerformanceMetrics()
}
