package handlers

import (
	"net/http"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/peerjs"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// PeerJSHandler handles PeerJS server requests
type PeerJSHandler struct {
	logger *zap.Logger
	server *peerjs.PeerServer
}

// NewPeerJSHandler creates a new PeerJS handler
func NewPeerJSHandler(logger *zap.Logger) *PeerJSHandler {
	server := peerjs.NewPeerServer(logger)

	// Start cleanup task
	server.StartCleanupTask(5*time.Minute, 10*time.Minute)

	return &PeerJSHandler{
		logger: logger,
		server: server,
	}
}

// HandleConnection handles a new WebSocket connection
func (h *PeerJSHandler) HandleConnection(c *gin.Context) {
	h.server.HandleConnection(c.Writer, c.Request)
}

// GetStats returns server statistics
func (h *PeerJSHandler) GetStats(c *gin.Context) {
	stats := map[string]interface{}{
		"peer_count": h.server.GetPeerCount(),
	}

	c.JSON(http.StatusOK, stats)
}

// RegisterRoutes registers the PeerJS routes
func (h *PeerJSHandler) RegisterRoutes(router *gin.Engine) {
	peerGroup := router.Group("/peerjs")
	{
		// WebSocket endpoint
		peerGroup.GET("/ws", h.HandleConnection)

		// Stats endpoint
		peerGroup.GET("/stats", h.GetStats)
	}
}
