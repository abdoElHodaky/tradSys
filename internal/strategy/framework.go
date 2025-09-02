package strategy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/models"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// LegacyStrategy defines the interface for trading strategies (legacy version)
// This is kept for backward compatibility with existing code
type LegacyStrategy interface {
	// GetName returns the strategy name
	GetName() string
	
	// GetDescription returns the strategy description
	GetDescription() string
	
	// GetSymbols returns the symbols this strategy trades
	GetSymbols() []string
	
	// Initialize prepares the strategy for trading
	Initialize(ctx context.Context) error
	
	// OnMarketData processes new market data
	OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error
	
	// OnOrderUpdate processes order updates
	OnOrderUpdate(ctx context.Context, order *models.Order) error
	
	// Shutdown cleans up resources
	Shutdown(ctx context.Context) error
}

// BaseStrategy provides common functionality for strategies
type BaseStrategy struct {
	// Strategy metadata
	name        string
	description string
	symbols     map[string]bool
	active      bool
	
	// Logger
	logger *zap.Logger
}

// GetName returns the strategy name
func (s *BaseStrategy) GetName() string {
	return s.name
}

// GetDescription returns the strategy description
func (s *BaseStrategy) GetDescription() string {
	return s.description
}

// GetSymbols returns the symbols this strategy trades
func (s *BaseStrategy) GetSymbols() []string {
	symbols := make([]string, 0, len(s.symbols))
	for symbol := range s.symbols {
		symbols = append(symbols, symbol)
	}
	return symbols
}

// OnOrderUpdate processes order updates
func (s *BaseStrategy) OnOrderUpdate(ctx context.Context, order *models.Order) error {
	// Default implementation does nothing
	return nil
}

// StrategyManager manages multiple trading strategies
type StrategyManager struct {
	// Registered strategies
	strategies map[string]LegacyStrategy
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Logger
	logger *zap.Logger
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager(logger *zap.Logger) *StrategyManager {
	if logger == nil {
		logger = zap.NewNop()
	}
	
	return &StrategyManager{
		strategies: make(map[string]LegacyStrategy),
		logger:     logger,
	}
}

// RegisterStrategy adds a strategy to the manager
func (m *StrategyManager) RegisterStrategy(strategy LegacyStrategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	name := strategy.GetName()
	if _, exists := m.strategies[name]; exists {
		return fmt.Errorf("strategy with name '%s' already registered", name)
	}
	
	m.strategies[name] = strategy
	m.logger.Info("Registered strategy", zap.String("name", name))
	return nil
}

// UnregisterStrategy removes a strategy from the manager
func (m *StrategyManager) UnregisterStrategy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if _, exists := m.strategies[name]; !exists {
		return fmt.Errorf("strategy with name '%s' not found", name)
	}
	
	delete(m.strategies, name)
	m.logger.Info("Unregistered strategy", zap.String("name", name))
	return nil
}

// GetStrategy returns a strategy by name
func (m *StrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	strategy, exists := m.strategies[name]
	if !exists {
		return nil, fmt.Errorf("strategy with name '%s' not found", name)
	}
	
	return strategy, nil
}

// GetAllStrategies returns all registered strategies
func (m *StrategyManager) GetAllStrategies() []Strategy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	strategies := make([]Strategy, 0, len(m.strategies))
	for _, strategy := range m.strategies {
		strategies = append(strategies, strategy)
	}
	
	return strategies
}

// InitializeAll initializes all registered strategies
func (m *StrategyManager) InitializeAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	for name, strategy := range m.strategies {
		if err := strategy.Initialize(ctx); err != nil {
			m.logger.Error("Failed to initialize strategy",
				zap.String("name", name),
				zap.Error(err))
			return fmt.Errorf("failed to initialize strategy '%s': %w", name, err)
		}
	}
	
	return nil
}

// ShutdownAll shuts down all registered strategies
func (m *StrategyManager) ShutdownAll(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var lastErr error
	for name, strategy := range m.strategies {
		if err := strategy.Shutdown(ctx); err != nil {
			m.logger.Error("Failed to shutdown strategy",
				zap.String("name", name),
				zap.Error(err))
			lastErr = err
		}
	}
	
	return lastErr
}

// ProcessMarketData distributes market data to relevant strategies
func (m *StrategyManager) ProcessMarketData(ctx context.Context, data *marketdata.MarketDataResponse) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	symbol := data.Symbol
	
	// Find strategies that trade this symbol
	for _, strategy := range m.strategies {
		for _, s := range strategy.GetSymbols() {
			if s == symbol {
				// Process in a goroutine to avoid blocking
				go func(s Strategy, d *marketdata.MarketDataResponse) {
					if err := s.OnMarketData(ctx, d); err != nil {
						m.logger.Error("Strategy failed to process market data",
							zap.String("name", s.GetName()),
							zap.String("symbol", d.Symbol),
							zap.Error(err))
					}
				}(strategy, data)
				break
			}
		}
	}
}

// ProcessOrderUpdate distributes order updates to all strategies
func (m *StrategyManager) ProcessOrderUpdate(ctx context.Context, order *models.Order) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	// Find strategies that trade this symbol
	for _, strategy := range m.strategies {
		for _, s := range strategy.GetSymbols() {
			if s == order.Symbol {
				if err := strategy.OnOrderUpdate(ctx, order); err != nil {
					m.logger.Error("Strategy failed to process order update",
						zap.String("name", strategy.GetName()),
						zap.String("symbol", order.Symbol),
						zap.String("orderId", order.ID),
						zap.Error(err))
				}
				break
			}
		}
	}
}

// OnMarketData processes new market data
func (s *BaseStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	// Default implementation does nothing
	return nil
}

// Initialize prepares the strategy for trading
func (s *BaseStrategy) Initialize(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// Shutdown cleans up resources
func (s *BaseStrategy) Shutdown(ctx context.Context) error {
	// Default implementation does nothing
	return nil
}

// GetPosition gets the current position for a symbol
func (s *BaseStrategy) GetPosition(ctx context.Context, symbol string) (*models.Position, error) {
	// This is a placeholder - in a real implementation, this would query the broker
	return nil, errors.New("not implemented")
}

// SubmitOrder submits an order to the broker
func (s *BaseStrategy) SubmitOrder(ctx context.Context, order *models.Order) error {
	// This is a placeholder - in a real implementation, this would submit to the broker
	return errors.New("not implemented")
}

// StatisticalArbitrageParams contains parameters for the statistical arbitrage strategy
type StatisticalArbitrageParams struct {
	// Pair symbols
	Symbol1 string
	Symbol2 string
	
	// Entry/exit thresholds in standard deviations
	EntryThreshold float64
	ExitThreshold  float64
	
	// Position sizing
	MaxPosition float64
	
	// Risk management
	StopLoss       float64
	TakeProfit     float64
	MaxHoldingTime time.Duration
	
	// Lookback period for calculating statistics
	LookbackPeriod int
	
	// Minimum number of data points required before trading
	MinDataPoints int
	
	// Rebalancing frequency
	RebalanceInterval time.Duration
	
	// Execution parameters
	ExecutionDelay time.Duration
}
