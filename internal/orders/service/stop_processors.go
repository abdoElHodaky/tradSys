package service

import (
	"errors"
	"fmt"
	"time"
)

// StopLimitOrderProcessor handles stop-limit orders
type StopLimitOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *StopLimitOrderProcessor) GetType() OrderType {
	return OrderTypeStopLimit
}

// Validate validates a stop-limit order
func (p *StopLimitOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if order.Price <= 0 {
		return errors.New("price must be positive for stop-limit orders")
	}

	if order.StopPrice <= 0 {
		return errors.New("stop price must be positive for stop-limit orders")
	}

	return nil
}

// Process processes a stop-limit order
func (p *StopLimitOrderProcessor) Process(order *Order) error {
	// Stop-limit orders wait for trigger condition
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()

	// In real implementation, monitor market price for trigger
	return nil
}

// CalculateExecutionPrice calculates execution price for stop-limit orders
func (p *StopLimitOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Stop-limit orders execute at limit price after trigger
	return order.Price
}

// StopMarketOrderProcessor handles stop-market orders
type StopMarketOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *StopMarketOrderProcessor) GetType() OrderType {
	return OrderTypeStopMarket
}

// Validate validates a stop-market order
func (p *StopMarketOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}

	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}

	if order.StopPrice <= 0 {
		return errors.New("stop price must be positive for stop-market orders")
	}

	return nil
}

// Process processes a stop-market order
func (p *StopMarketOrderProcessor) Process(order *Order) error {
	// Stop-market orders wait for trigger condition
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()

	return nil
}

// CalculateExecutionPrice calculates execution price for stop-market orders
func (p *StopMarketOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Stop-market orders execute at market price after trigger
	return marketPrice
}

// DefaultOrderProcessor handles unknown order types
type DefaultOrderProcessor struct{}

// GetType returns the default order type
func (p *DefaultOrderProcessor) GetType() OrderType {
	return ""
}

// Validate validates unknown order types
func (p *DefaultOrderProcessor) Validate(order *Order) error {
	return fmt.Errorf("unsupported order type: %s", order.Type)
}

// Process processes unknown order types
func (p *DefaultOrderProcessor) Process(order *Order) error {
	return fmt.Errorf("cannot process unsupported order type: %s", order.Type)
}

// CalculateExecutionPrice calculates execution price for unknown order types
func (p *DefaultOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	return 0
}
