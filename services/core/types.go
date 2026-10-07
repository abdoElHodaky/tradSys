package core

import (
	"context"
	"sync"

	"google.golang.org/grpc/health/grpc_health_v1"
)

// ServiceDiscovery handles service discovery in the mesh
type ServiceDiscovery struct {
	services map[string]*ServiceNode
	mu       sync.RWMutex
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery() *ServiceDiscovery {
	return &ServiceDiscovery{
		services: make(map[string]*ServiceNode),
	}
}

// Register adds a service to the discovery registry
func (sd *ServiceDiscovery) Register(service *ServiceNode) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	sd.services[service.ID] = service
	return nil
}

// Deregister removes a service from the discovery registry
func (sd *ServiceDiscovery) Deregister(serviceID string) error {
	sd.mu.Lock()
	defer sd.mu.Unlock()

	delete(sd.services, serviceID)
	return nil
}

// Discover finds services by type
func (sd *ServiceDiscovery) Discover(serviceType ServiceType) []*ServiceNode {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	var services []*ServiceNode
	for _, service := range sd.services {
		if service.ServiceType == serviceType {
			services = append(services, service)
		}
	}
	return services
}

// GetService retrieves a specific service by ID
func (sd *ServiceDiscovery) GetService(serviceID string) *ServiceNode {
	sd.mu.RLock()
	defer sd.mu.RUnlock()

	return sd.services[serviceID]
}

// LoadBalancer handles load balancing for service requests
type LoadBalancer struct {
	mu sync.RWMutex
}

// NewLoadBalancer creates a new load balancer instance
func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{}
}

// SelectService selects a service instance for load balancing
func (lb *LoadBalancer) SelectService(services []*ServiceNode) *ServiceNode {
	if len(services) == 0 {
		return nil
	}

	// Return healthy service or first one
	for _, s := range services {
		if s.Status == ServiceStatusHealthy {
			return s
		}
	}

	return services[0]
}

// HealthChecker monitors service health
type HealthChecker struct {
	checks map[string]bool
	mu     sync.RWMutex
}

// NewHealthChecker creates a new health checker instance
func NewHealthChecker() *HealthChecker {
	return &HealthChecker{
		checks: make(map[string]bool),
	}
}

// CheckHealth performs a health check on a service
func (hc *HealthChecker) CheckHealth(service *ServiceNode) bool {
	if service == nil || service.Conn == nil {
		return false
	}

	// Create health client and check
	client := grpc_health_v1.NewHealthClient(service.Conn)
	ctx := context.Background()

	resp, err := client.Check(ctx, &grpc_health_v1.HealthCheckRequest{})
	if err != nil {
		hc.mu.Lock()
		hc.checks[service.ID] = false
		hc.mu.Unlock()
		return false
	}

	healthy := resp.GetStatus() == grpc_health_v1.HealthCheckResponse_SERVING
	hc.mu.Lock()
	hc.checks[service.ID] = healthy
	hc.mu.Unlock()

	return healthy
}

// MetricsCollector collects metrics from services
type MetricsCollector struct {
	mu sync.Mutex
}

// NewMetricsCollector creates a new metrics collector instance
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// Collect gathers metrics from a service
func (mc *MetricsCollector) Collect(service *ServiceNode) map[string]interface{} {
	metrics := make(map[string]interface{})

	if service == nil {
		return metrics
	}

	metrics["service_id"] = service.ID
	metrics["service_name"] = service.Name
	metrics["status"] = service.Status
	metrics["last_seen"] = service.LastSeen

	return metrics
}