// Package assets provides type definitions for unified asset system
// Following KISS: Use type aliasing from pkg/types for shared types
package assets

import (
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// Type alias for AssetType from pkg/types
type AssetType = types.AssetType

// AssetStatus represents the status of an asset
type AssetStatus string

// AssetFilter provides filtering capabilities for asset searches
type AssetFilter struct {
	Symbol    string
	Name      string
	AssetType AssetType
	Exchange  string
	Region    string
	MinPrice  float64
	MaxPrice  float64
}

// AnalyticsRequest represents an analytics request
type AnalyticsRequest struct {
	UserID      string
	PortfolioID string
	AssetIDs    []string
	Metrics     []string
	TimeRange   TimeRange
	Benchmarks  []string
}

// ComplianceRequest represents a compliance request
type ComplianceRequest struct {
	UserID        string
	PortfolioID   string
	AssetIDs      []string
	Jurisdictions []string
	ReportType    string
}

// TimeRange represents a time range for analytics
type TimeRange struct {
	StartDate time.Time
	EndDate   time.Time
}

// SystemMetrics represents unified system metrics
type SystemMetrics struct {
	TotalAssets     int
	TotalPortfolios int
	ActiveUsers     int
	SystemUptime    time.Duration
	Timestamp       time.Time
}

// PortfolioPerformance represents portfolio performance metrics
type PortfolioPerformance struct {
	TotalReturn      float64
	AnnualizedReturn float64
	Volatility       float64
	SharpeRatio      float64
	MaxDrawdown      float64
	Beta             float64
	Alpha            float64
	LastUpdated      time.Time
}

// PortfolioRiskMetrics represents portfolio risk metrics
type PortfolioRiskMetrics struct {
	VaR95              float64
	VaR99              float64
	ExpectedShortfall  float64
	ConcentrationRisk  float64
	CurrencyRisk       float64
	LastCalculated     time.Time
}

// IslamicPortfolioInfo represents Islamic portfolio information
type IslamicPortfolioInfo struct {
	IsCompliant       bool
	ComplianceScore   float64
	ZakatDue          float64
	NonCompliantValue float64
	LastScreened      time.Time
}

// AssetStatus constants
const (
	AssetStatusActive    AssetStatus = "active"
	AssetStatusInactive  AssetStatus = "inactive"
	AssetStatusPending   AssetStatus = "pending"
	AssetStatusSuspended AssetStatus = "suspended"
)