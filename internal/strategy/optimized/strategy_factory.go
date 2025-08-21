package optimized

import (
	"context"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/resilience"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/workerpool"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/markcheno/go-talib"
	"go.uber.org/zap"
	"gonum.org/v1/gonum/stat"
)

// StrategyType represents the type of strategy
type StrategyType string

// Strategy types
const (
	MomentumStrategy          StrategyType = "momentum"
	MeanReversionStrategy     StrategyType = "mean_reversion"
	StatisticalArbitrageStrategy StrategyType = "statistical_arbitrage"
)

// StrategyParams contains common parameters for all strategies
type StrategyParams struct {
	Name           string
	Symbols        []string
	LookbackPeriod int
	UpdateInterval int // in seconds
}

// MomentumParams contains parameters for momentum strategies
type MomentumParams struct {
	StrategyParams
	FastPeriod  int
	SlowPeriod  int
	SignalPeriod int
	Threshold   float64
}

// MeanReversionParams contains parameters for mean reversion strategies
type MeanReversionParams struct {
	StrategyParams
	StdDevPeriod int
	EntryThreshold float64
	ExitThreshold  float64
}

// StatisticalArbitrageParams contains parameters for statistical arbitrage strategies
type StatisticalArbitrageParams struct {
	StrategyParams
	Symbol1       string
	Symbol2       string
	Ratio         float64
	ZScoreEntry   float64
	ZScoreExit    float64
	PositionSize  float64
	MaxPositions  int
}

// StrategyFactory creates optimized trading strategies
type StrategyFactory struct {
	logger         *zap.Logger
	workerPool     *workerpool.WorkerPoolFactory
	circuitBreaker *resilience.CircuitBreakerFactory
	metrics        *StrategyMetrics
	
	// Repositories
	pairRepo       *repositories.PairRepository
	statsRepo      *repositories.PairStatisticsRepository
	positionRepo   *repositories.PairPositionRepository
	
	// Object pools
	priceSeriesPool *sync.Pool
	
	// Strategy instances
	strategies     map[string]Strategy
	mu             sync.RWMutex
}

// NewStrategyFactory creates a new strategy factory
func NewStrategyFactory(
	logger *zap.Logger,
	workerPool *workerpool.WorkerPoolFactory,
	circuitBreaker *resilience.CircuitBreakerFactory,
	metrics *StrategyMetrics,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) *StrategyFactory {
	return &StrategyFactory{
		logger:         logger,
		workerPool:     workerPool,
		circuitBreaker: circuitBreaker,
		metrics:        metrics,
		pairRepo:       pairRepo,
		statsRepo:      statsRepo,
		positionRepo:   positionRepo,
		priceSeriesPool: &sync.Pool{
			New: func() interface{} {
				return make([]float64, 0, 1000)
			},
		},
		strategies:     make(map[string]Strategy),
	}
}

// CreateStrategy creates a new strategy of the specified type
func (f *StrategyFactory) CreateStrategy(ctx context.Context, strategyType StrategyType, params interface{}) (Strategy, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	
	var strategy Strategy
	var err error
	
	switch strategyType {
	case MomentumStrategy:
		momentumParams, ok := params.(MomentumParams)
		if !ok {
			return nil, errors.New("invalid parameters for momentum strategy")
		}
		strategy, err = f.createMomentumStrategy(ctx, momentumParams)
	
	case MeanReversionStrategy:
		meanReversionParams, ok := params.(MeanReversionParams)
		if !ok {
			return nil, errors.New("invalid parameters for mean reversion strategy")
		}
		strategy, err = f.createMeanReversionStrategy(ctx, meanReversionParams)
	
	case StatisticalArbitrageStrategy:
		statisticalArbitrageParams, ok := params.(StatisticalArbitrageParams)
		if !ok {
			return nil, errors.New("invalid parameters for statistical arbitrage strategy")
		}
		strategy, err = f.createStatisticalArbitrageStrategy(ctx, statisticalArbitrageParams)
	
	default:
		return nil, errors.New("unknown strategy type")
	}
	
	if err != nil {
		return nil, err
	}
	
	// Initialize the strategy
	if err := strategy.Initialize(ctx); err != nil {
		return nil, err
	}
	
	// Store the strategy
	f.strategies[strategy.GetName()] = strategy
	
	return strategy, nil
}

// GetStrategy returns a strategy by name
func (f *StrategyFactory) GetStrategy(name string) (Strategy, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	
	strategy, exists := f.strategies[name]
	return strategy, exists
}

// createMomentumStrategy creates a new momentum strategy
func (f *StrategyFactory) createMomentumStrategy(ctx context.Context, params MomentumParams) (Strategy, error) {
	// Create a new optimized momentum strategy
	strategy := &MomentumStrategy{
		BaseStrategy: &BaseStrategy{
			name:      params.Name,
			logger:    f.logger,
			isRunning: false,
		},
		symbols:       params.Symbols,
		lookbackPeriod: params.LookbackPeriod,
		updateInterval: params.UpdateInterval,
		fastPeriod:    params.FastPeriod,
		slowPeriod:    params.SlowPeriod,
		signalPeriod:  params.SignalPeriod,
		threshold:     params.Threshold,
		prices:        make(map[string][]float64),
		signals:       make(map[string]float64),
		positions:     make(map[string]float64),
		workerPool:    f.workerPool,
		metrics:       f.metrics,
	}
	
	return strategy, nil
}

// createMeanReversionStrategy creates a new mean reversion strategy
func (f *StrategyFactory) createMeanReversionStrategy(ctx context.Context, params MeanReversionParams) (Strategy, error) {
	// Create a new optimized mean reversion strategy
	strategy := &MeanReversionStrategy{
		BaseStrategy: &BaseStrategy{
			name:      params.Name,
			logger:    f.logger,
			isRunning: false,
		},
		symbols:        params.Symbols,
		lookbackPeriod: params.LookbackPeriod,
		updateInterval: params.UpdateInterval,
		stdDevPeriod:   params.StdDevPeriod,
		entryThreshold: params.EntryThreshold,
		exitThreshold:  params.ExitThreshold,
		prices:         make(map[string][]float64),
		zScores:        make(map[string]float64),
		positions:      make(map[string]float64),
		workerPool:     f.workerPool,
		metrics:        f.metrics,
	}
	
	return strategy, nil
}

// createStatisticalArbitrageStrategy creates a new statistical arbitrage strategy
func (f *StrategyFactory) createStatisticalArbitrageStrategy(ctx context.Context, params StatisticalArbitrageParams) (Strategy, error) {
	// Create a new optimized statistical arbitrage strategy
	strategy := &StatisticalArbitrageStrategy{
		BaseStrategy: &BaseStrategy{
			name:      params.Name,
			logger:    f.logger,
			isRunning: false,
		},
		symbol1:        params.Symbol1,
		symbol2:        params.Symbol2,
		ratio:          params.Ratio,
		zScoreEntry:    params.ZScoreEntry,
		zScoreExit:     params.ZScoreExit,
		positionSize:   params.PositionSize,
		maxPositions:   params.MaxPositions,
		lookbackPeriod: params.LookbackPeriod,
		updateInterval: params.UpdateInterval,
		prices1:        make([]float64, 0, params.LookbackPeriod),
		prices2:        make([]float64, 0, params.LookbackPeriod),
		positions:      make(map[string]*models.PairPosition),
		pairRepo:       f.pairRepo,
		statsRepo:      f.statsRepo,
		positionRepo:   f.positionRepo,
		workerPool:     f.workerPool,
		metrics:        f.metrics,
	}
	
	return strategy, nil
}

// BaseStrategy provides common functionality for all strategies
type BaseStrategy struct {
	name      string
	logger    *zap.Logger
	isRunning bool
	mu        sync.RWMutex
}

// GetName returns the name of the strategy
func (s *BaseStrategy) GetName() string {
	return s.name
}

// Initialize initializes the strategy
func (s *BaseStrategy) Initialize(ctx context.Context) error {
	s.logger.Info("Initializing strategy", zap.String("name", s.name))
	return nil
}

// Start starts the strategy
func (s *BaseStrategy) Start(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.isRunning {
		return ErrStrategyAlreadyRunning
	}
	
	s.isRunning = true
	s.logger.Info("Strategy started", zap.String("name", s.name))
	
	return nil
}

// Stop stops the strategy
func (s *BaseStrategy) Stop(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if !s.isRunning {
		return ErrStrategyNotRunning
	}
	
	s.isRunning = false
	s.logger.Info("Strategy stopped", zap.String("name", s.name))
	
	return nil
}

// IsRunning returns whether the strategy is running
func (s *BaseStrategy) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	return s.isRunning
}

// CalculateZScore calculates the z-score of a value using gonum/stat
func CalculateZScore(value float64, data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	
	mean, stdDev := stat.MeanStdDev(data, nil)
	if stdDev == 0 {
		return 0
	}
	
	return (value - mean) / stdDev
}

// CalculateMACD calculates MACD using go-talib
func CalculateMACD(prices []float64, fastPeriod, slowPeriod, signalPeriod int) ([]float64, []float64, []float64) {
	return talib.Macd(prices, fastPeriod, slowPeriod, signalPeriod)
}

// CalculateRSI calculates RSI using go-talib
func CalculateRSI(prices []float64, period int) []float64 {
	return talib.Rsi(prices, period)
}

// CalculateBollingerBands calculates Bollinger Bands using go-talib
func CalculateBollingerBands(prices []float64, period int, devUp, devDown float64) ([]float64, []float64, []float64) {
	return talib.BBands(prices, period, devUp, devDown, 0)
}

// CalculateEMA calculates EMA using go-talib
func CalculateEMA(prices []float64, period int) []float64 {
	return talib.Ema(prices, period)
}

// CalculateSMA calculates SMA using go-talib
func CalculateSMA(prices []float64, period int) []float64 {
	return talib.Sma(prices, period)
}

