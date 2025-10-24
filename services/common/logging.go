// Package common provides unified logging for all TradSys v3 services
package common

import (
	"context"
	"fmt"
	"os"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the standard logging interface for all TradSys services
type Logger interface {
	Debug(msg string, fields ...interface{})
	Info(msg string, fields ...interface{})
	Warn(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Fatal(msg string, fields ...interface{})
	
	With(fields ...interface{}) Logger
	WithContext(ctx context.Context) Logger
	WithService(service string) Logger
}

// StructuredLogger provides a structured logging implementation using Zap
type StructuredLogger struct {
	logger *zap.Logger
	fields []zap.Field
}

// NewStructuredLogger creates a new structured logger for a service
func NewStructuredLogger(serviceName string, level string) *StructuredLogger {
	config := zap.NewProductionConfig()
	
	// Set log level
	switch level {
	case "debug":
		config.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	case "info":
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	case "warn":
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	case "error":
		config.Level = zap.NewAtomicLevelAt(zap.ErrorLevel)
	default:
		config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)
	}
	
	// Configure output format
	config.Encoding = "json"
	config.EncoderConfig = zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "message",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}
	
	// Set initial fields
	config.InitialFields = map[string]interface{}{
		"service": serviceName,
		"version": "v3.0.0",
		"pid":     os.Getpid(),
	}
	
	logger, err := config.Build()
	if err != nil {
		// Fallback to development logger if production config fails
		logger, _ = zap.NewDevelopment()
	}
	
	return &StructuredLogger{
		logger: logger,
		fields: []zap.Field{},
	}
}

// Debug logs a debug message
func (sl *StructuredLogger) Debug(msg string, fields ...interface{}) {
	sl.logger.Debug(msg, sl.convertFields(fields...)...)
}

// Info logs an info message
func (sl *StructuredLogger) Info(msg string, fields ...interface{}) {
	sl.logger.Info(msg, sl.convertFields(fields...)...)
}

// Warn logs a warning message
func (sl *StructuredLogger) Warn(msg string, fields ...interface{}) {
	sl.logger.Warn(msg, sl.convertFields(fields...)...)
}

// Error logs an error message
func (sl *StructuredLogger) Error(msg string, fields ...interface{}) {
	sl.logger.Error(msg, sl.convertFields(fields...)...)
}

// Fatal logs a fatal message and exits
func (sl *StructuredLogger) Fatal(msg string, fields ...interface{}) {
	sl.logger.Fatal(msg, sl.convertFields(fields...)...)
}

// With returns a logger with additional fields
func (sl *StructuredLogger) With(fields ...interface{}) Logger {
	newFields := append(sl.fields, sl.convertFields(fields...)...)
	return &StructuredLogger{
		logger: sl.logger,
		fields: newFields,
	}
}

// WithContext returns a logger with context information
func (sl *StructuredLogger) WithContext(ctx context.Context) Logger {
	fields := sl.extractContextFields(ctx)
	return sl.With(fields...)
}

// WithService returns a logger with service information
func (sl *StructuredLogger) WithService(service string) Logger {
	return sl.With("service", service)
}

// convertFields converts interface{} fields to zap.Field
func (sl *StructuredLogger) convertFields(fields ...interface{}) []zap.Field {
	if len(fields)%2 != 0 {
		// If odd number of fields, add the last one as a generic field
		fields = append(fields, "")
	}
	
	zapFields := make([]zap.Field, 0, len(fields)/2+len(sl.fields))
	
	// Add existing fields
	zapFields = append(zapFields, sl.fields...)
	
	// Convert new fields
	for i := 0; i < len(fields); i += 2 {
		key, ok := fields[i].(string)
		if !ok {
			key = fmt.Sprintf("field_%d", i/2)
		}
		
		value := fields[i+1]
		zapFields = append(zapFields, zap.Any(key, value))
	}
	
	return zapFields
}

// extractContextFields extracts logging fields from context
func (sl *StructuredLogger) extractContextFields(ctx context.Context) []interface{} {
	var fields []interface{}
	
	// Extract request ID if present
	if requestID := ctx.Value("request_id"); requestID != nil {
		fields = append(fields, "request_id", requestID)
	}
	
	// Extract user ID if present
	if userID := ctx.Value("user_id"); userID != nil {
		fields = append(fields, "user_id", userID)
	}
	
	// Extract trace ID if present
	if traceID := ctx.Value("trace_id"); traceID != nil {
		fields = append(fields, "trace_id", traceID)
	}
	
	// Extract span ID if present
	if spanID := ctx.Value("span_id"); spanID != nil {
		fields = append(fields, "span_id", spanID)
	}
	
	return fields
}

// LoggerConfig represents logger configuration
type LoggerConfig struct {
	Level      string `yaml:"level" json:"level"`
	Format     string `yaml:"format" json:"format"`
	Output     string `yaml:"output" json:"output"`
	MaxSize    int    `yaml:"max_size" json:"max_size"`
	MaxBackups int    `yaml:"max_backups" json:"max_backups"`
	MaxAge     int    `yaml:"max_age" json:"max_age"`
	Compress   bool   `yaml:"compress" json:"compress"`
}

// DefaultLoggerConfig returns default logger configuration
func DefaultLoggerConfig() *LoggerConfig {
	return &LoggerConfig{
		Level:      "info",
		Format:     "json",
		Output:     "stdout",
		MaxSize:    100, // MB
		MaxBackups: 3,
		MaxAge:     28, // days
		Compress:   true,
	}
}

// LoggerFactory creates loggers with consistent configuration
type LoggerFactory struct {
	config *LoggerConfig
}

// NewLoggerFactory creates a new logger factory
func NewLoggerFactory(config *LoggerConfig) *LoggerFactory {
	if config == nil {
		config = DefaultLoggerConfig()
	}
	return &LoggerFactory{config: config}
}

// CreateLogger creates a new logger for a service
func (lf *LoggerFactory) CreateLogger(serviceName string) Logger {
	return NewStructuredLogger(serviceName, lf.config.Level)
}

// CreateLoggerWithFields creates a new logger with additional fields
func (lf *LoggerFactory) CreateLoggerWithFields(serviceName string, fields ...interface{}) Logger {
	logger := NewStructuredLogger(serviceName, lf.config.Level)
	return logger.With(fields...)
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                 `json:"level"`
	Service   string                 `json:"service"`
	Message   string                 `json:"message"`
	Fields    map[string]interface{} `json:"fields,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
}

// ContextKey represents a context key for logging
type ContextKey string

const (
	// RequestIDKey is the context key for request ID
	RequestIDKey ContextKey = "request_id"
	// UserIDKey is the context key for user ID
	UserIDKey ContextKey = "user_id"
	// TraceIDKey is the context key for trace ID
	TraceIDKey ContextKey = "trace_id"
	// SpanIDKey is the context key for span ID
	SpanIDKey ContextKey = "span_id"
	// ServiceKey is the context key for service name
	ServiceKey ContextKey = "service"
)

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// WithTraceID adds trace ID to context
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// WithSpanID adds span ID to context
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// WithService adds service name to context
func WithService(ctx context.Context, service string) context.Context {
	return context.WithValue(ctx, ServiceKey, service)
}

// GetRequestID extracts request ID from context
func GetRequestID(ctx context.Context) string {
	if requestID, ok := ctx.Value(RequestIDKey).(string); ok {
		return requestID
	}
	return ""
}

// GetUserID extracts user ID from context
func GetUserID(ctx context.Context) string {
	if userID, ok := ctx.Value(UserIDKey).(string); ok {
		return userID
	}
	return ""
}

// GetTraceID extracts trace ID from context
func GetTraceID(ctx context.Context) string {
	if traceID, ok := ctx.Value(TraceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// GetSpanID extracts span ID from context
func GetSpanID(ctx context.Context) string {
	if spanID, ok := ctx.Value(SpanIDKey).(string); ok {
		return spanID
	}
	return ""
}

// GetService extracts service name from context
func GetService(ctx context.Context) string {
	if service, ok := ctx.Value(ServiceKey).(string); ok {
		return service
	}
	return ""
}

// LogMiddleware provides logging middleware functionality
type LogMiddleware struct {
	logger Logger
}

// NewLogMiddleware creates a new log middleware
func NewLogMiddleware(logger Logger) *LogMiddleware {
	return &LogMiddleware{logger: logger}
}

// LogRequest logs an incoming request
func (lm *LogMiddleware) LogRequest(ctx context.Context, method, path string, duration time.Duration) {
	lm.logger.WithContext(ctx).Info("Request processed",
		"method", method,
		"path", path,
		"duration_ms", duration.Milliseconds(),
	)
}

// LogError logs an error with context
func (lm *LogMiddleware) LogError(ctx context.Context, err error, message string) {
	lm.logger.WithContext(ctx).Error(message,
		"error", err.Error(),
		"error_type", fmt.Sprintf("%T", err),
	)
}

// LogServiceCall logs a service call
func (lm *LogMiddleware) LogServiceCall(ctx context.Context, service, method string, duration time.Duration, err error) {
	fields := []interface{}{
		"target_service", service,
		"method", method,
		"duration_ms", duration.Milliseconds(),
	}
	
	if err != nil {
		fields = append(fields, "error", err.Error())
		lm.logger.WithContext(ctx).Error("Service call failed", fields...)
	} else {
		lm.logger.WithContext(ctx).Info("Service call completed", fields...)
	}
}

// Performance logging utilities

// PerformanceLogger logs performance metrics
type PerformanceLogger struct {
	logger Logger
}

// NewPerformanceLogger creates a new performance logger
func NewPerformanceLogger(logger Logger) *PerformanceLogger {
	return &PerformanceLogger{logger: logger}
}

// LogLatency logs latency metrics
func (pl *PerformanceLogger) LogLatency(ctx context.Context, operation string, duration time.Duration) {
	pl.logger.WithContext(ctx).Info("Performance metric",
		"metric_type", "latency",
		"operation", operation,
		"duration_ms", duration.Milliseconds(),
		"duration_ns", duration.Nanoseconds(),
	)
}

// LogThroughput logs throughput metrics
func (pl *PerformanceLogger) LogThroughput(ctx context.Context, operation string, count int64, duration time.Duration) {
	throughput := float64(count) / duration.Seconds()
	pl.logger.WithContext(ctx).Info("Performance metric",
		"metric_type", "throughput",
		"operation", operation,
		"count", count,
		"duration_s", duration.Seconds(),
		"throughput_per_sec", throughput,
	)
}

// LogResourceUsage logs resource usage metrics
func (pl *PerformanceLogger) LogResourceUsage(ctx context.Context, resource string, usage float64, limit float64) {
	utilizationPct := (usage / limit) * 100
	pl.logger.WithContext(ctx).Info("Performance metric",
		"metric_type", "resource_usage",
		"resource", resource,
		"usage", usage,
		"limit", limit,
		"utilization_pct", utilizationPct,
	)
}

// Audit logging utilities

// AuditLogger logs audit events
type AuditLogger struct {
	logger Logger
}

// NewAuditLogger creates a new audit logger
func NewAuditLogger(logger Logger) *AuditLogger {
	return &AuditLogger{logger: logger}
}

// LogUserAction logs a user action for audit purposes
func (al *AuditLogger) LogUserAction(ctx context.Context, userID, action, resource string, details map[string]interface{}) {
	fields := []interface{}{
		"audit_type", "user_action",
		"user_id", userID,
		"action", action,
		"resource", resource,
	}
	
	if details != nil {
		fields = append(fields, "details", details)
	}
	
	al.logger.WithContext(ctx).Info("Audit event", fields...)
}

// LogSystemEvent logs a system event for audit purposes
func (al *AuditLogger) LogSystemEvent(ctx context.Context, event, component string, details map[string]interface{}) {
	fields := []interface{}{
		"audit_type", "system_event",
		"event", event,
		"component", component,
	}
	
	if details != nil {
		fields = append(fields, "details", details)
	}
	
	al.logger.WithContext(ctx).Info("Audit event", fields...)
}

// LogSecurityEvent logs a security event for audit purposes
func (al *AuditLogger) LogSecurityEvent(ctx context.Context, event, severity string, details map[string]interface{}) {
	fields := []interface{}{
		"audit_type", "security_event",
		"event", event,
		"severity", severity,
	}
	
	if details != nil {
		fields = append(fields, "details", details)
	}
	
	al.logger.WithContext(ctx).Warn("Security audit event", fields...)
}
