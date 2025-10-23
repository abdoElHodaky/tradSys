package execution

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// Order represents a trading order
type Order struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Side      string    `json:"side"` // "buy" or "sell"
	Type      string    `json:"type"` // "market", "limit", "stop"
	Quantity  float64   `json:"quantity"`
	Price     float64   `json:"price"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	UserID    string    `json:"user_id"`
}

// Trade represents an executed trade
type Trade struct {
	ID          string    `json:"id"`
	BuyOrderID  string    `json:"buy_order_id"`
	SellOrderID string    `json:"sell_order_id"`
	Symbol      string    `json:"symbol"`
	Quantity    float64   `json:"quantity"`
	Price       float64   `json:"price"`
	Fee         float64   `json:"fee"`
	Commission  float64   `json:"commission"`
	ExecutedAt  time.Time `json:"executed_at"`
	BuyerID     string    `json:"buyer_id"`
	SellerID    string    `json:"seller_id"`
}

// ExecutionResult represents the result of a trade execution
type ExecutionResult struct {
	Success    bool          `json:"success"`
	Trade      *Trade        `json:"trade,omitempty"`
	Error      string        `json:"error,omitempty"`
	Latency    time.Duration `json:"latency"`
	ExecutedAt time.Time     `json:"executed_at"`
}

// ExecutionEngine handles trade execution with ultra-low latency
type ExecutionEngine struct {
	trades          map[string]*Trade
	orders          map[string]*Order
	mutex           sync.RWMutex
	metrics         map[string]interface{}
	totalExecutions int64
	successfulExecs int64
	feeRate         float64
	commissionRate  float64
}

// NewExecutionEngine creates a new execution engine
func NewExecutionEngine(logger interface{}) *ExecutionEngine {
	return &ExecutionEngine{
		trades:         make(map[string]*Trade),
		orders:         make(map[string]*Order),
		metrics:        make(map[string]interface{}),
		feeRate:        0.0001, // 0.01%
		commissionRate: 0.0005, // 0.05%
	}
}

// ExecuteTrade executes a trade between two orders with microsecond latency
func (ee *ExecutionEngine) ExecuteTrade(ctx context.Context, buyOrder, sellOrder *Order, executionPrice float64) (*ExecutionResult, error) {
	start := time.Now()

	// Validate orders
	if err := ee.validateOrders(buyOrder, sellOrder); err != nil {
		return &ExecutionResult{
			Success:    false,
			Error:      err.Error(),
			Latency:    time.Since(start),
			ExecutedAt: time.Now(),
		}, err
	}

	// Calculate execution quantity (minimum of both orders)
	executionQuantity := buyOrder.Quantity
	if sellOrder.Quantity < executionQuantity {
		executionQuantity = sellOrder.Quantity
	}

	// Calculate fees and commissions
	tradeValue := executionPrice * executionQuantity
	fee := tradeValue * ee.feeRate
	commission := tradeValue * ee.commissionRate

	// Create trade record
	tradeID := fmt.Sprintf("trade_%d_%s", time.Now().UnixNano(), buyOrder.Symbol)
	trade := &Trade{
		ID:          tradeID,
		BuyOrderID:  buyOrder.ID,
		SellOrderID: sellOrder.ID,
		Symbol:      buyOrder.Symbol,
		Quantity:    executionQuantity,
		Price:       executionPrice,
		Fee:         fee,
		Commission:  commission,
		ExecutedAt:  time.Now(),
		BuyerID:     buyOrder.UserID,
		SellerID:    sellOrder.UserID,
	}

	// Store trade and update orders
	ee.mutex.Lock()
	ee.trades[tradeID] = trade

	// Update order statuses
	buyOrder.Status = "filled"
	sellOrder.Status = "filled"
	buyOrder.UpdatedAt = time.Now()
	sellOrder.UpdatedAt = time.Now()

	ee.orders[buyOrder.ID] = buyOrder
	ee.orders[sellOrder.ID] = sellOrder

	// Update metrics
	atomic.AddInt64(&ee.totalExecutions, 1)
	atomic.AddInt64(&ee.successfulExecs, 1)
	ee.updateMetrics()
	ee.mutex.Unlock()

	latency := time.Since(start)

	return &ExecutionResult{
		Success:    true,
		Trade:      trade,
		Latency:    latency,
		ExecutedAt: trade.ExecutedAt,
	}, nil
}

// validateOrders validates that orders can be matched
func (ee *ExecutionEngine) validateOrders(buyOrder, sellOrder *Order) error {
	if buyOrder.Symbol != sellOrder.Symbol {
		return fmt.Errorf("symbol mismatch: %s != %s", buyOrder.Symbol, sellOrder.Symbol)
	}

	if buyOrder.Side != "buy" {
		return fmt.Errorf("first order must be a buy order")
	}

	if sellOrder.Side != "sell" {
		return fmt.Errorf("second order must be a sell order")
	}

	if buyOrder.Quantity <= 0 || sellOrder.Quantity <= 0 {
		return fmt.Errorf("order quantities must be positive")
	}

	if buyOrder.UserID == sellOrder.UserID {
		return fmt.Errorf("cannot match orders from the same user")
	}

	return nil
}

// GetTrade retrieves a trade by ID
func (ee *ExecutionEngine) GetTrade(tradeID string) (*Trade, bool) {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	trade, exists := ee.trades[tradeID]
	return trade, exists
}

// GetOrder retrieves an order by ID
func (ee *ExecutionEngine) GetOrder(orderID string) (*Order, bool) {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	order, exists := ee.orders[orderID]
	return order, exists
}

// GetTradesBySymbol returns all trades for a symbol
func (ee *ExecutionEngine) GetTradesBySymbol(symbol string) []*Trade {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	var trades []*Trade
	for _, trade := range ee.trades {
		if trade.Symbol == symbol {
			trades = append(trades, trade)
		}
	}

	return trades
}

// GetTradesByUser returns all trades for a user
func (ee *ExecutionEngine) GetTradesByUser(userID string) []*Trade {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	var trades []*Trade
	for _, trade := range ee.trades {
		if trade.BuyerID == userID || trade.SellerID == userID {
			trades = append(trades, trade)
		}
	}

	return trades
}

// updateMetrics updates internal performance metrics
func (ee *ExecutionEngine) updateMetrics() {
	totalExecs := atomic.LoadInt64(&ee.totalExecutions)
	successfulExecs := atomic.LoadInt64(&ee.successfulExecs)

	var successRate float64
	if totalExecs > 0 {
		successRate = float64(successfulExecs) / float64(totalExecs)
	}

	ee.metrics["total_executions"] = totalExecs
	ee.metrics["successful_executions"] = successfulExecs
	ee.metrics["success_rate"] = successRate
	ee.metrics["total_trades"] = int64(len(ee.trades))
	ee.metrics["last_execution"] = time.Now()
}

// GetPerformanceMetrics returns execution engine performance metrics
func (ee *ExecutionEngine) GetPerformanceMetrics() map[string]interface{} {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	// Update metrics before returning
	ee.updateMetrics()

	metrics := make(map[string]interface{})
	for k, v := range ee.metrics {
		metrics[k] = v
	}

	return metrics
}

// SetFeeRate sets the trading fee rate
func (ee *ExecutionEngine) SetFeeRate(rate float64) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.feeRate = rate
}

// SetCommissionRate sets the commission rate
func (ee *ExecutionEngine) SetCommissionRate(rate float64) {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()
	ee.commissionRate = rate
}

// GetExecutionStats returns execution statistics
func (ee *ExecutionEngine) GetExecutionStats() map[string]interface{} {
	ee.mutex.RLock()
	defer ee.mutex.RUnlock()

	stats := make(map[string]interface{})
	stats["total_trades"] = len(ee.trades)
	stats["total_orders"] = len(ee.orders)
	stats["fee_rate"] = ee.feeRate
	stats["commission_rate"] = ee.commissionRate

	// Calculate total volume and fees
	var totalVolume, totalFees, totalCommissions float64
	for _, trade := range ee.trades {
		totalVolume += trade.Price * trade.Quantity
		totalFees += trade.Fee
		totalCommissions += trade.Commission
	}

	stats["total_volume"] = totalVolume
	stats["total_fees"] = totalFees
	stats["total_commissions"] = totalCommissions

	return stats
}

// CancelOrder cancels an order
func (ee *ExecutionEngine) CancelOrder(orderID string) error {
	ee.mutex.Lock()
	defer ee.mutex.Unlock()

	order, exists := ee.orders[orderID]
	if !exists {
		return fmt.Errorf("order %s not found", orderID)
	}

	if order.Status == "filled" {
		return fmt.Errorf("cannot cancel filled order %s", orderID)
	}

	order.Status = "cancelled"
	order.UpdatedAt = time.Now()
	ee.orders[orderID] = order

	return nil
}
