package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.uber.org/zap"
)

// LazyComponentMetrics contains metrics for lazy-loaded components
type LazyComponentMetrics struct {
	// Component name
	Name string `json:"name"`
	
	// Component type
	Type string `json:"type"`
	
	// Whether the component is initialized
	Initialized bool `json:"initialized"`
	
	// Memory usage
	MemoryUsage int64 `json:"memory_usage"`
	
	// Priority
	Priority int `json:"priority"`
	
	// Last access time
	LastAccess time.Time `json:"last_access"`
	
	// Idle time
	IdleTime time.Duration `json:"idle_time"`
	
	// In use
	InUse bool `json:"in_use"`
}

// SystemMetrics contains system-wide metrics
type SystemMetrics struct {
	// Total memory usage
	TotalMemoryUsage int64 `json:"total_memory_usage"`
	
	// Memory limit
	MemoryLimit int64 `json:"memory_limit"`
	
	// Memory usage percentage
	MemoryUsagePercentage float64 `json:"memory_usage_percentage"`
	
	// Memory pressure level
	MemoryPressureLevel string `json:"memory_pressure_level"`
	
	// Number of components
	ComponentCount int `json:"component_count"`
	
	// Number of initialized components
	InitializedComponentCount int `json:"initialized_component_count"`
	
	// Number of components in use
	InUseComponentCount int `json:"in_use_component_count"`
}

// DashboardData contains data for the dashboard
type DashboardData struct {
	// System metrics
	System SystemMetrics `json:"system"`
	
	// Component metrics
	Components []LazyComponentMetrics `json:"components"`
	
	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// LazyComponentDashboard provides a dashboard for lazy-loaded components
type LazyComponentDashboard struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Logger
	logger *zap.Logger
	
	// HTTP server
	server *http.Server
	
	// Mutex for thread safety
	mu sync.RWMutex
	
	// Prometheus metrics
	metrics struct {
		totalMemoryUsage          prometheus.Gauge
		memoryUsagePercentage     prometheus.Gauge
		componentCount            prometheus.Gauge
		initializedComponentCount prometheus.Gauge
		inUseComponentCount       prometheus.Gauge
		componentMemoryUsage      *prometheus.GaugeVec
		componentInitialized      *prometheus.GaugeVec
		componentInUse            *prometheus.GaugeVec
		componentIdleTime         *prometheus.GaugeVec
	}
}

// NewLazyComponentDashboard creates a new lazy component dashboard
func NewLazyComponentDashboard(
	coordinator *coordination.ComponentCoordinator,
	logger *zap.Logger,
) *LazyComponentDashboard {
	dashboard := &LazyComponentDashboard{
		coordinator: coordinator,
		logger:      logger,
	}
	
	// Initialize Prometheus metrics
	dashboard.initPrometheusMetrics()
	
	return dashboard
}

// initPrometheusMetrics initializes Prometheus metrics
func (d *LazyComponentDashboard) initPrometheusMetrics() {
	// System metrics
	d.metrics.totalMemoryUsage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lazy_component_total_memory_usage_bytes",
		Help: "Total memory usage of lazy-loaded components in bytes",
	})
	
	d.metrics.memoryUsagePercentage = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lazy_component_memory_usage_percentage",
		Help: "Memory usage percentage of lazy-loaded components",
	})
	
	d.metrics.componentCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lazy_component_count",
		Help: "Number of lazy-loaded components",
	})
	
	d.metrics.initializedComponentCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lazy_component_initialized_count",
		Help: "Number of initialized lazy-loaded components",
	})
	
	d.metrics.inUseComponentCount = prometheus.NewGauge(prometheus.GaugeOpts{
		Name: "lazy_component_in_use_count",
		Help: "Number of lazy-loaded components in use",
	})
	
	// Component metrics
	d.metrics.componentMemoryUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_component_memory_usage_bytes",
			Help: "Memory usage of a lazy-loaded component in bytes",
		},
		[]string{"name", "type"},
	)
	
	d.metrics.componentInitialized = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_component_initialized",
			Help: "Whether a lazy-loaded component is initialized (1) or not (0)",
		},
		[]string{"name", "type"},
	)
	
	d.metrics.componentInUse = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_component_in_use",
			Help: "Whether a lazy-loaded component is in use (1) or not (0)",
		},
		[]string{"name", "type"},
	)
	
	d.metrics.componentIdleTime = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "lazy_component_idle_time_seconds",
			Help: "Idle time of a lazy-loaded component in seconds",
		},
		[]string{"name", "type"},
	)
	
	// Register metrics
	prometheus.MustRegister(
		d.metrics.totalMemoryUsage,
		d.metrics.memoryUsagePercentage,
		d.metrics.componentCount,
		d.metrics.initializedComponentCount,
		d.metrics.inUseComponentCount,
		d.metrics.componentMemoryUsage,
		d.metrics.componentInitialized,
		d.metrics.componentInUse,
		d.metrics.componentIdleTime,
	)
}

// Start starts the dashboard
func (d *LazyComponentDashboard) Start(addr string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	// Create a new HTTP server
	mux := http.NewServeMux()
	
	// Register handlers
	mux.HandleFunc("/api/metrics", d.handleMetrics)
	mux.HandleFunc("/api/components", d.handleComponents)
	mux.HandleFunc("/api/system", d.handleSystem)
	mux.HandleFunc("/api/dashboard", d.handleDashboard)
	
	// Register Prometheus metrics handler
	mux.Handle("/metrics", promhttp.Handler())
	
	// Serve static files
	mux.Handle("/", http.FileServer(http.Dir("./static")))
	
	// Create the server
	d.server = &http.Server{
		Addr:    addr,
		Handler: mux,
	}
	
	// Start updating metrics in the background
	go d.updateMetrics()
	
	// Start the server
	d.logger.Info("Starting lazy component dashboard", zap.String("addr", addr))
	return d.server.ListenAndServe()
}

// Stop stops the dashboard
func (d *LazyComponentDashboard) Stop(ctx context.Context) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	
	if d.server == nil {
		return nil
	}
	
	d.logger.Info("Stopping lazy component dashboard")
	return d.server.Shutdown(ctx)
}

// updateMetrics updates Prometheus metrics
func (d *LazyComponentDashboard) updateMetrics() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()
	
	for range ticker.C {
		// Get system metrics
		systemMetrics := d.getSystemMetrics()
		
		// Update system metrics
		d.metrics.totalMemoryUsage.Set(float64(systemMetrics.TotalMemoryUsage))
		d.metrics.memoryUsagePercentage.Set(systemMetrics.MemoryUsagePercentage)
		d.metrics.componentCount.Set(float64(systemMetrics.ComponentCount))
		d.metrics.initializedComponentCount.Set(float64(systemMetrics.InitializedComponentCount))
		d.metrics.inUseComponentCount.Set(float64(systemMetrics.InUseComponentCount))
		
		// Get component metrics
		componentMetrics := d.getComponentMetrics()
		
		// Update component metrics
		for _, metric := range componentMetrics {
			d.metrics.componentMemoryUsage.WithLabelValues(metric.Name, metric.Type).Set(float64(metric.MemoryUsage))
			
			if metric.Initialized {
				d.metrics.componentInitialized.WithLabelValues(metric.Name, metric.Type).Set(1)
			} else {
				d.metrics.componentInitialized.WithLabelValues(metric.Name, metric.Type).Set(0)
			}
			
			if metric.InUse {
				d.metrics.componentInUse.WithLabelValues(metric.Name, metric.Type).Set(1)
			} else {
				d.metrics.componentInUse.WithLabelValues(metric.Name, metric.Type).Set(0)
			}
			
			d.metrics.componentIdleTime.WithLabelValues(metric.Name, metric.Type).Set(metric.IdleTime.Seconds())
		}
	}
}

// getSystemMetrics gets system metrics
func (d *LazyComponentDashboard) getSystemMetrics() SystemMetrics {
	// Get memory manager
	memoryManager := d.coordinator.GetMemoryManager()
	
	// Get memory usage
	totalMemoryUsage := memoryManager.GetMemoryUsage()
	memoryLimit := memoryManager.GetMemoryLimit()
	memoryUsagePercentage := float64(totalMemoryUsage) / float64(memoryLimit) * 100
	
	// Get memory pressure level
	pressureLevel := memoryManager.GetMemoryPressureLevel()
	pressureLevelStr := "Low"
	switch pressureLevel {
	case coordination.MemoryPressureMedium:
		pressureLevelStr = "Medium"
	case coordination.MemoryPressureHigh:
		pressureLevelStr = "High"
	case coordination.MemoryPressureCritical:
		pressureLevelStr = "Critical"
	}
	
	// Get component info
	components := d.coordinator.ListComponents()
	
	// Count initialized and in-use components
	initializedCount := 0
	inUseCount := 0
	
	for _, component := range components {
		if component.Initialized {
			initializedCount++
		}
		
		// Check if the component is in use
		info, err := memoryManager.GetComponentInfo(component.Name)
		if err == nil && info.InUse {
			inUseCount++
		}
	}
	
	return SystemMetrics{
		TotalMemoryUsage:         totalMemoryUsage,
		MemoryLimit:              memoryLimit,
		MemoryUsagePercentage:    memoryUsagePercentage,
		MemoryPressureLevel:      pressureLevelStr,
		ComponentCount:           len(components),
		InitializedComponentCount: initializedCount,
		InUseComponentCount:      inUseCount,
	}
}

// getComponentMetrics gets component metrics
func (d *LazyComponentDashboard) getComponentMetrics() []LazyComponentMetrics {
	// Get memory manager
	memoryManager := d.coordinator.GetMemoryManager()
	
	// Get component info
	components := d.coordinator.ListComponents()
	
	// Create metrics
	metrics := make([]LazyComponentMetrics, 0, len(components))
	
	now := time.Now()
	
	for _, component := range components {
		// Get memory info
		memoryInfo, err := memoryManager.GetComponentInfo(component.Name)
		if err != nil {
			continue
		}
		
		// Calculate idle time
		idleTime := now.Sub(memoryInfo.LastAccess)
		
		metrics = append(metrics, LazyComponentMetrics{
			Name:        component.Name,
			Type:        component.Type,
			Initialized: component.Initialized,
			MemoryUsage: memoryInfo.MemoryUsage,
			Priority:    memoryInfo.Priority,
			LastAccess:  memoryInfo.LastAccess,
			IdleTime:    idleTime,
			InUse:       memoryInfo.InUse,
		})
	}
	
	// Sort by name
	sort.Slice(metrics, func(i, j int) bool {
		return metrics[i].Name < metrics[j].Name
	})
	
	return metrics
}

// handleMetrics handles the metrics API endpoint
func (d *LazyComponentDashboard) handleMetrics(w http.ResponseWriter, r *http.Request) {
	// Get dashboard data
	data := d.getDashboardData()
	
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	// Encode as JSON
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		d.logger.Error("Failed to encode metrics", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleComponents handles the components API endpoint
func (d *LazyComponentDashboard) handleComponents(w http.ResponseWriter, r *http.Request) {
	// Get component metrics
	metrics := d.getComponentMetrics()
	
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	// Encode as JSON
	err := json.NewEncoder(w).Encode(metrics)
	if err != nil {
		d.logger.Error("Failed to encode component metrics", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleSystem handles the system API endpoint
func (d *LazyComponentDashboard) handleSystem(w http.ResponseWriter, r *http.Request) {
	// Get system metrics
	metrics := d.getSystemMetrics()
	
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	// Encode as JSON
	err := json.NewEncoder(w).Encode(metrics)
	if err != nil {
		d.logger.Error("Failed to encode system metrics", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleDashboard handles the dashboard API endpoint
func (d *LazyComponentDashboard) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// Get dashboard data
	data := d.getDashboardData()
	
	// Set content type
	w.Header().Set("Content-Type", "application/json")
	
	// Encode as JSON
	err := json.NewEncoder(w).Encode(data)
	if err != nil {
		d.logger.Error("Failed to encode dashboard data", zap.Error(err))
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// getDashboardData gets dashboard data
func (d *LazyComponentDashboard) getDashboardData() DashboardData {
	return DashboardData{
		System:     d.getSystemMetrics(),
		Components: d.getComponentMetrics(),
		Timestamp:  time.Now(),
	}
}

// FormatBytes formats bytes as a human-readable string
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// FormatDuration formats a duration as a human-readable string
func FormatDuration(d time.Duration) string {
	if d.Hours() >= 24 {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		return fmt.Sprintf("%dd %dh", days, hours)
	} else if d.Hours() >= 1 {
		hours := int(d.Hours())
		minutes := int(d.Minutes()) % 60
		return fmt.Sprintf("%dh %dm", hours, minutes)
	} else if d.Minutes() >= 1 {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60
		return fmt.Sprintf("%dm %ds", minutes, seconds)
	} else {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
}

