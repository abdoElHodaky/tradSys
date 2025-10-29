package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Service represents an order management service with optimized architecture
type Service struct {
	// Core components
	engine           *order_matching.Engine
	processorRegistry *ProcessorRegistry
	stateMachine     *OrderStateMachine
	errorMapper      *ErrorCodeMapper
	
	// Validators using composition pattern
	orderValidator    *OrderValidator
	businessValidator *BusinessRuleValidator
	riskValidator     *RiskValidator
	cancelValidator   *CancelValidator
	updateValidator   *UpdateValidator
	
	// Data storage
	orders           map[string]*Order
	userOrders       map[string]map[string]bool
	symbolOrders     map[string]map[string]bool
	clientOrderIDs   map[string]string
	orderCache       *cache.Cache
	
	// Concurrency control
	mu     sync.RWMutex
	logger *zap.Logger
	
	// Context management
	ctx    context.Context
	cancel context.CancelFunc
	
	// Batch processing
	orderBatchChan chan orderOperation
}

// NewService creates a new order service with optimized architecture
func NewService(engine *order_matching.Engine, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())
	
	service := &Service{
		// Initialize core components
		engine:           engine,
		processorRegistry: NewProcessorRegistry(),
		stateMachine:     NewOrderStateMachine(),
		errorMapper:      NewErrorCodeMapper(),
		
		// Initialize validators
		orderValidator:    NewOrderValidator(),
		businessValidator: NewBusinessRuleValidator(),
		riskValidator:     NewRiskValidator(),
		cancelValidator:   NewCancelValidator(),
		updateValidator:   NewUpdateValidator(),
		
		// Initialize data storage
		orders:         make(map[string]*Order),
		userOrders:     make(map[string]map[string]bool),
		symbolOrders:   make(map[string]map[string]bool),
		clientOrderIDs: make(map[string]string),
		orderCache:     cache.New(5*time.Minute, 10*time.Minute),
		
		// Initialize concurrency control
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
		
		// Initialize batch processing
		orderBatchChan: make(chan orderOperation, 1000),
	}
	
	// Start batch processor
	go service.processBatchOperations()
	
	return service
}

// CreateOrder creates a new order using early return pattern and polymorphism
func (s *Service) CreateOrder(req *OrderRequest) (*Order, error) {
	if req == nil {
		return nil, errors.New("order request cannot be nil")
	}
	
	// Validate request using early return pattern
	if err := s.validateOrderRequest(req); err != nil {
		return nil, fmt.Errorf("validation failed: %w", err)
	}
	
	// Create order
	order := s.createOrderFromRequest(req)
	
	// Process order using polymorphism instead of switch statement
	if err := s.processorRegistry.ProcessOrder(order); err != nil {
		return nil, fmt.Errorf("processing failed: %w", err)
	}
	
	// Store order
	if err := s.storeOrder(order); err != nil {
		return nil, fmt.Errorf("storage failed: %w", err)
	}
	
	s.logger.Info("Order created successfully", 
		zap.String("order_id", order.ID),
		zap.String("user_id", order.UserID),
		zap.String("symbol", order.Symbol))
	
	return order, nil
}

// validateOrderRequest validates order request using composition of validators
func (s *Service) validateOrderRequest(req *OrderRequest) error {
	// Basic validation using early returns
	if err := s.orderValidator.Validate(req); err != nil {
		return err
	}
	
	// Business rule validation
	if err := s.businessValidator.ValidateBusinessRules(req); err != nil {
		return err
	}
	
	// Risk validation (simplified - would get current position from database)
	currentPosition := s.getCurrentPosition(req.UserID, req.Symbol)
	if err := s.riskValidator.ValidateRiskConstraints(req, currentPosition); err != nil {
		return err
	}
	
	return nil
}

// createOrderFromRequest creates an order from request
func (s *Service) createOrderFromRequest(req *OrderRequest) *Order {
	return &Order{
		ID:            uuid.New().String(),
		UserID:        req.UserID,
		ClientOrderID: req.ClientOrderID,
		Symbol:        req.Symbol,
		Side:          req.Side,
		Type:          req.Type,
		Price:         req.Price,
		StopPrice:     req.StopPrice,
		Quantity:      req.Quantity,
		Status:        OrderStatusNew,
		TimeInForce:   req.TimeInForce,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
		ExpiresAt:     req.ExpiresAt,
		Trades:        make([]*Trade, 0),
		Metadata:      req.Metadata,
	}
}

// storeOrder stores an order in the service
func (s *Service) storeOrder(order *Order) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// Store in main map
	s.orders[order.ID] = order
	
	// Index by user
	if s.userOrders[order.UserID] == nil {
		s.userOrders[order.UserID] = make(map[string]bool)
	}
	s.userOrders[order.UserID][order.ID] = true
	
	// Index by symbol
	if s.symbolOrders[order.Symbol] == nil {
		s.symbolOrders[order.Symbol] = make(map[string]bool)
	}
	s.symbolOrders[order.Symbol][order.ID] = true
	
	// Index by client order ID if provided
	if order.ClientOrderID != "" {
		s.clientOrderIDs[order.ClientOrderID] = order.ID
	}
	
	// Cache frequently accessed orders
	s.orderCache.Set(order.ID, order, cache.DefaultExpiration)
	
	return nil
}

// GetOrder retrieves an order by ID using early return pattern
func (s *Service) GetOrder(orderID string) (*Order, error) {
	if orderID == "" {
		return nil, errors.New("order ID cannot be empty")
	}
	
	// Try cache first
	if cached, found := s.orderCache.Get(orderID); found {
		if order, ok := cached.(*Order); ok {
			return order, nil
		}
	}
	
	// Fallback to main storage
	s.mu.RLock()
	order, exists := s.orders[orderID]
	s.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}
	
	// Update cache
	s.orderCache.Set(orderID, order, cache.DefaultExpiration)
	
	return order, nil
}

// Note: Order operations (CancelOrder, UpdateOrder, ListOrders) are now in core_operations.go

// Helper methods

// getCurrentPosition gets current position for risk validation
func (s *Service) getCurrentPosition(userID, symbol string) float64 {
	// Simplified implementation - would query database in real system
	return 0.0
}

// processBatchOperations processes batch operations asynchronously
func (s *Service) processBatchOperations() {
	for {
		select {
		case op := <-s.orderBatchChan:
			s.processBatchOperation(op)
		case <-s.ctx.Done():
			return
		}
	}
}

// processBatchOperation processes a single batch operation
func (s *Service) processBatchOperation(op orderOperation) {
	var result orderOperationResult
	
	switch op.opType {
	case "CREATE":
		// Process create operation
		result.order = op.order
		result.err = nil
	case "UPDATE":
		// Process update operation
		result.order = op.order
		result.err = nil
	case "CANCEL":
		// Process cancel operation
		result.order = op.order
		result.err = nil
	default:
		result.err = fmt.Errorf("unknown operation type: %s", op.opType)
	}
	
	// Send result back
	select {
	case op.resultCh <- result:
	case <-time.After(time.Second):
		s.logger.Warn("Failed to send batch operation result", 
			zap.String("op_type", op.opType),
			zap.String("request_id", op.requestID))
	}
}

// Close closes the service and cleans up resources
func (s *Service) Close() error {
	s.cancel()
	close(s.orderBatchChan)
	
	s.logger.Info("Order service closed successfully")
	return nil
}
