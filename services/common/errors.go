// Package common provides unified error handling for all TradSys v3 services
package common

import (
	"fmt"
	"time"
)

// ServiceError represents a standardized service error
type ServiceError struct {
	Code      string                 `json:"code"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service"`
	Timestamp time.Time              `json:"timestamp"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Cause     error                  `json:"-"`
}

// Error implements the error interface
func (e *ServiceError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Service, e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Service, e.Code, e.Message)
}

// Unwrap returns the underlying cause
func (e *ServiceError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *ServiceError) WithDetail(key string, value interface{}) *ServiceError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause adds a cause to the error
func (e *ServiceError) WithCause(cause error) *ServiceError {
	e.Cause = cause
	return e
}

// NewServiceError creates a new service error
func NewServiceError(service, code, message string) *ServiceError {
	return &ServiceError{
		Code:      code,
		Message:   message,
		Service:   service,
		Timestamp: time.Now(),
		Details:   make(map[string]interface{}),
	}
}

// Predefined Error Categories

// Authentication Errors
var (
	ErrAuthenticationFailed = &ServiceError{
		Code:    "AUTH_FAILED",
		Message: "Authentication failed",
	}
	ErrAuthorizationFailed = &ServiceError{
		Code:    "AUTHZ_FAILED",
		Message: "Authorization failed",
	}
	ErrTokenExpired = &ServiceError{
		Code:    "TOKEN_EXPIRED",
		Message: "Token has expired",
	}
	ErrTokenInvalid = &ServiceError{
		Code:    "TOKEN_INVALID",
		Message: "Token is invalid",
	}
)

// Validation Errors
var (
	ErrInvalidInput = &ServiceError{
		Code:    "INVALID_INPUT",
		Message: "Invalid input provided",
	}
	ErrValidationFailed = &ServiceError{
		Code:    "VALIDATION_FAILED",
		Message: "Validation failed",
	}
	ErrMissingRequired = &ServiceError{
		Code:    "MISSING_REQUIRED",
		Message: "Required field is missing",
	}
	ErrInvalidFormat = &ServiceError{
		Code:    "INVALID_FORMAT",
		Message: "Invalid format",
	}
)

// Resource Errors
var (
	ErrResourceNotFound = &ServiceError{
		Code:    "RESOURCE_NOT_FOUND",
		Message: "Resource not found",
	}
	ErrResourceExists = &ServiceError{
		Code:    "RESOURCE_EXISTS",
		Message: "Resource already exists",
	}
	ErrResourceLocked = &ServiceError{
		Code:    "RESOURCE_LOCKED",
		Message: "Resource is locked",
	}
	ErrResourceUnavailable = &ServiceError{
		Code:    "RESOURCE_UNAVAILABLE",
		Message: "Resource is unavailable",
	}
)

// Service Errors
var (
	ErrServiceUnavailable = &ServiceError{
		Code:    "SERVICE_UNAVAILABLE",
		Message: "Service is unavailable",
	}
	ErrServiceTimeout = &ServiceError{
		Code:    "SERVICE_TIMEOUT",
		Message: "Service timeout",
	}
	ErrServiceOverloaded = &ServiceError{
		Code:    "SERVICE_OVERLOADED",
		Message: "Service is overloaded",
	}
	ErrInternalError = &ServiceError{
		Code:    "INTERNAL_ERROR",
		Message: "Internal server error",
	}
)

// Business Logic Errors
var (
	ErrInsufficientFunds = &ServiceError{
		Code:    "INSUFFICIENT_FUNDS",
		Message: "Insufficient funds",
	}
	ErrOrderRejected = &ServiceError{
		Code:    "ORDER_REJECTED",
		Message: "Order was rejected",
	}
	ErrMarketClosed = &ServiceError{
		Code:    "MARKET_CLOSED",
		Message: "Market is closed",
	}
	ErrRiskLimitExceeded = &ServiceError{
		Code:    "RISK_LIMIT_EXCEEDED",
		Message: "Risk limit exceeded",
	}
	ErrComplianceViolation = &ServiceError{
		Code:    "COMPLIANCE_VIOLATION",
		Message: "Compliance violation",
	}
)

// Exchange Errors
var (
	ErrExchangeUnavailable = &ServiceError{
		Code:    "EXCHANGE_UNAVAILABLE",
		Message: "Exchange is unavailable",
	}
	ErrExchangeTimeout = &ServiceError{
		Code:    "EXCHANGE_TIMEOUT",
		Message: "Exchange timeout",
	}
	ErrInvalidSymbol = &ServiceError{
		Code:    "INVALID_SYMBOL",
		Message: "Invalid trading symbol",
	}
	ErrUnsupportedAsset = &ServiceError{
		Code:    "UNSUPPORTED_ASSET",
		Message: "Unsupported asset type",
	}
)

// Configuration Errors
var (
	ErrInvalidConfig = &ServiceError{
		Code:    "INVALID_CONFIG",
		Message: "Invalid configuration",
	}
	ErrConfigNotFound = &ServiceError{
		Code:    "CONFIG_NOT_FOUND",
		Message: "Configuration not found",
	}
	ErrConfigLoadFailed = &ServiceError{
		Code:    "CONFIG_LOAD_FAILED",
		Message: "Failed to load configuration",
	}
)

// Database Errors
var (
	ErrDatabaseUnavailable = &ServiceError{
		Code:    "DATABASE_UNAVAILABLE",
		Message: "Database is unavailable",
	}
	ErrDatabaseTimeout = &ServiceError{
		Code:    "DATABASE_TIMEOUT",
		Message: "Database timeout",
	}
	ErrDuplicateKey = &ServiceError{
		Code:    "DUPLICATE_KEY",
		Message: "Duplicate key violation",
	}
	ErrConstraintViolation = &ServiceError{
		Code:    "CONSTRAINT_VIOLATION",
		Message: "Database constraint violation",
	}
)

// Network Errors
var (
	ErrNetworkUnavailable = &ServiceError{
		Code:    "NETWORK_UNAVAILABLE",
		Message: "Network is unavailable",
	}
	ErrConnectionFailed = &ServiceError{
		Code:    "CONNECTION_FAILED",
		Message: "Connection failed",
	}
	ErrConnectionTimeout = &ServiceError{
		Code:    "CONNECTION_TIMEOUT",
		Message: "Connection timeout",
	}
)

// Rate Limiting Errors
var (
	ErrRateLimitExceeded = &ServiceError{
		Code:    "RATE_LIMIT_EXCEEDED",
		Message: "Rate limit exceeded",
	}
	ErrQuotaExceeded = &ServiceError{
		Code:    "QUOTA_EXCEEDED",
		Message: "Quota exceeded",
	}
)

// ErrorHandler provides centralized error handling
type ErrorHandler struct {
	logger Logger
}

// NewErrorHandler creates a new error handler
func NewErrorHandler(logger Logger) *ErrorHandler {
	return &ErrorHandler{
		logger: logger,
	}
}

// HandleError processes and logs errors
func (eh *ErrorHandler) HandleError(service string, err error) *ServiceError {
	if serviceErr, ok := err.(*ServiceError); ok {
		// Update service name if not set
		if serviceErr.Service == "" {
			serviceErr.Service = service
		}

		eh.logError(serviceErr)
		return serviceErr
	}

	// Convert unknown errors to internal errors
	internalErr := NewServiceError(service, "INTERNAL_ERROR", err.Error()).WithCause(err)
	eh.logError(internalErr)
	return internalErr
}

// logError logs the error with appropriate level
func (eh *ErrorHandler) logError(err *ServiceError) {
	fields := []interface{}{
		"service", err.Service,
		"code", err.Code,
		"message", err.Message,
		"timestamp", err.Timestamp,
	}

	if len(err.Details) > 0 {
		fields = append(fields, "details", err.Details)
	}

	if err.Cause != nil {
		fields = append(fields, "cause", err.Cause.Error())
	}

	// Log based on error severity
	switch err.Code {
	case "INTERNAL_ERROR", "SERVICE_UNAVAILABLE", "DATABASE_UNAVAILABLE":
		eh.logger.Error("Service error occurred", fields...)
	case "VALIDATION_FAILED", "INVALID_INPUT", "RESOURCE_NOT_FOUND":
		eh.logger.Warn("Service warning", fields...)
	default:
		eh.logger.Info("Service info", fields...)
	}
}

// WrapError wraps an existing error with service context
func WrapError(service, code, message string, cause error) *ServiceError {
	return NewServiceError(service, code, message).WithCause(cause)
}

// IsErrorCode checks if an error has a specific code
func IsErrorCode(err error, code string) bool {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Code == code
	}
	return false
}

// IsErrorType checks if an error is of a specific type
func IsErrorType(err error, errorType *ServiceError) bool {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Code == errorType.Code
	}
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) string {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Code
	}
	return "UNKNOWN_ERROR"
}

// GetErrorDetails extracts the error details from an error
func GetErrorDetails(err error) map[string]interface{} {
	if serviceErr, ok := err.(*ServiceError); ok {
		return serviceErr.Details
	}
	return nil
}

// ErrorBuilder provides a fluent interface for building errors
type ErrorBuilder struct {
	err *ServiceError
}

// NewErrorBuilder creates a new error builder
func NewErrorBuilder(service, code, message string) *ErrorBuilder {
	return &ErrorBuilder{
		err: NewServiceError(service, code, message),
	}
}

// WithDetail adds a detail to the error
func (eb *ErrorBuilder) WithDetail(key string, value interface{}) *ErrorBuilder {
	eb.err.WithDetail(key, value)
	return eb
}

// WithCause adds a cause to the error
func (eb *ErrorBuilder) WithCause(cause error) *ErrorBuilder {
	eb.err.WithCause(cause)
	return eb
}

// Build returns the built error
func (eb *ErrorBuilder) Build() *ServiceError {
	return eb.err
}

// Error returns the built error as an error interface
func (eb *ErrorBuilder) Error() error {
	return eb.err
}

// Validation helper functions

// ValidateRequired checks if a required field is present
func ValidateRequired(service, field string, value interface{}) error {
	if value == nil {
		return NewServiceError(service, "MISSING_REQUIRED", fmt.Sprintf("Required field '%s' is missing", field)).
			WithDetail("field", field)
	}

	// Check for empty strings
	if str, ok := value.(string); ok && str == "" {
		return NewServiceError(service, "MISSING_REQUIRED", fmt.Sprintf("Required field '%s' is empty", field)).
			WithDetail("field", field)
	}

	return nil
}

// ValidateFormat checks if a field has the correct format
func ValidateFormat(service, field string, value interface{}, validator func(interface{}) bool) error {
	if !validator(value) {
		return NewServiceError(service, "INVALID_FORMAT", fmt.Sprintf("Field '%s' has invalid format", field)).
			WithDetail("field", field).
			WithDetail("value", value)
	}
	return nil
}

// ValidateRange checks if a numeric value is within range
func ValidateRange(service, field string, value, min, max float64) error {
	if value < min || value > max {
		return NewServiceError(service, "INVALID_RANGE", fmt.Sprintf("Field '%s' must be between %f and %f", field, min, max)).
			WithDetail("field", field).
			WithDetail("value", value).
			WithDetail("min", min).
			WithDetail("max", max)
	}
	return nil
}

// ValidateLength checks if a string has the correct length
func ValidateLength(service, field string, value string, minLen, maxLen int) error {
	length := len(value)
	if length < minLen || length > maxLen {
		return NewServiceError(service, "INVALID_LENGTH", fmt.Sprintf("Field '%s' must be between %d and %d characters", field, minLen, maxLen)).
			WithDetail("field", field).
			WithDetail("value", value).
			WithDetail("length", length).
			WithDetail("min_length", minLen).
			WithDetail("max_length", maxLen)
	}
	return nil
}
