package lazy

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/trading/order_management"
	"github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
)

// LazyOrderService is a lazy-loaded wrapper for the order management service
type LazyOrderService struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name
	componentName string
	
	// Configuration
	config order_management.OrderServiceConfig
	
	// Logger
	logger *zap.Logger
}

// NewLazyOrderService creates a new lazy-loaded order service
func NewLazyOrderService(
	coordinator *coordination.ComponentCoordinator,
	config order_management.OrderServiceConfig,
	logger *zap.Logger,
) (*LazyOrderService, error) {
	componentName := "order-management-service"
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		return order_management.NewOrderService(config, log)
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(100*1024*1024), // 100MB estimate
		lazy.WithTimeout(20*time.Second),
		lazy.WithPriority(20), // High priority
	)
	
	// Register with the coordinator
	err := coordinator.RegisterComponent(
		componentName,
		"order-service",
		provider,
		[]string{}, // No dependencies
	)
	
	if err != nil {
		return nil, err
	}
	
	return &LazyOrderService{
		coordinator:   coordinator,
		componentName: componentName,
		config:        config,
		logger:        logger,
	}, nil
}

// CreateOrder creates a new order
func (s *LazyOrderService) CreateOrder(
	ctx context.Context,
	order *orders.Order,
) (*orders.OrderResponse, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(order_management.OrderService)
	if !ok {
		return nil, order_management.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.CreateOrder(ctx, order)
}

// GetOrder gets an order by ID
func (s *LazyOrderService) GetOrder(
	ctx context.Context,
	orderID string,
) (*orders.OrderResponse, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(order_management.OrderService)
	if !ok {
		return nil, order_management.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.GetOrder(ctx, orderID)
}

// UpdateOrder updates an order
func (s *LazyOrderService) UpdateOrder(
	ctx context.Context,
	orderID string,
	updates *orders.OrderUpdate,
) (*orders.OrderResponse, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(order_management.OrderService)
	if !ok {
		return nil, order_management.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.UpdateOrder(ctx, orderID, updates)
}

// CancelOrder cancels an order
func (s *LazyOrderService) CancelOrder(
	ctx context.Context,
	orderID string,
) (*orders.OrderResponse, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(order_management.OrderService)
	if !ok {
		return nil, order_management.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.CancelOrder(ctx, orderID)
}

// ListOrders lists orders
func (s *LazyOrderService) ListOrders(
	ctx context.Context,
	filter *orders.OrderFilter,
) ([]*orders.OrderResponse, error) {
	// Get the underlying service
	serviceInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return nil, err
	}
	
	// Cast to the actual service type
	service, ok := serviceInterface.(order_management.OrderService)
	if !ok {
		return nil, order_management.ErrInvalidServiceType
	}
	
	// Call the actual method
	return service.ListOrders(ctx, filter)
}

// Shutdown shuts down the service
func (s *LazyOrderService) Shutdown(ctx context.Context) error {
	return s.coordinator.ShutdownComponent(ctx, s.componentName)
}

