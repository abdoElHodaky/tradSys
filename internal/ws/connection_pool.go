package ws

import (
	"context"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// Connection represents a WebSocket connection
type Connection struct {
	ID        string
	Conn      *websocket.Conn
	Channel   string
	Symbol    string
	CreatedAt time.Time
	LastPing  time.Time
	mu        sync.RWMutex
}

// ConnectionPoolStats represents statistics for the connection pool
type ConnectionPoolStats struct {
	TotalConnections     int
	ConnectionsByChannel map[string]int
	ConnectionsBySymbol  map[string]int
	MessagesSent         uint64
	MessagesReceived     uint64
	LastStatsReset       time.Time
	mu                   sync.RWMutex
}

// ConnectionPool manages WebSocket connections
type ConnectionPool struct {
	connections map[string]*Connection
	stats       ConnectionPoolStats
	logger      *zap.Logger
	mu          sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(logger *zap.Logger) *ConnectionPool {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &ConnectionPool{
		connections: make(map[string]*Connection),
		stats: ConnectionPoolStats{
			ConnectionsByChannel: make(map[string]int),
			ConnectionsBySymbol:  make(map[string]int),
			LastStatsReset:       time.Now(),
			mu:                   sync.RWMutex{},
		},
		logger: logger,
		mu:     sync.RWMutex{},
	}
}

// AddConnection adds a connection to the pool
func (p *ConnectionPool) AddConnection(id string, conn *websocket.Conn, channel, symbol string) *Connection {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Create the connection
	connection := &Connection{
		ID:        id,
		Conn:      conn,
		Channel:   channel,
		Symbol:    symbol,
		CreatedAt: time.Now(),
		LastPing:  time.Now(),
		mu:        sync.RWMutex{},
	}

	// Add to the pool
	p.connections[id] = connection

	// Update stats
	p.stats.mu.Lock()
	p.stats.TotalConnections++
	p.stats.ConnectionsByChannel[channel]++
	if symbol != "" {
		p.stats.ConnectionsBySymbol[symbol]++
	}
	p.stats.mu.Unlock()

	p.logger.Debug("Added connection to pool",
		zap.String("id", id),
		zap.String("channel", channel),
		zap.String("symbol", symbol),
	)

	return connection
}

// RemoveConnection removes a connection from the pool
func (p *ConnectionPool) RemoveConnection(id string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Get the connection
	connection, exists := p.connections[id]
	if !exists {
		return
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.TotalConnections--
	p.stats.ConnectionsByChannel[connection.Channel]--
	if connection.Symbol != "" {
		p.stats.ConnectionsBySymbol[connection.Symbol]--
	}
	p.stats.mu.Unlock()

	// Remove from the pool
	delete(p.connections, id)

	p.logger.Debug("Removed connection from pool",
		zap.String("id", id),
		zap.String("channel", connection.Channel),
		zap.String("symbol", connection.Symbol),
	)
}

// GetConnection gets a connection by ID
func (p *ConnectionPool) GetConnection(id string) (*Connection, bool) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connection, exists := p.connections[id]
	return connection, exists
}

// GetConnectionsByChannel gets all connections for a channel
func (p *ConnectionPool) GetConnectionsByChannel(channel string) []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connections := make([]*Connection, 0)
	for _, connection := range p.connections {
		if connection.Channel == channel {
			connections = append(connections, connection)
		}
	}

	return connections
}

// GetConnectionsBySymbol gets all connections for a symbol
func (p *ConnectionPool) GetConnectionsBySymbol(symbol string) []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	connections := make([]*Connection, 0)
	for _, connection := range p.connections {
		if connection.Symbol == symbol {
			connections = append(connections, connection)
		}
	}

	return connections
}

// GetStats gets the connection pool statistics
func (p *ConnectionPool) GetStats() ConnectionPoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	statsCopy := ConnectionPoolStats{
		TotalConnections:      p.stats.TotalConnections,
		ConnectionsByChannel:  make(map[string]int),
		ConnectionsBySymbol:   make(map[string]int),
		MessagesSent:          p.stats.MessagesSent,
		MessagesReceived:      p.stats.MessagesReceived,
		LastStatsReset:        p.stats.LastStatsReset,
	}
	
	for channel, count := range p.stats.ConnectionsByChannel {
		statsCopy.ConnectionsByChannel[channel] = count
	}
	
	for symbol, count := range p.stats.ConnectionsBySymbol {
		statsCopy.ConnectionsBySymbol[symbol] = count
	}
	
	return statsCopy
}

// ResetStats resets the connection pool statistics
func (p *ConnectionPool) ResetStats() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.MessagesSent = 0
	p.stats.MessagesReceived = 0
	p.stats.LastStatsReset = time.Now()

	p.logger.Debug("Reset connection pool statistics")
}

// IncrementMessagesSent increments the messages sent counter
func (p *ConnectionPool) IncrementMessagesSent() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.MessagesSent++
}

// IncrementMessagesReceived increments the messages received counter
func (p *ConnectionPool) IncrementMessagesReceived() {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.MessagesReceived++
}

// CloseAll closes all connections in the pool
func (p *ConnectionPool) CloseAll(ctx context.Context) {
	p.mu.Lock()
	defer p.mu.Unlock()

	for id, connection := range p.connections {
		// Close the connection
		err := connection.Conn.Close()
		if err != nil {
			p.logger.Error("Failed to close connection",
				zap.String("id", id),
				zap.Error(err),
			)
		}

		// Remove from the pool
		delete(p.connections, id)
	}

	// Reset stats
	p.stats.mu.Lock()
	p.stats.TotalConnections = 0
	p.stats.ConnectionsByChannel = make(map[string]int)
	p.stats.ConnectionsBySymbol = make(map[string]int)
	p.stats.mu.Unlock()

	p.logger.Info("Closed all connections in pool")
}

// UpdateConnectionPing updates the last ping time for a connection
func (p *ConnectionPool) UpdateConnectionPing(id string) {
	p.mu.RLock()
	connection, exists := p.connections[id]
	p.mu.RUnlock()

	if !exists {
		return
	}

	connection.mu.Lock()
	connection.LastPing = time.Now()
	connection.mu.Unlock()
}

// CleanupStaleConnections removes connections that haven't pinged in a while
func (p *ConnectionPool) CleanupStaleConnections(ctx context.Context, maxAge time.Duration) int {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	removed := 0

	for id, connection := range p.connections {
		connection.mu.RLock()
		lastPing := connection.LastPing
		connection.mu.RUnlock()

		if now.Sub(lastPing) > maxAge {
			// Close the connection
			err := connection.Conn.Close()
			if err != nil {
				p.logger.Error("Failed to close stale connection",
					zap.String("id", id),
					zap.Error(err),
				)
			}

			// Update stats
			p.stats.mu.Lock()
			p.stats.TotalConnections--
			p.stats.ConnectionsByChannel[connection.Channel]--
			if connection.Symbol != "" {
				p.stats.ConnectionsBySymbol[connection.Symbol]--
			}
			p.stats.mu.Unlock()

			// Remove from the pool
			delete(p.connections, id)
			removed++

			p.logger.Debug("Removed stale connection",
				zap.String("id", id),
				zap.String("channel", connection.Channel),
				zap.String("symbol", connection.Symbol),
				zap.Duration("age", now.Sub(lastPing)),
			)
		}
	}

	if removed > 0 {
		p.logger.Info("Cleaned up stale connections",
			zap.Int("removed", removed),
		)
	}

	return removed
}

// BroadcastToChannel broadcasts a message to all connections in a channel
func (p *ConnectionPool) BroadcastToChannel(channel string, message []byte) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sent := 0
	for _, connection := range p.connections {
		if connection.Channel == channel {
			err := connection.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				p.logger.Error("Failed to broadcast message to connection",
					zap.String("id", connection.ID),
					zap.String("channel", channel),
					zap.Error(err),
				)
				continue
			}
			sent++
		}
	}

	// Update stats
	if sent > 0 {
		p.stats.mu.Lock()
		p.stats.MessagesSent += uint64(sent)
		p.stats.mu.Unlock()
	}

	return sent
}

// BroadcastToSymbol broadcasts a message to all connections for a symbol
func (p *ConnectionPool) BroadcastToSymbol(symbol string, message []byte) int {
	p.mu.RLock()
	defer p.mu.RUnlock()

	sent := 0
	for _, connection := range p.connections {
		if connection.Symbol == symbol {
			err := connection.Conn.WriteMessage(websocket.TextMessage, message)
			if err != nil {
				p.logger.Error("Failed to broadcast message to connection",
					zap.String("id", connection.ID),
					zap.String("symbol", symbol),
					zap.Error(err),
				)
				continue
			}
			sent++
		}
	}

	// Update stats
	if sent > 0 {
		p.stats.mu.Lock()
		p.stats.MessagesSent += uint64(sent)
		p.stats.mu.Unlock()
	}

	return sent
}

// SendToConnection sends a message to a specific connection
func (p *ConnectionPool) SendToConnection(id string, message []byte) error {
	p.mu.RLock()
	connection, exists := p.connections[id]
	p.mu.RUnlock()

	if !exists {
		return ErrConnectionNotFound
	}

	err := connection.Conn.WriteMessage(websocket.TextMessage, message)
	if err != nil {
		p.logger.Error("Failed to send message to connection",
			zap.String("id", id),
			zap.Error(err),
		)
		return err
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.MessagesSent++
	p.stats.mu.Unlock()

	return nil
}

// ErrConnectionNotFound is returned when a connection is not found
var ErrConnectionNotFound = fmt.Errorf("connection not found")

