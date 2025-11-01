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
	BASIC_TRADING    LicenseFeature = "BASIC_TRADING"
	ADVANCED_TRADING LicenseFeature = "ADVANCED_TRADING"
	HFT_TRADING      LicenseFeature = "HFT_TRADING"

	// Exchange Access
	EGX_ACCESS     LicenseFeature = "EGX_ACCESS"
	ADX_ACCESS     LicenseFeature = "ADX_ACCESS"
	MULTI_EXCHANGE LicenseFeature = "MULTI_EXCHANGE"

	// Asset Types
	BASIC_ASSETS    LicenseFeature = "BASIC_ASSETS"
	ADVANCED_ASSETS LicenseFeature = "ADVANCED_ASSETS"
	ISLAMIC_ASSETS  LicenseFeature = "ISLAMIC_ASSETS"
	CRYPTO_ASSETS   LicenseFeature = "CRYPTO_ASSETS"

	// Analytics & Reporting
	BASIC_ANALYTICS     LicenseFeature = "BASIC_ANALYTICS"
	ADVANCED_ANALYTICS  LicenseFeature = "ADVANCED_ANALYTICS"
	REAL_TIME_ANALYTICS LicenseFeature = "REAL_TIME_ANALYTICS"

	// Islamic Finance
	SHARIA_COMPLIANCE LicenseFeature = "SHARIA_COMPLIANCE"
	ZAKAT_CALCULATION LicenseFeature = "ZAKAT_CALCULATION"
	HALAL_SCREENING   LicenseFeature = "HALAL_SCREENING"

	// API & Integration
	REST_API        LicenseFeature = "REST_API"
	WEBSOCKET_API   LicenseFeature = "WEBSOCKET_API"
	THIRD_PARTY_API LicenseFeature = "THIRD_PARTY_API"
)

// LicenseConfig represents the configuration for a license tier
type LicenseConfig struct {
	Tier           LicenseTier      `json:"tier"`
	Features       []LicenseFeature `json:"features"`
	Quotas         map[string]int64 `json:"quotas"`
	RateLimits     map[string]int64 `json:"rate_limits"`
	ExpirationDate time.Time        `json:"expiration_date"`
	MaxUsers       int              `json:"max_users"`
	MaxAssets      int              `json:"max_assets"`
	MaxOrders      int64            `json:"max_orders"`
}

// License represents a user's license
type License struct {
	ID             string                 `json:"id"`
	UserID         string                 `json:"user_id"`
	OrganizationID string                 `json:"organization_id"`
	Tier           LicenseTier            `json:"tier"`
	Features       []LicenseFeature       `json:"features"`
	Quotas         map[string]int64       `json:"quotas"`
	RateLimits     map[string]int64       `json:"rate_limits"`
	IssuedAt       time.Time              `json:"issued_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
	IsActive       bool                   `json:"is_active"`
	MaxUsers       int                    `json:"max_users"`
	MaxAssets      int                    `json:"max_assets"`
	MaxOrders      int64                  `json:"max_orders"`
	Metadata       map[string]interface{} `json:"metadata"`
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
	TierID       string             `json:"tier_id"`
	Name         string             `json:"name"`
	Description  string             `json:"description"`
	BaseFee      float64            `json:"base_fee"`
	Currency     string             `json:"currency"`
	BillingCycle string             `json:"billing_cycle"` // "monthly", "annual"
	UsageRates   map[string]float64 `json:"usage_rates"`
	OverageRates map[string]float64 `json:"overage_rates"`
	Features     []LicenseFeature   `json:"features"`
	Quotas       map[string]int64   `json:"quotas"`
	IsActive     bool               `json:"is_active"`
}

// Bill represents a billing statement
type Bill struct {
	ID            string             `json:"id"`
	UserID        string             `json:"user_id"`
	BillingPeriod string             `json:"billing_period"`
	StartDate     time.Time          `json:"start_date"`
	EndDate       time.Time          `json:"end_date"`
	BaseFee       float64            `json:"base_fee"`
	UsageFees     map[string]float64 `json:"usage_fees"`
	OverageFees   map[string]float64 `json:"overage_fees"`
	Total         float64            `json:"total"`
	Currency      string             `json:"currency"`
	Status        string             `json:"status"` // "pending", "paid", "overdue"
	DueDate       time.Time          `json:"due_date"`
	CreatedAt     time.Time          `json:"created_at"`
	PaidAt        *time.Time         `json:"paid_at,omitempty"`
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
	ErrLicenseNotFound    = &LicenseError{Code: "LICENSE_NOT_FOUND", Message: "license not found"}
	ErrLicenseExpired     = &LicenseError{Code: "LICENSE_EXPIRED", Message: "license has expired"}
	ErrFeatureNotLicensed = &LicenseError{Code: "FEATURE_NOT_LICENSED", Message: "feature not included in license"}
	ErrQuotaExceeded      = &LicenseError{Code: "QUOTA_EXCEEDED", Message: "usage quota exceeded"}
	ErrRateLimitExceeded  = &LicenseError{Code: "RATE_LIMIT_EXCEEDED", Message: "rate limit exceeded"}
	ErrLicenseInactive    = &LicenseError{Code: "LICENSE_INACTIVE", Message: "license is inactive"}
	ErrInvalidLicenseTier = &LicenseError{Code: "INVALID_LICENSE_TIER", Message: "invalid license tier"}
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

// IsValid checks if the license is active and not expired
func (l *License) IsValid() bool {
	return l.IsActive && !l.IsExpired()
}

// CanGrant checks if the license can grant access to a specific feature
func (l *License) CanGrant(feature LicenseFeature) bool {
	return l.IsActive && !l.IsExpired() && l.MaxUsers > 0 && l.HasFeature(feature)
}

// CanAccessExchange checks if the license allows access to a specific exchange
func (l *License) CanAccessExchange(exchange string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch exchange {
	case "EGX":
		return l.HasFeature(EGX_ACCESS) || l.HasFeature(MULTI_EXCHANGE)
	case "ADX":
		return l.HasFeature(ADX_ACCESS) || l.HasFeature(MULTI_EXCHANGE)
	default:
		return l.HasFeature(MULTI_EXCHANGE)
	}
}

// CanPerformTradingType checks if the license allows a specific trading type
func (l *License) CanPerformTradingType(tradingType string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch tradingType {
	case "basic":
		return l.HasFeature(BASIC_TRADING)
	case "advanced":
		return l.HasFeature(ADVANCED_TRADING) || l.HasFeature(HFT_TRADING)
	case "hft":
		return l.HasFeature(HFT_TRADING)
	default:
		return false
	}
}

// CanAccessAssetType checks if the license allows access to specific asset types
func (l *License) CanAccessAssetType(assetType string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch assetType {
	case "basic":
		return l.HasFeature(BASIC_ASSETS)
	case "advanced":
		return l.HasFeature(ADVANCED_ASSETS)
	case "islamic":
		return l.HasFeature(ISLAMIC_ASSETS)
	case "crypto":
		return l.HasFeature(CRYPTO_ASSETS)
	default:
		return false
	}
}

// CanUseAnalytics checks if the license allows analytics features
func (l *License) CanUseAnalytics(analyticsLevel string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch analyticsLevel {
	case "basic":
		return l.HasFeature(BASIC_ANALYTICS)
	case "advanced":
		return l.HasFeature(ADVANCED_ANALYTICS)
	case "realtime":
		return l.HasFeature(REAL_TIME_ANALYTICS)
	default:
		return false
	}
}

// CanUseAPI checks if the license allows API access
func (l *License) CanUseAPI(apiType string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch apiType {
	case "rest":
		return l.HasFeature(REST_API)
	case "websocket":
		return l.HasFeature(WEBSOCKET_API)
	case "third_party":
		return l.HasFeature(THIRD_PARTY_API)
	default:
		return false
	}
}

// CanUseIslamicFeatures checks if the license allows Islamic finance features
func (l *License) CanUseIslamicFeatures(featureType string) bool {
	if !l.IsValid() {
		return false
	}
	
	switch featureType {
	case "compliance":
		return l.HasFeature(SHARIA_COMPLIANCE)
	case "zakat":
		return l.HasFeature(ZAKAT_CALCULATION)
	case "screening":
		return l.HasFeature(HALAL_SCREENING)
	default:
		return false
	}
}

// HasCapacity checks if the license has capacity for additional users
func (l *License) HasCapacity() bool {
	return l.IsValid() && l.MaxUsers > 0
}

// HasQuotaRemaining checks if there's remaining quota for a usage type
func (l *License) HasQuotaRemaining(usageType string, currentUsage int64) bool {
	if !l.IsValid() {
		return false
	}
	
	quota := l.GetQuota(usageType)
	return quota == 0 || currentUsage < quota // 0 means unlimited
}

// IsWithinRateLimit checks if usage is within rate limits
func (l *License) IsWithinRateLimit(usageType string, currentRate int64) bool {
	if !l.IsValid() {
		return false
	}
	
	rateLimit := l.GetRateLimit(usageType)
	return rateLimit == 0 || currentRate <= rateLimit // 0 means unlimited
}

// CanExecuteOrder checks if the license allows order execution with given parameters
func (l *License) CanExecuteOrder(exchange, tradingType, assetType string) bool {
	return l.CanAccessExchange(exchange) && 
		   l.CanPerformTradingType(tradingType) && 
		   l.CanAccessAssetType(assetType)
}

// GetUsagePercentage calculates usage percentage for a quota type
func (l *License) GetUsagePercentage(usageType string, currentUsage int64) float64 {
	quota := l.GetQuota(usageType)
	if quota == 0 {
		return 0.0 // Unlimited
	}
	
	percentage := float64(currentUsage) / float64(quota) * 100.0
	if percentage > 100.0 {
		return 100.0
	}
	return percentage
}

// GetRemainingQuota calculates remaining quota for a usage type
func (l *License) GetRemainingQuota(usageType string, currentUsage int64) int64 {
	quota := l.GetQuota(usageType)
	if quota == 0 {
		return -1 // Unlimited
	}
	
	remaining := quota - currentUsage
	if remaining < 0 {
		return 0
	}
	return remaining
}

// IsNearExpiry checks if the license is near expiry (within specified duration)
func (l *License) IsNearExpiry(threshold time.Duration) bool {
	return time.Until(l.ExpiresAt) <= threshold
}

// GetTimeUntilExpiry returns the duration until license expires
func (l *License) GetTimeUntilExpiry() time.Duration {
	return time.Until(l.ExpiresAt)
}
