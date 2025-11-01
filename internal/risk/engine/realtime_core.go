// 🎯 **Risk Engine Core Service**
// Generated using TradSys Code Splitting Standards
//
// This file contains the main service struct, constructor, and core API methods
// for the RealTime Risk Engine component. It follows the established patterns for
// service initialization, lifecycle management, and primary business operations.
//
// File size limit: 350 lines

package risk_management

import (
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/common/pool"
	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"go.uber.org/zap"
)

// NewRealTimeRiskEngine creates a new real-time risk engine
func NewRealTimeRiskEngine(config *RiskEngineConfig, logger *zap.Logger) *RealTimeRiskEngine {
	if config == nil {
		config = &RiskEngineConfig{
			MaxLatency:            time.Microsecond * 10, // 10μs for HFT
			EnablePreTradeChecks:  true,
			EnablePostTradeChecks: true,
			EnableVaRCalculation:  true,
			EnableCircuitBreaker:  true,
			VaRConfidenceLevel:    0.95, // 95%
			VaRTimeHorizon:        24 * time.Hour,
			MaxPositionSize:       1000000,
			MaxOrderSize:          100000,
			MaxDailyLoss:          50000,
			StressTestEnabled:     false,
		}
	}

	engine := &RealTimeRiskEngine{
		config:      config,
		logger:      logger,
		metrics:     &RiskMetrics{LastUpdateTime: time.Now()},
		stopChannel: make(chan struct{}),
		eventChannel: make(chan *RiskEvent, 10000), // Large buffer for HFT
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
			Symbol:              "DEFAULT",
			PercentageThreshold: 0.10,
			TimeWindow:          time.Minute,
			CooldownPeriod:      5 * time.Minute,
			CreatedAt:           time.Now(),
		},
	}

	// Initialize object pools for performance
	engine.eventPool = pool.NewObjectPool(func() interface{} {
		return &RiskEvent{}
	}, 1000)

	engine.checkPool = pool.NewObjectPool(func() interface{} {
		return &RiskCheckResponse{}
	}, 1000)

	return engine
}

// Start starts the real-time risk engine
func (e *RealTimeRiskEngine) Start() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return errors.New("risk engine is already running")
	}

	e.logger.Info("Starting real-time risk engine",
		zap.Any("config", e.config))

	// Start event processing goroutine
	go e.processEvents()

	// Start VaR calculation if enabled
	if e.config.EnableVaRCalculation {
		go e.calculateVaRPeriodically()
	}

	// Start circuit breaker monitoring if enabled
	if e.config.EnableCircuitBreaker {
		go e.monitorCircuitBreaker()
	}

	return nil
}

// Stop stops the real-time risk engine
func (e *RealTimeRiskEngine) Stop() error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return errors.New("risk engine is not running")
	}

	close(e.stopChannel)

	e.logger.Info("Real-time risk engine stopped")
	return nil
}

// PerformRiskCheck performs a risk check with HFT latency requirements
func (e *RealTimeRiskEngine) PerformRiskCheck(req *RiskCheckRequest) (*RiskCheckResponse, error) {
	startTime := time.Now()
	defer func() {
		latency := time.Since(startTime)
		e.updateMetrics(latency)

		// Log if latency exceeds target
		if latency > e.config.MaxLatency {
			e.logger.Warn("Risk check exceeded latency target",
				zap.Duration("latency", latency),
				zap.Duration("target", e.config.MaxLatency),
				zap.String("check_id", req.ID))
		}
	}()

	// Early return for nil request
	if req == nil {
		return nil, errors.New("risk check request cannot be nil")
	}

	// Validate request using early return pattern
	if err := e.validateRiskCheckRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Get response from pool for performance
	response := e.checkPool.Get().(*RiskCheckResponse)
	defer e.checkPool.Put(response)

	// Reset response
	*response = RiskCheckResponse{
		ID:        req.ID,
		Timestamp: time.Now(),
	}

	// Perform type-specific risk check
	switch req.Type {
	case RiskEventTypePreTrade:
		return e.performPreTradeCheck(req, response)
	case RiskEventTypePostTrade:
		return e.performPostTradeCheck(req, response)
	case RiskEventTypePositionRisk:
		return e.performPositionRiskCheck(req, response)
	case RiskEventTypeVaRUpdate:
		return e.performVaRCheck(req, response)
	default:
		return nil, fmt.Errorf("unsupported risk check type: %s", req.Type)
	}
}

// CheckPreTrade performs pre-trade risk checks
func (e *RealTimeRiskEngine) CheckPreTrade(order *types.Order) (*RiskCheckResponse, error) {
	// Early return for nil order
	if order == nil {
		return nil, errors.New("order cannot be nil")
	}

	req := &RiskCheckRequest{
		ID:        order.ID,
		Type:      RiskEventTypePreTrade,
		Symbol:    order.Symbol,
		Order:     order,
		Timestamp: time.Now(),
	}

	return e.PerformRiskCheck(req)
}

// CheckPostTrade performs post-trade risk checks
func (e *RealTimeRiskEngine) CheckPostTrade(order *types.Order) (*RiskCheckResponse, error) {
	// Early return for nil order
	if order == nil {
		return nil, errors.New("order cannot be nil")
	}

	req := &RiskCheckRequest{
		ID:        order.ID,
		Type:      RiskEventTypePostTrade,
		Symbol:    order.Symbol,
		Order:     order,
		Timestamp: time.Now(),
	}

	return e.PerformRiskCheck(req)
}

// GetMetrics returns current risk engine metrics
func (e *RealTimeRiskEngine) GetMetrics() *RiskMetrics {
	e.mu.RLock()
	defer e.mu.RUnlock()

	// Return a copy to prevent race conditions
	return &RiskMetrics{
		ChecksPerSecond:     e.metrics.ChecksPerSecond,
		AverageLatency:      e.metrics.AverageLatency,
		MaxLatency:          e.metrics.MaxLatency,
		TotalChecks:         e.metrics.TotalChecks,
		RejectedOrders:      e.metrics.RejectedOrders,
		CircuitBreakerTrips: e.metrics.CircuitBreakerTrips,
		LastUpdateTime:      e.metrics.LastUpdateTime,
	}
}

// GetCurrentVaR returns the current Value at Risk
func (e *RealTimeRiskEngine) GetCurrentVaR() float64 {
	e.varCalculator.mu.RLock()
	defer e.varCalculator.mu.RUnlock()
	return e.varCalculator.currentVaR
}

// IsCircuitBreakerTripped checks if circuit breaker is tripped
func (e *RealTimeRiskEngine) IsCircuitBreakerTripped(symbol string) bool {
	// For now, check the default circuit breaker
	// In a full implementation, this would check symbol-specific breakers
	return e.circuitBreaker.IsTriggeredFlag
}

// processEvents processes risk events asynchronously
func (e *RealTimeRiskEngine) processEvents() {
	for {
		select {
		case event := <-e.eventChannel:
			e.handleRiskEvent(event)
		case <-e.stopChannel:
			return
		}
	}
}

// handleRiskEvent handles a risk event
func (e *RealTimeRiskEngine) handleRiskEvent(event *RiskEvent) {
	switch event.Type {
	case RiskEventTypeCircuitBreak:
		e.handleCircuitBreakerEvent(event)
	case RiskEventTypeLimitBreach:
		e.handleLimitBreachEvent(event)
	case RiskEventTypeVaRUpdate:
		e.handleVaRUpdateEvent(event)
	default:
		e.logger.Debug("Unhandled risk event type",
			zap.String("type", string(event.Type)),
			zap.String("symbol", event.Symbol))
	}
}

// calculateVaRPeriodically calculates VaR periodically
func (e *RealTimeRiskEngine) calculateVaRPeriodically() {
	ticker := time.NewTicker(time.Hour) // Calculate VaR every hour
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if e.config.EnableVaRCalculation {
				e.calculateVaR()
			}
		case <-e.stopChannel:
			return
		}
	}
}

// monitorCircuitBreaker monitors circuit breaker conditions
func (e *RealTimeRiskEngine) monitorCircuitBreaker() {
	ticker := time.NewTicker(time.Second) // Check every second
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if e.config.EnableCircuitBreaker {
				e.checkCircuitBreakerConditions()
			}
		case <-e.stopChannel:
			return
		}
	}
}

// updateMetrics updates engine metrics
func (e *RealTimeRiskEngine) updateMetrics(latency time.Duration) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.metrics.TotalChecks++
	e.metrics.AverageLatency = (e.metrics.AverageLatency + latency) / 2

	if latency > e.metrics.MaxLatency {
		e.metrics.MaxLatency = latency
	}

	// Calculate checks per second
	now := time.Now()
	if !e.metrics.LastUpdateTime.IsZero() {
		duration := now.Sub(e.metrics.LastUpdateTime).Seconds()
		if duration > 0 {
			e.metrics.ChecksPerSecond = 1.0 / duration
		}
	}
	e.metrics.LastUpdateTime = now
}

// validateRiskCheckRequest validates a risk check request
func (e *RealTimeRiskEngine) validateRiskCheckRequest(req *RiskCheckRequest) error {
	// Early return for nil request
	if req == nil {
		return errors.New("request cannot be nil")
	}

	if req.ID == "" {
		return errors.New("request ID is required")
	}

	if req.Symbol == "" {
		return errors.New("symbol is required")
	}

	// Validate timestamp
	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now()
	}

	// Check if timestamp is not too old (for HFT, 1 minute is old)
	if time.Since(req.Timestamp) > time.Minute {
		return errors.New("request timestamp is too old")
	}

	return nil
}
