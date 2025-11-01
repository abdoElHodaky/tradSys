// Package durability provides robust error handling, resilience patterns, and monitoring
package durability

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// ErrorSeverity defines the severity level of errors
type ErrorSeverity int

const (
	SeverityLow ErrorSeverity = iota
	SeverityMedium
	SeverityHigh
	SeverityCritical
)

// TradingError represents a structured error with context
type TradingError struct {
	Code      string
	Message   string
	Severity  ErrorSeverity
	Context   map[string]interface{}
	Timestamp time.Time
	Cause     error
}

func (e *TradingError) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Message, e.Cause)
}

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger *zap.Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger *zap.Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// HandleError processes errors with appropriate logging and actions
func (h *ErrorHandler) HandleError(ctx context.Context, err *TradingError) {
	fields := []zap.Field{
		zap.String("error_code", err.Code),
		zap.String("message", err.Message),
		zap.Int("severity", int(err.Severity)),
		zap.Time("timestamp", err.Timestamp),
		zap.Any("context", err.Context),
	}

	if err.Cause != nil {
		fields = append(fields, zap.Error(err.Cause))
	}

	switch err.Severity {
	case SeverityLow:
		h.logger.Info("Low severity error", fields...)
	case SeverityMedium:
		h.logger.Warn("Medium severity error", fields...)
	case SeverityHigh:
		h.logger.Error("High severity error", fields...)
	case SeverityCritical:
		h.logger.Fatal("Critical error", fields...)
	}
}

// WrapError creates a TradingError from a standard error
func WrapError(code, message string, severity ErrorSeverity, cause error, context map[string]interface{}) *TradingError {
	return &TradingError{
		Code:      code,
		Message:   message,
		Severity:  severity,
		Context:   context,
		Timestamp: time.Now(),
		Cause:     cause,
	}
}

// Common error codes
const (
	ErrCodeOrderValidation = "ORDER_VALIDATION"
	ErrCodeRiskCheck       = "RISK_CHECK"
	ErrCodeMarketData      = "MARKET_DATA"
	ErrCodeExecution       = "EXECUTION"
	ErrCodeCompliance      = "COMPLIANCE"
	ErrCodeWebSocket       = "WEBSOCKET"
	ErrCodeDatabase        = "DATABASE"
	ErrCodeExternalAPI     = "EXTERNAL_API"
)
