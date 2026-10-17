// Package compliance provides centralized compliance types for TradSys v3
// Islamic-specific types follow KISS principle by keeping Islamic finance
// compliance definitions in a dedicated file for better organization
package compliance

import (
	"time"
)

// ============================================================================
// Islamic Finance Types
// ============================================================================

// ComplianceLevel defines Sharia compliance levels
type ComplianceLevel int

const (
	ComplianceLevelHalal ComplianceLevel = iota
	ComplianceLevelDoubtful
	ComplianceLevelHaram
	ComplianceLevelUnderReview
)

// ShariaTypes defines Sharia-related constants
type ShariaTypes string

const (
	ShariaTypeHalal   ShariaTypes = "halal"
	ShariaTypePartial ShariaTypes = "partial"
	ShariaTypeHaram   ShariaTypes = "haram"
	ShariaTypePending ShariaTypes = "pending"
)

// ShariaBoard represents a Sharia supervisory board
type ShariaBoard struct {
	ID          string
	Name        string
	Country     string
	Scholars    []ShariaScholar
	Methodology string
	IsActive    bool
	LastReview  time.Time
}

// ShariaScholar represents a Sharia scholar
type ShariaScholar struct {
	Name           string
	Qualification  string
	Specialization []string
	IsActive       bool
}

// ShariaRule represents an Islamic finance compliance rule
type ShariaRule struct {
	RuleID          string
	Description     string
	ShariaBoard     string
	AssetTypes      []string
	ComplianceLevel ComplianceLevel
	LastUpdated     time.Time
}

// IslamicInfo represents Islamic finance information
type IslamicInfo struct {
	IsHalal         bool
	ShariaBoard     string
	LastScreened    time.Time
	ComplianceScore float64
	Restrictions    []string
}

// ShariaComplianceReport represents Sharia compliance report
type ShariaComplianceReport struct {
	Symbol          string
	ComplianceLevel ComplianceLevel
	ShariaBoard     string
	LastScreened    time.Time
	ComplianceScore float64
	Restrictions    []string
	Recommendations []string
}

// ScreeningResult represents asset screening result
type ScreeningResult struct {
	Symbol          string
	IsCompliant     bool
	ComplianceScore float64
	Violations      []string
	Recommendations []string
	ScreenedAt      time.Time
}

// IslamicHolding represents an Islamic fund holding
type IslamicHolding struct {
	Symbol          string
	Name            string
	Weight          float64
	ComplianceScore float64
	ShariaBoard     string
}

// ZakatCalculation represents Zakat calculation result
type ZakatCalculation struct {
	TotalValue    float64
	ZakableAmount float64
	ZakatDue      float64
	Rate          float64
	Currency      string
	CalculatedAt  time.Time
}

// ============================================================================
// UAE Regulatory Types
// ============================================================================

// UAEComplianceInfo contains UAE market compliance information
type UAEComplianceInfo struct {
	ExchangeRegulationLevel string
	FSSARegistered          bool
	LastUpdated             time.Time
}

// SCAComplianceSettings contains SCA-specific compliance settings
type SCAComplianceSettings struct {
	ReportingFrequency string
	ReportingFormat    string
	ReportDestination  string
	LicenseType        string
	LicenseExpiry      time.Time
	LicensingVendor    string
}

// RegionalRegulation represents region-specific regulations
type RegionalRegulation struct {
	RegulationCode string
	Name           string
	Description    string
	EffectiveFrom  time.Time
	EffectiveTo    time.Time
	IsCurrent      bool
}