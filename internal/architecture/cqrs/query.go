package cqrs

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

// Query represents a query in the CQRS pattern
type Query interface {
	// QueryName returns the name of the query
	QueryName() string
}

// QueryHandler represents a query handler that returns a result
type QueryHandler interface {
	// Handle handles a query and returns a result
	Handle(ctx context.Context, query Query) (interface{}, error)
}

// QueryHandlerFunc is a function that implements the QueryHandler interface
type QueryHandlerFunc func(ctx context.Context, query Query) (interface{}, error)

// Handle handles a query and returns a result
func (f QueryHandlerFunc) Handle(ctx context.Context, query Query) (interface{}, error) {
	return f(ctx, query)
}

// QueryBus dispatches queries to their handlers
type QueryBus struct {
	handlers map[string]QueryHandler
	mu       sync.RWMutex
}

// NewQueryBus creates a new query bus
func NewQueryBus() *QueryBus {
	return &QueryBus{
		handlers: make(map[string]QueryHandler),
	}
}

// RegisterHandler registers a handler for a specific query type
func (qb *QueryBus) RegisterHandler(queryType reflect.Type, handler QueryHandler) error {
	if queryType.Kind() != reflect.Ptr {
		return errors.New("query type must be a pointer type")
	}

	queryName := queryType.Elem().Name()

	qb.mu.Lock()
	defer qb.mu.Unlock()

	if _, exists := qb.handlers[queryName]; exists {
		return errors.New("handler already registered for query: " + queryName)
	}

	qb.handlers[queryName] = handler
	return nil
}

// RegisterHandlerFunc registers a handler function for a specific query type
func (qb *QueryBus) RegisterHandlerFunc(queryType reflect.Type, handler func(ctx context.Context, query Query) (interface{}, error)) error {
	return qb.RegisterHandler(queryType, QueryHandlerFunc(handler))
}

// Dispatch dispatches a query to its handler and returns the result
func (qb *QueryBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
	if query == nil {
		return nil, errors.New("query cannot be nil")
	}

	queryName := query.QueryName()

	qb.mu.RLock()
	handler, exists := qb.handlers[queryName]
	qb.mu.RUnlock()

	if !exists {
		return nil, errors.New("no handler registered for query: " + queryName)
	}

	return handler.Handle(ctx, query)
}

// Register is an alias for RegisterHandler for compatibility
func (qb *QueryBus) Register(queryType reflect.Type, handler QueryHandler) error {
	return qb.RegisterHandler(queryType, handler)
}
