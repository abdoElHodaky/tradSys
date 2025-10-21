package core

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sync"
)

// Command represents a command in the CQRS pattern
type Command interface {
	// CommandName returns the name of the command
	CommandName() string
}

// Handler represents a command handler
type Handler interface {
	// Handle handles a command
	Handle(ctx context.Context, command Command) error
}

// HandlerFunc is a function that implements the Handler interface
type HandlerFunc func(ctx context.Context, command Command) error

// Handle handles a command
func (f HandlerFunc) Handle(ctx context.Context, command Command) error {
	return f(ctx, command)
}

// Bus represents a command bus
type Bus interface {
	// Register registers a handler for a command
	Register(commandType reflect.Type, handler Handler) error
	
	// RegisterFunc registers a handler function for a command
	RegisterFunc(commandType reflect.Type, handler func(ctx context.Context, command Command) error) error
	
	// Dispatch dispatches a command to its handler
	Dispatch(ctx context.Context, command Command) error
}

// DefaultBus provides a default implementation of the Bus interface
type DefaultBus struct {
	handlers map[string]Handler
	mu       sync.RWMutex
}

// NewDefaultBus creates a new default command bus
func NewDefaultBus() *DefaultBus {
	return &DefaultBus{
		handlers: make(map[string]Handler),
	}
}

// Register registers a handler for a command
func (b *DefaultBus) Register(commandType reflect.Type, handler Handler) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	
	// Create a zero value of the command type
	command, ok := reflect.New(commandType).Elem().Interface().(Command)
	if !ok {
		return fmt.Errorf("command type %s does not implement Command interface", commandType.Name())
	}
	
	// Get the command name
	commandName := command.CommandName()
	
	// Check if a handler is already registered for the command
	if _, exists := b.handlers[commandName]; exists {
		return fmt.Errorf("handler already registered for command %s", commandName)
	}
	
	// Register the handler
	b.handlers[commandName] = handler
	
	return nil
}

// RegisterFunc registers a handler function for a command
func (b *DefaultBus) RegisterFunc(commandType reflect.Type, handler func(ctx context.Context, command Command) error) error {
	return b.Register(commandType, HandlerFunc(handler))
}

// Dispatch dispatches a command to its handler
func (b *DefaultBus) Dispatch(ctx context.Context, command Command) error {
	b.mu.RLock()
	defer b.mu.RUnlock()
	
	// Get the command name
	commandName := command.CommandName()
	
	// Get the handler for the command
	handler, exists := b.handlers[commandName]
	if !exists {
		return fmt.Errorf("no handler registered for command %s", commandName)
	}
	
	// Handle the command
	return handler.Handle(ctx, command)
}

// Common errors
var (
	ErrCommandNotFound = errors.New("command not found")
	ErrHandlerNotFound = errors.New("handler not found")
)
