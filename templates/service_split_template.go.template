// 🎯 **Service Split Template**
// Based on successful Orders Service pattern

package {PACKAGE_NAME}

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// TYPES.GO TEMPLATE
// =============================================================================

// {COMPONENT}Status represents the status of a {component}
type {COMPONENT}Status string

const (
	{COMPONENT}StatusNew     {COMPONENT}Status = "new"
	{COMPONENT}StatusPending {COMPONENT}Status = "pending"
	{COMPONENT}StatusActive  {COMPONENT}Status = "active"
	{COMPONENT}StatusStopped {COMPONENT}Status = "stopped"
)

// {COMPONENT}Config contains configuration for the {component}
type {COMPONENT}Config struct {
	MaxLatency     time.Duration `json:"max_latency"`     // Target latency requirement
	EnableFeatureX bool          `json:"enable_feature_x"` // Feature toggle
	MaxSize        int64         `json:"max_size"`        // Maximum size limit
}

// {COMPONENT}Metrics tracks {component} performance
type {COMPONENT}Metrics struct {
	ProcessedCount int64         `json:"processed_count"`
	AverageLatency time.Duration `json:"average_latency"`
	LastUpdateTime time.Time     `json:"last_update_time"`
}

// =============================================================================
// CORE.GO TEMPLATE
// =============================================================================

// {COMPONENT}Service represents the main service struct
type {COMPONENT}Service struct {
	config    *{COMPONENT}Config
	logger    *zap.Logger
	metrics   *{COMPONENT}Metrics
	processor *{COMPONENT}Processor
	validator *{COMPONENT}Validator
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// New{COMPONENT}Service creates a new {component} service
func New{COMPONENT}Service(config *{COMPONENT}Config, logger *zap.Logger) *{COMPONENT}Service {
	if config == nil {
		config = &{COMPONENT}Config{
			MaxLatency: time.Microsecond * 100,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	service := &{COMPONENT}Service{
		config:  config,
		logger:  logger,
		metrics: &{COMPONENT}Metrics{LastUpdateTime: time.Now()},
		ctx:     ctx,
		cancel:  cancel,
	}

	service.processor = New{COMPONENT}Processor(config, logger)
	service.validator = New{COMPONENT}Validator()

	return service
}

// Start starts the {component} service
func (s *{COMPONENT}Service) Start() error {
	s.logger.Info("Starting {component} service")
	return nil
}

// Stop stops the {component} service
func (s *{COMPONENT}Service) Stop() error {
	s.cancel()
	s.logger.Info("{COMPONENT} service stopped")
	return nil
}

// =============================================================================
// PROCESSORS.GO TEMPLATE
// =============================================================================

// {COMPONENT}Processor defines the interface for processing
type {COMPONENT}Processor interface {
	Process(req *{COMPONENT}Request) (*{COMPONENT}Response, error)
	Validate(req *{COMPONENT}Request) error
}

// {COMPONENT}ProcessorImpl implements the processor interface
type {COMPONENT}ProcessorImpl struct {
	config *{COMPONENT}Config
	logger *zap.Logger
}

// New{COMPONENT}Processor creates a new processor
func New{COMPONENT}Processor(config *{COMPONENT}Config, logger *zap.Logger) *{COMPONENT}ProcessorImpl {
	return &{COMPONENT}ProcessorImpl{
		config: config,
		logger: logger,
	}
}

// Process processes a request using early return pattern
func (p *{COMPONENT}ProcessorImpl) Process(req *{COMPONENT}Request) (*{COMPONENT}Response, error) {
	// Early return for nil request
	if req == nil {
		return nil, errors.New("request cannot be nil")
	}

	// Validate request
	if err := p.Validate(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}

	// Process the request
	response := &{COMPONENT}Response{
		ID:        req.ID,
		Status:    {COMPONENT}StatusActive,
		Timestamp: time.Now(),
	}

	return response, nil
}

// =============================================================================
// VALIDATORS.GO TEMPLATE
// =============================================================================

// {COMPONENT}Validator provides validation functionality
type {COMPONENT}Validator struct {
	validators []ValidatorFunc
}

// ValidatorFunc represents a validation function
type ValidatorFunc func(*{COMPONENT}Request) error

// New{COMPONENT}Validator creates a new validator
func New{COMPONENT}Validator() *{COMPONENT}Validator {
	return &{COMPONENT}Validator{
		validators: []ValidatorFunc{
			validateID,
			validateUserID,
		},
	}
}

// Validate validates a request using early return pattern
func (v *{COMPONENT}Validator) Validate(req *{COMPONENT}Request) error {
	// Early return for nil request
	if req == nil {
		return errors.New("request cannot be nil")
	}

	// Run all validators
	for _, validator := range v.validators {
		if err := validator(req); err != nil {
			return err
		}
	}

	return nil
}

// validateID validates the ID field
func validateID(req *{COMPONENT}Request) error {
	if req.ID == "" {
		return errors.New("ID is required")
	}
	return nil
}

// validateUserID validates the user ID field
func validateUserID(req *{COMPONENT}Request) error {
	if req.UserID == "" {
		return errors.New("user ID is required")
	}
	return nil
}

