package optimization

import (
	"context"
	"errors"
	"math"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/strategy"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrEvaluationFailed = errors.New("strategy evaluation failed")
)

// StrategyEvaluator evaluates strategies
type StrategyEvaluator struct {
	// Factory is the strategy factory
	factory strategy.StrategyFactory

	// Logger
	logger *zap.Logger

	// Backtester is the backtester
	backtester *Backtester
}

// NewStrategyEvaluator creates a new StrategyEvaluator
func NewStrategyEvaluator(
	factory strategy.StrategyFactory,
	backtester *Backtester,
	logger *zap.Logger,
) *StrategyEvaluator {
	return &StrategyEvaluator{
		factory:    factory,
		backtester: backtester,
		logger:     logger,
	}
}

// Evaluate evaluates a strategy
func (e *StrategyEvaluator) Evaluate(
	ctx context.Context,
	config strategy.StrategyConfig,
) (map[string]float64, error) {
	// Create the strategy
	s, err := e.factory.CreateStrategy(config)
	if err != nil {
		return nil, err
	}

	// Run backtest
	result, err := e.backtester.Backtest(ctx, s)
	if err != nil {
		return nil, err
	}

	// Calculate metrics
	metrics := e.calculateMetrics(result)

	return metrics, nil
}

// calculateMetrics calculates performance metrics
func (e *StrategyEvaluator) calculateMetrics(result *BacktestResult) map[string]float64 {
	metrics := make(map[string]float64)

	// Total return
	metrics["total_return"] = result.TotalPnL

	// Annualized return
	if result.Duration > 0 {
		years := float64(result.Duration) / float64(365*24*time.Hour)
		if years > 0 {
			metrics["annualized_return"] = math.Pow(1+result.TotalPnL/result.InitialCapital, 1/years) - 1
		}
	}

	// Max drawdown
	metrics["max_drawdown"] = result.MaxDrawdown

	// Sharpe ratio
	if result.StdDevReturns > 0 {
		metrics["sharpe_ratio"] = result.AvgDailyReturn / result.StdDevReturns * math.Sqrt(252)
	}

	// Sortino ratio
	if result.DownsideDeviation > 0 {
		metrics["sortino_ratio"] = result.AvgDailyReturn / result.DownsideDeviation * math.Sqrt(252)
	}

	// Win rate
	if result.TotalTrades > 0 {
		metrics["win_rate"] = float64(result.WinningTrades) / float64(result.TotalTrades)
	}

	// Profit factor
	if result.GrossLoss != 0 {
		metrics["profit_factor"] = math.Abs(result.GrossProfit / result.GrossLoss)
	}

	// Average trade
	if result.TotalTrades > 0 {
		metrics["avg_trade"] = result.TotalPnL / float64(result.TotalTrades)
	}

	// Average winning trade
	if result.WinningTrades > 0 {
		metrics["avg_winning_trade"] = result.GrossProfit / float64(result.WinningTrades)
	}

	// Average losing trade
	if result.LosingTrades > 0 {
		metrics["avg_losing_trade"] = result.GrossLoss / float64(result.LosingTrades)
	}

	// Maximum consecutive wins
	metrics["max_consecutive_wins"] = float64(result.MaxConsecutiveWins)

	// Maximum consecutive losses
	metrics["max_consecutive_losses"] = float64(result.MaxConsecutiveLosses)

	// Calmar ratio
	if result.MaxDrawdown > 0 {
		metrics["calmar_ratio"] = metrics["annualized_return"] / result.MaxDrawdown
	}

	return metrics
}

// Backtester performs backtesting
type Backtester struct {
	// Logger
	logger *zap.Logger
}

// NewBacktester creates a new Backtester
func NewBacktester(logger *zap.Logger) *Backtester {
	return &Backtester{
		logger: logger,
	}
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	// Strategy is the strategy name
	Strategy string

	// InitialCapital is the initial capital
	InitialCapital float64

	// FinalCapital is the final capital
	FinalCapital float64

	// TotalPnL is the total profit and loss
	TotalPnL float64

	// MaxDrawdown is the maximum drawdown
	MaxDrawdown float64

	// AvgDailyReturn is the average daily return
	AvgDailyReturn float64

	// StdDevReturns is the standard deviation of returns
	StdDevReturns float64

	// DownsideDeviation is the downside deviation
	DownsideDeviation float64

	// TotalTrades is the total number of trades
	TotalTrades int

	// WinningTrades is the number of winning trades
	WinningTrades int

	// LosingTrades is the number of losing trades
	LosingTrades int

	// GrossProfit is the gross profit
	GrossProfit float64

	// GrossLoss is the gross loss
	GrossLoss float64

	// MaxConsecutiveWins is the maximum consecutive wins
	MaxConsecutiveWins int

	// MaxConsecutiveLosses is the maximum consecutive losses
	MaxConsecutiveLosses int

	// Duration is the duration of the backtest
	Duration time.Duration

	// DailyReturns are the daily returns
	DailyReturns []float64

	// Trades are the trades
	Trades []*BacktestTrade
}

// BacktestTrade represents a trade in a backtest
type BacktestTrade struct {
	// Symbol is the trading symbol
	Symbol string

	// Side is the side of the trade
	Side string

	// EntryTime is the entry time
	EntryTime time.Time

	// EntryPrice is the entry price
	EntryPrice float64

	// ExitTime is the exit time
	ExitTime time.Time

	// ExitPrice is the exit price
	ExitPrice float64

	// Quantity is the quantity
	Quantity float64

	// PnL is the profit and loss
	PnL float64
}

// Backtest performs a backtest
func (b *Backtester) Backtest(
	ctx context.Context,
	s strategy.Strategy,
) (*BacktestResult, error) {
	// In a real implementation, this would run a backtest using historical data
	// For now, we'll return a dummy result
	
	// This is a placeholder implementation
	// In a real system, you would:
	// 1. Load historical data
	// 2. Initialize the strategy
	// 3. Run the strategy on the historical data
	// 4. Track trades and performance
	// 5. Calculate performance metrics
	
	result := &BacktestResult{
		Strategy:            s.Name(),
		InitialCapital:      10000,
		FinalCapital:        11000,
		TotalPnL:            1000,
		MaxDrawdown:         0.05,
		AvgDailyReturn:      0.001,
		StdDevReturns:       0.01,
		DownsideDeviation:   0.008,
		TotalTrades:         50,
		WinningTrades:       30,
		LosingTrades:        20,
		GrossProfit:         1500,
		GrossLoss:           -500,
		MaxConsecutiveWins:  5,
		MaxConsecutiveLosses: 3,
		Duration:            30 * 24 * time.Hour,
		DailyReturns:        make([]float64, 30),
		Trades:              make([]*BacktestTrade, 0),
	}

	return result, nil
}

