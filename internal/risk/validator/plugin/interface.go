package plugin

import (
	"context"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"go.uber.org/zap"
)

// PluginInfoSymbol is the name of the symbol that plugins must export
const PluginInfoSymbol = "PluginInfo"

// CreateValidatorSymbol is the name of the symbol that plugins must export
const CreateValidatorSymbol = "CreateValidator"

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
	
	// ValidatorType is the type of validator provided by this plugin
	ValidatorType string `json:"validator_type"`
	
	// MinCoreVersion is the minimum core version required by this plugin
	MinCoreVersion string `json:"min_core_version"`
	
	// MaxCoreVersion is the maximum core version supported by this plugin
	MaxCoreVersion string `json:"max_core_version"`
	
	// Dependencies is a list of other plugins that this plugin depends on
	Dependencies []string `json:"dependencies"`
}

// ValidatorConfig is the configuration for a validator
type ValidatorConfig struct {
	// Name is the name of the validator
	Name string `json:"name"`
	
	// Type is the type of validator
	Type string `json:"type"`
	
	// Params is a map of parameters for the validator
	Params map[string]interface{} `json:"params"`
	
	// Enabled indicates whether the validator is enabled
	Enabled bool `json:"enabled"`
	
	// Priority is the priority of the validator (lower values are checked first)
	Priority int `json:"priority"`
}

// RiskValidator defines the interface for a risk validator
type RiskValidator interface {
	// Validate validates an order against risk rules
	Validate(ctx context.Context, order *orders.Order) (bool, string, error)
	
	// GetName returns the name of the validator
	GetName() string
	
	// GetDescription returns the description of the validator
	GetDescription() string
	
	// GetType returns the type of validator
	GetType() string
	
	// GetPriority returns the priority of the validator
	GetPriority() int
	
	// IsEnabled returns whether the validator is enabled
	IsEnabled() bool
	
	// SetEnabled sets whether the validator is enabled
	SetEnabled(enabled bool)
}

// RiskValidatorPlugin defines the interface for a risk validator plugin
type RiskValidatorPlugin interface {
	// GetValidatorType returns the type of validator provided by this plugin
	GetValidatorType() string
	
	// CreateValidator creates a validator instance
	CreateValidator(config ValidatorConfig, logger *zap.Logger) (RiskValidator, error)
}

// ValidatorResult represents the result of a validation
type ValidatorResult struct {
	// Valid indicates whether the validation passed
	Valid bool
	
	// Reason is the reason for the validation result
	Reason string
	
	// ValidatorName is the name of the validator that produced the result
	ValidatorName string
	
	// ValidatorType is the type of validator that produced the result
	ValidatorType string
	
	// Order is the order that was validated
	Order *orders.Order
	
	// Timestamp is the time of the validation
	Timestamp int64
}

// NewValidatorResult creates a new validator result
func NewValidatorResult(valid bool, reason string, validatorName string, validatorType string, order *orders.Order) *ValidatorResult {
	return &ValidatorResult{
		Valid:         valid,
		Reason:        reason,
		ValidatorName: validatorName,
		ValidatorType: validatorType,
		Order:         order,
		Timestamp:     risk.Now().Unix(),
	}
}

