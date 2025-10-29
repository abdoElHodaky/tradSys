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

// Note: Concrete processor implementations are now in separate files:
// - order_processors.go: MarketOrderProcessor, LimitOrderProcessor
// - stop_processors.go: StopLimitOrderProcessor, StopMarketOrderProcessor, DefaultOrderProcessor

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
