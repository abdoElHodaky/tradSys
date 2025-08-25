package plugin

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/zap"
)

// PerformanceData contains performance data for a matching algorithm
type PerformanceData struct {
	// AverageLatency is the average latency in microseconds
	AverageLatency int `json:"average_latency"`
	
	// MaxLatency is the maximum latency in microseconds
	MaxLatency int `json:"max_latency"`
	
	// MinLatency is the minimum latency in microseconds
	MinLatency int `json:"min_latency"`
	
	// AverageThroughput is the average throughput in orders per second
	AverageThroughput int `json:"average_throughput"`
	
	// MaxThroughput is the maximum throughput in orders per second
	MaxThroughput int `json:"max_throughput"`
	
	// MinThroughput is the minimum throughput in orders per second
	MinThroughput int `json:"min_throughput"`
	
	// MemoryUsage is the memory usage in bytes
	MemoryUsage int64 `json:"memory_usage"`
	
	// CPUUsage is the CPU usage (0-100)
	CPUUsage int `json:"cpu_usage"`
	
	// LastUpdated is the time the performance data was last updated
	LastUpdated time.Time `json:"last_updated"`
	
	// SampleCount is the number of samples used to calculate the averages
	SampleCount int `json:"sample_count"`
}

// Manager is a manager for matching algorithm plugins
type Manager struct {
	logger         *zap.Logger
	registry       *Registry
	loader         *Loader
	performanceData map[string]*PerformanceData
	performanceMu  sync.RWMutex
}

// NewManager creates a new manager
func NewManager(
	logger *zap.Logger,
	registry *Registry,
	loader *Loader,
) *Manager {
	return &Manager{
		logger:         logger,
		registry:       registry,
		loader:         loader,
		performanceData: make(map[string]*PerformanceData),
	}
}

// RegisterPlugin registers a plugin
func (m *Manager) RegisterPlugin(plugin MatchingAlgorithmPlugin) error {
	err := m.registry.RegisterPlugin(plugin)
	if err != nil {
		return err
	}
	
	// Initialize performance data
	m.initializePerformanceData(plugin.GetPluginInfo())
	
	return nil
}

// GetPlugin gets a plugin by algorithm type
func (m *Manager) GetPlugin(algorithmType string) (MatchingAlgorithmPlugin, error) {
	return m.registry.GetPlugin(algorithmType)
}

// ListPlugins lists all plugins
func (m *Manager) ListPlugins() []MatchingAlgorithmPlugin {
	return m.registry.ListPlugins()
}

// UnregisterPlugin unregisters a plugin
func (m *Manager) UnregisterPlugin(algorithmType string) error {
	err := m.registry.UnregisterPlugin(algorithmType)
	if err != nil {
		return err
	}
	
	// Remove performance data
	m.performanceMu.Lock()
	delete(m.performanceData, algorithmType)
	m.performanceMu.Unlock()
	
	return nil
}

// LoadPlugin loads a plugin from a file
func (m *Manager) LoadPlugin(filePath string) (MatchingAlgorithmPlugin, error) {
	plugin, err := m.loader.LoadPlugin(filePath)
	if err != nil {
		return nil, err
	}
	
	// Initialize performance data
	m.initializePerformanceData(plugin.GetPluginInfo())
	
	return plugin, nil
}

// LoadPlugins loads all plugins from a directory
func (m *Manager) LoadPlugins(dirPath string) ([]MatchingAlgorithmPlugin, error) {
	plugins, err := m.loader.LoadPlugins(dirPath)
	if err != nil {
		return nil, err
	}
	
	// Initialize performance data for all plugins
	for _, plugin := range plugins {
		m.initializePerformanceData(plugin.GetPluginInfo())
	}
	
	return plugins, nil
}

// CreateAlgorithm creates a matching algorithm
func (m *Manager) CreateAlgorithm(
	algorithmType string,
	config matching.AlgorithmConfig,
	logger *zap.Logger,
) (matching.MatchingAlgorithm, error) {
	return m.registry.CreateAlgorithm(algorithmType, config, logger)
}

// Initialize initializes all plugins
func (m *Manager) Initialize(ctx context.Context) error {
	return m.registry.Initialize(ctx)
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	return m.registry.Shutdown(ctx)
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

// initializePerformanceData initializes performance data for a plugin
func (m *Manager) initializePerformanceData(info *PluginInfo) {
	m.performanceMu.Lock()
	defer m.performanceMu.Unlock()
	
	// Check if performance data already exists
	if _, ok := m.performanceData[info.AlgorithmType]; ok {
		return
	}
	
	// Initialize performance data
	m.performanceData[info.AlgorithmType] = &PerformanceData{
		AverageLatency:    info.PerformanceProfile.Latency,
		MaxLatency:        info.PerformanceProfile.Latency,
		MinLatency:        info.PerformanceProfile.Latency,
		AverageThroughput: info.PerformanceProfile.Throughput,
		MaxThroughput:     info.PerformanceProfile.Throughput,
		MinThroughput:     info.PerformanceProfile.Throughput,
		MemoryUsage:       info.PerformanceProfile.MemoryUsage,
		CPUUsage:          info.PerformanceProfile.CPUUsage,
		LastUpdated:       time.Now(),
		SampleCount:       1,
	}
}

// UpdatePerformanceData updates performance data for an algorithm
func (m *Manager) UpdatePerformanceData(
	algorithmType string,
	latency int,
	throughput int,
	memoryUsage int64,
	cpuUsage int,
) {
	m.performanceMu.Lock()
	defer m.performanceMu.Unlock()
	
	// Check if performance data exists
	data, ok := m.performanceData[algorithmType]
	if !ok {
		// Initialize performance data
		m.performanceData[algorithmType] = &PerformanceData{
			AverageLatency:    latency,
			MaxLatency:        latency,
			MinLatency:        latency,
			AverageThroughput: throughput,
			MaxThroughput:     throughput,
			MinThroughput:     throughput,
			MemoryUsage:       memoryUsage,
			CPUUsage:          cpuUsage,
			LastUpdated:       time.Now(),
			SampleCount:       1,
		}
		return
	}
	
	// Update performance data
	data.AverageLatency = (data.AverageLatency*data.SampleCount + latency) / (data.SampleCount + 1)
	data.MaxLatency = max(data.MaxLatency, latency)
	data.MinLatency = min(data.MinLatency, latency)
	
	data.AverageThroughput = (data.AverageThroughput*data.SampleCount + throughput) / (data.SampleCount + 1)
	data.MaxThroughput = max(data.MaxThroughput, throughput)
	data.MinThroughput = min(data.MinThroughput, throughput)
	
	data.MemoryUsage = memoryUsage
	data.CPUUsage = cpuUsage
	data.LastUpdated = time.Now()
	data.SampleCount++
}

// GetPerformanceData gets performance data for an algorithm
func (m *Manager) GetPerformanceData(algorithmType string) (*PerformanceData, error) {
	m.performanceMu.RLock()
	defer m.performanceMu.RUnlock()
	
	data, ok := m.performanceData[algorithmType]
	if !ok {
		return nil, fmt.Errorf("performance data not found for algorithm type: %s", algorithmType)
	}
	
	return data, nil
}

// GetAllPerformanceData gets performance data for all algorithms
func (m *Manager) GetAllPerformanceData() map[string]*PerformanceData {
	m.performanceMu.RLock()
	defer m.performanceMu.RUnlock()
	
	data := make(map[string]*PerformanceData, len(m.performanceData))
	for k, v := range m.performanceData {
		data[k] = v
	}
	
	return data
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

