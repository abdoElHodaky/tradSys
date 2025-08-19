package discovery

import (
	"context"
	"errors"
	"sync"
	"time"

	"go-micro.dev/v4/registry"
	"go.uber.org/zap"
)

// Common errors
var (
	ErrServiceNotFound = errors.New("service not found")
	ErrNoHealthyNodes  = errors.New("no healthy nodes available")
)

// ServiceDiscovery provides service discovery functionality
type ServiceDiscovery struct {
	registry registry.Registry
	logger   *zap.Logger
	cache    map[string][]*registry.Service
	cacheMu  sync.RWMutex
	ttl      time.Duration
	lastSync time.Time
}

// NewServiceDiscovery creates a new service discovery instance
func NewServiceDiscovery(reg registry.Registry, logger *zap.Logger) *ServiceDiscovery {
	sd := &ServiceDiscovery{
		registry: reg,
		logger:   logger,
		cache:    make(map[string][]*registry.Service),
		ttl:      time.Minute, // Default TTL for cache entries
	}

	// Start a background goroutine to refresh the cache
	go sd.refreshCache()

	return sd
}

// refreshCache periodically refreshes the service cache
func (sd *ServiceDiscovery) refreshCache() {
	ticker := time.NewTicker(sd.ttl / 2)
	defer ticker.Stop()

	for range ticker.C {
		sd.cacheMu.Lock()
		// Clear the cache
		sd.cache = make(map[string][]*registry.Service)
		sd.lastSync = time.Now()
		sd.cacheMu.Unlock()

		sd.logger.Debug("Refreshed service discovery cache")
	}
}

// GetService gets a service by name
func (sd *ServiceDiscovery) GetService(ctx context.Context, name string) ([]*registry.Service, error) {
	// Check the cache first
	sd.cacheMu.RLock()
	if services, ok := sd.cache[name]; ok && time.Since(sd.lastSync) < sd.ttl {
		sd.cacheMu.RUnlock()
		return services, nil
	}
	sd.cacheMu.RUnlock()

	// Cache miss or expired, get from registry
	services, err := sd.registry.GetService(name)
	if err != nil {
		sd.logger.Error("Failed to get service from registry", 
			zap.String("service", name), 
			zap.Error(err))
		return nil, err
	}

	if len(services) == 0 {
		return nil, ErrServiceNotFound
	}

	// Update the cache
	sd.cacheMu.Lock()
	sd.cache[name] = services
	sd.cacheMu.Unlock()

	return services, nil
}

// ListServices lists all services
func (sd *ServiceDiscovery) ListServices(ctx context.Context) ([]*registry.Service, error) {
	services, err := sd.registry.ListServices()
	if err != nil {
		sd.logger.Error("Failed to list services from registry", zap.Error(err))
		return nil, err
	}

	return services, nil
}

// RegisterService registers a service with the registry
func (sd *ServiceDiscovery) RegisterService(ctx context.Context, service *registry.Service) error {
	err := sd.registry.Register(service)
	if err != nil {
		sd.logger.Error("Failed to register service", 
			zap.String("service", service.Name), 
			zap.Error(err))
		return err
	}

	sd.logger.Info("Registered service", zap.String("service", service.Name))
	return nil
}

// DeregisterService deregisters a service from the registry
func (sd *ServiceDiscovery) DeregisterService(ctx context.Context, service *registry.Service) error {
	err := sd.registry.Deregister(service)
	if err != nil {
		sd.logger.Error("Failed to deregister service", 
			zap.String("service", service.Name), 
			zap.Error(err))
		return err
	}

	sd.logger.Info("Deregistered service", zap.String("service", service.Name))
	return nil
}

// Watch watches for service changes
func (sd *ServiceDiscovery) Watch(ctx context.Context, service string) (registry.Watcher, error) {
	return sd.registry.Watch(registry.WatchService(service))
}

// ServiceSelector provides service selection functionality
type ServiceSelector struct {
	discovery *ServiceDiscovery
	logger    *zap.Logger
	strategy  SelectionStrategy
}

// SelectionStrategy defines the interface for service selection strategies
type SelectionStrategy interface {
	Select(services []*registry.Service) (*registry.Node, error)
}

// NewServiceSelector creates a new service selector
func NewServiceSelector(discovery *ServiceDiscovery, logger *zap.Logger, strategy SelectionStrategy) *ServiceSelector {
	return &ServiceSelector{
		discovery: discovery,
		logger:    logger,
		strategy:  strategy,
	}
}

// Select selects a node for a service
func (ss *ServiceSelector) Select(ctx context.Context, service string) (*registry.Node, error) {
	services, err := ss.discovery.GetService(ctx, service)
	if err != nil {
		return nil, err
	}

	node, err := ss.strategy.Select(services)
	if err != nil {
		ss.logger.Error("Failed to select node", 
			zap.String("service", service), 
			zap.Error(err))
		return nil, err
	}

	return node, nil
}

// RoundRobinStrategy implements a round-robin selection strategy
type RoundRobinStrategy struct {
	counters map[string]int
	mu       sync.Mutex
}

// NewRoundRobinStrategy creates a new round-robin selection strategy
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		counters: make(map[string]int),
	}
}

// Select selects a node using round-robin
func (s *RoundRobinStrategy) Select(services []*registry.Service) (*registry.Node, error) {
	if len(services) == 0 {
		return nil, ErrServiceNotFound
	}

	// Flatten nodes from all services
	var nodes []*registry.Node
	for _, service := range services {
		for _, node := range service.Nodes {
			if node.Metadata["status"] == "healthy" {
				nodes = append(nodes, node)
			}
		}
	}

	if len(nodes) == 0 {
		return nil, ErrNoHealthyNodes
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the counter for this service
	counter := s.counters[services[0].Name]
	
	// Select the node
	node := nodes[counter%len(nodes)]
	
	// Increment the counter
	s.counters[services[0].Name] = (counter + 1) % len(nodes)
	
	return node, nil
}

// RandomStrategy implements a random selection strategy
type RandomStrategy struct{}

// NewRandomStrategy creates a new random selection strategy
func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

// Select selects a node randomly
func (s *RandomStrategy) Select(services []*registry.Service) (*registry.Node, error) {
	if len(services) == 0 {
		return nil, ErrServiceNotFound
	}

	// Flatten nodes from all services
	var nodes []*registry.Node
	for _, service := range services {
		for _, node := range service.Nodes {
			if node.Metadata["status"] == "healthy" {
				nodes = append(nodes, node)
			}
		}
	}

	if len(nodes) == 0 {
		return nil, ErrNoHealthyNodes
	}

	// Select a random node
	return nodes[time.Now().UnixNano()%int64(len(nodes))], nil
}

