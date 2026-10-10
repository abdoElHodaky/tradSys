// Package assets provides position management for TradSys v3
package assets

import (
	"sync"
	"time"
)

// Position represents a trading position
type Position struct {
	Symbol       string
	Quantity     float64
	AveragePrice float64
	Exchange     string
	LastUpdated  time.Time
}

// PositionManager manages positions across exchanges
type PositionManager struct {
	positions map[string]*Position
	mu        sync.RWMutex
}

// NewPositionManager creates a new position manager
func NewPositionManager() *PositionManager {
	return &PositionManager{
		positions: make(map[string]*Position),
	}
}

// AddPosition adds a position
func (pm *PositionManager) AddPosition(symbol string, position *Position) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	pm.positions[symbol] = position
}

// GetPosition retrieves a position
func (pm *PositionManager) GetPosition(symbol string) *Position {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.positions[symbol]
}