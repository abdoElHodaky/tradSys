package performance

import (
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketOptimizer provides performance optimizations for WebSocket connections
type WebSocketOptimizer struct {
	logger           *zap.Logger
	bufferPool       *sync.Pool
	compressionLevel int
	writeBufferSize  int
	readBufferSize   int
	messageStats     *MessageStats
}

// WebSocketOptimizerOptions contains options for the WebSocket optimizer
type WebSocketOptimizerOptions struct {
	CompressionLevel int
	WriteBufferSize  int
	ReadBufferSize   int
	BufferSize       int
}

// MessageStats tracks WebSocket message statistics
type MessageStats struct {
	TotalMessages     int64
	TotalBytes        int64
	CompressedBytes   int64
	CompressionRatio  float64
	AverageLatency    time.Duration
	MaxLatency        time.Duration
	MessagesSent      int64
	MessagesReceived  int64
	ErrorCount        int64
	lastCalculated    time.Time
	mutex             sync.RWMutex
}

// NewWebSocketOptimizer creates a new WebSocket optimizer
func NewWebSocketOptimizer(logger *zap.Logger, options WebSocketOptimizerOptions) *WebSocketOptimizer {
	// Set default values if not provided
	if options.CompressionLevel == 0 {
		options.CompressionLevel = websocket.DefaultCompressionLevel
	}
	if options.WriteBufferSize == 0 {
		options.WriteBufferSize = 4096
	}
	if options.ReadBufferSize == 0 {
		options.ReadBufferSize = 4096
	}
	if options.BufferSize == 0 {
		options.BufferSize = 1024
	}

	// Create buffer pool
	bufferPool := &sync.Pool{
		New: func() interface{} {
			return make([]byte, options.BufferSize)
		},
	}

	return &WebSocketOptimizer{
		logger:           logger,
		bufferPool:       bufferPool,
		compressionLevel: options.CompressionLevel,
		writeBufferSize:  options.WriteBufferSize,
		readBufferSize:   options.ReadBufferSize,
		messageStats:     &MessageStats{lastCalculated: time.Now()},
	}
}

// GetBuffer gets a buffer from the pool
func (o *WebSocketOptimizer) GetBuffer() []byte {
	return o.bufferPool.Get().([]byte)
}

// PutBuffer returns a buffer to the pool
func (o *WebSocketOptimizer) PutBuffer(buffer []byte) {
	// Clear buffer before returning to pool
	for i := range buffer {
		buffer[i] = 0
	}
	o.bufferPool.Put(buffer)
}

// OptimizeConnection optimizes a WebSocket connection
func (o *WebSocketOptimizer) OptimizeConnection(conn *websocket.Conn) {
	// Set buffer sizes
	conn.SetReadLimit(int64(o.readBufferSize))
	conn.SetWriteBuffer(o.writeBufferSize)
	conn.SetReadBuffer(o.readBufferSize)

	// Enable compression
	conn.EnableWriteCompression(true)
	conn.SetCompressionLevel(o.compressionLevel)
}

// TrackMessageSent tracks a sent message
func (o *WebSocketOptimizer) TrackMessageSent(messageType int, messageSize int, compressedSize int, latency time.Duration) {
	o.messageStats.mutex.Lock()
	defer o.messageStats.mutex.Unlock()

	o.messageStats.TotalMessages++
	o.messageStats.MessagesSent++
	o.messageStats.TotalBytes += int64(messageSize)
	o.messageStats.CompressedBytes += int64(compressedSize)

	// Update latency statistics
	o.messageStats.AverageLatency = (o.messageStats.AverageLatency*time.Duration(o.messageStats.TotalMessages-1) + latency) / time.Duration(o.messageStats.TotalMessages)
	if latency > o.messageStats.MaxLatency {
		o.messageStats.MaxLatency = latency
	}

	// Calculate compression ratio
	if o.messageStats.TotalBytes > 0 {
		o.messageStats.CompressionRatio = float64(o.messageStats.CompressedBytes) / float64(o.messageStats.TotalBytes)
	}
}

// TrackMessageReceived tracks a received message
func (o *WebSocketOptimizer) TrackMessageReceived(messageType int, messageSize int) {
	o.messageStats.mutex.Lock()
	defer o.messageStats.mutex.Unlock()

	o.messageStats.TotalMessages++
	o.messageStats.MessagesReceived++
	o.messageStats.TotalBytes += int64(messageSize)
}

// TrackError tracks an error
func (o *WebSocketOptimizer) TrackError(err error) {
	o.messageStats.mutex.Lock()
	defer o.messageStats.mutex.Unlock()

	o.messageStats.ErrorCount++
	o.logger.Error("WebSocket error", zap.Error(err))
}

// GetMessageStats returns the current message statistics
func (o *WebSocketOptimizer) GetMessageStats() MessageStats {
	o.messageStats.mutex.RLock()
	defer o.messageStats.mutex.RUnlock()

	return *o.messageStats
}

// ResetMessageStats resets the message statistics
func (o *WebSocketOptimizer) ResetMessageStats() {
	o.messageStats.mutex.Lock()
	defer o.messageStats.mutex.Unlock()

	o.messageStats.TotalMessages = 0
	o.messageStats.TotalBytes = 0
	o.messageStats.CompressedBytes = 0
	o.messageStats.CompressionRatio = 0
	o.messageStats.AverageLatency = 0
	o.messageStats.MaxLatency = 0
	o.messageStats.MessagesSent = 0
	o.messageStats.MessagesReceived = 0
	o.messageStats.ErrorCount = 0
	o.messageStats.lastCalculated = time.Now()
}

// OptimizedWriter is a wrapper for WebSocket connection with optimized writing
type OptimizedWriter struct {
	conn      *websocket.Conn
	optimizer *WebSocketOptimizer
}

// NewOptimizedWriter creates a new optimized writer
func (o *WebSocketOptimizer) NewOptimizedWriter(conn *websocket.Conn) *OptimizedWriter {
	return &OptimizedWriter{
		conn:      conn,
		optimizer: o,
	}
}

// WriteMessage writes a message to the WebSocket connection with optimization
func (w *OptimizedWriter) WriteMessage(messageType int, data []byte) error {
	startTime := time.Now()

	// Get buffer from pool
	buffer := w.optimizer.GetBuffer()
	defer w.optimizer.PutBuffer(buffer)

	// Copy data to buffer if it fits
	var messageData []byte
	if len(data) <= len(buffer) {
		copy(buffer, data)
		messageData = buffer[:len(data)]
	} else {
		messageData = data
	}

	// Write message
	err := w.conn.WriteMessage(messageType, messageData)
	if err != nil {
		w.optimizer.TrackError(err)
		return err
	}

	// Track message statistics
	latency := time.Since(startTime)
	w.optimizer.TrackMessageSent(messageType, len(data), len(messageData), latency)

	return nil
}

// WriteJSON writes a JSON message to the WebSocket connection with optimization
func (w *OptimizedWriter) WriteJSON(v interface{}) error {
	startTime := time.Now()

	// Write JSON message
	err := w.conn.WriteJSON(v)
	if err != nil {
		w.optimizer.TrackError(err)
		return err
	}

	// Track message statistics (approximate size)
	latency := time.Since(startTime)
	w.optimizer.TrackMessageSent(websocket.TextMessage, 0, 0, latency)

	return nil
}

// Close closes the WebSocket connection
func (w *OptimizedWriter) Close() error {
	return w.conn.Close()
}
