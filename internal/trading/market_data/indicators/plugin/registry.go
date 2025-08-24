package plugin

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/trading/market_data/indicators"
	"go.uber.org/zap"
)

// IndicatorRegistry manages technical indicators
type IndicatorRegistry struct {
	logger     *zap.Logger
	loader     *PluginLoader
	indicators map[string]indicators.Indicator
	mu         sync.RWMutex
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry(pluginDir string, logger *zap.Logger) *IndicatorRegistry {
	return &IndicatorRegistry{
		logger:     logger,
		loader:     NewPluginLoader(pluginDir, logger),
		indicators: make(map[string]indicators.Indicator),
	}
}

// Initialize initializes the indicator registry
func (r *IndicatorRegistry) Initialize() error {
	// Load plugins
	if err := r.loader.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}
	
	return nil
}

// GetIndicator gets or creates a technical indicator
func (r *IndicatorRegistry) GetIndicator(indicatorName string, params indicators.IndicatorParams) (indicators.Indicator, error) {
	// Create a cache key
	cacheKey := fmt.Sprintf("%s-%v", indicatorName, params)
	
	r.mu.RLock()
	indicator, exists := r.indicators[cacheKey]
	r.mu.RUnlock()
	
	if exists {
		return indicator, nil
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if indicator, exists = r.indicators[cacheKey]; exists {
		return indicator, nil
	}
	
	// Get the plugin
	plugin, ok := r.loader.GetPlugin(indicatorName)
	if !ok {
		return nil, fmt.Errorf("technical indicator plugin not found: %s", indicatorName)
	}
	
	// Create the indicator
	indicator, err := plugin.CreateIndicator(params, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create technical indicator: %w", err)
	}
	
	// Store the indicator
	r.indicators[cacheKey] = indicator
	
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

