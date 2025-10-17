package main

import (
	"context"
	"log"
	"net"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	pb "github.com/abdoElHodaky/tradSys/proto/orders"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a gRPC server
	s := grpc.NewServer()

	// Create and register the orders handler
	handler := orders.NewHandler(logger)
	pb.RegisterOrderServiceServer(s, &orderServiceServer{handler: handler})

	// Listen on port 50052
	lis, err := net.Listen("tcp", ":50052")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	logger.Info("Orders service starting on :50052")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// orderServiceServer wraps the handler to implement the gRPC interface
type orderServiceServer struct {
	pb.UnimplementedOrderServiceServer
	handler *orders.Handler
}

func (s *orderServiceServer) CreateOrder(ctx context.Context, req *pb.CreateOrderRequest) (*pb.OrderResponse, error) {
	return s.handler.CreateOrder(ctx, req)
}

func (s *orderServiceServer) GetOrder(ctx context.Context, req *pb.GetOrderRequest) (*pb.OrderResponse, error) {
	return s.handler.GetOrder(ctx, req)
}

func (s *orderServiceServer) CancelOrder(ctx context.Context, req *pb.CancelOrderRequest) (*pb.OrderResponse, error) {
	return s.handler.CancelOrder(ctx, req)
}

func (s *orderServiceServer) GetOrders(ctx context.Context, req *pb.GetOrdersRequest) (*pb.GetOrdersResponse, error) {
	return s.handler.GetOrders(ctx, req)
}

func (s *orderServiceServer) StreamOrders(req *pb.StreamOrdersRequest, stream pb.OrderService_StreamOrdersServer) error {
	return s.handler.StreamOrders(req, stream)
}
