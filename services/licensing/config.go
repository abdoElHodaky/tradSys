// Package licensing provides license configuration and tier definitions
package licensing

import (
	"fmt"
	"time"
)

// LicenseConfigs defines the configuration for each license tier
var LicenseConfigs = map[LicenseTier]*LicenseConfig{
	BASIC: {
		Tier: BASIC,
		Features: []LicenseFeature{
			BASIC_TRADING, EGX_ACCESS, BASIC_ASSETS, 
			BASIC_ANALYTICS, REST_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        1000,
			"api_calls_per_minute":  100,
			"websocket_connections": 10,
			"market_data_symbols":   50,
			"portfolio_assets":      50,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":     10,
			"api_calls_per_second":  5,
			"market_data_requests":  20,
		},
		MaxUsers:  5,
		MaxAssets: 50,
		MaxOrders: 1000,
	},
	PROFESSIONAL: {
		Tier: PROFESSIONAL,
		Features: []LicenseFeature{
			BASIC_TRADING, ADVANCED_TRADING, EGX_ACCESS, ADX_ACCESS,
			BASIC_ASSETS, ADVANCED_ASSETS, BASIC_ANALYTICS, 
			ADVANCED_ANALYTICS, REST_API, WEBSOCKET_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        10000,
			"api_calls_per_minute":  1000,
			"websocket_connections": 100,
			"market_data_symbols":   500,
			"portfolio_assets":      500,
			"reports_per_month":     100,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":     100,
			"api_calls_per_second":  50,
			"market_data_requests":  200,
		},
		MaxUsers:  25,
		MaxAssets: 500,
		MaxOrders: 10000,
	},
	ENTERPRISE: {
		Tier: ENTERPRISE,
		Features: []LicenseFeature{
			BASIC_TRADING, ADVANCED_TRADING, HFT_TRADING,
			EGX_ACCESS, ADX_ACCESS, MULTI_EXCHANGE,
			BASIC_ASSETS, ADVANCED_ASSETS, CRYPTO_ASSETS,
			BASIC_ANALYTICS, ADVANCED_ANALYTICS, REAL_TIME_ANALYTICS,
			REST_API, WEBSOCKET_API, THIRD_PARTY_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        -1, // unlimited
			"api_calls_per_minute":  -1, // unlimited
			"websocket_connections": -1, // unlimited
			"market_data_symbols":   -1, // unlimited
			"portfolio_assets":      -1, // unlimited
			"reports_per_month":     -1, // unlimited
		},
		RateLimits: map[string]int64{
			"orders_per_minute":     -1, // unlimited
			"api_calls_per_second":  -1, // unlimited
			"market_data_requests":  -1, // unlimited
		},
		MaxUsers:  -1, // unlimited
		MaxAssets: -1, // unlimited
		MaxOrders: -1, // unlimited
	},
	ISLAMIC: {
		Tier: ISLAMIC,
		Features: []LicenseFeature{
			BASIC_TRADING, ADVANCED_TRADING, EGX_ACCESS, ADX_ACCESS,
			BASIC_ASSETS, ISLAMIC_ASSETS, BASIC_ANALYTICS, 
			ADVANCED_ANALYTICS, SHARIA_COMPLIANCE, ZAKAT_CALCULATION,
			HALAL_SCREENING, REST_API, WEBSOCKET_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        5000,
			"api_calls_per_minute":  500,
			"websocket_connections": 50,
			"market_data_symbols":   300,
			"portfolio_assets":      300,
			"sharia_screenings":     1000,
			"zakat_calculations":    100,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":     50,
			"api_calls_per_second":  25,
			"market_data_requests":  100,
			"sharia_requests":       10,
		},
		MaxUsers:  15,
		MaxAssets: 300,
		MaxOrders: 5000,
	},
}

// BillingPlans defines the billing configuration for each license tier
var BillingPlans = map[LicenseTier]*BillingPlan{
	BASIC: {
		TierID:      "basic",
		Name:        "Basic Plan",
		Description: "Essential trading features for individual traders",
		BaseFee:     99.00,
		Currency:    "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       0.01,  // $0.01 per order
			"api_calls_per_minute": 0.001, // $0.001 per API call
			"market_data_symbols":  0.10,  // $0.10 per symbol
		},
		OverageRates: map[string]float64{
			"orders_per_day":       0.02,  // $0.02 per order over quota
			"api_calls_per_minute": 0.002, // $0.002 per API call over quota
			"market_data_symbols":  0.20,  // $0.20 per symbol over quota
		},
		Features: LicenseConfigs[BASIC].Features,
		Quotas:   LicenseConfigs[BASIC].Quotas,
		IsActive: true,
	},
	PROFESSIONAL: {
		TierID:      "professional",
		Name:        "Professional Plan",
		Description: "Advanced trading features for professional traders",
		BaseFee:     499.00,
		Currency:    "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       0.005, // $0.005 per order
			"api_calls_per_minute": 0.0005, // $0.0005 per API call
			"market_data_symbols":  0.05,  // $0.05 per symbol
			"reports_per_month":    1.00,  // $1.00 per report
		},
		OverageRates: map[string]float64{
			"orders_per_day":       0.01,  // $0.01 per order over quota
			"api_calls_per_minute": 0.001, // $0.001 per API call over quota
			"market_data_symbols":  0.10,  // $0.10 per symbol over quota
			"reports_per_month":    2.00,  // $2.00 per report over quota
		},
		Features: LicenseConfigs[PROFESSIONAL].Features,
		Quotas:   LicenseConfigs[PROFESSIONAL].Quotas,
		IsActive: true,
	},
	ENTERPRISE: {
		TierID:      "enterprise",
		Name:        "Enterprise Plan",
		Description: "Unlimited features for institutional trading",
		BaseFee:     2999.00,
		Currency:    "USD",
		BillingCycle: "monthly",
		UsageRates:  map[string]float64{}, // No usage rates - unlimited
		OverageRates: map[string]float64{}, // No overage rates - unlimited
		Features:    LicenseConfigs[ENTERPRISE].Features,
		Quotas:      LicenseConfigs[ENTERPRISE].Quotas,
		IsActive:    true,
	},
	ISLAMIC: {
		TierID:      "islamic",
		Name:        "Islamic Finance Plan",
		Description: "Sharia-compliant trading with Islamic finance features",
		BaseFee:     299.00,
		Currency:    "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       0.008, // $0.008 per order
			"api_calls_per_minute": 0.0008, // $0.0008 per API call
			"market_data_symbols":  0.08,  // $0.08 per symbol
			"sharia_screenings":    0.50,  // $0.50 per screening
			"zakat_calculations":   2.00,  // $2.00 per calculation
		},
		OverageRates: map[string]float64{
			"orders_per_day":       0.016, // $0.016 per order over quota
			"api_calls_per_minute": 0.0016, // $0.0016 per API call over quota
			"market_data_symbols":  0.16,  // $0.16 per symbol over quota
			"sharia_screenings":    1.00,  // $1.00 per screening over quota
			"zakat_calculations":   4.00,  // $4.00 per calculation over quota
		},
		Features: LicenseConfigs[ISLAMIC].Features,
		Quotas:   LicenseConfigs[ISLAMIC].Quotas,
		IsActive: true,
	},
}

// GetLicenseConfig returns the configuration for a specific license tier
func GetLicenseConfig(tier LicenseTier) (*LicenseConfig, bool) {
	config, exists := LicenseConfigs[tier]
	return config, exists
}

// GetBillingPlan returns the billing plan for a specific license tier
func GetBillingPlan(tier LicenseTier) (*BillingPlan, bool) {
	plan, exists := BillingPlans[tier]
	return plan, exists
}

// GetAllLicenseTiers returns all available license tiers
func GetAllLicenseTiers() []LicenseTier {
	return []LicenseTier{BASIC, PROFESSIONAL, ENTERPRISE, ISLAMIC}
}

// GetTierFeatures returns all features for a specific tier
func GetTierFeatures(tier LicenseTier) []LicenseFeature {
	if config, exists := LicenseConfigs[tier]; exists {
		return config.Features
	}
	return []LicenseFeature{}
}

// HasFeature checks if a tier includes a specific feature
func HasFeature(tier LicenseTier, feature LicenseFeature) bool {
	features := GetTierFeatures(tier)
	for _, f := range features {
		if f == feature {
			return true
		}
	}
	return false
}

// GetTierQuota returns the quota for a specific usage type and tier
func GetTierQuota(tier LicenseTier, usageType string) int64 {
	if config, exists := LicenseConfigs[tier]; exists {
		if quota, exists := config.Quotas[usageType]; exists {
			return quota
		}
	}
	return 0
}

// GetTierRateLimit returns the rate limit for a specific usage type and tier
func GetTierRateLimit(tier LicenseTier, usageType string) int64 {
	if config, exists := LicenseConfigs[tier]; exists {
		if limit, exists := config.RateLimits[usageType]; exists {
			return limit
		}
	}
	return 0
}

// CreateLicense creates a new license for a user
func CreateLicense(userID, organizationID string, tier LicenseTier, duration time.Duration) (*License, error) {
	config, exists := GetLicenseConfig(tier)
	if !exists {
		return nil, ErrInvalidLicenseTier
	}
	
	now := time.Now()
	license := &License{
		ID:             generateLicenseID(),
		UserID:         userID,
		OrganizationID: organizationID,
		Tier:           tier,
		Features:       config.Features,
		Quotas:         config.Quotas,
		RateLimits:     config.RateLimits,
		IssuedAt:       now,
		ExpiresAt:      now.Add(duration),
		IsActive:       true,
		MaxUsers:       config.MaxUsers,
		MaxAssets:      config.MaxAssets,
		MaxOrders:      config.MaxOrders,
		Metadata:       make(map[string]interface{}),
	}
	
	return license, nil
}

// UpgradeLicense upgrades a license to a higher tier
func UpgradeLicense(license *License, newTier LicenseTier) error {
	newConfig, exists := GetLicenseConfig(newTier)
	if !exists {
		return ErrInvalidLicenseTier
	}
	
	// Update license with new tier configuration
	license.Tier = newTier
	license.Features = newConfig.Features
	license.Quotas = newConfig.Quotas
	license.RateLimits = newConfig.RateLimits
	license.MaxUsers = newConfig.MaxUsers
	license.MaxAssets = newConfig.MaxAssets
	license.MaxOrders = newConfig.MaxOrders
	
	return nil
}

// ExtendLicense extends the expiration date of a license
func ExtendLicense(license *License, duration time.Duration) {
	license.ExpiresAt = license.ExpiresAt.Add(duration)
}

// DeactivateLicense deactivates a license
func DeactivateLicense(license *License) {
	license.Active = false
}

// ReactivateLicense reactivates a license
func ReactivateLicense(license *License) {
	license.Active = true
}

// generateLicenseID generates a unique license ID
func generateLicenseID() string {
	// This would typically use a more sophisticated ID generation
	// For now, using a simple timestamp-based approach
	return fmt.Sprintf("LIC_%d", time.Now().UnixNano())
}

// ValidateLicenseConfig validates a license configuration
func ValidateLicenseConfig(config *LicenseConfig) error {
	if !config.Tier.IsValid() {
		return ErrInvalidLicenseTier
	}
	
	for _, feature := range config.Features {
		if !feature.IsValid() {
			return &LicenseError{
				Code:    "INVALID_FEATURE",
				Message: fmt.Sprintf("invalid feature: %s", feature),
			}
		}
	}
	
	return nil
}

// GetRecommendedTier returns the recommended tier based on usage patterns
func GetRecommendedTier(usage map[string]int64) LicenseTier {
	// Simple recommendation logic based on usage
	ordersPerDay := usage["orders_per_day"]
	apiCallsPerMinute := usage["api_calls_per_minute"]
	
	// Enterprise tier for high usage
	if ordersPerDay > 5000 || apiCallsPerMinute > 500 {
		return ENTERPRISE
	}
	
	// Professional tier for medium usage
	if ordersPerDay > 1000 || apiCallsPerMinute > 100 {
		return PROFESSIONAL
	}
	
	// Basic tier for low usage
	return BASIC
}
