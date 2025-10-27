package risk

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/abdoElHodaky/tradSys/pkg/matching"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Note: Type definitions moved to risk_types.go to avoid duplicates

// RiskLimit represents a risk limit
type RiskLimit struct {
	// ID is the unique identifier for the risk limit
	ID string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Type is the type of risk limit
	Type RiskLimitType
	// Value is the limit value
	Value float64
	// CreatedAt is the time the risk limit was created
	CreatedAt time.Time
	// UpdatedAt is the time the risk limit was last updated
	UpdatedAt time.Time
	// Enabled indicates whether the risk limit is enabled
	Enabled bool
}

// RiskOperation represents a batch operation on risk data
type RiskOperation struct {
	// OpType is the operation type
	OpType string
	// UserID is the user ID
	UserID string
	// Symbol is the trading symbol
	Symbol string
	// Data is the operation data
	Data interface{}
	// ResultCh is the result channel
	ResultCh chan RiskOperationResult
}

// RiskOperationResult represents the result of a risk operation
type RiskOperationResult struct {
	// Success indicates whether the operation was successful
	Success bool
	// Error is the error if the operation failed
	Error error
	// Data is the result data
	Data interface{}
}

// Service represents a risk management service
type Service struct {
	// OrderEngine is the order matching engine
	OrderEngine *matching.Engine
	// OrderService is the order management service
	OrderService *orders.Service
	// Positions is a map of user ID and symbol to position
	Positions map[string]map[string]*riskengine.Position
	// RiskLimits is a map of user ID to risk limits
	RiskLimits map[string][]*RiskLimit
	// CircuitBreakers is a map of symbol to circuit breaker
	CircuitBreakers map[string]*riskengine.CircuitBreaker
	// PositionCache is a cache for frequently accessed positions
	PositionCache *cache.Cache
	// RiskLimitCache is a cache for frequently accessed risk limits
	RiskLimitCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Batch processing channel for risk operations
	riskBatchChan chan RiskOperation
	// Market data channel for price updates
	marketDataChan chan MarketDataUpdate
}

// MarketDataUpdate represents a market data update
type MarketDataUpdate struct {
	// Symbol is the trading symbol
	Symbol string
	// Price is the current price
	Price float64
	// Timestamp is the time of the update
	Timestamp time.Time
}

// Utility functions
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
