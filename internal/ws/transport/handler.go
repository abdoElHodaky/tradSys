package websocket

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketHandler handles WebSocket connections
type WebSocketHandler struct {
	hub      *Hub
	logger   *zap.Logger
	upgrader websocket.Upgrader
}

// WebSocketHandlerConfig contains configuration for the WebSocket handler
type WebSocketHandlerConfig struct {
	// ReadBufferSize is the size of the read buffer for the WebSocket connection
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer for the WebSocket connection
	WriteBufferSize int

	// CheckOrigin is a function that checks the origin of the WebSocket connection
	CheckOrigin func(r *http.Request) bool

	// PingInterval is the interval at which ping messages are sent
	PingInterval time.Duration

	// PongWait is the time to wait for a pong response
	PongWait time.Duration

	// WriteWait is the time to wait for a write to complete
	WriteWait time.Duration
}

// DefaultWebSocketHandlerConfig returns the default configuration
func DefaultWebSocketHandlerConfig() WebSocketHandlerConfig {
	return WebSocketHandlerConfig{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin:     func(r *http.Request) bool { return true },
		PingInterval:    30 * time.Second,
		PongWait:        60 * time.Second,
		WriteWait:       10 * time.Second,
	}
}

// NewWebSocketHandler creates a new WebSocket handler
func NewWebSocketHandler(hub *Hub, logger *zap.Logger, config WebSocketHandlerConfig) *WebSocketHandler {
	return &WebSocketHandler{
		hub:    hub,
		logger: logger,
		upgrader: websocket.Upgrader{
			ReadBufferSize:  config.ReadBufferSize,
			WriteBufferSize: config.WriteBufferSize,
			CheckOrigin:     config.CheckOrigin,
		},
	}
}

// HandleConnection handles a WebSocket connection
func (h *WebSocketHandler) HandleConnection(c *gin.Context) {
	// Upgrade the HTTP connection to a WebSocket connection
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		h.logger.Error("Failed to upgrade connection", zap.Error(err))
		return
	}

	// Get the client ID from the query parameters
	clientID := c.Query("client_id")
	if clientID == "" {
		clientID = c.GetString("user_id")
	}

	// Create a new client
	client := NewClient(clientID, conn, h.hub, h.logger)

	// Register the client with the hub
	h.hub.Register(client)

	// Start the client's read and write pumps
	go client.ReadPump()
	go client.WritePump()
}

// RegisterRoutes registers the WebSocket routes with the Gin router
func (h *WebSocketHandler) RegisterRoutes(router *gin.Engine) {
	// Register the WebSocket route
	router.GET("/ws", h.HandleConnection)

	// Register the market data WebSocket route
	router.GET("/ws/marketdata", h.HandleConnection)

	// Register the orders WebSocket route
	router.GET("/ws/orders", h.HandleConnection)
}
