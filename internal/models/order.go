package models

import (
	"time"
)

// Order represents a trading order
type Order struct {
	// Order ID
	ID string `json:"id"`
	
	// Symbol to trade
	Symbol string `json:"symbol"`
	
	// Order side: "buy" or "sell"
	Side string `json:"side"`
	
	// Order type: "market", "limit", "stop", "stop_limit"
	Type string `json:"type"`
	
	// Order quantity
	Quantity float64 `json:"quantity"`
	
	// Price for limit orders
	Price float64 `json:"price,omitempty"`
	
	// Stop price for stop orders
	StopPrice float64 `json:"stop_price,omitempty"`
	
	// Time in force: "gtc", "ioc", "fok"
	TimeInForce string `json:"time_in_force,omitempty"`
	
	// Order status
	Status string `json:"status"`
	
	// Filled quantity
	FilledQuantity float64 `json:"filled_quantity"`
	
	// Average fill price
	AverageFillPrice float64 `json:"average_fill_price,omitempty"`
	
	// Creation time
	CreatedAt time.Time `json:"created_at"`
	
	// Last update time
	UpdatedAt time.Time `json:"updated_at"`
	
	// Execution time
	ExecutedAt time.Time `json:"executed_at,omitempty"`
	
	// Cancellation time
	CancelledAt time.Time `json:"cancelled_at,omitempty"`
	
	// Client order ID
	ClientOrderID string `json:"client_order_id,omitempty"`
	
	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewOrder creates a new order with default values
func NewOrder(symbol, side, orderType string, quantity float64) *Order {
	now := time.Now()
	return &Order{
		Symbol:         symbol,
		Side:           side,
		Type:           orderType,
		Quantity:       quantity,
		Status:         "new",
		FilledQuantity: 0,
		CreatedAt:      now,
		UpdatedAt:      now,
		Metadata:       make(map[string]interface{}),
	}
}

// IsFilled returns true if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.Status == "filled" || o.FilledQuantity >= o.Quantity
}

// IsActive returns true if the order is still active
func (o *Order) IsActive() bool {
	return o.Status == "new" || o.Status == "partially_filled"
}

// IsCancelled returns true if the order was cancelled
func (o *Order) IsCancelled() bool {
	return o.Status == "cancelled"
}

// IsRejected returns true if the order was rejected
func (o *Order) IsRejected() bool {
	return o.Status == "rejected"
}

// RemainingQuantity returns the unfilled quantity
func (o *Order) RemainingQuantity() float64 {
	return o.Quantity - o.FilledQuantity
}

// Fill updates the order with fill information
func (o *Order) Fill(quantity, price float64) {
	o.FilledQuantity += quantity
	
	// Update average fill price
	if o.AverageFillPrice == 0 {
		o.AverageFillPrice = price
	} else {
		totalValue := o.AverageFillPrice*(o.FilledQuantity-quantity) + price*quantity
		o.AverageFillPrice = totalValue / o.FilledQuantity
	}
	
	// Update status
	if o.FilledQuantity >= o.Quantity {
		o.Status = "filled"
		o.ExecutedAt = time.Now()
	} else if o.FilledQuantity > 0 {
		o.Status = "partially_filled"
	}
	
	o.UpdatedAt = time.Now()
}

// Cancel marks the order as cancelled
func (o *Order) Cancel() {
	if o.IsActive() {
		o.Status = "cancelled"
		o.CancelledAt = time.Now()
		o.UpdatedAt = time.Now()
	}
}

// Reject marks the order as rejected
func (o *Order) Reject(reason string) {
	o.Status = "rejected"
	o.Metadata["reject_reason"] = reason
	o.UpdatedAt = time.Now()
}

