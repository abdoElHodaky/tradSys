package risk

import (
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed    CircuitBreakerState = "closed"    // Normal operation
	CircuitBreakerOpen      CircuitBreakerState = "open"      // Trading halted
	CircuitBreakerHalfOpen  CircuitBreakerState = "half_open" // Testing recovery
)

// HaltReason represents the reason for a trading halt
type HaltReason string

const (
	HaltReasonVolatility     HaltReason = "volatility"
	HaltReasonVolume         HaltReason = "volume"
	HaltReasonPriceMove      HaltReason = "price_move"
	HaltReasonRiskLimit      HaltReason = "risk_limit"
	HaltReasonSystemError    HaltReason = "system_error"
	HaltReasonManual         HaltReason = "manual"
	HaltReasonRegulatory     HaltReason = "regulatory"
)

// CircuitBreakerConfig represents configuration for a circuit breaker
type CircuitBreakerConfig struct {
	Symbol                string        `json:"symbol"`
	MaxVolatility         float64       `json:"max_volatility"`          // Maximum allowed volatility
	MaxPriceMove          float64       `json:"max_price_move"`          // Maximum price move percentage
	MaxVolumeSpike        float64       `json:"max_volume_spike"`        // Maximum volume spike multiplier
	MinRecoveryTime       time.Duration `json:"min_recovery_time"`       // Minimum time before recovery attempt
	MaxRecoveryTime       time.Duration `json:"max_recovery_time"`       // Maximum time in open state
	VolatilityWindow      time.Duration `json:"volatility_window"`       // Time window for volatility calculation
	PriceMoveWindow       time.Duration `json:"price_move_window"`       // Time window for price move calculation
	VolumeWindow          time.Duration `json:"volume_window"`           // Time window for volume calculation
	RecoveryTestOrders    int           `json:"recovery_test_orders"`    // Number of test orders in half-open state
	Enabled               bool          `json:"enabled"`
}

// CircuitBreakerStatus represents the current status of a circuit breaker
type CircuitBreakerStatus struct {
	Symbol           string              `json:"symbol"`
	State            CircuitBreakerState `json:"state"`
	HaltReason       HaltReason          `json:"halt_reason,omitempty"`
	HaltedAt         *time.Time          `json:"halted_at,omitempty"`
	ResumedAt        *time.Time          `json:"resumed_at,omitempty"`
	HaltCount        int                 `json:"halt_count"`
	LastHaltDuration time.Duration       `json:"last_halt_duration"`
	CurrentVolatility float64            `json:"current_volatility"`
	CurrentPriceMove  float64            `json:"current_price_move"`
	CurrentVolumeSpike float64           `json:"current_volume_spike"`
	TestOrderCount    int                `json:"test_order_count"`
}

// PriceData represents price data for circuit breaker calculations
type PriceData struct {
	Symbol    string    `json:"symbol"`
	Price     float64   `json:"price"`
	Volume    float64   `json:"volume"`
	Timestamp time.Time `json:"timestamp"`
}

// CircuitBreakerSystem manages circuit breakers for trading halts
type CircuitBreakerSystem struct {
	breakers    map[string]*CircuitBreakerStatus
	configs     map[string]*CircuitBreakerConfig
	priceData   map[string][]*PriceData // symbol -> price history
	logger      *zap.Logger
	mu          sync.RWMutex
	
	// Performance metrics
	haltCount       int64
	resumeCount     int64
	avgHaltDuration time.Duration
	
	// Global circuit breaker
	globalHalt      bool
	globalHaltTime  *time.Time
	globalHaltReason HaltReason
}

// NewCircuitBreakerSystem creates a new circuit breaker system
func NewCircuitBreakerSystem(logger *zap.Logger) *CircuitBreakerSystem {
	return &CircuitBreakerSystem{
		breakers:  make(map[string]*CircuitBreakerStatus),
		configs:   make(map[string]*CircuitBreakerConfig),
		priceData: make(map[string][]*PriceData),
		logger:    logger,
	}
}

// AddCircuitBreaker adds a circuit breaker for a symbol
func (cbs *CircuitBreakerSystem) AddCircuitBreaker(symbol string, config *CircuitBreakerConfig) {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	config.Symbol = symbol
	cbs.configs[symbol] = config
	cbs.breakers[symbol] = &CircuitBreakerStatus{
		Symbol: symbol,
		State:  CircuitBreakerClosed,
	}
	cbs.priceData[symbol] = make([]*PriceData, 0)

	cbs.logger.Info("Circuit breaker added",
		zap.String("symbol", symbol),
		zap.Float64("max_volatility", config.MaxVolatility),
		zap.Float64("max_price_move", config.MaxPriceMove),
	)
}

// UpdatePriceData updates price data and checks for circuit breaker triggers
func (cbs *CircuitBreakerSystem) UpdatePriceData(data *PriceData) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	// Add price data
	if cbs.priceData[data.Symbol] == nil {
		cbs.priceData[data.Symbol] = make([]*PriceData, 0)
	}
	cbs.priceData[data.Symbol] = append(cbs.priceData[data.Symbol], data)

	// Clean old data
	cbs.cleanOldPriceData(data.Symbol)

	// Check circuit breaker conditions
	if config, exists := cbs.configs[data.Symbol]; exists && config.Enabled {
		if breaker, exists := cbs.breakers[data.Symbol]; exists {
			return cbs.checkCircuitBreakerConditions(data.Symbol, config, breaker)
		}
	}

	return nil
}

// cleanOldPriceData removes old price data outside the calculation windows
func (cbs *CircuitBreakerSystem) cleanOldPriceData(symbol string) {
	config, exists := cbs.configs[symbol]
	if !exists {
		return
	}

	// Determine the maximum window we need to keep data for
	maxWindow := config.VolatilityWindow
	if config.PriceMoveWindow > maxWindow {
		maxWindow = config.PriceMoveWindow
	}
	if config.VolumeWindow > maxWindow {
		maxWindow = config.VolumeWindow
	}

	cutoff := time.Now().Add(-maxWindow)
	priceHistory := cbs.priceData[symbol]
	
	// Find the first index to keep
	keepIndex := 0
	for i, data := range priceHistory {
		if data.Timestamp.After(cutoff) {
			keepIndex = i
			break
		}
	}

	// Keep only recent data
	if keepIndex > 0 {
		cbs.priceData[symbol] = priceHistory[keepIndex:]
	}
}

// checkCircuitBreakerConditions checks if circuit breaker should trigger
func (cbs *CircuitBreakerSystem) checkCircuitBreakerConditions(symbol string, config *CircuitBreakerConfig, breaker *CircuitBreakerStatus) error {
	now := time.Now()
	priceHistory := cbs.priceData[symbol]

	if len(priceHistory) < 2 {
		return nil // Need at least 2 data points
	}

	// Check volatility
	volatility := cbs.calculateVolatility(symbol, config.VolatilityWindow)
	breaker.CurrentVolatility = volatility
	if volatility > config.MaxVolatility {
		return cbs.triggerCircuitBreaker(symbol, HaltReasonVolatility, 
			fmt.Sprintf("Volatility %.4f exceeds limit %.4f", volatility, config.MaxVolatility))
	}

	// Check price movement
	priceMove := cbs.calculatePriceMove(symbol, config.PriceMoveWindow)
	breaker.CurrentPriceMove = priceMove
	if priceMove > config.MaxPriceMove {
		return cbs.triggerCircuitBreaker(symbol, HaltReasonPriceMove,
			fmt.Sprintf("Price move %.2f%% exceeds limit %.2f%%", priceMove*100, config.MaxPriceMove*100))
	}

	// Check volume spike
	volumeSpike := cbs.calculateVolumeSpike(symbol, config.VolumeWindow)
	breaker.CurrentVolumeSpike = volumeSpike
	if volumeSpike > config.MaxVolumeSpike {
		return cbs.triggerCircuitBreaker(symbol, HaltReasonVolume,
			fmt.Sprintf("Volume spike %.2fx exceeds limit %.2fx", volumeSpike, config.MaxVolumeSpike))
	}

	// Check if breaker should recover
	if breaker.State == CircuitBreakerOpen && breaker.HaltedAt != nil {
		if now.Sub(*breaker.HaltedAt) >= config.MinRecoveryTime {
			cbs.attemptRecovery(symbol, config, breaker)
		}
	}

	return nil
}

// calculateVolatility calculates volatility over a time window
func (cbs *CircuitBreakerSystem) calculateVolatility(symbol string, window time.Duration) float64 {
	priceHistory := cbs.priceData[symbol]
	cutoff := time.Now().Add(-window)

	var prices []float64
	for _, data := range priceHistory {
		if data.Timestamp.After(cutoff) {
			prices = append(prices, data.Price)
		}
	}

	if len(prices) < 2 {
		return 0
	}

	// Calculate returns
	var returns []float64
	for i := 1; i < len(prices); i++ {
		ret := (prices[i] - prices[i-1]) / prices[i-1]
		returns = append(returns, ret)
	}

	// Calculate standard deviation of returns
	if len(returns) == 0 {
		return 0
	}

	mean := 0.0
	for _, ret := range returns {
		mean += ret
	}
	mean /= float64(len(returns))

	variance := 0.0
	for _, ret := range returns {
		variance += (ret - mean) * (ret - mean)
	}
	variance /= float64(len(returns))

	return variance // Return variance as volatility measure
}

// calculatePriceMove calculates maximum price movement over a time window
func (cbs *CircuitBreakerSystem) calculatePriceMove(symbol string, window time.Duration) float64 {
	priceHistory := cbs.priceData[symbol]
	cutoff := time.Now().Add(-window)

	var prices []float64
	for _, data := range priceHistory {
		if data.Timestamp.After(cutoff) {
			prices = append(prices, data.Price)
		}
	}

	if len(prices) < 2 {
		return 0
	}

	minPrice := prices[0]
	maxPrice := prices[0]
	for _, price := range prices {
		if price < minPrice {
			minPrice = price
		}
		if price > maxPrice {
			maxPrice = price
		}
	}

	return (maxPrice - minPrice) / minPrice
}

// calculateVolumeSpike calculates volume spike compared to average
func (cbs *CircuitBreakerSystem) calculateVolumeSpike(symbol string, window time.Duration) float64 {
	priceHistory := cbs.priceData[symbol]
	cutoff := time.Now().Add(-window)

	var volumes []float64
	var recentVolume float64
	recentCutoff := time.Now().Add(-time.Minute) // Last minute volume

	for _, data := range priceHistory {
		if data.Timestamp.After(cutoff) {
			volumes = append(volumes, data.Volume)
			if data.Timestamp.After(recentCutoff) {
				recentVolume += data.Volume
			}
		}
	}

	if len(volumes) < 2 {
		return 0
	}

	// Calculate average volume
	avgVolume := 0.0
	for _, vol := range volumes {
		avgVolume += vol
	}
	avgVolume /= float64(len(volumes))

	if avgVolume == 0 {
		return 0
	}

	return recentVolume / avgVolume
}

// triggerCircuitBreaker triggers a circuit breaker
func (cbs *CircuitBreakerSystem) triggerCircuitBreaker(symbol string, reason HaltReason, message string) error {
	breaker := cbs.breakers[symbol]
	if breaker.State == CircuitBreakerOpen {
		return nil // Already halted
	}

	now := time.Now()
	breaker.State = CircuitBreakerOpen
	breaker.HaltReason = reason
	breaker.HaltedAt = &now
	breaker.HaltCount++
	breaker.TestOrderCount = 0

	cbs.haltCount++

	cbs.logger.Warn("Circuit breaker triggered",
		zap.String("symbol", symbol),
		zap.String("reason", string(reason)),
		zap.String("message", message),
		zap.Time("halted_at", now),
	)

	return nil
}

// attemptRecovery attempts to recover from a circuit breaker halt
func (cbs *CircuitBreakerSystem) attemptRecovery(symbol string, config *CircuitBreakerConfig, breaker *CircuitBreakerStatus) {
	if breaker.State != CircuitBreakerOpen {
		return
	}

	// Check if maximum halt time exceeded
	if breaker.HaltedAt != nil && time.Since(*breaker.HaltedAt) > config.MaxRecoveryTime {
		cbs.resumeTrading(symbol, "Maximum halt time exceeded")
		return
	}

	// Move to half-open state for testing
	breaker.State = CircuitBreakerHalfOpen
	breaker.TestOrderCount = 0

	cbs.logger.Info("Circuit breaker entering half-open state",
		zap.String("symbol", symbol),
	)
}

// TestOrder tests an order in half-open state
func (cbs *CircuitBreakerSystem) TestOrder(symbol string) bool {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	breaker, exists := cbs.breakers[symbol]
	if !exists || breaker.State != CircuitBreakerHalfOpen {
		return breaker.State == CircuitBreakerClosed
	}

	config := cbs.configs[symbol]
	breaker.TestOrderCount++

	// If enough test orders passed, resume trading
	if breaker.TestOrderCount >= config.RecoveryTestOrders {
		cbs.resumeTrading(symbol, "Test orders successful")
	}

	return true
}

// resumeTrading resumes trading for a symbol
func (cbs *CircuitBreakerSystem) resumeTrading(symbol string, reason string) {
	breaker := cbs.breakers[symbol]
	now := time.Now()

	if breaker.HaltedAt != nil {
		breaker.LastHaltDuration = now.Sub(*breaker.HaltedAt)
		cbs.avgHaltDuration = (cbs.avgHaltDuration + breaker.LastHaltDuration) / 2
	}

	breaker.State = CircuitBreakerClosed
	breaker.ResumedAt = &now
	breaker.HaltReason = ""
	breaker.TestOrderCount = 0

	cbs.resumeCount++

	cbs.logger.Info("Trading resumed",
		zap.String("symbol", symbol),
		zap.String("reason", reason),
		zap.Duration("halt_duration", breaker.LastHaltDuration),
	)
}

// IsHalted checks if trading is halted for a symbol
func (cbs *CircuitBreakerSystem) IsHalted(symbol string) bool {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()

	// Check global halt
	if cbs.globalHalt {
		return true
	}

	// Check symbol-specific halt
	if breaker, exists := cbs.breakers[symbol]; exists {
		return breaker.State == CircuitBreakerOpen
	}

	return false
}

// ManualHalt manually halts trading for a symbol
func (cbs *CircuitBreakerSystem) ManualHalt(symbol string, reason string) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	_, exists := cbs.breakers[symbol]
	if !exists {
		return fmt.Errorf("no circuit breaker found for symbol %s", symbol)
	}

	return cbs.triggerCircuitBreaker(symbol, HaltReasonManual, reason)
}

// ManualResume manually resumes trading for a symbol
func (cbs *CircuitBreakerSystem) ManualResume(symbol string) error {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	breaker, exists := cbs.breakers[symbol]
	if !exists {
		return fmt.Errorf("no circuit breaker found for symbol %s", symbol)
	}

	if breaker.State == CircuitBreakerClosed {
		return fmt.Errorf("trading is not halted for symbol %s", symbol)
	}

	cbs.resumeTrading(symbol, "Manual resume")
	return nil
}

// GlobalHalt halts all trading
func (cbs *CircuitBreakerSystem) GlobalHalt(reason HaltReason, message string) {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	cbs.globalHalt = true
	now := time.Now()
	cbs.globalHaltTime = &now
	cbs.globalHaltReason = reason

	cbs.logger.Warn("Global trading halt",
		zap.String("reason", string(reason)),
		zap.String("message", message),
	)
}

// GlobalResume resumes all trading
func (cbs *CircuitBreakerSystem) GlobalResume() {
	cbs.mu.Lock()
	defer cbs.mu.Unlock()

	cbs.globalHalt = false
	cbs.globalHaltTime = nil
	cbs.globalHaltReason = ""

	cbs.logger.Info("Global trading resumed")
}

// GetStatus returns the status of a circuit breaker
func (cbs *CircuitBreakerSystem) GetStatus(symbol string) (*CircuitBreakerStatus, bool) {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()

	status, exists := cbs.breakers[symbol]
	return status, exists
}

// GetAllStatuses returns all circuit breaker statuses
func (cbs *CircuitBreakerSystem) GetAllStatuses() map[string]*CircuitBreakerStatus {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()

	statuses := make(map[string]*CircuitBreakerStatus)
	for symbol, status := range cbs.breakers {
		statuses[symbol] = status
	}
	return statuses
}

// GetPerformanceMetrics returns circuit breaker performance metrics
func (cbs *CircuitBreakerSystem) GetPerformanceMetrics() map[string]interface{} {
	cbs.mu.RLock()
	defer cbs.mu.RUnlock()

	return map[string]interface{}{
		"total_halts":         cbs.haltCount,
		"total_resumes":       cbs.resumeCount,
		"avg_halt_duration":   cbs.avgHaltDuration.String(),
		"active_breakers":     len(cbs.breakers),
		"global_halt":         cbs.globalHalt,
		"global_halt_reason":  string(cbs.globalHaltReason),
	}
}
