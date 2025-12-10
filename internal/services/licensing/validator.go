// Package licensing provides high-performance license validation
package licensing

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Validator provides high-performance license validation with caching
type Validator struct {
	cache       CacheInterface
	db          DatabaseInterface
	rateLimiter RateLimiterInterface
	metrics     MetricsInterface
	config      *ValidatorConfig
	mu          sync.RWMutex
}

// ValidatorConfig holds configuration for the license validator
type ValidatorConfig struct {
	CacheTTL          time.Duration
	ValidationTimeout time.Duration
	MaxCacheSize      int
	EnableMetrics     bool
	EnableRateLimit   bool
}

// CacheInterface defines the caching interface
type CacheInterface interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
	Exists(ctx context.Context, key string) bool
}

// DatabaseInterface defines the database interface for license operations
type DatabaseInterface interface {
	GetLicense(ctx context.Context, userID string) (*License, error)
	UpdateLicense(ctx context.Context, license *License) error
	GetUsage(ctx context.Context, userID, usageType string) (int64, error)
	IncrementUsage(ctx context.Context, userID, usageType string, amount int64) error
}

// RateLimiterInterface defines the rate limiting interface
type RateLimiterInterface interface {
	Allow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error)
	GetUsage(ctx context.Context, key string, window time.Duration) (int64, error)
}

// MetricsInterface defines the metrics collection interface
type MetricsInterface interface {
	RecordValidationLatency(duration time.Duration)
	RecordCacheHit()
	RecordCacheMiss()
	RecordValidationResult(valid bool)
	RecordQuotaExceeded(feature string)
}

// NewValidator creates a new license validator
func NewValidator(cache CacheInterface, db DatabaseInterface, rateLimiter RateLimiterInterface, metrics MetricsInterface, config *ValidatorConfig) *Validator {
	if config == nil {
		config = GetDefaultValidatorConfig()
	}

	return &Validator{
		cache:       cache,
		db:          db,
		rateLimiter: rateLimiter,
		metrics:     metrics,
		config:      config,
	}
}

// ValidateFeature validates if a user has access to a specific feature
func (v *Validator) ValidateFeature(ctx context.Context, userID string, feature LicenseFeature) (*ValidationResult, error) {
	start := time.Now()
	defer func() {
		if v.config.EnableMetrics && v.metrics != nil {
			v.metrics.RecordValidationLatency(time.Since(start))
		}
	}()

	// Create validation context with timeout
	validationCtx, cancel := context.WithTimeout(ctx, v.config.ValidationTimeout)
	defer cancel()

	// Check cache first for sub-millisecond response
	cacheKey := fmt.Sprintf("license_validation:%s:%s", userID, feature)
	if cached, err := v.getCachedValidation(validationCtx, cacheKey); err == nil && cached != nil {
		if v.config.EnableMetrics && v.metrics != nil {
			v.metrics.RecordCacheHit()
		}
		return cached, nil
	}

	if v.config.EnableMetrics && v.metrics != nil {
		v.metrics.RecordCacheMiss()
	}

	// Fetch license from database
	license, err := v.db.GetLicense(validationCtx, userID)
	if err != nil {
		result := &ValidationResult{
			Valid:  false,
			Reason: "license_not_found",
		}
		if v.config.EnableMetrics && v.metrics != nil {
			v.metrics.RecordValidationResult(false)
		}
		return result, err
	}

	// Validate license
	result := v.validateLicense(validationCtx, license, feature)

	// Cache result for fast subsequent access
	if err := v.cacheValidation(validationCtx, cacheKey, result); err != nil {
		// Log error but don't fail validation
		fmt.Printf("Failed to cache validation result: %v\n", err)
	}

	if v.config.EnableMetrics && v.metrics != nil {
		v.metrics.RecordValidationResult(result.Valid)
		if !result.Valid && result.Reason == "quota_exceeded" {
			v.metrics.RecordQuotaExceeded(string(feature))
		}
	}

	return result, nil
}

// ValidateQuota validates if a user is within their usage quota
func (v *Validator) ValidateQuota(ctx context.Context, userID, usageType string, amount int64) (*ValidationResult, error) {
	// Get license
	license, err := v.db.GetLicense(ctx, userID)
	if err != nil {
		return &ValidationResult{Valid: false, Reason: "license_not_found"}, err
	}

	// Check if license is active
	if !license.IsValid() {
		return &ValidationResult{Valid: false, Reason: "license_inactive"}, nil
	}

	// Get quota for usage type
	quota := license.GetQuota(usageType)
	if quota == -1 {
		// Unlimited quota
		return &ValidationResult{Valid: true, QuotaLimit: -1}, nil
	}

	// Get current usage
	currentUsage, err := v.db.GetUsage(ctx, userID, usageType)
	if err != nil {
		return &ValidationResult{Valid: false, Reason: "usage_check_failed"}, err
	}

	// Check if adding the amount would exceed quota
	if currentUsage+amount > quota {
		return &ValidationResult{
			Valid:      false,
			Reason:     "quota_exceeded",
			QuotaUsed:  currentUsage,
			QuotaLimit: quota,
		}, nil
	}

	return &ValidationResult{
		Valid:      true,
		QuotaUsed:  currentUsage,
		QuotaLimit: quota,
	}, nil
}

// ValidateRateLimit validates if a user is within their rate limit
func (v *Validator) ValidateRateLimit(ctx context.Context, userID, usageType string) (*ValidationResult, error) {
	if !v.config.EnableRateLimit || v.rateLimiter == nil {
		return &ValidationResult{Valid: true}, nil
	}

	// Get license
	license, err := v.db.GetLicense(ctx, userID)
	if err != nil {
		return &ValidationResult{Valid: false, Reason: "license_not_found"}, err
	}

	// Get rate limit for usage type
	rateLimit := license.GetRateLimit(usageType)
	if rateLimit == -1 {
		// Unlimited rate
		return &ValidationResult{Valid: true}, nil
	}

	// Check rate limit (per minute)
	rateLimitKey := fmt.Sprintf("rate_limit:%s:%s", userID, usageType)
	allowed, err := v.rateLimiter.Allow(ctx, rateLimitKey, rateLimit, time.Minute)
	if err != nil {
		return &ValidationResult{Valid: false, Reason: "rate_limit_check_failed"}, err
	}

	if !allowed {
		return &ValidationResult{Valid: false, Reason: "rate_limit_exceeded"}, nil
	}

	return &ValidationResult{Valid: true}, nil
}

// RecordUsage records usage for a user
func (v *Validator) RecordUsage(ctx context.Context, userID, usageType string, amount int64) error {
	return v.db.IncrementUsage(ctx, userID, usageType, amount)
}

// GetUsageStats returns usage statistics for a user
func (v *Validator) GetUsageStats(ctx context.Context, userID, usageType string) (*UsageStats, error) {
	// Get license
	license, err := v.db.GetLicense(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Get current usage
	usage, err := v.db.GetUsage(ctx, userID, usageType)
	if err != nil {
		return nil, err
	}

	quota := license.GetQuota(usageType)
	percentage := float64(0)
	if quota > 0 {
		percentage = float64(usage) / float64(quota) * 100
	}

	return &UsageStats{
		UserID:      userID,
		UsageType:   usageType,
		Used:        usage,
		Quota:       quota,
		Percentage:  percentage,
		ResetTime:   getNextResetTime(),
		LastUpdated: time.Now(),
	}, nil
}

// validateLicense performs the actual license validation
func (v *Validator) validateLicense(ctx context.Context, license *License, feature LicenseFeature) *ValidationResult {
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
func (v *Validator) getCachedValidation(ctx context.Context, key string) (*ValidationResult, error) {
	if v.cache == nil {
		return nil, fmt.Errorf("cache not available")
	}

	cached, err := v.cache.Get(ctx, key)
	if err != nil {
		return nil, err
	}

	if result, ok := cached.(*ValidationResult); ok {
		return result, nil
	}

	return nil, fmt.Errorf("invalid cached data type")
}

// cacheValidation stores validation result in cache
func (v *Validator) cacheValidation(ctx context.Context, key string, result *ValidationResult) error {
	if v.cache == nil {
		return nil
	}

	return v.cache.Set(ctx, key, result, v.config.CacheTTL)
}

// getNextResetTime returns the next quota reset time (daily reset)
func getNextResetTime() time.Time {
	now := time.Now()
	tomorrow := now.AddDate(0, 0, 1)
	return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, tomorrow.Location())
}

// GetDefaultValidatorConfig returns default validator configuration
func GetDefaultValidatorConfig() *ValidatorConfig {
	return &ValidatorConfig{
		CacheTTL:          5 * time.Minute,
		ValidationTimeout: 5 * time.Second,
		MaxCacheSize:      10000,
		EnableMetrics:     true,
		EnableRateLimit:   true,
	}
}
