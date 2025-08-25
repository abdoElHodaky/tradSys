package lazy

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/internal/risk/validator/plugin"
	"go.uber.org/zap"
)

// LazyValidatorRegistry is a lazy-loaded registry for risk validator plugins
type LazyValidatorRegistry struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name prefix
	componentNamePrefix string
	
	// Configuration
	config plugin.RegistryConfig
	
	// Logger
	logger *zap.Logger
	
	// Lock manager for thread safety
	lockManager *coordination.LockManager
	
	// Plugin loader
	loader *plugin.Loader
	
	// Active validators
	activeValidators map[string]bool
	activeValidatorsMu sync.RWMutex
}

// NewLazyValidatorRegistry creates a new lazy-loaded validator registry
func NewLazyValidatorRegistry(
	coordinator *coordination.ComponentCoordinator,
	lockManager *coordination.LockManager,
	config plugin.RegistryConfig,
	logger *zap.Logger,
) (*LazyValidatorRegistry, error) {
	componentNamePrefix := "risk-validator-"
	
	// Register the lock for validator operations
	lockManager.RegisterLock("risk-validators", &sync.Mutex{})
	
	// Create the plugin loader
	loader := plugin.NewLoader(plugin.LoaderConfig{
		PluginDir: config.PluginDir,
	}, logger)
	
	return &LazyValidatorRegistry{
		coordinator:         coordinator,
		componentNamePrefix: componentNamePrefix,
		config:              config,
		logger:              logger,
		lockManager:         lockManager,
		loader:              loader,
		activeValidators:    make(map[string]bool),
	}, nil
}

// GetValidator gets a validator by name
func (r *LazyValidatorRegistry) GetValidator(
	ctx context.Context,
	validatorName string,
) (risk.Validator, error) {
	componentName := r.componentNamePrefix + validatorName
	
	// Check if the component is already registered
	_, err := r.coordinator.GetComponentInfo(componentName)
	if err != nil {
		// Component not registered, register it
		err = r.registerValidator(ctx, validatorName, componentName)
		if err != nil {
			return nil, err
		}
	}
	
	// Get the component
	validatorInterface, err := r.coordinator.GetComponent(ctx, componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual validator type
	validator, ok := validatorInterface.(risk.Validator)
	if !ok {
		return nil, fmt.Errorf("invalid validator type for validator %s", validatorName)
	}
	
	// Update active validators
	r.activeValidatorsMu.Lock()
	r.activeValidators[validatorName] = true
	r.activeValidatorsMu.Unlock()
	
	return validator, nil
}

// registerValidator registers a validator with the coordinator
func (r *LazyValidatorRegistry) registerValidator(
	ctx context.Context,
	validatorName string,
	componentName string,
) error {
	// Acquire the lock to prevent concurrent validator creation
	err := r.lockManager.AcquireLock("risk-validators", "validator-registry")
	if err != nil {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}
	defer r.lockManager.ReleaseLock("risk-validators", "validator-registry")
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		// Load the plugin
		pluginInfo, createFn, err := r.loader.LoadValidatorPlugin(validatorName)
		if err != nil {
			return nil, err
		}
		
		// Create the validator
		validatorConfig := plugin.ValidatorConfig{
			Name: validatorName,
			Type: pluginInfo.ValidatorType,
		}
		
		validator, err := createFn(validatorConfig, log)
		if err != nil {
			return nil, err
		}
		
		return validator, nil
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		r.logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(30*1024*1024), // 30MB estimate
		lazy.WithTimeout(20*time.Second),
		lazy.WithPriority(35), // Medium priority
	)
	
	// Register with the coordinator
	return r.coordinator.RegisterComponent(
		componentName,
		"risk-validator",
		provider,
		[]string{}, // No dependencies
	)
}

// ReleaseValidator releases a validator
func (r *LazyValidatorRegistry) ReleaseValidator(
	ctx context.Context,
	validatorName string,
) error {
	componentName := r.componentNamePrefix + validatorName
	
	// Update active validators
	r.activeValidatorsMu.Lock()
	delete(r.activeValidators, validatorName)
	r.activeValidatorsMu.Unlock()
	
	// Shutdown the component
	return r.coordinator.ShutdownComponent(ctx, componentName)
}

// ValidateOrder validates an order using a specific validator
func (r *LazyValidatorRegistry) ValidateOrder(
	ctx context.Context,
	validatorName string,
	order *orders.Order,
) (bool, string, error) {
	// Get the validator
	validator, err := r.GetValidator(ctx, validatorName)
	if err != nil {
		return false, "", err
	}
	
	// Call the actual method
	return validator.ValidateOrder(ctx, order)
}

// ListActiveValidators lists active validators
func (r *LazyValidatorRegistry) ListActiveValidators() []string {
	r.activeValidatorsMu.RLock()
	defer r.activeValidatorsMu.RUnlock()
	
	validators := make([]string, 0, len(r.activeValidators))
	for validator := range r.activeValidators {
		validators = append(validators, validator)
	}
	
	return validators
}

// GetPluginInfo gets information about a plugin
func (r *LazyValidatorRegistry) GetPluginInfo(
	validatorName string,
) (*plugin.PluginInfo, error) {
	pluginInfo, _, err := r.loader.LoadValidatorPlugin(validatorName)
	return pluginInfo, err
}

// ShutdownAll shuts down all validators
func (r *LazyValidatorRegistry) ShutdownAll(ctx context.Context) error {
	r.activeValidatorsMu.RLock()
	activeValidators := make([]string, 0, len(r.activeValidators))
	for validator := range r.activeValidators {
		activeValidators = append(activeValidators, validator)
	}
	r.activeValidatorsMu.RUnlock()
	
	var lastErr error
	for _, validator := range activeValidators {
		err := r.ReleaseValidator(ctx, validator)
		if err != nil {
			lastErr = err
			r.logger.Error("Failed to release validator",
				zap.String("validator", validator),
				zap.Error(err),
			)
		}
	}
	
	return lastErr
}

