package common

import (
	"context"
	"fmt"
	"time"
)

// Exchange represents a unified interface for all exchange integrations
type Exchange interface {
	// Basic exchange operations
	GetExchangeInfo() *ExchangeInfo
	Connect(ctx context.Context) error
	Disconnect(ctx context.Context) error
	IsConnected() bool
	
	// Market data operations
	GetMarketData(ctx context.Context, symbol string) (*MarketData, error)
	GetOrderBook(ctx context.Context, symbol string, depth int) (*OrderBook, error)
	GetTrades(ctx context.Context, symbol string, limit int) ([]*Trade, error)
	GetKlines(ctx context.Context, symbol string, interval string, limit int) ([]*Kline, error)
	
	// Trading operations
	PlaceOrder(ctx context.Context, order *OrderRequest) (*OrderResponse, error)
	CancelOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOrder(ctx context.Context, orderID string) (*OrderResponse, error)
	GetOpenOrders(ctx context.Context, symbol string) ([]*OrderResponse, error)
	GetOrderHistory(ctx context.Context, symbol string, limit int) ([]*OrderResponse, error)
	
	// Account operations
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)
	GetBalances(ctx context.Context) ([]*Balance, error)
	GetPositions(ctx context.Context) ([]*Position, error)
	
	// Symbol operations
	GetSymbols(ctx context.Context) ([]*Symbol, error)
	GetSymbolInfo(ctx context.Context, symbol string) (*Symbol, error)
	
	// Compliance operations (for regulated exchanges)
	ValidateCompliance(ctx context.Context, request *ComplianceRequest) (*ComplianceResponse, error)
	
	// Health and monitoring
	GetHealth(ctx context.Context) (*HealthStatus, error)
	GetStats(ctx context.Context) (*ExchangeStats, error)
}

// ExchangeInfo contains basic information about an exchange
type ExchangeInfo struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Country         string            `json:"country"`
	Region          string            `json:"region"`
	Timezone        string            `json:"timezone"`
	Currency        string            `json:"currency"`
	Languages       []string          `json:"languages"`
	AssetTypes      []AssetType       `json:"asset_types"`
	TradingHours    *TradingSchedule  `json:"trading_hours"`
	Features        []ExchangeFeature `json:"features"`
	ComplianceTypes []ComplianceType  `json:"compliance_types"`
	APIVersion      string            `json:"api_version"`
	LastUpdated     time.Time         `json:"last_updated"`
}

// AssetType represents different types of tradeable assets
type AssetType string

const (
	AssetTypeStock       AssetType = "stock"
	AssetTypeBond        AssetType = "bond"
	AssetTypeETF         AssetType = "etf"
	AssetTypeMutualFund  AssetType = "mutual_fund"
	AssetTypeCommodity   AssetType = "commodity"
	AssetTypeCurrency    AssetType = "currency"
	AssetTypeCrypto      AssetType = "crypto"
	AssetTypeDerivative  AssetType = "derivative"
	AssetTypeSukuk       AssetType = "sukuk"       // Islamic bond
	AssetTypeIslamicFund AssetType = "islamic_fund" // Sharia-compliant fund
)

// ExchangeFeature represents features supported by an exchange
type ExchangeFeature string

const (
	FeatureSpotTrading     ExchangeFeature = "spot_trading"
	FeatureMarginTrading   ExchangeFeature = "margin_trading"
	FeatureFuturesTrading  ExchangeFeature = "futures_trading"
	FeatureOptionsTrading  ExchangeFeature = "options_trading"
	FeatureIslamicTrading  ExchangeFeature = "islamic_trading"
	FeatureAlgoTrading     ExchangeFeature = "algo_trading"
	FeatureHighFrequency   ExchangeFeature = "high_frequency"
	FeatureMarketMaking    ExchangeFeature = "market_making"
)

// ComplianceType represents different compliance frameworks
type ComplianceType string

const (
	ComplianceTypeSharia ComplianceType = "sharia"
	ComplianceTypeSEC    ComplianceType = "sec"
	ComplianceTypeMiFID  ComplianceType = "mifid"
	ComplianceTypeSCA    ComplianceType = "sca"    // UAE Securities and Commodities Authority
	ComplianceTypeADGM   ComplianceType = "adgm"   // Abu Dhabi Global Market
	ComplianceTypeDIFC   ComplianceType = "difc"   // Dubai International Financial Centre
)

// TradingSchedule represents trading hours and schedules
type TradingSchedule struct {
	Timezone      string                    `json:"timezone"`
	MarketHours   map[string]*TradingHours  `json:"market_hours"`   // day of week -> hours
	Holidays      []time.Time               `json:"holidays"`
	SpecialHours  map[string]*TradingHours  `json:"special_hours"`  // special dates
	LastUpdated   time.Time                 `json:"last_updated"`
}

// TradingHours represents trading hours for a specific day
type TradingHours struct {
	IsOpen    bool      `json:"is_open"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Breaks    []*Break  `json:"breaks,omitempty"`
}

// Break represents a trading break (e.g., lunch break)
type Break struct {
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Reason    string    `json:"reason"`
}

// MarketData represents real-time market data
type MarketData struct {
	Symbol      string    `json:"symbol"`
	Price       float64   `json:"price"`
	Volume      float64   `json:"volume"`
	High24h     float64   `json:"high_24h"`
	Low24h      float64   `json:"low_24h"`
	Change24h   float64   `json:"change_24h"`
	ChangePerc  float64   `json:"change_perc"`
	BidPrice    float64   `json:"bid_price"`
	AskPrice    float64   `json:"ask_price"`
	BidSize     float64   `json:"bid_size"`
	AskSize     float64   `json:"ask_size"`
	LastTrade   time.Time `json:"last_trade"`
	Timestamp   time.Time `json:"timestamp"`
}

// OrderBook represents order book data
type OrderBook struct {
	Symbol    string       `json:"symbol"`
	Bids      []*PriceLevel `json:"bids"`
	Asks      []*PriceLevel `json:"asks"`
	Timestamp time.Time    `json:"timestamp"`
}

// PriceLevel represents a price level in the order book
type PriceLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Count    int     `json:"count,omitempty"`
}

// Trade represents a trade execution
type Trade struct {
	ID        string    `json:"id"`
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Quantity  float64   `json:"quantity"`
	Side      OrderSide `json:"side"`
	Timestamp time.Time `json:"timestamp"`
}

// Kline represents candlestick data
type Kline struct {
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	OpenTime  time.Time `json:"open_time"`
	CloseTime time.Time `json:"close_time"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Trades    int       `json:"trades"`
}

// OrderRequest represents an order placement request
type OrderRequest struct {
	Symbol      string      `json:"symbol"`
	Side        OrderSide   `json:"side"`
	Type        OrderType   `json:"type"`
	Quantity    float64     `json:"quantity"`
	Price       float64     `json:"price,omitempty"`
	StopPrice   float64     `json:"stop_price,omitempty"`
	TimeInForce TimeInForce `json:"time_in_force"`
	ClientID    string      `json:"client_id,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// OrderResponse represents an order response
type OrderResponse struct {
	ID              string      `json:"id"`
	ClientID        string      `json:"client_id,omitempty"`
	Symbol          string      `json:"symbol"`
	Side            OrderSide   `json:"side"`
	Type            OrderType   `json:"type"`
	Quantity        float64     `json:"quantity"`
	Price           float64     `json:"price,omitempty"`
	StopPrice       float64     `json:"stop_price,omitempty"`
	FilledQuantity  float64     `json:"filled_quantity"`
	Status          OrderStatus `json:"status"`
	TimeInForce     TimeInForce `json:"time_in_force"`
	CreatedAt       time.Time   `json:"created_at"`
	UpdatedAt       time.Time   `json:"updated_at"`
	Trades          []*Trade    `json:"trades,omitempty"`
}

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeMarket    OrderType = "market"
	OrderTypeLimit     OrderType = "limit"
	OrderTypeStopLoss  OrderType = "stop_loss"
	OrderTypeStopLimit OrderType = "stop_limit"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusExpired         OrderStatus = "expired"
)

// TimeInForce represents order time in force
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Cancelled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
	TimeInForceDAY TimeInForce = "DAY" // Day order
)

// AccountInfo represents account information
type AccountInfo struct {
	AccountID   string    `json:"account_id"`
	UserID      string    `json:"user_id"`
	AccountType string    `json:"account_type"`
	Status      string    `json:"status"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Permissions []string  `json:"permissions"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// Balance represents account balance
type Balance struct {
	Asset     string  `json:"asset"`
	Free      float64 `json:"free"`
	Locked    float64 `json:"locked"`
	Total     float64 `json:"total"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Position represents a trading position
type Position struct {
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	Size          float64   `json:"size"`
	EntryPrice    float64   `json:"entry_price"`
	MarkPrice     float64   `json:"mark_price"`
	UnrealizedPnL float64   `json:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl"`
	Margin        float64   `json:"margin"`
	UpdatedAt     time.Time `json:"updated_at"`
}

// Symbol represents a tradeable symbol
type Symbol struct {
	Symbol          string    `json:"symbol"`
	BaseAsset       string    `json:"base_asset"`
	QuoteAsset      string    `json:"quote_asset"`
	Status          string    `json:"status"`
	AssetType       AssetType `json:"asset_type"`
	MinQuantity     float64   `json:"min_quantity"`
	MaxQuantity     float64   `json:"max_quantity"`
	StepSize        float64   `json:"step_size"`
	MinPrice        float64   `json:"min_price"`
	MaxPrice        float64   `json:"max_price"`
	TickSize        float64   `json:"tick_size"`
	MinNotional     float64   `json:"min_notional"`
	IslamicCompliant bool     `json:"islamic_compliant,omitempty"`
	Permissions     []string  `json:"permissions"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// ComplianceRequest represents a compliance validation request
type ComplianceRequest struct {
	Type      ComplianceType             `json:"type"`
	UserID    string                     `json:"user_id"`
	Symbol    string                     `json:"symbol,omitempty"`
	OrderData *OrderRequest              `json:"order_data,omitempty"`
	Metadata  map[string]interface{}     `json:"metadata,omitempty"`
}

// ComplianceResponse represents a compliance validation response
type ComplianceResponse struct {
	Approved    bool                   `json:"approved"`
	Reason      string                 `json:"reason,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	ValidUntil  *time.Time             `json:"valid_until,omitempty"`
}

// HealthStatus represents exchange health status
type HealthStatus struct {
	Status      string                 `json:"status"`
	Timestamp   time.Time              `json:"timestamp"`
	Services    map[string]string      `json:"services"`
	Latency     time.Duration          `json:"latency"`
	Uptime      time.Duration          `json:"uptime"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ExchangeStats represents exchange statistics
type ExchangeStats struct {
	TotalVolume24h    float64                `json:"total_volume_24h"`
	TotalTrades24h    int64                  `json:"total_trades_24h"`
	ActiveSymbols     int                    `json:"active_symbols"`
	ConnectedUsers    int                    `json:"connected_users"`
	AverageLatency    time.Duration          `json:"average_latency"`
	OrderBookDepth    map[string]int         `json:"order_book_depth"`
	TopSymbolsByVolume []*SymbolStats        `json:"top_symbols_by_volume"`
	Timestamp         time.Time              `json:"timestamp"`
}

// SymbolStats represents statistics for a specific symbol
type SymbolStats struct {
	Symbol     string  `json:"symbol"`
	Volume24h  float64 `json:"volume_24h"`
	Trades24h  int64   `json:"trades_24h"`
	Change24h  float64 `json:"change_24h"`
	High24h    float64 `json:"high_24h"`
	Low24h     float64 `json:"low_24h"`
}

// ExchangeError represents an exchange-specific error
type ExchangeError struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Details   string `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *ExchangeError) Error() string {
	return fmt.Sprintf("exchange error [%s]: %s", e.Code, e.Message)
}

// Common error codes
const (
	ErrorCodeInvalidSymbol     = "INVALID_SYMBOL"
	ErrorCodeInsufficientFunds = "INSUFFICIENT_FUNDS"
	ErrorCodeOrderNotFound     = "ORDER_NOT_FOUND"
	ErrorCodeMarketClosed      = "MARKET_CLOSED"
	ErrorCodeComplianceFailure = "COMPLIANCE_FAILURE"
	ErrorCodeRateLimitExceeded = "RATE_LIMIT_EXCEEDED"
	ErrorCodeConnectionFailed  = "CONNECTION_FAILED"
	ErrorCodeInvalidRequest    = "INVALID_REQUEST"
	ErrorCodeUnauthorized      = "UNAUTHORIZED"
	ErrorCodeInternalError     = "INTERNAL_ERROR"
)
