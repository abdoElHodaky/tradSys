package risk

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/core/matching"
	"github.com/abdoElHodaky/tradSys/internal/orders"
	riskengine "github.com/abdoElHodaky/tradSys/internal/risk/engine"
	"github.com/patrickmn/go-cache"
	"go.uber.org/zap"
)

// NewService creates a new risk management service
func NewService(orderEngine *order_matching.Engine, orderService *orders.Service, logger *zap.Logger) *Service {
	ctx, cancel := context.WithCancel(context.Background())

	service := &Service{
		OrderEngine:     orderEngine,
		OrderService:    orderService,
		Positions:       make(map[string]map[string]*riskengine.Position),
		RiskLimits:      make(map[string][]*RiskLimit),
		CircuitBreakers: make(map[string]*riskengine.CircuitBreaker),
		PositionCache:   cache.New(5*time.Minute, 10*time.Minute),
		RiskLimitCache:  cache.New(5*time.Minute, 10*time.Minute),
		logger:          logger,
		ctx:             ctx,
		cancel:          cancel,
		riskBatchChan:   make(chan RiskOperation, 1000),
		marketDataChan:  make(chan MarketDataUpdate, 1000),
	}

	// Start batch processor
	go service.processBatchOperations()

	// Start market data processor
	go service.processMarketData()

	// Start circuit breaker checker
	go service.checkCircuitBreakers()

	// Subscribe to trades from the order matching engine
	go service.subscribeToTrades()

	return service
}

// processBatchOperations processes batch operations for risk data
func (s *Service) processBatchOperations() {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	batch := make([]RiskOperation, 0, 100)

	for {
		select {
		case <-s.ctx.Done():
			return
		case op := <-s.riskBatchChan:
			batch = append(batch, op)

			// Process batch if it's full
			if len(batch) >= 100 {
				s.processBatch(batch)
				batch = make([]RiskOperation, 0, 100)
			}
		case <-ticker.C:
			// Process remaining operations in batch
			if len(batch) > 0 {
				s.processBatch(batch)
				batch = make([]RiskOperation, 0, 100)
			}
		}
	}
}

// processBatch processes a batch of risk operations
func (s *Service) processBatch(batch []RiskOperation) {
	// Group operations by type
	updatePositionOps := make([]RiskOperation, 0)
	checkLimitOps := make([]RiskOperation, 0)
	addLimitOps := make([]RiskOperation, 0)

	for _, op := range batch {
		switch op.OpType {
		case "update_position":
			updatePositionOps = append(updatePositionOps, op)
		case "check_limit":
			checkLimitOps = append(checkLimitOps, op)
		case "add_limit":
			addLimitOps = append(addLimitOps, op)
		}
	}

	// Process update position operations
	if len(updatePositionOps) > 0 {
		s.processUpdatePositionBatch(updatePositionOps)
	}

	// Process check limit operations
	if len(checkLimitOps) > 0 {
		s.processCheckLimitBatch(checkLimitOps)
	}

	// Process add limit operations
	if len(addLimitOps) > 0 {
		s.processAddLimitBatch(addLimitOps)
	}
}

// Stop stops the service
func (s *Service) Stop() {
	s.cancel()
}

