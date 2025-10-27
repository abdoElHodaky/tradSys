package risk

import (
	"context"
	"math"
	"sync"
	"time"

	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// RiskMonitor handles real-time risk monitoring and alerting
type RiskMonitor struct {
	logger         *zap.Logger
	ctx            context.Context
	cancel         context.CancelFunc
	mu             sync.RWMutex
	
	// Market data processing
	marketDataChan chan MarketDataUpdate
	
	// Circuit breakers
	circuitBreakers map[string]*riskengine.CircuitBreaker
	
	// Monitoring state
	isRunning      bool
	lastHealthCheck time.Time
	
	// Caches for performance
	positionCache  *cache.Cache
	metricsCache   *cache.Cache
	
	// Callbacks for alerts
	alertCallbacks []AlertCallback
}

// AlertCallback defines the signature for alert callbacks
type AlertCallback func(alert *RiskAlert)

// RiskAlert represents a risk alert
type RiskAlert struct {
	ID          string                 `json:"id"`
	Type        RiskAlertType          `json:"type"`
	Severity    RiskAlertSeverity      `json:"severity"`
	UserID      string                 `json:"user_id,omitempty"`
	Symbol      string                 `json:"symbol,omitempty"`
	Message     string                 `json:"message"`
	Details     map[string]interface{} `json:"details"`
	Timestamp   time.Time              `json:"timestamp"`
	Acknowledged bool                  `json:"acknowledged"`
}

// RiskAlertType represents the type of risk alert
type RiskAlertType string

const (
	RiskAlertTypePositionLimit     RiskAlertType = "position_limit"
	RiskAlertTypeExposureLimit     RiskAlertType = "exposure_limit"
	RiskAlertTypeDrawdownLimit     RiskAlertType = "drawdown_limit"
	RiskAlertTypeCircuitBreaker    RiskAlertType = "circuit_breaker"
	RiskAlertTypeVolatilitySpike   RiskAlertType = "volatility_spike"
	RiskAlertTypeConcentrationRisk RiskAlertType = "concentration_risk"
	RiskAlertTypeMarginCall        RiskAlertType = "margin_call"
)

// RiskAlertSeverity represents the severity of a risk alert
type RiskAlertSeverity string

const (
	RiskAlertSeverityInfo     RiskAlertSeverity = "info"
	RiskAlertSeverityWarning  RiskAlertSeverity = "warning"
	RiskAlertSeverityCritical RiskAlertSeverity = "critical"
)

// NewRiskMonitor creates a new risk monitor
func NewRiskMonitor(logger *zap.Logger) *RiskMonitor {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RiskMonitor{
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		marketDataChan:  make(chan MarketDataUpdate, 1000),
		circuitBreakers: make(map[string]*riskengine.CircuitBreaker),
		positionCache:   cache.New(5*time.Minute, 10*time.Minute),
		metricsCache:    cache.New(1*time.Minute, 5*time.Minute),
		alertCallbacks:  make([]AlertCallback, 0),
		lastHealthCheck: time.Now(),
	}
}

// Start starts the risk monitor
func (rm *RiskMonitor) Start() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if rm.isRunning {
		return nil
	}
	
	rm.isRunning = true
	rm.logger.Info("Starting risk monitor")
	
	// Start market data processor
	go rm.processMarketData()
	
	// Start circuit breaker checker
	go rm.checkCircuitBreakers()
	
	// Start health monitor
	go rm.healthMonitor()
	
	return nil
}

// Stop stops the risk monitor
func (rm *RiskMonitor) Stop() error {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	if !rm.isRunning {
		return nil
	}
	
	rm.logger.Info("Stopping risk monitor")
	rm.cancel()
	rm.isRunning = false
	
	return nil
}

// AddAlertCallback adds a callback for risk alerts
func (rm *RiskMonitor) AddAlertCallback(callback AlertCallback) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.alertCallbacks = append(rm.alertCallbacks, callback)
}

// UpdateMarketData updates market data for risk monitoring
func (rm *RiskMonitor) UpdateMarketData(update MarketDataUpdate) {
	select {
	case rm.marketDataChan <- update:
	default:
		rm.logger.Warn("Market data channel full, dropping update", 
			zap.String("symbol", update.Symbol))
	}
}

// AddCircuitBreaker adds a circuit breaker for a symbol
func (rm *RiskMonitor) AddCircuitBreaker(symbol string, percentageThreshold float64, cooldownPeriod time.Duration) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	rm.circuitBreakers[symbol] = riskengine.NewCircuitBreaker(percentageThreshold, cooldownPeriod)
	rm.logger.Info("Added circuit breaker", 
		zap.String("symbol", symbol),
		zap.Float64("threshold", percentageThreshold))
}

// RemoveCircuitBreaker removes a circuit breaker for a symbol
func (rm *RiskMonitor) RemoveCircuitBreaker(symbol string) {
	rm.mu.Lock()
	defer rm.mu.Unlock()
	
	delete(rm.circuitBreakers, symbol)
	rm.logger.Info("Removed circuit breaker", zap.String("symbol", symbol))
}

// MonitorPosition monitors a position for risk violations
func (rm *RiskMonitor) MonitorPosition(userID, symbol string, position *riskengine.Position, limits []*RiskLimit) {
	// Check position against limits
	for _, limit := range limits {
		if !limit.IsEnabled() {
			continue
		}
		
		if limit.Symbol != "" && limit.Symbol != symbol {
			continue
		}
		
		violation := rm.checkPositionViolation(position, limit)
		if violation != nil {
			rm.sendAlert(violation)
		}
	}
	
	// Cache position for monitoring
	cacheKey := userID + ":" + symbol
	rm.positionCache.Set(cacheKey, position, cache.DefaultExpiration)
}

// processMarketData processes market data updates
func (rm *RiskMonitor) processMarketData() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case update := <-rm.marketDataChan:
			rm.handleMarketDataUpdate(update)
		case <-ticker.C:
			// Periodic monitoring tasks
			rm.performPeriodicChecks()
		}
	}
}

// handleMarketDataUpdate handles a single market data update
func (rm *RiskMonitor) handleMarketDataUpdate(update MarketDataUpdate) {
	// Update unrealized PnL for positions
	rm.updatePositionPnL(update.Symbol, update.Price)
	
	// Check circuit breakers
	rm.checkCircuitBreaker(update.Symbol, update.Price, update.Timestamp)
	
	// Check for volatility spikes
	rm.checkVolatilitySpike(update)
	
	// Update metrics cache
	rm.updateMetricsCache(update)
}

// updatePositionPnL updates unrealized PnL for all positions in a symbol
func (rm *RiskMonitor) updatePositionPnL(symbol string, price float64) {
	// Get all cached positions for this symbol
	for key, item := range rm.positionCache.Items() {
		if position, ok := item.Object.(*riskengine.Position); ok {
			if position.Symbol == symbol {
				// Update unrealized PnL
				position.UnrealizedPnL = position.Quantity * (price - position.AveragePrice)
				position.LastUpdateTime = time.Now()
				
				// Check for significant PnL changes
				rm.checkPnLAlert(position)
			}
		}
	}
}

// checkCircuitBreaker checks circuit breaker conditions
func (rm *RiskMonitor) checkCircuitBreaker(symbol string, price float64, timestamp time.Time) {
	rm.mu.RLock()
	breaker, exists := rm.circuitBreakers[symbol]
	rm.mu.RUnlock()
	
	if !exists {
		return
	}
	
	// Get previous price from cache
	cacheKey := "price:" + symbol
	var previousPrice float64
	if cached, found := rm.metricsCache.Get(cacheKey); found {
		previousPrice = cached.(float64)
	} else {
		// First price update, store and return
		rm.metricsCache.Set(cacheKey, price, cache.DefaultExpiration)
		return
	}
	
	// Calculate price change percentage
	priceChange := (price - previousPrice) / previousPrice * 100
	
	// Check if circuit breaker should trip
	if breaker.ShouldTrip(priceChange) {
		alert := &RiskAlert{
			ID:        generateAlertID(),
			Type:      RiskAlertTypeCircuitBreaker,
			Severity:  RiskAlertSeverityCritical,
			Symbol:    symbol,
			Message:   "Circuit breaker tripped due to excessive price movement",
			Timestamp: timestamp,
			Details: map[string]interface{}{
				"price_change":  priceChange,
				"current_price": price,
				"previous_price": previousPrice,
				"threshold":     breaker.Threshold,
			},
		}
		rm.sendAlert(alert)
	}
	
	// Update price cache
	rm.metricsCache.Set(cacheKey, price, cache.DefaultExpiration)
}

// checkVolatilitySpike checks for volatility spikes
func (rm *RiskMonitor) checkVolatilitySpike(update MarketDataUpdate) {
	// Simple volatility spike detection
	// In a real implementation, this would use more sophisticated methods
	
	cacheKey := "volatility:" + update.Symbol
	var recentPrices []float64
	
	if cached, found := rm.metricsCache.Get(cacheKey); found {
		recentPrices = cached.([]float64)
	} else {
		recentPrices = make([]float64, 0, 20)
	}
	
	// Add current price
	recentPrices = append(recentPrices, update.Price)
	
	// Keep only last 20 prices
	if len(recentPrices) > 20 {
		recentPrices = recentPrices[1:]
	}
	
	// Calculate volatility if we have enough data
	if len(recentPrices) >= 10 {
		volatility := rm.calculateVolatility(recentPrices)
		
		// Check for spike (simplified threshold)
		if volatility > 0.05 { // 5% volatility threshold
			alert := &RiskAlert{
				ID:        generateAlertID(),
				Type:      RiskAlertTypeVolatilitySpike,
				Severity:  RiskAlertSeverityWarning,
				Symbol:    update.Symbol,
				Message:   "High volatility detected",
				Timestamp: update.Timestamp,
				Details: map[string]interface{}{
					"volatility": volatility,
					"threshold":  0.05,
				},
			}
			rm.sendAlert(alert)
		}
	}
	
	// Update cache
	rm.metricsCache.Set(cacheKey, recentPrices, cache.DefaultExpiration)
}

// checkCircuitBreakers performs periodic circuit breaker checks
func (rm *RiskMonitor) checkCircuitBreakers() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.mu.RLock()
			for symbol, breaker := range rm.circuitBreakers {
				if breaker.IsTripped() && breaker.CanReset() {
					breaker.Reset()
					rm.logger.Info("Circuit breaker reset", zap.String("symbol", symbol))
				}
			}
			rm.mu.RUnlock()
		}
	}
}

// performPeriodicChecks performs periodic monitoring checks
func (rm *RiskMonitor) performPeriodicChecks() {
	// Check for stale positions
	rm.checkStalePositions()
	
	// Check concentration risk
	rm.checkConcentrationRisk()
	
	// Update health check timestamp
	rm.lastHealthCheck = time.Now()
}

// checkStalePositions checks for positions that haven't been updated recently
func (rm *RiskMonitor) checkStalePositions() {
	staleThreshold := 5 * time.Minute
	now := time.Now()
	
	for key, item := range rm.positionCache.Items() {
		if position, ok := item.Object.(*riskengine.Position); ok {
			if now.Sub(position.LastUpdateTime) > staleThreshold {
				rm.logger.Warn("Stale position detected",
					zap.String("user_id", position.UserID),
					zap.String("symbol", position.Symbol),
					zap.Duration("age", now.Sub(position.LastUpdateTime)))
			}
		}
	}
}

// checkConcentrationRisk checks for concentration risk across all positions
func (rm *RiskMonitor) checkConcentrationRisk() {
	// Group positions by user
	userPositions := make(map[string][]*riskengine.Position)
	
	for _, item := range rm.positionCache.Items() {
		if position, ok := item.Object.(*riskengine.Position); ok {
			userPositions[position.UserID] = append(userPositions[position.UserID], position)
		}
	}
	
	// Check concentration for each user
	for userID, positions := range userPositions {
		concentration := rm.calculateConcentrationRisk(positions)
		
		if concentration > 0.4 { // 40% concentration threshold
			alert := &RiskAlert{
				ID:        generateAlertID(),
				Type:      RiskAlertTypeConcentrationRisk,
				Severity:  RiskAlertSeverityWarning,
				UserID:    userID,
				Message:   "High concentration risk detected",
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"concentration": concentration,
					"threshold":     0.4,
					"position_count": len(positions),
				},
			}
			rm.sendAlert(alert)
		}
	}
}

// healthMonitor monitors the health of the risk monitor
func (rm *RiskMonitor) healthMonitor() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-rm.ctx.Done():
			return
		case <-ticker.C:
			rm.performHealthCheck()
		}
	}
}

// performHealthCheck performs a health check
func (rm *RiskMonitor) performHealthCheck() {
	now := time.Now()
	
	// Check if market data is flowing
	if now.Sub(rm.lastHealthCheck) > 2*time.Minute {
		rm.logger.Warn("Risk monitor health check: no recent activity")
	}
	
	// Check cache sizes
	positionCount := rm.positionCache.ItemCount()
	metricsCount := rm.metricsCache.ItemCount()
	
	rm.logger.Debug("Risk monitor health check",
		zap.Int("cached_positions", positionCount),
		zap.Int("cached_metrics", metricsCount),
		zap.Int("circuit_breakers", len(rm.circuitBreakers)))
}

// Helper methods

// checkPositionViolation checks if a position violates a limit
func (rm *RiskMonitor) checkPositionViolation(position *riskengine.Position, limit *RiskLimit) *RiskAlert {
	switch limit.Type {
	case RiskLimitTypePosition:
		if abs(position.Quantity) > limit.Value {
			return &RiskAlert{
				ID:        generateAlertID(),
				Type:      RiskAlertTypePositionLimit,
				Severity:  RiskAlertSeverityWarning,
				UserID:    position.UserID,
				Symbol:    position.Symbol,
				Message:   "Position limit exceeded",
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"position": position.Quantity,
					"limit":    limit.Value,
					"limit_id": limit.ID,
				},
			}
		}
	case RiskLimitTypeDrawdown:
		if position.UnrealizedPnL < -limit.Value {
			return &RiskAlert{
				ID:        generateAlertID(),
				Type:      RiskAlertTypeDrawdownLimit,
				Severity:  RiskAlertSeverityCritical,
				UserID:    position.UserID,
				Symbol:    position.Symbol,
				Message:   "Drawdown limit exceeded",
				Timestamp: time.Now(),
				Details: map[string]interface{}{
					"unrealized_pnl": position.UnrealizedPnL,
					"limit":          limit.Value,
					"limit_id":       limit.ID,
				},
			}
		}
	}
	
	return nil
}

// checkPnLAlert checks for significant PnL changes
func (rm *RiskMonitor) checkPnLAlert(position *riskengine.Position) {
	// Check for significant losses (simplified)
	if position.UnrealizedPnL < -10000 { // $10k loss threshold
		alert := &RiskAlert{
			ID:        generateAlertID(),
			Type:      RiskAlertTypeDrawdownLimit,
			Severity:  RiskAlertSeverityWarning,
			UserID:    position.UserID,
			Symbol:    position.Symbol,
			Message:   "Significant unrealized loss detected",
			Timestamp: time.Now(),
			Details: map[string]interface{}{
				"unrealized_pnl": position.UnrealizedPnL,
			},
		}
		rm.sendAlert(alert)
	}
}

// calculateVolatility calculates volatility from price series
func (rm *RiskMonitor) calculateVolatility(prices []float64) float64 {
	if len(prices) < 2 {
		return 0
	}
	
	// Calculate returns
	returns := make([]float64, len(prices)-1)
	for i := 1; i < len(prices); i++ {
		returns[i-1] = (prices[i] - prices[i-1]) / prices[i-1]
	}
	
	// Calculate mean return
	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))
	
	// Calculate variance
	variance := 0.0
	for _, ret := range returns {
		diff := ret - mean
		variance += diff * diff
	}
	variance /= float64(len(returns) - 1)
	
	// Return standard deviation (volatility)
	return math.Sqrt(variance)
}

// calculateConcentrationRisk calculates concentration risk for positions
func (rm *RiskMonitor) calculateConcentrationRisk(positions []*riskengine.Position) float64 {
	if len(positions) == 0 {
		return 0
	}
	
	totalValue := 0.0
	maxValue := 0.0
	
	for _, pos := range positions {
		value := abs(pos.Quantity * pos.AveragePrice)
		totalValue += value
		if value > maxValue {
			maxValue = value
		}
	}
	
	if totalValue == 0 {
		return 0
	}
	
	return maxValue / totalValue
}

// sendAlert sends a risk alert to all registered callbacks
func (rm *RiskMonitor) sendAlert(alert *RiskAlert) {
	rm.logger.Info("Risk alert generated",
		zap.String("type", string(alert.Type)),
		zap.String("severity", string(alert.Severity)),
		zap.String("message", alert.Message))
	
	// Send to all callbacks
	for _, callback := range rm.alertCallbacks {
		go func(cb AlertCallback) {
			defer func() {
				if r := recover(); r != nil {
					rm.logger.Error("Alert callback panicked", zap.Any("panic", r))
				}
			}()
			cb(alert)
		}(callback)
	}
}

// generateAlertID generates a unique alert ID
func generateAlertID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(6)
}

// randomString generates a random string of specified length
func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[time.Now().UnixNano()%int64(len(charset))]
	}
	return string(b)
}

// abs returns the absolute value of x
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}


