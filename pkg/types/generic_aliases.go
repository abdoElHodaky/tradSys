package types

import (
	"context"
	"fmt"
	"time"
)

// Go 1.24 Generic Type Aliases for cleaner API definitions

// Attributes represents a generic key-value map for flexible data storage
// Using Go 1.24's generic type aliases for better type safety
type Attributes[K comparable, V any] = map[K]V

// StringAttributes is a common alias for string-keyed attributes
type StringAttributes = Attributes[string, interface{}]

// Metadata represents metadata with string keys and any values
type Metadata = Attributes[string, interface{}]

// OrderAttributes represents order-specific attributes
type OrderAttributes = Attributes[string, interface{}]

// Set represents a generic set implementation using Go 1.24 type aliases
type Set[T comparable] = map[T]struct{}

// StringSet is a common alias for string sets
type StringSet = Set[string]

// OrderIDSet represents a set of order IDs
type OrderIDSet = Set[string]

// SymbolSet represents a set of trading symbols
type SymbolSet = Set[string]

// EventHandler represents a generic event handler function
type EventHandler[T any] = func(ctx context.Context, event T) error

// QueryHandler represents a generic query handler function
type QueryHandler[Q any, R any] = func(ctx context.Context, query Q) (R, error)

// CommandHandler represents a generic command handler function
type CommandHandler[C any] = func(ctx context.Context, command C) error

// Repository represents a generic repository interface
type Repository[T any, ID comparable] interface {
	Create(ctx context.Context, entity T) error
	GetByID(ctx context.Context, id ID) (T, error)
	Update(ctx context.Context, entity T) error
	Delete(ctx context.Context, id ID) error
	List(ctx context.Context, limit, offset int) ([]T, error)
}

// OrderRepository is a specialized repository for orders
type OrderRepository = Repository[Order, string]

// TradeRepository is a specialized repository for trades
type TradeRepository = Repository[Trade, string]

// PositionRepository is a specialized repository for positions
type PositionRepository = Repository[Position, string]

// Cache represents a generic cache interface
type Cache[K comparable, V any] interface {
	Get(key K) (V, bool)
	Set(key K, value V, ttl time.Duration)
	Delete(key K)
	Clear()
	Size() int
}

// OrderCache is a specialized cache for orders
type OrderCache = Cache[string, Order]

// PriceCache is a specialized cache for prices
type PriceCache = Cache[string, float64]

// Validator represents a generic validator function
type Validator[T any] = func(ctx context.Context, data T) error

// OrderValidator is a specialized validator for orders
type OrderValidator = Validator[Order]

// TradeValidator is a specialized validator for trades
type TradeValidator = Validator[Trade]

// Transformer represents a generic data transformer
type Transformer[From any, To any] = func(from From) (To, error)

// OrderTransformer transforms between different order representations
type OrderTransformer[T any] = Transformer[Order, T]

// TradeTransformer transforms between different trade representations
type TradeTransformer[T any] = Transformer[Trade, T]

// Publisher represents a generic event publisher
type Publisher[T any] interface {
	Publish(ctx context.Context, topic string, event T) error
}

// Subscriber represents a generic event subscriber
type Subscriber[T any] interface {
	Subscribe(ctx context.Context, topic string, handler EventHandler[T]) error
	Unsubscribe(ctx context.Context, topic string) error
}

// EventBus combines publisher and subscriber interfaces
type EventBus[T any] interface {
	Publisher[T]
	Subscriber[T]
}

// OrderEventBus is a specialized event bus for order events
type OrderEventBus = EventBus[Order]

// TradeEventBus is a specialized event bus for trade events
type TradeEventBus = EventBus[Trade]

// Result represents a generic result type with error handling
type Result[T any] struct {
	Value T
	Error error
}

// NewResult creates a new result with a value
func NewResult[T any](value T) Result[T] {
	return Result[T]{Value: value}
}

// NewResultWithError creates a new result with an error
func NewResultWithError[T any](err error) Result[T] {
	var zero T
	return Result[T]{Value: zero, Error: err}
}

// IsSuccess returns true if the result has no error
func (r Result[T]) IsSuccess() bool {
	return r.Error == nil
}

// IsError returns true if the result has an error
func (r Result[T]) IsError() bool {
	return r.Error != nil
}

// Unwrap returns the value if successful, otherwise panics
func (r Result[T]) Unwrap() T {
	if r.Error != nil {
		panic(r.Error)
	}
	return r.Value
}

// UnwrapOr returns the value if successful, otherwise returns the default
func (r Result[T]) UnwrapOr(defaultValue T) T {
	if r.Error != nil {
		return defaultValue
	}
	return r.Value
}

// Option represents a generic optional type
type Option[T any] struct {
	value   T
	present bool
}

// Some creates an Option with a value
func Some[T any](value T) Option[T] {
	return Option[T]{value: value, present: true}
}

// None creates an empty Option
func None[T any]() Option[T] {
	var zero T
	return Option[T]{value: zero, present: false}
}

// IsSome returns true if the option has a value
func (o Option[T]) IsSome() bool {
	return o.present
}

// IsNone returns true if the option has no value
func (o Option[T]) IsNone() bool {
	return !o.present
}

// Unwrap returns the value if present, otherwise panics
func (o Option[T]) Unwrap() T {
	if !o.present {
		panic("called Unwrap on None option")
	}
	return o.value
}

// UnwrapOr returns the value if present, otherwise returns the default
func (o Option[T]) UnwrapOr(defaultValue T) T {
	if !o.present {
		return defaultValue
	}
	return o.value
}

// Map applies a function to the value if present
func (o Option[T]) Map(f func(T) T) Option[T] {
	if !o.present {
		return None[T]()
	}
	return Some(f(o.value))
}

// HealthStatus represents the health status of a component
type HealthStatus struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	Details   Metadata  `json:"details,omitempty"`
}

// Service errors using Go 1.24 patterns
var (
	ErrServiceAlreadyStarted = NewError("service_already_started", "service already started")
	ErrServiceNotRunning     = NewError("service_not_running", "service not running")
)

// TradingError represents a trading-specific error with enhanced context
type TradingError struct {
	Code      string    `json:"code"`
	Message   string    `json:"message"`
	Details   Metadata  `json:"details,omitempty"`
	Timestamp time.Time `json:"timestamp"`
	Cause     error     `json:"cause,omitempty"`
}

// Error implements the error interface
func (e *TradingError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *TradingError) Unwrap() error {
	return e.Cause
}

// NewError creates a new TradingError
func NewError(code, message string) *TradingError {
	return &TradingError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		Details:   make(Metadata),
	}
}

// NewErrorWithCause creates a new TradingError with a cause
func NewErrorWithCause(code, message string, cause error) *TradingError {
	return &TradingError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
		Details:   make(Metadata),
	}
}

// WithDetail adds a detail to the error
func (e *TradingError) WithDetail(key string, value interface{}) *TradingError {
	e.Details[key] = value
	return e
}
