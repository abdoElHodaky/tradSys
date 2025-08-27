package marketdata

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest"
)

// MockMarketDataRepository is a mock implementation of the market data repository
type MockMarketDataRepository struct {
	mock.Mock
}

// GetPairs returns mock pairs
func (m *MockMarketDataRepository) GetPairs() ([]models.Pair, error) {
	args := m.Called()
	return args.Get(0).([]models.Pair), args.Error(1)
}

// GetPairBySymbol returns a mock pair by symbol
func (m *MockMarketDataRepository) GetPairBySymbol(symbol string) (*models.Pair, error) {
	args := m.Called(symbol)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Pair), args.Error(1)
}

// SaveMarketData saves mock market data
func (m *MockMarketDataRepository) SaveMarketData(data *models.MarketData) error {
	args := m.Called(data)
	return args.Error(0)
}

// GetMarketData returns mock market data
func (m *MockMarketDataRepository) GetMarketData(symbol string, start, end time.Time) ([]models.MarketData, error) {
	args := m.Called(symbol, start, end)
	return args.Get(0).([]models.MarketData), args.Error(1)
}

// MockExternalProvider is a mock implementation of the external market data provider
type MockExternalProvider struct {
	mock.Mock
}

// GetMarketData returns mock market data from an external provider
func (m *MockExternalProvider) GetMarketData(symbol string, interval string) (*models.MarketData, error) {
	args := m.Called(symbol, interval)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.MarketData), args.Error(1)
}

func TestMarketDataService_GetPairs(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock repository
	mockRepo := new(MockMarketDataRepository)
	mockRepo.On("GetPairs").Return([]models.Pair{
		{ID: 1, Symbol: "BTC/USD", BaseAsset: "BTC", QuoteAsset: "USD"},
		{ID: 2, Symbol: "ETH/USD", BaseAsset: "ETH", QuoteAsset: "USD"},
	}, nil)

	// Create a mock external provider
	mockProvider := new(MockExternalProvider)

	// Create a service
	service := NewService(ServiceParams{
		Logger:             logger,
		Repository:         mockRepo,
		ExternalProviders:  []ExternalProvider{mockProvider},
		UpdateInterval:     time.Second,
		SupportedSymbols:   []string{"BTC/USD", "ETH/USD"},
		SupportedIntervals: []string{"1m", "5m", "1h"},
	})

	// Get pairs
	pairs, err := service.GetPairs()

	// Verify the result
	assert.NoError(t, err)
	assert.Len(t, pairs, 2)
	assert.Equal(t, "BTC/USD", pairs[0].Symbol)
	assert.Equal(t, "ETH/USD", pairs[1].Symbol)

	// Verify that the mock was called
	mockRepo.AssertExpectations(t)
}

func TestMarketDataService_GetMarketData(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock repository
	mockRepo := new(MockMarketDataRepository)
	
	// Set up the mock to return market data
	now := time.Now()
	start := now.Add(-1 * time.Hour)
	end := now
	
	mockData := []models.MarketData{
		{
			ID:        1,
			Symbol:    "BTC/USD",
			Timestamp: start.Add(10 * time.Minute),
			Open:      10000.0,
			High:      10100.0,
			Low:       9900.0,
			Close:     10050.0,
			Volume:    1.5,
		},
		{
			ID:        2,
			Symbol:    "BTC/USD",
			Timestamp: start.Add(20 * time.Minute),
			Open:      10050.0,
			High:      10200.0,
			Low:       10000.0,
			Close:     10150.0,
			Volume:    2.0,
		},
	}
	
	mockRepo.On("GetMarketData", "BTC/USD", start, end).Return(mockData, nil)

	// Create a mock external provider
	mockProvider := new(MockExternalProvider)

	// Create a service
	service := NewService(ServiceParams{
		Logger:             logger,
		Repository:         mockRepo,
		ExternalProviders:  []ExternalProvider{mockProvider},
		UpdateInterval:     time.Second,
		SupportedSymbols:   []string{"BTC/USD", "ETH/USD"},
		SupportedIntervals: []string{"1m", "5m", "1h"},
	})

	// Get market data
	data, err := service.GetMarketData("BTC/USD", start, end)

	// Verify the result
	assert.NoError(t, err)
	assert.Len(t, data, 2)
	assert.Equal(t, "BTC/USD", data[0].Symbol)
	assert.Equal(t, 10000.0, data[0].Open)
	assert.Equal(t, 10050.0, data[1].Close)

	// Verify that the mock was called
	mockRepo.AssertExpectations(t)
}

func TestMarketDataService_UpdateMarketData(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock repository
	mockRepo := new(MockMarketDataRepository)
	
	// Create a mock external provider
	mockProvider := new(MockExternalProvider)
	
	// Set up the mock to return market data
	now := time.Now()
	mockData := &models.MarketData{
		Symbol:    "BTC/USD",
		Timestamp: now,
		Open:      10000.0,
		High:      10100.0,
		Low:       9900.0,
		Close:     10050.0,
		Volume:    1.5,
	}
	
	mockProvider.On("GetMarketData", "BTC/USD", "1m").Return(mockData, nil)
	mockRepo.On("SaveMarketData", mockData).Return(nil)

	// Create a service
	service := NewService(ServiceParams{
		Logger:             logger,
		Repository:         mockRepo,
		ExternalProviders:  []ExternalProvider{mockProvider},
		UpdateInterval:     time.Second,
		SupportedSymbols:   []string{"BTC/USD"},
		SupportedIntervals: []string{"1m"},
	})

	// Update market data
	err := service.UpdateMarketData(context.Background())

	// Verify the result
	assert.NoError(t, err)

	// Verify that the mocks were called
	mockProvider.AssertExpectations(t)
	mockRepo.AssertExpectations(t)
}

func TestMarketDataService_GetPairBySymbol(t *testing.T) {
	// Create a test logger
	logger := zaptest.NewLogger(t)

	// Create a mock repository
	mockRepo := new(MockMarketDataRepository)
	
	// Set up the mock to return a pair
	mockPair := &models.Pair{
		ID:         1,
		Symbol:     "BTC/USD",
		BaseAsset:  "BTC",
		QuoteAsset: "USD",
	}
	
	mockRepo.On("GetPairBySymbol", "BTC/USD").Return(mockPair, nil)
	mockRepo.On("GetPairBySymbol", "INVALID").Return(nil, models.ErrPairNotFound)

	// Create a mock external provider
	mockProvider := new(MockExternalProvider)

	// Create a service
	service := NewService(ServiceParams{
		Logger:             logger,
		Repository:         mockRepo,
		ExternalProviders:  []ExternalProvider{mockProvider},
		UpdateInterval:     time.Second,
		SupportedSymbols:   []string{"BTC/USD"},
		SupportedIntervals: []string{"1m"},
	})

	// Get a valid pair
	pair, err := service.GetPairBySymbol("BTC/USD")

	// Verify the result
	assert.NoError(t, err)
	assert.NotNil(t, pair)
	assert.Equal(t, "BTC/USD", pair.Symbol)
	assert.Equal(t, "BTC", pair.BaseAsset)
	assert.Equal(t, "USD", pair.QuoteAsset)

	// Get an invalid pair
	pair, err = service.GetPairBySymbol("INVALID")

	// Verify the result
	assert.Error(t, err)
	assert.Nil(t, pair)
	assert.Equal(t, models.ErrPairNotFound, err)

	// Verify that the mock was called
	mockRepo.AssertExpectations(t)
}

