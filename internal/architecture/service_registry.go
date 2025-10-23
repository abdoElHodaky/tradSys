package architecture

import (
	"reflect"
	"sync"
)

// ServiceRegistry provides a centralized registry for service discovery
// and dependency management in the HFT platform
type ServiceRegistry struct {
	services map[string]interface{}
	mu       sync.RWMutex
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]interface{}),
	}
}

// Register registers a service with the registry
func (r *ServiceRegistry) Register(name string, service interface{}) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = service
}

// Get retrieves a service from the registry
func (r *ServiceRegistry) Get(name string) (interface{}, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	service, exists := r.services[name]
	return service, exists
}

// GetTyped retrieves a service from the registry with type assertion
func (r *ServiceRegistry) GetTyped(name string, target interface{}) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	service, exists := r.services[name]
	if !exists {
		return false
	}

	// Use type assertion to set the target pointer
	switch ptr := target.(type) {
	case *interface{}:
		*ptr = service
		return true
	default:
		// For other types, we need to check if the service can be assigned to the target type
		targetValue := reflect.ValueOf(target)
		if targetValue.Kind() != reflect.Ptr || targetValue.IsNil() {
			return false
		}

		serviceValue := reflect.ValueOf(service)
		targetElemType := targetValue.Elem().Type()

		if !serviceValue.Type().AssignableTo(targetElemType) {
			return false
		}

		targetValue.Elem().Set(serviceValue)
		return true
	}
}

// Unregister removes a service from the registry
func (r *ServiceRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}

// ListServices returns a list of all registered service names
func (r *ServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	services := make([]string, 0, len(r.services))
	for name := range r.services {
		services = append(services, name)
	}

	return services
}
