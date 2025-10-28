package websocket

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub    *Hub
	logger *zap.Logger
	config *WebSocketHandlerConfig
}

// WebSocketHandlerConfig contains configuration for the WebSocket handler
type WebSocketHandlerConfig struct {
	Path string
}

// DefaultWebSocketHandlerConfig returns default configuration
func DefaultWebSocketHandlerConfig() *WebSocketHandlerConfig {
	return &WebSocketHandlerConfig{
		Path: "/ws",
	}
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, logger *zap.Logger, config *WebSocketHandlerConfig) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
		config: config,
	}
}

// RegisterRoutes registers WebSocket routes
func (h *WebSocketHandler) RegisterRoutes(router gin.IRouter) {
	router.GET(h.config.Path, h.handleWebSocket)
}

// handleWebSocket handles WebSocket upgrade requests
func (h *WebSocketHandler) handleWebSocket(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Generate client ID
	clientID := uuid.New().String()

	// Create new client
	client := NewClient(h.hub, conn, clientID, h.logger)

	// Register client with hub
	h.hub.RegisterClient(client)

	// Start client pumps
	client.Start()

	h.logger.Info("WebSocket connection established", zap.String("client_id", clientID))
}
