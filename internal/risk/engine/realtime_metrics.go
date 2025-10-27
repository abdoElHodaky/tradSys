package engine

import (
	"sync"
	"time"
)

// RiskMetrics tracks risk engine performance
type RiskMetrics struct {
	ChecksPerSecond     float64       `json:"checks_per_second"`
	AverageLatency      time.Duration `json:"average_latency"`
	MaxLatency          time.Duration `json:"max_latency"`
	TotalChecks         int64         `json:"total_checks"`
	RejectedOrders      int64         `json:"rejected_orders"`
	CircuitBreakerTrips int64         `json:"circuit_breaker_trips"`
	LastUpdateTime      time.Time     `json:"last_update_time"`
}

// PositionManager manages real-time positions
type PositionManager struct {
	positions     sync.Map // map[string]*Position
	totalPnL      float64
	dailyPnL      float64
	unrealizedPnL float64
	realizedPnL   float64
	mu            sync.RWMutex
}

// Position represents a trading position
type Position struct {
	Symbol         string    `json:"symbol"`
	Quantity       float64   `json:"quantity"`
	AveragePrice   float64   `json:"average_price"`
	MarketPrice    float64   `json:"market_price"`
	UnrealizedPnL  float64   `json:"unrealized_pnl"`
	RealizedPnL    float64   `json:"realized_pnl"`
	Delta          float64   `json:"delta"`
	Gamma          float64   `json:"gamma"`
	Vega           float64   `json:"vega"`
	Theta          float64   `json:"theta"`
	LastUpdateTime time.Time `json:"last_update_time"`
}

// LimitManager manages trading limits
type LimitManager struct {
	positionLimits   map[string]float64 // symbol -> max position
	orderLimits      map[string]float64 // symbol -> max order size
	dailyLossLimit   float64
	currentDailyLoss float64
	mu               sync.RWMutex
}

// VaRCalculator calculates Value at Risk
type VaRCalculator struct {
	enabled           bool
	confidenceLevel   float64
	timeHorizon       time.Duration
	historicalReturns map[string][]float64 // symbol -> returns
	correlationMatrix map[string]map[string]float64
	currentVaR        float64
	lastCalculation   time.Time
	mu                sync.RWMutex
}

// CircuitBreaker implements circuit breaker functionality
type CircuitBreaker struct {
	enabled              bool
	volatilityThreshold  float64
	priceChangeThreshold float64
	volumeThreshold      float64
	isTripped            bool
	tripTime             time.Time
	cooldownPeriod       time.Duration
	referencePrice       float64
	mu                   sync.RWMutex
}

