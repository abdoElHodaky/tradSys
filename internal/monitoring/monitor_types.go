package monitoring

import (
	"context"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"go.uber.org/zap"
)

// UnifiedMonitor provides comprehensive system monitoring and metrics collection
type UnifiedMonitor struct {
	// Core components
	metricsCollector *UnifiedMetricsCollector
	alertManager     *UnifiedAlertManager
	healthChecker    *HealthChecker
	performanceTracker *PerformanceTracker
	
	// Configuration
	config *MonitorConfig
	logger *zap.Logger
	
	// Metrics registry
	registry *prometheus.Registry
	
	// Lifecycle management
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex
	
	// System state
	isRunning bool
	startTime time.Time
}

// MonitorConfig contains monitoring configuration
type MonitorConfig struct {
	MetricsInterval     time.Duration `json:"metrics_interval"`
	HealthCheckInterval time.Duration `json:"health_check_interval"`
	AlertCheckInterval  time.Duration `json:"alert_check_interval"`
	RetentionPeriod     time.Duration `json:"retention_period"`
	EnablePrometheus    bool          `json:"enable_prometheus"`
	EnableAlerts        bool          `json:"enable_alerts"`
	EnableHealthChecks  bool          `json:"enable_health_checks"`
	MaxMetricsHistory   int           `json:"max_metrics_history"`
}

// SystemMetrics represents comprehensive system metrics
type SystemMetrics struct {
	// Trading metrics
	OrdersPerSecond     float64 `json:"orders_per_second"`
	TradesPerSecond     float64 `json:"trades_per_second"`
	MatchingLatency     float64 `json:"matching_latency_ms"`
	OrderBookDepth      int     `json:"order_book_depth"`
	ActiveConnections   int64   `json:"active_connections"`
	
	// Performance metrics
	CPUUsage           float64 `json:"cpu_usage_percent"`
	MemoryUsage        float64 `json:"memory_usage_percent"`
	DiskUsage          float64 `json:"disk_usage_percent"`
	NetworkThroughput  float64 `json:"network_throughput_mbps"`
	
	// Application metrics
	ErrorRate          float64 `json:"error_rate_percent"`
	ResponseTime       float64 `json:"response_time_ms"`
	ThroughputRPS      float64 `json:"throughput_rps"`
	CacheHitRate       float64 `json:"cache_hit_rate_percent"`
	
	// Business metrics
	TotalVolume        float64   `json:"total_volume"`
	TotalTrades        int64     `json:"total_trades"`
	ActiveUsers        int64     `json:"active_users"`
	ComplianceScore    float64   `json:"compliance_score"`
	
	// Timestamp
	Timestamp time.Time `json:"timestamp"`
}

// UnifiedHealthStatus represents system health status
type UnifiedHealthStatus struct {
	Overall    HealthState            `json:"overall"`
	Components map[string]HealthState `json:"components"`
	Timestamp  time.Time              `json:"timestamp"`
	Uptime     time.Duration          `json:"uptime"`
	Details    map[string]interface{} `json:"details,omitempty"`
}

// HealthState represents health state
type HealthState string

const (
	HealthStateHealthy   HealthState = "healthy"
	HealthStateWarning   HealthState = "warning"
	HealthStateCritical  HealthState = "critical"
	HealthStateUnknown   HealthState = "unknown"
)

// UnifiedAlert represents a system alert
type UnifiedAlert struct {
	ID          string                 `json:"id"`
	Type        AlertType              `json:"type"`
	Severity    AlertSeverity          `json:"severity"`
	Title       string                 `json:"title"`
	Message     string                 `json:"message"`
	Component   string                 `json:"component"`
	Timestamp   time.Time              `json:"timestamp"`
	Resolved    bool                   `json:"resolved"`
	ResolvedAt  *time.Time             `json:"resolved_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// AlertType represents alert type
type AlertType string

const (
	AlertTypePerformance AlertType = "performance"
	AlertTypeError       AlertType = "error"
	AlertTypeHealth      AlertType = "health"
	AlertTypeSecurity    AlertType = "security"
	AlertTypeBusiness    AlertType = "business"
)

// AlertSeverity represents alert severity
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// UnifiedMetricsCollector collects system metrics
type UnifiedMetricsCollector struct {
	config  *MonitorConfig
	logger  *zap.Logger
	metrics *SystemMetrics
	mu      sync.RWMutex
}

// UnifiedAlertManager manages system alerts
type UnifiedAlertManager struct {
	config    *MonitorConfig
	logger    *zap.Logger
	alerts    map[string]*UnifiedAlert
	rules     []*AlertRule
	mu        sync.RWMutex
}

// AlertRule represents an alert rule
type AlertRule struct {
	ID          string        `json:"id"`
	Name        string        `json:"name"`
	Description string        `json:"description"`
	Condition   string        `json:"condition"`
	Threshold   float64       `json:"threshold"`
	Duration    time.Duration `json:"duration"`
	Severity    AlertSeverity `json:"severity"`
	Enabled     bool          `json:"enabled"`
}

// HealthChecker performs health checks
type HealthChecker struct {
	config     *MonitorConfig
	logger     *zap.Logger
	checks     map[string]UnifiedHealthCheck
	lastStatus *UnifiedHealthStatus
	mu         sync.RWMutex
}

// UnifiedHealthCheck represents a health check function for the unified monitor
type UnifiedHealthCheck func() (HealthState, map[string]interface{}, error)

// PerformanceTracker tracks performance metrics
type PerformanceTracker struct {
	config         *MonitorConfig
	logger         *zap.Logger
	metricsHistory []*SystemMetrics
	mu             sync.RWMutex
}

// MetricsSnapshot represents a snapshot of metrics at a point in time
type MetricsSnapshot struct {
	Metrics   *SystemMetrics `json:"metrics"`
	Timestamp time.Time      `json:"timestamp"`
}

// PerformanceReport represents a performance analysis report
type PerformanceReport struct {
	Period      string                 `json:"period"`
	StartTime   time.Time              `json:"start_time"`
	EndTime     time.Time              `json:"end_time"`
	Summary     *PerformanceSummary    `json:"summary"`
	Trends      map[string]float64     `json:"trends"`
	Anomalies   []*PerformanceAnomaly  `json:"anomalies"`
	Recommendations []string           `json:"recommendations"`
}

// PerformanceSummary represents performance summary statistics
type PerformanceSummary struct {
	AvgOrdersPerSecond   float64 `json:"avg_orders_per_second"`
	AvgTradesPerSecond   float64 `json:"avg_trades_per_second"`
	AvgMatchingLatency   float64 `json:"avg_matching_latency"`
	AvgCPUUsage          float64 `json:"avg_cpu_usage"`
	AvgMemoryUsage       float64 `json:"avg_memory_usage"`
	AvgErrorRate         float64 `json:"avg_error_rate"`
	AvgResponseTime      float64 `json:"avg_response_time"`
	PeakOrdersPerSecond  float64 `json:"peak_orders_per_second"`
	PeakTradesPerSecond  float64 `json:"peak_trades_per_second"`
	MaxMatchingLatency   float64 `json:"max_matching_latency"`
	MaxCPUUsage          float64 `json:"max_cpu_usage"`
	MaxMemoryUsage       float64 `json:"max_memory_usage"`
	MaxErrorRate         float64 `json:"max_error_rate"`
	MaxResponseTime      float64 `json:"max_response_time"`
}

// PerformanceAnomaly represents a detected performance anomaly
type PerformanceAnomaly struct {
	Type        string    `json:"type"`
	Metric      string    `json:"metric"`
	Value       float64   `json:"value"`
	Expected    float64   `json:"expected"`
	Deviation   float64   `json:"deviation"`
	Timestamp   time.Time `json:"timestamp"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
}

// ComponentStatus represents the status of a system component
type ComponentStatus struct {
	Name        string                 `json:"name"`
	Status      HealthState            `json:"status"`
	LastCheck   time.Time              `json:"last_check"`
	Message     string                 `json:"message"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// MonitoringEvent represents a monitoring event
type MonitoringEvent struct {
	ID        string                 `json:"id"`
	Type      string                 `json:"type"`
	Source    string                 `json:"source"`
	Message   string                 `json:"message"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// AlertNotification represents an alert notification
type AlertNotification struct {
	Alert     *UnifiedAlert `json:"alert"`
	Channel   string        `json:"channel"`
	Recipient string        `json:"recipient"`
	SentAt    time.Time     `json:"sent_at"`
	Status    string        `json:"status"`
}

// MetricsAggregation represents aggregated metrics over a time period
type MetricsAggregation struct {
	Period    string                 `json:"period"`
	StartTime time.Time              `json:"start_time"`
	EndTime   time.Time              `json:"end_time"`
	Count     int                    `json:"count"`
	Min       map[string]float64     `json:"min"`
	Max       map[string]float64     `json:"max"`
	Avg       map[string]float64     `json:"avg"`
	Sum       map[string]float64     `json:"sum"`
	Percentiles map[string]map[string]float64 `json:"percentiles"` // metric -> percentile -> value
}
