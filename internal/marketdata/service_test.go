package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestService_AddMarketDataSource(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := external.NewManager(logger)

	service := &Service{
		logger:          logger,
		ExternalManager: manager,
		Cache:           cache.New(5*time.Minute, 10*time.Minute),
	}

	tests := []struct {
		name        string
		source      string
		config      interface{}
		expectError bool
	}{
		{
			name:        "successful source addition",
			source:      "binance",
			config:      map[string]interface{}{"api_key": "test"},
			expectError: false,
		},
		{
			name:        "empty source name",
			source:      "",
			config:      nil,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.AddMarketDataSource(context.Background(), tt.source, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_GetMarketData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := external.NewManager(logger)
	testCache := cache.New(5*time.Minute, 10*time.Minute)

	service := &Service{
		logger:          logger,
		ExternalManager: manager,
		Cache:           testCache,
	}

	tests := []struct {
		name        string
		symbol      string
		timeRange   interface{}
		expectError bool
	}{
		{
			name:        "empty symbol",
			symbol:      "",
			timeRange:   "1h",
			expectError: true,
		},
		{
			name:        "cache hit",
			symbol:      "ETHUSDT",
			timeRange:   "1h",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testCache.Flush()

			// For cache hit test, set up cache
			if tt.name == "cache hit" {
				testCache.Set("market_data:ETHUSDT", map[string]interface{}{
					"symbol": "ETHUSDT",
					"price":  3000,
				}, cache.DefaultExpiration)
			}

			result, err := service.GetMarketData(context.Background(), tt.symbol, tt.timeRange)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func TestService_GetTicker(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	manager := external.NewManager(logger)
	testCache := cache.New(5*time.Minute, 10*time.Minute)

	// Add a provider to the manager for testing
	_ = manager.AddSource("binance", map[string]interface{}{"api_key": "test"})

	service := &Service{
		logger:          logger,
		ExternalManager: manager,
		Cache:           testCache,
	}

	tests := []struct {
		name        string
		symbol      string
		expectError bool
	}{
		{
			name:        "valid symbol",
			symbol:      "BTCUSDT",
			expectError: true, // Will fail because we don't have real API credentials
		},
		{
			name:        "empty symbol",
			symbol:      "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := service.GetTicker(context.Background(), tt.symbol)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}
		})
	}
}

func BenchmarkService_GetMarketData(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	testCache := cache.New(5*time.Minute, 10*time.Minute)

	service := &Service{
		logger: logger,
		Cache:  testCache,
	}

	ctx := context.Background()
	symbol := "BTCUSDT"
	timeRange := "1h"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetMarketData(ctx, symbol, timeRange)
	}
}

func BenchmarkService_GetTicker(b *testing.B) {
	logger, _ := zap.NewDevelopment()
	manager := external.NewManager(logger)
	testCache := cache.New(5*time.Minute, 10*time.Minute)

	// Add a provider to the manager for testing
	_ = manager.AddSource("binance", map[string]interface{}{"api_key": "test"})

	service := &Service{
		logger:          logger,
		ExternalManager: manager,
		Cache:           testCache,
	}

	ctx := context.Background()
	symbol := "BTCUSDT"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetTicker(ctx, symbol)
	}
}
