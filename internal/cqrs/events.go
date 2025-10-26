package cqrs

import (
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Base Event Implementation

type BaseEvent struct {
	ID          string      `json:"id"`
	AggregateID string      `json:"aggregate_id"`
	Type        string      `json:"type"`
	Timestamp   time.Time   `json:"timestamp"`
	Data        interface{} `json:"data"`
}

func (e *BaseEvent) GetID() string          { return e.ID }
func (e *BaseEvent) GetType() string        { return e.Type }
func (e *BaseEvent) GetAggregateID() string { return e.AggregateID }
func (e *BaseEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *BaseEvent) GetData() interface{}   { return e.Data }

// Order Events

type OrderCreatedEvent struct {
	ID            string                `json:"id"`
	AggregateID   string                `json:"aggregate_id"`
	OrderID       string                `json:"order_id"`
	UserID        string                `json:"user_id"`
	Symbol        string                `json:"symbol"`
	Side          types.OrderSide       `json:"side"`
	Type          types.OrderType       `json:"type"`
	Price         float64               `json:"price"`
	Quantity      float64               `json:"quantity"`
	TimeInForce   types.TimeInForce     `json:"time_in_force"`
	StopPrice     *float64              `json:"stop_price,omitempty"`
	Timestamp     time.Time             `json:"timestamp"`
}

func (e *OrderCreatedEvent) GetID() string          { return e.ID }
func (e *OrderCreatedEvent) GetType() string        { return "OrderCreated" }
func (e *OrderCreatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *OrderCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OrderCreatedEvent) GetData() interface{}   { return e }

type OrderUpdatedEvent struct {
	ID                string            `json:"id"`
	AggregateID       string            `json:"aggregate_id"`
	OrderID           string            `json:"order_id"`
	FilledQuantity    float64           `json:"filled_quantity"`
	RemainingQuantity float64           `json:"remaining_quantity"`
	Status            types.OrderStatus `json:"status"`
	Timestamp         time.Time         `json:"timestamp"`
}

func (e *OrderUpdatedEvent) GetID() string          { return e.ID }
func (e *OrderUpdatedEvent) GetType() string        { return "OrderUpdated" }
func (e *OrderUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *OrderUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OrderUpdatedEvent) GetData() interface{}   { return e }

type OrderCancelledEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Reason      string    `json:"reason"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *OrderCancelledEvent) GetID() string          { return e.ID }
func (e *OrderCancelledEvent) GetType() string        { return "OrderCancelled" }
func (e *OrderCancelledEvent) GetAggregateID() string { return e.AggregateID }
func (e *OrderCancelledEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OrderCancelledEvent) GetData() interface{}   { return e }

type OrderMatchedEvent struct {
	ID               string          `json:"id"`
	AggregateID      string          `json:"aggregate_id"`
	OrderID          string          `json:"order_id"`
	MatchedOrderID   string          `json:"matched_order_id"`
	TradeID          string          `json:"trade_id"`
	Price            float64         `json:"price"`
	Quantity         float64         `json:"quantity"`
	RemainingQuantity float64        `json:"remaining_quantity"`
	TakerSide        types.OrderSide `json:"taker_side"`
	Timestamp        time.Time       `json:"timestamp"`
}

func (e *OrderMatchedEvent) GetID() string          { return e.ID }
func (e *OrderMatchedEvent) GetType() string        { return "OrderMatched" }
func (e *OrderMatchedEvent) GetAggregateID() string { return e.AggregateID }
func (e *OrderMatchedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OrderMatchedEvent) GetData() interface{}   { return e }

type OrderFilledEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	OrderID     string    `json:"order_id"`
	UserID      string    `json:"user_id"`
	Symbol      string    `json:"symbol"`
	TotalFilled float64   `json:"total_filled"`
	AveragePrice float64  `json:"average_price"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *OrderFilledEvent) GetID() string          { return e.ID }
func (e *OrderFilledEvent) GetType() string        { return "OrderFilled" }
func (e *OrderFilledEvent) GetAggregateID() string { return e.AggregateID }
func (e *OrderFilledEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OrderFilledEvent) GetData() interface{}   { return e }

// Trade Events

type TradeCreatedEvent struct {
	ID           string          `json:"id"`
	AggregateID  string          `json:"aggregate_id"`
	TradeID      string          `json:"trade_id"`
	Symbol       string          `json:"symbol"`
	BuyOrderID   string          `json:"buy_order_id"`
	SellOrderID  string          `json:"sell_order_id"`
	Price        float64         `json:"price"`
	Quantity     float64         `json:"quantity"`
	Value        float64         `json:"value"`
	BuyUserID    string          `json:"buy_user_id"`
	SellUserID   string          `json:"sell_user_id"`
	TakerSide    types.OrderSide `json:"taker_side"`
	MakerOrderID string          `json:"maker_order_id"`
	TakerOrderID string          `json:"taker_order_id"`
	Timestamp    time.Time       `json:"timestamp"`
}

func (e *TradeCreatedEvent) GetID() string          { return e.ID }
func (e *TradeCreatedEvent) GetType() string        { return "TradeCreated" }
func (e *TradeCreatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *TradeCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *TradeCreatedEvent) GetData() interface{}   { return e }

type TradeSettledEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	TradeID     string    `json:"trade_id"`
	Status      string    `json:"status"`
	SettledAt   time.Time `json:"settled_at"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *TradeSettledEvent) GetID() string          { return e.ID }
func (e *TradeSettledEvent) GetType() string        { return "TradeSettled" }
func (e *TradeSettledEvent) GetAggregateID() string { return e.AggregateID }
func (e *TradeSettledEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *TradeSettledEvent) GetData() interface{}   { return e }

// Market Data Events

type MarketDataUpdatedEvent struct {
	ID               string    `json:"id"`
	AggregateID      string    `json:"aggregate_id"`
	Symbol           string    `json:"symbol"`
	LastPrice        float64   `json:"last_price"`
	BidPrice         float64   `json:"bid_price"`
	AskPrice         float64   `json:"ask_price"`
	Volume           float64   `json:"volume"`
	High24h          float64   `json:"high_24h"`
	Low24h           float64   `json:"low_24h"`
	Change24h        float64   `json:"change_24h"`
	ChangePercent24h float64   `json:"change_percent_24h"`
	Timestamp        time.Time `json:"timestamp"`
}

func (e *MarketDataUpdatedEvent) GetID() string          { return e.ID }
func (e *MarketDataUpdatedEvent) GetType() string        { return "MarketDataUpdated" }
func (e *MarketDataUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *MarketDataUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *MarketDataUpdatedEvent) GetData() interface{}   { return e }

type OHLCVUpdatedEvent struct {
	ID        string    `json:"id"`
	AggregateID string  `json:"aggregate_id"`
	Symbol    string    `json:"symbol"`
	Interval  string    `json:"interval"`
	Open      float64   `json:"open"`
	High      float64   `json:"high"`
	Low       float64   `json:"low"`
	Close     float64   `json:"close"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

func (e *OHLCVUpdatedEvent) GetID() string          { return e.ID }
func (e *OHLCVUpdatedEvent) GetType() string        { return "OHLCVUpdated" }
func (e *OHLCVUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *OHLCVUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *OHLCVUpdatedEvent) GetData() interface{}   { return e }

// Position Events

type PositionUpdatedEvent struct {
	ID           string    `json:"id"`
	AggregateID  string    `json:"aggregate_id"`
	UserID       string    `json:"user_id"`
	Symbol       string    `json:"symbol"`
	Quantity     float64   `json:"quantity"`
	AveragePrice float64   `json:"average_price"`
	MarketValue  float64   `json:"market_value"`
	UnrealizedPL float64   `json:"unrealized_pl"`
	RealizedPL   float64   `json:"realized_pl"`
	Timestamp    time.Time `json:"timestamp"`
}

func (e *PositionUpdatedEvent) GetID() string          { return e.ID }
func (e *PositionUpdatedEvent) GetType() string        { return "PositionUpdated" }
func (e *PositionUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *PositionUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *PositionUpdatedEvent) GetData() interface{}   { return e }

// User Events

type UserCreatedEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	UserID      string    `json:"user_id"`
	Email       string    `json:"email"`
	Username    string    `json:"username"`
	Status      string    `json:"status"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *UserCreatedEvent) GetID() string          { return e.ID }
func (e *UserCreatedEvent) GetType() string        { return "UserCreated" }
func (e *UserCreatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *UserCreatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UserCreatedEvent) GetData() interface{}   { return e }

type UserBalanceUpdatedEvent struct {
	ID          string    `json:"id"`
	AggregateID string    `json:"aggregate_id"`
	UserID      string    `json:"user_id"`
	Asset       string    `json:"asset"`
	Balance     float64   `json:"balance"`
	Available   float64   `json:"available"`
	Locked      float64   `json:"locked"`
	Timestamp   time.Time `json:"timestamp"`
}

func (e *UserBalanceUpdatedEvent) GetID() string          { return e.ID }
func (e *UserBalanceUpdatedEvent) GetType() string        { return "UserBalanceUpdated" }
func (e *UserBalanceUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *UserBalanceUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *UserBalanceUpdatedEvent) GetData() interface{}   { return e }

// System Events

type SystemHealthUpdatedEvent struct {
	ID          string                 `json:"id"`
	AggregateID string                 `json:"aggregate_id"`
	Component   string                 `json:"component"`
	Status      string                 `json:"status"`
	Metrics     map[string]interface{} `json:"metrics"`
	Timestamp   time.Time              `json:"timestamp"`
}

func (e *SystemHealthUpdatedEvent) GetID() string          { return e.ID }
func (e *SystemHealthUpdatedEvent) GetType() string        { return "SystemHealthUpdated" }
func (e *SystemHealthUpdatedEvent) GetAggregateID() string { return e.AggregateID }
func (e *SystemHealthUpdatedEvent) GetTimestamp() time.Time { return e.Timestamp }
func (e *SystemHealthUpdatedEvent) GetData() interface{}   { return e }

// Event Factory

type EventFactory struct{}

func NewEventFactory() *EventFactory {
	return &EventFactory{}
}

func (f *EventFactory) CreateOrderCreatedEvent(order *types.Order) *OrderCreatedEvent {
	return &OrderCreatedEvent{
		ID:          generateEventID(),
		AggregateID: order.ID,
		OrderID:     order.ID,
		UserID:      order.UserID,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Type:        order.Type,
		Price:       order.Price,
		Quantity:    order.Quantity,
		TimeInForce: order.TimeInForce,
		StopPrice:   order.StopPrice,
		Timestamp:   time.Now(),
	}
}

func (f *EventFactory) CreateOrderUpdatedEvent(order *types.Order) *OrderUpdatedEvent {
	return &OrderUpdatedEvent{
		ID:                generateEventID(),
		AggregateID:       order.ID,
		OrderID:           order.ID,
		FilledQuantity:    order.FilledQuantity,
		RemainingQuantity: order.RemainingQuantity,
		Status:            order.Status,
		Timestamp:         time.Now(),
	}
}

func (f *EventFactory) CreateOrderCancelledEvent(order *types.Order, reason string) *OrderCancelledEvent {
	return &OrderCancelledEvent{
		ID:          generateEventID(),
		AggregateID: order.ID,
		OrderID:     order.ID,
		UserID:      order.UserID,
		Reason:      reason,
		Timestamp:   time.Now(),
	}
}

func (f *EventFactory) CreateTradeCreatedEvent(trade *types.Trade) *TradeCreatedEvent {
	return &TradeCreatedEvent{
		ID:           generateEventID(),
		AggregateID:  trade.ID,
		TradeID:      trade.ID,
		Symbol:       trade.Symbol,
		BuyOrderID:   trade.BuyOrderID,
		SellOrderID:  trade.SellOrderID,
		Price:        trade.Price,
		Quantity:     trade.Quantity,
		Value:        trade.Value,
		BuyUserID:    trade.BuyUserID,
		SellUserID:   trade.SellUserID,
		TakerSide:    trade.TakerSide,
		MakerOrderID: trade.MakerOrderID,
		TakerOrderID: trade.TakerOrderID,
		Timestamp:    time.Now(),
	}
}

func (f *EventFactory) CreateMarketDataUpdatedEvent(marketData *types.MarketData) *MarketDataUpdatedEvent {
	return &MarketDataUpdatedEvent{
		ID:               generateEventID(),
		AggregateID:      marketData.Symbol,
		Symbol:           marketData.Symbol,
		LastPrice:        marketData.LastPrice,
		BidPrice:         marketData.BidPrice,
		AskPrice:         marketData.AskPrice,
		Volume:           marketData.Volume,
		High24h:          marketData.High24h,
		Low24h:           marketData.Low24h,
		Change24h:        marketData.Change24h,
		ChangePercent24h: marketData.ChangePercent24h,
		Timestamp:        time.Now(),
	}
}

func (f *EventFactory) CreateOHLCVUpdatedEvent(ohlcv *types.OHLCV) *OHLCVUpdatedEvent {
	return &OHLCVUpdatedEvent{
		ID:          generateEventID(),
		AggregateID: ohlcv.Symbol + "_" + ohlcv.Interval,
		Symbol:      ohlcv.Symbol,
		Interval:    ohlcv.Interval,
		Open:        ohlcv.Open,
		High:        ohlcv.High,
		Low:         ohlcv.Low,
		Close:       ohlcv.Close,
		Volume:      ohlcv.Volume,
		Timestamp:   time.Now(),
	}
}

// Event Serialization Helpers

type SerializedEvent struct {
	ID          string                 `json:"id"`
	AggregateID string                 `json:"aggregate_id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Version     int                    `json:"version"`
}

func SerializeEvent(event Event) *SerializedEvent {
	return &SerializedEvent{
		ID:          event.GetID(),
		AggregateID: event.GetAggregateID(),
		Type:        event.GetType(),
		Data:        convertToMap(event.GetData()),
		Timestamp:   event.GetTimestamp(),
		Version:     1,
	}
}

func convertToMap(data interface{}) map[string]interface{} {
	// This is a simplified implementation
	// In a real system, you'd use proper JSON marshaling/unmarshaling
	result := make(map[string]interface{})
	
	// Use reflection or JSON marshaling to convert struct to map
	// For now, return empty map as placeholder
	return result
}

// Event Stream

type EventStream struct {
	AggregateID string  `json:"aggregate_id"`
	Events      []Event `json:"events"`
	Version     int     `json:"version"`
}

func NewEventStream(aggregateID string) *EventStream {
	return &EventStream{
		AggregateID: aggregateID,
		Events:      make([]Event, 0),
		Version:     0,
	}
}

func (es *EventStream) AddEvent(event Event) {
	es.Events = append(es.Events, event)
	es.Version++
}

func (es *EventStream) GetEvents() []Event {
	return es.Events
}

func (es *EventStream) GetVersion() int {
	return es.Version
}

// Event Metadata

type EventMetadata struct {
	CorrelationID string                 `json:"correlation_id"`
	CausationID   string                 `json:"causation_id"`
	UserID        string                 `json:"user_id"`
	Source        string                 `json:"source"`
	Headers       map[string]interface{} `json:"headers"`
}

type EventWithMetadata struct {
	Event    Event         `json:"event"`
	Metadata EventMetadata `json:"metadata"`
}

func NewEventWithMetadata(event Event, metadata EventMetadata) *EventWithMetadata {
	return &EventWithMetadata{
		Event:    event,
		Metadata: metadata,
	}
}
