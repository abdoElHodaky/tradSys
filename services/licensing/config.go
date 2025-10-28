// Package licensing provides license configuration and tier definitions.
package licensing

import (
	"fmt"
	"time"
)

// License configuration constants.
const (
	// Basic tier limits.
	BasicMaxSymbols         = 50
	BasicMaxAssets          = 50
	BasicMaxUsers           = 1
	BasicMaxOrders          = 1000
	BasicAPICallsPerSecond  = 5
	BasicMarketDataRequests = 20

	// Professional tier limits.
	ProfessionalMaxSymbols         = 500
	ProfessionalMaxAssets          = 500
	ProfessionalMaxUsers           = 25
	ProfessionalMaxOrders          = 10000
	ProfessionalOrdersPerDay       = 10000
	ProfessionalAPICallsPerSecond  = 50
	ProfessionalMarketDataRequests = 200

	// Enterprise tier limits.
	EnterpriseMaxSymbols        = 300
	EnterpriseMaxAssets         = 300
	EnterpriseMaxUsers          = 15
	EnterpriseMaxOrders         = 5000
	EnterpriseOrdersPerDay      = 5000
	EnterpriseAPICallsPerMinute = 500
	EnterpriseWebsocketConns    = 50
	EnterpriseOrdersPerMinute   = 50
	EnterpriseAPICallsPerSecond = 25

	// Billing constants.
	BasicBaseFee        = 99.00
	ProfessionalBaseFee = 499.00
	EnterpriseBaseFee   = 2999.00
	IslamicBaseFee      = 299.00

	// Usage fees.
	BasicOrderFee      = 0.01
	BasicAPICallFee    = 0.001
	BasicSymbolFee     = 0.10
	BasicOverageFactor = 2.0

	ProfessionalOrderFee      = 0.005
	ProfessionalAPICallFee    = 0.0005
	ProfessionalSymbolFee     = 0.05
	ProfessionalReportFee     = 1.00
	ProfessionalOverageFactor = 2.0

	IslamicOrderFee      = 0.008
	IslamicAPICallFee    = 0.0008
	IslamicSymbolFee     = 0.08
	IslamicShariaFee     = 0.50
	IslamicZakatFee      = 2.00
	IslamicOverageFactor = 2.0

	// Cache configuration.
	DefaultMaxCacheSize = 10000
)

// LicenseConfigs defines the configuration for each license tier.
var LicenseConfigs = map[LicenseTier]*LicenseConfig{
	BASIC: {
		Tier: BASIC,
		Features: []LicenseFeature{
			BASIC_TRADING, EGX_ACCESS, BASIC_ASSETS,
			BASIC_ANALYTICS, REST_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        BasicMaxOrders,
			"api_calls_per_minute":  100,
			"websocket_connections": 10,
			"market_data_symbols":   BasicMaxSymbols,
			"portfolio_assets":      BasicMaxAssets,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":    10,
			"api_calls_per_second": BasicAPICallsPerSecond,
			"market_data_requests": BasicMarketDataRequests,
		},
		MaxUsers:  5,
		MaxAssets: BasicMaxAssets,
		MaxOrders: BasicMaxOrders,
	},
	PROFESSIONAL: {
		Tier: PROFESSIONAL,
		Features: []LicenseFeature{
			BASIC_TRADING, ADVANCED_TRADING, EGX_ACCESS, ADX_ACCESS,
			BASIC_ASSETS, ADVANCED_ASSETS, BASIC_ANALYTICS,
			ADVANCED_ANALYTICS, REST_API, WEBSOCKET_API,
		},
		Quotas: map[string]int64{
			"orders_per_day":        ProfessionalOrdersPerDay,
			"api_calls_per_minute":  1000,
			"websocket_connections": 100,
			"market_data_symbols":   ProfessionalMaxSymbols,
			"portfolio_assets":      ProfessionalMaxAssets,
			"reports_per_month":     100,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":    100,
			"api_calls_per_second": ProfessionalAPICallsPerSecond,
			"market_data_requests": ProfessionalMarketDataRequests,
		},
		MaxUsers:  ProfessionalMaxUsers,
		MaxAssets: ProfessionalMaxAssets,
		MaxOrders: ProfessionalMaxOrders,
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
			"orders_per_day":        -1, // unlimited.
			"api_calls_per_minute":  -1, // unlimited.
			"websocket_connections": -1, // unlimited.
			"market_data_symbols":   -1, // unlimited.
			"portfolio_assets":      -1, // unlimited.
			"reports_per_month":     -1, // unlimited.
		},
		RateLimits: map[string]int64{
			"orders_per_minute":    -1, // unlimited.
			"api_calls_per_second": -1, // unlimited.
			"market_data_requests": -1, // unlimited.
		},
		MaxUsers:  -1, // unlimited.
		MaxAssets: -1, // unlimited.
		MaxOrders: -1, // unlimited.
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
			"orders_per_day":        EnterpriseOrdersPerDay,
			"api_calls_per_minute":  EnterpriseAPICallsPerMinute,
			"websocket_connections": EnterpriseWebsocketConns,
			"market_data_symbols":   EnterpriseMaxSymbols,
			"portfolio_assets":      EnterpriseMaxAssets,
			"sharia_screenings":     1000,
			"zakat_calculations":    100,
		},
		RateLimits: map[string]int64{
			"orders_per_minute":    EnterpriseOrdersPerMinute,
			"api_calls_per_second": EnterpriseAPICallsPerSecond,
			"market_data_requests": 100,
			"sharia_requests":      10,
		},
		MaxUsers:  EnterpriseMaxUsers,
		MaxAssets: EnterpriseMaxAssets,
		MaxOrders: EnterpriseMaxOrders,
	},
}

// BillingPlans defines the billing configuration for each license tier.
var BillingPlans = map[LicenseTier]*BillingPlan{
	BASIC: {
		TierID:       "basic",
		Name:         "Basic Plan",
		Description:  "Essential trading features for individual traders",
		BaseFee:      BasicBaseFee,
		Currency:     "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       BasicOrderFee,   // $0.01 per order
			"api_calls_per_minute": BasicAPICallFee, // $0.001 per API call
			"market_data_symbols":  BasicSymbolFee,  // $0.10 per symbol
		},
		OverageRates: map[string]float64{
			"orders_per_day":       BasicOrderFee * BasicOverageFactor,   // $0.02 per order over quota
			"api_calls_per_minute": BasicAPICallFee * BasicOverageFactor, // $0.002 per API call over quota
			"market_data_symbols":  BasicSymbolFee * BasicOverageFactor,  // $0.20 per symbol over quota
		},
		Features: LicenseConfigs[BASIC].Features,
		Quotas:   LicenseConfigs[BASIC].Quotas,
		IsActive: true,
	},
	PROFESSIONAL: {
		TierID:       "professional",
		Name:         "Professional Plan",
		Description:  "Advanced trading features for professional traders",
		BaseFee:      ProfessionalBaseFee,
		Currency:     "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       ProfessionalOrderFee,   // $0.005 per order
			"api_calls_per_minute": ProfessionalAPICallFee, // $0.0005 per API call
			"market_data_symbols":  ProfessionalSymbolFee,  // $0.05 per symbol
			"reports_per_month":    ProfessionalReportFee,  // $1.00 per report
		},
		OverageRates: map[string]float64{
			"orders_per_day":       ProfessionalOrderFee * ProfessionalOverageFactor,   // $0.01 per order over quota
			"api_calls_per_minute": ProfessionalAPICallFee * ProfessionalOverageFactor, // $0.001 per API call
			"market_data_symbols":  ProfessionalSymbolFee * ProfessionalOverageFactor,  // $0.10 per symbol over quota
			"reports_per_month":    ProfessionalReportFee * ProfessionalOverageFactor,  // $2.00 per report over quota
		},
		Features: LicenseConfigs[PROFESSIONAL].Features,
		Quotas:   LicenseConfigs[PROFESSIONAL].Quotas,
		IsActive: true,
	},
	ENTERPRISE: {
		TierID:       "enterprise",
		Name:         "Enterprise Plan",
		Description:  "Unlimited features for institutional trading",
		BaseFee:      EnterpriseBaseFee,
		Currency:     "USD",
		BillingCycle: "monthly",
		UsageRates:   map[string]float64{}, // No usage rates - unlimited.
		OverageRates: map[string]float64{}, // No overage rates - unlimited.
		Features:     LicenseConfigs[ENTERPRISE].Features,
		Quotas:       LicenseConfigs[ENTERPRISE].Quotas,
		IsActive:     true,
	},
	ISLAMIC: {
		TierID:       "islamic",
		Name:         "Islamic Finance Plan",
		Description:  "Sharia-compliant trading with Islamic finance features",
		BaseFee:      IslamicBaseFee,
		Currency:     "USD",
		BillingCycle: "monthly",
		UsageRates: map[string]float64{
			"orders_per_day":       IslamicOrderFee,   // $0.008 per order
			"api_calls_per_minute": IslamicAPICallFee, // $0.0008 per API call
			"market_data_symbols":  IslamicSymbolFee,  // $0.08 per symbol
			"sharia_screenings":    IslamicShariaFee,  // $0.50 per screening
			"zakat_calculations":   IslamicZakatFee,   // $2.00 per calculation
		},
		OverageRates: map[string]float64{
			"orders_per_day":       IslamicOrderFee * IslamicOverageFactor,   // $0.016 per order over quota
			"api_calls_per_minute": IslamicAPICallFee * IslamicOverageFactor, // $0.0016 per API call over quota
			"market_data_symbols":  IslamicSymbolFee * IslamicOverageFactor,  // $0.16 per symbol over quota
			"sharia_screenings":    IslamicShariaFee * IslamicOverageFactor,  // $1.00 per screening over quota
			"zakat_calculations":   IslamicZakatFee * IslamicOverageFactor,   // $4.00 per calculation over quota
		},
		Features: LicenseConfigs[ISLAMIC].Features,
		Quotas:   LicenseConfigs[ISLAMIC].Quotas,
		IsActive: true,
	},
}

// GetLicenseConfig returns the configuration for a specific license tier.
func GetLicenseConfig(tier LicenseTier) (*LicenseConfig, bool) {
	config, exists := LicenseConfigs[tier]
	return config, exists
}

// GetBillingPlan returns the billing plan for a specific license tier.
func GetBillingPlan(tier LicenseTier) (*BillingPlan, bool) {
	plan, exists := BillingPlans[tier]
	return plan, exists
}

// GetAllLicenseTiers returns all available license tiers.
func GetAllLicenseTiers() []LicenseTier {
	return []LicenseTier{BASIC, PROFESSIONAL, ENTERPRISE, ISLAMIC}
}

// GetTierFeatures returns all features for a specific tier.
func GetTierFeatures(tier LicenseTier) []LicenseFeature {
	if config, exists := LicenseConfigs[tier]; exists {
		return config.Features
	}

	return []LicenseFeature{}
}

// HasFeature checks if a tier includes a specific feature.
func HasFeature(tier LicenseTier, feature LicenseFeature) bool {
	features := GetTierFeatures(tier)
	for _, f := range features {
		if f == feature {
			return true
		}
	}

	return false
}

// GetTierQuota returns the quota for a specific usage type and tier.
func GetTierQuota(tier LicenseTier, usageType string) int64 {
	if config, exists := LicenseConfigs[tier]; exists {
		if quota, exists := config.Quotas[usageType]; exists {
			return quota
		}
	}

	return 0
}

// GetTierRateLimit returns the rate limit for a specific usage type and tier.
func GetTierRateLimit(tier LicenseTier, usageType string) int64 {
	if config, exists := LicenseConfigs[tier]; exists {
		if limit, exists := config.RateLimits[usageType]; exists {
			return limit
		}
	}

	return 0
}

// CreateLicense creates a new license for a user.
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

// UpgradeLicense upgrades a license to a higher tier.
func UpgradeLicense(license *License, newTier LicenseTier) error {
	newConfig, exists := GetLicenseConfig(newTier)
	if !exists {
		return ErrInvalidLicenseTier
	}

	// Update license with new tier configuration.
	license.Tier = newTier
	license.Features = newConfig.Features
	license.Quotas = newConfig.Quotas
	license.RateLimits = newConfig.RateLimits
	license.MaxUsers = newConfig.MaxUsers
	license.MaxAssets = newConfig.MaxAssets
	license.MaxOrders = newConfig.MaxOrders

	return nil
}

// ExtendLicense extends the expiration date of a license.
func ExtendLicense(license *License, duration time.Duration) {
	license.ExpiresAt = license.ExpiresAt.Add(duration)
}

// DeactivateLicense deactivates a license.
func DeactivateLicense(license *License) {
	license.IsActive = false
}

// ReactivateLicense reactivates a license.
func ReactivateLicense(license *License) {
	license.IsActive = true
}

// generateLicenseID generates a unique license ID.
func generateLicenseID() string {
	// This would typically use a more sophisticated ID generation.
	// For now, using a simple timestamp-based approach.
	return fmt.Sprintf("LIC_%d", time.Now().UnixNano())
}

// ValidateLicenseConfig validates a license configuration.
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

// GetRecommendedTier returns the recommended tier based on usage patterns.
func GetRecommendedTier(usage map[string]int64) LicenseTier {
	// Simple recommendation logic based on usage.
	ordersPerDay := usage["orders_per_day"]
	apiCallsPerMinute := usage["api_calls_per_minute"]

	// Enterprise tier for high usage.
	if ordersPerDay > 5000 || apiCallsPerMinute > 500 {
		return ENTERPRISE
	}

	// Professional tier for medium usage.
	if ordersPerDay > 1000 || apiCallsPerMinute > 100 {
		return PROFESSIONAL
	}

	// Basic tier for low usage.
	return BASIC
}
