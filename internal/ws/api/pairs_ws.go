package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"
	
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// PairsWebSocketHandler handles WebSocket connections for pairs trading
type PairsWebSocketHandler struct {
	logger       *zap.Logger
	pairRepo     *repositories.PairRepository
	statsRepo    *repositories.PairStatisticsRepository
	positionRepo *repositories.PairPositionRepository
	clients      map[*websocket.Conn]map[string]bool // Map of clients to subscribed pair IDs
	clientsMu    sync.RWMutex
}

// NewPairsWebSocketHandler creates a new pairs WebSocket handler
func NewPairsWebSocketHandler(
	logger *zap.Logger,
	pairRepo *repositories.PairRepository,
	statsRepo *repositories.PairStatisticsRepository,
	positionRepo *repositories.PairPositionRepository,
) *PairsWebSocketHandler {
	return &PairsWebSocketHandler{
		logger:       logger,
		pairRepo:     pairRepo,
		statsRepo:    statsRepo,
		positionRepo: positionRepo,
		clients:      make(map[*websocket.Conn]map[string]bool),
	}
}

// HandleConnection handles a WebSocket connection
func (h *PairsWebSocketHandler) HandleConnection(conn *websocket.Conn) {
	// Register client
	h.clientsMu.Lock()
	h.clients[conn] = make(map[string]bool)
	h.clientsMu.Unlock()
	
	// Clean up on disconnect
	defer func() {
		h.clientsMu.Lock()
		delete(h.clients, conn)
		h.clientsMu.Unlock()
		conn.Close()
	}()
	
	// Start a goroutine to handle messages from the client
	go h.readPump(conn)
	
	// Start a goroutine to send periodic updates
	go h.writePump(conn)
}

// readPump reads messages from the client
func (h *PairsWebSocketHandler) readPump(conn *websocket.Conn) {
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				h.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}
		
		// Parse the message
		var msg struct {
			Action string   `json:"action"`
			PairIDs []string `json:"pair_ids"`
		}
		
		if err := json.Unmarshal(message, &msg); err != nil {
			h.logger.Error("Failed to parse WebSocket message", zap.Error(err))
			continue
		}
		
		// Handle the message
		switch msg.Action {
		case "subscribe":
			h.clientsMu.Lock()
			for _, pairID := range msg.PairIDs {
				h.clients[conn][pairID] = true
			}
			h.clientsMu.Unlock()
			
		case "unsubscribe":
			h.clientsMu.Lock()
			for _, pairID := range msg.PairIDs {
				delete(h.clients[conn], pairID)
			}
			h.clientsMu.Unlock()
			
		default:
			h.logger.Warn("Unknown WebSocket action", zap.String("action", msg.Action))
		}
	}
}

// writePump sends updates to the client
func (h *PairsWebSocketHandler) writePump(conn *websocket.Conn) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			// Get subscribed pair IDs
			h.clientsMu.RLock()
			pairIDs := make([]string, 0, len(h.clients[conn]))
			for pairID := range h.clients[conn] {
				pairIDs = append(pairIDs, pairID)
			}
			h.clientsMu.RUnlock()
			
			// If no subscriptions, continue
			if len(pairIDs) == 0 {
				continue
			}
			
			// Get updates for each pair
			for _, pairID := range pairIDs {
				// Get latest statistics
				stats, err := h.statsRepo.GetLatestStatistics(context.Background(), pairID)
				if err != nil {
					continue
				}
				
				// Get open positions
				positions, err := h.positionRepo.GetOpenPositions(context.Background(), pairID)
				if err != nil {
					continue
				}
				
				// Create update message
				update := struct {
					Type      string                `json:"type"`
					PairID    string                `json:"pair_id"`
					Stats     *models.PairStatistics `json:"stats"`
					Positions []*models.PairPosition `json:"positions"`
				}{
					Type:      "pair_update",
					PairID:    pairID,
					Stats:     stats,
					Positions: positions,
				}
				
				// Send update
				if err := conn.WriteJSON(update); err != nil {
					h.logger.Error("Failed to send WebSocket update",
						zap.Error(err),
						zap.String("pair_id", pairID))
					return
				}
			}
		}
	}
}

// BroadcastPairUpdate broadcasts an update for a specific pair to all subscribed clients
func (h *PairsWebSocketHandler) BroadcastPairUpdate(pairID string, stats *models.PairStatistics) {
	h.clientsMu.RLock()
	defer h.clientsMu.RUnlock()
	
	for conn, subscriptions := range h.clients {
		if subscriptions[pairID] {
			update := struct {
				Type   string                `json:"type"`
				PairID string                `json:"pair_id"`
				Stats  *models.PairStatistics `json:"stats"`
			}{
				Type:   "pair_update",
				PairID: pairID,
				Stats:  stats,
			}
			
			if err := conn.WriteJSON(update); err != nil {
				h.logger.Error("Failed to broadcast WebSocket update",
					zap.Error(err),
					zap.String("pair_id", pairID))
			}
		}
	}
}
