package cqrs

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/zap"
)

// CommandHandler is a function that handles a command
type CommandHandler func(ctx context.Context, command interface{}) (interface{}, error)

// CommandBus is a simple command bus implementation
type CommandBus struct {
	handlers map[string]CommandHandler
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewCommandBus creates a new command bus
func NewCommandBus(logger *zap.Logger) *CommandBus {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &CommandBus{
		handlers: make(map[string]CommandHandler),
		logger:   logger,
	}
}

// Register registers a command handler for a specific command type
func (b *CommandBus) Register(commandType interface{}, handler CommandHandler) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	commandName := getTypeName(commandType)
	if _, exists := b.handlers[commandName]; exists {
		return fmt.Errorf("handler already registered for command type: %s", commandName)
	}

	b.handlers[commandName] = handler
	b.logger.Debug("Registered command handler", zap.String("commandType", commandName))
	return nil
}

// Dispatch dispatches a command to its registered handler
func (b *CommandBus) Dispatch(ctx context.Context, command interface{}) (interface{}, error) {
	b.mu.RLock()
	commandName := getTypeName(command)
	handler, exists := b.handlers[commandName]
	b.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("no handler registered for command type: %s", commandName)
	}

	b.logger.Debug("Dispatching command", zap.String("commandType", commandName))
	
	result, err := handler(ctx, command)
	if err != nil {
		// Handle different error types
		switch e := err.(type) {
		case *ValidationError:
			b.logger.Debug("Command validation error",
				zap.String("commandType", commandName),
				zap.String("error", e.Error()),
			)
		case error:
			b.logger.Debug("Command handler error",
				zap.String("commandType", commandName),
				zap.Error(e),
			)
		}
		return nil, fmt.Errorf("command execution failed: %w", err)
	}

	return result, nil
}

// ValidationError represents a validation error
type ValidationError struct {
	Message string
	Field   string
}

// Error implements the error interface
func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error on field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

// MiddlewareFunc is a function that wraps a command handler
type MiddlewareFunc func(next CommandHandler) CommandHandler

// Use adds middleware to the command bus
func (b *CommandBus) Use(middleware MiddlewareFunc) {
	b.mu.Lock()
	defer b.mu.Unlock()

	for commandType, handler := range b.handlers {
		b.handlers[commandType] = middleware(handler)
	}
}

// LoggingMiddleware creates a middleware that logs command execution
func LoggingMiddleware(logger *zap.Logger) MiddlewareFunc {
	return func(next CommandHandler) CommandHandler {
		return func(ctx context.Context, command interface{}) (interface{}, error) {
			commandName := getTypeName(command)
			
			logger.Debug("Executing command",
				zap.String("commandType", commandName),
			)
			
			result, err := next(ctx, command)
			
			if err != nil {
				logger.Debug("Command execution failed",
					zap.String("commandType", commandName),
					zap.Error(err),
				)
				return nil, err
			}
			
			logger.Debug("Command executed successfully",
				zap.String("commandType", commandName),
			)
			
			return result, nil
		}
	}
}

// getTypeName returns the name of the type
func getTypeName(value interface{}) string {
	t := reflect.TypeOf(value)
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t.String()
}

// ErrorHandlingMiddleware creates a middleware that handles errors
func ErrorHandlingMiddleware() MiddlewareFunc {
	return func(next CommandHandler) CommandHandler {
		return func(ctx context.Context, command interface{}) (result interface{}, err error) {
			defer func() {
				if r := recover(); r != nil {
					switch v := r.(type) {
					case string:
						err = errors.New(v)
					case error:
						err = v
					default:
						err = fmt.Errorf("unknown panic: %v", r)
					}
				}
			}()
			
			return next(ctx, command)
		}
	}
}

