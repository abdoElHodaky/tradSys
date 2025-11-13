package logging

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	tradingContext "github.com/abdoElHodaky/tradSys/pkg/context"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	require.NotNil(t, config)
	
	assert.Equal(t, InfoLevel, config.Level)
	assert.False(t, config.Development)
	assert.False(t, config.DisableCaller)
	assert.False(t, config.DisableStacktrace)
	assert.Equal(t, []string{"stdout"}, config.OutputPaths)
	assert.Equal(t, []string{"stderr"}, config.ErrorOutputPaths)
}

func TestNewTradingLogger(t *testing.T) {
	t.Run("WithDefaultConfig", func(t *testing.T) {
		logger, err := NewTradingLogger(nil)
		require.NoError(t, err)
		require.NotNil(t, logger)
		
		// Test basic logging
		logger.Info("Test message")
		logger.Debug("Debug message") // Should not appear with default config
	})

	t.Run("WithCustomConfig", func(t *testing.T) {
		config := &Config{
			Level:       DebugLevel,
			Development: true,
		}
		
		logger, err := NewTradingLogger(config)
		require.NoError(t, err)
		require.NotNil(t, logger)
		
		// Test basic logging
		logger.Debug("Debug message") // Should appear with debug level
		logger.Info("Info message")
	})
}

func TestNewDevelopmentLogger(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
	
	// Test logging methods
	logger.Debug("Development debug message")
	logger.Info("Development info message")
	logger.Warn("Development warning message")
}

func TestNewProductionLogger(t *testing.T) {
	logger, err := NewProductionLogger()
	require.NoError(t, err)
	require.NotNil(t, logger)
	
	// Test logging methods
	logger.Info("Production info message")
	logger.Warn("Production warning message")
	logger.Error("Production error message")
}

func TestTradingLoggerWithContext(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	
	ctx := context.Background()
	ctx = tradingContext.WithUserID(ctx, "user123")
	ctx = tradingContext.WithOrderID(ctx, "order456")
	ctx = tradingContext.WithSymbol(ctx, "AAPL")
	ctx = tradingContext.WithExchange(ctx, "NASDAQ")
	
	contextLogger := logger.WithContext(ctx)
	require.NotNil(t, contextLogger)
	
	// Test that context information is included
	contextLogger.Info("Message with context")
}

func TestTradingLoggerWithFields(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	
	fields := map[string]interface{}{
		"custom_field": "custom_value",
		"number":       42,
		"boolean":      true,
	}
	
	fieldLogger := logger.WithFields(fields)
	require.NotNil(t, fieldLogger)
	
	fieldLogger.Info("Message with custom fields")
}

func TestTradingLoggerMethods(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	
	// Test structured logging methods
	logger.Debug("Debug message", zap.String("key", "value"))
	logger.Info("Info message", zap.Int("count", 10))
	logger.Warn("Warning message", zap.Bool("flag", true))
	logger.Error("Error message", zap.Duration("duration", time.Second))
	
	// Test formatted logging methods
	logger.Debugf("Debug formatted: %s", "value")
	logger.Infof("Info formatted: %d", 42)
	logger.Warnf("Warning formatted: %t", true)
	logger.Errorf("Error formatted: %v", time.Now())
}

func TestTradingSpecificLogging(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	
	ctx := context.Background()
	ctx = tradingContext.WithUserID(ctx, "user123")
	ctx = tradingContext.WithOrderID(ctx, "order456")
	
	// Test trading-specific logging methods
	logger.LogOrderEvent(ctx, "order_created", "order456", zap.String("symbol", "AAPL"))
	logger.LogTradeEvent(ctx, "trade_executed", "trade789", zap.Float64("price", 150.25))
	logger.LogRiskEvent(ctx, "risk_threshold_exceeded", "HIGH", zap.String("reason", "position_size"))
	logger.LogPerformanceEvent(ctx, "order_processing", 50*time.Millisecond, zap.String("engine", "matching"))
	logger.LogComplianceEvent(ctx, "compliance_check", "PASSED", zap.String("rule", "islamic_finance"))
	logger.LogSystemEvent(ctx, "service_started", "order_service", zap.Int("port", 8080))
}

func TestGlobalLogger(t *testing.T) {
	// Test global logger initialization
	config := &Config{
		Level:       DebugLevel,
		Development: true,
	}
	
	err := InitGlobalLogger(config)
	require.NoError(t, err)
	
	globalLogger := GetGlobalLogger()
	require.NotNil(t, globalLogger)
	
	// Test global convenience functions
	Debug("Global debug message", zap.String("source", "test"))
	Info("Global info message", zap.String("source", "test"))
	Warn("Global warning message", zap.String("source", "test"))
	Error("Global error message", zap.String("source", "test"))
	
	// Test global formatted functions
	Debugf("Global debug formatted: %s", "test")
	Infof("Global info formatted: %d", 123)
	Warnf("Global warning formatted: %t", true)
	Errorf("Global error formatted: %v", time.Now())
	
	// Test global context and fields functions
	ctx := context.Background()
	ctx = tradingContext.WithUserID(ctx, "global_user")
	
	contextLogger := WithContext(ctx)
	require.NotNil(t, contextLogger)
	contextLogger.Info("Global context logger message")
	
	fieldsLogger := WithFields(map[string]interface{}{"global": true})
	require.NotNil(t, fieldsLogger)
	fieldsLogger.Info("Global fields logger message")
}

func TestLogLevelParsing(t *testing.T) {
	testCases := []struct {
		input    LogLevel
		expected string
	}{
		{DebugLevel, "debug"},
		{InfoLevel, "info"},
		{WarnLevel, "warn"},
		{ErrorLevel, "error"},
		{PanicLevel, "panic"},
		{FatalLevel, "fatal"},
	}
	
	for _, tc := range testCases {
		t.Run(string(tc.input), func(t *testing.T) {
			level := parseLogLevel(tc.input)
			assert.NotNil(t, level)
		})
	}
}

func TestLoggerSync(t *testing.T) {
	logger, err := NewDevelopmentLogger()
	require.NoError(t, err)
	
	logger.Info("Message before sync")
	
	err = logger.Sync()
	// Sync might return an error on some systems (like when stdout is not syncable)
	// but that's okay for testing purposes
	if err != nil {
		t.Logf("Sync returned error (this is often expected): %v", err)
	}
	
	// Test global sync
	err = Sync()
	if err != nil {
		t.Logf("Global sync returned error (this is often expected): %v", err)
	}
}

// Benchmark tests
func BenchmarkTradingLoggerInfo(b *testing.B) {
	logger, _ := NewProductionLogger()
	
	for i := 0; i < b.N; i++ {
		logger.Info("Benchmark message", zap.Int("iteration", i))
	}
}

func BenchmarkTradingLoggerWithContext(b *testing.B) {
	logger, _ := NewProductionLogger()
	ctx := context.Background()
	ctx = tradingContext.WithUserID(ctx, "user123")
	ctx = tradingContext.WithOrderID(ctx, "order456")
	
	for i := 0; i < b.N; i++ {
		contextLogger := logger.WithContext(ctx)
		contextLogger.Info("Benchmark message with context")
	}
}

func BenchmarkTradingLoggerWithFields(b *testing.B) {
	logger, _ := NewProductionLogger()
	fields := map[string]interface{}{
		"field1": "value1",
		"field2": 42,
		"field3": true,
	}
	
	for i := 0; i < b.N; i++ {
		fieldLogger := logger.WithFields(fields)
		fieldLogger.Info("Benchmark message with fields")
	}
}

func BenchmarkGlobalLoggerInfo(b *testing.B) {
	config := &Config{
		Level:       InfoLevel,
		Development: false,
	}
	InitGlobalLogger(config)
	
	for i := 0; i < b.N; i++ {
		Info("Global benchmark message", zap.Int("iteration", i))
	}
}

func BenchmarkLogOrderEvent(b *testing.B) {
	logger, _ := NewProductionLogger()
	ctx := context.Background()
	ctx = tradingContext.WithUserID(ctx, "user123")
	
	for i := 0; i < b.N; i++ {
		logger.LogOrderEvent(ctx, "order_created", "order123", zap.String("symbol", "AAPL"))
	}
}
