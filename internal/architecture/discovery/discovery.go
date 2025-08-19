package discovery

import (
	"context"
	"sync"
	"time"

	"github.com/asim/go-micro/v3/registry"
	"go.uber.org/zap"
)

// ServiceDiscovery provides service discovery functionality
type ServiceDiscovery struct {
	registry registry.Registry
	logger   *zap.Logger
	cache    map[string][]*registry.Service
	cacheTTL time.Duration
	cacheMu  sync.RWMutex
}

// NewServiceDiscovery creates a new service discovery
func NewServiceDiscovery(registry registry.Registry, logger *zap.Logger) *ServiceDiscovery {
	return &ServiceDiscovery{
		registry: registry,
		logger:   logger,
		cache:    make(map[string][]*registry.Service),
		cacheTTL: 30 * time.Second,
	}
}

// RegisterService registers a service with the registry
func (d *ServiceDiscovery) RegisterService(ctx context.Context, service *registry.Service) error {
	d.logger.Info("Registering service",
		zap.String("name", service.Name),
		zap.Int("nodes", len(service.Nodes)))

	return d.registry.Register(service)
}

// DeregisterService deregisters a service from the registry
func (d *ServiceDiscovery) DeregisterService(ctx context.Context, service *registry.Service) error {
	d.logger.Info("Deregistering service",
		zap.String("name", service.Name),
		zap.Int("nodes", len(service.Nodes)))

	return d.registry.Deregister(service)
}

// GetService gets a service from the registry
func (d *ServiceDiscovery) GetService(ctx context.Context, name string) ([]*registry.Service, error) {
	// Check the cache
	d.cacheMu.RLock()
	services, ok := d.cache[name]
	d.cacheMu.RUnlock()

	if ok {
		return services, nil
	}

	// Get the service from the registry
	services, err := d.registry.GetService(name)
	if err != nil {
		return nil, err
	}

	// Update the cache
	d.cacheMu.Lock()
	d.cache[name] = services
	d.cacheMu.Unlock()

	// Start a goroutine to clear the cache after TTL
	go func() {
		time.Sleep(d.cacheTTL)
		d.cacheMu.Lock()
		delete(d.cache, name)
		d.cacheMu.Unlock()
	}()

	return services, nil
}

// ListServices lists all services in the registry
func (d *ServiceDiscovery) ListServices(ctx context.Context) ([]*registry.Service, error) {
	return d.registry.ListServices()
}

// Watch watches for changes in the registry
func (d *ServiceDiscovery) Watch(ctx context.Context, name string) (registry.Watcher, error) {
	return d.registry.Watch(registry.WatchService(name))
}

// ServiceSelector provides service selection functionality
type ServiceSelector struct {
	discovery *ServiceDiscovery
	logger    *zap.Logger
	strategy  SelectionStrategy
}

// SelectionStrategy represents a strategy for selecting a service node
type SelectionStrategy interface {
	Select(nodes []*registry.Node) (*registry.Node, error)
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
func (s *ServiceSelector) Select(ctx context.Context, name string) (*registry.Node, error) {
	// Get the service
	services, err := s.discovery.GetService(ctx, name)
	if err != nil {
		return nil, err
	}

	// Check if the service has nodes
	if len(services) == 0 || len(services[0].Nodes) == 0 {
		return nil, registry.ErrNotFound
	}

	// Select a node using the strategy
	return s.strategy.Select(services[0].Nodes)
}

// RoundRobinStrategy provides round-robin selection strategy
type RoundRobinStrategy struct {
	index int
	mu    sync.Mutex
}

// NewRoundRobinStrategy creates a new round-robin strategy
func NewRoundRobinStrategy() *RoundRobinStrategy {
	return &RoundRobinStrategy{
		index: 0,
	}
}

// Select selects a node using round-robin strategy
func (s *RoundRobinStrategy) Select(nodes []*registry.Node) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, registry.ErrNotFound
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Get the next node
	node := nodes[s.index%len(nodes)]

	// Increment the index
	s.index++

	return node, nil
}

// RandomStrategy provides random selection strategy
type RandomStrategy struct{}

// NewRandomStrategy creates a new random strategy
func NewRandomStrategy() *RandomStrategy {
	return &RandomStrategy{}
}

// Select selects a node using random strategy
func (s *RandomStrategy) Select(nodes []*registry.Node) (*registry.Node, error) {
	if len(nodes) == 0 {
		return nil, registry.ErrNotFound
	}

	// Get a random node
	return nodes[time.Now().UnixNano()%int64(len(nodes))], nil
}

