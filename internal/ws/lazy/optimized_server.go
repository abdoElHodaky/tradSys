package lazy

import (
	"context"
	"net/http"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/coordination"
	"github.com/abdoElHodaky/tradSys/internal/architecture/fx/lazy"
	"github.com/abdoElHodaky/tradSys/internal/ws"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// LazyOptimizedWebSocketServer is a lazy-loaded wrapper for the optimized WebSocket server
type LazyOptimizedWebSocketServer struct {
	// Component coordinator
	coordinator *coordination.ComponentCoordinator
	
	// Component name
	componentName string
	
	// Configuration
	config ws.WebSocketConfig
	
	// Logger
	logger *zap.Logger
}

// NewLazyOptimizedWebSocketServer creates a new lazy-loaded WebSocket server
func NewLazyOptimizedWebSocketServer(
	coordinator *coordination.ComponentCoordinator,
	config ws.WebSocketConfig,
	logger *zap.Logger,
) (*LazyOptimizedWebSocketServer, error) {
	componentName := "websocket-server"
	
	// Create the provider function
	providerFn := func(log *zap.Logger) (interface{}, error) {
		return ws.NewOptimizedWebSocketServer(config, log)
	}
	
	// Create the lazy provider
	provider := lazy.NewEnhancedLazyProvider(
		componentName,
		providerFn,
		logger,
		nil, // Metrics will be handled by the coordinator
		lazy.WithMemoryEstimate(200*1024*1024), // 200MB estimate
		lazy.WithTimeout(15*time.Second),
		lazy.WithPriority(10), // Very high priority
	)
	
	// Register with the coordinator
	err := coordinator.RegisterComponent(
		componentName,
		"websocket",
		provider,
		[]string{}, // No dependencies
	)
	
	if err != nil {
		return nil, err
	}
	
	return &LazyOptimizedWebSocketServer{
		coordinator:   coordinator,
		componentName: componentName,
		config:        config,
		logger:        logger,
	}, nil
}

// HandleConnection handles a WebSocket connection
func (s *LazyOptimizedWebSocketServer) HandleConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	
	// Get the underlying server
	serverInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		s.logger.Error("Failed to get WebSocket server",
			zap.Error(err),
		)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Cast to the actual server type
	server, ok := serverInterface.(*ws.OptimizedWebSocketServer)
	if !ok {
		s.logger.Error("Invalid WebSocket server type")
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	
	// Call the actual method
	server.HandleConnection(w, r)
}

// RegisterHandler registers a message handler
func (s *LazyOptimizedWebSocketServer) RegisterHandler(messageType string, handler ws.MessageHandler) error {
	// Create a background context
	ctx := context.Background()
	
	// Get the underlying server
	serverInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual server type
	server, ok := serverInterface.(*ws.OptimizedWebSocketServer)
	if !ok {
		return ws.ErrInvalidServerType
	}
	
	// Call the actual method
	return server.RegisterHandler(messageType, handler)
}

// BroadcastMessage broadcasts a message to all connections
func (s *LazyOptimizedWebSocketServer) BroadcastMessage(ctx context.Context, message []byte) error {
	// Get the underlying server
	serverInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual server type
	server, ok := serverInterface.(*ws.OptimizedWebSocketServer)
	if !ok {
		return ws.ErrInvalidServerType
	}
	
	// Call the actual method
	return server.BroadcastMessage(ctx, message)
}

// SendMessage sends a message to a specific connection
func (s *LazyOptimizedWebSocketServer) SendMessage(ctx context.Context, conn *websocket.Conn, message []byte) error {
	// Get the underlying server
	serverInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return err
	}
	
	// Cast to the actual server type
	server, ok := serverInterface.(*ws.OptimizedWebSocketServer)
	if !ok {
		return ws.ErrInvalidServerType
	}
	
	// Call the actual method
	return server.SendMessage(ctx, conn, message)
}

// GetConnectionCount gets the number of active connections
func (s *LazyOptimizedWebSocketServer) GetConnectionCount(ctx context.Context) (int, error) {
	// Get the underlying server
	serverInterface, err := s.coordinator.GetComponent(ctx, s.componentName)
	if err != nil {
		return 0, err
	}
	
	// Cast to the actual server type
	server, ok := serverInterface.(*ws.OptimizedWebSocketServer)
	if !ok {
		return 0, ws.ErrInvalidServerType
	}
	
	// Call the actual method
	return server.GetConnectionCount(ctx)
}

// Shutdown shuts down the server
func (s *LazyOptimizedWebSocketServer) Shutdown(ctx context.Context) error {
	return s.coordinator.ShutdownComponent(ctx, s.componentName)
}

