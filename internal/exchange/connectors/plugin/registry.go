package plugin

import (
	"fmt"
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"go.uber.org/zap"
)

// ConnectorRegistry manages exchange connectors
type ConnectorRegistry struct {
	logger    *zap.Logger
	loader    *PluginLoader
	connectors map[string]connectors.ExchangeConnector
	mu        sync.RWMutex
}

// NewConnectorRegistry creates a new connector registry
func NewConnectorRegistry(pluginDir string, logger *zap.Logger) *ConnectorRegistry {
	return &ConnectorRegistry{
		logger:     logger,
		loader:     NewPluginLoader(pluginDir, logger),
		connectors: make(map[string]connectors.ExchangeConnector),
	}
}

// Initialize initializes the connector registry
func (r *ConnectorRegistry) Initialize() error {
	// Load plugins
	if err := r.loader.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}
	
	return nil
}

// GetConnector gets or creates an exchange connector
func (r *ConnectorRegistry) GetConnector(exchangeName string, config connectors.ExchangeConfig) (connectors.ExchangeConnector, error) {
	r.mu.RLock()
	connector, exists := r.connectors[exchangeName]
	r.mu.RUnlock()
	
	if exists {
		return connector, nil
	}
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if connector, exists = r.connectors[exchangeName]; exists {
		return connector, nil
	}
	
	// Get the plugin
	plugin, ok := r.loader.GetPlugin(exchangeName)
	if !ok {
		return nil, fmt.Errorf("exchange connector plugin not found: %s", exchangeName)
	}
	
	// Create the connector
	connector, err := plugin.CreateConnector(config, r.logger)
	if err != nil {
		return nil, fmt.Errorf("failed to create exchange connector: %w", err)
	}
	
	// Store the connector
	r.connectors[exchangeName] = connector
	
	r.logger.Info("Created exchange connector", zap.String("exchange", exchangeName))
	
	return connector, nil
}

// ListAvailableExchanges lists available exchanges
func (r *ConnectorRegistry) ListAvailableExchanges() []string {
	plugins := r.loader.GetAvailablePlugins()
	
	exchanges := make([]string, 0, len(plugins))
	for _, plugin := range plugins {
		exchanges = append(exchanges, plugin.ExchangeName)
	}
	
	return exchanges
}

// GetConnectorInfo gets information about a connector
func (r *ConnectorRegistry) GetConnectorInfo(exchangeName string) (*PluginInfo, bool) {
	plugin, ok := r.loader.GetPlugin(exchangeName)
	if !ok {
		return nil, false
	}
	
	info := plugin.(*pluginWrapper).info
	return info, true
}

