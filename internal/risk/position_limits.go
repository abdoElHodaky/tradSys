package risk

import (
	"context"
	"errors"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/math"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrPositionLimitExceeded = errors.New("position limit exceeded")
	ErrLimitNotFound         = errors.New("limit not found")
)

// PositionLimit represents a position limit
type PositionLimit struct {
	// Symbol is the trading symbol
	Symbol string

	// AccountID is the account ID
	AccountID string

	// MaxLong is the maximum long position
	MaxLong float64

	// MaxShort is the maximum short position (as a positive number)
	MaxShort float64

	// MaxNet is the maximum net position (long - short)
	MaxNet float64

	// MaxGross is the maximum gross position (long + short)
	MaxGross float64
}

// PositionLimitManager manages position limits
type PositionLimitManager struct {
	// Logger
	logger *zap.Logger

	// Limits by symbol and account
	limits map[string]map[string]*PositionLimit

	// Default limits
	defaultLimits map[string]*PositionLimit

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewPositionLimitManager creates a new PositionLimitManager
func NewPositionLimitManager(logger *zap.Logger) *PositionLimitManager {
	return &PositionLimitManager{
		logger:        logger,
		limits:        make(map[string]map[string]*PositionLimit),
		defaultLimits: make(map[string]*PositionLimit),
	}
}

// SetLimit sets a position limit
func (m *PositionLimitManager) SetLimit(limit *PositionLimit) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Initialize map for symbol if needed
	if _, exists := m.limits[limit.Symbol]; !exists {
		m.limits[limit.Symbol] = make(map[string]*PositionLimit)
	}

	// Set the limit
	m.limits[limit.Symbol][limit.AccountID] = limit

	m.logger.Info("Set position limit",
		zap.String("symbol", limit.Symbol),
		zap.String("account_id", limit.AccountID),
		zap.Float64("max_long", limit.MaxLong),
		zap.Float64("max_short", limit.MaxShort),
		zap.Float64("max_net", limit.MaxNet),
		zap.Float64("max_gross", limit.MaxGross))
}

// SetDefaultLimit sets a default position limit for a symbol
func (m *PositionLimitManager) SetDefaultLimit(symbol string, limit *PositionLimit) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.defaultLimits[symbol] = limit

	m.logger.Info("Set default position limit",
		zap.String("symbol", symbol),
		zap.Float64("max_long", limit.MaxLong),
		zap.Float64("max_short", limit.MaxShort),
		zap.Float64("max_net", limit.MaxNet),
		zap.Float64("max_gross", limit.MaxGross))
}

// GetLimit gets a position limit
func (m *PositionLimitManager) GetLimit(symbol, accountID string) (*PositionLimit, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Check for specific limit
	if symbolLimits, exists := m.limits[symbol]; exists {
		if limit, exists := symbolLimits[accountID]; exists {
			return limit, nil
		}
	}

	// Check for default limit
	if limit, exists := m.defaultLimits[symbol]; exists {
		return limit, nil
	}

	return nil, ErrLimitNotFound
}

// CheckLimit checks if a position would exceed limits
func (m *PositionLimitManager) CheckLimit(
	symbol, accountID string,
	currentLong, currentShort, deltaLong, deltaShort float64,
) error {
	// Get the limit
	limit, err := m.GetLimit(symbol, accountID)
	if err != nil {
		// No limit found, allow the position
		return nil
	}

	// Calculate new positions
	newLong := currentLong + deltaLong
	newShort := currentShort + deltaShort
	newNet := newLong - newShort
	newGross := newLong + newShort

	// Check limits
	if limit.MaxLong > 0 && newLong > limit.MaxLong {
		return errors.New("long position limit exceeded")
	}

	if limit.MaxShort > 0 && newShort > limit.MaxShort {
		return errors.New("short position limit exceeded")
	}

	if limit.MaxNet > 0 && math.Abs(newNet) > limit.MaxNet {
		return errors.New("net position limit exceeded")
	}

	if limit.MaxGross > 0 && newGross > limit.MaxGross {
		return errors.New("gross position limit exceeded")
	}

	return nil
}

// RemoveLimit removes a position limit
func (m *PositionLimitManager) RemoveLimit(symbol, accountID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if symbolLimits, exists := m.limits[symbol]; exists {
		delete(symbolLimits, accountID)
		if len(symbolLimits) == 0 {
			delete(m.limits, symbol)
		}
	}

	m.logger.Info("Removed position limit",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID))
}

// RemoveDefaultLimit removes a default position limit
func (m *PositionLimitManager) RemoveDefaultLimit(symbol string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.defaultLimits, symbol)

	m.logger.Info("Removed default position limit",
		zap.String("symbol", symbol))
}

// GetAllLimits gets all position limits
func (m *PositionLimitManager) GetAllLimits() map[string]map[string]*PositionLimit {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy of the limits
	limits := make(map[string]map[string]*PositionLimit)
	for symbol, symbolLimits := range m.limits {
		limits[symbol] = make(map[string]*PositionLimit)
		for accountID, limit := range symbolLimits {
			limits[symbol][accountID] = limit
		}
	}

	return limits
}

// GetAllDefaultLimits gets all default position limits
func (m *PositionLimitManager) GetAllDefaultLimits() map[string]*PositionLimit {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// Create a copy of the default limits
	defaultLimits := make(map[string]*PositionLimit)
	for symbol, limit := range m.defaultLimits {
		defaultLimits[symbol] = limit
	}

	return defaultLimits
}

// Use math.Abs from internal/math package instead
