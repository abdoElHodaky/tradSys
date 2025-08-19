package query

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/zap"
)

// Query represents a query in the CQRS pattern
type Query interface {
	QueryName() string
}

// Handler represents a query handler in the CQRS pattern
type Handler interface {
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// HandlerFunc is a function that implements the Handler interface
type HandlerFunc func(ctx context.Context, query Query) (interface{}, error)

// Handle handles a query
func (f HandlerFunc) Handle(ctx context.Context, query Query) (interface{}, error) {
	return f(ctx, query)
}

// QueryBus represents a query bus in the CQRS pattern
type QueryBus struct {
	handlers map[string]Handler
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewQueryBus creates a new query bus
func NewQueryBus() *QueryBus {
	return &QueryBus{
		handlers: make(map[string]Handler),
	}
}

// SetLogger sets the logger for the query bus
func (b *QueryBus) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// Register registers a handler for a query
func (b *QueryBus) Register(queryType reflect.Type, handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	// Create a zero value of the query type
	query, ok := reflect.New(queryType).Elem().Interface().(Query)
	if !ok {
		return fmt.Errorf("query type %s does not implement Query interface", queryType.Name())
	}

	// Get the query name
	queryName := query.QueryName()

	// Check if a handler is already registered for the query
	if _, exists := b.handlers[queryName]; exists {
		return fmt.Errorf("handler already registered for query %s", queryName)
	}

	// Register the handler
	b.handlers[queryName] = handler

	if b.logger != nil {
		b.logger.Info("Registered query handler",
			zap.String("query", queryName),
			zap.String("handler", reflect.TypeOf(handler).String()))
	}

	return nil
}

// RegisterFunc registers a handler function for a query
func (b *QueryBus) RegisterFunc(queryType reflect.Type, handler func(ctx context.Context, query Query) (interface{}, error)) error {
	return b.Register(queryType, HandlerFunc(handler))
}

// Dispatch dispatches a query to its handler
func (b *QueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	// Get the query name
	queryName := query.QueryName()

	// Get the handler for the query
	handler, exists := b.handlers[queryName]
	if !exists {
		return nil, fmt.Errorf("no handler registered for query %s", queryName)
	}

	// Handle the query
	if b.logger != nil {
		b.logger.Debug("Dispatching query",
			zap.String("query", queryName),
			zap.String("handler", reflect.TypeOf(handler).String()))
	}

	return handler.Handle(ctx, query)
}

// GetOrderQuery represents a query to get an order
type GetOrderQuery struct {
	OrderID string
	UserID  string
}

// QueryName returns the name of the query
func (q *GetOrderQuery) QueryName() string {
	return "GetOrder"
}

// GetOrdersQuery represents a query to get orders
type GetOrdersQuery struct {
	UserID    string
	AccountID string
	Symbol    string
	Side      string
	Status    string
	StartTime int64
	EndTime   int64
	Limit     int32
	Offset    int32
}

// QueryName returns the name of the query
func (q *GetOrdersQuery) QueryName() string {
	return "GetOrders"
}

// GetMarketDataQuery represents a query to get market data
type GetMarketDataQuery struct {
	Symbol   string
	Interval string
}

// QueryName returns the name of the query
func (q *GetMarketDataQuery) QueryName() string {
	return "GetMarketData"
}

// GetHistoricalDataQuery represents a query to get historical market data
type GetHistoricalDataQuery struct {
	Symbol    string
	Interval  string
	StartTime int64
	EndTime   int64
	Limit     int32
}

// QueryName returns the name of the query
func (q *GetHistoricalDataQuery) QueryName() string {
	return "GetHistoricalData"
}

