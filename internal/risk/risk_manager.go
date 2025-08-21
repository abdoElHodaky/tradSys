package risk

import (
	"context"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RiskManager manages risk components
type RiskManager struct {
	// Logger
	logger *zap.Logger

	// Position limit manager
	positionLimitManager *PositionLimitManager

	// Exposure tracker
	exposureTracker *ExposureTracker

	// Risk validator
	riskValidator *RiskValidator

	// Risk reporter
	riskReporter *RiskReporter

	// Running state
	running bool

	// Mutex for thread safety
	mu sync.RWMutex

	// Context for cancellation
	ctx context.Context
	cancel context.CancelFunc

	// Exposure validation interval
	exposureValidationInterval time.Duration

	// Exposure validation ticker
	ticker *time.Ticker

	// Exposure validation callback
	exposureValidationCallback func(accountID string, result *RiskCheckResult, err error)
}

// NewRiskManager creates a new RiskManager
func NewRiskManager(
	logger *zap.Logger,
	positionLimitManager *PositionLimitManager,
	exposureTracker *ExposureTracker,
	riskValidator *RiskValidator,
	riskReporter *RiskReporter,
) *RiskManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &RiskManager{
		logger:                   logger,
		positionLimitManager:     positionLimitManager,
		exposureTracker:          exposureTracker,
		riskValidator:            riskValidator,
		riskReporter:             riskReporter,
		running:                  false,
		ctx:                      ctx,
		cancel:                   cancel,
		exposureValidationInterval: 5 * time.Minute,
	}
}

// Start starts the risk manager
func (m *RiskManager) Start(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.running {
		return nil
	}

	// Start risk reporter
	err := m.riskReporter.Start(ctx)
	if err != nil {
		return err
	}

	// Start exposure validation
	m.ticker = time.NewTicker(m.exposureValidationInterval)

	go func() {
		for {
			select {
			case <-m.ticker.C:
				m.validateAllExposures()
			case <-m.ctx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()

	m.running = true

	m.logger.Info("Started risk manager",
		zap.Duration("exposure_validation_interval", m.exposureValidationInterval))

	return nil
}

// Stop stops the risk manager
func (m *RiskManager) Stop(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.running {
		return nil
	}

	// Stop risk reporter
	err := m.riskReporter.Stop(ctx)
	if err != nil {
		return err
	}

	// Stop exposure validation
	if m.ticker != nil {
		m.ticker.Stop()
	}

	m.cancel()
	m.running = false

	m.logger.Info("Stopped risk manager")

	return nil
}

// IsRunning returns whether the risk manager is running
func (m *RiskManager) IsRunning() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.running
}

// SetExposureValidationInterval sets the exposure validation interval
func (m *RiskManager) SetExposureValidationInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.exposureValidationInterval = interval

	if m.ticker != nil {
		m.ticker.Reset(interval)
	}

	m.logger.Info("Set exposure validation interval",
		zap.Duration("interval", interval))
}

// SetExposureValidationCallback sets the exposure validation callback
func (m *RiskManager) SetExposureValidationCallback(
	callback func(accountID string, result *RiskCheckResult, err error),
) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.exposureValidationCallback = callback
}

// validateAllExposures validates all exposures
func (m *RiskManager) validateAllExposures() {
	// Get all exposures
	exposures := m.exposureTracker.GetAllExposures()

	// Validate each exposure
	for accountID := range exposures {
		m.validateExposure(accountID)
	}
}

// validateExposure validates an exposure
func (m *RiskManager) validateExposure(accountID string) {
	// Validate exposure
	result, err := m.riskValidator.ValidateExposure(m.ctx, accountID)

	// Call callback if set
	m.mu.RLock()
	callback := m.exposureValidationCallback
	m.mu.RUnlock()

	if callback != nil {
		callback(accountID, result, err)
	}

	// Log result
	if err != nil {
		m.logger.Error("Exposure validation failed",
			zap.String("account_id", accountID),
			zap.Error(err))
	} else if result != nil && !result.Passed {
		m.logger.Warn("Exposure validation failed",
			zap.String("account_id", accountID),
			zap.String("reason", result.Reason),
			zap.Any("details", result.Details))

		// Generate risk report
		m.riskReporter.GenerateReport(accountID)
	}
}

// SetPositionLimit sets a position limit
func (m *RiskManager) SetPositionLimit(limit *PositionLimit) {
	m.positionLimitManager.SetLimit(limit)
}

// SetDefaultPositionLimit sets a default position limit
func (m *RiskManager) SetDefaultPositionLimit(symbol string, limit *PositionLimit) {
	m.positionLimitManager.SetDefaultLimit(symbol, limit)
}

// SetRiskLimit sets a risk limit
func (m *RiskManager) SetRiskLimit(limit *RiskLimit) {
	m.riskValidator.SetRiskLimit(limit)
}

// UpdatePosition updates a position
func (m *RiskManager) UpdatePosition(
	symbol, accountID string,
	deltaLong, deltaShort, longPrice, shortPrice float64,
) {
	m.exposureTracker.UpdatePosition(
		symbol,
		accountID,
		deltaLong,
		deltaShort,
		longPrice,
		shortPrice,
	)
}

// UpdateMarketData updates market data
func (m *RiskManager) UpdateMarketData(symbol string, price float64) {
	m.exposureTracker.UpdateMarketData(symbol, price)
}

// SetBeta sets the beta value for a symbol
func (m *RiskManager) SetBeta(symbol string, beta float64) {
	m.exposureTracker.SetBeta(symbol, beta)
}

// SetSector sets the sector for a symbol
func (m *RiskManager) SetSector(symbol, sector string) {
	m.exposureTracker.SetSector(symbol, sector)
}

// SetCurrency sets the currency for a symbol
func (m *RiskManager) SetCurrency(symbol, currency string) {
	m.exposureTracker.SetCurrency(symbol, currency)
}

// GetPosition gets a position
func (m *RiskManager) GetPosition(symbol, accountID string) *Position {
	return m.exposureTracker.GetPosition(symbol, accountID)
}

// GetExposure gets an exposure
func (m *RiskManager) GetExposure(accountID string) *Exposure {
	return m.exposureTracker.GetExposure(accountID)
}

// GenerateRiskReport generates a risk report
func (m *RiskManager) GenerateRiskReport(accountID string) *RiskReport {
	return m.riskReporter.GenerateReport(accountID)
}

// GetRiskReports gets risk reports
func (m *RiskManager) GetRiskReports(accountID string, limit int) []*RiskReport {
	return m.riskReporter.GetReports(accountID, limit)
}

// ExportRiskReportToJSON exports a risk report to JSON
func (m *RiskManager) ExportRiskReportToJSON(report *RiskReport, filePath string) error {
	return m.riskReporter.ExportReportToJSON(report, filePath)
}

// ExportRiskReportToCSV exports a risk report to CSV
func (m *RiskManager) ExportRiskReportToCSV(report *RiskReport, filePath string) error {
	return m.riskReporter.ExportReportToCSV(report, filePath)
}

