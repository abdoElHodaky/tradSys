package command

import (
	"context"
	"fmt"
	"reflect"
	"sync"

	"go.uber.org/zap"
)

// Command represents a command in the CQRS pattern
type Command interface {
	CommandName() string
}

// Handler represents a command handler in the CQRS pattern
type Handler interface {
	Handle(ctx context.Context, command Command) error
}

// HandlerFunc is a function that implements the Handler interface
type HandlerFunc func(ctx context.Context, command Command) error

// Handle handles a command
func (f HandlerFunc) Handle(ctx context.Context, command Command) error {
	return f(ctx, command)
}

// CommandBus represents a command bus in the CQRS pattern
type CommandBus struct {
	handlers map[string]Handler
	logger   *zap.Logger
	mu       sync.RWMutex
}

// NewCommandBus creates a new command bus
func NewCommandBus() *CommandBus {
	return &CommandBus{
		handlers: make(map[string]Handler),
	}
}

// SetLogger sets the logger for the command bus
func (b *CommandBus) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// Register registers a handler for a command
func (b *CommandBus) Register(commandType reflect.Type, handler Handler) error {
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

	if b.logger != nil {
		b.logger.Info("Registered command handler",
			zap.String("command", commandName),
			zap.String("handler", reflect.TypeOf(handler).String()))
	}

	return nil
}

// RegisterFunc registers a handler function for a command
func (b *CommandBus) RegisterFunc(commandType reflect.Type, handler func(ctx context.Context, command Command) error) error {
	return b.Register(commandType, HandlerFunc(handler))
}

// Dispatch dispatches a command to its handler
func (b *CommandBus) Dispatch(ctx context.Context, command Command) error {
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
	if b.logger != nil {
		b.logger.Debug("Dispatching command",
			zap.String("command", commandName),
			zap.String("handler", reflect.TypeOf(handler).String()))
	}

	return handler.Handle(ctx, command)
}

// CreateOrderCommand represents a command to create an order
type CreateOrderCommand struct {
	UserID       string
	AccountID    string
	Symbol       string
	Side         string
	Type         string
	Quantity     float64
	Price        float64
	StopPrice    float64
	TimeInForce  string
	ClientOrderID string
}

// CommandName returns the name of the command
func (c *CreateOrderCommand) CommandName() string {
	return "CreateOrder"
}

// CancelOrderCommand represents a command to cancel an order
type CancelOrderCommand struct {
	OrderID string
	UserID  string
}

// CommandName returns the name of the command
func (c *CancelOrderCommand) CommandName() string {
	return "CancelOrder"
}

// UpdateOrderCommand represents a command to update an order
type UpdateOrderCommand struct {
	OrderID     string
	UserID      string
	Quantity    float64
	Price       float64
	StopPrice   float64
	TimeInForce string
}

// CommandName returns the name of the command
func (c *UpdateOrderCommand) CommandName() string {
	return "UpdateOrder"
}

