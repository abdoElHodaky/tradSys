package order_matching

import (
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"github.com/google/uuid"
)

// Use types from the shared types package
type OrderType = types.OrderType
type OrderSide = types.OrderSide
type OrderStatus = types.OrderStatus
type Order = types.Order

// Constants from types package
const (
	OrderTypeLimit             = types.OrderTypeLimit
	OrderTypeMarket            = types.OrderTypeMarket
	OrderTypeStop              = types.OrderTypeStop
	OrderTypeStopLimit         = types.OrderTypeStopLimit
	OrderTypeStopMarket        = types.OrderTypeStopMarket
	OrderSideBuy               = types.OrderSideBuy
	OrderSideSell              = types.OrderSideSell
	OrderStatusNew             = types.OrderStatusNew
	OrderStatusPartiallyFilled = types.OrderStatusPartiallyFilled
	OrderStatusFilled          = types.OrderStatusFilled
	OrderStatusCanceled        = types.OrderStatusCanceled
	OrderStatusCancelled       = types.OrderStatusCancelled
	OrderStatusRejected        = types.OrderStatusRejected
	OrderStatusExpired         = types.OrderStatusExpired
)

// Trade represents a trade execution
type Trade struct {
	// ID is the unique identifier for the trade
	ID string `json:"id"`
	// Symbol is the trading symbol
	Symbol string `json:"symbol"`
	// Price is the price of the trade
	Price float64 `json:"price"`
	// Quantity is the quantity of the trade
	Quantity float64 `json:"quantity"`
	// BuyOrderID is the buy order ID
	BuyOrderID string `json:"buy_order_id"`
	// SellOrderID is the sell order ID
	SellOrderID string `json:"sell_order_id"`
	// Timestamp is when the trade was executed
	Timestamp time.Time `json:"timestamp"`
	// IsMaker indicates if this is a maker trade
	IsMaker bool `json:"is_maker"`
	// TakerSide indicates which side was the taker
	TakerSide OrderSide `json:"taker_side"`
	
	// Settlement information
	SettlementStatus string `json:"settlement_status"`
	SettledAt        *time.Time `json:"settled_at,omitempty"`
	
	// Fee information
	MakerFee float64 `json:"maker_fee"`
	TakerFee float64 `json:"taker_fee"`
	
	// Metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// TradeExecution represents the result of a trade execution
type TradeExecution struct {
	Trades       []*Trade `json:"trades"`
	UpdatedOrder *Order   `json:"updated_order"`
	Error        error    `json:"error,omitempty"`
}

// MatchResult represents the result of order matching
type MatchResult struct {
	Trades           []*Trade `json:"trades"`
	RemainingOrder   *Order   `json:"remaining_order,omitempty"`
	FullyMatched     bool     `json:"fully_matched"`
	PartiallyMatched bool     `json:"partially_matched"`
	TotalQuantity    float64  `json:"total_quantity"`
	WeightedPrice    float64  `json:"weighted_price"`
}

// OrderBookSnapshot represents a snapshot of the order book
type OrderBookSnapshot struct {
	Symbol    string                 `json:"symbol"`
	Timestamp time.Time              `json:"timestamp"`
	Bids      []*OrderBookLevel      `json:"bids"`
	Asks      []*OrderBookLevel      `json:"asks"`
	LastTrade *Trade                 `json:"last_trade,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// OrderBookLevel represents a price level in the order book
type OrderBookLevel struct {
	Price    float64 `json:"price"`
	Quantity float64 `json:"quantity"`
	Count    int     `json:"count"`
}

// MarketDepth represents market depth information
type MarketDepth struct {
	Symbol     string    `json:"symbol"`
	Timestamp  time.Time `json:"timestamp"`
	BidDepth   float64   `json:"bid_depth"`
	AskDepth   float64   `json:"ask_depth"`
	Spread     float64   `json:"spread"`
	MidPrice   float64   `json:"mid_price"`
	BestBid    float64   `json:"best_bid"`
	BestAsk    float64   `json:"best_ask"`
	TotalBids  int       `json:"total_bids"`
	TotalAsks  int       `json:"total_asks"`
}

// EngineStats represents matching engine statistics
type EngineStats struct {
	Symbol           string    `json:"symbol"`
	Timestamp        time.Time `json:"timestamp"`
	TotalTrades      int64     `json:"total_trades"`
	TotalVolume      float64   `json:"total_volume"`
	TotalValue       float64   `json:"total_value"`
	OrdersProcessed  int64     `json:"orders_processed"`
	OrdersMatched    int64     `json:"orders_matched"`
	OrdersCancelled  int64     `json:"orders_cancelled"`
	AverageLatency   float64   `json:"average_latency_ms"`
	PeakLatency      float64   `json:"peak_latency_ms"`
	ThroughputTPS    float64   `json:"throughput_tps"`
	
	// Performance metrics
	MatchingTime     time.Duration `json:"matching_time"`
	BookUpdateTime   time.Duration `json:"book_update_time"`
	NotificationTime time.Duration `json:"notification_time"`
}

// NewTrade creates a new trade
func NewTrade(symbol string, price, quantity float64, buyOrderID, sellOrderID string) *Trade {
	return &Trade{
		ID:          uuid.New().String(),
		Symbol:      symbol,
		Price:       price,
		Quantity:    quantity,
		BuyOrderID:  buyOrderID,
		SellOrderID: sellOrderID,
		Timestamp:   time.Now(),
		Metadata:    make(map[string]interface{}),
		SettlementStatus: "pending",
	}
}

// NewMatchResult creates a new match result
func NewMatchResult() *MatchResult {
	return &MatchResult{
		Trades: make([]*Trade, 0),
	}
}

// AddTrade adds a trade to the match result
func (mr *MatchResult) AddTrade(trade *Trade) {
	mr.Trades = append(mr.Trades, trade)
	mr.TotalQuantity += trade.Quantity
	
	// Calculate weighted average price
	if len(mr.Trades) == 1 {
		mr.WeightedPrice = trade.Price
	} else {
		totalValue := 0.0
		totalQuantity := 0.0
		for _, t := range mr.Trades {
			totalValue += t.Price * t.Quantity
			totalQuantity += t.Quantity
		}
		if totalQuantity > 0 {
			mr.WeightedPrice = totalValue / totalQuantity
		}
	}
}

// SetRemainingOrder sets the remaining order after matching
func (mr *MatchResult) SetRemainingOrder(order *Order) {
	mr.RemainingOrder = order
	mr.FullyMatched = order == nil || order.RemainingQuantity == 0
	mr.PartiallyMatched = !mr.FullyMatched && len(mr.Trades) > 0
}

// GetTotalValue returns the total value of all trades
func (mr *MatchResult) GetTotalValue() float64 {
	totalValue := 0.0
	for _, trade := range mr.Trades {
		totalValue += trade.Price * trade.Quantity
	}
	return totalValue
}

// GetTradeCount returns the number of trades
func (mr *MatchResult) GetTradeCount() int {
	return len(mr.Trades)
}

// IsEmpty returns true if no trades were executed
func (mr *MatchResult) IsEmpty() bool {
	return len(mr.Trades) == 0
}

// GetValue returns the total value of the trade
func (t *Trade) GetValue() float64 {
	return t.Price * t.Quantity
}

// IsBuyerMaker returns true if the buyer was the maker
func (t *Trade) IsBuyerMaker() bool {
	return t.TakerSide == OrderSideSell
}

// IsSellerMaker returns true if the seller was the maker
func (t *Trade) IsSellerMaker() bool {
	return t.TakerSide == OrderSideBuy
}

// SetSettled marks the trade as settled
func (t *Trade) SetSettled() {
	t.SettlementStatus = "settled"
	now := time.Now()
	t.SettledAt = &now
}

// SetFailed marks the trade as failed
func (t *Trade) SetFailed() {
	t.SettlementStatus = "failed"
}

// IsSettled returns true if the trade is settled
func (t *Trade) IsSettled() bool {
	return t.SettlementStatus == "settled"
}

// IsPending returns true if the trade is pending settlement
func (t *Trade) IsPending() bool {
	return t.SettlementStatus == "pending"
}

// IsFailed returns true if the trade settlement failed
func (t *Trade) IsFailed() bool {
	return t.SettlementStatus == "failed"
}

// NewOrderBookSnapshot creates a new order book snapshot
func NewOrderBookSnapshot(symbol string) *OrderBookSnapshot {
	return &OrderBookSnapshot{
		Symbol:    symbol,
		Timestamp: time.Now(),
		Bids:      make([]*OrderBookLevel, 0),
		Asks:      make([]*OrderBookLevel, 0),
		Metadata:  make(map[string]interface{}),
	}
}

// AddBidLevel adds a bid level to the snapshot
func (obs *OrderBookSnapshot) AddBidLevel(price, quantity float64, count int) {
	obs.Bids = append(obs.Bids, &OrderBookLevel{
		Price:    price,
		Quantity: quantity,
		Count:    count,
	})
}

// AddAskLevel adds an ask level to the snapshot
func (obs *OrderBookSnapshot) AddAskLevel(price, quantity float64, count int) {
	obs.Asks = append(obs.Asks, &OrderBookLevel{
		Price:    price,
		Quantity: quantity,
		Count:    count,
	})
}

// GetSpread returns the bid-ask spread
func (obs *OrderBookSnapshot) GetSpread() float64 {
	if len(obs.Bids) == 0 || len(obs.Asks) == 0 {
		return 0
	}
	return obs.Asks[0].Price - obs.Bids[0].Price
}

// GetMidPrice returns the mid price
func (obs *OrderBookSnapshot) GetMidPrice() float64 {
	if len(obs.Bids) == 0 || len(obs.Asks) == 0 {
		return 0
	}
	return (obs.Bids[0].Price + obs.Asks[0].Price) / 2
}

// NewMarketDepth creates a new market depth
func NewMarketDepth(symbol string) *MarketDepth {
	return &MarketDepth{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}
}

// UpdateFromSnapshot updates market depth from order book snapshot
func (md *MarketDepth) UpdateFromSnapshot(snapshot *OrderBookSnapshot) {
	md.Timestamp = snapshot.Timestamp
	
	if len(snapshot.Bids) > 0 {
		md.BestBid = snapshot.Bids[0].Price
		md.BidDepth = 0
		for _, level := range snapshot.Bids {
			md.BidDepth += level.Quantity
		}
		md.TotalBids = len(snapshot.Bids)
	}
	
	if len(snapshot.Asks) > 0 {
		md.BestAsk = snapshot.Asks[0].Price
		md.AskDepth = 0
		for _, level := range snapshot.Asks {
			md.AskDepth += level.Quantity
		}
		md.TotalAsks = len(snapshot.Asks)
	}
	
	md.Spread = md.GetSpread()
	md.MidPrice = md.GetMidPrice()
}

// GetSpread returns the bid-ask spread
func (md *MarketDepth) GetSpread() float64 {
	if md.BestBid == 0 || md.BestAsk == 0 {
		return 0
	}
	return md.BestAsk - md.BestBid
}

// GetMidPrice returns the mid price
func (md *MarketDepth) GetMidPrice() float64 {
	if md.BestBid == 0 || md.BestAsk == 0 {
		return 0
	}
	return (md.BestBid + md.BestAsk) / 2
}

// NewEngineStats creates new engine statistics
func NewEngineStats(symbol string) *EngineStats {
	return &EngineStats{
		Symbol:    symbol,
		Timestamp: time.Now(),
	}
}

// UpdateStats updates the engine statistics
func (es *EngineStats) UpdateStats(trades []*Trade, processingTime time.Duration) {
	es.Timestamp = time.Now()
	es.TotalTrades += int64(len(trades))
	es.OrdersProcessed++
	
	if len(trades) > 0 {
		es.OrdersMatched++
		for _, trade := range trades {
			es.TotalVolume += trade.Quantity
			es.TotalValue += trade.GetValue()
		}
	}
	
	// Update latency metrics
	latencyMs := float64(processingTime.Nanoseconds()) / 1e6
	if es.AverageLatency == 0 {
		es.AverageLatency = latencyMs
	} else {
		es.AverageLatency = (es.AverageLatency + latencyMs) / 2
	}
	
	if latencyMs > es.PeakLatency {
		es.PeakLatency = latencyMs
	}
	
	// Calculate throughput (simplified)
	if es.Timestamp.Sub(time.Time{}).Seconds() > 0 {
		es.ThroughputTPS = float64(es.OrdersProcessed) / es.Timestamp.Sub(time.Time{}).Seconds()
	}
}
