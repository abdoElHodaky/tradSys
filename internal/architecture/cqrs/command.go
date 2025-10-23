package cqrs

import (
	"context"
	"errors"
	"reflect"
	"sync"
)

// Command represents a command in the CQRS pattern
type Command interface {
	CommandName() string
}

// CommandHandler handles a specific command type
type CommandHandler interface {
	Handle(ctx context.Context, command Command) error
}

// CommandHandlerFunc is a function that implements CommandHandler
type CommandHandlerFunc func(ctx context.Context, command Command) error

// Handle implements CommandHandler
func (f CommandHandlerFunc) Handle(ctx context.Context, command Command) error {
	return f(ctx, command)
}

// CommandBus dispatches commands to their handlers
type CommandBus struct {
	handlers map[string]CommandHandler
	mu       sync.RWMutex
}

// NewCommandBus creates a new command bus
func NewCommandBus() *CommandBus {
	return &CommandBus{
		handlers: make(map[string]CommandHandler),
	}
}

// RegisterHandler registers a handler for a specific command type
func (cb *CommandBus) RegisterHandler(commandType reflect.Type, handler CommandHandler) error {
	if commandType.Kind() != reflect.Ptr {
		return errors.New("command type must be a pointer type")
	}

	commandName := commandType.Elem().Name()

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if _, exists := cb.handlers[commandName]; exists {
		return errors.New("handler already registered for command: " + commandName)
	}

	cb.handlers[commandName] = handler
	return nil
}

// RegisterHandlerFunc registers a handler function for a specific command type
func (cb *CommandBus) RegisterHandlerFunc(commandType reflect.Type, handler func(ctx context.Context, command Command) error) error {
	return cb.RegisterHandler(commandType, CommandHandlerFunc(handler))
}

// Dispatch dispatches a command to its handler
func (cb *CommandBus) Dispatch(ctx context.Context, command Command) error {
	if command == nil {
		return errors.New("command cannot be nil")
	}

	commandName := command.CommandName()

	cb.mu.RLock()
	handler, exists := cb.handlers[commandName]
	cb.mu.RUnlock()

	if !exists {
		return errors.New("no handler registered for command: " + commandName)
	}

	return handler.Handle(ctx, command)
}

// HasHandler checks if a handler is registered for a specific command type
func (cb *CommandBus) HasHandler(commandType reflect.Type) bool {
	if commandType.Kind() != reflect.Ptr {
		return false
	}

	commandName := commandType.Elem().Name()

	cb.mu.RLock()
	defer cb.mu.RUnlock()

	_, exists := cb.handlers[commandName]
	return exists
}
