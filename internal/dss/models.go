package dss

import (
	"time"
)

// User represents a system user
type User struct {
	ID       string
	Username string
	Role     string
	Tier     string
}

// AnalysisRequest represents a request to analyze market data
type AnalysisRequest struct {
	Symbol     string     `json:"symbol"`
	Timeframe  string     `json:"timeframe"`
	Indicators []string   `json:"indicators"`
	StartTime  time.Time  `json:"start_time"`
	EndTime    time.Time  `json:"end_time"`
	Parameters interface{} `json:"parameters,omitempty"`
}

// AnalysisResult represents the result of a market data analysis
type AnalysisResult struct {
	AnalysisID string                 `json:"analysis_id"`
	Status     string                 `json:"status"`
	Symbol     string                 `json:"symbol"`
	Timeframe  string                 `json:"timeframe"`
	Results    map[string]interface{} `json:"results"`
	CreatedAt  time.Time              `json:"created_at"`
	CompletedAt *time.Time            `json:"completed_at,omitempty"`
}

// CustomAnalysisRequest represents a request for custom analysis
type CustomAnalysisRequest struct {
	Symbol     string                 `json:"symbol"`
	Timeframe  string                 `json:"timeframe"`
	Algorithm  string                 `json:"algorithm"`
	Parameters map[string]interface{} `json:"parameters"`
	StartTime  time.Time              `json:"start_time"`
	EndTime    time.Time              `json:"end_time"`
}

// RecommendationRequest represents a request for trading recommendations
type RecommendationRequest struct {
	Symbol      string                 `json:"symbol"`
	Strategy    string                 `json:"strategy"`
	RiskProfile string                 `json:"risk_profile"`
	PositionSize string                `json:"position_size"`
	Context     map[string]interface{} `json:"context,omitempty"`
}

// Recommendation represents a trading recommendation
type Recommendation struct {
	RecommendationID string                 `json:"recommendation_id"`
	Timestamp        time.Time              `json:"timestamp"`
	Symbol           string                 `json:"symbol"`
	Action           string                 `json:"action"`
	Confidence       float64                `json:"confidence"`
	PriceTarget      float64                `json:"price_target,omitempty"`
	StopLoss         float64                `json:"stop_loss,omitempty"`
	TimeHorizon      string                 `json:"time_horizon"`
	PositionSize     map[string]interface{} `json:"position_size,omitempty"`
	Reasoning        []string               `json:"reasoning,omitempty"`
	Expiration       time.Time              `json:"expiration,omitempty"`
}

// ExecuteRecommendationRequest represents a request to execute a recommendation
type ExecuteRecommendationRequest struct {
	RecommendationID string  `json:"recommendation_id"`
	PositionSize     float64 `json:"position_size,omitempty"`
	CustomPrice      float64 `json:"custom_price,omitempty"`
}

// ExecutionResult represents the result of executing a recommendation
type ExecutionResult struct {
	ExecutionID      string    `json:"execution_id"`
	RecommendationID string    `json:"recommendation_id"`
	OrderID          string    `json:"order_id"`
	Symbol           string    `json:"symbol"`
	Action           string    `json:"action"`
	Price            float64   `json:"price"`
	Size             float64   `json:"size"`
	Status           string    `json:"status"`
	Timestamp        time.Time `json:"timestamp"`
}

// ModelRequest represents a request to create or update a model
type ModelRequest struct {
	Name           string                 `json:"name"`
	Description    string                 `json:"description"`
	Type           string                 `json:"type"`
	Parameters     map[string]interface{} `json:"parameters"`
	Signals        map[string]interface{} `json:"signals"`
	RiskManagement map[string]interface{} `json:"risk_management,omitempty"`
}

// Model represents an analysis model
type Model struct {
	ModelID     string                 `json:"model_id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Parameters  map[string]interface{} `json:"parameters"`
	Signals     map[string]interface{} `json:"signals"`
	Status      string                 `json:"status"`
	Version     int                    `json:"version"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

// BacktestRequest represents a request to run a backtest
type BacktestRequest struct {
	ModelID         string                 `json:"model_id"`
	Symbols         []string               `json:"symbols"`
	Timeframe       string                 `json:"timeframe"`
	StartTime       time.Time              `json:"start_time"`
	EndTime         time.Time              `json:"end_time"`
	InitialCapital  float64                `json:"initial_capital"`
	Parameters      map[string]interface{} `json:"parameters,omitempty"`
	RiskManagement  map[string]interface{} `json:"risk_management,omitempty"`
}

// BacktestResult represents the result of a backtest
type BacktestResult struct {
	BacktestID      string     `json:"backtest_id"`
	ModelID         string     `json:"model_id"`
	Status          string     `json:"status"`
	Progress        float64    `json:"progress"`
	StartTime       time.Time  `json:"start_time"`
	EndTime         time.Time  `json:"end_time"`
	InitialCapital  float64    `json:"initial_capital"`
	FinalCapital    float64    `json:"final_capital,omitempty"`
	TotalReturn     float64    `json:"total_return,omitempty"`
	AnnualizedReturn float64   `json:"annualized_return,omitempty"`
	MaxDrawdown     float64    `json:"max_drawdown,omitempty"`
	SharpeRatio     float64    `json:"sharpe_ratio,omitempty"`
	TotalTrades     int        `json:"total_trades,omitempty"`
	WinRate         float64    `json:"win_rate,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
}

// BacktestTrade represents a trade from a backtest
type BacktestTrade struct {
	TradeID    string    `json:"trade_id"`
	BacktestID string    `json:"backtest_id"`
	Symbol     string    `json:"symbol"`
	Side       string    `json:"side"`
	EntryTime  time.Time `json:"entry_time"`
	EntryPrice float64   `json:"entry_price"`
	ExitTime   time.Time `json:"exit_time,omitempty"`
	ExitPrice  float64   `json:"exit_price,omitempty"`
	Size       float64   `json:"size"`
	PnL        float64   `json:"pnl,omitempty"`
	PnLPercent float64   `json:"pnl_percent,omitempty"`
	Status     string    `json:"status"`
	Signal     string    `json:"signal,omitempty"`
}

// AlertRequest represents a request to create or update an alert
type AlertRequest struct {
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	Symbol               string                   `json:"symbol"`
	Conditions           []map[string]interface{} `json:"conditions"`
	NotificationChannels []string                 `json:"notification_channels"`
	WebhookURL           string                   `json:"webhook_url,omitempty"`
	CooldownPeriod       string                   `json:"cooldown_period,omitempty"`
}

// Alert represents an alert configuration
type Alert struct {
	AlertID              string                   `json:"alert_id"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	Symbol               string                   `json:"symbol"`
	Conditions           []map[string]interface{} `json:"conditions"`
	NotificationChannels []string                 `json:"notification_channels"`
	WebhookURL           string                   `json:"webhook_url,omitempty"`
	CooldownPeriod       string                   `json:"cooldown_period,omitempty"`
	Status               string                   `json:"status"`
	CreatedAt            time.Time                `json:"created_at"`
	UpdatedAt            time.Time                `json:"updated_at"`
	LastTriggeredAt      *time.Time               `json:"last_triggered_at,omitempty"`
}

// AlertEvent represents an alert trigger event
type AlertEvent struct {
	EventID    string                 `json:"event_id"`
	AlertID    string                 `json:"alert_id"`
	Symbol     string                 `json:"symbol"`
	Condition  string                 `json:"condition"`
	Value      interface{}            `json:"value"`
	Message    string                 `json:"message"`
	Data       map[string]interface{} `json:"data,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// WebhookRequest represents a request to create or update a webhook
type WebhookRequest struct {
	URL         string            `json:"url"`
	Description string            `json:"description"`
	Events      []string          `json:"events"`
	Secret      string            `json:"secret,omitempty"`
	Headers     map[string]string `json:"headers,omitempty"`
	Status      string            `json:"status,omitempty"`
}

// Webhook represents a webhook configuration
type Webhook struct {
	WebhookID   string            `json:"webhook_id"`
	URL         string            `json:"url"`
	Description string            `json:"description"`
	Events      []string          `json:"events"`
	Headers     map[string]string `json:"headers,omitempty"`
	Status      string            `json:"status"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// WebSocketMessage represents a message sent over WebSocket
type WebSocketMessage struct {
	Type      string                 `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Symbol    string                 `json:"symbol,omitempty"`
	Symbols   []string               `json:"symbols,omitempty"`
	Data      map[string]interface{} `json:"data"`
}

// Indicator represents a technical indicator
type Indicator struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Category    string                 `json:"category"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// MarketData represents current market data for a symbol
type MarketData struct {
	Symbol        string    `json:"symbol"`
	Price         float64   `json:"price"`
	Volume24h     float64   `json:"volume_24h"`
	Change24h     float64   `json:"change_24h"`
	Change24hPct  float64   `json:"change_24h_percent"`
	High24h       float64   `json:"high_24h"`
	Low24h        float64   `json:"low_24h"`
	BidPrice      float64   `json:"bid_price"`
	AskPrice      float64   `json:"ask_price"`
	BidSize       float64   `json:"bid_size"`
	AskSize       float64   `json:"ask_size"`
	LastUpdated   time.Time `json:"last_updated"`
}

// Candle represents OHLCV data for a specific timeframe
type Candle struct {
	Timestamp time.Time `json:"timestamp"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
}

// Trade represents a market trade
type Trade struct {
	TradeID   string    `json:"trade_id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Size      float64   `json:"size"`
	Side      string    `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

// OrderBookLevel represents a level in the order book
type OrderBookLevel struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// OrderBook represents the order book for a symbol
type OrderBook struct {
	Symbol    string           `json:"symbol"`
	Bids      []OrderBookLevel `json:"bids"`
	Asks      []OrderBookLevel `json:"asks"`
	Timestamp time.Time        `json:"timestamp"`
}

// PaginationResponse represents pagination metadata
type PaginationResponse struct {
	NextCursor string `json:"next_cursor,omitempty"`
	HasMore    bool   `json:"has_more"`
}

// ErrorResponse represents an API error response
type ErrorResponse struct {
	Error struct {
		Code           string                 `json:"code"`
		Message        string                 `json:"message"`
		Details        interface{}            `json:"details,omitempty"`
		RequestID      string                 `json:"request_id,omitempty"`
		DocumentationURL string               `json:"documentation_url,omitempty"`
	} `json:"error"`
}

