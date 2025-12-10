// Package matching provides concrete implementations of matching engines.
package matching

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// BasicEngine implements a basic matching engine
type BasicEngine struct {
	*types.BaseEngine
}

// Start starts the basic engine
func (e *BasicEngine) Start(ctx context.Context) error {
	e.SetRunning(true)
	return nil
}

// Stop stops the basic engine
func (e *BasicEngine) Stop(ctx context.Context) error {
	e.SetRunning(false)
	return nil
}

// ProcessOrder processes an order in the basic engine
func (e *BasicEngine) ProcessOrder(order *types.Order) (*types.Trade, error) {
	if !e.IsRunning() {
		return nil, types.NewEngineError("ENGINE_NOT_RUNNING", "Engine is not running", "", types.SeverityHigh)
	}
	
	// Basic order processing logic
	trade := &types.Trade{
		ID:          fmt.Sprintf("trade_%d", time.Now().UnixNano()),
		Symbol:      order.Symbol,
		BuyOrderID:  order.ID,
		SellOrderID: order.ID,
		Price:       order.Price,
		Quantity:    order.Quantity,
		Value:       order.Price * order.Quantity,
		Timestamp:   time.Now(),
		TakerSide:   order.Side,
	}
	
	return trade, nil
}

// AdvancedEngine implements an advanced matching engine with price improvement
type AdvancedEngine struct {
	*types.BaseEngine
	priceImprovementEnabled bool
}

// Start starts the advanced engine
func (e *AdvancedEngine) Start(ctx context.Context) error {
	e.SetRunning(true)
	return nil
}

// Stop stops the advanced engine
func (e *AdvancedEngine) Stop(ctx context.Context) error {
	e.SetRunning(false)
	return nil
}

// ProcessOrder processes an order with price improvement
func (e *AdvancedEngine) ProcessOrder(order *types.Order) (*types.Trade, error) {
	if !e.IsRunning() {
		return nil, types.NewEngineError("ENGINE_NOT_RUNNING", "Engine is not running", "", types.SeverityHigh)
	}
	
	// Advanced order processing with price improvement
	improvedPrice := order.Price
	if e.priceImprovementEnabled {
		// Apply price improvement logic
		if order.Side == types.OrderSideBuy {
			improvedPrice = order.Price * 0.999 // Slight improvement for buyer
		} else {
			improvedPrice = order.Price * 1.001 // Slight improvement for seller
		}
	}
	
	trade := &types.Trade{
		ID:          fmt.Sprintf("trade_%d", time.Now().UnixNano()),
		Symbol:      order.Symbol,
		BuyOrderID:  order.ID,
		SellOrderID: order.ID,
		Price:       improvedPrice,
		Quantity:    order.Quantity,
		Value:       improvedPrice * order.Quantity,
		Timestamp:   time.Now(),
		TakerSide:   order.Side,
	}
	
	return trade, nil
}

// HFTEngine implements a high-frequency trading optimized engine
type HFTEngine struct {
	*types.BaseEngine
	ultraLowLatency bool
}

// Start starts the HFT engine
func (e *HFTEngine) Start(ctx context.Context) error {
	e.SetRunning(true)
	return nil
}

// Stop stops the HFT engine
func (e *HFTEngine) Stop(ctx context.Context) error {
	e.SetRunning(false)
	return nil
}

// ProcessOrder processes an order with ultra-low latency
func (e *HFTEngine) ProcessOrder(order *types.Order) (*types.Trade, error) {
	if !e.IsRunning() {
		return nil, types.NewEngineError("ENGINE_NOT_RUNNING", "Engine is not running", "", types.SeverityHigh)
	}
	
	startTime := time.Now()
	
	// Ultra-fast order processing
	trade := &types.Trade{
		ID:          fmt.Sprintf("hft_trade_%d", time.Now().UnixNano()),
		Symbol:      order.Symbol,
		BuyOrderID:  order.ID,
		SellOrderID: order.ID,
		Price:       order.Price,
		Quantity:    order.Quantity,
		Value:       order.Price * order.Quantity,
		Timestamp:   time.Now(),
		TakerSide:   order.Side,
	}
	
	// Track latency for HFT
	latency := time.Since(startTime)
	stats := e.GetStats()
	if stats.MinLatency == 0 || latency < stats.MinLatency {
		stats.MinLatency = latency
		stats.MinLatencyNanos = latency.Nanoseconds()
	}
	if latency > stats.MaxLatency {
		stats.MaxLatency = latency
		stats.MaxLatencyNanos = latency.Nanoseconds()
	}
	
	return trade, nil
}

// OptimizedEngine implements an optimized engine for high throughput
type OptimizedEngine struct {
	*types.BaseEngine
	highThroughput bool
}

// Start starts the optimized engine
func (e *OptimizedEngine) Start(ctx context.Context) error {
	e.SetRunning(true)
	return nil
}

// Stop stops the optimized engine
func (e *OptimizedEngine) Stop(ctx context.Context) error {
	e.SetRunning(false)
	return nil
}

// ProcessOrder processes an order with high throughput optimization
func (e *OptimizedEngine) ProcessOrder(order *types.Order) (*types.Trade, error) {
	if !e.IsRunning() {
		return nil, types.NewEngineError("ENGINE_NOT_RUNNING", "Engine is not running", "", types.SeverityHigh)
	}
	
	// High-throughput order processing
	trade := &types.Trade{
		ID:          fmt.Sprintf("opt_trade_%d", time.Now().UnixNano()),
		Symbol:      order.Symbol,
		BuyOrderID:  order.ID,
		SellOrderID: order.ID,
		Price:       order.Price,
		Quantity:    order.Quantity,
		Value:       order.Price * order.Quantity,
		Timestamp:   time.Now(),
		TakerSide:   order.Side,
	}
	
	// Update throughput statistics
	stats := e.GetStats()
	stats.OrdersProcessed++
	stats.TradesExecuted++
	stats.TotalVolume += order.Quantity
	stats.TotalValue += trade.Value
	
	return trade, nil
}

// ComplianceEngine implements a compliance-aware matching engine
type ComplianceEngine struct {
	*types.BaseEngine
	complianceEnabled bool
}

// Start starts the compliance engine
func (e *ComplianceEngine) Start(ctx context.Context) error {
	e.SetRunning(true)
	return nil
}

// Stop stops the compliance engine
func (e *ComplianceEngine) Stop(ctx context.Context) error {
	e.SetRunning(false)
	return nil
}

// ProcessOrder processes an order with compliance checks
func (e *ComplianceEngine) ProcessOrder(order *types.Order) (*types.Trade, error) {
	if !e.IsRunning() {
		return nil, types.NewEngineError("ENGINE_NOT_RUNNING", "Engine is not running", "", types.SeverityHigh)
	}
	
	// Perform compliance checks
	if e.complianceEnabled {
		if err := e.performComplianceChecks(order); err != nil {
			return nil, err
		}
	}
	
	trade := &types.Trade{
		ID:          fmt.Sprintf("comp_trade_%d", time.Now().UnixNano()),
		Symbol:      order.Symbol,
		BuyOrderID:  order.ID,
		SellOrderID: order.ID,
		Price:       order.Price,
		Quantity:    order.Quantity,
		Value:       order.Price * order.Quantity,
		Timestamp:   time.Now(),
		TakerSide:   order.Side,
	}
	
	return trade, nil
}

// performComplianceChecks performs compliance validation on an order
func (e *ComplianceEngine) performComplianceChecks(order *types.Order) error {
	config := e.GetConfig()
	
	// Check order size limits
	if order.Quantity > config.MaxOrderSize {
		return types.NewEngineError(
			"ORDER_SIZE_EXCEEDED",
			"Order size exceeds maximum allowed",
			fmt.Sprintf("Order size: %.2f, Max allowed: %.2f", order.Quantity, config.MaxOrderSize),
			types.SeverityHigh,
		)
	}
	
	// Check price deviation limits
	if config.PriceDeviationLimit > 0 {
		// This would typically check against market price
		// For now, just validate the price is positive
		if order.Price <= 0 {
			return types.NewEngineError(
				"INVALID_PRICE",
				"Order price must be positive",
				fmt.Sprintf("Order price: %.2f", order.Price),
				types.SeverityMedium,
			)
		}
	}
	
	return nil
}
