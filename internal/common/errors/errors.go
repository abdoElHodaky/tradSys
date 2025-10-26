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
	
	// Market data errors
	ErrSymbolNotFound   ErrorCode = "SYMBOL_NOT_FOUND"
	ErrMarketClosed     ErrorCode = "MARKET_CLOSED"
	ErrInvalidPrice     ErrorCode = "INVALID_PRICE"
	
	// System errors
	ErrDatabaseConnection ErrorCode = "DATABASE_CONNECTION"
	ErrServiceUnavailable ErrorCode = "SERVICE_UNAVAILABLE"
	ErrTimeout           ErrorCode = "TIMEOUT"
	ErrRateLimited       ErrorCode = "RATE_LIMITED"
	
	// Authentication errors
	ErrUnauthorized      ErrorCode = "UNAUTHORIZED"
	ErrInvalidToken      ErrorCode = "INVALID_TOKEN"
	ErrTokenExpired      ErrorCode = "TOKEN_EXPIRED"
	
	// Risk management errors
	ErrRiskLimitExceeded ErrorCode = "RISK_LIMIT_EXCEEDED"
	ErrPositionLimitExceeded ErrorCode = "POSITION_LIMIT_EXCEEDED"
	
	// Matching engine errors
	ErrMatchingFailed    ErrorCode = "MATCHING_FAILED"
	ErrEngineOverloaded  ErrorCode = "ENGINE_OVERLOADED"
)

// TradSysError represents a structured error in the trading system
type TradSysError struct {
	Code      ErrorCode              `json:"code"`
	Message   string                 `json:"message"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
	Cause     error                  `json:"-"`
}

// Error implements the error interface
func (e *TradSysError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %s (caused by: %v)", e.Code, e.Message, e.Cause)
	}
	return fmt.Sprintf("%s: %s", e.Code, e.Message)
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

// New creates a new TradSysError
func New(code ErrorCode, message string) *TradSysError {
	_, file, line, _ := runtime.Caller(1)
	return &TradSysError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
	}
}

// Newf creates a new TradSysError with formatted message
func Newf(code ErrorCode, format string, args ...interface{}) *TradSysError {
	return New(code, fmt.Sprintf(format, args...))
}

// Wrap wraps an existing error with a TradSysError
func Wrap(err error, code ErrorCode, message string) *TradSysError {
	if err == nil {
		return nil
	}
	
	_, file, line, _ := runtime.Caller(1)
	return &TradSysError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
		File:      file,
		Line:      line,
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

// IsRetryable determines if an error is retryable
func IsRetryable(err error) bool {
	code := GetErrorCode(err)
	switch code {
	case ErrTimeout, ErrServiceUnavailable, ErrDatabaseConnection, ErrEngineOverloaded:
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
		 ErrInvalidToken, ErrTokenExpired, ErrSymbolNotFound:
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
		 ErrMatchingFailed, ErrEngineOverloaded:
		return true
	default:
		return false
	}
}
