package context

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTradingContext(t *testing.T) {
	tc := NewTradingContext()
	require.NotNil(t, tc)

	t.Run("WithTimeout", func(t *testing.T) {
		ctx, cancel := tc.WithTimeout(5 * time.Second)
		defer cancel()
		
		assert.NotNil(t, ctx)
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("WithDefaultTimeout", func(t *testing.T) {
		ctx, cancel := tc.WithDefaultTimeout()
		defer cancel()
		
		assert.NotNil(t, ctx)
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("WithCancel", func(t *testing.T) {
		ctx, cancel := tc.WithCancel()
		defer cancel()
		
		assert.NotNil(t, ctx)
		assert.NoError(t, ctx.Err())
		
		cancel()
		// Give it a moment to propagate
		time.Sleep(1 * time.Millisecond)
		assert.Error(t, ctx.Err())
	})

	t.Run("WithDeadline", func(t *testing.T) {
		deadline := time.Now().Add(5 * time.Second)
		ctx, cancel := tc.WithDeadline(deadline)
		defer cancel()
		
		assert.NotNil(t, ctx)
		ctxDeadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.Equal(t, deadline.Unix(), ctxDeadline.Unix())
	})
}

func TestStandardContextFunctions(t *testing.T) {
	testCases := []struct {
		name     string
		function func() (context.Context, context.CancelFunc)
		timeout  time.Duration
	}{
		{"ForOrderProcessing", ForOrderProcessing, 5 * time.Second},
		{"ForRiskCalculation", ForRiskCalculation, 10 * time.Second},
		{"ForMarketDataFetch", ForMarketDataFetch, 3 * time.Second},
		{"ForDatabaseOperation", ForDatabaseOperation, 15 * time.Second},
		{"ForWebSocketConnection", ForWebSocketConnection, 30 * time.Second},
		{"ForExchangeAPI", ForExchangeAPI, 10 * time.Second},
		{"ForComplianceCheck", ForComplianceCheck, 20 * time.Second},
		{"ForSystemShutdown", ForSystemShutdown, 30 * time.Second},
		{"ForHighFrequencyTrading", ForHighFrequencyTrading, 100 * time.Millisecond},
		{"ForBatchProcessing", ForBatchProcessing, 5 * time.Minute},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := tc.function()
			defer cancel()
			
			assert.NotNil(t, ctx)
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)
			assert.True(t, deadline.After(time.Now()))
			
			// Verify the timeout is approximately correct (within 1 second tolerance)
			expectedDeadline := time.Now().Add(tc.timeout)
			timeDiff := deadline.Sub(expectedDeadline)
			assert.True(t, timeDiff < time.Second && timeDiff > -time.Second,
				"Expected deadline around %v, got %v (diff: %v)", expectedDeadline, deadline, timeDiff)
		})
	}
}

func TestContextKeys(t *testing.T) {
	ctx := context.Background()

	t.Run("UserID", func(t *testing.T) {
		userID := "user123"
		ctx = WithUserID(ctx, userID)
		
		retrievedUserID, ok := GetUserID(ctx)
		assert.True(t, ok)
		assert.Equal(t, userID, retrievedUserID)
	})

	t.Run("OrderID", func(t *testing.T) {
		orderID := "order456"
		ctx = WithOrderID(ctx, orderID)
		
		retrievedOrderID, ok := GetOrderID(ctx)
		assert.True(t, ok)
		assert.Equal(t, orderID, retrievedOrderID)
	})

	t.Run("SessionID", func(t *testing.T) {
		sessionID := "session789"
		ctx = WithSessionID(ctx, sessionID)
		
		retrievedSessionID, ok := GetSessionID(ctx)
		assert.True(t, ok)
		assert.Equal(t, sessionID, retrievedSessionID)
	})

	t.Run("TradeID", func(t *testing.T) {
		tradeID := "trade101"
		ctx = WithTradeID(ctx, tradeID)
		
		retrievedTradeID, ok := GetTradeID(ctx)
		assert.True(t, ok)
		assert.Equal(t, tradeID, retrievedTradeID)
	})

	t.Run("Symbol", func(t *testing.T) {
		symbol := "AAPL"
		ctx = WithSymbol(ctx, symbol)
		
		retrievedSymbol, ok := GetSymbol(ctx)
		assert.True(t, ok)
		assert.Equal(t, symbol, retrievedSymbol)
	})

	t.Run("Exchange", func(t *testing.T) {
		exchange := "NASDAQ"
		ctx = WithExchange(ctx, exchange)
		
		retrievedExchange, ok := GetExchange(ctx)
		assert.True(t, ok)
		assert.Equal(t, exchange, retrievedExchange)
	})

	t.Run("RequestID", func(t *testing.T) {
		requestID := "req123"
		ctx = WithRequestID(ctx, requestID)
		
		retrievedRequestID, ok := GetRequestID(ctx)
		assert.True(t, ok)
		assert.Equal(t, requestID, retrievedRequestID)
	})

	t.Run("CorrelationID", func(t *testing.T) {
		correlationID := "corr456"
		ctx = WithCorrelationID(ctx, correlationID)
		
		retrievedCorrelationID, ok := GetCorrelationID(ctx)
		assert.True(t, ok)
		assert.Equal(t, correlationID, retrievedCorrelationID)
	})

	t.Run("TraceID", func(t *testing.T) {
		traceID := "trace789"
		ctx = WithTraceID(ctx, traceID)
		
		retrievedTraceID, ok := GetTraceID(ctx)
		assert.True(t, ok)
		assert.Equal(t, traceID, retrievedTraceID)
	})

	t.Run("SpanID", func(t *testing.T) {
		spanID := "span101"
		ctx = WithSpanID(ctx, spanID)
		
		retrievedSpanID, ok := GetSpanID(ctx)
		assert.True(t, ok)
		assert.Equal(t, spanID, retrievedSpanID)
	})
}

func TestContextKeyNotFound(t *testing.T) {
	ctx := context.Background()

	_, ok := GetUserID(ctx)
	assert.False(t, ok)

	_, ok = GetOrderID(ctx)
	assert.False(t, ok)

	_, ok = GetSessionID(ctx)
	assert.False(t, ok)

	_, ok = GetTradeID(ctx)
	assert.False(t, ok)

	_, ok = GetSymbol(ctx)
	assert.False(t, ok)

	_, ok = GetExchange(ctx)
	assert.False(t, ok)

	_, ok = GetRequestID(ctx)
	assert.False(t, ok)

	_, ok = GetCorrelationID(ctx)
	assert.False(t, ok)

	_, ok = GetTraceID(ctx)
	assert.False(t, ok)

	_, ok = GetSpanID(ctx)
	assert.False(t, ok)
}

func TestWithTradingMetadata(t *testing.T) {
	ctx := context.Background()
	userID := "user123"
	orderID := "order456"
	symbol := "AAPL"
	exchange := "NASDAQ"

	ctx = WithTradingMetadata(ctx, userID, orderID, symbol, exchange)

	retrievedUserID, ok := GetUserID(ctx)
	assert.True(t, ok)
	assert.Equal(t, userID, retrievedUserID)

	retrievedOrderID, ok := GetOrderID(ctx)
	assert.True(t, ok)
	assert.Equal(t, orderID, retrievedOrderID)

	retrievedSymbol, ok := GetSymbol(ctx)
	assert.True(t, ok)
	assert.Equal(t, symbol, retrievedSymbol)

	retrievedExchange, ok := GetExchange(ctx)
	assert.True(t, ok)
	assert.Equal(t, exchange, retrievedExchange)
}

func TestWithTracingMetadata(t *testing.T) {
	ctx := context.Background()
	traceID := "trace123"
	spanID := "span456"
	correlationID := "corr789"

	ctx = WithTracingMetadata(ctx, traceID, spanID, correlationID)

	retrievedTraceID, ok := GetTraceID(ctx)
	assert.True(t, ok)
	assert.Equal(t, traceID, retrievedTraceID)

	retrievedSpanID, ok := GetSpanID(ctx)
	assert.True(t, ok)
	assert.Equal(t, spanID, retrievedSpanID)

	retrievedCorrelationID, ok := GetCorrelationID(ctx)
	assert.True(t, ok)
	assert.Equal(t, correlationID, retrievedCorrelationID)
}

// Benchmark tests
func BenchmarkWithUserID(b *testing.B) {
	ctx := context.Background()
	userID := "user123"
	
	for i := 0; i < b.N; i++ {
		_ = WithUserID(ctx, userID)
	}
}

func BenchmarkGetUserID(b *testing.B) {
	ctx := WithUserID(context.Background(), "user123")
	
	for i := 0; i < b.N; i++ {
		_, _ = GetUserID(ctx)
	}
}

func BenchmarkWithTradingMetadata(b *testing.B) {
	ctx := context.Background()
	userID := "user123"
	orderID := "order456"
	symbol := "AAPL"
	exchange := "NASDAQ"
	
	for i := 0; i < b.N; i++ {
		_ = WithTradingMetadata(ctx, userID, orderID, symbol, exchange)
	}
}

func BenchmarkForOrderProcessing(b *testing.B) {
	for i := 0; i < b.N; i++ {
		ctx, cancel := ForOrderProcessing()
		cancel()
		_ = ctx
	}
}
