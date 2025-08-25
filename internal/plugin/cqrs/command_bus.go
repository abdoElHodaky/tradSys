package cqrs

import (
	"context"
	"fmt"
	"reflect"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CommandHandler is the interface that all command handlers must implement
type CommandHandler interface {
	// Type returns the type of command this handler can process
	Type() reflect.Type
	
	// Handle processes the command
	Handle(ctx context.Context, command interface{}) error
}

// CommandHandlerFunc is a function that handles a command
type CommandHandlerFunc func(ctx context.Context, command interface{}) error

// CommandMiddleware is a function that wraps a command handler
type CommandMiddleware func(CommandHandlerFunc) CommandHandlerFunc

// CommandBus dispatches commands to their handlers
type CommandBus struct {
	handlers   map[reflect.Type]CommandHandler
	middleware []CommandMiddleware
	logger     *zap.Logger
	mu         sync.RWMutex
}

// NewCommandBus creates a new command bus
func NewCommandBus(logger *zap.Logger) *CommandBus {
	return &CommandBus{
		handlers:   make(map[reflect.Type]CommandHandler),
		middleware: []CommandMiddleware{},
		logger:     logger,
	}
}

// RegisterHandler registers a command handler
func (b *CommandBus) RegisterHandler(handler CommandHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	commandType := handler.Type()
	if _, exists := b.handlers[commandType]; exists {
		return fmt.Errorf("handler already registered for command type %v", commandType)
	}
	
	b.handlers[commandType] = handler
	b.logger.Debug("Registered command handler", 
		zap.String("command_type", commandType.String()))
	
	return nil
}

// RegisterMiddleware registers middleware to be applied to all commands
func (b *CommandBus) RegisterMiddleware(middleware CommandMiddleware) {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	b.middleware = append(b.middleware, middleware)
}

// Dispatch sends a command to its handler
func (b *CommandBus) Dispatch(ctx context.Context, command interface{}) error {
	b.mu.RLock()
	commandType := reflect.TypeOf(command)
	handler, exists := b.handlers[commandType]
	middleware := make([]CommandMiddleware, len(b.middleware))
	copy(middleware, b.middleware)
	b.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("no handler registered for command type %v", commandType)
	}
	
	// Apply middleware
	next := handler.Handle
	for i := len(middleware) - 1; i >= 0; i-- {
		next = middleware[i](next)
	}
	
	return next(ctx, command)
}

// TypedCommandHandler is a generic command handler for a specific command type
type TypedCommandHandler[T any] struct {
	handleFunc func(ctx context.Context, command T) error
}

// NewTypedCommandHandler creates a new typed command handler
func NewTypedCommandHandler[T any](handleFunc func(ctx context.Context, command T) error) *TypedCommandHandler[T] {
	return &TypedCommandHandler[T]{
		handleFunc: handleFunc,
	}
}

// Type returns the type of command this handler can process
func (h *TypedCommandHandler[T]) Type() reflect.Type {
	var t T
	return reflect.TypeOf(t)
}

// Handle processes the command
func (h *TypedCommandHandler[T]) Handle(ctx context.Context, command interface{}) error {
	typedCommand, ok := command.(T)
	if !ok {
		return fmt.Errorf("invalid command type: expected %T, got %T", *new(T), command)
	}
	
	return h.handleFunc(ctx, typedCommand)
}

// LoggingMiddleware logs commands before and after execution
func LoggingMiddleware(logger *zap.Logger) CommandMiddleware {
	return func(next CommandHandlerFunc) CommandHandlerFunc {
		return func(ctx context.Context, command interface{}) error {
			start := time.Now()
			logger.Debug("Executing command", 
				zap.String("command_type", reflect.TypeOf(command).String()))
			
			err := next(ctx, command)
			
			logger.Debug("Command executed",
				zap.String("command_type", reflect.TypeOf(command).String()),
				zap.Duration("duration", time.Since(start)),
				zap.Error(err))
			
			return err
		}
	}
}

// ValidationMiddleware validates commands before execution
func ValidationMiddleware() CommandMiddleware {
	return func(next CommandHandlerFunc) CommandHandlerFunc {
		return func(ctx context.Context, command interface{}) error {
			// Check if the command implements the Validator interface
			if validator, ok := command.(interface{ Validate() error }); ok {
				if err := validator.Validate(); err != nil {
					return fmt.Errorf("command validation failed: %w", err)
				}
			}
			
			return next(ctx, command)
		}
	}
}

// RecoveryMiddleware recovers from panics in command handlers
func RecoveryMiddleware(logger *zap.Logger) CommandMiddleware {
	return func(next CommandHandlerFunc) CommandHandlerFunc {
		return func(ctx context.Context, command interface{}) (err error) {
			defer func() {
				if r := recover(); r != nil {
					logger.Error("Recovered from panic in command handler",
						zap.String("command_type", reflect.TypeOf(command).String()),
						zap.Any("panic", r))
					
					switch x := r.(type) {
					case string:
						err = fmt.Errorf("command handler panic: %s", x)
					case error:
						err = fmt.Errorf("command handler panic: %w", x)
					default:
						err = fmt.Errorf("command handler panic: %v", x)
					}
				}
			}()
			
			return next(ctx, command)
		}
	}
}
