package common

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ServiceRegistry manages the lifecycle of multiple services
type ServiceRegistry struct {
	services map[string]ServiceInterface
	mu       sync.RWMutex
	logger   *zap.Logger
	
	// Startup order management
	startupOrder []string
	dependencies map[string][]string
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(logger *zap.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		services:     make(map[string]ServiceInterface),
		logger:       logger,
		startupOrder: make([]string, 0),
		dependencies: make(map[string][]string),
	}
}

// Register registers a service with the registry
func (sr *ServiceRegistry) Register(name string, service ServiceInterface) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if _, exists := sr.services[name]; exists {
		return fmt.Errorf("service %s already registered", name)
	}
	
	sr.services[name] = service
	sr.startupOrder = append(sr.startupOrder, name)
	
	sr.logger.Info("Service registered", 
		zap.String("name", name),
		zap.String("version", service.Version()),
	)
	
	return nil
}

// Unregister removes a service from the registry
func (sr *ServiceRegistry) Unregister(name string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	if _, exists := sr.services[name]; !exists {
		return fmt.Errorf("service %s not found", name)
	}
	
	delete(sr.services, name)
	
	// Remove from startup order
	for i, serviceName := range sr.startupOrder {
		if serviceName == name {
			sr.startupOrder = append(sr.startupOrder[:i], sr.startupOrder[i+1:]...)
			break
		}
	}
	
	// Remove dependencies
	delete(sr.dependencies, name)
	for service, deps := range sr.dependencies {
		for i, dep := range deps {
			if dep == name {
				sr.dependencies[service] = append(deps[:i], deps[i+1:]...)
				break
			}
		}
	}
	
	sr.logger.Info("Service unregistered", zap.String("name", name))
	return nil
}

// Get retrieves a service by name
func (sr *ServiceRegistry) Get(name string) (ServiceInterface, error) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	service, exists := sr.services[name]
	if !exists {
		return nil, fmt.Errorf("service %s not found", name)
	}
	
	return service, nil
}

// List returns all registered service names
func (sr *ServiceRegistry) List() []string {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	names := make([]string, 0, len(sr.services))
	for name := range sr.services {
		names = append(names, name)
	}
	
	return names
}

// SetDependency sets a dependency relationship between services
func (sr *ServiceRegistry) SetDependency(service, dependency string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()
	
	// Check if both services exist
	if _, exists := sr.services[service]; !exists {
		return fmt.Errorf("service %s not found", service)
	}
	if _, exists := sr.services[dependency]; !exists {
		return fmt.Errorf("dependency service %s not found", dependency)
	}
	
	// Check for circular dependencies
	if sr.hasCircularDependency(service, dependency) {
		return fmt.Errorf("circular dependency detected between %s and %s", service, dependency)
	}
	
	if sr.dependencies[service] == nil {
		sr.dependencies[service] = make([]string, 0)
	}
	
	// Add dependency if not already present
	for _, dep := range sr.dependencies[service] {
		if dep == dependency {
			return nil // Already exists
		}
	}
	
	sr.dependencies[service] = append(sr.dependencies[service], dependency)
	
	sr.logger.Info("Dependency set",
		zap.String("service", service),
		zap.String("dependency", dependency),
	)
	
	return nil
}

// hasCircularDependency checks for circular dependencies
func (sr *ServiceRegistry) hasCircularDependency(service, newDep string) bool {
	visited := make(map[string]bool)
	return sr.checkCircular(newDep, service, visited)
}

// checkCircular recursively checks for circular dependencies
func (sr *ServiceRegistry) checkCircular(current, target string, visited map[string]bool) bool {
	if current == target {
		return true
	}
	
	if visited[current] {
		return false
	}
	
	visited[current] = true
	
	for _, dep := range sr.dependencies[current] {
		if sr.checkCircular(dep, target, visited) {
			return true
		}
	}
	
	return false
}

// StartAll starts all services in dependency order
func (sr *ServiceRegistry) StartAll(ctx context.Context) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	sr.logger.Info("Starting all services", zap.Int("count", len(sr.services)))
	
	// Calculate startup order based on dependencies
	startOrder, err := sr.calculateStartupOrder()
	if err != nil {
		return fmt.Errorf("failed to calculate startup order: %w", err)
	}
	
	// Start services in order
	for _, serviceName := range startOrder {
		service := sr.services[serviceName]
		
		sr.logger.Info("Starting service", zap.String("name", serviceName))
		
		if err := service.Start(ctx); err != nil {
			sr.logger.Error("Failed to start service",
				zap.String("name", serviceName),
				zap.Error(err),
			)
			
			// Stop already started services
			sr.stopStartedServices(ctx, startOrder, serviceName)
			return fmt.Errorf("failed to start service %s: %w", serviceName, err)
		}
		
		sr.logger.Info("Service started successfully", zap.String("name", serviceName))
	}
	
	sr.logger.Info("All services started successfully")
	return nil
}

// StopAll stops all services in reverse dependency order
func (sr *ServiceRegistry) StopAll(ctx context.Context) error {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	sr.logger.Info("Stopping all services", zap.Int("count", len(sr.services)))
	
	// Calculate startup order and reverse it for shutdown
	startOrder, err := sr.calculateStartupOrder()
	if err != nil {
		return fmt.Errorf("failed to calculate shutdown order: %w", err)
	}
	
	// Reverse the order for shutdown
	for i := len(startOrder) - 1; i >= 0; i-- {
		serviceName := startOrder[i]
		service := sr.services[serviceName]
		
		sr.logger.Info("Stopping service", zap.String("name", serviceName))
		
		if err := service.Stop(ctx); err != nil {
			sr.logger.Error("Failed to stop service",
				zap.String("name", serviceName),
				zap.Error(err),
			)
			// Continue stopping other services even if one fails
		} else {
			sr.logger.Info("Service stopped successfully", zap.String("name", serviceName))
		}
	}
	
	sr.logger.Info("All services stopped")
	return nil
}

// calculateStartupOrder calculates the order in which services should be started
func (sr *ServiceRegistry) calculateStartupOrder() ([]string, error) {
	// Topological sort to handle dependencies
	inDegree := make(map[string]int)
	adjList := make(map[string][]string)
	
	// Initialize
	for serviceName := range sr.services {
		inDegree[serviceName] = 0
		adjList[serviceName] = make([]string, 0)
	}
	
	// Build adjacency list and calculate in-degrees
	for service, deps := range sr.dependencies {
		for _, dep := range deps {
			adjList[dep] = append(adjList[dep], service)
			inDegree[service]++
		}
	}
	
	// Kahn's algorithm for topological sorting
	queue := make([]string, 0)
	for service, degree := range inDegree {
		if degree == 0 {
			queue = append(queue, service)
		}
	}
	
	result := make([]string, 0)
	
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)
		
		for _, neighbor := range adjList[current] {
			inDegree[neighbor]--
			if inDegree[neighbor] == 0 {
				queue = append(queue, neighbor)
			}
		}
	}
	
	// Check for circular dependencies
	if len(result) != len(sr.services) {
		return nil, fmt.Errorf("circular dependency detected")
	}
	
	return result, nil
}

// stopStartedServices stops services that were already started during a failed startup
func (sr *ServiceRegistry) stopStartedServices(ctx context.Context, startOrder []string, failedService string) {
	for _, serviceName := range startOrder {
		if serviceName == failedService {
			break
		}
		
		service := sr.services[serviceName]
		if err := service.Stop(ctx); err != nil {
			sr.logger.Error("Failed to stop service during cleanup",
				zap.String("name", serviceName),
				zap.Error(err),
			)
		}
	}
}

// HealthCheck returns the health status of all services
func (sr *ServiceRegistry) HealthCheck() map[string]HealthStatus {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	health := make(map[string]HealthStatus)
	for name, service := range sr.services {
		health[name] = service.Health()
	}
	
	return health
}

// GetOverallHealth returns the overall health status of the system
func (sr *ServiceRegistry) GetOverallHealth() HealthStatus {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	overallStatus := "healthy"
	unhealthyServices := make([]string, 0)
	degradedServices := make([]string, 0)
	
	for name, service := range sr.services {
		health := service.Health()
		switch health.Status {
		case "unhealthy":
			overallStatus = "unhealthy"
			unhealthyServices = append(unhealthyServices, name)
		case "degraded":
			if overallStatus == "healthy" {
				overallStatus = "degraded"
			}
			degradedServices = append(degradedServices, name)
		}
	}
	
	message := "All services healthy"
	if len(unhealthyServices) > 0 {
		message = fmt.Sprintf("Unhealthy services: %v", unhealthyServices)
	} else if len(degradedServices) > 0 {
		message = fmt.Sprintf("Degraded services: %v", degradedServices)
	}
	
	return HealthStatus{
		Status:    overallStatus,
		Message:   message,
		Timestamp: time.Now(),
		Details: map[string]string{
			"total_services":     fmt.Sprintf("%d", len(sr.services)),
			"unhealthy_services": fmt.Sprintf("%d", len(unhealthyServices)),
			"degraded_services":  fmt.Sprintf("%d", len(degradedServices)),
		},
	}
}
