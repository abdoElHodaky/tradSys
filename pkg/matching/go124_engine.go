package matching

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Go124MatchingEngine implements the OrderMatchingEngine interface
// using Go 1.24 features and optimized patterns
type Go124MatchingEngine struct {
	mu          sync.RWMutex
	orderBooks  map[string]*OrderBook
	orders      types.OrderCache
	trades      []types.Trade
	metrics     types.Metadata
	running     bool
	ctx         context.Context
	cancel      context.CancelFunc
	eventBus    types.OrderEventBus
	riskManager interfaces.RiskManager
}

// NewGo124MatchingEngine creates a new Go 1.24 optimized matching engine
func NewGo124MatchingEngine(
	cache types.OrderCache,
	eventBus types.OrderEventBus,
	riskManager interfaces.RiskManager,
) *Go124MatchingEngine {
	ctx, cancel := context.WithCancel(context.Background())

	return &Go124MatchingEngine{
		orderBooks:  make(map[string]*OrderBook),
		orders:      cache,
		trades:      make([]types.Trade, 0),
		metrics:     make(types.Metadata),
		ctx:         ctx,
		cancel:      cancel,
		eventBus:    eventBus,
		riskManager: riskManager,
	}
}

// AddOrder implements the MatchingEngine interface
func (e *Go124MatchingEngine) AddOrder(ctx context.Context, order types.Order) (types.Result[types.Trade], error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Validate order using risk manager
	if e.riskManager != nil {
		validationResult := e.riskManager.ValidateOrder(ctx, order)
		if validationResult.IsError() {
			return types.NewResultWithError[types.Trade](validationResult.Error), nil
		}
		if !validationResult.Unwrap() {
			return types.NewResultWithError[types.Trade](fmt.Errorf("order failed risk validation")), nil
		}
	}

	// Get or create order book for symbol
	orderBook := e.getOrCreateOrderBook(order.Symbol)

	// Cache the order
	e.orders.Set(order.ID, order, time.Hour)

	// Try to match the order
	trade, matched := e.matchOrder(orderBook, order)

	if matched {
		// Update metrics
		e.updateMetrics("trades_executed", 1)
		e.updateMetrics("volume_traded", trade.Value)

		// Publish trade event
		if e.eventBus != nil {
			// Note: This would need proper event publishing implementation
			// e.eventBus.Publish(ctx, "trade.executed", trade)
		}

		return types.NewResult(trade), nil
	}

	// Add order to book if not fully matched
	e.addOrderToBook(orderBook, order)

	// Update metrics
	e.updateMetrics("orders_added", 1)

	// Return empty trade result
	var emptyTrade types.Trade
	return types.NewResult(emptyTrade), nil
}

// CancelOrder implements the MatchingEngine interface
func (e *Go124MatchingEngine) CancelOrder(ctx context.Context, orderID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Get order from cache
	order, exists := e.orders.Get(orderID)
	if !exists {
		return fmt.Errorf("order not found: %s", orderID)
	}

	// Remove from order book
	orderBook := e.orderBooks[order.Symbol]
	if orderBook != nil {
		e.removeOrderFromBook(orderBook, order)
	}

	// Remove from cache
	e.orders.Delete(orderID)

	// Update metrics
	e.updateMetrics("orders_canceled", 1)

	return nil
}

// GetOrderBook implements the MatchingEngine interface
func (e *Go124MatchingEngine) GetOrderBook(ctx context.Context, symbol string) (types.OrderBook, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	orderBook := e.orderBooks[symbol]
	if orderBook == nil {
		return types.OrderBook{}, fmt.Errorf("order book not found for symbol: %s", symbol)
	}

	// Convert internal order book to public format
	return e.convertOrderBook(orderBook), nil
}

// GetTrades implements the MatchingEngine interface
func (e *Go124MatchingEngine) GetTrades(ctx context.Context, symbol string, limit int) ([]types.Trade, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	var symbolTrades []types.Trade
	for _, trade := range e.trades {
		if trade.Symbol == symbol {
			symbolTrades = append(symbolTrades, trade)
			if len(symbolTrades) >= limit {
				break
			}
		}
	}

	return symbolTrades, nil
}

// GetMetrics implements the MatchingEngine interface
func (e *Go124MatchingEngine) GetMetrics(ctx context.Context) types.Metadata {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Create a copy of metrics to avoid race conditions
	metricsCopy := make(types.Metadata)
	for k, v := range e.metrics {
		metricsCopy[k] = v
	}

	// Add runtime metrics
	metricsCopy["order_books_count"] = len(e.orderBooks)
	metricsCopy["cached_orders_count"] = e.orders.Size()
	metricsCopy["total_trades_count"] = len(e.trades)
	metricsCopy["engine_status"] = e.getStatus()
	metricsCopy["last_updated"] = time.Now()

	return metricsCopy
}

// Start implements the MatchingEngine interface
func (e *Go124MatchingEngine) Start(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if e.running {
		return fmt.Errorf("matching engine already running")
	}

	e.running = true
	e.updateMetrics("engine_started_at", time.Now())

	// Start background processes
	go e.metricsCollector()
	go e.orderBookMaintenance()

	return nil
}

// Stop implements the MatchingEngine interface
func (e *Go124MatchingEngine) Stop(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if !e.running {
		return fmt.Errorf("matching engine not running")
	}

	e.running = false
	e.cancel()
	e.updateMetrics("engine_stopped_at", time.Now())

	return nil
}

// Health implements the MatchingEngine interface
func (e *Go124MatchingEngine) Health() types.HealthStatus {
	e.mu.RLock()
	defer e.mu.RUnlock()

	status := "healthy"
	message := "Matching engine is operating normally"

	if !e.running {
		status = "stopped"
		message = "Matching engine is not running"
	}

	details := make(types.Metadata)
	details["order_books"] = len(e.orderBooks)
	details["cached_orders"] = e.orders.Size()
	details["total_trades"] = len(e.trades)

	return types.HealthStatus{
		Status:    status,
		Message:   message,
		Timestamp: time.Now(),
		Details:   details,
	}
}

// Private helper methods

func (e *Go124MatchingEngine) getOrCreateOrderBook(symbol string) *OrderBook {
	orderBook := e.orderBooks[symbol]
	if orderBook == nil {
		orderBook = &OrderBook{
			Symbol:    symbol,
			UpdatedAt: time.Now(),
		}
		e.orderBooks[symbol] = orderBook
	}
	return orderBook
}

func (e *Go124MatchingEngine) matchOrder(orderBook *OrderBook, order types.Order) (types.Trade, bool) {
	// Simplified matching logic - in a real implementation this would be more sophisticated
	var trade types.Trade

	// For now, return no match
	return trade, false
}

func (e *Go124MatchingEngine) addOrderToBook(orderBook *OrderBook, order types.Order) {
	// Simplified order book management - in a real implementation this would maintain price-time priority
	orderBook.UpdatedAt = time.Now()
}

func (e *Go124MatchingEngine) removeOrderFromBook(orderBook *OrderBook, order types.Order) {
	// Simplified order removal - in a real implementation this would properly manage the order book structure
	orderBook.UpdatedAt = time.Now()
}

func (e *Go124MatchingEngine) convertOrderBook(internal *OrderBook) types.OrderBook {
	// Convert internal order book format to public API format
	return types.OrderBook{
		Symbol: internal.Symbol,
		Bids:   []*types.OrderBookLevel{},
		Asks:   []*types.OrderBookLevel{},
	}
}

func (e *Go124MatchingEngine) updateMetrics(key string, value interface{}) {
	e.metrics[key] = value
}

func (e *Go124MatchingEngine) getStatus() string {
	if e.running {
		return "running"
	}
	return "stopped"
}

func (e *Go124MatchingEngine) metricsCollector() {
	ticker := time.NewTicker(time.Second * 30)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			e.updateMetrics("last_metrics_update", time.Now())
			e.mu.Unlock()
		}
	}
}

func (e *Go124MatchingEngine) orderBookMaintenance() {
	ticker := time.NewTicker(time.Minute * 5)
	defer ticker.Stop()

	for {
		select {
		case <-e.ctx.Done():
			return
		case <-ticker.C:
			e.mu.Lock()
			// Perform order book maintenance tasks
			for _, orderBook := range e.orderBooks {
				orderBook.UpdatedAt = time.Now()
			}
			e.mu.Unlock()
		}
	}
}

// Ensure Go124MatchingEngine implements the interface
var _ interfaces.OrderMatchingEngine = (*Go124MatchingEngine)(nil)
