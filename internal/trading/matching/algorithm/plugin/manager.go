package plugin

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/matching"
	"go.uber.org/zap"
)

// Manager manages matching algorithm plugins
type Manager struct {
	logger            *zap.Logger
	registry          *Registry
	loader            *Loader
	metrics           *lazy.AdaptiveMetrics
	contextPropagator *lazy.ContextPropagator
	algorithmCache    map[string]matching.MatchingAlgorithm
	cacheMu           sync.RWMutex
	coreVersion       string
	performanceData   map[string]*PerformanceData
	performanceMu     sync.RWMutex
}

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
	
	// LastUpdated is the time when the performance data was last updated
	LastUpdated time.Time `json:"last_updated"`
	
	// SampleCount is the number of samples used to calculate the performance data
	SampleCount int `json:"sample_count"`
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
		algorithmCache:    make(map[string]matching.MatchingAlgorithm),
		coreVersion:       coreVersion,
		performanceData:   make(map[string]*PerformanceData),
	}
}

// RegisterPlugin registers a plugin
func (m *Manager) RegisterPlugin(plugin MatchingAlgorithmPlugin) error {
	return m.registry.RegisterPlugin(plugin)
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
	// Remove from cache
	m.cacheMu.Lock()
	delete(m.algorithmCache, algorithmType)
	m.cacheMu.Unlock()
	
	// Remove performance data
	m.performanceMu.Lock()
	delete(m.performanceData, algorithmType)
	m.performanceMu.Unlock()
	
	return m.registry.UnregisterPlugin(algorithmType)
}

// LoadPlugin loads a plugin from a file
func (m *Manager) LoadPlugin(filePath string) (MatchingAlgorithmPlugin, error) {
	return m.loader.LoadPlugin(filePath)
}

// LoadPlugins loads all plugins from a directory
func (m *Manager) LoadPlugins(dirPath string) ([]MatchingAlgorithmPlugin, error) {
	return m.loader.LoadPlugins(dirPath)
}

// LoadAllPlugins loads all plugins from all configured directories
func (m *Manager) LoadAllPlugins() ([]MatchingAlgorithmPlugin, error) {
	return m.loader.LoadAllPlugins()
}

// CreateAlgorithm creates a matching algorithm
func (m *Manager) CreateAlgorithm(
	algorithmType string,
	config matching.AlgorithmConfig,
	logger *zap.Logger,
) (matching.MatchingAlgorithm, error) {
	// Check cache first
	m.cacheMu.RLock()
	algorithm, ok := m.algorithmCache[algorithmType]
	m.cacheMu.RUnlock()
	
	if ok {
		return algorithm, nil
	}
	
	// Create new algorithm
	algorithm, err := m.registry.CreateAlgorithm(algorithmType, config, logger)
	if err != nil {
		return nil, err
	}
	
	// Add to cache
	m.cacheMu.Lock()
	m.algorithmCache[algorithmType] = algorithm
	m.cacheMu.Unlock()
	
	// Initialize performance data
	m.initializePerformanceData(algorithmType)
	
	return algorithm, nil
}

// CreateAlgorithmWithContext creates a matching algorithm with context
func (m *Manager) CreateAlgorithmWithContext(
	ctx context.Context,
	algorithmType string,
	config matching.AlgorithmConfig,
	logger *zap.Logger,
) (matching.MatchingAlgorithm, error) {
	// Create a channel for the result
	resultCh := make(chan struct {
		algorithm matching.MatchingAlgorithm
		err       error
	})
	
	// Create algorithm in a goroutine
	go func() {
		algorithm, err := m.CreateAlgorithm(algorithmType, config, logger)
		resultCh <- struct {
			algorithm matching.MatchingAlgorithm
			err       error
		}{algorithm, err}
	}()
	
	// Wait for the result or context cancellation
	select {
	case result := <-resultCh:
		return result.algorithm, result.err
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
	
	m.logger.Info("Loaded matching algorithm plugins", zap.Int("count", len(plugins)))
	
	// Initialize all plugins
	return m.registry.Initialize(ctx)
}

// Shutdown shuts down all plugins
func (m *Manager) Shutdown(ctx context.Context) error {
	// Clear algorithm cache
	m.cacheMu.Lock()
	m.algorithmCache = make(map[string]matching.MatchingAlgorithm)
	m.cacheMu.Unlock()
	
	// Clear performance data
	m.performanceMu.Lock()
	m.performanceData = make(map[string]*PerformanceData)
	m.performanceMu.Unlock()
	
	// Shutdown all plugins
	return m.registry.Shutdown(ctx)
}

// LoadAllPluginsWithContext loads all plugins with context
func (m *Manager) LoadAllPluginsWithContext(ctx context.Context) ([]MatchingAlgorithmPlugin, error) {
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

// GetAlgorithmCache gets the algorithm cache
func (m *Manager) GetAlgorithmCache() map[string]matching.MatchingAlgorithm {
	m.cacheMu.RLock()
	defer m.cacheMu.RUnlock()
	
	cache := make(map[string]matching.MatchingAlgorithm, len(m.algorithmCache))
	for k, v := range m.algorithmCache {
		cache[k] = v
	}
	
	return cache
}

// ClearAlgorithmCache clears the algorithm cache
func (m *Manager) ClearAlgorithmCache() {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	m.algorithmCache = make(map[string]matching.MatchingAlgorithm)
}

// RemoveFromAlgorithmCache removes an algorithm from the cache
func (m *Manager) RemoveFromAlgorithmCache(algorithmType string) {
	m.cacheMu.Lock()
	defer m.cacheMu.Unlock()
	
	delete(m.algorithmCache, algorithmType)
}

// initializePerformanceData initializes performance data for an algorithm
func (m *Manager) initializePerformanceData(algorithmType string) {
	m.performanceMu.Lock()
	defer m.performanceMu.Unlock()
	
	// Check if performance data already exists
	if _, ok := m.performanceData[algorithmType]; ok {
		return
	}
	
	// Get plugin info
	plugin, err := m.GetPlugin(algorithmType)
	if err != nil {
		m.logger.Error("Failed to get plugin for performance data initialization",
			zap.String("algorithm_type", algorithmType),
			zap.Error(err))
		return
	}
	
	info := plugin.GetPluginInfo()
	
	// Create performance data
	m.performanceData[algorithmType] = &PerformanceData{
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

