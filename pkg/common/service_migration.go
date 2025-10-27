package common

import (
	"context"
	"fmt"
	"reflect"
	"time"

	"go.uber.org/zap"
)

// ServiceMigrator helps migrate existing services to BaseService pattern
type ServiceMigrator struct {
	logger *zap.Logger
}

// NewServiceMigrator creates a new service migrator
func NewServiceMigrator(logger *zap.Logger) *ServiceMigrator {
	return &ServiceMigrator{
		logger: logger,
	}
}

// MigrationTemplate provides a template for service migration
type MigrationTemplate struct {
	ServiceName    string
	ServiceVersion string
	StartFunc      func(ctx context.Context) error
	StopFunc       func(ctx context.Context) error
	HealthFunc     func() HealthStatus
	Dependencies   []string
}

// MigrateService migrates an existing service to BaseService pattern
func (sm *ServiceMigrator) MigrateService(template *MigrationTemplate) ServiceInterface {
	baseService := NewBaseService(template.ServiceName, template.ServiceVersion, sm.logger)
	
	// Set lifecycle hooks
	if template.StartFunc != nil {
		baseService.SetStartHook(template.StartFunc)
	}
	
	if template.StopFunc != nil {
		baseService.SetStopHook(template.StopFunc)
	}
	
	// Create migrated service wrapper
	migratedService := &MigratedService{
		BaseService: baseService,
		healthFunc:  template.HealthFunc,
	}
	
	sm.logger.Info("Service migrated to BaseService pattern",
		zap.String("service", template.ServiceName),
		zap.String("version", template.ServiceVersion),
	)
	
	return migratedService
}

// MigratedService wraps BaseService with custom health function
type MigratedService struct {
	*BaseService
	healthFunc func() HealthStatus
}

// Health returns custom health status if provided, otherwise base health
func (ms *MigratedService) Health() HealthStatus {
	if ms.healthFunc != nil {
		return ms.healthFunc()
	}
	return ms.BaseService.Health()
}

// ServicePattern represents different service patterns found in the codebase
type ServicePattern struct {
	Name        string
	Description string
	Example     interface{}
	Complexity  MigrationComplexity
}

// MigrationComplexity indicates how complex the migration will be
type MigrationComplexity int

const (
	MigrationComplexitySimple MigrationComplexity = iota
	MigrationComplexityModerate
	MigrationComplexityComplex
)

// String returns the string representation of migration complexity
func (mc MigrationComplexity) String() string {
	switch mc {
	case MigrationComplexitySimple:
		return "simple"
	case MigrationComplexityModerate:
		return "moderate"
	case MigrationComplexityComplex:
		return "complex"
	default:
		return "unknown"
	}
}

// GetServicePatterns returns all identified service patterns in the codebase
func (sm *ServiceMigrator) GetServicePatterns() []ServicePattern {
	return []ServicePattern{
		{
			Name:        "BasicService",
			Description: "Simple service with Start/Stop methods",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "HTTPService",
			Description: "HTTP server service with graceful shutdown",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "DatabaseService",
			Description: "Database connection service with health checks",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "MessageQueueService",
			Description: "Message queue consumer/producer service",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "CacheService",
			Description: "Cache service with connection pooling",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "MetricsService",
			Description: "Metrics collection and reporting service",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "LoggingService",
			Description: "Centralized logging service",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "AuthService",
			Description: "Authentication and authorization service",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "TradingService",
			Description: "Trading execution service with complex state",
			Complexity:  MigrationComplexityComplex,
		},
		{
			Name:        "RiskService",
			Description: "Risk management service with real-time processing",
			Complexity:  MigrationComplexityComplex,
		},
		{
			Name:        "MarketDataService",
			Description: "Market data ingestion and processing service",
			Complexity:  MigrationComplexityComplex,
		},
		{
			Name:        "OrderService",
			Description: "Order management service with persistence",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "PositionService",
			Description: "Position tracking and management service",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "ComplianceService",
			Description: "Compliance checking and reporting service",
			Complexity:  MigrationComplexityComplex,
		},
		{
			Name:        "NotificationService",
			Description: "Notification and alerting service",
			Complexity:  MigrationComplexitySimple,
		},
		{
			Name:        "WebSocketService",
			Description: "WebSocket connection management service",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "ExchangeService",
			Description: "Exchange connectivity service",
			Complexity:  MigrationComplexityComplex,
		},
		{
			Name:        "AnalyticsService",
			Description: "Analytics and reporting service",
			Complexity:  MigrationComplexityModerate,
		},
		{
			Name:        "BackupService",
			Description: "Data backup and recovery service",
			Complexity:  MigrationComplexitySimple,
		},
	}
}

// AnalyzeService analyzes an existing service and suggests migration approach
func (sm *ServiceMigrator) AnalyzeService(service interface{}) *ServiceAnalysis {
	analysis := &ServiceAnalysis{
		ServiceType: reflect.TypeOf(service).String(),
		Methods:     make([]string, 0),
		Complexity:  MigrationComplexitySimple,
	}
	
	// Use reflection to analyze service structure
	serviceValue := reflect.ValueOf(service)
	serviceType := reflect.TypeOf(service)
	
	// Handle pointer types
	if serviceType.Kind() == reflect.Ptr {
		serviceType = serviceType.Elem()
		serviceValue = serviceValue.Elem()
	}
	
	// Analyze methods
	for i := 0; i < serviceType.NumMethod(); i++ {
		method := serviceType.Method(i)
		analysis.Methods = append(analysis.Methods, method.Name)
		
		// Check for standard service methods
		switch method.Name {
		case "Start":
			analysis.HasStart = true
		case "Stop":
			analysis.HasStop = true
		case "Health", "HealthCheck":
			analysis.HasHealth = true
		case "Name":
			analysis.HasName = true
		case "Version":
			analysis.HasVersion = true
		}
	}
	
	// Analyze fields
	if serviceType.Kind() == reflect.Struct {
		for i := 0; i < serviceType.NumField(); i++ {
			field := serviceType.Field(i)
			analysis.Fields = append(analysis.Fields, field.Name)
			
			// Check for complex state
			if field.Type.Kind() == reflect.Chan ||
			   field.Type.Kind() == reflect.Map ||
			   (field.Type.Kind() == reflect.Slice && field.Type.Elem().Kind() == reflect.Struct) {
				analysis.Complexity = MigrationComplexityModerate
			}
		}
	}
	
	// Determine migration strategy
	analysis.MigrationStrategy = sm.determineMigrationStrategy(analysis)
	
	return analysis
}

// ServiceAnalysis contains the analysis results of a service
type ServiceAnalysis struct {
	ServiceType       string
	Methods           []string
	Fields            []string
	HasStart          bool
	HasStop           bool
	HasHealth         bool
	HasName           bool
	HasVersion        bool
	Complexity        MigrationComplexity
	MigrationStrategy MigrationStrategy
}

// MigrationStrategy defines the approach for migrating a service
type MigrationStrategy int

const (
	MigrationStrategyDirect MigrationStrategy = iota
	MigrationStrategyWrapper
	MigrationStrategyRefactor
	MigrationStrategyRewrite
)

// String returns the string representation of migration strategy
func (ms MigrationStrategy) String() string {
	switch ms {
	case MigrationStrategyDirect:
		return "direct"
	case MigrationStrategyWrapper:
		return "wrapper"
	case MigrationStrategyRefactor:
		return "refactor"
	case MigrationStrategyRewrite:
		return "rewrite"
	default:
		return "unknown"
	}
}

// determineMigrationStrategy determines the best migration strategy
func (sm *ServiceMigrator) determineMigrationStrategy(analysis *ServiceAnalysis) MigrationStrategy {
	// If service already has all required methods, direct migration
	if analysis.HasStart && analysis.HasStop && analysis.HasHealth && analysis.HasName && analysis.HasVersion {
		return MigrationStrategyDirect
	}
	
	// If service has some methods but not all, wrapper approach
	if analysis.HasStart || analysis.HasStop {
		return MigrationStrategyWrapper
	}
	
	// Based on complexity
	switch analysis.Complexity {
	case MigrationComplexitySimple:
		return MigrationStrategyWrapper
	case MigrationComplexityModerate:
		return MigrationStrategyRefactor
	case MigrationComplexityComplex:
		return MigrationStrategyRewrite
	default:
		return MigrationStrategyWrapper
	}
}

// GenerateMigrationPlan generates a migration plan for a service
func (sm *ServiceMigrator) GenerateMigrationPlan(analysis *ServiceAnalysis) *MigrationPlan {
	plan := &MigrationPlan{
		ServiceType: analysis.ServiceType,
		Strategy:    analysis.MigrationStrategy,
		Steps:       make([]MigrationStep, 0),
		EstimatedEffort: sm.estimateEffort(analysis),
	}
	
	switch analysis.MigrationStrategy {
	case MigrationStrategyDirect:
		plan.Steps = append(plan.Steps, MigrationStep{
			Description: "Replace service interface with ServiceInterface",
			Effort:      "Low",
		})
		
	case MigrationStrategyWrapper:
		plan.Steps = append(plan.Steps,
			MigrationStep{
				Description: "Create BaseService wrapper",
				Effort:      "Low",
			},
			MigrationStep{
				Description: "Implement missing interface methods",
				Effort:      "Medium",
			},
			MigrationStep{
				Description: "Update service registration",
				Effort:      "Low",
			},
		)
		
	case MigrationStrategyRefactor:
		plan.Steps = append(plan.Steps,
			MigrationStep{
				Description: "Analyze service dependencies",
				Effort:      "Medium",
			},
			MigrationStep{
				Description: "Refactor service structure",
				Effort:      "High",
			},
			MigrationStep{
				Description: "Implement BaseService pattern",
				Effort:      "Medium",
			},
			MigrationStep{
				Description: "Update tests and documentation",
				Effort:      "Medium",
			},
		)
		
	case MigrationStrategyRewrite:
		plan.Steps = append(plan.Steps,
			MigrationStep{
				Description: "Design new service architecture",
				Effort:      "High",
			},
			MigrationStep{
				Description: "Implement new service with BaseService",
				Effort:      "Very High",
			},
			MigrationStep{
				Description: "Migrate data and state",
				Effort:      "High",
			},
			MigrationStep{
				Description: "Comprehensive testing",
				Effort:      "High",
			},
		)
	}
	
	return plan
}

// MigrationPlan contains the plan for migrating a service
type MigrationPlan struct {
	ServiceType     string
	Strategy        MigrationStrategy
	Steps           []MigrationStep
	EstimatedEffort string
}

// MigrationStep represents a single step in the migration process
type MigrationStep struct {
	Description string
	Effort      string
}

// estimateEffort estimates the effort required for migration
func (sm *ServiceMigrator) estimateEffort(analysis *ServiceAnalysis) string {
	switch analysis.Complexity {
	case MigrationComplexitySimple:
		return "1-2 days"
	case MigrationComplexityModerate:
		return "3-5 days"
	case MigrationComplexityComplex:
		return "1-2 weeks"
	default:
		return "Unknown"
	}
}

// BatchMigrationPlan creates a plan for migrating multiple services
func (sm *ServiceMigrator) BatchMigrationPlan(services []interface{}) *BatchMigrationPlan {
	batchPlan := &BatchMigrationPlan{
		TotalServices: len(services),
		Plans:         make([]*MigrationPlan, 0, len(services)),
		CreatedAt:     time.Now(),
	}
	
	// Analyze each service
	for _, service := range services {
		analysis := sm.AnalyzeService(service)
		plan := sm.GenerateMigrationPlan(analysis)
		batchPlan.Plans = append(batchPlan.Plans, plan)
	}
	
	// Calculate totals
	batchPlan.calculateTotals()
	
	return batchPlan
}

// BatchMigrationPlan contains plans for migrating multiple services
type BatchMigrationPlan struct {
	TotalServices   int
	Plans           []*MigrationPlan
	SimpleCount     int
	ModerateCount   int
	ComplexCount    int
	EstimatedDays   int
	CreatedAt       time.Time
}

// calculateTotals calculates totals for the batch migration plan
func (bmp *BatchMigrationPlan) calculateTotals() {
	for _, plan := range bmp.Plans {
		switch plan.EstimatedEffort {
		case "1-2 days":
			bmp.SimpleCount++
			bmp.EstimatedDays += 2
		case "3-5 days":
			bmp.ModerateCount++
			bmp.EstimatedDays += 4
		case "1-2 weeks":
			bmp.ComplexCount++
			bmp.EstimatedDays += 10
		}
	}
}
