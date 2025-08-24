package plugin

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ValidatorPluginRegistry manages risk validator plugins
type ValidatorPluginRegistry struct {
	logger  *zap.Logger
	plugins map[string]RiskValidatorPlugin
	fileMap map[string]string // Maps file paths to validator types
	mu      sync.RWMutex
}

// NewValidatorPluginRegistry creates a new validator plugin registry
func NewValidatorPluginRegistry(logger *zap.Logger) *ValidatorPluginRegistry {
	return &ValidatorPluginRegistry{
		logger:  logger,
		plugins: make(map[string]RiskValidatorPlugin),
		fileMap: make(map[string]string),
	}
}

// RegisterPlugin registers a validator plugin
func (r *ValidatorPluginRegistry) RegisterPlugin(validatorType string, plugin RiskValidatorPlugin) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.plugins[validatorType] = plugin
	
	// If the plugin has a file path, map it
	if wrapper, ok := plugin.(*ValidatorPluginWrapper); ok {
		r.fileMap[wrapper.FilePath] = validatorType
	}
	
	r.logger.Info("Registered validator plugin",
		zap.String("validator_type", validatorType))
}

// UnregisterPlugin unregisters a validator plugin
func (r *ValidatorPluginRegistry) UnregisterPlugin(validatorType string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Find the file path if it exists
	var filePath string
	for path, vType := range r.fileMap {
		if vType == validatorType {
			filePath = path
			break
		}
	}
	
	// Remove from plugins
	delete(r.plugins, validatorType)
	
	// Remove from file map if found
	if filePath != "" {
		delete(r.fileMap, filePath)
	}
	
	r.logger.Info("Unregistered validator plugin",
		zap.String("validator_type", validatorType))
}

// UnregisterPluginByFile unregisters a validator plugin by file path
func (r *ValidatorPluginRegistry) UnregisterPluginByFile(filePath string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Find the validator type
	validatorType, ok := r.fileMap[filePath]
	if !ok {
		return
	}
	
	// Remove from plugins
	delete(r.plugins, validatorType)
	
	// Remove from file map
	delete(r.fileMap, filePath)
	
	r.logger.Info("Unregistered validator plugin by file",
		zap.String("file", filePath),
		zap.String("validator_type", validatorType))
}

// GetPlugin gets a validator plugin by type
func (r *ValidatorPluginRegistry) GetPlugin(validatorType string) (RiskValidatorPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, ok := r.plugins[validatorType]
	return plugin, ok
}

// CreateValidator creates a validator instance from a plugin
func (r *ValidatorPluginRegistry) CreateValidator(validatorType string, config ValidatorConfig, logger *zap.Logger) (RiskValidator, error) {
	r.mu.RLock()
	plugin, ok := r.plugins[validatorType]
	r.mu.RUnlock()
	
	if !ok {
		return nil, fmt.Errorf("validator plugin not found: %s", validatorType)
	}
	
	return plugin.CreateValidator(config, logger)
}

// ListPlugins lists all registered plugins
func (r *ValidatorPluginRegistry) ListPlugins() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugins := make([]string, 0, len(r.plugins))
	for validatorType := range r.plugins {
		plugins = append(plugins, validatorType)
	}
	
	return plugins
}

// GetPluginInfo gets information about a plugin
func (r *ValidatorPluginRegistry) GetPluginInfo(validatorType string) (*PluginInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	plugin, ok := r.plugins[validatorType]
	if !ok {
		return nil, false
	}
	
	// If it's a wrapper, return the info
	if wrapper, ok := plugin.(*ValidatorPluginWrapper); ok {
		return wrapper.Info, true
	}
	
	return nil, false
}

// GetPluginByFile gets a plugin by file path
func (r *ValidatorPluginRegistry) GetPluginByFile(filePath string) (RiskValidatorPlugin, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	validatorType, ok := r.fileMap[filePath]
	if !ok {
		return nil, false
	}
	
	plugin, ok := r.plugins[validatorType]
	return plugin, ok
}

