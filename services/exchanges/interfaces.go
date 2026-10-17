// Package exchanges defines common types and interfaces for exchange services
package exchanges

import (
	"context"
	"time"
)

// ExchangeService defines the interface for exchange services
// followed by Interface Segregation Principle
type ExchangeService interface {
	// Core trading operations
	SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error)
	GetMarketData(ctx context.Context, symbol string, assetType AssetType) (*MarketData, error)
	GetAssetInfo(ctx context.Context, symbol string) (*AssetInfo, error)
	GetTradingStatus() *TradingStatus

	// Lifecycle management
	Shutdown(ctx context.Context) error
}

// MarketDataService defines market data operations (Interface Segregation)
type MarketDataService interface {
	GetMarketData(ctx context.Context, symbol string, assetType AssetType) (*MarketData, error)
	Subscribe(ctx context.Context, symbols []string, callback func(*MarketDataUpdate)) error
	StartRealTimeFeeds()
	Stop()
}

// OrderService defines order-related operations
type OrderService interface {
	SubmitOrder(ctx context.Context, order *Order) (*OrderResponse, error)
}

// RiskService defines risk management operations
type RiskService interface {
	AssessOrder(order *Order) error
}

// ComplianceService defines compliance validation
type ComplianceService interface {
	ValidateOrder(order *Order) error
}

// Connector defines exchange connection operations
type Connector interface {
	Connect() error
	Disconnect() error
}

// BaseExchangeService provides common functionality for exchange services
// This follows the Template Method pattern and DRY principle
type BaseExchangeService struct {
	exchangeID         string
	region             string
	assetTypes         []AssetType
	tradingHours       *TradingSchedule
	performanceMonitor *PerformanceMonitor
}

// NewBaseExchangeService creates a base exchange service
func NewBaseExchangeService(
	exchangeID, region string,
	assetTypes []AssetType,
	tradingHours *TradingSchedule,
) *BaseExchangeService {
	return &BaseExchangeService{
		exchangeID:         exchangeID,
		region:             region,
		assetTypes:         assetTypes,
		tradingHours:       tradingHours,
		performanceMonitor: NewPerformanceMonitor(),
	}
}

// IsAssetSupported checks if an asset type is supported
func (s *BaseExchangeService) IsAssetSupported(assetType AssetType) bool {
	return len(s.assetTypes) > 0 && containsAssetType(s.assetTypes, assetType)
}

// IsMarketOpen checks if market is currently open
func (s *BaseExchangeService) IsMarketOpen(now time.Time) bool {
	if s.tradingHours == nil {
		return false
	}
	return now.Hour() >= s.tradingHours.MarketOpen.Hour() &&
		now.Hour() < s.tradingHours.MarketClose.Hour()
}

// GetTradingStatus returns current trading status
func (s *BaseExchangeService) GetTradingStatus(exchangeName string, now time.Time) *TradingStatus {
	return &TradingStatus{
		Exchange:    exchangeName,
		IsOpen:      s.IsMarketOpen(now),
		CurrentTime: now,
		NextOpen:    s.getNextMarketOpen(now),
		NextClose:   s.getNextMarketClose(now),
	}
}

// Helper function following DRY principle (instead of repeating slices.Contains)
func containsAssetType(types []AssetType, t AssetType) bool {
	for _, tt := range types {
		if tt == t {
			return true
		}
	}
	return false
}

// getNextMarketOpen returns next market opening time
func (s *BaseExchangeService) getNextMarketOpen(_ time.Time) time.Time {
	// Default implementation - can be overridden by embedding
	return s.tradingHours.MarketOpen
}

// getNextMarketClose returns next market closing time
func (s *BaseExchangeService) getNextMarketClose(_ time.Time) time.Time {
	return s.tradingHours.MarketClose
}
