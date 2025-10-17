package main

import (
	"context"
	"log"
	"net"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	pb "github.com/abdoElHodaky/tradSys/proto/risk"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a gRPC server
	s := grpc.NewServer()

	// Create and register the risk handler
	handler := risk.NewHandler(logger)
	pb.RegisterRiskServiceServer(s, &riskServiceServer{handler: handler})

	// Listen on port 50051
	lis, err := net.Listen("tcp", ":50051")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	logger.Info("Risk service starting on :50051")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// riskServiceServer wraps the handler to implement the gRPC interface
type riskServiceServer struct {
	pb.UnimplementedRiskServiceServer
	handler *risk.Handler
}

func (s *riskServiceServer) ValidateOrder(ctx context.Context, req *pb.ValidateOrderRequest) (*pb.ValidateOrderResponse, error) {
	return s.handler.ValidateOrder(ctx, req)
}

func (s *riskServiceServer) GetAccountRisk(ctx context.Context, req *pb.AccountRiskRequest) (*pb.AccountRiskResponse, error) {
	return s.handler.GetAccountRisk(ctx, req)
}

func (s *riskServiceServer) GetPositionRisk(ctx context.Context, req *pb.PositionRiskRequest) (*pb.PositionRiskResponse, error) {
	return s.handler.GetPositionRisk(ctx, req)
}

func (s *riskServiceServer) GetOrderRisk(ctx context.Context, req *pb.OrderRiskRequest) (*pb.OrderRiskResponse, error) {
	return s.handler.GetOrderRisk(ctx, req)
}

func (s *riskServiceServer) UpdateRiskLimits(ctx context.Context, req *pb.UpdateRiskLimitsRequest) (*pb.UpdateRiskLimitsResponse, error) {
	return s.handler.UpdateRiskLimits(ctx, req)
}
