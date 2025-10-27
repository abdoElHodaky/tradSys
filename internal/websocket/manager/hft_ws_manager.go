package manager

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"

	"github.com/abdoElHodaky/tradSys/internal/trading/metrics"
	"github.com/abdoElHodaky/tradSys/pkg/common/pool"
)

// HFTWebSocketManager manages WebSocket connections with HFT optimizations
type HFTWebSocketManager struct {
	// Connection management
	connections sync.Map // map[string]*HFTConnection
	connCount   int64

	// Message handling
	messagePool     *pool.WebSocketMessagePool
	pricePool       *pool.PriceMessagePool
	orderUpdatePool *pool.OrderUpdateMessagePool

	// Broadcasting
	broadcastChan chan *BroadcastMessage

	// Configuration
	config *HFTWebSocketConfig

	// Upgrader
	upgrader websocket.Upgrader

	// Context for shutdown
	ctx    context.Context
	cancel context.CancelFunc

	// Metrics
	metrics *metrics.BaselineMetrics
}

// HFTWebSocketConfig contains WebSocket configuration
type HFTWebSocketConfig struct {
	ReadBufferSize   int           `yaml:"read_buffer_size" default:"4096"`
	WriteBufferSize  int           `yaml:"write_buffer_size" default:"4096"`
	HandshakeTimeout time.Duration `yaml:"handshake_timeout" default:"10s"`
	ReadTimeout      time.Duration `yaml:"read_timeout" default:"60s"`
	WriteTimeout     time.Duration `yaml:"write_timeout" default:"10s"`
	PongTimeout      time.Duration `yaml:"pong_timeout" default:"60s"`
	PingPeriod       time.Duration `yaml:"ping_period" default:"54s"`
	MaxMessageSize   int64         `yaml:"max_message_size" default:"512"`
	BinaryProtocol   bool          `yaml:"binary_protocol" default:"true"`
	CompressionLevel int           `yaml:"compression_level" default:"1"`
	BroadcastWorkers int           `yaml:"broadcast_workers" default:"4"`
	BroadcastBuffer  int           `yaml:"broadcast_buffer" default:"1000"`
}

// HFTConnection represents an optimized WebSocket connection
type HFTConnection struct {
	ID      string
	UserID  string
	Conn    *websocket.Conn
	Send    chan []byte
	Manager *HFTWebSocketManager

	// Subscription management
	Subscriptions sync.Map // map[string]bool

	// Connection state
	LastPong time.Time
	Created  time.Time

	// Context for cleanup
	ctx    context.Context
	cancel context.CancelFunc
}

// BroadcastMessage represents a message to be broadcasted
type BroadcastMessage struct {
	Channel string
	Data    interface{}
	Filter  func(*HFTConnection) bool // Optional filter function
}

// NewHFTWebSocketManager creates a new HFT WebSocket manager
func NewHFTWebSocketManager(config *HFTWebSocketConfig) *HFTWebSocketManager {
	if config == nil {
		config = &HFTWebSocketConfig{
			ReadBufferSize:   4096,
			WriteBufferSize:  4096,
			HandshakeTimeout: 10 * time.Second,
			ReadTimeout:      60 * time.Second,
			WriteTimeout:     10 * time.Second,
			PongTimeout:      60 * time.Second,
			PingPeriod:       54 * time.Second,
			MaxMessageSize:   512,
			BinaryProtocol:   true,
			CompressionLevel: 1,
			BroadcastWorkers: 4,
			BroadcastBuffer:  1000,
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	manager := &HFTWebSocketManager{
		messagePool:     pool.NewWebSocketMessagePool(),
		pricePool:       pool.NewPriceMessagePool(),
		orderUpdatePool: pool.NewOrderUpdateMessagePool(),
		broadcastChan:   make(chan *BroadcastMessage, config.BroadcastBuffer),
		config:          config,
		ctx:             ctx,
		cancel:          cancel,
		upgrader: websocket.Upgrader{
			ReadBufferSize:   config.ReadBufferSize,
			WriteBufferSize:  config.WriteBufferSize,
			HandshakeTimeout: config.HandshakeTimeout,
			CheckOrigin: func(r *http.Request) bool {
				// In production, implement proper origin checking
				return true
			},
		},
	}

	// Initialize metrics
	manager.metrics = metrics.GlobalMetrics
	if manager.metrics == nil {
		metrics.InitMetrics()
		manager.metrics = metrics.GlobalMetrics
	}

	// Start broadcast workers
	for i := 0; i < config.BroadcastWorkers; i++ {
		go manager.broadcastWorker()
	}

	return manager
}

// HandleConnection handles a new WebSocket connection
func (m *HFTWebSocketManager) HandleConnection(c *gin.Context) {
	tracker := metrics.TrackWSLatency()
	defer tracker.Finish()

	// Upgrade connection
	conn, err := m.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		m.metrics.RecordError()
		return
	}

	// Create HFT connection
	userID := c.GetString("user_id") // From auth middleware
	if userID == "" {
		conn.Close()
		return
	}

	hftConn := m.createConnection(conn, userID)

	// Register connection
	m.connections.Store(hftConn.ID, hftConn)
	atomic.AddInt64(&m.connCount, 1)
	m.metrics.UpdateActiveConnections(int(atomic.LoadInt64(&m.connCount)))

	// Start connection handlers
	go hftConn.readPump()
	go hftConn.writePump()
}

// createConnection creates a new HFT connection
func (m *HFTWebSocketManager) createConnection(conn *websocket.Conn, userID string) *HFTConnection {
	ctx, cancel := context.WithCancel(m.ctx)

	hftConn := &HFTConnection{
		ID:      fmt.Sprintf("%s-%d", userID, time.Now().UnixNano()),
		UserID:  userID,
		Conn:    conn,
		Send:    make(chan []byte, 256),
		Manager: m,
		Created: time.Now(),
		ctx:     ctx,
		cancel:  cancel,
	}

	// Configure connection
	conn.SetReadLimit(m.config.MaxMessageSize)
	conn.SetReadDeadline(time.Now().Add(m.config.PongTimeout))
	conn.SetPongHandler(func(string) error {
		hftConn.LastPong = time.Now()
		conn.SetReadDeadline(time.Now().Add(m.config.PongTimeout))
		return nil
	})

	return hftConn
}

// readPump handles reading messages from the WebSocket connection
func (c *HFTConnection) readPump() {
	defer func() {
		c.Manager.unregisterConnection(c)
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		default:
			_, message, err := c.Conn.ReadMessage()
			if err != nil {
				if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
					c.Manager.metrics.RecordError()
				}
				return
			}

			// Process message
			c.handleMessage(message)
		}
	}
}

// writePump handles writing messages to the WebSocket connection
func (c *HFTConnection) writePump() {
	ticker := time.NewTicker(c.Manager.config.PingPeriod)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case <-c.ctx.Done():
			return
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteTimeout))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			tracker := metrics.TrackWSLatency()

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Manager.metrics.RecordError()
				tracker.Finish()
				return
			}

			tracker.Finish()

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(c.Manager.config.WriteTimeout))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming WebSocket messages
func (c *HFTConnection) handleMessage(message []byte) {
	// Get pooled message
	msg := c.Manager.messagePool.Get()
	defer c.Manager.messagePool.Put(msg)

	// Parse message
	if err := json.Unmarshal(message, msg); err != nil {
		c.Manager.metrics.RecordError()
		return
	}

	// Handle different message types
	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg)
	case "unsubscribe":
		c.handleUnsubscribe(msg)
	case "ping":
		c.handlePing(msg)
	default:
		// Unknown message type
		c.Manager.metrics.RecordError()
	}
}

// handleSubscribe handles subscription requests
func (c *HFTConnection) handleSubscribe(msg *pool.WebSocketMessage) {
	if msg.Channel == "" {
		return
	}

	c.Subscriptions.Store(msg.Channel, true)

	// Send confirmation
	response := c.Manager.messagePool.Get()
	defer c.Manager.messagePool.Put(response)

	response.Type = "subscribed"
	response.Channel = msg.Channel
	response.Timestamp = time.Now().UnixNano()
	response.RequestID = msg.RequestID

	c.sendMessage(response)
}

// handleUnsubscribe handles unsubscription requests
func (c *HFTConnection) handleUnsubscribe(msg *pool.WebSocketMessage) {
	if msg.Channel == "" {
		return
	}

	c.Subscriptions.Delete(msg.Channel)

	// Send confirmation
	response := c.Manager.messagePool.Get()
	defer c.Manager.messagePool.Put(response)

	response.Type = "unsubscribed"
	response.Channel = msg.Channel
	response.Timestamp = time.Now().UnixNano()
	response.RequestID = msg.RequestID

	c.sendMessage(response)
}

// handlePing handles ping messages
func (c *HFTConnection) handlePing(msg *pool.WebSocketMessage) {
	response := c.Manager.messagePool.Get()
	defer c.Manager.messagePool.Put(response)

	response.Type = "pong"
	response.Timestamp = time.Now().UnixNano()
	response.RequestID = msg.RequestID

	c.sendMessage(response)
}

// sendMessage sends a message to the connection
func (c *HFTConnection) sendMessage(msg *pool.WebSocketMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		c.Manager.metrics.RecordError()
		return
	}

	select {
	case c.Send <- data:
	case <-c.ctx.Done():
	default:
		// Channel is full, close connection
		c.cancel()
	}
}

// unregisterConnection removes a connection from the manager
func (m *HFTWebSocketManager) unregisterConnection(conn *HFTConnection) {
	m.connections.Delete(conn.ID)
	atomic.AddInt64(&m.connCount, -1)
	m.metrics.UpdateActiveConnections(int(atomic.LoadInt64(&m.connCount)))
	conn.cancel()
	close(conn.Send)
}

// BroadcastPriceUpdate broadcasts a price update to subscribed connections
func (m *HFTWebSocketManager) BroadcastPriceUpdate(symbol string, price, volume float64) {
	priceMsg := m.pricePool.Get()
	defer m.pricePool.Put(priceMsg)

	priceMsg.Symbol = symbol
	priceMsg.Price = price
	priceMsg.Volume = volume
	priceMsg.Timestamp = time.Now().UnixNano()

	broadcast := &BroadcastMessage{
		Channel: fmt.Sprintf("price.%s", symbol),
		Data:    priceMsg,
		Filter: func(conn *HFTConnection) bool {
			_, subscribed := conn.Subscriptions.Load(fmt.Sprintf("price.%s", symbol))
			return subscribed
		},
	}

	select {
	case m.broadcastChan <- broadcast:
	default:
		// Broadcast channel is full, drop message
		m.metrics.RecordError()
	}
}

// BroadcastOrderUpdate broadcasts an order update to the specific user
func (m *HFTWebSocketManager) BroadcastOrderUpdate(userID, orderID, symbol, side, status string, filledQty, avgPrice float64) {
	orderMsg := m.orderUpdatePool.Get()
	defer m.orderUpdatePool.Put(orderMsg)

	orderMsg.OrderID = orderID
	orderMsg.Symbol = symbol
	orderMsg.Side = side
	orderMsg.Status = status
	orderMsg.FilledQuantity = filledQty
	orderMsg.AveragePrice = avgPrice
	orderMsg.Timestamp = time.Now().UnixNano()

	broadcast := &BroadcastMessage{
		Channel: fmt.Sprintf("orders.%s", userID),
		Data:    orderMsg,
		Filter: func(conn *HFTConnection) bool {
			return conn.UserID == userID
		},
	}

	select {
	case m.broadcastChan <- broadcast:
	default:
		m.metrics.RecordError()
	}
}

// broadcastWorker processes broadcast messages
func (m *HFTWebSocketManager) broadcastWorker() {
	for {
		select {
		case <-m.ctx.Done():
			return
		case broadcast := <-m.broadcastChan:
			m.processBroadcast(broadcast)
		}
	}
}

// processBroadcast sends a broadcast message to matching connections
func (m *HFTWebSocketManager) processBroadcast(broadcast *BroadcastMessage) {
	msg := m.messagePool.Get()
	defer m.messagePool.Put(msg)

	msg.Type = "data"
	msg.Channel = broadcast.Channel
	msg.Data = broadcast.Data
	msg.Timestamp = time.Now().UnixNano()

	data, err := json.Marshal(msg)
	if err != nil {
		m.metrics.RecordError()
		return
	}

	// Send to matching connections
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*HFTConnection)

		// Apply filter if provided
		if broadcast.Filter != nil && !broadcast.Filter(conn) {
			return true
		}

		// Check if connection is subscribed to channel
		if _, subscribed := conn.Subscriptions.Load(broadcast.Channel); !subscribed {
			return true
		}

		// Send message
		select {
		case conn.Send <- data:
		case <-conn.ctx.Done():
		default:
			// Connection is slow, skip
		}

		return true
	})
}

// GetConnectionCount returns the current number of active connections
func (m *HFTWebSocketManager) GetConnectionCount() int64 {
	return atomic.LoadInt64(&m.connCount)
}

// GetConnectionStats returns connection statistics
func (m *HFTWebSocketManager) GetConnectionStats() map[string]interface{} {
	stats := make(map[string]interface{})

	stats["total_connections"] = atomic.LoadInt64(&m.connCount)
	stats["broadcast_buffer_size"] = len(m.broadcastChan)
	stats["broadcast_buffer_capacity"] = cap(m.broadcastChan)

	// Count connections by user
	userCounts := make(map[string]int)
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*HFTConnection)
		userCounts[conn.UserID]++
		return true
	})
	stats["connections_by_user"] = userCounts

	return stats
}

// Shutdown gracefully shuts down the WebSocket manager
func (m *HFTWebSocketManager) Shutdown(timeout time.Duration) error {
	// Cancel context to stop workers
	m.cancel()

	// Close all connections
	m.connections.Range(func(key, value interface{}) bool {
		conn := value.(*HFTConnection)
		conn.cancel()
		conn.Conn.Close()
		return true
	})

	// Wait for shutdown with timeout
	done := make(chan struct{})
	go func() {
		// Wait for all connections to close
		for atomic.LoadInt64(&m.connCount) > 0 {
			time.Sleep(10 * time.Millisecond)
		}
		close(done)
	}()

	select {
	case <-done:
		return nil
	case <-time.After(timeout):
		return fmt.Errorf("shutdown timeout exceeded")
	}
}
