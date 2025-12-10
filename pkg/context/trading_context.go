// Package context provides standardized context patterns for TradSys trading operations
package context

import (
	"context"
	"time"
)

// TradingContext provides standardized context creation for trading operations
type TradingContext struct {
	defaultTimeout time.Duration
}

// NewTradingContext creates a new trading context manager
func NewTradingContext() *TradingContext {
	return &TradingContext{
		defaultTimeout: 30 * time.Second,
	}
}

// WithTimeout creates a context with the specified timeout
func (tc *TradingContext) WithTimeout(timeout time.Duration) (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), timeout)
}

// WithDefaultTimeout creates a context with the default timeout (30 seconds)
func (tc *TradingContext) WithDefaultTimeout() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), tc.defaultTimeout)
}

// WithCancel creates a cancellable context
func (tc *TradingContext) WithCancel() (context.Context, context.CancelFunc) {
	return context.WithCancel(context.Background())
}

// WithDeadline creates a context with the specified deadline
func (tc *TradingContext) WithDeadline(deadline time.Time) (context.Context, context.CancelFunc) {
	return context.WithDeadline(context.Background(), deadline)
}

// Standard context creation functions for common trading operations

// ForOrderProcessing creates a context optimized for order processing (5 second timeout)
func ForOrderProcessing() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// ForRiskCalculation creates a context optimized for risk calculations (10 second timeout)
func ForRiskCalculation() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// ForMarketDataFetch creates a context optimized for market data fetching (3 second timeout)
func ForMarketDataFetch() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 3*time.Second)
}

// ForDatabaseOperation creates a context optimized for database operations (15 second timeout)
func ForDatabaseOperation() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 15*time.Second)
}

// ForWebSocketConnection creates a context optimized for WebSocket connections (30 second timeout)
func ForWebSocketConnection() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// ForExchangeAPI creates a context optimized for exchange API calls (10 second timeout)
func ForExchangeAPI() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 10*time.Second)
}

// ForComplianceCheck creates a context optimized for compliance checks (20 second timeout)
func ForComplianceCheck() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 20*time.Second)
}

// ForSystemShutdown creates a context optimized for system shutdown (30 second timeout)
func ForSystemShutdown() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 30*time.Second)
}

// ForHighFrequencyTrading creates a context optimized for HFT operations (100ms timeout)
func ForHighFrequencyTrading() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 100*time.Millisecond)
}

// ForBatchProcessing creates a context optimized for batch processing (5 minute timeout)
func ForBatchProcessing() (context.Context, context.CancelFunc) {
	return context.WithTimeout(context.Background(), 5*time.Minute)
}

// ContextKey represents a context key type for trading operations
type ContextKey string

// Standard context keys for trading operations
const (
	UserIDKey        ContextKey = "user_id"
	OrderIDKey       ContextKey = "order_id"
	SessionIDKey     ContextKey = "session_id"
	TradeIDKey       ContextKey = "trade_id"
	SymbolKey        ContextKey = "symbol"
	ExchangeKey      ContextKey = "exchange"
	RequestIDKey     ContextKey = "request_id"
	CorrelationIDKey ContextKey = "correlation_id"
	TraceIDKey       ContextKey = "trace_id"
	SpanIDKey        ContextKey = "span_id"
)

// WithUserID adds user ID to context
func WithUserID(ctx context.Context, userID string) context.Context {
	return context.WithValue(ctx, UserIDKey, userID)
}

// GetUserID retrieves user ID from context
func GetUserID(ctx context.Context) (string, bool) {
	userID, ok := ctx.Value(UserIDKey).(string)
	return userID, ok
}

// WithOrderID adds order ID to context
func WithOrderID(ctx context.Context, orderID string) context.Context {
	return context.WithValue(ctx, OrderIDKey, orderID)
}

// GetOrderID retrieves order ID from context
func GetOrderID(ctx context.Context) (string, bool) {
	orderID, ok := ctx.Value(OrderIDKey).(string)
	return orderID, ok
}

// WithSessionID adds session ID to context
func WithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, SessionIDKey, sessionID)
}

// GetSessionID retrieves session ID from context
func GetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(SessionIDKey).(string)
	return sessionID, ok
}

// WithTradeID adds trade ID to context
func WithTradeID(ctx context.Context, tradeID string) context.Context {
	return context.WithValue(ctx, TradeIDKey, tradeID)
}

// GetTradeID retrieves trade ID from context
func GetTradeID(ctx context.Context) (string, bool) {
	tradeID, ok := ctx.Value(TradeIDKey).(string)
	return tradeID, ok
}

// WithSymbol adds symbol to context
func WithSymbol(ctx context.Context, symbol string) context.Context {
	return context.WithValue(ctx, SymbolKey, symbol)
}

// GetSymbol retrieves symbol from context
func GetSymbol(ctx context.Context) (string, bool) {
	symbol, ok := ctx.Value(SymbolKey).(string)
	return symbol, ok
}

// WithExchange adds exchange to context
func WithExchange(ctx context.Context, exchange string) context.Context {
	return context.WithValue(ctx, ExchangeKey, exchange)
}

// GetExchange retrieves exchange from context
func GetExchange(ctx context.Context) (string, bool) {
	exchange, ok := ctx.Value(ExchangeKey).(string)
	return exchange, ok
}

// WithRequestID adds request ID to context
func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, RequestIDKey, requestID)
}

// GetRequestID retrieves request ID from context
func GetRequestID(ctx context.Context) (string, bool) {
	requestID, ok := ctx.Value(RequestIDKey).(string)
	return requestID, ok
}

// WithCorrelationID adds correlation ID to context
func WithCorrelationID(ctx context.Context, correlationID string) context.Context {
	return context.WithValue(ctx, CorrelationIDKey, correlationID)
}

// GetCorrelationID retrieves correlation ID from context
func GetCorrelationID(ctx context.Context) (string, bool) {
	correlationID, ok := ctx.Value(CorrelationIDKey).(string)
	return correlationID, ok
}

// WithTraceID adds trace ID to context for distributed tracing
func WithTraceID(ctx context.Context, traceID string) context.Context {
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// GetTraceID retrieves trace ID from context
func GetTraceID(ctx context.Context) (string, bool) {
	traceID, ok := ctx.Value(TraceIDKey).(string)
	return traceID, ok
}

// WithSpanID adds span ID to context for distributed tracing
func WithSpanID(ctx context.Context, spanID string) context.Context {
	return context.WithValue(ctx, SpanIDKey, spanID)
}

// GetSpanID retrieves span ID from context
func GetSpanID(ctx context.Context) (string, bool) {
	spanID, ok := ctx.Value(SpanIDKey).(string)
	return spanID, ok
}

// WithTradingMetadata adds multiple trading-related metadata to context
func WithTradingMetadata(ctx context.Context, userID, orderID, symbol, exchange string) context.Context {
	ctx = WithUserID(ctx, userID)
	ctx = WithOrderID(ctx, orderID)
	ctx = WithSymbol(ctx, symbol)
	ctx = WithExchange(ctx, exchange)
	return ctx
}

// WithTracingMetadata adds distributed tracing metadata to context
func WithTracingMetadata(ctx context.Context, traceID, spanID, correlationID string) context.Context {
	ctx = WithTraceID(ctx, traceID)
	ctx = WithSpanID(ctx, spanID)
	ctx = WithCorrelationID(ctx, correlationID)
	return ctx
}
