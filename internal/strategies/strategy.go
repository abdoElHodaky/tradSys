package strategies

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
)

// Strategy represents a trading strategy interface
type Strategy interface {
	// Name returns the strategy name
	Name() string
	
	// Execute executes the strategy and returns orders to place
	Execute(ctx context.Context, marketData *MarketData) ([]*models.Order, error)
	
	// Initialize initializes the strategy with parameters
	Initialize(params map[string]interface{}) error
	
	// Cleanup performs cleanup when strategy is stopped
	Cleanup() error
}

// MarketData represents market data for strategy execution
type MarketData struct {
	Symbol    string
	Price     float64
	Volume    float64
	Timestamp time.Time
	Bid       float64
	Ask       float64
	High      float64
	Low       float64
	Open      float64
	Close     float64
}

// Manager manages trading strategies
type Manager struct {
	strategies map[string]Strategy
	active     map[string]bool
}

// NewManager creates a new strategy manager
func NewManager() *Manager {
	return &Manager{
		strategies: make(map[string]Strategy),
		active:     make(map[string]bool),
	}
}

// RegisterStrategy registers a new strategy
func (m *Manager) RegisterStrategy(name string, strategy Strategy) error {
	m.strategies[name] = strategy
	m.active[name] = false
	return nil
}

// StartStrategy starts a strategy
func (m *Manager) StartStrategy(name string) error {
	if _, exists := m.strategies[name]; !exists {
		return ErrStrategyNotFound
	}
	m.active[name] = true
	return nil
}

// StopStrategy stops a strategy
func (m *Manager) StopStrategy(name string) error {
	if _, exists := m.strategies[name]; !exists {
		return ErrStrategyNotFound
	}
	m.active[name] = false
	return nil
}

// ExecuteStrategies executes all active strategies
func (m *Manager) ExecuteStrategies(ctx context.Context, marketData *MarketData) ([]*models.Order, error) {
	var allOrders []*models.Order
	
	for name, strategy := range m.strategies {
		if !m.active[name] {
			continue
		}
		
		orders, err := strategy.Execute(ctx, marketData)
		if err != nil {
			// Log error but continue with other strategies
			continue
		}
		
		allOrders = append(allOrders, orders...)
	}
	
	return allOrders, nil
}

// GetActiveStrategies returns list of active strategy names
func (m *Manager) GetActiveStrategies() []string {
	var active []string
	for name, isActive := range m.active {
		if isActive {
			active = append(active, name)
		}
	}
	return active
}

// Errors
var (
	ErrStrategyNotFound = fmt.Errorf("strategy not found")
)
