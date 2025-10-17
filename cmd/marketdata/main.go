package main

import (
	"context"
	"log"
	"net"

	"github.com/abdoElHodaky/tradSys/internal/marketdata"
	pb "github.com/abdoElHodaky/tradSys/proto/marketdata"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Create a gRPC server
	s := grpc.NewServer()

	// Create and register the market data handler
	handler := marketdata.NewHandler(logger)
	pb.RegisterMarketDataServiceServer(s, &marketDataServiceServer{handler: handler})

	// Listen on port 50053
	lis, err := net.Listen("tcp", ":50053")
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	logger.Info("Market Data service starting on :50053")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Failed to serve: %v", err)
	}
}

// marketDataServiceServer wraps the handler to implement the gRPC interface
type marketDataServiceServer struct {
	pb.UnimplementedMarketDataServiceServer
	handler *marketdata.Handler
}

func (s *marketDataServiceServer) GetMarketData(ctx context.Context, req *pb.MarketDataRequest) (*pb.MarketDataResponse, error) {
	return s.handler.GetMarketData(ctx, req)
}

func (s *marketDataServiceServer) StreamMarketData(req *pb.MarketDataRequest, stream pb.MarketDataService_StreamMarketDataServer) error {
	return s.handler.StreamMarketData(req, stream)
}

func (s *marketDataServiceServer) GetHistoricalData(ctx context.Context, req *pb.HistoricalDataRequest) (*pb.HistoricalDataResponse, error) {
	return s.handler.GetHistoricalData(ctx, req)
}

func (s *marketDataServiceServer) GetSymbols(ctx context.Context, req *pb.SymbolsRequest) (*pb.SymbolsResponse, error) {
	return s.handler.GetSymbols(ctx, req)
}
