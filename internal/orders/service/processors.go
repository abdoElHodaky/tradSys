package service

import (
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// OrderProcessor defines the interface for processing different order types
type OrderProcessor interface {
	Process(order *Order) error
	Validate(order *Order) error
	GetType() OrderType
	CalculateExecutionPrice(order *Order, marketPrice float64) float64
}

// ProcessorRegistry manages order processors using polymorphism instead of switch statements
type ProcessorRegistry struct {
	processors map[OrderType]OrderProcessor
}

// NewProcessorRegistry creates a new processor registry
func NewProcessorRegistry() *ProcessorRegistry {
	registry := &ProcessorRegistry{
		processors: make(map[OrderType]OrderProcessor),
	}
	
	// Register processors for each order type
	registry.RegisterProcessor(&MarketOrderProcessor{})
	registry.RegisterProcessor(&LimitOrderProcessor{})
	registry.RegisterProcessor(&StopLimitOrderProcessor{})
	registry.RegisterProcessor(&StopMarketOrderProcessor{})
	
	return registry
}

// RegisterProcessor registers a processor for an order type
func (r *ProcessorRegistry) RegisterProcessor(processor OrderProcessor) {
	r.processors[processor.GetType()] = processor
}

// GetProcessor returns the appropriate processor for an order type
func (r *ProcessorRegistry) GetProcessor(orderType OrderType) OrderProcessor {
	if processor, exists := r.processors[orderType]; exists {
		return processor
	}
	return &DefaultOrderProcessor{}
}

// ProcessOrder processes an order using the appropriate processor
func (r *ProcessorRegistry) ProcessOrder(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	processor := r.GetProcessor(order.Type)
	
	if err := processor.Validate(order); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}
	
	return processor.Process(order)
}

// MarketOrderProcessor handles market orders
type MarketOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *MarketOrderProcessor) GetType() OrderType {
	return OrderTypeMarket
}

// Validate validates a market order
func (p *MarketOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	
	// Market orders don't need price validation
	return nil
}

// Process processes a market order
func (p *MarketOrderProcessor) Process(order *Order) error {
	// Market orders execute immediately at current market price
	order.Status = OrderStatusFilled
	order.UpdatedAt = time.Now()
	
	// Create trade record
	trade := &Trade{
		ID:                  uuid.New().String(),
		OrderID:             order.ID,
		Symbol:              order.Symbol,
		Side:                order.Side,
		Price:               order.Price, // Set by market data
		Quantity:            order.Quantity,
		ExecutedAt:          time.Now(),
		Fee:                 p.calculateFee(order),
		FeeCurrency:         "USD",
		CounterPartyOrderID: "", // Set by matching engine
	}
	
	order.Trades = append(order.Trades, trade)
	order.FilledQuantity = order.Quantity
	
	return nil
}

// CalculateExecutionPrice calculates execution price for market orders
func (p *MarketOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Market orders execute at current market price
	return marketPrice
}

// calculateFee calculates trading fee for market orders
func (p *MarketOrderProcessor) calculateFee(order *Order) float64 {
	// Market orders typically have higher fees
	return order.Price * order.Quantity * 0.001 // 0.1% fee
}

// LimitOrderProcessor handles limit orders
type LimitOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *LimitOrderProcessor) GetType() OrderType {
	return OrderTypeLimit
}

// Validate validates a limit order
func (p *LimitOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	
	if order.Price <= 0 {
		return errors.New("price must be positive for limit orders")
	}
	
	return nil
}

// Process processes a limit order
func (p *LimitOrderProcessor) Process(order *Order) error {
	// Limit orders are placed in the order book
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()
	
	// Check if order can be immediately filled
	if p.canFillImmediately(order) {
		return p.fillOrder(order)
	}
	
	return nil
}

// CalculateExecutionPrice calculates execution price for limit orders
func (p *LimitOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Limit orders execute at their specified price or better
	if order.Side == OrderSideBuy {
		if marketPrice <= order.Price {
			return marketPrice
		}
	} else {
		if marketPrice >= order.Price {
			return marketPrice
		}
	}
	return order.Price
}

// canFillImmediately checks if limit order can be filled immediately
func (p *LimitOrderProcessor) canFillImmediately(order *Order) bool {
	// Simplified logic - in real implementation, check against order book
	return false
}

// fillOrder fills a limit order
func (p *LimitOrderProcessor) fillOrder(order *Order) error {
	order.Status = OrderStatusFilled
	order.FilledQuantity = order.Quantity
	order.UpdatedAt = time.Now()
	
	trade := &Trade{
		ID:          uuid.New().String(),
		OrderID:     order.ID,
		Symbol:      order.Symbol,
		Side:        order.Side,
		Price:       order.Price,
		Quantity:    order.Quantity,
		ExecutedAt:  time.Now(),
		Fee:         p.calculateFee(order),
		FeeCurrency: "USD",
	}
	
	order.Trades = append(order.Trades, trade)
	return nil
}

// calculateFee calculates trading fee for limit orders
func (p *LimitOrderProcessor) calculateFee(order *Order) float64 {
	// Limit orders typically have lower fees
	return order.Price * order.Quantity * 0.0005 // 0.05% fee
}

// StopLimitOrderProcessor handles stop-limit orders
type StopLimitOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *StopLimitOrderProcessor) GetType() OrderType {
	return OrderTypeStopLimit
}

// Validate validates a stop-limit order
func (p *StopLimitOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	
	if order.Price <= 0 {
		return errors.New("price must be positive for stop-limit orders")
	}
	
	if order.StopPrice <= 0 {
		return errors.New("stop price must be positive for stop-limit orders")
	}
	
	return nil
}

// Process processes a stop-limit order
func (p *StopLimitOrderProcessor) Process(order *Order) error {
	// Stop-limit orders wait for trigger condition
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()
	
	// In real implementation, monitor market price for trigger
	return nil
}

// CalculateExecutionPrice calculates execution price for stop-limit orders
func (p *StopLimitOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Stop-limit orders execute at limit price after trigger
	return order.Price
}

// StopMarketOrderProcessor handles stop-market orders
type StopMarketOrderProcessor struct{}

// GetType returns the order type this processor handles
func (p *StopMarketOrderProcessor) GetType() OrderType {
	return OrderTypeStopMarket
}

// Validate validates a stop-market order
func (p *StopMarketOrderProcessor) Validate(order *Order) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	if order.Quantity <= 0 {
		return errors.New("quantity must be positive")
	}
	
	if order.StopPrice <= 0 {
		return errors.New("stop price must be positive for stop-market orders")
	}
	
	return nil
}

// Process processes a stop-market order
func (p *StopMarketOrderProcessor) Process(order *Order) error {
	// Stop-market orders wait for trigger condition
	order.Status = OrderStatusPending
	order.UpdatedAt = time.Now()
	
	return nil
}

// CalculateExecutionPrice calculates execution price for stop-market orders
func (p *StopMarketOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	// Stop-market orders execute at market price after trigger
	return marketPrice
}

// DefaultOrderProcessor handles unknown order types
type DefaultOrderProcessor struct{}

// GetType returns the default order type
func (p *DefaultOrderProcessor) GetType() OrderType {
	return ""
}

// Validate validates unknown order types
func (p *DefaultOrderProcessor) Validate(order *Order) error {
	return fmt.Errorf("unsupported order type: %s", order.Type)
}

// Process processes unknown order types
func (p *DefaultOrderProcessor) Process(order *Order) error {
	return fmt.Errorf("cannot process unsupported order type: %s", order.Type)
}

// CalculateExecutionPrice calculates execution price for unknown order types
func (p *DefaultOrderProcessor) CalculateExecutionPrice(order *Order, marketPrice float64) float64 {
	return 0
}

// OrderStateMachine handles order state transitions using command pattern
type OrderStateMachine struct {
	transitions map[StateEventPair]StateTransition
}

// StateEventPair represents a state-event combination
type StateEventPair struct {
	State string
	Event string
}

// StateTransition represents a state transition
type StateTransition struct {
	ToState string
	Action  func(*Order) error
}

// NewOrderStateMachine creates a new order state machine
func NewOrderStateMachine() *OrderStateMachine {
	sm := &OrderStateMachine{
		transitions: make(map[StateEventPair]StateTransition),
	}
	
	// Define state transitions instead of using nested switch statements
	sm.addTransition("NEW", "VALIDATE", "PENDING", sm.validateOrder)
	sm.addTransition("NEW", "REJECT", "REJECTED", sm.rejectOrder)
	sm.addTransition("PENDING", "EXECUTE", "EXECUTED", sm.executeOrder)
	sm.addTransition("PENDING", "CANCEL", "CANCELLED", sm.cancelOrder)
	sm.addTransition("PENDING", "EXPIRE", "EXPIRED", sm.expireOrder)
	sm.addTransition("EXECUTED", "SETTLE", "SETTLED", sm.settleOrder)
	sm.addTransition("PARTIALLY_FILLED", "FILL", "FILLED", sm.fillOrder)
	sm.addTransition("PARTIALLY_FILLED", "CANCEL", "CANCELLED", sm.cancelOrder)
	
	return sm
}

// addTransition adds a state transition
func (sm *OrderStateMachine) addTransition(fromState, event, toState string, action func(*Order) error) {
	pair := StateEventPair{State: fromState, Event: event}
	sm.transitions[pair] = StateTransition{
		ToState: toState,
		Action:  action,
	}
}

// HandleEvent handles an order state event
func (sm *OrderStateMachine) HandleEvent(order *Order, event string) error {
	if order == nil {
		return errors.New("order cannot be nil")
	}
	
	pair := StateEventPair{State: string(order.Status), Event: event}
	transition, exists := sm.transitions[pair]
	
	if !exists {
		return fmt.Errorf("invalid transition: %s -> %s", order.Status, event)
	}
	
	// Execute transition action if defined
	if transition.Action != nil {
		if err := transition.Action(order); err != nil {
			return fmt.Errorf("transition action failed: %w", err)
		}
	}
	
	// Update order state
	order.Status = OrderStatus(transition.ToState)
	order.UpdatedAt = time.Now()
	
	return nil
}

// State transition actions

func (sm *OrderStateMachine) validateOrder(order *Order) error {
	// Validation logic
	return nil
}

func (sm *OrderStateMachine) rejectOrder(order *Order) error {
	// Rejection logic
	return nil
}

func (sm *OrderStateMachine) executeOrder(order *Order) error {
	// Execution logic
	return nil
}

func (sm *OrderStateMachine) cancelOrder(order *Order) error {
	// Cancellation logic
	return nil
}

func (sm *OrderStateMachine) expireOrder(order *Order) error {
	// Expiration logic
	return nil
}

func (sm *OrderStateMachine) settleOrder(order *Order) error {
	// Settlement logic
	return nil
}

func (sm *OrderStateMachine) fillOrder(order *Order) error {
	// Fill logic
	return nil
}

// ErrorCodeMapper handles error code mapping using map-based dispatch instead of switch
type ErrorCodeMapper struct {
	codeMap map[int]string
}

// NewErrorCodeMapper creates a new error code mapper
func NewErrorCodeMapper() *ErrorCodeMapper {
	return &ErrorCodeMapper{
		codeMap: map[int]string{
			1001: "INVALID_SYMBOL",
			1002: "INVALID_QUANTITY",
			1003: "INVALID_PRICE",
			1004: "INSUFFICIENT_BALANCE",
			1005: "ORDER_NOT_FOUND",
			1006: "DUPLICATE_ORDER",
			1007: "MARKET_CLOSED",
			1008: "INVALID_ORDER_TYPE",
			1009: "POSITION_LIMIT_EXCEEDED",
			1010: "RISK_LIMIT_EXCEEDED",
			2001: "NETWORK_ERROR",
			2002: "TIMEOUT_ERROR",
			2003: "SYSTEM_ERROR",
			2004: "DATABASE_ERROR",
			2005: "VALIDATION_ERROR",
			3001: "UNAUTHORIZED",
			3002: "FORBIDDEN",
			3003: "RATE_LIMITED",
			3004: "MAINTENANCE_MODE",
		},
	}
}

// MapErrorCode maps internal error codes to external error messages
func (m *ErrorCodeMapper) MapErrorCode(internalCode int) string {
	if code, exists := m.codeMap[internalCode]; exists {
		return code
	}
	return "UNKNOWN_ERROR"
}

// GetErrorCode returns the internal code for an error message
func (m *ErrorCodeMapper) GetErrorCode(errorMessage string) int {
	for code, message := range m.codeMap {
		if message == errorMessage {
			return code
		}
	}
	return 9999 // Unknown error code
}

// AddErrorMapping adds a new error code mapping
func (m *ErrorCodeMapper) AddErrorMapping(code int, message string) {
	m.codeMap[code] = message
}
