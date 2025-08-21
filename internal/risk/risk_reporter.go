package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
)

// RiskReport represents a risk report
type RiskReport struct {
	// Timestamp is the timestamp of the report
	Timestamp time.Time `json:"timestamp"`

	// AccountID is the account ID
	AccountID string `json:"account_id"`

	// Positions are the positions
	Positions map[string]*Position `json:"positions"`

	// Exposure is the exposure
	Exposure *Exposure `json:"exposure"`

	// RiskMetrics are the risk metrics
	RiskMetrics map[string]float64 `json:"risk_metrics"`
}

// RiskReporter generates risk reports
type RiskReporter struct {
	// Logger
	logger *zap.Logger

	// Exposure tracker
	exposureTracker *ExposureTracker

	// Reports by account
	reports map[string][]*RiskReport

	// Maximum number of reports to keep
	maxReports int

	// Mutex for thread safety
	mu sync.RWMutex

	// Report generation interval
	reportInterval time.Duration

	// Report generation ticker
	ticker *time.Ticker

	// Context for cancellation
	ctx context.Context
	cancel context.CancelFunc
}

// NewRiskReporter creates a new RiskReporter
func NewRiskReporter(
	logger *zap.Logger,
	exposureTracker *ExposureTracker,
) *RiskReporter {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &RiskReporter{
		logger:          logger,
		exposureTracker: exposureTracker,
		reports:         make(map[string][]*RiskReport),
		maxReports:      100,
		reportInterval:  15 * time.Minute,
		ctx:             ctx,
		cancel:          cancel,
	}
}

// Start starts the risk reporter
func (r *RiskReporter) Start(ctx context.Context) error {
	r.ticker = time.NewTicker(r.reportInterval)
	
	go func() {
		for {
			select {
			case <-r.ticker.C:
				r.generateReports()
			case <-r.ctx.Done():
				return
			case <-ctx.Done():
				return
			}
		}
	}()
	
	r.logger.Info("Started risk reporter",
		zap.Duration("interval", r.reportInterval))
	
	return nil
}

// Stop stops the risk reporter
func (r *RiskReporter) Stop(ctx context.Context) error {
	if r.ticker != nil {
		r.ticker.Stop()
	}
	
	r.cancel()
	
	r.logger.Info("Stopped risk reporter")
	
	return nil
}

// SetReportInterval sets the report generation interval
func (r *RiskReporter) SetReportInterval(interval time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.reportInterval = interval
	
	if r.ticker != nil {
		r.ticker.Reset(interval)
	}
	
	r.logger.Info("Set report interval",
		zap.Duration("interval", interval))
}

// generateReports generates risk reports for all accounts
func (r *RiskReporter) generateReports() {
	// Get all exposures
	exposures := r.exposureTracker.GetAllExposures()
	
	// Generate reports for each account
	for accountID, exposure := range exposures {
		r.generateReportForAccount(accountID, exposure)
	}
}

// generateReportForAccount generates a risk report for an account
func (r *RiskReporter) generateReportForAccount(accountID string, exposure *Exposure) {
	// Get all positions for the account
	positions := make(map[string]*Position)
	allPositions := r.exposureTracker.GetAllPositions()
	for symbol, symbolPositions := range allPositions {
		if position, exists := symbolPositions[accountID]; exists {
			positions[symbol] = position
		}
	}
	
	// Calculate risk metrics
	riskMetrics := r.calculateRiskMetrics(accountID, positions, exposure)
	
	// Create report
	report := &RiskReport{
		Timestamp:   time.Now(),
		AccountID:   accountID,
		Positions:   positions,
		Exposure:    exposure,
		RiskMetrics: riskMetrics,
	}
	
	// Store report
	r.mu.Lock()
	defer r.mu.Unlock()
	
	if _, exists := r.reports[accountID]; !exists {
		r.reports[accountID] = make([]*RiskReport, 0, r.maxReports)
	}
	
	r.reports[accountID] = append(r.reports[accountID], report)
	
	// Trim reports if needed
	if len(r.reports[accountID]) > r.maxReports {
		r.reports[accountID] = r.reports[accountID][1:]
	}
	
	r.logger.Info("Generated risk report",
		zap.String("account_id", accountID),
		zap.Time("timestamp", report.Timestamp),
		zap.Int("position_count", len(positions)),
		zap.Float64("notional_exposure", exposure.Notional),
		zap.Float64("beta_exposure", exposure.Beta))
}

// calculateRiskMetrics calculates risk metrics
func (r *RiskReporter) calculateRiskMetrics(
	accountID string,
	positions map[string]*Position,
	exposure *Exposure,
) map[string]float64 {
	metrics := make(map[string]float64)
	
	// Calculate total position value
	var totalValue float64
	for _, position := range positions {
		totalValue += position.UnrealizedPnL
	}
	
	// Calculate total unrealized PnL
	var totalUnrealizedPnL float64
	for _, position := range positions {
		totalUnrealizedPnL += position.UnrealizedPnL
	}
	
	// Calculate net exposure
	metrics["net_exposure"] = exposure.Notional
	
	// Calculate gross exposure
	var grossExposure float64
	for _, position := range positions {
		grossExposure += (position.Long + position.Short)
	}
	metrics["gross_exposure"] = grossExposure
	
	// Calculate leverage
	if totalValue > 0 {
		metrics["leverage"] = grossExposure / totalValue
	}
	
	// Calculate beta-adjusted exposure
	metrics["beta_exposure"] = exposure.Beta
	
	// Calculate unrealized PnL
	metrics["unrealized_pnl"] = totalUnrealizedPnL
	
	// Calculate unrealized PnL as percentage of total value
	if totalValue > 0 {
		metrics["unrealized_pnl_pct"] = totalUnrealizedPnL / totalValue
	}
	
	// More metrics could be added here
	
	return metrics
}

// GetReports gets risk reports for an account
func (r *RiskReporter) GetReports(accountID string, limit int) []*RiskReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	reports, exists := r.reports[accountID]
	if !exists {
		return nil
	}
	
	if limit <= 0 || limit > len(reports) {
		limit = len(reports)
	}
	
	result := make([]*RiskReport, limit)
	copy(result, reports[len(reports)-limit:])
	
	return result
}

// GetLatestReport gets the latest risk report for an account
func (r *RiskReporter) GetLatestReport(accountID string) *RiskReport {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	reports, exists := r.reports[accountID]
	if !exists || len(reports) == 0 {
		return nil
	}
	
	return reports[len(reports)-1]
}

// GenerateReport generates a risk report for an account on demand
func (r *RiskReporter) GenerateReport(accountID string) *RiskReport {
	// Get exposure
	exposure := r.exposureTracker.GetExposure(accountID)
	if exposure == nil {
		return nil
	}
	
	// Generate report
	r.generateReportForAccount(accountID, exposure)
	
	// Return the latest report
	return r.GetLatestReport(accountID)
}

// ExportReportToJSON exports a risk report to JSON
func (r *RiskReporter) ExportReportToJSON(report *RiskReport, filePath string) error {
	// Marshal report to JSON
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return err
	}
	
	// Write to file
	err = os.WriteFile(filePath, data, 0644)
	if err != nil {
		return err
	}
	
	r.logger.Info("Exported risk report to JSON",
		zap.String("account_id", report.AccountID),
		zap.Time("timestamp", report.Timestamp),
		zap.String("file_path", filePath))
	
	return nil
}

// ExportReportToCSV exports a risk report to CSV
func (r *RiskReporter) ExportReportToCSV(report *RiskReport, filePath string) error {
	// Open file
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	
	// Write header
	_, err = fmt.Fprintf(file, "Account ID: %s\n", report.AccountID)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Timestamp: %s\n\n", report.Timestamp.Format(time.RFC3339))
	if err != nil {
		return err
	}
	
	// Write positions
	_, err = fmt.Fprintf(file, "Positions:\n")
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Symbol,Long,Short,Avg Long Price,Avg Short Price,Unrealized PnL\n")
	if err != nil {
		return err
	}
	
	for symbol, position := range report.Positions {
		_, err = fmt.Fprintf(file, "%s,%.2f,%.2f,%.2f,%.2f,%.2f\n",
			symbol,
			position.Long,
			position.Short,
			position.AvgLongPrice,
			position.AvgShortPrice,
			position.UnrealizedPnL)
		if err != nil {
			return err
		}
	}
	
	// Write exposure
	_, err = fmt.Fprintf(file, "\nExposure:\n")
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Notional,%.2f\n", report.Exposure.Notional)
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Beta,%.2f\n", report.Exposure.Beta)
	if err != nil {
		return err
	}
	
	// Write sector exposure
	_, err = fmt.Fprintf(file, "\nSector Exposure:\n")
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Sector,Exposure\n")
	if err != nil {
		return err
	}
	
	for sector, exposure := range report.Exposure.Sector {
		_, err = fmt.Fprintf(file, "%s,%.2f\n", sector, exposure)
		if err != nil {
			return err
		}
	}
	
	// Write currency exposure
	_, err = fmt.Fprintf(file, "\nCurrency Exposure:\n")
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Currency,Exposure\n")
	if err != nil {
		return err
	}
	
	for currency, exposure := range report.Exposure.Currency {
		_, err = fmt.Fprintf(file, "%s,%.2f\n", currency, exposure)
		if err != nil {
			return err
		}
	}
	
	// Write risk metrics
	_, err = fmt.Fprintf(file, "\nRisk Metrics:\n")
	if err != nil {
		return err
	}
	
	_, err = fmt.Fprintf(file, "Metric,Value\n")
	if err != nil {
		return err
	}
	
	for metric, value := range report.RiskMetrics {
		_, err = fmt.Fprintf(file, "%s,%.2f\n", metric, value)
		if err != nil {
			return err
		}
	}
	
	r.logger.Info("Exported risk report to CSV",
		zap.String("account_id", report.AccountID),
		zap.Time("timestamp", report.Timestamp),
		zap.String("file_path", filePath))
	
	return nil
}

