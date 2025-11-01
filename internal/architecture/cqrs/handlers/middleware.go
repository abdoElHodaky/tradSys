package handlers

import (
	"context"
	"time"

	"go.uber.org/zap"
)

// CommandMiddleware represents middleware for command handling
type CommandMiddleware interface {
	// Execute executes the middleware
	Execute(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error
}

// CommandMiddlewareFunc is a function that implements the CommandMiddleware interface
type CommandMiddlewareFunc func(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error

// Execute executes the middleware
func (f CommandMiddlewareFunc) Execute(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error {
	return f(ctx, cmd, next)
}

// QueryMiddleware represents middleware for query handling
type QueryMiddleware interface {
	// Execute executes the middleware
	Execute(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error)
}

// QueryMiddlewareFunc is a function that implements the QueryMiddleware interface
type QueryMiddlewareFunc func(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error)

// Execute executes the middleware
func (f QueryMiddlewareFunc) Execute(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error) {
	return f(ctx, q, next)
}

// LoggingCommandMiddleware provides logging for command handling
type LoggingCommandMiddleware struct {
	logger *zap.Logger
}

// NewLoggingCommandMiddleware creates a new logging command middleware
func NewLoggingCommandMiddleware(logger *zap.Logger) *LoggingCommandMiddleware {
	return &LoggingCommandMiddleware{
		logger: logger,
	}
}

// Execute executes the middleware
func (m *LoggingCommandMiddleware) Execute(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error {
	start := time.Now()

	// Log the command
	m.logger.Info("Executing command",
		zap.String("command", cmd.CommandName()),
		zap.Any("payload", cmd))

	// Execute the next middleware
	err := next(ctx, cmd)

	// Log the result
	if err != nil {
		m.logger.Error("Command execution failed",
			zap.String("command", cmd.CommandName()),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err))
	} else {
		m.logger.Info("Command execution succeeded",
			zap.String("command", cmd.CommandName()),
			zap.Duration("duration", time.Since(start)))
	}

	return err
}

// LoggingQueryMiddleware provides logging for query handling
type LoggingQueryMiddleware struct {
	logger *zap.Logger
}

// NewLoggingQueryMiddleware creates a new logging query middleware
func NewLoggingQueryMiddleware(logger *zap.Logger) *LoggingQueryMiddleware {
	return &LoggingQueryMiddleware{
		logger: logger,
	}
}

// Execute executes the middleware
func (m *LoggingQueryMiddleware) Execute(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error) {
	start := time.Now()

	// Log the query
	m.logger.Info("Executing query",
		zap.String("query", q.QueryName()),
		zap.Any("payload", q))

	// Execute the next middleware
	result, err := next(ctx, q)

	// Log the result
	if err != nil {
		m.logger.Error("Query execution failed",
			zap.String("query", q.QueryName()),
			zap.Duration("duration", time.Since(start)),
			zap.Error(err))
	} else {
		m.logger.Info("Query execution succeeded",
			zap.String("query", q.QueryName()),
			zap.Duration("duration", time.Since(start)))
	}

	return result, err
}

// MetricsCommandMiddleware provides metrics for command handling
type MetricsCommandMiddleware struct {
	// Metrics collector would be added here
}

// NewMetricsCommandMiddleware creates a new metrics command middleware
func NewMetricsCommandMiddleware() *MetricsCommandMiddleware {
	return &MetricsCommandMiddleware{}
}

// Execute executes the middleware
func (m *MetricsCommandMiddleware) Execute(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error {
	start := time.Now()

	// Execute the next middleware
	err := next(ctx, cmd)

	// Record metrics
	duration := time.Since(start)

	// Record command execution time
	// m.metrics.RecordCommandExecution(cmd.CommandName(), duration, err == nil)

	return err
}

// MetricsQueryMiddleware provides metrics for query handling
type MetricsQueryMiddleware struct {
	// Metrics collector would be added here
}

// NewMetricsQueryMiddleware creates a new metrics query middleware
func NewMetricsQueryMiddleware() *MetricsQueryMiddleware {
	return &MetricsQueryMiddleware{}
}

// Execute executes the middleware
func (m *MetricsQueryMiddleware) Execute(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error) {
	start := time.Now()

	// Execute the next middleware
	result, err := next(ctx, q)

	// Record metrics
	duration := time.Since(start)

	// Record query execution time
	// m.metrics.RecordQueryExecution(q.QueryName(), duration, err == nil)

	return result, err
}

// ValidationCommandMiddleware provides validation for command handling
type ValidationCommandMiddleware struct {
	validators map[string]func(cmd command.Command) error
}

// NewValidationCommandMiddleware creates a new validation command middleware
func NewValidationCommandMiddleware() *ValidationCommandMiddleware {
	return &ValidationCommandMiddleware{
		validators: make(map[string]func(cmd command.Command) error),
	}
}

// RegisterValidator registers a validator for a command
func (m *ValidationCommandMiddleware) RegisterValidator(commandName string, validator func(cmd command.Command) error) {
	m.validators[commandName] = validator
}

// Execute executes the middleware
func (m *ValidationCommandMiddleware) Execute(ctx context.Context, cmd command.Command, next func(ctx context.Context, cmd command.Command) error) error {
	// Get the validator for the command
	validator, exists := m.validators[cmd.CommandName()]
	if exists {
		// Validate the command
		err := validator(cmd)
		if err != nil {
			return err
		}
	}

	// Execute the next middleware
	return next(ctx, cmd)
}

// ValidationQueryMiddleware provides validation for query handling
type ValidationQueryMiddleware struct {
	validators map[string]func(q query.Query) error
}

// NewValidationQueryMiddleware creates a new validation query middleware
func NewValidationQueryMiddleware() *ValidationQueryMiddleware {
	return &ValidationQueryMiddleware{
		validators: make(map[string]func(q query.Query) error),
	}
}

// RegisterValidator registers a validator for a query
func (m *ValidationQueryMiddleware) RegisterValidator(queryName string, validator func(q query.Query) error) {
	m.validators[queryName] = validator
}

// Execute executes the middleware
func (m *ValidationQueryMiddleware) Execute(ctx context.Context, q query.Query, next func(ctx context.Context, q query.Query) (interface{}, error)) (interface{}, error) {
	// Get the validator for the query
	validator, exists := m.validators[q.QueryName()]
	if exists {
		// Validate the query
		err := validator(q)
		if err != nil {
			return nil, err
		}
	}

	// Execute the next middleware
	return next(ctx, q)
}
