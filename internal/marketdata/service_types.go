package marketdata

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/abdoElHodaky/tradSys/internal/marketdata/external"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Service represents a market data service
type Service struct {
	// MarketDataRepository is the market data repository
	MarketDataRepository *repositories.MarketDataRepository
	// ExternalManager is the external market data provider manager
	ExternalManager *external.Manager
	// Cache is a cache for market data
	Cache *cache.Cache
	// Subscriptions is a map of subscription ID to subscription
	Subscriptions map[string]*Subscription
	// SymbolSubscriptions is a map of symbol to subscriptions
	SymbolSubscriptions map[string]map[string]*Subscription
	// Logger
	logger *zap.Logger
	// Config
	config *config.Config
	// Mutex for thread safety
	mu sync.RWMutex
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
}

// Subscription represents a market data subscription
type Subscription struct {
	// ID is the unique identifier for the subscription
	ID string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Type is the type of market data
	Type external.MarketDataType
	// Interval is the interval for OHLCV data
	Interval string
	// Channel is the channel for sending market data
	Channel chan interface{}
	// CreatedAt is the time the subscription was created
	CreatedAt time.Time
}
