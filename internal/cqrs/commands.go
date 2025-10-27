package cqrs

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Command represents a command in the CQRS pattern
type Command interface {
	GetID() string
	GetType() string
	GetTimestamp() time.Time
}

// CommandHandler handles commands
type CommandHandler[T Command] interface {
	Handle(ctx context.Context, command T) error
}

// CommandBus routes commands to appropriate handlers
type CommandBus interface {
	Register(commandType string, handler interface{}) error
	Execute(ctx context.Context, command Command) error
}

// Event represents an event in the system
type Event interface {
	GetID() string
	GetType() string
	GetAggregateID() string
	GetTimestamp() time.Time
	GetData() interface{}
}

// EventStore persists events
type EventStore interface {
	SaveEvents(ctx context.Context, aggregateID string, events []Event, expectedVersion int) error
	GetEvents(ctx context.Context, aggregateID string, fromVersion int) ([]Event, error)
	GetAllEvents(ctx context.Context, fromTimestamp time.Time) ([]Event, error)
}

// Order Commands

type CreateOrderCommand struct {
	ID            string            `json:"id"`
	ClientOrderID string            `json:"client_order_id"`
	UserID        string            `json:"user_id"`
	Symbol        string            `json:"symbol"`
	Side          types.OrderSide   `json:"side"`
	Type          types.OrderType   `json:"type"`
	Price         float64           `json:"price"`
	Quantity      float64           `json:"quantity"`
	TimeInForce   types.TimeInForce `json:"time_in_force"`
	StopPrice     *float64          `json:"stop_price,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
}

func (c *CreateOrderCommand) GetID() string           { return c.ID }
func (c *CreateOrderCommand) GetType() string         { return "CreateOrder" }
func (c *CreateOrderCommand) GetTimestamp() time.Time { return c.Timestamp }

type CancelOrderCommand struct {
	ID        string    `json:"id"`
	OrderID   string    `json:"order_id"`
	UserID    string    `json:"user_id"`
	Reason    string    `json:"reason"`
	Timestamp time.Time `json:"timestamp"`
}

func (c *CancelOrderCommand) GetID() string           { return c.ID }
func (c *CancelOrderCommand) GetType() string         { return "CancelOrder" }
func (c *CancelOrderCommand) GetTimestamp() time.Time { return c.Timestamp }

type UpdateOrderCommand struct {
	ID                string            `json:"id"`
	OrderID           string            `json:"order_id"`
	FilledQuantity    float64           `json:"filled_quantity"`
	RemainingQuantity float64           `json:"remaining_quantity"`
	Status            types.OrderStatus `json:"status"`
	Timestamp         time.Time         `json:"timestamp"`
}

func (c *UpdateOrderCommand) GetID() string           { return c.ID }
func (c *UpdateOrderCommand) GetType() string         { return "UpdateOrder" }
func (c *UpdateOrderCommand) GetTimestamp() time.Time { return c.Timestamp }

// Trade Commands

type CreateTradeCommand struct {
	ID           string          `json:"id"`
	Symbol       string          `json:"symbol"`
	BuyOrderID   string          `json:"buy_order_id"`
	SellOrderID  string          `json:"sell_order_id"`
	Price        float64         `json:"price"`
	Quantity     float64         `json:"quantity"`
	BuyUserID    string          `json:"buy_user_id"`
	SellUserID   string          `json:"sell_user_id"`
	TakerSide    types.OrderSide `json:"taker_side"`
	MakerOrderID string          `json:"maker_order_id"`
	TakerOrderID string          `json:"taker_order_id"`
	Timestamp    time.Time       `json:"timestamp"`
}

func (c *CreateTradeCommand) GetID() string           { return c.ID }
func (c *CreateTradeCommand) GetType() string         { return "CreateTrade" }
func (c *CreateTradeCommand) GetTimestamp() time.Time { return c.Timestamp }

// Command Handlers

type OrderCommandHandler struct {
	repository interfaces.OrderRepository
	eventStore EventStore
	eventBus   interfaces.EventPublisher
	validator  interfaces.OrderValidator
	logger     interfaces.Logger
	metrics    interfaces.MetricsCollector
}

func NewOrderCommandHandler(
	repository interfaces.OrderRepository,
	eventStore EventStore,
	eventBus interfaces.EventPublisher,
	validator interfaces.OrderValidator,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *OrderCommandHandler {
	return &OrderCommandHandler{
		repository: repository,
		eventStore: eventStore,
		eventBus:   eventBus,
		validator:  validator,
		logger:     logger,
		metrics:    metrics,
	}
}

func (h *OrderCommandHandler) HandleCreateOrder(ctx context.Context, cmd *CreateOrderCommand) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordTimer("command_handler.create_order.duration", time.Since(start), map[string]string{
			"symbol": cmd.Symbol,
		})
	}()

	h.logger.Info("Processing create order command", "command_id", cmd.ID, "order_id", cmd.ID, "user_id", cmd.UserID)

	// Create domain object
	order := &types.Order{
		ID:                cmd.ID,
		ClientOrderID:     cmd.ClientOrderID,
		UserID:            cmd.UserID,
		Symbol:            cmd.Symbol,
		Side:              cmd.Side,
		Type:              cmd.Type,
		Price:             cmd.Price,
		Quantity:          cmd.Quantity,
		RemainingQuantity: cmd.Quantity,
		TimeInForce:       cmd.TimeInForce,
		StopPrice:         cmd.StopPrice,
		Status:            types.OrderStatusPending,
		CreatedAt:         cmd.Timestamp,
		UpdatedAt:         cmd.Timestamp,
	}

	// Validate order
	if err := h.validator.ValidateOrder(order); err != nil {
		h.metrics.IncrementCounter("command_handler.create_order.validation_failed", map[string]string{
			"symbol": cmd.Symbol,
		})
		return err
	}

	// Persist to write store
	if err := h.repository.Create(ctx, order); err != nil {
		h.metrics.IncrementCounter("command_handler.create_order.persistence_failed", map[string]string{
			"symbol": cmd.Symbol,
		})
		return err
	}

	// Create and save event
	event := &OrderCreatedEvent{
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

	if err := h.eventStore.SaveEvents(ctx, order.ID, []Event{event}, 0); err != nil {
		h.logger.Error("Failed to save event", "error", err, "event_id", event.ID)
		// Don't fail the command, but log the error
	}

	// Publish event
	orderEvent := &interfaces.OrderEvent{
		Type:      interfaces.OrderEventCreated,
		Order:     order,
		Timestamp: time.Now(),
		UserID:    order.UserID,
	}

	if err := h.eventBus.PublishOrderEvent(ctx, *orderEvent); err != nil {
		h.logger.Error("Failed to publish order event", "error", err, "order_id", order.ID)
	}

	h.metrics.IncrementCounter("command_handler.create_order.success", map[string]string{
		"symbol": cmd.Symbol,
	})

	h.logger.Info("Create order command processed successfully", "command_id", cmd.ID, "order_id", order.ID)
	return nil
}

func (h *OrderCommandHandler) HandleCancelOrder(ctx context.Context, cmd *CancelOrderCommand) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordTimer("command_handler.cancel_order.duration", time.Since(start), nil)
	}()

	h.logger.Info("Processing cancel order command", "command_id", cmd.ID, "order_id", cmd.OrderID)

	// Get existing order
	order, err := h.repository.GetByID(ctx, cmd.OrderID)
	if err != nil {
		h.metrics.IncrementCounter("command_handler.cancel_order.not_found", nil)
		return err
	}

	// Validate cancellation
	if !order.IsActive() {
		h.metrics.IncrementCounter("command_handler.cancel_order.invalid_state", map[string]string{
			"status": string(order.Status),
		})
		return &InvalidOrderStateError{
			OrderID: cmd.OrderID,
			Status:  order.Status,
			Message: "order cannot be cancelled in current state",
		}
	}

	// Update order status
	order.Status = types.OrderStatusCanceled
	order.UpdatedAt = cmd.Timestamp

	// Persist changes
	if err := h.repository.Update(ctx, order); err != nil {
		h.metrics.IncrementCounter("command_handler.cancel_order.persistence_failed", nil)
		return err
	}

	// Create and save event
	event := &OrderCancelledEvent{
		ID:          generateEventID(),
		AggregateID: order.ID,
		OrderID:     order.ID,
		UserID:      order.UserID,
		Reason:      cmd.Reason,
		Timestamp:   time.Now(),
	}

	if err := h.eventStore.SaveEvents(ctx, order.ID, []Event{event}, -1); err != nil {
		h.logger.Error("Failed to save event", "error", err, "event_id", event.ID)
	}

	// Publish event
	orderEvent := &interfaces.OrderEvent{
		Type:      interfaces.OrderEventCanceled,
		Order:     order,
		Timestamp: time.Now(),
		UserID:    order.UserID,
	}

	if err := h.eventBus.PublishOrderEvent(ctx, *orderEvent); err != nil {
		h.logger.Error("Failed to publish order event", "error", err, "order_id", order.ID)
	}

	h.metrics.IncrementCounter("command_handler.cancel_order.success", nil)
	h.logger.Info("Cancel order command processed successfully", "command_id", cmd.ID, "order_id", cmd.OrderID)
	return nil
}

func (h *OrderCommandHandler) HandleUpdateOrder(ctx context.Context, cmd *UpdateOrderCommand) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordTimer("command_handler.update_order.duration", time.Since(start), nil)
	}()

	// Get existing order
	order, err := h.repository.GetByID(ctx, cmd.OrderID)
	if err != nil {
		return err
	}

	// Update order fields
	order.FilledQuantity = cmd.FilledQuantity
	order.RemainingQuantity = cmd.RemainingQuantity
	order.Status = cmd.Status
	order.UpdatedAt = cmd.Timestamp

	// Persist changes
	if err := h.repository.Update(ctx, order); err != nil {
		return err
	}

	// Create and save event
	event := &OrderUpdatedEvent{
		ID:                generateEventID(),
		AggregateID:       order.ID,
		OrderID:           order.ID,
		FilledQuantity:    cmd.FilledQuantity,
		RemainingQuantity: cmd.RemainingQuantity,
		Status:            cmd.Status,
		Timestamp:         time.Now(),
	}

	if err := h.eventStore.SaveEvents(ctx, order.ID, []Event{event}, -1); err != nil {
		h.logger.Error("Failed to save event", "error", err, "event_id", event.ID)
	}

	// Publish event
	orderEvent := &interfaces.OrderEvent{
		Type:      interfaces.OrderEventUpdated,
		Order:     order,
		Timestamp: time.Now(),
		UserID:    order.UserID,
	}

	if err := h.eventBus.PublishOrderEvent(ctx, *orderEvent); err != nil {
		h.logger.Error("Failed to publish order event", "error", err, "order_id", order.ID)
	}

	return nil
}

// Trade Command Handler

type TradeCommandHandler struct {
	repository interfaces.TradeRepository
	eventStore EventStore
	eventBus   interfaces.EventPublisher
	logger     interfaces.Logger
	metrics    interfaces.MetricsCollector
}

func NewTradeCommandHandler(
	repository interfaces.TradeRepository,
	eventStore EventStore,
	eventBus interfaces.EventPublisher,
	logger interfaces.Logger,
	metrics interfaces.MetricsCollector,
) *TradeCommandHandler {
	return &TradeCommandHandler{
		repository: repository,
		eventStore: eventStore,
		eventBus:   eventBus,
		logger:     logger,
		metrics:    metrics,
	}
}

func (h *TradeCommandHandler) HandleCreateTrade(ctx context.Context, cmd *CreateTradeCommand) error {
	start := time.Now()
	defer func() {
		h.metrics.RecordTimer("command_handler.create_trade.duration", time.Since(start), map[string]string{
			"symbol": cmd.Symbol,
		})
	}()

	// Create domain object
	trade := &types.Trade{
		ID:           cmd.ID,
		Symbol:       cmd.Symbol,
		BuyOrderID:   cmd.BuyOrderID,
		SellOrderID:  cmd.SellOrderID,
		Price:        cmd.Price,
		Quantity:     cmd.Quantity,
		Value:        cmd.Price * cmd.Quantity,
		BuyUserID:    cmd.BuyUserID,
		SellUserID:   cmd.SellUserID,
		TakerSide:    cmd.TakerSide,
		MakerOrderID: cmd.MakerOrderID,
		TakerOrderID: cmd.TakerOrderID,
		Timestamp:    cmd.Timestamp,
	}

	// Persist to write store
	if err := h.repository.Create(ctx, trade); err != nil {
		return err
	}

	// Create and save event
	event := &TradeCreatedEvent{
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

	if err := h.eventStore.SaveEvents(ctx, trade.ID, []Event{event}, 0); err != nil {
		h.logger.Error("Failed to save event", "error", err, "event_id", event.ID)
	}

	// Publish event
	tradeEvent := &interfaces.TradeEvent{
		Type:      interfaces.TradeEventExecuted,
		Trade:     trade,
		Timestamp: time.Now(),
	}

	if err := h.eventBus.PublishTradeEvent(ctx, tradeEvent); err != nil {
		h.logger.Error("Failed to publish trade event", "error", err, "trade_id", trade.ID)
	}

	return nil
}

// Error types

type InvalidOrderStateError struct {
	OrderID string
	Status  types.OrderStatus
	Message string
}

func (e *InvalidOrderStateError) Error() string {
	return e.Message
}

// Helper functions

func generateEventID() string {
	return "event_" + generateID()
}

func generateID() string {
	return time.Now().Format("20060102150405") + "_" + randomString(8)
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}
