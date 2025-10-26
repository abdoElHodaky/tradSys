package errors

import (
	"fmt"
	"runtime"
	"time"
)

// ErrorCode represents different types of errors in the system
type ErrorCode string

const (
	// Order related errors
	ErrInvalidOrder     ErrorCode = "INVALID_ORDER"
	ErrOrderNotFound    ErrorCode = "ORDER_NOT_FOUND"
	ErrOrderCanceled    ErrorCode = "ORDER_CANCELED"
	ErrInsufficientFunds ErrorCode = "INSUFFICIENT_FUNDS"
	ErrOrderExpired     ErrorCode = "ORDER_EXPIRED"
	ErrDuplicateOrder   ErrorCode = "DUPLICATE_ORDER"
	
	// Market data errors
	ErrSymbolNotFound   ErrorCode = "SYMBOL_NOT_FOUND"
	ErrMarketClosed     ErrorCode = "MARKET_CLOSED"
	ErrInvalidPrice     ErrorCode = "INVALID_PRICE"
	ErrInvalidQuantity  ErrorCode = "INVALID_QUANTITY"
	ErrPriceOutOfRange  ErrorCode = "PRICE_OUT_OF_RANGE"
	
	// System errors
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrRateLimited       ErrorCode = "RATE_LIMITED"
	ErrInternalError     ErrorCode = "INTERNAL_ERROR"
	
	// Authentication errors
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrInvalidToken      ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired      ErrorCode = "TOKEN_EXPIRED"
	ErrPermissionDenied  ErrorCode = "PERMISSION_DENIED"
	
	// Risk management errors
	ErrRiskLimitExceeded ErrorCode = "RISK_LIMIT_EXCEEDED"
	ErrPositionLimitExceeded ErrorCode = "POSITION_LIMIT_EXCEEDED"
	ErrDailyLimitExceeded ErrorCode = "DAILY_LIMIT_EXCEEDED"
	ErrLeverageExceeded  ErrorCode = "LEVERAGE_EXCEEDED"
	
	// Matching engine errors
	ErrMatchingFailed    ErrorCode = "MATCHING_FAILED"
	ErrEngineOverloaded  ErrorCode = "ENGINE_OVERLOADED"
	ErrOrderBookFull     ErrorCode = "ORDER_BOOK_FULL"
	ErrCrossedMarket     ErrorCode = "CROSSED_MARKET"
	
	// Validation errors
	ErrValidationFailed  ErrorCode = "VALIDATION_FAILED"
	ErrInvalidInput      ErrorCode = "INVALID_INPUT"
	ErrMissingField      ErrorCode = "MISSING_FIELD"
	ErrInvalidFormat     ErrorCode = "INVALID_FORMAT"
	
	// Configuration errors
	ErrConfigurationError ErrorCode = "CONFIGURATION_ERROR"
	ErrMissingConfiguration ErrorCode = "MISSING_CONFIGURATION"
	ErrInvalidConfiguration ErrorCode = "INVALID_CONFIGURATION"
)

// ErrorSeverity represents the severity level of an error
type ErrorSeverity string

const (
	SeverityLow      ErrorSeverity = "low"
	SeverityMedium   ErrorSeverity = "medium"
	SeverityHigh     ErrorSeverity = "high"
	SeverityCritical ErrorSeverity = "critical"
)

// TradSysError represents a structured error in the trading system
type TradSysError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Severity  ErrorSeverity          `json:"severity"`
	Timestamp time.Time              `json:"timestamp"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Function  string                 `json:"function,omitempty"`
	Cause     error                  `json:"-"`
	UserID    string                 `json:"user_id,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
}

// Error implements the error interface
func (e *TradSysError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("[%s] %s: %s (caused by: %v)", e.Code, e.Severity, e.Message, e.Cause)
	}
	return fmt.Sprintf("[%s] %s: %s", e.Code, e.Severity, e.Message)
}

// Unwrap returns the underlying cause
func (e *TradSysError) Unwrap() error {
	return e.Cause
}

// WithDetail adds a detail to the error
func (e *TradSysError) WithDetail(key string, value interface{}) *TradSysError {
	if e.Details == nil {
		e.Details = make(map[string]interface{})
	}
	e.Details[key] = value
	return e
}

// WithCause adds a cause to the error
func (e *TradSysError) WithCause(cause error) *TradSysError {
	e.Cause = cause
	return e
}

// WithUserID adds a user ID to the error for tracking
func (e *TradSysError) WithUserID(userID string) *TradSysError {
	e.UserID = userID
	return e
}

// WithRequestID adds a request ID to the error for tracking
func (e *TradSysError) WithRequestID(requestID string) *TradSysError {
	e.RequestID = requestID
	return e
}

// WithTraceID adds a trace ID to the error for distributed tracing
func (e *TradSysError) WithTraceID(traceID string) *TradSysError {
	e.TraceID = traceID
	return e
}

// New creates a new TradSysError
func New(code ErrorCode, message string) *TradSysError {
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
	}
	
	return &TradSysError{
		Code:      code,
		Message:   message,
		Severity:  getSeverityForCode(code),
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Function:  funcName,
	}
}

// Newf creates a new TradSysError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *TradSysError {
	return New(code, fmt.Sprintf(format, args...))
}

// NewWithSeverity creates a new TradSysError with custom severity
func NewWithSeverity(code ErrorCode, message string, severity ErrorSeverity) *TradSysError {
	err := New(code, message)
	err.Severity = severity
	return err
}

// Wrap wraps an existing error with a TradSysError
func Wrap(err error, code ErrorCode, message string) *TradSysError {
	if err == nil {
		return nil
	}
	
	pc, file, line, _ := runtime.Caller(1)
	fn := runtime.FuncForPC(pc)
	var funcName string
	if fn != nil {
		funcName = fn.Name()
	}
	
	return &TradSysError{
		Code:      code,
		Message:   message,
		Severity:  getSeverityForCode(code),
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
		Function:  funcName,
		Cause:     err,
	}
}

// Wrapf wraps an existing error with a formatted TradSysError
func Wrapf(err error, code ErrorCode, format string, args ...interface{}) *TradSysError {
	return Wrap(err, code, fmt.Sprintf(format, args...))
}

// Is checks if an error is of a specific error code
func Is(err error, code ErrorCode) bool {
	var tradSysErr *TradSysError
	if As(err, &tradSysErr) {
		return tradSysErr.Code == code
	}
	return false
}

// As finds the first error in err's chain that matches target
func As(err error, target interface{}) bool {
	if err == nil {
		return false
	}
	
	if tradSysErr, ok := err.(*TradSysError); ok {
		if targetPtr, ok := target.(**TradSysError); ok {
			*targetPtr = tradSysErr
			return true
		}
	}
	
	// Check if the error implements Unwrap
	if unwrapper, ok := err.(interface{ Unwrap() error }); ok {
		return As(unwrapper.Unwrap(), target)
	}
	
	return false
}

// GetErrorCode extracts the error code from an error
func GetErrorCode(err error) ErrorCode {
	var tradSysErr *TradSysError
	if As(err, &tradSysErr) {
		return tradSysErr.Code
	}
	return ""
}

// GetErrorDetails extracts the error details from an error
func GetErrorDetails(err error) map[string]interface{} {
	var tradSysErr *TradSysError
	if As(err, &tradSysErr) {
		return tradSysErr.Details
	}
	return nil
}

// GetErrorSeverity extracts the error severity from an error
func GetErrorSeverity(err error) ErrorSeverity {
	var tradSysErr *TradSysError
	if As(err, &tradSysErr) {
		return tradSysErr.Severity
	}
	return SeverityLow
}

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	code := GetErrorCode(err)
	switch code {
	case ErrTimeout, ErrServiceUnavailable, ErrDatabaseConnection, 
		 ErrEngineOverloaded, ErrRateLimited, ErrInternalError:
		return true
	default:
		return false
	}
}

// IsClientError determines if an error is a client error (4xx)
func IsClientError(err error) bool {
	code := GetErrorCode(err)
	switch code {
	case ErrInvalidOrder, ErrOrderNotFound, ErrInvalidPrice, ErrUnauthorized, 
		 ErrInvalidToken, ErrTokenExpired, ErrSymbolNotFound, ErrValidationFailed,
		 ErrInvalidInput, ErrMissingField, ErrInvalidFormat, ErrPermissionDenied,
		 ErrInvalidQuantity, ErrDuplicateOrder:
		return true
	default:
		return false
	}
}

// IsServerError determines if an error is a server error (5xx)
func IsServerError(err error) bool {
	code := GetErrorCode(err)
	switch code {
	case ErrDatabaseConnection, ErrServiceUnavailable, ErrTimeout, 
		 ErrMatchingFailed, ErrEngineOverloaded, ErrInternalError,
		 ErrConfigurationError:
		return true
	default:
		return false
	}
}

// IsCritical determines if an error is critical and requires immediate attention
func IsCritical(err error) bool {
	severity := GetErrorSeverity(err)
	return severity == SeverityCritical
}

// getSeverityForCode returns the default severity for an error code
func getSeverityForCode(code ErrorCode) ErrorSeverity {
	switch code {
	case ErrDatabaseConnection, ErrServiceUnavailable, ErrEngineOverloaded,
		 ErrInternalError, ErrConfigurationError:
		return SeverityCritical
	case ErrMatchingFailed, ErrOrderBookFull, ErrRiskLimitExceeded,
		 ErrPositionLimitExceeded, ErrTimeout:
		return SeverityHigh
	case ErrInvalidOrder, ErrOrderNotFound, ErrUnauthorized, ErrInvalidToken,
		 ErrSymbolNotFound, ErrMarketClosed, ErrRateLimited:
		return SeverityMedium
	default:
		return SeverityLow
	}
}

// ErrorGroup represents a collection of errors
type ErrorGroup struct {
	errors []error
}

// NewErrorGroup creates a new error group
func NewErrorGroup() *ErrorGroup {
	return &ErrorGroup{
		errors: make([]error, 0),
	}
}

// Add adds an error to the group
func (eg *ErrorGroup) Add(err error) {
	if err != nil {
		eg.errors = append(eg.errors, err)
	}
}

// HasErrors returns true if the group has any errors
func (eg *ErrorGroup) HasErrors() bool {
	return len(eg.errors) > 0
}

// Errors returns all errors in the group
func (eg *ErrorGroup) Errors() []error {
	return eg.errors
}

// Error implements the error interface
func (eg *ErrorGroup) Error() string {
	if len(eg.errors) == 0 {
		return ""
	}
	if len(eg.errors) == 1 {
		return eg.errors[0].Error()
	}
	return fmt.Sprintf("multiple errors occurred: %d errors", len(eg.errors))
}

// First returns the first error in the group
func (eg *ErrorGroup) First() error {
	if len(eg.errors) == 0 {
		return nil
	}
	return eg.errors[0]
}

// ErrorHandler defines the interface for error handling
type ErrorHandler interface {
	HandleError(err error) error
	ShouldRetry(err error) bool
	GetRetryDelay(err error, attempt int) time.Duration
}

// DefaultErrorHandler provides default error handling behavior
type DefaultErrorHandler struct {
	MaxRetries   int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	BackoffFactor float64
}

// NewDefaultErrorHandler creates a new default error handler
func NewDefaultErrorHandler() *DefaultErrorHandler {
	return &DefaultErrorHandler{
		MaxRetries:   3,
		BaseDelay:    100 * time.Millisecond,
		MaxDelay:     5 * time.Second,
		BackoffFactor: 2.0,
	}
}

// HandleError handles an error according to the default policy
func (h *DefaultErrorHandler) HandleError(err error) error {
	// Log the error, send metrics, etc.
	return err
}

// ShouldRetry determines if an error should be retried
func (h *DefaultErrorHandler) ShouldRetry(err error) bool {
	return IsRetryable(err)
}

// GetRetryDelay calculates the delay before retrying
func (h *DefaultErrorHandler) GetRetryDelay(err error, attempt int) time.Duration {
	if attempt <= 0 {
		return h.BaseDelay
	}
	
	delay := time.Duration(float64(h.BaseDelay) * float64(attempt) * h.BackoffFactor)
	if delay > h.MaxDelay {
		delay = h.MaxDelay
	}
	
	return delay
}
