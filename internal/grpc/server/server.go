package server

import (
	"context"
	"net"
	"runtime"
	"time"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// Server provides a gRPC server
type Server struct {
	server   *grpc.Server
	listener net.Listener
	logger   *zap.Logger
	options  ServerOptions
}

// ServerOptions contains options for the server
type ServerOptions struct {
	MaxConnectionIdle     time.Duration
	MaxConnectionAge      time.Duration
	MaxConnectionAgeGrace time.Duration
	Time                  time.Duration
	Timeout               time.Duration
	MaxConcurrentStreams  uint32
	MaxRecvMsgSize        int
	MaxSendMsgSize        int
	NumServerWorkers      int
}

// DefaultServerOptions returns default server options
func DefaultServerOptions() ServerOptions {
	return ServerOptions{
		MaxConnectionIdle:     15 * time.Minute,
		MaxConnectionAge:      30 * time.Minute,
		MaxConnectionAgeGrace: 5 * time.Minute,
		Time:                  5 * time.Second,
		Timeout:               1 * time.Second,
		MaxConcurrentStreams:  1000,
		MaxRecvMsgSize:        50 * 1024 * 1024, // 50MB
		MaxSendMsgSize:        50 * 1024 * 1024, // 50MB
		NumServerWorkers:      runtime.NumCPU(),
	}
}

// NewServer creates a new server
func NewServer(logger *zap.Logger, options ServerOptions) *Server {
	// Create server options
	serverOptions := []grpc.ServerOption{
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             options.Time,
			PermitWithoutStream: true,
		}),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     options.MaxConnectionIdle,
			MaxConnectionAge:      options.MaxConnectionAge,
			MaxConnectionAgeGrace: options.MaxConnectionAgeGrace,
			Time:                  options.Time,
			Timeout:               options.Timeout,
		}),
		grpc.MaxConcurrentStreams(options.MaxConcurrentStreams),
		grpc.MaxRecvMsgSize(options.MaxRecvMsgSize),
		grpc.MaxSendMsgSize(options.MaxSendMsgSize),
		grpc.NumStreamWorkers(uint32(options.NumServerWorkers)),
	}

	// Create the server
	server := grpc.NewServer(serverOptions...)

	// Enable reflection
	reflection.Register(server)

	return &Server{
		server:  server,
		logger:  logger,
		options: options,
	}
}

// RegisterService registers a service with the server
func (s *Server) RegisterService(registerFunc func(server *grpc.Server)) {
	registerFunc(s.server)
}

// Start starts the server
func (s *Server) Start(ctx context.Context, address string) error {
	// Create a listener
	listener, err := net.Listen("tcp", address)
	if err != nil {
		return err
	}

	s.listener = listener

	// Log server start
	s.logger.Info("Starting gRPC server",
		zap.String("address", address),
		zap.Int("workers", s.options.NumServerWorkers))

	// Start the server
	return s.server.Serve(listener)
}

// Stop stops the server
func (s *Server) Stop() {
	s.logger.Info("Stopping gRPC server")
	s.server.GracefulStop()
}

// GetServer returns the underlying gRPC server
func (s *Server) GetServer() *grpc.Server {
	return s.server
}

// GetListener returns the underlying listener
func (s *Server) GetListener() net.Listener {
	return s.listener
}

