package engine

import (
	"context"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// RealTimeRiskEngine provides real-time risk management with HFT performance
type RealTimeRiskEngine struct {
	config          *RiskEngineConfig
	logger          *zap.Logger
	positionManager *PositionManager
	limitManager    *LimitManager
	varCalculator   *VaRCalculator
	circuitBreaker  *CircuitBreaker
	metrics         *RiskMetrics
	eventPool       *pool.ObjectPool
	checkPool       *pool.ObjectPool
	isRunning       int32
	stopChannel     chan struct{}
	eventChannel    chan *RiskEvent
	mu              sync.RWMutex
}

// NewRealTimeRiskEngine creates a new real-time risk engine
func NewRealTimeRiskEngine(config *RiskEngineConfig, logger *zap.Logger) *RealTimeRiskEngine {
	engine := &RealTimeRiskEngine{
		config:          config,
		logger:          logger,
		metrics:         &RiskMetrics{LastUpdateTime: time.Now()},
		stopChannel:     make(chan struct{}),
		eventChannel:    make(chan *RiskEvent, 10000),
		positionManager: &PositionManager{},
		limitManager: &LimitManager{
			positionLimits: make(map[string]float64),
			orderLimits:    make(map[string]float64),
			dailyLossLimit: config.MaxDailyLoss,
		},
		varCalculator: &VaRCalculator{
			enabled:           config.EnableVaRCalculation,
			confidenceLevel:   config.VaRConfidenceLevel,
			timeHorizon:       config.VaRTimeHorizon,
			historicalReturns: make(map[string][]float64),
			correlationMatrix: make(map[string]map[string]float64),
		},
		circuitBreaker: &CircuitBreaker{
			enabled:              config.EnableCircuitBreaker,
			volatilityThreshold:  0.05,    // 5% volatility threshold
			priceChangeThreshold: 0.10,    // 10% price change threshold
			volumeThreshold:      1000000, // Volume threshold
			cooldownPeriod:       time.Minute * 5,
		},
	}

	// Initialize object pools for performance
	engine.eventPool = pool.NewObjectPool(func() interface{} {
		return &RiskEvent{}
	}, 1000)

	engine.checkPool = pool.NewObjectPool(func() interface{} {
		return &RiskCheck{}
	}, 1000)

	return engine
}

// Start starts the risk engine
func (e *RealTimeRiskEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("risk engine is already running")
	}

	e.logger.Info("Starting real-time risk engine",
		zap.Any("config", e.config))

	// Start event processing goroutine
	go e.processEvents(ctx)

	// Start VaR calculation goroutine if enabled
	if e.config.EnableVaRCalculation {
		go e.calculateVaRPeriodically(ctx)
	}

	// Start circuit breaker monitoring if enabled
	if e.config.EnableCircuitBreaker {
		go e.monitorCircuitBreaker(ctx)
	}

	return nil
}

// Stop stops the risk engine
func (e *RealTimeRiskEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return fmt.Errorf("risk engine is not running")
	}

	close(e.stopChannel)
	e.logger.Info("Real-time risk engine stopped")
	return nil
}

// PreTradeCheck performs pre-trade risk checks with HFT latency requirements
func (e *RealTimeRiskEngine) PreTradeCheck(order *types.Order) (*RiskCheck, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)

		// Log if latency exceeds target
		if latency > e.config.MaxLatency {
			e.logger.Warn("Pre-trade check exceeded latency target",
				zap.Duration("latency", latency),
				zap.Duration("target", e.config.MaxLatency),
				zap.String("order_id", order.ID))
		}
	}()

	if !e.config.EnablePreTradeChecks {
		return &RiskCheck{
			CheckType: "pre_trade",
			Passed:    true,
			Message:   "Pre-trade checks disabled",
			Latency:   time.Since(startTime),
			Timestamp: time.Now(),
		}, nil
	}

	// Check circuit breaker first (fastest check)
	if e.circuitBreaker.isTripped {
		return &RiskCheck{
			CheckType: "circuit_breaker",
			Passed:    false,
			Message:   "Circuit breaker is active",
			Latency:   time.Since(startTime),
			Timestamp: time.Now(),
		}, fmt.Errorf("circuit breaker is active")
	}

	// Check order size limits
	if err := e.checkOrderSizeLimits(order); err != nil {
		return &RiskCheck{
			CheckType:    "order_size",
			Passed:       false,
			CurrentValue: order.Quantity,
			LimitValue:   e.limitManager.orderLimits[order.Symbol],
			Message:      err.Error(),
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		}, err
	}

	// Check position limits
	if err := e.checkPositionLimits(order); err != nil {
		return &RiskCheck{
			CheckType: "position_limit",
			Passed:    false,
			Message:   err.Error(),
			Latency:   time.Since(startTime),
			Timestamp: time.Now(),
		}, err
	}

	// Check daily loss limits
	if err := e.checkDailyLossLimits(order); err != nil {
		return &RiskCheck{
			CheckType:    "daily_loss",
			Passed:       false,
			CurrentValue: e.limitManager.currentDailyLoss,
			LimitValue:   e.limitManager.dailyLossLimit,
			Message:      err.Error(),
			Latency:      time.Since(startTime),
			Timestamp:    time.Now(),
		}, err
	}

	// All checks passed
	atomic.AddInt64(&e.metrics.TotalChecks, 1)
	return &RiskCheck{
		CheckType: "pre_trade",
		Passed:    true,
		Message:   "All pre-trade checks passed",
		Latency:   time.Since(startTime),
		Timestamp: time.Now(),
	}, nil
}

// PostTradeCheck performs post-trade risk checks
func (e *RealTimeRiskEngine) PostTradeCheck(order *types.Order, trades []*Trade) error {
	if !e.config.EnablePostTradeChecks {
		return nil
	}

	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)
	}()

	// Update positions
	for _, trade := range trades {
		e.updatePosition(trade)
	}

	// Check if any limits were breached after the trade
	if err := e.checkPostTradeLimits(order); err != nil {
		e.publishEvent(&RiskEvent{
			Type:      EventLimitBreach,
			Symbol:    order.Symbol,
			Data:      order,
			Severity:  SeverityError,
			Message:   err.Error(),
			Timestamp: time.Now(),
		})
		return err
	}

	return nil
}

// checkOrderSizeLimits checks order size limits
func (e *RealTimeRiskEngine) checkOrderSizeLimits(order *types.Order) error {
	e.limitManager.mu.RLock()
	limit, exists := e.limitManager.orderLimits[order.Symbol]
	e.limitManager.mu.RUnlock()

	if !exists {
		limit = e.config.MaxOrderSize
	}

	if order.Quantity > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("order size %f exceeds limit %f for symbol %s",
			order.Quantity, limit, order.Symbol)
	}

	return nil
}

// checkPositionLimits checks position limits
func (e *RealTimeRiskEngine) checkPositionLimits(order *types.Order) error {
	// Get current position
	position := e.getPosition(order.Symbol)

	// Calculate new position after order
	var newQuantity float64
	if order.Side == types.OrderSideBuy {
		newQuantity = position.Quantity + order.Quantity
	} else {
		newQuantity = position.Quantity - order.Quantity
	}

	// Check against position limits
	e.limitManager.mu.RLock()
	limit, exists := e.limitManager.positionLimits[order.Symbol]
	e.limitManager.mu.RUnlock()

	if !exists {
		limit = e.config.MaxPositionSize
	}

	if math.Abs(newQuantity) > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("position size %f would exceed limit %f for symbol %s",
			math.Abs(newQuantity), limit, order.Symbol)
	}

	return nil
}

// checkDailyLossLimits checks daily loss limits
func (e *RealTimeRiskEngine) checkDailyLossLimits(order *types.Order) error {
	e.limitManager.mu.RLock()
	currentLoss := e.limitManager.currentDailyLoss
	limit := e.limitManager.dailyLossLimit
	e.limitManager.mu.RUnlock()

	if currentLoss > limit {
		atomic.AddInt64(&e.metrics.RejectedOrders, 1)
		return fmt.Errorf("daily loss %f exceeds limit %f", currentLoss, limit)
	}

	return nil
}

// checkPostTradeLimits checks limits after trade execution
func (e *RealTimeRiskEngine) checkPostTradeLimits(order *types.Order) error {
	// This would include more sophisticated checks after trade execution
	// For now, just return nil
	return nil
}

// getPosition gets the current position for a symbol
func (e *RealTimeRiskEngine) getPosition(symbol string) *Position {
	if pos, exists := e.positionManager.positions.Load(symbol); exists {
		return pos.(*Position)
	}

	// Return empty position if not found
	return &Position{
		Symbol:         symbol,
		Quantity:       0,
		AveragePrice:   0,
		MarketPrice:    0,
		LastUpdateTime: time.Now(),
	}
}

// updatePosition updates position after a trade
func (e *RealTimeRiskEngine) updatePosition(trade *Trade) {
	// This is a simplified implementation
	// In practice, this would be more complex with proper P&L calculation
	position := e.getPosition(trade.Symbol)

	// Update position quantity and average price
	// This is a basic implementation - real systems would be more sophisticated
	if trade.TakerSide == types.OrderSideBuy {
		position.Quantity += trade.Quantity
	} else {
		position.Quantity -= trade.Quantity
	}

	position.LastUpdateTime = time.Now()
	e.positionManager.positions.Store(trade.Symbol, position)
}

// updateMetrics updates risk engine metrics
func (e *RealTimeRiskEngine) updateMetrics(latency time.Duration) {
	if latency > e.metrics.MaxLatency {
		e.metrics.MaxLatency = latency
	}

	// Simple moving average
	e.metrics.AverageLatency = (e.metrics.AverageLatency + latency) / 2
	e.metrics.LastUpdateTime = time.Now()
}
