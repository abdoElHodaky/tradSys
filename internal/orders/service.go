package orders

import (
	"context"
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/google/uuid"
	cache "github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// Note: Type definitions moved to types.go to avoid duplication













// Service represents an order management service
type Service struct {
	// Engine is the order matching engine
	Engine *order_matching.Engine
	// Orders is a map of order ID to order
	Orders map[string]*Order
	// UserOrders is a map of user ID to order IDs
	UserOrders map[string]map[string]bool
	// SymbolOrders is a map of symbol to order IDs
	SymbolOrders map[string]map[string]bool
	// ClientOrderIDs is a map of client order ID to order ID
	ClientOrderIDs map[string]string
	// OrderCache is a cache for frequently accessed orders
	OrderCache *cache.Cache
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Context
	ctx context.Context
	// Cancel function
	cancel context.CancelFunc
	// Batch processing channel for order operations
	orderBatchChan chan orderOperation
}

// Note: orderOperation and orderOperationResult types moved to batch_processor.go

// NewService creates a new order management service
func NewService(engine *order_matching.Engine, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		Engine:         engine,
		Orders:         make(map[string]*Order),
		UserOrders:     make(map[string]map[string]bool),
		SymbolOrders:   make(map[string]map[string]bool),
		ClientOrderIDs: make(map[string]string),
		OrderCache:     cache.New(5*time.Minute, 10*time.Minute),
		logger:         logger,
		ctx:            ctx,
		cancel:         cancel,
		orderBatchChan: make(chan orderOperation, 1000),
	}

	// Start order expiry checker
	go service.checkOrderExpiry()

	// Start batch processor
	go service.processBatchOperations()

	return service
}

// processBatchOperations processes batch operations for orders
func (s *Service) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]orderOperation, 0, 100)

	for {
		select {
		case <-s.ctx.Done():
			return
		case op := <-s.orderBatchChan:
			batch = append(batch, op)

			// Process batch if it's full
			if len(batch) >= 100 {
				s.processBatch(batch)
				batch = make([]orderOperation, 0, 100)
			}
		case <-ticker.C:
			// Process remaining operations in batch
			if len(batch) > 0 {
				s.processBatch(batch)
				batch = make([]orderOperation, 0, 100)
			}
		}
	}
}

// processBatch processes a batch of order operations
func (s *Service) processBatch(batch []orderOperation) {
	// Group operations by type
	addOps := make([]orderOperation, 0)
	updateOps := make([]orderOperation, 0)
	cancelOps := make([]orderOperation, 0)

	for _, op := range batch {
		switch op.opType {
		case "add":
			addOps = append(addOps, op)
		case "update":
			updateOps = append(updateOps, op)
		case "cancel":
			cancelOps = append(cancelOps, op)
		}
	}

	// Process add operations
	if len(addOps) > 0 {
		s.processAddBatch(addOps)
	}

	// Process update operations
	if len(updateOps) > 0 {
		s.processUpdateBatch(updateOps)
	}

	// Process cancel operations
	if len(cancelOps) > 0 {
		s.processCancelBatch(cancelOps)
	}
}

// processAddBatch processes a batch of add operations
func (s *Service) processAddBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		order := op.order

		// Add order to maps
		s.Orders[order.ID] = order

		// Add to user orders
		if _, exists := s.UserOrders[order.UserID]; !exists {
			s.UserOrders[order.UserID] = make(map[string]bool)
		}
		s.UserOrders[order.UserID][order.ID] = true

		// Add to symbol orders
		if _, exists := s.SymbolOrders[order.Symbol]; !exists {
			s.SymbolOrders[order.Symbol] = make(map[string]bool)
		}
		s.SymbolOrders[order.Symbol][order.ID] = true

		// Add to client order IDs
		if order.ClientOrderID != "" {
			s.ClientOrderIDs[order.ClientOrderID] = order.ID
		}

		// Add to cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)
		if order.ClientOrderID != "" {
			s.OrderCache.Set("client:"+order.ClientOrderID, order.ID, cache.DefaultExpiration)
		}

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// processUpdateBatch processes a batch of update operations
func (s *Service) processUpdateBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		order := op.order

		// Update order in maps
		s.Orders[order.ID] = order

		// Update in cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// processCancelBatch processes a batch of cancel operations
func (s *Service) processCancelBatch(ops []orderOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, op := range ops {
		orderID := op.requestID
		order, exists := s.Orders[orderID]

		if !exists {
			op.resultCh <- orderOperationResult{order: nil, err: ErrOrderNotFound}
			continue
		}

		// Update order status
		order.Status = OrderStatusCancelled
		order.UpdatedAt = time.Now()

		// Update in cache
		s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)

		// Send result
		op.resultCh <- orderOperationResult{order: order, err: nil}
	}
}

// PlaceOrder places an order
func (s *Service) PlaceOrder(ctx context.Context, request *OrderRequest) (*Order, error) {
	// Validate request
	if err := s.validateOrderRequest(request); err != nil {
		return nil, err
	}

	// Check if client order ID already exists
	if request.ClientOrderID != "" {
		// Check cache first
		if _, found := s.OrderCache.Get("client:" + request.ClientOrderID); found {
			return nil, ErrDuplicateClientOrderID
		}

		s.mu.RLock()
		_, exists := s.ClientOrderIDs[request.ClientOrderID]
		s.mu.RUnlock()
		if exists {
			return nil, ErrDuplicateClientOrderID
		}
	}

	// Create order
	order := &Order{
		ID:             uuid.New().String(),
		UserID:         request.UserID,
		ClientOrderID:  request.ClientOrderID,
		Symbol:         request.Symbol,
		Side:           request.Side,
		Type:           request.Type,
		Price:          request.Price,
		StopPrice:      request.StopPrice,
		Quantity:       request.Quantity,
		FilledQuantity: 0,
		Status:         OrderStatusNew,
		TimeInForce:    request.TimeInForce,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
		ExpiresAt:      request.ExpiresAt,
		Trades:         make([]*Trade, 0),
		Metadata:       request.Metadata,
	}

	// Set expiry time for day orders
	if order.TimeInForce == TimeInForceDAY && (order.ExpiresAt == nil || order.ExpiresAt.IsZero()) {
		// Set expiry time to end of day
		now := time.Now()
		endOfDay := time.Date(now.Year(), now.Month(), now.Day(), 23, 59, 59, 999999999, now.Location())
		order.ExpiresAt = &endOfDay
	}

	// Use batch processing for better performance
	resultCh := make(chan orderOperationResult, 1)
	s.orderBatchChan <- orderOperation{
		opType:    "add",
		order:     order,
		requestID: order.ID,
		resultCh:  resultCh,
	}

	// Wait for result
	result := <-resultCh
	if result.err != nil {
		return nil, result.err
	}

	// Add to cache
	s.OrderCache.Set(order.ID, order, cache.DefaultExpiration)
	if order.ClientOrderID != "" {
		s.OrderCache.Set("client:"+order.ClientOrderID, order.ID, cache.DefaultExpiration)
	}

	// Place order in matching engine
	engineOrder := &order_matching.Order{
		ID:             order.ID,
		Symbol:         order.Symbol,
		Side:           order_matching.OrderSide(order.Side),
		Type:           order_matching.OrderType(order.Type),
		Price:          order.Price,
		Quantity:       order.Quantity,
		FilledQuantity: 0,
		Status:         order_matching.OrderStatus(order.Status),
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		ClientOrderID:  order.ClientOrderID,
		UserID:         order.UserID,
		StopPrice:      order.StopPrice,
		TimeInForce:    string(order.TimeInForce),
	}

	// Handle immediate-or-cancel and fill-or-kill orders
	if order.TimeInForce == TimeInForceIOC || order.TimeInForce == TimeInForceFOK {
		// Place order in matching engine
		trades, err := s.Engine.PlaceOrder(engineOrder)
		if err != nil {
			// Remove order from maps
			s.mu.Lock()
			delete(s.Orders, order.ID)
			delete(s.UserOrders[order.UserID], order.ID)
			delete(s.SymbolOrders[order.Symbol], order.ID)
			if order.ClientOrderID != "" {
				delete(s.ClientOrderIDs, order.ClientOrderID)
			}
			s.mu.Unlock()
			return nil, err
		}

		// Update order with trades
		s.mu.Lock()
		order.FilledQuantity = engineOrder.FilledQuantity
		order.Status = OrderStatus(engineOrder.Status)
		order.UpdatedAt = time.Now()

		// Add trades to order
		for _, trade := range trades {
			orderTrade := &Trade{
				ID:          trade.ID,
				OrderID:     order.ID,
				Symbol:      trade.Symbol,
				Side:        OrderSide(trade.TakerSide),
				Price:       trade.Price,
				Quantity:    trade.Quantity,
				ExecutedAt:  trade.Timestamp,
				Fee:         trade.TakerFee,
				FeeCurrency: order.Symbol,
				CounterPartyOrderID: func() string {
					if trade.MakerSide == order_matching.OrderSide(order.Side) {
						return trade.BuyOrderID
					}
					return trade.SellOrderID
				}(),
				Metadata: make(map[string]interface{}),
			}
			order.Trades = append(order.Trades, orderTrade)
		}

		// Cancel unfilled quantity for IOC orders
		if order.TimeInForce == TimeInForceIOC && order.FilledQuantity < order.Quantity {
			order.Status = OrderStatusCancelled
		}

		// Cancel order for FOK orders if not fully filled
		if order.TimeInForce == TimeInForceFOK && order.FilledQuantity < order.Quantity {
			order.Status = OrderStatusCancelled
			// Cancel order in matching engine
			s.Engine.CancelOrder(order.Symbol, order.ID)
		}
		s.mu.Unlock()

		return order, nil
	}

	// Place order in matching engine
	trades, err := s.Engine.PlaceOrder(engineOrder)
	if err != nil {
		// Remove order from maps
		s.mu.Lock()
		delete(s.Orders, order.ID)
		delete(s.UserOrders[order.UserID], order.ID)
		delete(s.SymbolOrders[order.Symbol], order.ID)
		if order.ClientOrderID != "" {
			delete(s.ClientOrderIDs, order.ClientOrderID)
		}
		s.mu.Unlock()
		return nil, err
	}

	// Update order with trades
	s.mu.Lock()
	order.FilledQuantity = engineOrder.FilledQuantity
	order.Status = OrderStatus(engineOrder.Status)
	order.UpdatedAt = time.Now()

	// Add trades to order
	for _, trade := range trades {
		orderTrade := &Trade{
			ID:          trade.ID,
			OrderID:     order.ID,
			Symbol:      trade.Symbol,
			Side:        OrderSide(trade.TakerSide),
			Price:       trade.Price,
			Quantity:    trade.Quantity,
			ExecutedAt:  trade.Timestamp,
			Fee:         trade.TakerFee,
			FeeCurrency: order.Symbol,
			CounterPartyOrderID: func() string {
				if trade.MakerSide == order_matching.OrderSide(order.Side) {
					return trade.BuyOrderID
				}
				return trade.SellOrderID
			}(),
			Metadata: make(map[string]interface{}),
		}
		order.Trades = append(order.Trades, orderTrade)
	}
	s.mu.Unlock()

	return order, nil
}

// CancelOrder cancels an order
func (s *Service) CancelOrder(ctx context.Context, request *OrderCancelRequest) (*Order, error) {
	// Get order
	var orderID string
	if request.OrderID != "" {
		orderID = request.OrderID
	} else if request.ClientOrderID != "" {
		s.mu.RLock()
		var exists bool
		orderID, exists = s.ClientOrderIDs[request.ClientOrderID]
		s.mu.RUnlock()
		if !exists {
			return nil, ErrOrderNotFound
		}
	} else {
		return nil, ErrInvalidRequest
	}

	// Get order
	s.mu.RLock()
	order, exists := s.Orders[orderID]
	s.mu.RUnlock()
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Check if order belongs to user
	if order.UserID != request.UserID {
		return nil, ErrUnauthorized
	}

	// Check if order can be cancelled
	if order.Status != OrderStatusNew && order.Status != OrderStatusPartiallyFilled {
		return nil, ErrInvalidOrderStatus
	}

	// Cancel order in matching engine
	err := s.Engine.CancelOrder(order.Symbol, order.ID)
	if err != nil {
		return nil, err
	}

	// Update order status
	s.mu.Lock()
	order.Status = OrderStatusCancelled
	order.UpdatedAt = time.Now()
	s.mu.Unlock()

	return order, nil
}

// GetOrder gets an order
func (s *Service) GetOrder(ctx context.Context, orderID string) (*Order, error) {
	// Check cache first
	if cachedOrder, found := s.OrderCache.Get(orderID); found {
		return cachedOrder.(*Order), nil
	}

	// If not in cache, check the map
	s.mu.RLock()
	order, exists := s.Orders[orderID]
	s.mu.RUnlock()

	if !exists {
		return nil, ErrOrderNotFound
	}

	// Add to cache for future requests
	s.OrderCache.Set(orderID, order, cache.DefaultExpiration)

	return order, nil
}

// GetOrderByClientOrderID gets an order by client order ID
func (s *Service) GetOrderByClientOrderID(ctx context.Context, clientOrderID string) (*Order, error) {
	// Check cache first
	if cachedOrderID, found := s.OrderCache.Get("client:" + clientOrderID); found {
		orderID := cachedOrderID.(string)
		if cachedOrder, found := s.OrderCache.Get(orderID); found {
			return cachedOrder.(*Order), nil
		}
	}

	// If not in cache, check the maps
	s.mu.RLock()
	orderID, exists := s.ClientOrderIDs[clientOrderID]
	if !exists {
		s.mu.RUnlock()
		return nil, ErrOrderNotFound
	}

	order, exists := s.Orders[orderID]
	s.mu.RUnlock()
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Add to cache for future requests
	s.OrderCache.Set(orderID, order, cache.DefaultExpiration)
	s.OrderCache.Set("client:"+clientOrderID, orderID, cache.DefaultExpiration)

	return order, nil
}

// GetOrders gets orders
func (s *Service) GetOrders(ctx context.Context, filter *OrderFilter) ([]*Order, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Get order IDs
	orderIDs := make(map[string]bool)

	// Filter by user ID
	if filter.UserID != "" {
		userOrders, exists := s.UserOrders[filter.UserID]
		if !exists {
			return []*Order{}, nil
		}

		// Add user orders to order IDs
		for orderID := range userOrders {
			orderIDs[orderID] = true
		}
	}

	// Filter by symbol
	if filter.Symbol != "" {
		symbolOrders, exists := s.SymbolOrders[filter.Symbol]
		if !exists {
			return []*Order{}, nil
		}

		// If user ID filter is applied, intersect with symbol orders
		if filter.UserID != "" {
			for orderID := range orderIDs {
				if !symbolOrders[orderID] {
					delete(orderIDs, orderID)
				}
			}
		} else {
			// Add symbol orders to order IDs
			for orderID := range symbolOrders {
				orderIDs[orderID] = true
			}
		}
	}

	// If no filters applied, get all orders
	if filter.UserID == "" && filter.Symbol == "" {
		for orderID := range s.Orders {
			orderIDs[orderID] = true
		}
	}

	// Get orders
	orders := make([]*Order, 0, len(orderIDs))
	for orderID := range orderIDs {
		order := s.Orders[orderID]

		// Filter by side
		if filter.Side != nil && order.Side != *filter.Side {
			continue
		}

		// Filter by type
		if filter.Type != nil && order.Type != *filter.Type {
			continue
		}

		// Filter by status
		if filter.Status != nil && order.Status != *filter.Status {
			continue
		}

		// Filter by start time
		if filter.StartTime != nil && !filter.StartTime.IsZero() && order.CreatedAt.Before(*filter.StartTime) {
			continue
		}

		// Filter by end time
		if filter.EndTime != nil && !filter.EndTime.IsZero() && order.CreatedAt.After(*filter.EndTime) {
			continue
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// UpdateOrder updates an order
func (s *Service) UpdateOrder(ctx context.Context, request *OrderUpdateRequest) (*Order, error) {
	// Get order
	var orderID string
	if request.OrderID != "" {
		orderID = request.OrderID
	} else if request.ClientOrderID != "" {
		s.mu.RLock()
		var exists bool
		orderID, exists = s.ClientOrderIDs[request.ClientOrderID]
		s.mu.RUnlock()
		if !exists {
			return nil, ErrOrderNotFound
		}
	} else {
		return nil, ErrInvalidRequest
	}

	// Get order
	s.mu.RLock()
	order, exists := s.Orders[orderID]
	s.mu.RUnlock()
	if !exists {
		return nil, ErrOrderNotFound
	}

	// Check if order belongs to user
	if order.UserID != request.UserID {
		return nil, ErrUnauthorized
	}

	// Check if order can be updated
	if order.Status != OrderStatusNew {
		return nil, ErrInvalidOrderStatus
	}

	// Cancel existing order
	err := s.Engine.CancelOrder(order.Symbol, order.ID)
	if err != nil {
		return nil, err
	}

	// Update order
	s.mu.Lock()

	// Update price if provided
	if request.Price > 0 {
		order.Price = request.Price
	}

	// Update stop price if provided
	if request.StopPrice > 0 {
		order.StopPrice = request.StopPrice
	}

	// Update quantity if provided
	if request.Quantity > 0 {
		order.Quantity = request.Quantity
	}

	// Update time in force if provided
	if request.TimeInForce != "" {
		order.TimeInForce = request.TimeInForce
	}

	// Update expires at if provided
	if !request.ExpiresAt.IsZero() {
		order.ExpiresAt = request.ExpiresAt
	}

	// Update status and timestamp
	order.Status = OrderStatusNew
	order.UpdatedAt = time.Now()
	s.mu.Unlock()

	// Place updated order in matching engine
	engineOrder := &order_matching.Order{
		ID:             order.ID,
		Symbol:         order.Symbol,
		Side:           order_matching.OrderSide(order.Side),
		Type:           order_matching.OrderType(order.Type),
		Price:          order.Price,
		Quantity:       order.Quantity,
		FilledQuantity: order.FilledQuantity,
		Status:         order_matching.OrderStatus(order.Status),
		CreatedAt:      order.CreatedAt,
		UpdatedAt:      order.UpdatedAt,
		ClientOrderID:  order.ClientOrderID,
		UserID:         order.UserID,
		StopPrice:      order.StopPrice,
		TimeInForce:    string(order.TimeInForce),
	}

	// Place order in matching engine
	trades, err := s.Engine.PlaceOrder(engineOrder)
	if err != nil {
		return nil, err
	}

	// Update order with trades
	s.mu.Lock()
	order.FilledQuantity = engineOrder.FilledQuantity
	order.Status = OrderStatus(engineOrder.Status)
	order.UpdatedAt = time.Now()

	// Add trades to order
	for _, trade := range trades {
		orderTrade := &Trade{
			ID:          trade.ID,
			OrderID:     order.ID,
			Symbol:      trade.Symbol,
			Side:        OrderSide(trade.TakerSide),
			Price:       trade.Price,
			Quantity:    trade.Quantity,
			ExecutedAt:  trade.Timestamp,
			Fee:         trade.TakerFee,
			FeeCurrency: order.Symbol,
			CounterPartyOrderID: func() string {
				if trade.MakerSide == order_matching.OrderSide(order.Side) {
					return trade.BuyOrderID
				}
				return trade.SellOrderID
			}(),
			Metadata: make(map[string]interface{}),
		}
		order.Trades = append(order.Trades, orderTrade)
	}
	s.mu.Unlock()

	return order, nil
}

// validateOrderRequest validates an order request
func (s *Service) validateOrderRequest(request *OrderRequest) error {
	// Check required fields
	if request.UserID == "" {
		return ErrInvalidRequest
	}
	if request.Symbol == "" {
		return ErrInvalidRequest
	}
	if request.Side != OrderSideBuy && request.Side != OrderSideSell {
		return ErrInvalidRequest
	}
	if request.Type != OrderTypeLimit && request.Type != OrderTypeMarket && request.Type != OrderTypeStopLimit && request.Type != OrderTypeStopMarket {
		return ErrInvalidRequest
	}
	if request.Quantity <= 0 {
		return ErrInvalidRequest
	}

	// Check price for limit orders
	if (request.Type == OrderTypeLimit || request.Type == OrderTypeStopLimit) && request.Price <= 0 {
		return ErrInvalidRequest
	}

	// Check stop price for stop orders
	if (request.Type == OrderTypeStopLimit || request.Type == OrderTypeStopMarket) && request.StopPrice <= 0 {
		return ErrInvalidRequest
	}

	// Check time in force
	if request.TimeInForce == "" {
		request.TimeInForce = TimeInForceGTC
	} else if request.TimeInForce != TimeInForceGTC && request.TimeInForce != TimeInForceIOC && request.TimeInForce != TimeInForceFOK && request.TimeInForce != TimeInForceDAY {
		return ErrInvalidRequest
	}

	return nil
}

// checkOrderExpiry checks for expired orders
func (s *Service) checkOrderExpiry() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			now := time.Now()

			// Get expired orders
			s.mu.RLock()
			expiredOrders := make([]*Order, 0)
			for _, order := range s.Orders {
				if order.ExpiresAt != nil && !order.ExpiresAt.IsZero() && now.After(*order.ExpiresAt) && (order.Status == OrderStatusNew || order.Status == OrderStatusPartiallyFilled) {
					expiredOrders = append(expiredOrders, order)
				}
			}
			s.mu.RUnlock()

			// Process in batches for better performance
			if len(expiredOrders) > 0 {
				batchSize := 100
				for i := 0; i < len(expiredOrders); i += batchSize {
					end := i + batchSize
					if end > len(expiredOrders) {
						end = len(expiredOrders)
					}

					batch := expiredOrders[i:end]
					s.processExpiredOrdersBatch(batch, now)
				}
			}
		}
	}
}

// processExpiredOrdersBatch processes a batch of expired orders
func (s *Service) processExpiredOrdersBatch(orders []*Order, now time.Time) {
	// Create a wait group for concurrent processing
	var wg sync.WaitGroup
	wg.Add(len(orders))

	// Process orders concurrently
	for _, order := range orders {
		go func(order *Order) {
			defer wg.Done()

			// Cancel order in matching engine
			err := s.Engine.CancelOrder(order.Symbol, order.ID)
			if err != nil {
				s.logger.Error("Failed to cancel expired order",
					zap.String("order_id", order.ID),
					zap.Error(err))
				return
			}

			// Update order status using batch operation
			resultCh := make(chan orderOperationResult, 1)
			order.Status = OrderStatusExpired
			order.UpdatedAt = now

			s.orderBatchChan <- orderOperation{
				opType:    "update",
				order:     order,
				requestID: order.ID,
				resultCh:  resultCh,
			}

			// Wait for result
			<-resultCh

			s.logger.Info("Order expired",
				zap.String("order_id", order.ID),
				zap.String("symbol", order.Symbol),
				zap.String("user_id", order.UserID))
		}(order)
	}

	// Wait for all orders to be processed
	wg.Wait()
}

// Stop stops the service
func (s *Service) Stop() {
	s.cancel()
}

// Note: Error definitions moved to errors.go to avoid duplication
