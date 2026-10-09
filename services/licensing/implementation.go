// Package licensing provides high-performance license validation
package licensing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// LicenseValidator validates licenses for asset access
type LicenseValidator struct {
	cache       CacheInterface
	db          DatabaseInterface
	rateLimiter RateLimiterInterface
	metrics     MetricsInterface
	config      *ValidatorConfig
	mu          sync.RWMutex
}

// NewLicenseValidator creates a new license validator
func NewLicenseValidator(cache CacheInterface, db DatabaseInterface, rateLimiter RateLimiterInterface, metrics MetricsInterface, config *ValidatorConfig) *LicenseValidator {
	if config == nil {
		config = GetDefaultValidatorConfig()
	}

	return &LicenseValidator{
		cache:       cache,
		db:          db,
		rateLimiter: rateLimiter,
		metrics:     metrics,
		config:      config,
	}
}

// Validate validates a license for a specific feature
func (lv *LicenseValidator) Validate(ctx context.Context, userID string, feature LicenseFeature) (*ValidationResult, error) {
	start := time.Now()
	defer func() {
		if lv.config.EnableMetrics && lv.metrics != nil {
			lv.metrics.RecordValidationLatency(time.Since(start))
		}
	}()

	// Create validation context with timeout
	validationCtx, cancel := context.WithTimeout(ctx, lv.config.ValidationTimeout)
	defer cancel()

	// Check cache first for sub-millisecond response
	cacheKey := fmt.Sprintf("license_validation:%s:%s", userID, feature)
	if cached, err := lv.getCachedValidation(validationCtx, cacheKey); err == nil && cached != nil {
		if lv.config.EnableMetrics && lv.metrics != nil {
			lv.metrics.RecordCacheHit()
		}
		return cached, nil
	}

	if lv.config.EnableMetrics && lv.metrics != nil {
		lv.metrics.RecordCacheMiss()
	}

	// Fetch license from database
	license, err := lv.db.GetLicense(validationCtx, userID)
	if err != nil {
		result := &ValidationResult{
			Valid:  false,
			Reason: "license_not_found",
		}
		if lv.config.EnableMetrics && lv.metrics != nil {
			lv.metrics.RecordValidationResult(false)
		}
		return result, err
	}

	// Validate license
	result := lv.validateLicense(validationCtx, license, feature)

	// Cache result for fast subsequent access
	if err := lv.cacheValidation(validationCtx, cacheKey, result); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to cache validation result: %v\n", err)
	}

	if lv.config.EnableMetrics && lv.metrics != nil {
		lv.metrics.RecordValidationResult(result.Valid)
		if !result.Valid && result.Reason == "quota_exceeded" {
			lv.metrics.RecordQuotaExceeded(string(feature))
		}
	}

	return result, nil
}

// validateLicense performs the actual license validation
func (lv *LicenseValidator) validateLicense(ctx context.Context, license *License, feature LicenseFeature) *ValidationResult {
	// Check if license is active
	if !license.IsActive {
		return &ValidationResult{
			Valid:     false,
			Reason:    "license_inactive",
			ExpiresAt: license.ExpiresAt,
		}
	}

	// Check if license has expired
	if license.IsExpired() {
		return &ValidationResult{
			Valid:     false,
			Reason:    "license_expired",
			ExpiresAt: license.ExpiresAt,
		}
	}

	// Check if license includes the feature
	if !license.HasFeature(feature) {
		return &ValidationResult{
			Valid:         false,
			Reason:        "feature_not_licensed",
			ExpiresAt:     license.ExpiresAt,
			RemainingTime: int64(time.Until(license.ExpiresAt).Seconds()),
		}
	}

	return &ValidationResult{
		Valid:         true,
		ExpiresAt:     license.ExpiresAt,
		RemainingTime: int64(time.Until(license.ExpiresAt).Seconds()),
	}
}

// getCachedValidation retrieves validation result from cache
func (lv *LicenseValidator) getCachedValidation(ctx context.Context, key string) (*ValidationResult, error) {
	if lv.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	cached, err := lv.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if result, ok := cached.(*ValidationResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid cached data type")
}

// cacheValidation stores validation result in cache
func (lv *LicenseValidator) cacheValidation(ctx context.Context, key string, result *ValidationResult) error {
	if lv.cache == nil {
		return nil
	}

	return lv.cache.Set(ctx, key, result, lv.config.CacheTTL)
}

// QuotaManager manages usage quotas for licenses
type QuotaManager struct {
	db    QuotaDatabase
	cache QuotaCache
	mu    sync.RWMutex
}

// QuotaDatabase interface for quota operations
type QuotaDatabase interface {
	GetUsage(ctx context.Context, userID, usageType string) (int64, error)
	IncrementUsage(ctx context.Context, userID, usageType string, amount int64) error
}

// QuotaCache interface for quota caching
type QuotaCache interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

// NewQuotaManager creates a new quota manager
func NewQuotaManager(db QuotaDatabase, cache QuotaCache) *QuotaManager {
	return &QuotaManager{
		db:    db,
		cache: cache,
	}
}

// CheckQuota checks if usage is within quota limits
func (qm *QuotaManager) CheckQuota(ctx context.Context, userID, usageType string, amount int64, quota int64) bool {
	if quota == -1 {
		return true // Unlimited quota
	}

	currentUsage, err := qm.db.GetUsage(ctx, userID, usageType)
	if err != nil {
		return false
	}

	return currentUsage+amount <= quota
}

// RecordUsage records usage for quota tracking
func (qm *QuotaManager) RecordUsage(ctx context.Context, userID, usageType string, amount int64) error {
	return qm.db.IncrementUsage(ctx, userID, usageType, amount)
}

// BillingEngine handles billing calculations
type BillingEngine struct {
	plans map[LicenseTier]*BillingPlan
	mu    sync.RWMutex
}

// NewBillingEngine creates a new billing engine
func NewBillingEngine() *BillingEngine {
	return &BillingEngine{
		plans: make(map[LicenseTier]*BillingPlan),
	}
}

// CalculateBilling calculates billing for a license
func (be *BillingEngine) CalculateBilling(license *License, usage map[string]int64) *Bill {
	plan, exists := BillingPlans[license.Tier]
	if !exists {
		return &Bill{}
	}

	bill := &Bill{
		ID:            fmt.Sprintf("bill_%d", time.Now().UnixNano()),
		UserID:        license.UserID,
		BillingPeriod: time.Now().UTC().Format("2006-01"),
		StartDate:     time.Now().AddDate(0, 0, -30),
		EndDate:       time.Now(),
		BaseFee:       plan.BaseFee,
		UsageFees:     make(map[string]float64),
		OverageFees:   make(map[string]float64),
		Status:        "pending",
	}

	// Calculate usage fees
	for usageType, amount := range usage {
		if rate, exists := plan.UsageRates[usageType]; exists {
			bill.UsageFees[usageType] = rate * float64(amount)
		}

		// Calculate overage fees
		quota := license.GetQuota(usageType)
		if quota != -1 && amount > quota {
			if overageRate, exists := plan.OverageRates[usageType]; exists {
				bill.OverageFees[usageType] = overageRate * float64(amount-quota)
			}
		}
	}

	// Calculate total
	total := bill.BaseFee
	for _, fee := range bill.UsageFees {
		total += fee
	}
	for _, fee := range bill.OverageFees {
		total += fee
	}
	bill.Total = total
	bill.Currency = plan.Currency
	bill.DueDate = time.Now().Add(30 * 24 * time.Hour)
	bill.CreatedAt = time.Now()

	return bill
}

// UsageTracker tracks usage for billing
type UsageTracker struct {
	mu sync.RWMutex
}

// UsageEntry represents a usage entry
type UsageEntry struct {
	UserID    string
	UsageType string
	Amount    int64
	Timestamp time.Time
}

// NewUsageTracker creates a new usage tracker
func NewUsageTracker() *UsageTracker {
	return &UsageTracker{}
}

// RecordUsage records usage
func (ut *UsageTracker) RecordUsage(userID, usageType string, amount int64) {
	// Implementation would persist to database
}
