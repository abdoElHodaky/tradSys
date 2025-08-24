package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators"
	"go.uber.org/zap"
)

// IndicatorRegistry manages technical indicators
type IndicatorRegistry struct {
	logger     *zap.Logger
	loader     *PluginLoader
	indicators map[string]indicators.Indicator
	mu         sync.RWMutex
	// Track indicator usage for cleanup
	lastUsed map[string]time.Time
	// Track indicator health
	health map[string]bool
	// Cache key format: indicatorName-paramHash
	cacheKeys map[string]string
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry(pluginDir string, logger *zap.Logger) *IndicatorRegistry {
	return &IndicatorRegistry{
		logger:     logger,
		loader:     NewPluginLoader(pluginDir, logger),
		indicators: make(map[string]indicators.Indicator),
		lastUsed:   make(map[string]time.Time),
		health:     make(map[string]bool),
		cacheKeys:  make(map[string]string),
	}
}

// Initialize initializes the indicator registry
func (r *IndicatorRegistry) Initialize() error {
	// Load plugins
	if err := r.loader.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}
	
	// Start the cleanup goroutine
	go r.cleanupUnusedIndicators()
	
	return nil
}

// generateCacheKey generates a cache key for an indicator with parameters
func (r *IndicatorRegistry) generateCacheKey(indicatorName string, params indicators.IndicatorParams) string {
	// Simple hash of parameters
	paramHash := fmt.Sprintf("%d-%s-%.2f-%v", params.Period, params.Source, params.Alpha, params.CustomParams)
	return fmt.Sprintf("%s-%s", indicatorName, paramHash)
}

// GetIndicator gets or creates a technical indicator
func (r *IndicatorRegistry) GetIndicator(indicatorName string, params indicators.IndicatorParams) (indicators.Indicator, error) {
	// Generate cache key
	cacheKey := r.generateCacheKey(indicatorName, params)
	
	r.mu.RLock()
	indicator, exists := r.indicators[cacheKey]
	if exists {
		// Update last used time
		r.lastUsed[cacheKey] = time.Now()
		r.mu.RUnlock()
		return indicator, nil
	}
	r.mu.RUnlock()
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if indicator, exists = r.indicators[cacheKey]; exists {
		r.lastUsed[cacheKey] = time.Now()
		return indicator, nil
	}
	
	// Get the plugin
	plugin, ok := r.loader.GetPlugin(indicatorName)
	if !ok {
		return nil, fmt.Errorf("technical indicator plugin not found: %s", indicatorName)
	}
	
	// Create the indicator with panic recovery
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic while creating indicator: %v", r)
				r.logger.Error("Panic while creating indicator",
					zap.String("indicator", indicatorName),
					zap.Any("panic", r))
			}
		}()
		
		// Create the indicator
		indicator, err = plugin.CreateIndicator(params, r.logger)
	}()
	
	if err != nil {
		r.health[indicatorName] = false
		return nil, fmt.Errorf("failed to create technical indicator: %w", err)
	}
	
	// Store the indicator
	r.indicators[cacheKey] = indicator
	r.lastUsed[cacheKey] = time.Now()
	r.health[indicatorName] = true
	r.cacheKeys[cacheKey] = indicatorName
	
	r.logger.Info("Created technical indicator", 
		zap.String("indicator", indicatorName),
		zap.Any("params", params))
	
	return indicator, nil
}

// ListAvailableIndicators lists available indicators
func (r *IndicatorRegistry) ListAvailableIndicators() []string {
	plugins := r.loader.GetAvailablePlugins()
	
	indicators := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		indicators = append(indicators, plugin.IndicatorName)
	}
	
	return indicators
}

// GetIndicatorInfo gets information about an indicator
func (r *IndicatorRegistry) GetIndicatorInfo(indicatorName string) (*PluginInfo, bool) {
	plugin, ok := r.loader.GetPlugin(indicatorName)
	if !ok {
		return nil, false
	}
	
	info := plugin.(*pluginWrapper).info
	return info, true
}

// RemoveIndicator removes an indicator from the registry
func (r *IndicatorRegistry) RemoveIndicator(cacheKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Remove the indicator from the registry
	delete(r.indicators, cacheKey)
	delete(r.lastUsed, cacheKey)
	delete(r.cacheKeys, cacheKey)
	
	r.logger.Info("Removed technical indicator", zap.String("cache_key", cacheKey))
}

// GetIndicatorHealth gets the health status of an indicator
func (r *IndicatorRegistry) GetIndicatorHealth(indicatorName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.health[indicatorName]
}

// SetIndicatorHealth sets the health status of an indicator
func (r *IndicatorRegistry) SetIndicatorHealth(indicatorName string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.health[indicatorName] = healthy
}

// cleanupUnusedIndicators periodically cleans up unused indicators
func (r *IndicatorRegistry) cleanupUnusedIndicators() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		
		now := time.Now()
		for cacheKey, lastUsed := range r.lastUsed {
			// If the indicator hasn't been used in the last hour, remove it
			if now.Sub(lastUsed) > time.Hour {
				if _, exists := r.indicators[cacheKey]; exists {
					indicatorName := r.cacheKeys[cacheKey]
					r.logger.Info("Removing unused indicator",
						zap.String("indicator", indicatorName),
						zap.String("cache_key", cacheKey),
						zap.Duration("unused_for", now.Sub(lastUsed)))
					
					// Remove the indicator from the registry
					delete(r.indicators, cacheKey)
					delete(r.lastUsed, cacheKey)
					delete(r.cacheKeys, cacheKey)
				}
			}
		}
		
		r.mu.Unlock()
	}
}

// Shutdown shuts down the registry and all indicators
func (r *IndicatorRegistry) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Clear the registry
	r.indicators = make(map[string]indicators.Indicator)
	r.lastUsed = make(map[string]time.Time)
	r.health = make(map[string]bool)
	r.cacheKeys = make(map[string]string)
	
	return nil
}

