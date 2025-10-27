// Package licensing provides enterprise licensing system for TradSys v3
package licensing

import (
	"time"
)

// LicenseTier represents different licensing tiers
type LicenseTier string

const (
	BASIC        LicenseTier = "BASIC"
	PROFESSIONAL LicenseTier = "PROFESSIONAL"
	ENTERPRISE   LicenseTier = "ENTERPRISE"
	ISLAMIC      LicenseTier = "ISLAMIC"
)

// LicenseFeature represents individual features that can be licensed
type LicenseFeature string

const (
	// Trading Features
	BASIC_TRADING      LicenseFeature = "BASIC_TRADING"
	ADVANCED_TRADING   LicenseFeature = "ADVANCED_TRADING"
	HFT_TRADING        LicenseFeature = "HFT_TRADING"
	
	// Exchange Access
	EGX_ACCESS         LicenseFeature = "EGX_ACCESS"
	ADX_ACCESS         LicenseFeature = "ADX_ACCESS"
	MULTI_EXCHANGE     LicenseFeature = "MULTI_EXCHANGE"
	
	// Asset Types
	BASIC_ASSETS       LicenseFeature = "BASIC_ASSETS"
	ADVANCED_ASSETS    LicenseFeature = "ADVANCED_ASSETS"
	ISLAMIC_ASSETS     LicenseFeature = "ISLAMIC_ASSETS"
	CRYPTO_ASSETS      LicenseFeature = "CRYPTO_ASSETS"
	
	// Analytics & Reporting
	BASIC_ANALYTICS    LicenseFeature = "BASIC_ANALYTICS"
	ADVANCED_ANALYTICS LicenseFeature = "ADVANCED_ANALYTICS"
	REAL_TIME_ANALYTICS LicenseFeature = "REAL_TIME_ANALYTICS"
	
	// Islamic Finance
	SHARIA_COMPLIANCE  LicenseFeature = "SHARIA_COMPLIANCE"
	ZAKAT_CALCULATION  LicenseFeature = "ZAKAT_CALCULATION"
	HALAL_SCREENING    LicenseFeature = "HALAL_SCREENING"
	
	// API & Integration
	REST_API           LicenseFeature = "REST_API"
	WEBSOCKET_API      LicenseFeature = "WEBSOCKET_API"
	THIRD_PARTY_API    LicenseFeature = "THIRD_PARTY_API"
)

// LicenseConfig represents the configuration for a license tier
type LicenseConfig struct {
	Tier            LicenseTier                `json:"tier"`
	Features        []LicenseFeature           `json:"features"`
	Quotas          map[string]int64           `json:"quotas"`
	RateLimits      map[string]int64           `json:"rate_limits"`
	ExpirationDate  time.Time                  `json:"expiration_date"`
	MaxUsers        int                        `json:"max_users"`
	MaxAssets       int                        `json:"max_assets"`
	MaxOrders       int64                      `json:"max_orders"`
}

// License represents a user's license
type License struct {
	ID              string                     `json:"id"`
	UserID          string                     `json:"user_id"`
	OrganizationID  string                     `json:"organization_id"`
	Tier            LicenseTier                `json:"tier"`
	Features        []LicenseFeature           `json:"features"`
	Quotas          map[string]int64           `json:"quotas"`
	RateLimits      map[string]int64           `json:"rate_limits"`
	IssuedAt        time.Time                  `json:"issued_at"`
	ExpiresAt       time.Time                  `json:"expires_at"`
	Active          bool                       `json:"is_active"`
	MaxUsers        int                        `json:"max_users"`
	MaxAssets       int                        `json:"max_assets"`
	MaxOrders       int64                      `json:"max_orders"`
	Metadata        map[string]interface{}     `json:"metadata"`
}

// ValidationResult represents the result of license validation
type ValidationResult struct {
	Valid         bool      `json:"valid"`
	Reason        string    `json:"reason"`
	QuotaUsed     int64     `json:"quota_used"`
	QuotaLimit    int64     `json:"quota_limit"`
	ExpiresAt     time.Time `json:"expires_at"`
	RemainingTime int64     `json:"remaining_time_seconds"`
}

// UsageStats represents usage statistics for a user
type UsageStats struct {
	UserID      string    `json:"user_id"`
	UsageType   string    `json:"usage_type"`
	Used        int64     `json:"used"`
	Quota       int64     `json:"quota"`
	Percentage  float64   `json:"percentage"`
	ResetTime   time.Time `json:"reset_time"`
	LastUpdated time.Time `json:"last_updated"`
}

// BillingPlan represents a billing plan for a license tier
type BillingPlan struct {
	TierID          string                     `json:"tier_id"`
	Name            string                     `json:"name"`
	Description     string                     `json:"description"`
	BaseFee         float64                    `json:"base_fee"`
	Currency        string                     `json:"currency"`
	BillingCycle    string                     `json:"billing_cycle"` // "monthly", "annual"
	UsageRates      map[string]float64         `json:"usage_rates"`
	OverageRates    map[string]float64         `json:"overage_rates"`
	Features        []LicenseFeature           `json:"features"`
	Quotas          map[string]int64           `json:"quotas"`
	IsActive        bool                       `json:"is_active"`
}

// Bill represents a billing statement
type Bill struct {
	ID            string                     `json:"id"`
	UserID        string                     `json:"user_id"`
	BillingPeriod string                     `json:"billing_period"`
	StartDate     time.Time                  `json:"start_date"`
	EndDate       time.Time                  `json:"end_date"`
	BaseFee       float64                    `json:"base_fee"`
	UsageFees     map[string]float64         `json:"usage_fees"`
	OverageFees   map[string]float64         `json:"overage_fees"`
	Total         float64                    `json:"total"`
	Currency      string                     `json:"currency"`
	Status        string                     `json:"status"` // "pending", "paid", "overdue"
	DueDate       time.Time                  `json:"due_date"`
	CreatedAt     time.Time                  `json:"created_at"`
	PaidAt        *time.Time                 `json:"paid_at,omitempty"`
}

// LicenseError represents licensing-related errors
type LicenseError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Feature string `json:"feature,omitempty"`
	UserID  string `json:"user_id,omitempty"`
}

func (e *LicenseError) Error() string {
	return e.Message
}

// Common license errors
var (
	ErrLicenseNotFound     = &LicenseError{Code: "LICENSE_NOT_FOUND", Message: "license not found"}
	ErrLicenseExpired      = &LicenseError{Code: "LICENSE_EXPIRED", Message: "license has expired"}
	ErrFeatureNotLicensed  = &LicenseError{Code: "FEATURE_NOT_LICENSED", Message: "feature not included in license"}
	ErrQuotaExceeded       = &LicenseError{Code: "QUOTA_EXCEEDED", Message: "usage quota exceeded"}
	ErrRateLimitExceeded   = &LicenseError{Code: "RATE_LIMIT_EXCEEDED", Message: "rate limit exceeded"}
	ErrLicenseInactive     = &LicenseError{Code: "LICENSE_INACTIVE", Message: "license is inactive"}
	ErrInvalidLicenseTier  = &LicenseError{Code: "INVALID_LICENSE_TIER", Message: "invalid license tier"}
)

// IsValid checks if a license tier is valid
func (lt LicenseTier) IsValid() bool {
	switch lt {
	case BASIC, PROFESSIONAL, ENTERPRISE, ISLAMIC:
		return true
	default:
		return false
	}
}

// String returns the string representation of LicenseTier
func (lt LicenseTier) String() string {
	return string(lt)
}

// IsValid checks if a license feature is valid
func (lf LicenseFeature) IsValid() bool {
	switch lf {
	case BASIC_TRADING, ADVANCED_TRADING, HFT_TRADING,
		 EGX_ACCESS, ADX_ACCESS, MULTI_EXCHANGE,
		 BASIC_ASSETS, ADVANCED_ASSETS, ISLAMIC_ASSETS, CRYPTO_ASSETS,
		 BASIC_ANALYTICS, ADVANCED_ANALYTICS, REAL_TIME_ANALYTICS,
		 SHARIA_COMPLIANCE, ZAKAT_CALCULATION, HALAL_SCREENING,
		 REST_API, WEBSOCKET_API, THIRD_PARTY_API:
		return true
	default:
		return false
	}
}

// String returns the string representation of LicenseFeature
func (lf LicenseFeature) String() string {
	return string(lf)
}

// IsExpired checks if the license has expired
func (l *License) IsExpired() bool {
	return time.Now().After(l.ExpiresAt)
}

// HasFeature checks if the license includes a specific feature
func (l *License) HasFeature(feature LicenseFeature) bool {
	for _, f := range l.Features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetQuota returns the quota for a specific usage type
func (l *License) GetQuota(usageType string) int64 {
	if quota, exists := l.Quotas[usageType]; exists {
		return quota
	}
	return 0
}

// GetRateLimit returns the rate limit for a specific usage type
func (l *License) GetRateLimit(usageType string) int64 {
	if limit, exists := l.RateLimits[usageType]; exists {
		return limit
	}
	return 0
}

// IsActive checks if the license is active and not expired
func (l *License) IsActive() bool {
	return l.Active && !l.IsExpired()
}
