package websocket

import (
	"sync"
	"time"

	"go.uber.org/zap"
)

// ConnectionPool manages a pool of WebSocket connections
type ConnectionPool struct {
	// Map of connections by channel
	channelConnections map[string]map[*Connection]bool
	// Map of connections by symbol
	symbolConnections map[string]map[*Connection]bool
	// All connections
	allConnections map[*Connection]bool
	// Mutex for thread safety
	mu sync.RWMutex
	// Logger
	logger *zap.Logger
	// Stats
	stats ConnectionPoolStats
}

// ConnectionPoolStats contains statistics about the connection pool
type ConnectionPoolStats struct {
	TotalConnections     int
	ConnectionsByChannel map[string]int
	ConnectionsBySymbol  map[string]int
	MessagesSent         int64
	MessagesReceived     int64
	LastStatsReset       time.Time
	mu                   sync.RWMutex
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(logger *zap.Logger) *ConnectionPool {
	return &ConnectionPool{
		channelConnections: make(map[string]map[*Connection]bool),
		symbolConnections:  make(map[string]map[*Connection]bool),
		allConnections:     make(map[*Connection]bool),
		logger:             logger,
		stats: ConnectionPoolStats{
			ConnectionsByChannel: make(map[string]int),
			ConnectionsBySymbol:  make(map[string]int),
			LastStatsReset:       time.Now(),
		},
	}
}

// AddConnection adds a connection to the pool
func (p *ConnectionPool) AddConnection(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Add to all connections
	p.allConnections[conn] = true

	// Update stats
	p.stats.mu.Lock()
	p.stats.TotalConnections++
	p.stats.mu.Unlock()

	p.logger.Debug("Connection added to pool",
		zap.String("remote_addr", conn.conn.RemoteAddr().String()),
		zap.Int("total_connections", p.stats.TotalConnections))
}

// RemoveConnection removes a connection from the pool
func (p *ConnectionPool) RemoveConnection(conn *Connection) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from all connections
	delete(p.allConnections, conn)

	// Remove from channel connections
	for channel := range conn.channels {
		if connections, ok := p.channelConnections[channel]; ok {
			delete(connections, conn)

			// Update stats
			p.stats.mu.Lock()
			p.stats.ConnectionsByChannel[channel]--
			if p.stats.ConnectionsByChannel[channel] <= 0 {
				delete(p.stats.ConnectionsByChannel, channel)
			}
			p.stats.mu.Unlock()

			// Clean up empty channel maps
			if len(connections) == 0 {
				delete(p.channelConnections, channel)
			}
		}
	}

	// Remove from symbol connections
	if conn.symbol != "" {
		if connections, ok := p.symbolConnections[conn.symbol]; ok {
			delete(connections, conn)

			// Update stats
			p.stats.mu.Lock()
			p.stats.ConnectionsBySymbol[conn.symbol]--
			if p.stats.ConnectionsBySymbol[conn.symbol] <= 0 {
				delete(p.stats.ConnectionsBySymbol, conn.symbol)
			}
			p.stats.mu.Unlock()

			// Clean up empty symbol maps
			if len(connections) == 0 {
				delete(p.symbolConnections, conn.symbol)
			}
		}
	}

	// Update stats
	p.stats.mu.Lock()
	p.stats.TotalConnections--
	p.stats.mu.Unlock()

	p.logger.Debug("Connection removed from pool",
		zap.String("remote_addr", conn.conn.RemoteAddr().String()),
		zap.Int("total_connections", p.stats.TotalConnections))
}

// SubscribeToChannel subscribes a connection to a channel
func (p *ConnectionPool) SubscribeToChannel(conn *Connection, channel string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Initialize channel map if it doesn't exist
	if _, ok := p.channelConnections[channel]; !ok {
		p.channelConnections[channel] = make(map[*Connection]bool)
	}

	// Add connection to channel
	p.channelConnections[channel][conn] = true

	// Update stats
	p.stats.mu.Lock()
	p.stats.ConnectionsByChannel[channel]++
	p.stats.mu.Unlock()

	p.logger.Debug("Connection subscribed to channel",
		zap.String("remote_addr", conn.conn.RemoteAddr().String()),
		zap.String("channel", channel),
		zap.Int("channel_connections", p.stats.ConnectionsByChannel[channel]))
}

// UnsubscribeFromChannel unsubscribes a connection from a channel
func (p *ConnectionPool) UnsubscribeFromChannel(conn *Connection, channel string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove connection from channel
	if connections, ok := p.channelConnections[channel]; ok {
		delete(connections, conn)

		// Update stats
		p.stats.mu.Lock()
		p.stats.ConnectionsByChannel[channel]--
		if p.stats.ConnectionsByChannel[channel] <= 0 {
			delete(p.stats.ConnectionsByChannel, channel)
		}
		p.stats.mu.Unlock()

		// Clean up empty channel maps
		if len(connections) == 0 {
			delete(p.channelConnections, channel)
		}

		p.logger.Debug("Connection unsubscribed from channel",
			zap.String("remote_addr", conn.conn.RemoteAddr().String()),
			zap.String("channel", channel))
	}
}

// SetSymbol sets the symbol for a connection
func (p *ConnectionPool) SetSymbol(conn *Connection, symbol string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Remove from old symbol if it exists
	if conn.symbol != "" {
		if connections, ok := p.symbolConnections[conn.symbol]; ok {
			delete(connections, conn)

			// Update stats
			p.stats.mu.Lock()
			p.stats.ConnectionsBySymbol[conn.symbol]--
			if p.stats.ConnectionsBySymbol[conn.symbol] <= 0 {
				delete(p.stats.ConnectionsBySymbol, conn.symbol)
			}
			p.stats.mu.Unlock()

			// Clean up empty symbol maps
			if len(connections) == 0 {
				delete(p.symbolConnections, conn.symbol)
			}
		}
	}

	// Set new symbol
	conn.symbol = symbol

	// Add to new symbol if it's not empty
	if symbol != "" {
		// Initialize symbol map if it doesn't exist
		if _, ok := p.symbolConnections[symbol]; !ok {
			p.symbolConnections[symbol] = make(map[*Connection]bool)
		}

		// Add connection to symbol
		p.symbolConnections[symbol][conn] = true

		// Update stats
		p.stats.mu.Lock()
		p.stats.ConnectionsBySymbol[symbol]++
		p.stats.mu.Unlock()

		p.logger.Debug("Connection symbol set",
			zap.String("remote_addr", conn.conn.RemoteAddr().String()),
			zap.String("symbol", symbol))
	}
}

// GetConnectionsByChannel gets all connections subscribed to a channel
func (p *ConnectionPool) GetConnectionsByChannel(channel string) []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var connections []*Connection

	if channelConns, ok := p.channelConnections[channel]; ok {
		for conn := range channelConns {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetConnectionsBySymbol gets all connections for a symbol
func (p *ConnectionPool) GetConnectionsBySymbol(symbol string) []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var connections []*Connection

	if symbolConns, ok := p.symbolConnections[symbol]; ok {
		for conn := range symbolConns {
			connections = append(connections, conn)
		}
	}

	return connections
}

// GetConnectionsByChannelAndSymbol gets all connections subscribed to a channel and symbol
func (p *ConnectionPool) GetConnectionsByChannelAndSymbol(channel, symbol string) []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var connections []*Connection

	if channelConns, ok := p.channelConnections[channel]; ok {
		if symbol == "" {
			// If no symbol specified, return all connections for the channel
			for conn := range channelConns {
				connections = append(connections, conn)
			}
		} else {
			// If symbol specified, filter by symbol
			for conn := range channelConns {
				if conn.symbol == symbol {
					connections = append(connections, conn)
				}
			}
		}
	}

	return connections
}

// GetAllConnections gets all connections in the pool
func (p *ConnectionPool) GetAllConnections() []*Connection {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var connections []*Connection

	for conn := range p.allConnections {
		connections = append(connections, conn)
	}

	return connections
}

// GetStats gets the connection pool statistics
func (p *ConnectionPool) GetStats() ConnectionPoolStats {
	p.stats.mu.RLock()
	defer p.stats.mu.RUnlock()

	// Create a copy to avoid race conditions
	statsCopy := ConnectionPoolStats{
		TotalConnections:     p.stats.TotalConnections,
		ConnectionsByChannel: make(map[string]int),
		ConnectionsBySymbol:  make(map[string]int),
		MessagesSent:         p.stats.MessagesSent,
		MessagesReceived:     p.stats.MessagesReceived,
		LastStatsReset:       p.stats.LastStatsReset,
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
}

// IncrementMessagesSent increments the messages sent counter
func (p *ConnectionPool) IncrementMessagesSent(count int64) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.MessagesSent += count
}

// IncrementMessagesReceived increments the messages received counter
func (p *ConnectionPool) IncrementMessagesReceived(count int64) {
	p.stats.mu.Lock()
	defer p.stats.mu.Unlock()

	p.stats.MessagesReceived += count
}
