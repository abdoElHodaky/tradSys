package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/exchange/connectors"
	"go.uber.org/zap"
)

// ConnectorRegistry manages exchange connectors
type ConnectorRegistry struct {
	logger     *zap.Logger
	loader     *PluginLoader
	connectors map[string]connectors.ExchangeConnector
	mu         sync.RWMutex
	// Track connector usage for cleanup
	lastUsed map[string]time.Time
	// Track connector health
	health map[string]bool
}

// NewConnectorRegistry creates a new connector registry
func NewConnectorRegistry(pluginDir string, logger *zap.Logger) *ConnectorRegistry {
	return &ConnectorRegistry{
		logger:     logger,
		loader:     NewPluginLoader(pluginDir, logger),
		connectors: make(map[string]connectors.ExchangeConnector),
		lastUsed:   make(map[string]time.Time),
		health:     make(map[string]bool),
	}
}

// Initialize initializes the connector registry
func (r *ConnectorRegistry) Initialize() error {
	// Load plugins
	if err := r.loader.LoadPlugins(); err != nil {
		return fmt.Errorf("failed to load plugins: %w", err)
	}
	
	// Start the cleanup goroutine
	go r.cleanupUnusedConnectors()
	
	return nil
}

// GetConnector gets or creates an exchange connector
func (r *ConnectorRegistry) GetConnector(exchangeName string, config connectors.ExchangeConfig) (connectors.ExchangeConnector, error) {
	r.mu.RLock()
	connector, exists := r.connectors[exchangeName]
	if exists {
		// Update last used time
		r.lastUsed[exchangeName] = time.Now()
		r.mu.RUnlock()
		return connector, nil
	}
	r.mu.RUnlock()
	
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check again in case another goroutine created it while we were waiting for the lock
	if connector, exists = r.connectors[exchangeName]; exists {
		r.lastUsed[exchangeName] = time.Now()
		return connector, nil
	}
	
	// Get the plugin
	plugin, ok := r.loader.GetPlugin(exchangeName)
	if !ok {
		return nil, fmt.Errorf("exchange connector plugin not found: %s", exchangeName)
	}
	
	// Create the connector with panic recovery
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("panic while creating connector: %v", r)
				r.logger.Error("Panic while creating connector",
					zap.String("exchange", exchangeName),
					zap.Any("panic", r))
			}
		}()
		
		// Create the connector
		connector, err = plugin.CreateConnector(config, r.logger)
	}()
	
	if err != nil {
		r.health[exchangeName] = false
		return nil, fmt.Errorf("failed to create exchange connector: %w", err)
	}
	
	// Initialize the connector
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	if err := connector.Initialize(ctx); err != nil {
		r.health[exchangeName] = false
		return nil, fmt.Errorf("failed to initialize exchange connector: %w", err)
	}
	
	// Store the connector
	r.connectors[exchangeName] = connector
	r.lastUsed[exchangeName] = time.Now()
	r.health[exchangeName] = true
	
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

// CloseConnector closes a connector and removes it from the registry
func (r *ConnectorRegistry) CloseConnector(exchangeName string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	connector, exists := r.connectors[exchangeName]
	if !exists {
		return fmt.Errorf("connector for exchange %s not found", exchangeName)
	}
	
	// Close the connector
	if err := connector.Close(); err != nil {
		r.logger.Warn("Error closing connector",
			zap.String("exchange", exchangeName),
			zap.Error(err))
	}
	
	// Remove the connector from the registry
	delete(r.connectors, exchangeName)
	delete(r.lastUsed, exchangeName)
	delete(r.health, exchangeName)
	
	r.logger.Info("Closed exchange connector", zap.String("exchange", exchangeName))
	
	return nil
}

// GetConnectorHealth gets the health status of a connector
func (r *ConnectorRegistry) GetConnectorHealth(exchangeName string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.health[exchangeName]
}

// SetConnectorHealth sets the health status of a connector
func (r *ConnectorRegistry) SetConnectorHealth(exchangeName string, healthy bool) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.health[exchangeName] = healthy
}

// cleanupUnusedConnectors periodically cleans up unused connectors
func (r *ConnectorRegistry) cleanupUnusedConnectors() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		r.mu.Lock()
		
		now := time.Now()
		for exchangeName, lastUsed := range r.lastUsed {
			// If the connector hasn't been used in the last hour, close it
			if now.Sub(lastUsed) > time.Hour {
				connector, exists := r.connectors[exchangeName]
				if exists {
					r.logger.Info("Closing unused connector",
						zap.String("exchange", exchangeName),
						zap.Duration("unused_for", now.Sub(lastUsed)))
					
					// Close the connector
					if err := connector.Close(); err != nil {
						r.logger.Warn("Error closing connector",
							zap.String("exchange", exchangeName),
							zap.Error(err))
					}
					
					// Remove the connector from the registry
					delete(r.connectors, exchangeName)
					delete(r.lastUsed, exchangeName)
					delete(r.health, exchangeName)
				}
			}
		}
		
		r.mu.Unlock()
	}
}

// Shutdown shuts down the registry and all connectors
func (r *ConnectorRegistry) Shutdown() error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	var errs []error
	
	// Close all connectors
	for exchangeName, connector := range r.connectors {
		r.logger.Info("Closing connector during shutdown", zap.String("exchange", exchangeName))
		
		if err := connector.Close(); err != nil {
			r.logger.Warn("Error closing connector",
				zap.String("exchange", exchangeName),
				zap.Error(err))
			errs = append(errs, fmt.Errorf("failed to close connector %s: %w", exchangeName, err))
		}
	}
	
	// Clear the registry
	r.connectors = make(map[string]connectors.ExchangeConnector)
	r.lastUsed = make(map[string]time.Time)
	r.health = make(map[string]bool)
	
	if len(errs) > 0 {
		return fmt.Errorf("errors shutting down registry: %v", errs)
	}
	
	return nil
}

