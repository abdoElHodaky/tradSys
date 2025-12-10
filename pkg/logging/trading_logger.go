// Package logging provides standardized structured logging for TradSys
package logging

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	tradingContext "github.com/abdoElHodaky/tradSys/pkg/context"
)

// LogLevel represents the logging level
type LogLevel string

const (
	DebugLevel LogLevel = "debug"
	InfoLevel  LogLevel = "info"
	WarnLevel  LogLevel = "warn"
	ErrorLevel LogLevel = "error"
	PanicLevel LogLevel = "panic"
	FatalLevel LogLevel = "fatal"
)

// TradingLogger provides structured logging for trading operations
type TradingLogger struct {
	logger *zap.Logger
	sugar  *zap.SugaredLogger
}

// Config represents logger configuration
type Config struct {
	Level            LogLevel `json:"level" yaml:"level"`
	Development      bool     `json:"development" yaml:"development"`
	DisableCaller    bool     `json:"disable_caller" yaml:"disable_caller"`
	DisableStacktrace bool    `json:"disable_stacktrace" yaml:"disable_stacktrace"`
	OutputPaths      []string `json:"output_paths" yaml:"output_paths"`
	ErrorOutputPaths []string `json:"error_output_paths" yaml:"error_output_paths"`
}

// DefaultConfig returns the default logger configuration
func DefaultConfig() *Config {
	return &Config{
		Level:            InfoLevel,
		Development:      false,
		DisableCaller:    false,
		DisableStacktrace: false,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
}

// NewTradingLogger creates a new trading logger with the given configuration
func NewTradingLogger(config *Config) (*TradingLogger, error) {
	if config == nil {
		config = DefaultConfig()
	}

	zapConfig := zap.Config{
		Level:             zap.NewAtomicLevelAt(parseLogLevel(config.Level)),
		Development:       config.Development,
		DisableCaller:     config.DisableCaller,
		DisableStacktrace: config.DisableStacktrace,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: "json",
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:        "timestamp",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "message",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.LowercaseLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		},
		OutputPaths:      config.OutputPaths,
		ErrorOutputPaths: config.ErrorOutputPaths,
	}

	logger, err := zapConfig.Build()
	if err != nil {
		return nil, fmt.Errorf("failed to build logger: %w", err)
	}

	return &TradingLogger{
		logger: logger,
		sugar:  logger.Sugar(),
	}, nil
}

// NewDevelopmentLogger creates a logger optimized for development
func NewDevelopmentLogger() (*TradingLogger, error) {
	config := &Config{
		Level:            DebugLevel,
		Development:      true,
		DisableCaller:    false,
		DisableStacktrace: false,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return NewTradingLogger(config)
}

// NewProductionLogger creates a logger optimized for production
func NewProductionLogger() (*TradingLogger, error) {
	config := &Config{
		Level:            InfoLevel,
		Development:      false,
		DisableCaller:    true,
		DisableStacktrace: true,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}
	return NewTradingLogger(config)
}

// parseLogLevel converts LogLevel to zapcore.Level
func parseLogLevel(level LogLevel) zapcore.Level {
	switch level {
	case DebugLevel:
		return zapcore.DebugLevel
	case InfoLevel:
		return zapcore.InfoLevel
	case WarnLevel:
		return zapcore.WarnLevel
	case ErrorLevel:
		return zapcore.ErrorLevel
	case PanicLevel:
		return zapcore.PanicLevel
	case FatalLevel:
		return zapcore.FatalLevel
	default:
		return zapcore.InfoLevel
	}
}

// WithContext adds context information to the logger
func (tl *TradingLogger) WithContext(ctx context.Context) *TradingLogger {
	fields := []zap.Field{}

	if userID, ok := tradingContext.GetUserID(ctx); ok {
		fields = append(fields, zap.String("user_id", userID))
	}
	if orderID, ok := tradingContext.GetOrderID(ctx); ok {
		fields = append(fields, zap.String("order_id", orderID))
	}
	if sessionID, ok := tradingContext.GetSessionID(ctx); ok {
		fields = append(fields, zap.String("session_id", sessionID))
	}
	if tradeID, ok := tradingContext.GetTradeID(ctx); ok {
		fields = append(fields, zap.String("trade_id", tradeID))
	}
	if symbol, ok := tradingContext.GetSymbol(ctx); ok {
		fields = append(fields, zap.String("symbol", symbol))
	}
	if exchange, ok := tradingContext.GetExchange(ctx); ok {
		fields = append(fields, zap.String("exchange", exchange))
	}
	if requestID, ok := tradingContext.GetRequestID(ctx); ok {
		fields = append(fields, zap.String("request_id", requestID))
	}
	if correlationID, ok := tradingContext.GetCorrelationID(ctx); ok {
		fields = append(fields, zap.String("correlation_id", correlationID))
	}
	if traceID, ok := tradingContext.GetTraceID(ctx); ok {
		fields = append(fields, zap.String("trace_id", traceID))
	}
	if spanID, ok := tradingContext.GetSpanID(ctx); ok {
		fields = append(fields, zap.String("span_id", spanID))
	}

	// Convert zap.Field to interface{} for sugar logger
	sugarFields := make([]interface{}, 0, len(fields)*2)
	for _, field := range fields {
		sugarFields = append(sugarFields, field.Key, field.Interface)
	}

	return &TradingLogger{
		logger: tl.logger.With(fields...),
		sugar:  tl.sugar.With(sugarFields...),
	}
}

// WithFields adds structured fields to the logger
func (tl *TradingLogger) WithFields(fields map[string]interface{}) *TradingLogger {
	zapFields := make([]zap.Field, 0, len(fields))
	sugarFields := make([]interface{}, 0, len(fields)*2)
	
	for key, value := range fields {
		zapFields = append(zapFields, zap.Any(key, value))
		sugarFields = append(sugarFields, key, value)
	}

	return &TradingLogger{
		logger: tl.logger.With(zapFields...),
		sugar:  tl.sugar.With(sugarFields...),
	}
}

// Debug logs a debug message
func (tl *TradingLogger) Debug(msg string, fields ...zap.Field) {
	tl.logger.Debug(msg, fields...)
}

// Info logs an info message
func (tl *TradingLogger) Info(msg string, fields ...zap.Field) {
	tl.logger.Info(msg, fields...)
}

// Warn logs a warning message
func (tl *TradingLogger) Warn(msg string, fields ...zap.Field) {
	tl.logger.Warn(msg, fields...)
}

// Error logs an error message
func (tl *TradingLogger) Error(msg string, fields ...zap.Field) {
	tl.logger.Error(msg, fields...)
}

// Panic logs a panic message and panics
func (tl *TradingLogger) Panic(msg string, fields ...zap.Field) {
	tl.logger.Panic(msg, fields...)
}

// Fatal logs a fatal message and exits
func (tl *TradingLogger) Fatal(msg string, fields ...zap.Field) {
	tl.logger.Fatal(msg, fields...)
}

// Debugf logs a debug message with formatting
func (tl *TradingLogger) Debugf(template string, args ...interface{}) {
	tl.sugar.Debugf(template, args...)
}

// Infof logs an info message with formatting
func (tl *TradingLogger) Infof(template string, args ...interface{}) {
	tl.sugar.Infof(template, args...)
}

// Warnf logs a warning message with formatting
func (tl *TradingLogger) Warnf(template string, args ...interface{}) {
	tl.sugar.Warnf(template, args...)
}

// Errorf logs an error message with formatting
func (tl *TradingLogger) Errorf(template string, args ...interface{}) {
	tl.sugar.Errorf(template, args...)
}

// Panicf logs a panic message with formatting and panics
func (tl *TradingLogger) Panicf(template string, args ...interface{}) {
	tl.sugar.Panicf(template, args...)
}

// Fatalf logs a fatal message with formatting and exits
func (tl *TradingLogger) Fatalf(template string, args ...interface{}) {
	tl.sugar.Fatalf(template, args...)
}

// Sync flushes any buffered log entries
func (tl *TradingLogger) Sync() error {
	return tl.logger.Sync()
}

// Trading-specific logging methods

// LogOrderEvent logs an order-related event
func (tl *TradingLogger) LogOrderEvent(ctx context.Context, event string, orderID string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "order"),
		zap.String("event", event),
		zap.String("order_id", orderID),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Info("Order event", allFields...)
}

// LogTradeEvent logs a trade-related event
func (tl *TradingLogger) LogTradeEvent(ctx context.Context, event string, tradeID string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "trade"),
		zap.String("event", event),
		zap.String("trade_id", tradeID),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Info("Trade event", allFields...)
}

// LogRiskEvent logs a risk-related event
func (tl *TradingLogger) LogRiskEvent(ctx context.Context, event string, riskLevel string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "risk"),
		zap.String("event", event),
		zap.String("risk_level", riskLevel),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Warn("Risk event", allFields...)
}

// LogPerformanceEvent logs a performance-related event
func (tl *TradingLogger) LogPerformanceEvent(ctx context.Context, operation string, duration time.Duration, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "performance"),
		zap.String("operation", operation),
		zap.Duration("duration", duration),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Info("Performance event", allFields...)
}

// LogComplianceEvent logs a compliance-related event
func (tl *TradingLogger) LogComplianceEvent(ctx context.Context, event string, status string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "compliance"),
		zap.String("event", event),
		zap.String("status", status),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Info("Compliance event", allFields...)
}

// LogSystemEvent logs a system-related event
func (tl *TradingLogger) LogSystemEvent(ctx context.Context, event string, component string, fields ...zap.Field) {
	allFields := append(fields,
		zap.String("event_type", "system"),
		zap.String("event", event),
		zap.String("component", component),
		zap.Time("timestamp", time.Now()),
	)
	tl.WithContext(ctx).Info("System event", allFields...)
}

// Global logger instance
var globalLogger *TradingLogger

// InitGlobalLogger initializes the global logger
func InitGlobalLogger(config *Config) error {
	logger, err := NewTradingLogger(config)
	if err != nil {
		return err
	}
	globalLogger = logger
	return nil
}

// GetGlobalLogger returns the global logger instance
func GetGlobalLogger() *TradingLogger {
	if globalLogger == nil {
		// Fallback to development logger if global logger is not initialized
		logger, err := NewDevelopmentLogger()
		if err != nil {
			// Last resort: create a basic logger
			zapLogger, _ := zap.NewDevelopment()
			globalLogger = &TradingLogger{
				logger: zapLogger,
				sugar:  zapLogger.Sugar(),
			}
		} else {
			globalLogger = logger
		}
	}
	return globalLogger
}

// Convenience functions using the global logger

// Debug logs a debug message using the global logger
func Debug(msg string, fields ...zap.Field) {
	GetGlobalLogger().Debug(msg, fields...)
}

// Info logs an info message using the global logger
func Info(msg string, fields ...zap.Field) {
	GetGlobalLogger().Info(msg, fields...)
}

// Warn logs a warning message using the global logger
func Warn(msg string, fields ...zap.Field) {
	GetGlobalLogger().Warn(msg, fields...)
}

// Error logs an error message using the global logger
func Error(msg string, fields ...zap.Field) {
	GetGlobalLogger().Error(msg, fields...)
}

// Panic logs a panic message using the global logger and panics
func Panic(msg string, fields ...zap.Field) {
	GetGlobalLogger().Panic(msg, fields...)
}

// Fatal logs a fatal message using the global logger and exits
func Fatal(msg string, fields ...zap.Field) {
	GetGlobalLogger().Fatal(msg, fields...)
}

// Debugf logs a debug message with formatting using the global logger
func Debugf(template string, args ...interface{}) {
	GetGlobalLogger().Debugf(template, args...)
}

// Infof logs an info message with formatting using the global logger
func Infof(template string, args ...interface{}) {
	GetGlobalLogger().Infof(template, args...)
}

// Warnf logs a warning message with formatting using the global logger
func Warnf(template string, args ...interface{}) {
	GetGlobalLogger().Warnf(template, args...)
}

// Errorf logs an error message with formatting using the global logger
func Errorf(template string, args ...interface{}) {
	GetGlobalLogger().Errorf(template, args...)
}

// Panicf logs a panic message with formatting using the global logger and panics
func Panicf(template string, args ...interface{}) {
	GetGlobalLogger().Panicf(template, args...)
}

// Fatalf logs a fatal message with formatting using the global logger and exits
func Fatalf(template string, args ...interface{}) {
	GetGlobalLogger().Fatalf(template, args...)
}

// WithContext adds context information to the global logger
func WithContext(ctx context.Context) *TradingLogger {
	return GetGlobalLogger().WithContext(ctx)
}

// WithFields adds structured fields to the global logger
func WithFields(fields map[string]interface{}) *TradingLogger {
	return GetGlobalLogger().WithFields(fields)
}

// Sync flushes any buffered log entries from the global logger
func Sync() error {
	return GetGlobalLogger().Sync()
}

// init initializes the global logger with default configuration
func init() {
	// Try to initialize with production config if in production environment
	config := DefaultConfig()
	if os.Getenv("ENVIRONMENT") == "development" {
		config.Level = DebugLevel
		config.Development = true
	}
	
	if err := InitGlobalLogger(config); err != nil {
		// Fallback to basic logger if initialization fails
		zapLogger, _ := zap.NewDevelopment()
		globalLogger = &TradingLogger{
			logger: zapLogger,
			sugar:  zapLogger.Sugar(),
		}
	}
}
