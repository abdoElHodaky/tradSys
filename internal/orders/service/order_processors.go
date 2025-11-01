package service

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// MarketOrderProcessor handles market orders
type MarketOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *MarketOrderProcessor) GetType() OrderType {
	return OrderTypeMarket
}

// Validate validates a market order
func (p *MarketOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	// Market orders don't need price validation
	return nil
}

// Process processes a market order
func (p *MarketOrderProcessor) Process(order *Order) error {
	// Market orders execute immediately at current market price
	order.Status = OrderStatusFilled
	order.UpdatedAt = time.Now()

	// Create trade record
	trade := &Trade{
		ID:                  uuid.New().String(),
		OrderID:             order.ID,
		Symbol:              order.Symbol,
		Side:                order.Side,
		Price:               order.Price, // Set by market data
		Quantity:            order.Quantity,
		ExecutedAt:          time.Now(),
		Fee:                 p.calculateFee(order),
		FeeCurrency:         "USD",
		CounterPartyOrderID: "", // Set by matching engine
	}

	order.Trades = append(order.Trades, trade)
	order.FilledQuantity = order.Quantity

	return nil
}

// CalculateExecutionPrice calculates execution price for market orders
func (p *MarketOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Market orders execute at current market price
	return marketPrice
}

// calculateFee calculates trading fee for market orders
func (p *MarketOrderProcessor) calculateFee(order *Order) float64 {
	// Market orders typically have higher fees
	return order.Price * order.Quantity * 0.001 // 0.1% fee
}

// LimitOrderProcessor handles limit orders
type LimitOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *LimitOrderProcessor) GetType() OrderType {
	return OrderTypeLimit
}

// Validate validates a limit order
func (p *LimitOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if order.Price <= 0 {
		return errors.New("price must be positive for limit orders")
	}

	return nil
}

// Process processes a limit order
func (p *LimitOrderProcessor) Process(order *Order) error {
	// Limit orders are placed in the order book
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()

	// Check if order can be immediately filled
	if p.canFillImmediately(order) {
		return p.fillOrder(order)
	}

	return nil
}

// CalculateExecutionPrice calculates execution price for limit orders
func (p *LimitOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Limit orders execute at their specified price or better
	if order.Side == OrderSideBuy {
		if marketPrice <= order.Price {
			return marketPrice
		}
	} else {
		if marketPrice >= order.Price {
			return marketPrice
		}
	}
	return order.Price
}

// canFillImmediately checks if limit order can be filled immediately
func (p *LimitOrderProcessor) canFillImmediately(order *Order) bool {
	// Simplified logic - in real implementation, check against order book
	return false
}

// fillOrder fills a limit order
func (p *LimitOrderProcessor) fillOrder(order *Order) error {
	order.Status = OrderStatusFilled
	order.FilledQuantity = order.Quantity
	order.UpdatedAt = time.Now()

	trade := &Trade{
		ID:          uuid.New().String(),
		OrderID:     order.ID,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Price:       order.Price,
		Quantity:    order.Quantity,
		ExecutedAt:  time.Now(),
		Fee:         p.calculateFee(order),
		FeeCurrency: "USD",
	}

	order.Trades = append(order.Trades, trade)
	return nil
}

// calculateFee calculates trading fee for limit orders
func (p *LimitOrderProcessor) calculateFee(order *Order) float64 {
	// Limit orders typically have lower fees
	return order.Price * order.Quantity * 0.0005 // 0.05% fee
}
