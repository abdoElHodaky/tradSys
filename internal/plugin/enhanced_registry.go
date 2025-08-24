package plugin

import (
	"errors"
	"fmt"
	"sync"

	"github.com/Masterminds/semver/v3"
	"go.uber.org/zap"
)

// PluginInfo contains information about a plugin
type PluginInfo struct {
	// Name is the name of the plugin
	Name string `json:"name"`
	
	// Version is the version of the plugin
	Version string `json:"version"`
	
	// Author is the author of the plugin
	Author string `json:"author"`
	
	// Description is a description of the plugin
	Description string `json:"description"`
	
	// Type is the type of plugin
	Type string `json:"type"`
	
	// MinCoreVersion is the minimum core version required by this plugin
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version supported by this plugin
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins that this plugin depends on
	Dependencies []PluginDependency `json:"dependencies"`
}

// PluginDependency represents a dependency on another plugin
type PluginDependency struct {
	// Name is the name of the plugin
	Name string `json:"name"`
	
	// Type is the type of plugin
	Type string `json:"type"`
	
	// VersionConstraint is the version constraint for the dependency
	VersionConstraint string `json:"version_constraint"`
}

// EnhancedPluginRegistry is an enhanced registry for plugins that addresses
// conflicts and bottlenecks identified in the analysis.
type EnhancedPluginRegistry struct {
	logger       *zap.Logger
	plugins      map[string]map[string]interface{} // type -> name -> plugin
	pluginInfo   map[string]map[string]*PluginInfo // type -> name -> info
	dependencies map[string]map[string][]PluginDependency // type -> name -> dependencies
	mu           sync.RWMutex
	coreVersion  string
}

// NewEnhancedPluginRegistry creates a new enhanced plugin registry
func NewEnhancedPluginRegistry(logger *zap.Logger, coreVersion string) *EnhancedPluginRegistry {
	return &EnhancedPluginRegistry{
		logger:       logger,
		plugins:      make(map[string]map[string]interface{}),
		pluginInfo:   make(map[string]map[string]*PluginInfo),
		dependencies: make(map[string]map[string][]PluginDependency),
		coreVersion:  coreVersion,
	}
}

// RegisterPlugin registers a plugin
func (r *EnhancedPluginRegistry) RegisterPlugin(pluginType string, name string, plugin interface{}, info *PluginInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if the plugin type exists
	if _, ok := r.plugins[pluginType]; !ok {
		r.plugins[pluginType] = make(map[string]interface{})
		r.pluginInfo[pluginType] = make(map[string]*PluginInfo)
		r.dependencies[pluginType] = make(map[string][]PluginDependency)
	}
	
	// Check if the plugin already exists
	if _, ok := r.plugins[pluginType][name]; ok {
		return fmt.Errorf("plugin already registered: %s/%s", pluginType, name)
	}
	
	// Validate core version compatibility
	if err := r.validateCoreVersionCompatibility(info); err != nil {
		return fmt.Errorf("core version compatibility check failed: %w", err)
	}
	
	// Register the plugin
	r.plugins[pluginType][name] = plugin
	r.pluginInfo[pluginType][name] = info
	
	// Store dependencies
	if info.Dependencies != nil {
		r.dependencies[pluginType][name] = info.Dependencies
	}
	
	r.logger.Info("Registered plugin",
		zap.String("type", pluginType),
		zap.String("name", name),
		zap.String("version", info.Version))
	
	return nil
}

// UnregisterPlugin unregisters a plugin
func (r *EnhancedPluginRegistry) UnregisterPlugin(pluginType string, name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	// Check if the plugin type exists
	if _, ok := r.plugins[pluginType]; !ok {
		return fmt.Errorf("plugin type not found: %s", pluginType)
	}
	
	// Check if the plugin exists
	if _, ok := r.plugins[pluginType][name]; !ok {
		return fmt.Errorf("plugin not found: %s/%s", pluginType, name)
	}
	
	// Check if other plugins depend on this one
	for depType, plugins := range r.dependencies {
		for depName, deps := range plugins {
			for _, dep := range deps {
				if dep.Type == pluginType && dep.Name == name {
					return fmt.Errorf("cannot unregister plugin %s/%s: plugin %s/%s depends on it",
						pluginType, name, depType, depName)
				}
			}
		}
	}
	
	// Unregister the plugin
	delete(r.plugins[pluginType], name)
	delete(r.pluginInfo[pluginType], name)
	delete(r.dependencies[pluginType], name)
	
	r.logger.Info("Unregistered plugin",
		zap.String("type", pluginType),
		zap.String("name", name))
	
	return nil
}

// GetPlugin gets a plugin
func (r *EnhancedPluginRegistry) GetPlugin(pluginType string, name string) (interface{}, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if the plugin type exists
	if _, ok := r.plugins[pluginType]; !ok {
		return nil, fmt.Errorf("plugin type not found: %s", pluginType)
	}
	
	// Check if the plugin exists
	plugin, ok := r.plugins[pluginType][name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s/%s", pluginType, name)
	}
	
	return plugin, nil
}

// GetPluginInfo gets information about a plugin
func (r *EnhancedPluginRegistry) GetPluginInfo(pluginType string, name string) (*PluginInfo, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if the plugin type exists
	if _, ok := r.pluginInfo[pluginType]; !ok {
		return nil, fmt.Errorf("plugin type not found: %s", pluginType)
	}
	
	// Check if the plugin exists
	info, ok := r.pluginInfo[pluginType][name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s/%s", pluginType, name)
	}
	
	return info, nil
}

// ListPlugins lists all plugins of a type
func (r *EnhancedPluginRegistry) ListPlugins(pluginType string) []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check if the plugin type exists
	if _, ok := r.plugins[pluginType]; !ok {
		return []string{}
	}
	
	// Get the plugin names
	names := make([]string, 0, len(r.plugins[pluginType]))
	for name := range r.plugins[pluginType] {
		names = append(names, name)
	}
	
	return names
}

// ListPluginTypes lists all plugin types
func (r *EnhancedPluginRegistry) ListPluginTypes() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Get the plugin types
	types := make([]string, 0, len(r.plugins))
	for pluginType := range r.plugins {
		types = append(types, pluginType)
	}
	
	return types
}

// ValidateDependencies validates that all plugin dependencies are satisfied
func (r *EnhancedPluginRegistry) ValidateDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Check each plugin's dependencies
	for pluginType, plugins := range r.dependencies {
		for pluginName, deps := range plugins {
			for _, dep := range deps {
				// Check if the dependency exists
				depInfo, err := r.getPluginInfoNoLock(dep.Type, dep.Name)
				if err != nil {
					return fmt.Errorf("dependency not found for plugin %s/%s: %w",
						pluginType, pluginName, err)
				}
				
				// Check version constraint if specified
				if dep.VersionConstraint != "" {
					if err := r.validateVersionConstraint(depInfo.Version, dep.VersionConstraint); err != nil {
						return fmt.Errorf("version constraint not satisfied for plugin %s/%s dependency on %s/%s: %w",
							pluginType, pluginName, dep.Type, dep.Name, err)
					}
				}
			}
		}
	}
	
	return nil
}

// getPluginInfoNoLock gets plugin info without locking (for internal use)
func (r *EnhancedPluginRegistry) getPluginInfoNoLock(pluginType string, name string) (*PluginInfo, error) {
	// Check if the plugin type exists
	if _, ok := r.pluginInfo[pluginType]; !ok {
		return nil, fmt.Errorf("plugin type not found: %s", pluginType)
	}
	
	// Check if the plugin exists
	info, ok := r.pluginInfo[pluginType][name]
	if !ok {
		return nil, fmt.Errorf("plugin not found: %s/%s", pluginType, name)
	}
	
	return info, nil
}

// validateCoreVersionCompatibility validates that a plugin is compatible with the core version
func (r *EnhancedPluginRegistry) validateCoreVersionCompatibility(info *PluginInfo) error {
	// If no constraints are specified, assume compatibility
	if info.MinCoreVersion == "" && info.MaxCoreVersion == "" {
		return nil
	}
	
	// Parse core version
	coreVer, err := semver.NewVersion(r.coreVersion)
	if err != nil {
		return fmt.Errorf("invalid core version: %w", err)
	}
	
	// Check minimum core version
	if info.MinCoreVersion != "" {
		minVer, err := semver.NewVersion(info.MinCoreVersion)
		if err != nil {
			return fmt.Errorf("invalid minimum core version: %w", err)
		}
		
		if coreVer.LessThan(minVer) {
			return fmt.Errorf("core version %s is less than minimum required version %s",
				r.coreVersion, info.MinCoreVersion)
		}
	}
	
	// Check maximum core version
	if info.MaxCoreVersion != "" {
		maxVer, err := semver.NewVersion(info.MaxCoreVersion)
		if err != nil {
			return fmt.Errorf("invalid maximum core version: %w", err)
		}
		
		if coreVer.GreaterThan(maxVer) {
			return fmt.Errorf("core version %s is greater than maximum supported version %s",
				r.coreVersion, info.MaxCoreVersion)
		}
	}
	
	return nil
}

// validateVersionConstraint validates that a version satisfies a constraint
func (r *EnhancedPluginRegistry) validateVersionConstraint(version string, constraint string) error {
	// Parse version
	ver, err := semver.NewVersion(version)
	if err != nil {
		return fmt.Errorf("invalid version: %w", err)
	}
	
	// Parse constraint
	c, err := semver.NewConstraint(constraint)
	if err != nil {
		return fmt.Errorf("invalid version constraint: %w", err)
	}
	
	// Check constraint
	if !c.Check(ver) {
		return fmt.Errorf("version %s does not satisfy constraint %s", version, constraint)
	}
	
	return nil
}

// GetDependencyGraph returns a dependency graph for all plugins
func (r *EnhancedPluginRegistry) GetDependencyGraph() map[string]map[string][]string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create the graph
	graph := make(map[string]map[string][]string)
	
	// Add all plugins to the graph
	for pluginType, plugins := range r.plugins {
		graph[pluginType] = make(map[string][]string)
		for name := range plugins {
			graph[pluginType][name] = make([]string, 0)
		}
	}
	
	// Add dependencies
	for pluginType, plugins := range r.dependencies {
		for pluginName, deps := range plugins {
			for _, dep := range deps {
				depKey := fmt.Sprintf("%s/%s", dep.Type, dep.Name)
				graph[pluginType][pluginName] = append(graph[pluginType][pluginName], depKey)
			}
		}
	}
	
	return graph
}

// DetectCircularDependencies detects circular dependencies in the plugin registry
func (r *EnhancedPluginRegistry) DetectCircularDependencies() error {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	// Create a map to track visited plugins
	visited := make(map[string]bool)
	
	// Create a map to track plugins in the current path
	inPath := make(map[string]bool)
	
	// Check each plugin
	for pluginType, plugins := range r.plugins {
		for name := range plugins {
			pluginKey := fmt.Sprintf("%s/%s", pluginType, name)
			
			// Skip if already visited
			if visited[pluginKey] {
				continue
			}
			
			// Check for circular dependencies
			if err := r.detectCircularDependenciesDFS(pluginType, name, visited, inPath); err != nil {
				return err
			}
		}
	}
	
	return nil
}

// detectCircularDependenciesDFS performs a depth-first search to detect circular dependencies
func (r *EnhancedPluginRegistry) detectCircularDependenciesDFS(
	pluginType string,
	name string,
	visited map[string]bool,
	inPath map[string]bool,
) error {
	pluginKey := fmt.Sprintf("%s/%s", pluginType, name)
	
	// Check if already in path (circular dependency)
	if inPath[pluginKey] {
		return fmt.Errorf("circular dependency detected: %s", pluginKey)
	}
	
	// Mark as visited and in path
	visited[pluginKey] = true
	inPath[pluginKey] = true
	
	// Check dependencies
	if deps, ok := r.dependencies[pluginType][name]; ok {
		for _, dep := range deps {
			depKey := fmt.Sprintf("%s/%s", dep.Type, dep.Name)
			
			// Skip if already visited and not in path
			if visited[depKey] && !inPath[depKey] {
				continue
			}
			
			// Check dependency
			if err := r.detectCircularDependenciesDFS(dep.Type, dep.Name, visited, inPath); err != nil {
				return err
			}
		}
	}
	
	// Remove from path
	inPath[pluginKey] = false
	
	return nil
}

// GetPluginsByDependency returns all plugins that depend on a specific plugin
func (r *EnhancedPluginRegistry) GetPluginsByDependency(depType string, depName string) []struct {
	Type string
	Name string
} {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	result := make([]struct {
		Type string
		Name string
	}, 0)
	
	// Check each plugin's dependencies
	for pluginType, plugins := range r.dependencies {
		for pluginName, deps := range plugins {
			for _, dep := range deps {
				if dep.Type == depType && dep.Name == depName {
					result = append(result, struct {
						Type string
						Name string
					}{
						Type: pluginType,
						Name: pluginName,
					})
					break
				}
			}
		}
	}
	
	return result
}

// SetCoreVersion sets the core version
func (r *EnhancedPluginRegistry) SetCoreVersion(version string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	
	r.coreVersion = version
}

// GetCoreVersion gets the core version
func (r *EnhancedPluginRegistry) GetCoreVersion() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	return r.coreVersion
}

// ValidateAllPlugins validates all plugins in the registry
func (r *EnhancedPluginRegistry) ValidateAllPlugins() error {
	// Validate dependencies
	if err := r.ValidateDependencies(); err != nil {
		return err
	}
	
	// Detect circular dependencies
	if err := r.DetectCircularDependencies(); err != nil {
		return err
	}
	
	// Validate core version compatibility for all plugins
	r.mu.RLock()
	defer r.mu.RUnlock()
	
	for pluginType, plugins := range r.pluginInfo {
		for name, info := range plugins {
			if err := r.validateCoreVersionCompatibility(info); err != nil {
				return fmt.Errorf("core version compatibility check failed for plugin %s/%s: %w",
					pluginType, name, err)
			}
		}
	}
	
	return nil
}

// ErrPluginNotFound is returned when a plugin is not found
var ErrPluginNotFound = errors.New("plugin not found")

// ErrPluginTypeNotFound is returned when a plugin type is not found
var ErrPluginTypeNotFound = errors.New("plugin type not found")

// ErrPluginAlreadyRegistered is returned when a plugin is already registered
var ErrPluginAlreadyRegistered = errors.New("plugin already registered")

// ErrIncompatibleCoreVersion is returned when a plugin is incompatible with the core version
var ErrIncompatibleCoreVersion = errors.New("incompatible core version")

// ErrCircularDependency is returned when a circular dependency is detected
var ErrCircularDependency = errors.New("circular dependency detected")

// ErrDependencyNotSatisfied is returned when a dependency is not satisfied
var ErrDependencyNotSatisfied = errors.New("dependency not satisfied")

