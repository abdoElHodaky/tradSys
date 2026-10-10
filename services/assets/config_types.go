// Package assets provides configuration types for TradSys v3
package assets

// ServiceConfig holds service configuration
type ServiceConfig struct {
	Name        string
	Endpoint    string
	Enabled     bool
	Workers     int
	MaxAlloc    int64
	LogLevel    string
}

// ConfigStore stores configurations
type ConfigStore struct{}

// ConfigValidator validates configurations
type ConfigValidator struct{}

// ConfigChangeNotifier notifies of changes
type ConfigChangeNotifier struct{}