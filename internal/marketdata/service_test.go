package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/patrickmn/go-cache"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
)

// MockExternalManager mocks the external manager
type MockExternalManager struct {
	mock.Mock
}

func (m *MockExternalManager) AddProvider(source string, config interface{}) error {
	args := m.Called(source, config)
	return args.Error(0)
}

func (m *MockExternalManager) GetHistoricalData(symbol string, timeRange interface{}) (interface{}, error) {
	args := m.Called(symbol, timeRange)
	return args.Get(0), args.Error(1)
}

func (m *MockExternalManager) Start(ctx context.Context) error {
	args := m.Called(ctx)
	return args.Error(0)
}

func (m *MockExternalManager) Stop() error {
	args := m.Called()
	return args.Error(0)
}

func TestService_AddMarketDataSource(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockManager := new(MockExternalManager)
	
	service := &Service{
		logger:          logger,
		externalManager: mockManager,
		cache:           cache.New(5*time.Minute, 10*time.Minute),
	}

	tests := []struct {
		name        string
		source      string
		config      interface{}
		setupMock   func()
		expectError bool
	}{
		{
			name:   "successful source addition",
			source: "binance",
			config: map[string]string{"api_key": "test"},
			setupMock: func() {
				mockManager.On("AddProvider", "binance", mock.Anything).Return(nil)
			},
			expectError: false,
		},
		{
			name:        "empty source name",
			source:      "",
			config:      nil,
			setupMock:   func() {},
			expectError: true,
		},
		{
			name:   "manager error",
			source: "coinbase",
			config: map[string]string{"api_key": "test"},
			setupMock: func() {
				mockManager.On("AddProvider", "coinbase", mock.Anything).Return(assert.AnError)
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager.ExpectedCalls = nil
			tt.setupMock()

			err := service.AddMarketDataSource(context.Background(), tt.source, tt.config)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestService_GetMarketData(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mockManager := new(MockExternalManager)
	testCache := cache.New(5*time.Minute, 10*time.Minute)
	
	service := &Service{
		logger:          logger,
		externalManager: mockManager,
		cache:           testCache,
	}

	tests := []struct {
		name        string
		symbol      string
		timeRange   interface{}
		setupMock   func()
		setupCache  func()
		expectError bool
	}{
		{
			name:      "successful data retrieval",
			symbol:    "BTCUSDT",
			timeRange: "1h",
			setupMock: func() {
				mockManager.On("GetHistoricalData", "BTCUSDT", "1h").Return(
					map[string]interface{}{"price": 50000}, nil)
			},
			setupCache: func() {},
			expectError: false,
		},
		{
			name:        "empty symbol",
			symbol:      "",
			timeRange:   "1h",
			setupMock:   func() {},
			setupCache:  func() {},
			expectError: true,
		},
		{
			name:      "cache hit",
			symbol:    "ETHUSDT",
			timeRange: "1h",
			setupMock: func() {},
			setupCache: func() {
				testCache.Set("market_data:ETHUSDT", map[string]interface{}{
					"symbol": "ETHUSDT",
					"price":  3000,
				}, cache.DefaultExpiration)
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockManager.ExpectedCalls = nil
			testCache.Flush()
			tt.setupMock()
			tt.setupCache()

			result, err := service.GetMarketData(context.Background(), tt.symbol, tt.timeRange)

			if tt.expectError {
				assert.Error(t, err)
				assert.Nil(t, result)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, result)
			}

			mockManager.AssertExpectations(t)
		})
	}
}

func TestService_GetTicker(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	testCache := cache.New(5*time.Minute, 10*time.Minute)
	
	service := &Service{
		logger: logger,
		cache:  testCache,
	}

	tests := []struct {
		name        string
		symbol      string
		expectError bool
	}{
		{
			name:        "valid symbol",
			symbol:      "BTCUSDT",
			expectError: false,
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
		cache:  testCache,
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
	testCache := cache.New(5*time.Minute, 10*time.Minute)
	
	service := &Service{
		logger: logger,
		cache:  testCache,
	}

	ctx := context.Background()
	symbol := "BTCUSDT"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetTicker(ctx, symbol)
	}
}

