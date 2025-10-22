package ws

import (
	"time"

	"github.com/abdoElHodaky/tradSys/proto/ws"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
)

// BinaryMessageHandler handles binary WebSocket messages using Protocol Buffers
type BinaryMessageHandler struct {
	logger *zap.Logger
	server *EnhancedServer
}

// NewBinaryMessageHandler creates a new binary message handler
func NewBinaryMessageHandler(logger *zap.Logger, server *EnhancedServer) *BinaryMessageHandler {
	return &BinaryMessageHandler{
		logger: logger,
		server: server,
	}
}

// HandleIncomingMessage handles an incoming binary message
func (h *BinaryMessageHandler) HandleIncomingMessage(conn *Connection, data []byte) error {
	// Parse the message
	var message ws.WebSocketMessage
	if err := proto.Unmarshal(data, &message); err != nil {
		h.logger.Error("Failed to unmarshal binary message",
			zap.Error(err),
			zap.String("remote_addr", conn.conn.RemoteAddr().String()))
		return err
	}
	
	// Update stats
	h.server.pool.IncrementMessagesReceived(1)
	
	// Handle the message based on its type
	switch message.Type {
	case "subscribe":
		return h.handleSubscription(conn, &message)
	case "unsubscribe":
		return h.handleUnsubscription(conn, &message)
	case "ping":
		return h.handlePing(conn, &message)
	default:
		// Forward the message to the appropriate handler based on the channel
		return h.handleChannelMessage(conn, &message)
	}
}

// handleSubscription handles a subscription message
func (h *BinaryMessageHandler) handleSubscription(conn *Connection, message *ws.WebSocketMessage) error {
	// Extract subscription details
	if message.GetSubscription() == nil {
		h.logger.Error("Invalid subscription message",
			zap.String("remote_addr", conn.conn.RemoteAddr().String()))
		return sendErrorMessage(conn, "Invalid subscription message", 400)
	}
	
	subscription := message.GetSubscription()
	channel := subscription.Channel
	symbol := subscription.Symbol
	
	// Subscribe to the channel
	conn.mu.Lock()
	conn.channels[channel] = true
	conn.mu.Unlock()
	
	// Update the connection pool
	h.server.pool.SubscribeToChannel(conn, channel)
	
	// If symbol is provided, update it
	if symbol != "" {
		h.server.pool.SetSymbol(conn, symbol)
	}
	
	// Send confirmation
	response := &ws.WebSocketMessage{
		Type:      "subscribed",
		Channel:   channel,
		Symbol:    symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Subscription{
			Subscription: &ws.SubscriptionPayload{
				Action:  "subscribe",
				Channel: channel,
				Symbol:  symbol,
				Success: true,
				Message: "Subscribed successfully",
			},
		},
	}
	
	return sendBinaryMessage(conn, response)
}

// handleUnsubscription handles an unsubscription message
func (h *BinaryMessageHandler) handleUnsubscription(conn *Connection, message *ws.WebSocketMessage) error {
	// Extract subscription details
	if message.GetSubscription() == nil {
		h.logger.Error("Invalid unsubscription message",
			zap.String("remote_addr", conn.conn.RemoteAddr().String()))
		return sendErrorMessage(conn, "Invalid unsubscription message", 400)
	}
	
	subscription := message.GetSubscription()
	channel := subscription.Channel
	
	// Unsubscribe from the channel
	conn.mu.Lock()
	delete(conn.channels, channel)
	conn.mu.Unlock()
	
	// Update the connection pool
	h.server.pool.UnsubscribeFromChannel(conn, channel)
	
	// Send confirmation
	response := &ws.WebSocketMessage{
		Type:      "unsubscribed",
		Channel:   channel,
		Symbol:    message.Symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Subscription{
			Subscription: &ws.SubscriptionPayload{
				Action:  "unsubscribe",
				Channel: channel,
				Symbol:  message.Symbol,
				Success: true,
				Message: "Unsubscribed successfully",
			},
		},
	}
	
	return sendBinaryMessage(conn, response)
}

// handlePing handles a ping message
func (h *BinaryMessageHandler) handlePing(conn *Connection, message *ws.WebSocketMessage) error {
	// Send pong response
	response := &ws.WebSocketMessage{
		Type:      "pong",
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Heartbeat{
			Heartbeat: &ws.HeartbeatPayload{
				Timestamp: time.Now().UnixMilli(),
			},
		},
	}
	
	return sendBinaryMessage(conn, response)
}

// handleChannelMessage handles a message for a specific channel
func (h *BinaryMessageHandler) handleChannelMessage(conn *Connection, message *ws.WebSocketMessage) error {
	// For now, just log the message
	h.logger.Debug("Received channel message",
		zap.String("type", message.Type),
		zap.String("channel", message.Channel),
		zap.String("symbol", message.Symbol),
		zap.String("remote_addr", conn.conn.RemoteAddr().String()))
	
	// In a real implementation, this would forward the message to the appropriate service
	return nil
}

// sendErrorMessage sends an error message to the client
func sendErrorMessage(conn *Connection, message string, code int32) error {
	errorMsg := &ws.WebSocketMessage{
		Type:      "error",
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Error{
			Error: &ws.ErrorPayload{
				Code:    code,
				Message: message,
			},
		},
	}
	
	return sendBinaryMessage(conn, errorMsg)
}

// sendBinaryMessage sends a binary message to the client
func sendBinaryMessage(conn *Connection, message *ws.WebSocketMessage) error {
	// Serialize the message
	data, err := proto.Marshal(message)
	if err != nil {
		return err
	}
	
	// Send the message
	conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
	return conn.conn.WriteMessage(websocket.BinaryMessage, data)
}

// BroadcastBinaryMessage broadcasts a binary message to all subscribed clients
func (h *BinaryMessageHandler) BroadcastBinaryMessage(message *ws.WebSocketMessage) error {
	// Get connections for the channel and symbol
	connections := h.server.pool.GetConnectionsByChannelAndSymbol(message.Channel, message.Symbol)
	
	// Serialize the message once
	data, err := proto.Marshal(message)
	if err != nil {
		h.logger.Error("Failed to marshal binary message for broadcast",
			zap.Error(err),
			zap.String("type", message.Type),
			zap.String("channel", message.Channel),
			zap.String("symbol", message.Symbol))
		return err
	}
	
	// Send to all connections
	for _, conn := range connections {
		conn.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
		if err := conn.conn.WriteMessage(websocket.BinaryMessage, data); err != nil {
			h.logger.Error("Failed to broadcast binary message",
				zap.Error(err),
				zap.String("remote_addr", conn.conn.RemoteAddr().String()))
			// Continue sending to other connections
			continue
		}
	}
	
	// Update stats
	h.server.pool.IncrementMessagesSent(int64(len(connections)))
	
	return nil
}

// CreateMarketDataMessage creates a market data message
func CreateMarketDataMessage(symbol string, lastPrice, bidPrice, askPrice, bidSize, askSize, volume float64) *ws.WebSocketMessage {
	return &ws.WebSocketMessage{
		Type:      "marketData",
		Channel:   "marketData",
		Symbol:    symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_MarketData{
			MarketData: &ws.MarketDataPayload{
				Symbol:    symbol,
				LastPrice: lastPrice,
				BidPrice:  bidPrice,
				AskPrice:  askPrice,
				BidSize:   bidSize,
				AskSize:   askSize,
				Volume:    volume,
				Timestamp: time.Now().UnixMilli(),
			},
		},
	}
}

// CreateOrderMessage creates an order message
func CreateOrderMessage(orderID, clientOrderID, symbol, side, orderType, status string, price, quantity, filledQty, avgFillPrice float64) *ws.WebSocketMessage {
	return &ws.WebSocketMessage{
		Type:      "order",
		Channel:   "orders",
		Symbol:    symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Order{
			Order: &ws.OrderPayload{
				OrderId:        orderID,
				ClientOrderId:  clientOrderID,
				Symbol:         symbol,
				Side:           side,
				Type:           orderType,
				Status:         status,
				Price:          price,
				Quantity:       quantity,
				FilledQuantity: filledQty,
				AvgFillPrice:   avgFillPrice,
				Timestamp:      time.Now().UnixMilli(),
			},
		},
	}
}

// CreateTradeMessage creates a trade message
func CreateTradeMessage(tradeID, orderID, symbol, side, exchange, feeCurrency string, price, quantity, fee float64) *ws.WebSocketMessage {
	return &ws.WebSocketMessage{
		Type:      "trade",
		Channel:   "trades",
		Symbol:    symbol,
		Timestamp: time.Now().UnixMilli(),
		Payload: &ws.WebSocketMessage_Trade{
			Trade: &ws.TradePayload{
				TradeId:     tradeID,
				OrderId:     orderID,
				Symbol:      symbol,
				Side:        side,
				Price:       price,
				Quantity:    quantity,
				Timestamp:   time.Now().UnixMilli(),
				Exchange:    exchange,
				Fee:         fee,
				FeeCurrency: feeCurrency,
			},
		},
	}
}
