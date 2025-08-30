package decisionsupport

import (
	"context"
	"time"
)

// AnalysisRequest represents a request for decision support analysis
type AnalysisRequest struct {
	// Symbol is the trading symbol to analyze
	Symbol string `json:"symbol"`
	
	// Timeframe is the timeframe for analysis (e.g., "1h", "1d")
	Timeframe string `json:"timeframe"`
	
	// StartTime is the start time for historical data
	StartTime time.Time `json:"start_time"`
	
	// EndTime is the end time for historical data
	EndTime time.Time `json:"end_time"`
	
	// Indicators is a list of technical indicators to include
	Indicators []string `json:"indicators"`
	
	// Parameters is a map of additional parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// Recommendation represents a trading recommendation
type Recommendation struct {
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	
	// Action is the recommended action (e.g., "buy", "sell", "hold")
	Action string `json:"action"`
	
	// Price is the recommended price
	Price float64 `json:"price"`
	
	// Quantity is the recommended quantity
	Quantity float64 `json:"quantity"`
	
	// Confidence is the confidence level (0-100)
	Confidence float64 `json:"confidence"`
	
	// Rationale is the rationale for the recommendation
	Rationale string `json:"rationale"`
	
	// Timestamp is when the recommendation was generated
	Timestamp time.Time `json:"timestamp"`
	
	// ExpiresAt is when the recommendation expires
	ExpiresAt time.Time `json:"expires_at"`
	
	// Indicators is a map of indicator values that contributed to the recommendation
	Indicators map[string]float64 `json:"indicators"`
}

// ScenarioRequest represents a request for scenario analysis
type ScenarioRequest struct {
	// BaseSymbol is the base trading symbol
	BaseSymbol string `json:"base_symbol"`
	
	// Scenarios is a list of scenarios to analyze
	Scenarios []Scenario `json:"scenarios"`
	
	// Portfolio is the current portfolio
	Portfolio Portfolio `json:"portfolio"`
	
	// RiskParameters is a map of risk parameters
	RiskParameters map[string]float64 `json:"risk_parameters"`
}

// Scenario represents a market scenario
type Scenario struct {
	// Name is the name of the scenario
	Name string `json:"name"`
	
	// Description is a description of the scenario
	Description string `json:"description"`
	
	// PriceChange is the expected price change
	PriceChange float64 `json:"price_change"`
	
	// VolatilityChange is the expected volatility change
	VolatilityChange float64 `json:"volatility_change"`
	
	// Probability is the estimated probability of the scenario
	Probability float64 `json:"probability"`
	
	// AdditionalFactors is a map of additional factors
	AdditionalFactors map[string]interface{} `json:"additional_factors"`
}

// Portfolio represents a trading portfolio
type Portfolio struct {
	// Positions is a list of positions
	Positions []Position `json:"positions"`
	
	// Cash is the available cash
	Cash float64 `json:"cash"`
	
	// TotalValue is the total portfolio value
	TotalValue float64 `json:"total_value"`
}

// Position represents a position in a portfolio
type Position struct {
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	
	// Quantity is the position quantity
	Quantity float64 `json:"quantity"`
	
	// EntryPrice is the average entry price
	EntryPrice float64 `json:"entry_price"`
	
	// CurrentPrice is the current market price
	CurrentPrice float64 `json:"current_price"`
	
	// UnrealizedPL is the unrealized profit/loss
	UnrealizedPL float64 `json:"unrealized_pl"`
}

// BacktestRequest represents a request for backtesting
type BacktestRequest struct {
	// Strategy is the strategy to backtest
	Strategy string `json:"strategy"`
	
	// Symbols is a list of symbols to include
	Symbols []string `json:"symbols"`
	
	// StartTime is the start time for the backtest
	StartTime time.Time `json:"start_time"`
	
	// EndTime is the end time for the backtest
	EndTime time.Time `json:"end_time"`
	
	// InitialCapital is the initial capital
	InitialCapital float64 `json:"initial_capital"`
	
	// Parameters is a map of strategy parameters
	Parameters map[string]interface{} `json:"parameters"`
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	// Strategy is the strategy that was tested
	Strategy string `json:"strategy"`
	
	// StartTime is the start time of the backtest
	StartTime time.Time `json:"start_time"`
	
	// EndTime is the end time of the backtest
	EndTime time.Time `json:"end_time"`
	
	// InitialCapital is the initial capital
	InitialCapital float64 `json:"initial_capital"`
	
	// FinalCapital is the final capital
	FinalCapital float64 `json:"final_capital"`
	
	// TotalReturn is the total return
	TotalReturn float64 `json:"total_return"`
	
	// AnnualizedReturn is the annualized return
	AnnualizedReturn float64 `json:"annualized_return"`
	
	// SharpeRatio is the Sharpe ratio
	SharpeRatio float64 `json:"sharpe_ratio"`
	
	// MaxDrawdown is the maximum drawdown
	MaxDrawdown float64 `json:"max_drawdown"`
	
	// WinRate is the win rate
	WinRate float64 `json:"win_rate"`
	
	// Trades is a list of trades
	Trades []BacktestTrade `json:"trades"`
	
	// EquityCurve is the equity curve
	EquityCurve map[string]float64 `json:"equity_curve"`
}

// BacktestTrade represents a trade in a backtest
type BacktestTrade struct {
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	
	// EntryTime is when the trade was entered
	EntryTime time.Time `json:"entry_time"`
	
	// EntryPrice is the entry price
	EntryPrice float64 `json:"entry_price"`
	
	// ExitTime is when the trade was exited
	ExitTime time.Time `json:"exit_time"`
	
	// ExitPrice is the exit price
	ExitPrice float64 `json:"exit_price"`
	
	// Quantity is the trade quantity
	Quantity float64 `json:"quantity"`
	
	// ProfitLoss is the profit/loss
	ProfitLoss float64 `json:"profit_loss"`
	
	// Side is the trade side (buy/sell)
	Side string `json:"side"`
}

// AlertConfiguration represents an alert configuration
type AlertConfiguration struct {
	// Name is the name of the alert
	Name string `json:"name"`
	
	// Description is a description of the alert
	Description string `json:"description"`
	
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	
	// Condition is the alert condition
	Condition string `json:"condition"`
	
	// Threshold is the alert threshold
	Threshold float64 `json:"threshold"`
	
	// NotificationChannels is a list of notification channels
	NotificationChannels []string `json:"notification_channels"`
	
	// Enabled indicates if the alert is enabled
	Enabled bool `json:"enabled"`
}

// Alert represents an alert
type Alert struct {
	// ID is the alert ID
	ID string `json:"id"`
	
	// ConfigurationID is the ID of the alert configuration
	ConfigurationID string `json:"configuration_id"`
	
	// Timestamp is when the alert was triggered
	Timestamp time.Time `json:"timestamp"`
	
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	
	// Message is the alert message
	Message string `json:"message"`
	
	// Value is the value that triggered the alert
	Value float64 `json:"value"`
	
	// Threshold is the alert threshold
	Threshold float64 `json:"threshold"`
	
	// Acknowledged indicates if the alert has been acknowledged
	Acknowledged bool `json:"acknowledged"`
}

// DecisionSupportService defines the interface for the decision support service
type DecisionSupportService interface {
	// Analyze analyzes market data and returns recommendations
	Analyze(ctx context.Context, request AnalysisRequest) ([]Recommendation, error)
	
	// GetRecommendations gets trading recommendations
	GetRecommendations(ctx context.Context, symbol string, limit int) ([]Recommendation, error)
	
	// AnalyzeScenarios analyzes different market scenarios
	AnalyzeScenarios(ctx context.Context, request ScenarioRequest) (map[string]interface{}, error)
	
	// Backtest runs a backtest of a trading strategy
	Backtest(ctx context.Context, request BacktestRequest) (*BacktestResult, error)
	
	// GetInsights gets market insights for a symbol
	GetInsights(ctx context.Context, symbol string) (map[string]interface{}, error)
	
	// OptimizePortfolio optimizes a portfolio
	OptimizePortfolio(ctx context.Context, portfolio Portfolio, objective string) (*Portfolio, error)
	
	// ConfigureAlert configures an alert
	ConfigureAlert(ctx context.Context, config AlertConfiguration) (string, error)
	
	// GetAlerts gets current alerts
	GetAlerts(ctx context.Context, acknowledged bool) ([]Alert, error)
	
	// AcknowledgeAlert acknowledges an alert
	AcknowledgeAlert(ctx context.Context, alertID string) error
}

