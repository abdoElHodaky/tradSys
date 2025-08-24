package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/market_data"
	"go.uber.org/zap"
)

// Manager manages indicator plugins
type Manager struct {
	logger            *zap.Logger
	registry          *Registry
	loader            *Loader
	metrics           *lazy.AdaptiveMetrics
	contextPropagator *lazy.ContextPropagator
	indicatorCache    map[string]market_data.Indicator
	cacheMu           sync.RWMutex
	coreVersion       string
}

// NewManager creates a new manager
func NewManager(
	logger *zap.Logger,
	metrics *lazy.AdaptiveMetrics,
	contextPropagator *lazy.ContextPropagator,
	pluginDirs []string,
	coreVersion string,
) *Manager {
	registry := NewRegistry(logger, coreVersion)
	loader := NewLoader(logger, registry, pluginDirs)
	
	return &Manager{
		logger:            logger,
		registry:          registry,
		loader:            loader,
		metrics:           metrics,
		contextPropagator: contextPropagator,
		indicatorCache:    make(map[string]market_data.Indicator),
		coreVersion:       coreVersion,
	}
}

// RegisterPlugin registers a plugin
func (m *Manager) RegisterPlugin(plugin IndicatorPlugin) error {
	return m.registry.RegisterPlugin(plugin)
}

// GetPlugin gets a plugin by indicator type
func (m *Manager) GetPlugin(indicatorType string) (IndicatorPlugin, error) {
	return m.registry.GetPlugin(indicatorType)
}

// ListPlugins lists all plugins
func (m *Manager) ListPlugins() []IndicatorPlugin {
	return m.registry.ListPlugins()
}

// UnregisterPlugin unregisters a plugin
func (m *Manager) UnregisterPlugin(indicatorType string) error {
	// Remove from cache
	m.cacheMu.Lock()
	delete(m.indicatorCache, indicatorType)
	m.cacheMu.Unlock()
	
	return m.registry.UnregisterPlugin(indicatorType)
}

// LoadPlugin loads a plugin from a file
func (m *Manager) LoadPlugin(filePath string) (IndicatorPlugin, error) {
	return m.loader.LoadPlugin(filePath)
}

// LoadPlugins loads all plugins from a directory
func (m *Manager) LoadPlugins(dirPath string) ([]IndicatorPlugin, error) {
	return m.loader.LoadPlugins(dirPath)
}

// LoadAllPlugins loads all plugins from all configured directories
func (m *Manager) LoadAllPlugins() ([]IndicatorPlugin, error) {
	return m.loader.LoadAllPlugins()
}

// CreateIndicator creates an indicator
func (m *Manager) CreateIndicator(
	indicatorType string,
	config market_data.IndicatorConfig,
	logger *zap.Logger,
) (market_data.Indicator, error) {
	// Check cache first
	m.cacheMu.RLock()
	indicator, ok := m.indicatorCache[indicatorType]
	m.cacheMu.RUnlock()
	
	if ok {
		return indicator, nil
	}
	
	// Create new indicator
	indicator, err := m.registry.CreateIndicator(indicatorType, config, logger)
	if err != nil {
		return nil, err
	}
	
	// Add to cache
	m.cacheMu.Lock()
	m.indicatorCache[indicatorType] = indicator
	m.cacheMu.Unlock()
	
	return indicator, nil
}

// CreateIndicatorWithContext creates an indicator with context
func (m *Manager) CreateIndicatorWithContext(
	ctx context.Context,
	indicatorType string,
	config market_data.IndicatorConfig,
	logger *zap.Logger,
) (market_data.Indicator, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		indicator market_data.Indicator
		err       error
	})
	
	// Create indicator in a goroutine
	go func() {
		indicator, err := m.CreateIndicator(indicatorType, config, logger)
		resultCh <- struct {
			indicator market_data.Indicator
			err       error
		}{indicator, err}
	}()
	
	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.indicator, result.err
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// Initialize initializes all plugins
func (m *Manager) Initialize(ctx context.Context) error {
	// Load all plugins
	plugins, err := m.LoadAllPluginsWithContext(ctx)
	if err != nil {
		m.logger.Error("Failed to load all plugins", zap.Error(err))
		// Continue with initialization of already loaded plugins
	}
	
	m.logger.Info("Loaded indicator plugins", zap.Int("count", len(plugins)))
	
	// Initialize all plugins
	return m.registry.Initialize(ctx)
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	// Clear indicator cache
	m.cacheMu.Lock()
	m.indicatorCache = make(map[string]market_data.Indicator)
	m.cacheMu.Unlock()
	
	// Shutdown all plugins
	return m.registry.Shutdown(ctx)
}

// LoadAllPluginsWithContext loads all plugins with context
func (m *Manager) LoadAllPluginsWithContext(ctx context.Context) ([]IndicatorPlugin, error) {
	return m.loader.LoadAllPluginsWithContext(ctx)
}

// AddPluginDirectory adds a plugin directory
func (m *Manager) AddPluginDirectory(dirPath string) {
	m.loader.AddPluginDirectory(dirPath)
}

// RemovePluginDirectory removes a plugin directory
func (m *Manager) RemovePluginDirectory(dirPath string) {
	m.loader.RemovePluginDirectory(dirPath)
}

// GetPluginDirectories gets the plugin directories
func (m *Manager) GetPluginDirectories() []string {
	return m.loader.GetPluginDirectories()
}

// StartBackgroundScanner starts a background scanner for new plugins
func (m *Manager) StartBackgroundScanner(ctx context.Context, scanInterval time.Duration) error {
	return m.loader.StartBackgroundScanner(ctx, scanInterval)
}

// StopBackgroundScanner stops the background scanner
func (m *Manager) StopBackgroundScanner() {
	m.loader.StopBackgroundScanner()
}

// SetCoreVersion sets the core version
func (m *Manager) SetCoreVersion(version string) {
	m.coreVersion = version
	m.registry.SetCoreVersion(version)
}

// GetCoreVersion gets the core version
func (m *Manager) GetCoreVersion() string {
	return m.coreVersion
}

// ValidateAllPlugins validates all plugins
func (m *Manager) ValidateAllPlugins() error {
	return m.registry.ValidateAllPlugins()
}

// GetIndicatorCache gets the indicator cache
func (m *Manager) GetIndicatorCache() map[string]market_data.Indicator {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	
	cache := make(map[string]market_data.Indicator, len(m.indicatorCache))
	for k, v := range m.indicatorCache {
		cache[k] = v
	}
	
	return cache
}

// ClearIndicatorCache clears the indicator cache
func (m *Manager) ClearIndicatorCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	m.indicatorCache = make(map[string]market_data.Indicator)
}

// RemoveFromIndicatorCache removes an indicator from the cache
func (m *Manager) RemoveFromIndicatorCache(indicatorType string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	delete(m.indicatorCache, indicatorType)
}

