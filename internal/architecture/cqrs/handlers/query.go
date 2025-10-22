package handlers

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Query represents a query in the CQRS pattern
type Query interface {
	// QueryName returns the name of the query
	QueryName() string
}

// Handler represents a query handler
type Handler[T any] interface {
	// Handle handles a query and returns a result
	Handle(ctx context.Context, query Query) (T, error)
}

// HandlerFunc is a function that implements the Handler interface
type HandlerFunc[T any] func(ctx context.Context, query Query) (T, error)

// Handle handles a query and returns a result
func (f HandlerFunc[T]) Handle(ctx context.Context, query Query) (T, error) {
	return f(ctx, query)
}

// Bus represents a query bus
type Bus interface {
	// Register registers a handler for a query
	Register(queryType reflect.Type, handler interface{}) error
	
	// Dispatch dispatches a query to its handler and returns a result
	Dispatch(ctx context.Context, query Query) (interface{}, error)
}

// DefaultBus provides a default implementation of the Bus interface
type DefaultBus struct {
	handlers map[string]interface{}
	mu       sync.RWMutex
}

// NewDefaultBus creates a new default query bus
func NewDefaultBus() *DefaultBus {
	return &DefaultBus{
		handlers: make(map[string]interface{}),
	}
}

// Register registers a handler for a query
func (b *DefaultBus) Register(queryType reflect.Type, handler interface{}) error {
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
	
	// Check if the handler implements the Handler interface
	handlerType := reflect.TypeOf(handler)
	if handlerType.Kind() != reflect.Func && !handlerImplementsHandler(handlerType) {
		return fmt.Errorf("handler does not implement Handler interface")
	}
	
	// Register the handler
	b.handlers[queryName] = handler
	
	return nil
}

// Dispatch dispatches a query to its handler and returns a result
func (b *DefaultBus) Dispatch(ctx context.Context, query Query) (interface{}, error) {
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
	return b.invokeHandler(ctx, handler, query)
}

// invokeHandler invokes a handler with a query
func (b *DefaultBus) invokeHandler(ctx context.Context, handler interface{}, query Query) (interface{}, error) {
	// Check if the handler is a function
	if handlerFunc, ok := handler.(func(context.Context, Query) (interface{}, error)); ok {
		return handlerFunc(ctx, query)
	}
	
	// Get the handler type
	handlerType := reflect.TypeOf(handler)
	
	// Check if the handler implements the Handler interface
	if handlerImplementsHandler(handlerType) {
		// Get the Handle method
		method := reflect.ValueOf(handler).MethodByName("Handle")
		
		// Call the Handle method
		results := method.Call([]reflect.Value{
			reflect.ValueOf(ctx),
			reflect.ValueOf(query),
		})
		
		// Check for an error
		if !results[1].IsNil() {
			return nil, results[1].Interface().(error)
		}
		
		// Return the result
		return results[0].Interface(), nil
	}
	
	return nil, fmt.Errorf("invalid handler type")
}

// handlerImplementsHandler checks if a type implements the Handler interface
func handlerImplementsHandler(t reflect.Type) bool {
	// Check if the type has a Handle method
	if method, exists := t.MethodByName("Handle"); exists {
		// Check if the method has the correct signature
		if method.Type.NumIn() == 3 && method.Type.NumOut() == 2 {
			// Check if the first parameter is context.Context
			if method.Type.In(1).AssignableTo(reflect.TypeOf((*context.Context)(nil)).Elem()) {
				// Check if the second parameter is Query
				if method.Type.In(2).AssignableTo(reflect.TypeOf((*Query)(nil)).Elem()) {
					// Check if the second return value is error
					if method.Type.Out(1).AssignableTo(reflect.TypeOf((*error)(nil)).Elem()) {
						return true
					}
				}
			}
		}
	}
	
	return false
}

// Common errors
var (
	ErrQueryNotFound   = errors.New("query not found")
	ErrHandlerNotFound = errors.New("handler not found")
)

