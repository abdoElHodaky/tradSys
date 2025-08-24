package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"go.uber.org/zap"
)

// Manager manages exchange connector plugins
type Manager struct {
	logger           *zap.Logger
	registry         *Registry
	loader           *Loader
	metrics          *lazy.AdaptiveMetrics
	contextPropagator *lazy.ContextPropagator
	connectorCache   map[string]connectors.ExchangeConnector
	cacheMu          sync.RWMutex
	coreVersion      string
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
		logger:           logger,
		registry:         registry,
		loader:           loader,
		metrics:          metrics,
		contextPropagator: contextPropagator,
		connectorCache:   make(map[string]connectors.ExchangeConnector),
		coreVersion:      coreVersion,
	}
}

// RegisterPlugin registers a plugin
func (m *Manager) RegisterPlugin(plugin ExchangeConnectorPlugin) error {
	return m.registry.RegisterPlugin(plugin)
}

// GetPlugin gets a plugin by exchange name
func (m *Manager) GetPlugin(exchangeName string) (ExchangeConnectorPlugin, error) {
	return m.registry.GetPlugin(exchangeName)
}

// ListPlugins lists all plugins
func (m *Manager) ListPlugins() []ExchangeConnectorPlugin {
	return m.registry.ListPlugins()
}

// UnregisterPlugin unregisters a plugin
func (m *Manager) UnregisterPlugin(exchangeName string) error {
	// Remove from cache
	m.cacheMu.Lock()
	delete(m.connectorCache, exchangeName)
	m.cacheMu.Unlock()
	
	return m.registry.UnregisterPlugin(exchangeName)
}

// LoadPlugin loads a plugin from a file
func (m *Manager) LoadPlugin(filePath string) (ExchangeConnectorPlugin, error) {
	return m.loader.LoadPlugin(filePath)
}

// LoadPlugins loads all plugins from a directory
func (m *Manager) LoadPlugins(dirPath string) ([]ExchangeConnectorPlugin, error) {
	return m.loader.LoadPlugins(dirPath)
}

// LoadAllPlugins loads all plugins from all configured directories
func (m *Manager) LoadAllPlugins() ([]ExchangeConnectorPlugin, error) {
	return m.loader.LoadAllPlugins()
}

// CreateConnector creates an exchange connector
func (m *Manager) CreateConnector(
	exchangeName string,
	config connectors.ExchangeConfig,
	logger *zap.Logger,
) (connectors.ExchangeConnector, error) {
	// Check cache first
	m.cacheMu.RLock()
	connector, ok := m.connectorCache[exchangeName]
	m.cacheMu.RUnlock()
	
	if ok {
		return connector, nil
	}
	
	// Create new connector
	connector, err := m.registry.CreateConnector(exchangeName, config, logger)
	if err != nil {
		return nil, err
	}
	
	// Add to cache
	m.cacheMu.Lock()
	m.connectorCache[exchangeName] = connector
	m.cacheMu.Unlock()
	
	return connector, nil
}

// CreateConnectorWithContext creates an exchange connector with context
func (m *Manager) CreateConnectorWithContext(
	ctx context.Context,
	exchangeName string,
	config connectors.ExchangeConfig,
	logger *zap.Logger,
) (connectors.ExchangeConnector, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		connector connectors.ExchangeConnector
		err       error
	})
	
	// Create connector in a goroutine
	go func() {
		connector, err := m.CreateConnector(exchangeName, config, logger)
		resultCh <- struct {
			connector connectors.ExchangeConnector
			err       error
		}{connector, err}
	}()
	
	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.connector, result.err
	case <-ctx.Done():
		return nil, fmt.Errorf("connector creation canceled: %w", ctx.Err())
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
	
	m.logger.Info("Loaded exchange connector plugins", zap.Int("count", len(plugins)))
	
	// Initialize all plugins
	return m.registry.Initialize(ctx)
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	// Clear connector cache
	m.cacheMu.Lock()
	m.connectorCache = make(map[string]connectors.ExchangeConnector)
	m.cacheMu.Unlock()
	
	// Shutdown all plugins
	return m.registry.Shutdown(ctx)
}

// LoadAllPluginsWithContext loads all plugins with context
func (m *Manager) LoadAllPluginsWithContext(ctx context.Context) ([]ExchangeConnectorPlugin, error) {
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

// GetConnectorCache gets the connector cache
func (m *Manager) GetConnectorCache() map[string]connectors.ExchangeConnector {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	
	cache := make(map[string]connectors.ExchangeConnector, len(m.connectorCache))
	for k, v := range m.connectorCache {
		cache[k] = v
	}
	
	return cache
}

// ClearConnectorCache clears the connector cache
func (m *Manager) ClearConnectorCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	m.connectorCache = make(map[string]connectors.ExchangeConnector)
}

// RemoveFromConnectorCache removes a connector from the cache
func (m *Manager) RemoveFromConnectorCache(exchangeName string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	delete(m.connectorCache, exchangeName)
}

