package models

import (
	"math"
	"time"
)

// Position represents a trading position
type Position struct {
	// Symbol identifier
	Symbol string `json:"symbol"`
	
	// Current quantity (positive for long, negative for short)
	Quantity float64 `json:"quantity"`
	
	// Average entry price
	EntryPrice float64 `json:"entry_price"`
	
	// Current market price
	CurrentPrice float64 `json:"current_price"`
	
	// Position value at entry
	EntryValue float64 `json:"entry_value"`
	
	// Current position value
	CurrentValue float64 `json:"current_value"`
	
	// Unrealized profit/loss
	UnrealizedPnL float64 `json:"unrealized_pnl"`
	
	// Unrealized profit/loss percentage
	UnrealizedPnLPercent float64 `json:"unrealized_pnl_percent"`
	
	// Realized profit/loss
	RealizedPnL float64 `json:"realized_pnl"`
	
	// Entry time
	EntryTime time.Time `json:"entry_time"`
	
	// Last update time
	UpdateTime time.Time `json:"update_time"`
	
	// For pair trading strategies
	Quantity1 float64 `json:"quantity1,omitempty"`
	Quantity2 float64 `json:"quantity2,omitempty"`
	Symbol1   string  `json:"symbol1,omitempty"`
	Symbol2   string  `json:"symbol2,omitempty"`
	
	// Additional metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// NewPosition creates a new position
func NewPosition(symbol string, quantity, entryPrice float64) *Position {
	now := time.Now()
	entryValue := quantity * entryPrice
	
	return &Position{
		Symbol:       symbol,
		Quantity:     quantity,
		EntryPrice:   entryPrice,
		CurrentPrice: entryPrice,
		EntryValue:   entryValue,
		CurrentValue: entryValue,
		EntryTime:    now,
		UpdateTime:   now,
		Metadata:     make(map[string]interface{}),
	}
}

// NewPairPosition creates a new pair trading position
func NewPairPosition(symbol1, symbol2 string, quantity1, quantity2, entryPrice1, entryPrice2 float64) *Position {
	now := time.Now()
	entryValue := (quantity1 * entryPrice1) + (quantity2 * entryPrice2)
	
	return &Position{
		Symbol:       symbol1 + "/" + symbol2,
		Symbol1:      symbol1,
		Symbol2:      symbol2,
		Quantity1:    quantity1,
		Quantity2:    quantity2,
		EntryPrice:   0, // Not applicable for pairs
		CurrentPrice: 0, // Not applicable for pairs
		EntryValue:   entryValue,
		CurrentValue: entryValue,
		EntryTime:    now,
		UpdateTime:   now,
		Metadata:     make(map[string]interface{}),
	}
}

// UpdatePrice updates the position with a new market price
func (p *Position) UpdatePrice(price float64) {
	p.CurrentPrice = price
	p.CurrentValue = p.Quantity * price
	p.UnrealizedPnL = p.CurrentValue - p.EntryValue
	
	if p.EntryValue != 0 {
		p.UnrealizedPnLPercent = (p.UnrealizedPnL / math.Abs(p.EntryValue)) * 100
	}
	
	p.UpdateTime = time.Now()
}

// UpdatePairPrices updates a pair position with new market prices
func (p *Position) UpdatePairPrices(price1, price2 float64) {
	currentValue := (p.Quantity1 * price1) + (p.Quantity2 * price2)
	p.CurrentValue = currentValue
	p.UnrealizedPnL = p.CurrentValue - p.EntryValue
	
	if p.EntryValue != 0 {
		p.UnrealizedPnLPercent = (p.UnrealizedPnL / math.Abs(p.EntryValue)) * 100
	}
	
	p.UpdateTime = time.Now()
}

// AddToPosition adds to an existing position
func (p *Position) AddToPosition(quantity, price float64) {
	if quantity == 0 {
		return
	}
	
	// Calculate new average entry price
	totalCost := p.EntryPrice*p.Quantity + price*quantity
	newQuantity := p.Quantity + quantity
	
	if newQuantity != 0 {
		p.EntryPrice = totalCost / newQuantity
	}
	
	p.Quantity = newQuantity
	p.EntryValue = p.Quantity * p.EntryPrice
	p.CurrentValue = p.Quantity * p.CurrentPrice
	p.UnrealizedPnL = p.CurrentValue - p.EntryValue
	
	if p.EntryValue != 0 {
		p.UnrealizedPnLPercent = (p.UnrealizedPnL / math.Abs(p.EntryValue)) * 100
	}
	
	p.UpdateTime = time.Now()
}

// ReducePosition reduces an existing position
func (p *Position) ReducePosition(quantity, price float64) float64 {
	if quantity <= 0 || p.Quantity == 0 {
		return 0
	}
	
	// Cap reduction at current position size
	if quantity > math.Abs(p.Quantity) {
		quantity = math.Abs(p.Quantity)
	}
	
	// Calculate realized P&L for this reduction
	var realizedPnL float64
	if p.Quantity > 0 {
		// Long position
		realizedPnL = (price - p.EntryPrice) * quantity
	} else {
		// Short position
		realizedPnL = (p.EntryPrice - price) * quantity
	}
	
	// Update position
	if p.Quantity > 0 {
		p.Quantity -= quantity
	} else {
		p.Quantity += quantity
	}
	
	p.RealizedPnL += realizedPnL
	p.EntryValue = p.Quantity * p.EntryPrice
	p.CurrentValue = p.Quantity * p.CurrentPrice
	p.UnrealizedPnL = p.CurrentValue - p.EntryValue
	
	if p.EntryValue != 0 {
		p.UnrealizedPnLPercent = (p.UnrealizedPnL / math.Abs(p.EntryValue)) * 100
	}
	
	p.UpdateTime = time.Now()
	
	return realizedPnL
}

// ClosePosition closes the entire position
func (p *Position) ClosePosition(price float64) float64 {
	return p.ReducePosition(math.Abs(p.Quantity), price)
}

// IsLong returns true if this is a long position
func (p *Position) IsLong() bool {
	return p.Quantity > 0
}

// IsShort returns true if this is a short position
func (p *Position) IsShort() bool {
	return p.Quantity < 0
}

// IsFlat returns true if the position is flat (no position)
func (p *Position) IsFlat() bool {
	return p.Quantity == 0
}

// Duration returns the duration the position has been open
func (p *Position) Duration() time.Duration {
	return time.Since(p.EntryTime)
}

