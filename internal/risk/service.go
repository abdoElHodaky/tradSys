package risk

import (
	"context"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	orderspb "github.com/abdoElHodaky/tradSys/proto/orders"
	pb "github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// Service implements the RiskService gRPC interface
type Service struct {
	pb.UnimplementedRiskServiceServer
	logger     *zap.Logger
	repository *repositories.RiskRepository
	orderRepo  *repositories.OrderRepository
}

// NewService creates a new risk service
func NewService(logger *zap.Logger, repository *repositories.RiskRepository) *Service {
	return &Service{
		logger:     logger,
		repository: repository,
	}
}

// SetOrderRepository sets the order repository
func (s *Service) SetOrderRepository(repo *repositories.OrderRepository) {
	s.orderRepo = repo
}

// CheckOrderRisk checks if an order passes risk checks
func (s *Service) CheckOrderRisk(ctx context.Context, req *pb.RiskCheckRequest) (*pb.RiskCheckResponse, error) {
	order := req.Order
	
	// Default account ID if not provided
	accountID := "default"
	if order.ClientId != "" {
		accountID = order.ClientId
	}
	
	// Get risk limits
	riskLimit, err := s.repository.GetRiskLimit(ctx, accountID, order.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk limits: %v", err)
	}
	
	// Check if risk limits are active
	if !riskLimit.Active {
		s.logger.Warn("Risk limits are disabled",
			zap.String("account_id", accountID),
			zap.String("symbol", order.Symbol))
		
		// Record risk check
		s.recordRiskCheck(ctx, order.OrderId, accountID, order.Symbol, false, "Risk limits are disabled")
		
		return &pb.RiskCheckResponse{
			Approved: false,
			Reason:   "Risk limits are disabled for this account",
		}, nil
	}
	
	// Check order size
	if order.Quantity > riskLimit.MaxOrderSize {
		s.logger.Warn("Order exceeds maximum order size",
			zap.String("order_id", order.OrderId),
			zap.Float64("quantity", order.Quantity),
			zap.Float64("max_order_size", riskLimit.MaxOrderSize))
		
		// Record risk check
		s.recordRiskCheck(ctx, order.OrderId, accountID, order.Symbol, false, "Order exceeds maximum order size")
		
		return &pb.RiskCheckResponse{
			Approved: false,
			Reason:   "Order exceeds maximum order size",
		}, nil
	}
	
	// Check position limits
	if s.orderRepo != nil {
		position, err := s.orderRepo.GetPosition(ctx, order.Symbol, accountID)
		if err != nil {
			s.logger.Error("Failed to get position",
				zap.Error(err),
				zap.String("symbol", order.Symbol),
				zap.String("account_id", accountID))
		} else {
			// Calculate new position after order
			newPosition := position.Quantity
			if order.Side == orderspb.OrderSide_BUY {
				newPosition += order.Quantity
			} else {
				newPosition -= order.Quantity
			}
			
			// Check if new position exceeds limits
			if newPosition > riskLimit.MaxPosition {
				s.logger.Warn("Order would exceed maximum position",
					zap.String("order_id", order.OrderId),
					zap.Float64("new_position", newPosition),
					zap.Float64("max_position", riskLimit.MaxPosition))
				
				// Record risk check
				s.recordRiskCheck(ctx, order.OrderId, accountID, order.Symbol, false, "Order would exceed maximum position")
				
				return &pb.RiskCheckResponse{
					Approved: false,
					Reason:   "Order would exceed maximum position",
				}, nil
			}
		}
	}
	
	// Check circuit breaker
	circuitBreaker, err := s.repository.GetCircuitBreaker(ctx, order.Symbol)
	if err != nil {
		s.logger.Error("Failed to get circuit breaker",
			zap.Error(err),
			zap.String("symbol", order.Symbol))
	} else if circuitBreaker.Triggered {
		// Check if circuit breaker has reset
		if circuitBreaker.ResetTime.After(time.Now()) {
			s.logger.Warn("Circuit breaker is triggered",
				zap.String("order_id", order.OrderId),
				zap.String("symbol", order.Symbol),
				zap.Time("reset_time", circuitBreaker.ResetTime))
			
			// Record risk check
			s.recordRiskCheck(ctx, order.OrderId, accountID, order.Symbol, false, "Circuit breaker is triggered")
			
			return &pb.RiskCheckResponse{
				Approved: false,
				Reason:   "Circuit breaker is triggered for this symbol",
			}, nil
		} else {
			// Reset circuit breaker
			circuitBreaker.Triggered = false
			if err := s.repository.UpdateCircuitBreaker(ctx, circuitBreaker); err != nil {
				s.logger.Error("Failed to reset circuit breaker",
					zap.Error(err),
					zap.String("symbol", order.Symbol))
			}
		}
	}
	
	// Record risk check
	s.recordRiskCheck(ctx, order.OrderId, accountID, order.Symbol, true, "Order passed all risk checks")
	
	return &pb.RiskCheckResponse{
		Approved: true,
		Reason:   "Order passed all risk checks",
	}, nil
}

// GetPosition gets the current position for a symbol
func (s *Service) GetPosition(ctx context.Context, req *pb.PositionRequest) (*pb.Position, error) {
	if s.orderRepo == nil {
		return nil, status.Errorf(codes.Internal, "order repository not set")
	}
	
	position, err := s.orderRepo.GetPosition(ctx, req.Symbol, req.AccountId)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get position: %v", err)
	}
	
	return &pb.Position{
		Symbol:        position.Symbol,
		AccountId:     position.AccountID,
		Quantity:      position.Quantity,
		AveragePrice:  position.AveragePrice,
		UnrealizedPnl: position.UnrealizedPnL,
		RealizedPnl:   position.RealizedPnL,
		Timestamp:     position.LastUpdated.UnixNano(),
	}, nil
}

// GetAllPositions gets all positions for an account
func (s *Service) GetAllPositions(ctx context.Context, req *pb.PositionRequest) (*pb.PositionList, error) {
	// This is a placeholder - in a real system, you would implement this
	// For now, we'll just return an empty list
	return &pb.PositionList{
		Positions: []*pb.Position{},
	}, nil
}

// GetRiskLimits gets risk limits for an account
func (s *Service) GetRiskLimits(ctx context.Context, req *pb.RiskLimitRequest) (*pb.RiskLimit, error) {
	riskLimit, err := s.repository.GetRiskLimit(ctx, req.AccountId, req.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get risk limits: %v", err)
	}
	
	return &pb.RiskLimit{
		AccountId:      riskLimit.AccountID,
		Symbol:         riskLimit.Symbol,
		MaxPosition:    riskLimit.MaxPosition,
		MaxOrderSize:   riskLimit.MaxOrderSize,
		MaxDailyLoss:   riskLimit.MaxDailyLoss,
		CurrentDailyLoss: riskLimit.CurrentDailyLoss,
		Active:         riskLimit.Active,
	}, nil
}

// GetCircuitBreakerStatus gets the status of a circuit breaker
func (s *Service) GetCircuitBreakerStatus(ctx context.Context, req *pb.CircuitBreakerRequest) (*pb.CircuitBreakerStatus, error) {
	circuitBreaker, err := s.repository.GetCircuitBreaker(ctx, req.Symbol)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to get circuit breaker: %v", err)
	}
	
	return &pb.CircuitBreakerStatus{
		Symbol:      circuitBreaker.Symbol,
		Triggered:   circuitBreaker.Triggered,
		Reason:      circuitBreaker.Reason,
		TriggerTime: circuitBreaker.TriggerTime.UnixNano(),
		ResetTime:   circuitBreaker.ResetTime.UnixNano(),
	}, nil
}

// recordRiskCheck records a risk check
func (s *Service) recordRiskCheck(ctx context.Context, orderID, accountID, symbol string, approved bool, reason string) {
	riskCheck := &models.RiskCheck{
		OrderID:   orderID,
		AccountID: accountID,
		Symbol:    symbol,
		Approved:  approved,
		Reason:    reason,
		CheckTime: time.Now(),
	}
	
	if err := s.repository.CreateRiskCheck(ctx, riskCheck); err != nil {
		s.logger.Error("Failed to record risk check",
			zap.Error(err),
			zap.String("order_id", orderID))
	}
}

