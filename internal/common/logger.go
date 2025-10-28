package common

import (
	"fmt"
	"log"
	"os"
	"time"
)

// Logger interface for unified logging across TradSys
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger
}

// DefaultLogger implements Logger using standard log package
type DefaultLogger struct {
	logger *log.Logger
	fields map[string]interface{}
}

// NewDefaultLogger creates a new default logger
func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "[TradSys] ", log.LstdFlags|log.Lshortfile),
		fields: make(map[string]interface{}),
	}
}

// NewDefaultLoggerWithPrefix creates a new default logger with custom prefix
func NewDefaultLoggerWithPrefix(prefix string) *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(os.Stdout, "[TradSys:"+prefix+"] ", log.LstdFlags|log.Lshortfile),
		fields: make(map[string]interface{}),
	}
}

func (l *DefaultLogger) Debug(msg string, fields ...interface{}) {
	l.logWithLevel("DEBUG", msg, fields...)
}

func (l *DefaultLogger) Info(msg string, fields ...interface{}) {
	l.logWithLevel("INFO", msg, fields...)
}

func (l *DefaultLogger) Warn(msg string, fields ...interface{}) {
	l.logWithLevel("WARN", msg, fields...)
}

func (l *DefaultLogger) Error(msg string, fields ...interface{}) {
	l.logWithLevel("ERROR", msg, fields...)
}

func (l *DefaultLogger) Fatal(msg string, fields ...interface{}) {
	l.logWithLevel("FATAL", msg, fields...)
	os.Exit(1)
}

func (l *DefaultLogger) WithField(key string, value interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	newFields[key] = value

	return &DefaultLogger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *DefaultLogger) WithFields(fields map[string]interface{}) Logger {
	newFields := make(map[string]interface{})
	for k, v := range l.fields {
		newFields[k] = v
	}
	for k, v := range fields {
		newFields[k] = v
	}

	return &DefaultLogger{
		logger: l.logger,
		fields: newFields,
	}
}

func (l *DefaultLogger) logWithLevel(level, msg string, fields ...interface{}) {
	// Build the log message with fields
	logMsg := msg

	// Add persistent fields
	if len(l.fields) > 0 {
		logMsg += " |"
		for k, v := range l.fields {
			logMsg += " " + k + "=" + formatValue(v)
		}
	}

	// Add additional fields
	if len(fields) > 0 {
		if len(l.fields) == 0 {
			logMsg += " |"
		}
		for i := 0; i < len(fields); i += 2 {
			if i+1 < len(fields) {
				logMsg += " " + formatValue(fields[i]) + "=" + formatValue(fields[i+1])
			}
		}
	}

	l.logger.Printf("[%s] %s", level, logMsg)
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case int, int32, int64:
		return fmt.Sprint(val)
	case float32, float64:
		return fmt.Sprint(val)
	case bool:
		return fmt.Sprint(val)
	case time.Time:
		return val.Format(time.RFC3339)
	case time.Duration:
		return val.String()
	default:
		return fmt.Sprint(val)
	}
}

// TradingLogger is a specialized logger for trading operations
type TradingLogger struct {
	Logger
	component string
}

// NewTradingLogger creates a new trading logger for a specific component
func NewTradingLogger(component string) *TradingLogger {
	return &TradingLogger{
		Logger:    NewDefaultLoggerWithPrefix(component),
		component: component,
	}
}

// LogOrder logs order-related events
func (tl *TradingLogger) LogOrder(orderID string, action string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"order_id":  orderID,
		"action":    action,
		"component": tl.component,
	}
	for k, v := range details {
		fields[k] = v
	}
	tl.WithFields(fields).Info("Order event")
}

// LogRisk logs risk-related events
func (tl *TradingLogger) LogRisk(riskType string, level string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"risk_type": riskType,
		"level":     level,
		"component": tl.component,
	}
	for k, v := range details {
		fields[k] = v
	}
	tl.WithFields(fields).Warn("Risk event")
}

// LogPerformance logs performance metrics
func (tl *TradingLogger) LogPerformance(operation string, duration time.Duration, details map[string]interface{}) {
	fields := map[string]interface{}{
		"operation": operation,
		"duration":  duration,
		"component": tl.component,
	}
	for k, v := range details {
		fields[k] = v
	}
	tl.WithFields(fields).Info("Performance metric")
}

// LogError logs errors with context
func (tl *TradingLogger) LogError(err error, operation string, details map[string]interface{}) {
	fields := map[string]interface{}{
		"error":     err.Error(),
		"operation": operation,
		"component": tl.component,
	}
	for k, v := range details {
		fields[k] = v
	}
	tl.WithFields(fields).Error("Operation failed")
}

// Global logger instance
var globalLogger Logger = NewDefaultLogger()

// SetGlobalLogger sets the global logger instance
func SetGlobalLogger(logger Logger) {
	globalLogger = logger
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() Logger {
	return globalLogger
}

// Convenience functions using global logger
func Debug(msg string, fields ...interface{}) {
	globalLogger.Debug(msg, fields...)
}

func Info(msg string, fields ...interface{}) {
	globalLogger.Info(msg, fields...)
}

func Warn(msg string, fields ...interface{}) {
	globalLogger.Warn(msg, fields...)
}

func Error(msg string, fields ...interface{}) {
	globalLogger.Error(msg, fields...)
}

func Fatal(msg string, fields ...interface{}) {
	globalLogger.Fatal(msg, fields...)
}
