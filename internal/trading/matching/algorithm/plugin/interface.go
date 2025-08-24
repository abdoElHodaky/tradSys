package plugin

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
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
	
	// AlgorithmType is the type of matching algorithm
	AlgorithmType string `json:"algorithm_type"`
	
	// APIVersion is the version of the matching algorithm API
	APIVersion string `json:"api_version"`
	
	// MinCoreVersion is the minimum core version required by this plugin
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version supported by this plugin
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins that this plugin depends on
	Dependencies []string `json:"dependencies"`
	
	// PerformanceProfile describes the performance characteristics of the algorithm
	PerformanceProfile PerformanceProfile `json:"performance_profile"`
}

// PerformanceProfile describes the performance characteristics of a matching algorithm
type PerformanceProfile struct {
	// MemoryUsage is the estimated memory usage in bytes
	MemoryUsage int64 `json:"memory_usage"`
	
	// CPUUsage is the estimated CPU usage (0-100)
	CPUUsage int `json:"cpu_usage"`
	
	// Latency is the estimated latency in microseconds
	Latency int `json:"latency"`
	
	// Throughput is the estimated throughput in orders per second
	Throughput int `json:"throughput"`
	
	// ScaleFactor is the estimated scale factor for memory usage as order book size increases
	ScaleFactor float64 `json:"scale_factor"`
}

// MatchingAlgorithmPlugin is the interface for matching algorithm plugins
type MatchingAlgorithmPlugin interface {
	// GetPluginInfo returns information about the plugin
	GetPluginInfo() *PluginInfo
	
	// CreateAlgorithm creates a matching algorithm
	CreateAlgorithm(config matching.AlgorithmConfig, logger *zap.Logger) (matching.MatchingAlgorithm, error)
	
	// Initialize initializes the plugin
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down the plugin
	Shutdown(ctx context.Context) error
}

// MatchingAlgorithmPluginFactory is a factory for creating matching algorithm plugins
type MatchingAlgorithmPluginFactory interface {
	// CreatePlugin creates a plugin
	CreatePlugin() (MatchingAlgorithmPlugin, error)
}

// MatchingAlgorithmPluginRegistry is a registry for matching algorithm plugins
type MatchingAlgorithmPluginRegistry interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin MatchingAlgorithmPlugin) error
	
	// GetPlugin gets a plugin by algorithm type
	GetPlugin(algorithmType string) (MatchingAlgorithmPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []MatchingAlgorithmPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(algorithmType string) error
}

// MatchingAlgorithmPluginLoader is a loader for matching algorithm plugins
type MatchingAlgorithmPluginLoader interface {
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (MatchingAlgorithmPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]MatchingAlgorithmPlugin, error)
}

// MatchingAlgorithmPluginManager manages matching algorithm plugins
type MatchingAlgorithmPluginManager interface {
	// RegisterPlugin registers a plugin
	RegisterPlugin(plugin MatchingAlgorithmPlugin) error
	
	// GetPlugin gets a plugin by algorithm type
	GetPlugin(algorithmType string) (MatchingAlgorithmPlugin, error)
	
	// ListPlugins lists all plugins
	ListPlugins() []MatchingAlgorithmPlugin
	
	// UnregisterPlugin unregisters a plugin
	UnregisterPlugin(algorithmType string) error
	
	// LoadPlugin loads a plugin from a file
	LoadPlugin(filePath string) (MatchingAlgorithmPlugin, error)
	
	// LoadPlugins loads all plugins from a directory
	LoadPlugins(dirPath string) ([]MatchingAlgorithmPlugin, error)
	
	// CreateAlgorithm creates a matching algorithm
	CreateAlgorithm(algorithmType string, config matching.AlgorithmConfig, logger *zap.Logger) (matching.MatchingAlgorithm, error)
	
	// Initialize initializes all plugins
	Initialize(ctx context.Context) error
	
	// Shutdown shuts down all plugins
	Shutdown(ctx context.Context) error
}

