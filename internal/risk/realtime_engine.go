package risk

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

// RiskEngineConfig contains configuration for the risk engine
type RiskEngineConfig struct {
	MaxLatency            time.Duration `json:"max_latency"` // Target: <10μs
	EnablePreTradeChecks  bool          `json:"enable_pre_trade_checks"`
	EnablePostTradeChecks bool          `json:"enable_post_trade_checks"`
	EnableVaRCalculation  bool          `json:"enable_var_calculation"`
	EnableCircuitBreaker  bool          `json:"enable_circuit_breaker"`
	VaRConfidenceLevel    float64       `json:"var_confidence_level"` // 95%, 99%
	VaRTimeHorizon        time.Duration `json:"var_time_horizon"`     // 1 day
	MaxPositionSize       float64       `json:"max_position_size"`
	MaxOrderSize          float64       `json:"max_order_size"`
	MaxDailyLoss          float64       `json:"max_daily_loss"`
	StressTestEnabled     bool          `json:"stress_test_enabled"`
}

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
	positions     sync.Map // map[string]*RealtimePosition
	totalPnL      float64
	dailyPnL      float64
	unrealizedPnL float64
	realizedPnL   float64
	mu            sync.RWMutex
}

// RealtimePosition represents a trading position for real-time risk monitoring
type RealtimePosition struct {
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
	mu                   sync.RWMutex
}

// RiskEvent represents a risk management event
type RiskEvent struct {
	Type      RiskEventType `json:"type"`
	Symbol    string        `json:"symbol"`
	Order     *types.Order  `json:"order,omitempty"`
	Position  *RealtimePosition     `json:"position,omitempty"`
	RiskCheck *RiskCheck    `json:"risk_check,omitempty"`
	Timestamp time.Time     `json:"timestamp"`
	Severity  RiskSeverity  `json:"severity"`
	Message   string        `json:"message"`
}

// RiskEventType defines types of risk events
type RiskEventType string

const (
	EventPreTradeCheck  RiskEventType = "pre_trade_check"
	EventPostTradeCheck RiskEventType = "post_trade_check"
	EventLimitBreach    RiskEventType = "limit_breach"
	EventCircuitBreaker RiskEventType = "circuit_breaker"
	EventVaRCalculation RiskEventType = "var_calculation"
	EventPositionUpdate RiskEventType = "position_update"
)

// RiskSeverity defines risk event severity levels
type RiskSeverity string

const (
	SeverityInfo     RiskSeverity = "info"
	SeverityWarning  RiskSeverity = "warning"
	SeverityError    RiskSeverity = "error"
	SeverityCritical RiskSeverity = "critical"
)

// RiskCheck represents a risk check result
type RiskCheck struct {
	CheckType    string        `json:"check_type"`
	Passed       bool          `json:"passed"`
	CurrentValue float64       `json:"current_value"`
	LimitValue   float64       `json:"limit_value"`
	Message      string        `json:"message"`
	Latency      time.Duration `json:"latency"`
	Timestamp    time.Time     `json:"timestamp"`
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
			Order:     order,
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
func (e *RealTimeRiskEngine) getPosition(symbol string) *RealtimePosition {
	if pos, exists := e.positionManager.positions.Load(symbol); exists {
		return pos.(*RealtimePosition)
	}

	// Return empty position if not found
	return &RealtimePosition{
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

// publishEvent publishes a risk event
func (e *RealTimeRiskEngine) publishEvent(event *RiskEvent) {
	select {
	case e.eventChannel <- event:
	default:
		e.logger.Warn("Risk event channel full, dropping event",
			zap.String("event_type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// processEvents processes risk events
func (e *RealTimeRiskEngine) processEvents(ctx context.Context) {
	for {
		select {
		case event := <-e.eventChannel:
			e.handleRiskEvent(event)
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// handleRiskEvent handles a risk event
func (e *RealTimeRiskEngine) handleRiskEvent(event *RiskEvent) {
	switch event.Type {
	case EventLimitBreach:
		e.logger.Error("Risk limit breach",
			zap.String("symbol", event.Symbol),
			zap.String("message", event.Message))
	case EventCircuitBreaker:
		e.logger.Warn("Circuit breaker event",
			zap.String("symbol", event.Symbol),
			zap.String("message", event.Message))
	default:
		e.logger.Info("Risk event",
			zap.String("type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// calculateVaRPeriodically calculates VaR periodically
func (e *RealTimeRiskEngine) calculateVaRPeriodically(ctx context.Context) {
	ticker := time.NewTicker(time.Minute * 5) // Calculate VaR every 5 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.calculateVaR()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// calculateVaR calculates Value at Risk
func (e *RealTimeRiskEngine) calculateVaR() {
	// This is a simplified VaR calculation
	// Real implementations would use more sophisticated models
	e.varCalculator.mu.Lock()
	defer e.varCalculator.mu.Unlock()

	// Calculate portfolio VaR using historical simulation
	// This is a placeholder implementation
	e.logger.Info("VaR calculation completed")
}

// monitorCircuitBreaker monitors circuit breaker conditions
func (e *RealTimeRiskEngine) monitorCircuitBreaker(ctx context.Context) {
	ticker := time.NewTicker(time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			e.checkCircuitBreakerConditions()
		case <-ctx.Done():
			return
		case <-e.stopChannel:
			return
		}
	}
}

// checkCircuitBreakerConditions checks if circuit breaker should be triggered
func (e *RealTimeRiskEngine) checkCircuitBreakerConditions() {
	// This is a simplified implementation
	// Real systems would monitor market conditions and trigger based on volatility, etc.

	e.circuitBreaker.mu.Lock()
	defer e.circuitBreaker.mu.Unlock()

	// Check if circuit breaker should be reset
	if e.circuitBreaker.isTripped &&
		time.Since(e.circuitBreaker.tripTime) > e.circuitBreaker.cooldownPeriod {
		e.circuitBreaker.isTripped = false
		e.logger.Info("Circuit breaker reset after cooldown period")
	}
}

// GetMetrics returns current risk metrics
func (e *RealTimeRiskEngine) GetMetrics() *RiskMetrics {
	return e.metrics
}

// GetPosition returns the current position for a symbol
func (e *RealTimeRiskEngine) GetPosition(symbol string) *RealtimePosition {
	return e.getPosition(symbol)
}

// Trade represents a trade (imported from order matching)
type Trade struct {
	ID        string
	Symbol    string
	Price     float64
	Quantity  float64
	TakerSide types.OrderSide
	Timestamp time.Time
}
