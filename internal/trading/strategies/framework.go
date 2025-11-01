package strategies

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
)

// Strategy represents a trading strategy
type Strategy interface {
	// Initialize initializes the strategy
	Initialize(ctx context.Context) error

	// Start starts the strategy
	Start(ctx context.Context) error

	// Stop stops the strategy
	Stop(ctx context.Context) error

	// OnMarketData processes market data updates
	OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error

	// OnOrderUpdate processes order updates
	OnOrderUpdate(ctx context.Context, order *services.Order) error

	// GetName returns the name of the strategy
	GetName() string

	// GetParameters returns the strategy parameters
	GetParameters() map[string]interface{}

	// SetParameters sets the strategy parameters
	SetParameters(params map[string]interface{}) error
}

// StrategyManager manages trading strategies
type StrategyManager struct {
	logger       *zap.Logger
	strategies   map[string]Strategy
	running      map[string]bool
	mu           sync.RWMutex
	orderService services.OrderService
	pairRepo     *repositories.PairRepository
	statsRepo    *repositories.PairStatisticsRepository
	positionRepo *repositories.PairPositionRepository
}

// NewStrategyManager creates a new strategy manager
func NewStrategyManager(
	logger *zap.Logger,
	orderService services.OrderService,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) *StrategyManager {
	return &StrategyManager{
		logger:       logger,
		strategies:   make(map[string]Strategy),
		running:      make(map[string]bool),
		orderService: orderService,
		pairRepo:     pairRepo,
		statsRepo:    statsRepo,
		positionRepo: positionRepo,
	}
}

// RegisterStrategy registers a strategy
func (m *StrategyManager) RegisterStrategy(strategy Strategy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	name := strategy.GetName()
	if _, exists := m.strategies[name]; exists {
		return ErrStrategyAlreadyRegistered
	}

	m.strategies[name] = strategy
	m.running[name] = false

	m.logger.Info("Strategy registered", zap.String("name", name))

	return nil
}

// UnregisterStrategy unregisters a strategy
func (m *StrategyManager) UnregisterStrategy(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.strategies[name]; !exists {
		return ErrStrategyNotFound
	}

	// Stop the strategy if it's running
	if m.running[name] {
		m.mu.Unlock()
		if err := m.StopStrategy(context.Background(), name); err != nil {
			m.mu.Lock()
			return err
		}
		m.mu.Lock()
	}

	delete(m.strategies, name)
	delete(m.running, name)

	m.logger.Info("Strategy unregistered", zap.String("name", name))

	return nil
}

// StartStrategy starts a strategy
func (m *StrategyManager) StartStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}

	if m.running[name] {
		return ErrStrategyAlreadyRunning
	}

	if err := strategy.Start(ctx); err != nil {
		return err
	}

	m.running[name] = true

	m.logger.Info("Strategy started", zap.String("name", name))

	return nil
}

// StopStrategy stops a strategy
func (m *StrategyManager) StopStrategy(ctx context.Context, name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return ErrStrategyNotFound
	}

	if !m.running[name] {
		return ErrStrategyNotRunning
	}

	if err := strategy.Stop(ctx); err != nil {
		return err
	}

	m.running[name] = false

	m.logger.Info("Strategy stopped", zap.String("name", name))

	return nil
}

// GetStrategy returns a strategy
func (m *StrategyManager) GetStrategy(name string) (Strategy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	strategy, exists := m.strategies[name]
	if !exists {
		return nil, ErrStrategyNotFound
	}

	return strategy, nil
}

// ListStrategies returns a list of registered strategies
func (m *StrategyManager) ListStrategies() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var strategies []string
	for name := range m.strategies {
		strategies = append(strategies, name)
	}

	return strategies
}

// IsStrategyRunning checks if a strategy is running
func (m *StrategyManager) IsStrategyRunning(name string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if _, exists := m.strategies[name]; !exists {
		return false, ErrStrategyNotFound
	}

	return m.running[name], nil
}

// <<<<<< codegen-bot/fix-order-model-syntax
// CreatePairsStrategy creates a new statistical arbitrage strategy
func (m *StrategyManager) CreatePairsStrategy(ctx context.Context, params StatisticalArbitrageParams) (Strategy, error) {
	// Create a new statistical arbitrage strategy
	strategy := NewStatisticalArbitrageStrategy(
		m.logger,
		params,
		m.orderService,
		m.pairRepo,
		m.statsRepo,
		m.positionRepo,
	)

	// Register the strategy
	if err := m.RegisterStrategy(strategy); err != nil {
		return nil, err
	}

	// Initialize the strategy
	if err := strategy.Initialize(ctx); err != nil {
		// Unregister the strategy if initialization fails
		m.UnregisterStrategy(strategy.GetName())
		return nil, err
	}

	return strategy, nil
}

// ProcessMarketData processes market data updates for all running strategies
func (m *StrategyManager) ProcessMarketData(ctx context.Context, data *marketdata.MarketDataResponse) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, strategy := range m.strategies {
		if m.running[name] {
			go func(s Strategy, d *marketdata.MarketDataResponse) {
				if err := s.OnMarketData(ctx, d); err != nil {
					m.logger.Error("Failed to process market data",
						zap.Error(err),
						zap.String("strategy", s.GetName()))
				}
			}(strategy, data)
		}
	}
}

// ProcessOrderUpdate processes order updates for all running strategies
func (m *StrategyManager) ProcessOrderUpdate(ctx context.Context, order *services.Order) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for name, strategy := range m.strategies {
		if m.running[name] {
			go func(s Strategy, o *services.Order) {
				if err := s.OnOrderUpdate(ctx, o); err != nil {
					m.logger.Error("Failed to process order update",
						zap.Error(err),
						zap.String("strategy", s.GetName()))
				}
			}(strategy, order)
		}
	}
}

// BaseStrategy provides a base implementation for strategies
type BaseStrategy struct {
	name       string
	logger     *zap.Logger
	parameters map[string]interface{}
	running    bool
	mu         sync.RWMutex
}

// NewBaseStrategy creates a new base strategy
func NewBaseStrategy(name string, logger *zap.Logger) *BaseStrategy {
	return &BaseStrategy{
		name:       name,
		logger:     logger,
		parameters: make(map[string]interface{}),
		running:    false,
	}
}

// Initialize initializes the strategy
func (s *BaseStrategy) Initialize(ctx context.Context) error {
	return nil
}

// Start starts the strategy
func (s *BaseStrategy) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		return ErrStrategyAlreadyRunning
	}

	s.running = true

	return nil
}

// Stop stops the strategy
func (s *BaseStrategy) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.running {
		return ErrStrategyNotRunning
	}

	s.running = false

	return nil
}

// OnMarketData processes market data updates
func (s *BaseStrategy) OnMarketData(ctx context.Context, data *marketdata.MarketDataResponse) error {
	// To be implemented by derived strategies
	return nil
}

// OnOrderUpdate processes order updates
func (s *BaseStrategy) OnOrderUpdate(ctx context.Context, order *services.Order) error {
	// To be implemented by derived strategies
	return nil
}

// GetName returns the name of the strategy
func (s *BaseStrategy) GetName() string {
	return s.name
}

// GetParameters returns the strategy parameters
func (s *BaseStrategy) GetParameters() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Create a copy to avoid race conditions
	params := make(map[string]interface{})
	for k, v := range s.parameters {
		params[k] = v
	}

	return params
}

// SetParameters sets the strategy parameters
func (s *BaseStrategy) SetParameters(params map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Validate parameters
	for k, v := range params {
		// Add validation logic here if needed
		s.parameters[k] = v
	}

	return nil
}

// IsRunning checks if the strategy is running
func (s *BaseStrategy) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return s.running
}

// StrategyResult represents the result of a strategy execution
type StrategyResult struct {
	Strategy   string
	Symbol     string
	Action     string
	Quantity   float64
	Price      float64
	Timestamp  time.Time
	Parameters map[string]interface{}
	Metrics    map[string]float64
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	Strategy       string
	StartTime      time.Time
	EndTime        time.Time
	Symbols        []string
	InitialCapital float64
	FinalCapital   float64
	PnL            float64
	Trades         []models.Trade
	Metrics        map[string]float64
}

// StatisticalArbitrageParams contains parameters for the statistical arbitrage strategy
type StatisticalArbitrageParams struct {
	Name           string
	PairID         string
	Symbol1        string
	Symbol2        string
	Ratio          float64
	ZScoreEntry    float64
	ZScoreExit     float64
	PositionSize   float64
	MaxPositions   int
	LookbackPeriod int
	UpdateInterval time.Duration
}

// Errors
var (
	ErrStrategyNotFound          = NewError("strategy not found")
	ErrStrategyAlreadyRegistered = NewError("strategy already registered")
	ErrStrategyAlreadyRunning    = NewError("strategy already running")
	ErrStrategyNotRunning        = NewError("strategy not running")
	ErrInvalidParameters         = NewError("invalid parameters")
)

// Error represents a strategy error
type Error struct {
	message string
}

// NewError creates a new error
func NewError(message string) *Error {
	return &Error{message: message}
}

// Error returns the error message
func (e *Error) Error() string {
	return e.message
}
