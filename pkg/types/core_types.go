package types

import (
	"time"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of an order
type OrderType string

const (
	OrderTypeMarket    OrderType = "market"
	OrderTypeLimit     OrderType = "limit"
	OrderTypeStop      OrderType = "stop"
	OrderTypeStopLimit OrderType = "stop_limit"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCanceled        OrderStatus = "canceled"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusExpired         OrderStatus = "expired"
)

// TimeInForce represents how long an order remains active
type TimeInForce string

const (
	TimeInForceGTC TimeInForce = "GTC" // Good Till Canceled
	TimeInForceIOC TimeInForce = "IOC" // Immediate Or Cancel
	TimeInForceFOK TimeInForce = "FOK" // Fill Or Kill
	TimeInForceDAY TimeInForce = "DAY" // Day order
)

// Order represents a trading order with all necessary fields
// Enhanced with Go 1.24 JSON improvements and generic attributes
type Order struct {
	ID                string      `json:"id" db:"id"`
	ClientOrderID     string      `json:"client_order_id" db:"client_order_id"`
	UserID            string      `json:"user_id" db:"user_id"`
	Symbol            string      `json:"symbol" db:"symbol"`
	Side              OrderSide   `json:"side" db:"side"`
	Type              OrderType   `json:"type" db:"type"`
	Price             float64     `json:"price" db:"price"`
	Quantity          float64     `json:"quantity" db:"quantity"`
	FilledQuantity    float64     `json:"filled_quantity" db:"filled_quantity"`
	RemainingQuantity float64     `json:"remaining_quantity" db:"remaining_quantity"`
	Status            OrderStatus `json:"status" db:"status"`
	TimeInForce       TimeInForce `json:"time_in_force" db:"time_in_force"`
	StopPrice         *float64    `json:"stop_price,omitempty" db:"stop_price"`
	CreatedAt         time.Time   `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time   `json:"updated_at" db:"updated_at"`
	ExpiresAt         *time.Time  `json:"expires_at,omitempty" db:"expires_at"`
	// Enhanced with generic attributes for extensibility
	Attributes OrderAttributes `json:"attributes,omitempty" db:"attributes"`
	// Metadata for additional order information
	Metadata Metadata `json:"metadata,omitempty" db:"metadata"`
}

// Trade represents a completed trade between two orders
type Trade struct {
	ID           string    `json:"id" db:"id"`
	Symbol       string    `json:"symbol" db:"symbol"`
	BuyOrderID   string    `json:"buy_order_id" db:"buy_order_id"`
	SellOrderID  string    `json:"sell_order_id" db:"sell_order_id"`
	Price        float64   `json:"price" db:"price"`
	Quantity     float64   `json:"quantity" db:"quantity"`
	Value        float64   `json:"value" db:"value"`
	Timestamp    time.Time `json:"timestamp" db:"timestamp"`
	BuyUserID    string    `json:"buy_user_id" db:"buy_user_id"`
	SellUserID   string    `json:"sell_user_id" db:"sell_user_id"`
	TakerSide    OrderSide `json:"taker_side" db:"taker_side"`
	MakerOrderID string    `json:"maker_order_id" db:"maker_order_id"`
	TakerOrderID string    `json:"taker_order_id" db:"taker_order_id"`
}

// Position represents a user's position in a symbol
type Position struct {
	ID            string    `json:"id" db:"id"`
	UserID        string    `json:"user_id" db:"user_id"`
	Symbol        string    `json:"symbol" db:"symbol"`
	Quantity      float64   `json:"quantity" db:"quantity"`
	AveragePrice  float64   `json:"average_price" db:"average_price"`
	MarketValue   float64   `json:"market_value" db:"market_value"`
	UnrealizedPnL float64   `json:"unrealized_pnl" db:"unrealized_pnl"`
	RealizedPnL   float64   `json:"realized_pnl" db:"realized_pnl"`
	LastUpdated   time.Time `json:"last_updated" db:"last_updated"`
}

// OrderBook represents the current state of buy and sell orders for a symbol
type OrderBook struct {
	Symbol    string            `json:"symbol"`
	Bids      []*OrderBookLevel `json:"bids"`
	Asks      []*OrderBookLevel `json:"asks"`
	Timestamp time.Time         `json:"timestamp"`
	Sequence  uint64            `json:"sequence"`
}

// OrderBookLevel represents a price level in the order book
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Count    int     `json:"count"`
}

// MarketData represents market data for a symbol
type MarketData struct {
	Symbol           string    `json:"symbol"`
	LastPrice        float64   `json:"last_price"`
	BidPrice         float64   `json:"bid_price"`
	AskPrice         float64   `json:"ask_price"`
	BidSize          float64   `json:"bid_size"`
	AskSize          float64   `json:"ask_size"`
	Volume           float64   `json:"volume"`
	High24h          float64   `json:"high_24h"`
	Low24h           float64   `json:"low_24h"`
	Change24h        float64   `json:"change_24h"`
	ChangePercent24h float64   `json:"change_percent_24h"`
	Timestamp        time.Time `json:"timestamp"`
}

// OHLCV represents Open, High, Low, Close, Volume data
type OHLCV struct {
	Symbol    string    `json:"symbol"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
	Interval  string    `json:"interval"`
}

// User represents a trading user
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Status    string    `json:"status" db:"status"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
}

// Account represents a user's trading account
type Account struct {
	ID               string    `json:"id" db:"id"`
	UserID           string    `json:"user_id" db:"user_id"`
	Balance          float64   `json:"balance" db:"balance"`
	AvailableBalance float64   `json:"available_balance" db:"available_balance"`
	LockedBalance    float64   `json:"locked_balance" db:"locked_balance"`
	Currency         string    `json:"currency" db:"currency"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}

// Symbol represents a tradable symbol
type Symbol struct {
	Symbol      string  `json:"symbol" db:"symbol"`
	BaseAsset   string  `json:"base_asset" db:"base_asset"`
	QuoteAsset  string  `json:"quote_asset" db:"quote_asset"`
	Status      string  `json:"status" db:"status"`
	MinPrice    float64 `json:"min_price" db:"min_price"`
	MaxPrice    float64 `json:"max_price" db:"max_price"`
	TickSize    float64 `json:"tick_size" db:"tick_size"`
	MinQuantity float64 `json:"min_quantity" db:"min_quantity"`
	MaxQuantity float64 `json:"max_quantity" db:"max_quantity"`
	StepSize    float64 `json:"step_size" db:"step_size"`
	MinNotional float64 `json:"min_notional" db:"min_notional"`
}

// IsValid checks if an order is valid
func (o *Order) IsValid() bool {
	if o.Symbol == "" || o.UserID == "" {
		return false
	}
	if o.Side != OrderSideBuy && o.Side != OrderSideSell {
		return false
	}
	if o.Quantity <= 0 {
		return false
	}
	if o.Type == OrderTypeLimit && o.Price <= 0 {
		return false
	}
	return true
}

// IsFilled checks if an order is completely filled
func (o *Order) IsFilled() bool {
	return o.Status == OrderStatusFilled
}

// IsActive checks if an order is active (can be matched)
func (o *Order) IsActive() bool {
	return o.Status == OrderStatusPending || o.Status == OrderStatusPartiallyFilled
}

// GetRemainingQuantity returns the remaining quantity to be filled
func (o *Order) GetRemainingQuantity() float64 {
	return o.Quantity - o.FilledQuantity
}

// CalculateValue calculates the total value of a trade
func (t *Trade) CalculateValue() float64 {
	return t.Price * t.Quantity
}

// GetSpread returns the bid-ask spread
func (md *MarketData) GetSpread() float64 {
	return md.AskPrice - md.BidPrice
}

// GetSpreadPercent returns the bid-ask spread as a percentage
func (md *MarketData) GetSpreadPercent() float64 {
	if md.BidPrice == 0 {
		return 0
	}
	return (md.GetSpread() / md.BidPrice) * 100
}

// GetMidPrice returns the mid price between bid and ask
func (md *MarketData) GetMidPrice() float64 {
	return (md.BidPrice + md.AskPrice) / 2
}

// GetBestBid returns the best bid price and quantity
func (ob *OrderBook) GetBestBid() *OrderBookLevel {
	if len(ob.Bids) == 0 {
		return nil
	}
	return ob.Bids[0]
}

// GetBestAsk returns the best ask price and quantity
func (ob *OrderBook) GetBestAsk() *OrderBookLevel {
	if len(ob.Asks) == 0 {
		return nil
	}
	return ob.Asks[0]
}

// GetSpread returns the bid-ask spread from the order book
func (ob *OrderBook) GetSpread() float64 {
	bestBid := ob.GetBestBid()
	bestAsk := ob.GetBestAsk()
	if bestBid == nil || bestAsk == nil {
		return 0
	}
	return bestAsk.Price - bestBid.Price
}

// GetMidPrice returns the mid price from the order book
func (ob *OrderBook) GetMidPrice() float64 {
	bestBid := ob.GetBestBid()
	bestAsk := ob.GetBestAsk()
	if bestBid == nil || bestAsk == nil {
		return 0
	}
	return (bestBid.Price + bestAsk.Price) / 2
}

// RiskMetrics represents risk metrics for a user or position
type RiskMetrics struct {
	UserID           string    `json:"user_id" db:"user_id"`
	TotalExposure    float64   `json:"total_exposure" db:"total_exposure"`
	AvailableMargin  float64   `json:"available_margin" db:"available_margin"`
	UsedMargin       float64   `json:"used_margin" db:"used_margin"`
	MarginRatio      float64   `json:"margin_ratio" db:"margin_ratio"`
	PnL              float64   `json:"pnl" db:"pnl"`
	UnrealizedPnL    float64   `json:"unrealized_pnl" db:"unrealized_pnl"`
	RealizedPnL      float64   `json:"realized_pnl" db:"realized_pnl"`
	RiskScore        float64   `json:"risk_score" db:"risk_score"`
	VaR              float64   `json:"var" db:"var"` // Value at Risk
	MaxDrawdown      float64   `json:"max_drawdown" db:"max_drawdown"`
	Leverage         float64   `json:"leverage" db:"leverage"`
	LastUpdated      time.Time `json:"last_updated" db:"last_updated"`
	CreatedAt        time.Time `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" db:"updated_at"`
}
