package risk

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// Position represents a position
type Position struct {
	// Symbol is the trading symbol
	Symbol string

	// AccountID is the account ID
	AccountID string

	// Long is the long position
	Long float64

	// Short is the short position (as a positive number)
	Short float64

	// AvgLongPrice is the average price of the long position
	AvgLongPrice float64

	// AvgShortPrice is the average price of the short position
	AvgShortPrice float64

	// UnrealizedPnL is the unrealized profit and loss
	UnrealizedPnL float64

	// LastUpdateTime is the last update time
	LastUpdateTime time.Time
}

// Exposure represents an exposure
type Exposure struct {
	// AccountID is the account ID
	AccountID string

	// Notional is the notional exposure
	Notional float64

	// Beta is the beta-adjusted exposure
	Beta float64

	// Sector is the sector exposure
	Sector map[string]float64

	// Currency is the currency exposure
	Currency map[string]float64

	// LastUpdateTime is the last update time
	LastUpdateTime time.Time
}

// ExposureTracker tracks positions and exposures
type ExposureTracker struct {
	// Logger
	logger *zap.Logger

	// Positions by symbol and account
	positions map[string]map[string]*Position

	// Exposures by account
	exposures map[string]*Exposure

	// Market data by symbol
	marketData map[string]float64

	// Beta values by symbol
	betas map[string]float64

	// Sector mappings by symbol
	sectors map[string]string

	// Currency mappings by symbol
	currencies map[string]string

	// Mutex for thread safety
	mu sync.RWMutex
}

// NewExposureTracker creates a new ExposureTracker
func NewExposureTracker(logger *zap.Logger) *ExposureTracker {
	return &ExposureTracker{
		logger:     logger,
		positions:  make(map[string]map[string]*Position),
		exposures:  make(map[string]*Exposure),
		marketData: make(map[string]float64),
		betas:      make(map[string]float64),
		sectors:    make(map[string]string),
		currencies: make(map[string]string),
	}
}

// UpdatePosition updates a position
func (t *ExposureTracker) UpdatePosition(
	symbol, accountID string,
	deltaLong, deltaShort, longPrice, shortPrice float64,
) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Initialize maps if needed
	if _, exists := t.positions[symbol]; !exists {
		t.positions[symbol] = make(map[string]*Position)
	}

	// Get or create position
	position, exists := t.positions[symbol][accountID]
	if !exists {
		position = &Position{
			Symbol:    symbol,
			AccountID: accountID,
		}
		t.positions[symbol][accountID] = position
	}

	// Update long position
	if deltaLong != 0 {
		if position.Long == 0 {
			position.Long = deltaLong
			position.AvgLongPrice = longPrice
		} else {
			// Calculate new average price
			newLong := position.Long + deltaLong
			if newLong > 0 {
				position.AvgLongPrice = (position.Long*position.AvgLongPrice + deltaLong*longPrice) / newLong
				position.Long = newLong
			} else {
				position.Long = 0
				position.AvgLongPrice = 0
			}
		}
	}

	// Update short position
	if deltaShort != 0 {
		if position.Short == 0 {
			position.Short = deltaShort
			position.AvgShortPrice = shortPrice
		} else {
			// Calculate new average price
			newShort := position.Short + deltaShort
			if newShort > 0 {
				position.AvgShortPrice = (position.Short*position.AvgShortPrice + deltaShort*shortPrice) / newShort
				position.Short = newShort
			} else {
				position.Short = 0
				position.AvgShortPrice = 0
			}
		}
	}

	// Update unrealized PnL
	t.updateUnrealizedPnL(position)

	// Update last update time
	position.LastUpdateTime = time.Now()

	// Update exposure
	t.updateExposure(accountID)

	t.logger.Info("Updated position",
		zap.String("symbol", symbol),
		zap.String("account_id", accountID),
		zap.Float64("long", position.Long),
		zap.Float64("short", position.Short),
		zap.Float64("avg_long_price", position.AvgLongPrice),
		zap.Float64("avg_short_price", position.AvgShortPrice),
		zap.Float64("unrealized_pnl", position.UnrealizedPnL))
}

// updateUnrealizedPnL updates the unrealized PnL for a position
func (t *ExposureTracker) updateUnrealizedPnL(position *Position) {
	// Get current market price
	price, exists := t.marketData[position.Symbol]
	if !exists {
		return
	}

	// Calculate unrealized PnL
	longPnL := position.Long * (price - position.AvgLongPrice)
	shortPnL := position.Short * (position.AvgShortPrice - price)
	position.UnrealizedPnL = longPnL + shortPnL
}

// updateExposure updates the exposure for an account
func (t *ExposureTracker) updateExposure(accountID string) {
	// Get or create exposure
	exposure, exists := t.exposures[accountID]
	if !exists {
		exposure = &Exposure{
			AccountID: accountID,
			Sector:    make(map[string]float64),
			Currency:  make(map[string]float64),
		}
		t.exposures[accountID] = exposure
	}

	// Reset exposures
	exposure.Notional = 0
	exposure.Beta = 0
	for k := range exposure.Sector {
		exposure.Sector[k] = 0
	}
	for k := range exposure.Currency {
		exposure.Currency[k] = 0
	}

	// Calculate exposures
	for symbol, positions := range t.positions {
		if position, exists := positions[accountID]; exists {
			// Get market price
			price, exists := t.marketData[symbol]
			if !exists {
				continue
			}

			// Calculate notional
			notional := (position.Long - position.Short) * price
			exposure.Notional += notional

			// Calculate beta-adjusted exposure
			beta, exists := t.betas[symbol]
			if exists {
				exposure.Beta += notional * beta
			}

			// Calculate sector exposure
			sector, exists := t.sectors[symbol]
			if exists {
				exposure.Sector[sector] += notional
			}

			// Calculate currency exposure
			currency, exists := t.currencies[symbol]
			if exists {
				exposure.Currency[currency] += notional
			}
		}
	}

	// Update last update time
	exposure.LastUpdateTime = time.Now()
}

// UpdateMarketData updates market data
func (t *ExposureTracker) UpdateMarketData(symbol string, price float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.marketData[symbol] = price

	// Update unrealized PnL for all positions in this symbol
	if positions, exists := t.positions[symbol]; exists {
		for _, position := range positions {
			t.updateUnrealizedPnL(position)
			t.updateExposure(position.AccountID)
		}
	}
}

// SetBeta sets the beta value for a symbol
func (t *ExposureTracker) SetBeta(symbol string, beta float64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.betas[symbol] = beta

	// Update exposures for all accounts with positions in this symbol
	if positions, exists := t.positions[symbol]; exists {
		for accountID := range positions {
			t.updateExposure(accountID)
		}
	}
}

// SetSector sets the sector for a symbol
func (t *ExposureTracker) SetSector(symbol, sector string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.sectors[symbol] = sector

	// Update exposures for all accounts with positions in this symbol
	if positions, exists := t.positions[symbol]; exists {
		for accountID := range positions {
			t.updateExposure(accountID)
		}
	}
}

// SetCurrency sets the currency for a symbol
func (t *ExposureTracker) SetCurrency(symbol, currency string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	t.currencies[symbol] = currency

	// Update exposures for all accounts with positions in this symbol
	if positions, exists := t.positions[symbol]; exists {
		for accountID := range positions {
			t.updateExposure(accountID)
		}
	}
}

// GetPosition gets a position
func (t *ExposureTracker) GetPosition(symbol, accountID string) *Position {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if positions, exists := t.positions[symbol]; exists {
		return positions[accountID]
	}

	return nil
}

// GetExposure gets an exposure
func (t *ExposureTracker) GetExposure(accountID string) *Exposure {
	t.mu.RLock()
	defer t.mu.RUnlock()

	return t.exposures[accountID]
}

// GetAllPositions gets all positions
func (t *ExposureTracker) GetAllPositions() map[string]map[string]*Position {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a copy of the positions
	positions := make(map[string]map[string]*Position)
	for symbol, symbolPositions := range t.positions {
		positions[symbol] = make(map[string]*Position)
		for accountID, position := range symbolPositions {
			// Create a copy of the position
			positionCopy := *position
			positions[symbol][accountID] = &positionCopy
		}
	}

	return positions
}

// GetAllExposures gets all exposures
func (t *ExposureTracker) GetAllExposures() map[string]*Exposure {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Create a copy of the exposures
	exposures := make(map[string]*Exposure)
	for accountID, exposure := range t.exposures {
		// Create a copy of the exposure
		exposureCopy := *exposure
		exposureCopy.Sector = make(map[string]float64)
		exposureCopy.Currency = make(map[string]float64)
		for k, v := range exposure.Sector {
			exposureCopy.Sector[k] = v
		}
		for k, v := range exposure.Currency {
			exposureCopy.Currency[k] = v
		}
		exposures[accountID] = &exposureCopy
	}

	return exposures
}

