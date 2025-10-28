package performance

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// WebSocketOptimizer provides optimization for WebSocket connections
type WebSocketOptimizer struct {
	// Configuration
	compressionLevel     int
	writeBufferSize      int
	readBufferSize       int
	writeDeadline        time.Duration
	readDeadline         time.Duration
	pongWait             time.Duration
	pingPeriod           time.Duration
	maxMessageSize       int64
	enableCompression    bool
	enableBinaryMessages bool
	batchingEnabled      bool
	batchSize            int
	batchInterval        time.Duration

	// Logging
	logger *zap.Logger

	// Connection pool
	connectionPool sync.Pool
}

// WebSocketOptimizerOptions contains options for the WebSocket optimizer
type WebSocketOptimizerOptions struct {
	CompressionLevel     int
	WriteBufferSize      int
	ReadBufferSize       int
	WriteDeadline        time.Duration
	ReadDeadline         time.Duration
	PongWait             time.Duration
	PingPeriod           time.Duration
	MaxMessageSize       int64
	EnableCompression    bool
	EnableBinaryMessages bool
	BatchingEnabled      bool
	BatchSize            int
	BatchInterval        time.Duration
}

// DefaultWebSocketOptimizerOptions returns default WebSocket optimizer options
func DefaultWebSocketOptimizerOptions() WebSocketOptimizerOptions {
	return WebSocketOptimizerOptions{
		CompressionLevel:     6, // Default compression level (1-9)
		WriteBufferSize:      4096,
		ReadBufferSize:       4096,
		WriteDeadline:        10 * time.Second,
		ReadDeadline:         60 * time.Second,
		PongWait:             60 * time.Second,
		PingPeriod:           (60 * time.Second * 9) / 10,
		MaxMessageSize:       512 * 1024, // 512KB
		EnableCompression:    true,
		EnableBinaryMessages: true,
		BatchingEnabled:      true,
		BatchSize:            10,
		BatchInterval:        100 * time.Millisecond,
	}
}

// NewWebSocketOptimizer creates a new WebSocket optimizer
func NewWebSocketOptimizer(logger *zap.Logger, options WebSocketOptimizerOptions) *WebSocketOptimizer {
	return &WebSocketOptimizer{
		compressionLevel:     options.CompressionLevel,
		writeBufferSize:      options.WriteBufferSize,
		readBufferSize:       options.ReadBufferSize,
		writeDeadline:        options.WriteDeadline,
		readDeadline:         options.ReadDeadline,
		pongWait:             options.PongWait,
		pingPeriod:           options.PingPeriod,
		maxMessageSize:       options.MaxMessageSize,
		enableCompression:    options.EnableCompression,
		enableBinaryMessages: options.EnableBinaryMessages,
		batchingEnabled:      options.BatchingEnabled,
		batchSize:            options.BatchSize,
		batchInterval:        options.BatchInterval,
		logger:               logger,
		connectionPool: sync.Pool{
			New: func() interface{} {
				return &websocket.Conn{}
			},
		},
	}
}

// GetUpgrader returns a WebSocket upgrader with optimized settings
func (o *WebSocketOptimizer) GetUpgrader() *websocket.Upgrader {
	return &websocket.Upgrader{
		ReadBufferSize:    o.readBufferSize,
		WriteBufferSize:   o.writeBufferSize,
		EnableCompression: o.enableCompression,
		CheckOrigin:       func(r *http.Request) bool { return true }, // Allow all origins
	}
}

// OptimizeConnection optimizes a WebSocket connection
func (o *WebSocketOptimizer) OptimizeConnection(conn *websocket.Conn) {
	// Set read limit
	conn.SetReadLimit(o.maxMessageSize)

	// Set read deadline
	conn.SetReadDeadline(time.Now().Add(o.pongWait))

	// Set pong handler
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(o.pongWait))
		return nil
	})

	// Enable compression if requested
	if o.enableCompression {
		conn.EnableWriteCompression(true)
		conn.SetCompressionLevel(o.compressionLevel)
	}
}

// BatchedWriter provides batched writing for WebSocket connections
type BatchedWriter struct {
	conn          *websocket.Conn
	messages      []interface{}
	messageType   int
	batchSize     int
	batchInterval time.Duration
	mu            sync.Mutex
	timer         *time.Timer
	logger        *zap.Logger
}

// NewBatchedWriter creates a new batched writer
func (o *WebSocketOptimizer) NewBatchedWriter(conn *websocket.Conn) *BatchedWriter {
	writer := &BatchedWriter{
		conn:          conn,
		messages:      make([]interface{}, 0, o.batchSize),
		messageType:   websocket.TextMessage,
		batchSize:     o.batchSize,
		batchInterval: o.batchInterval,
		logger:        o.logger,
	}

	// Set message type based on configuration
	if o.enableBinaryMessages {
		writer.messageType = websocket.BinaryMessage
	}

	// Start the timer
	writer.timer = time.AfterFunc(o.batchInterval, func() {
		writer.Flush()
	})

	return writer
}

// Write writes a message to the batched writer
func (w *BatchedWriter) Write(message interface{}) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Add the message to the batch
	w.messages = append(w.messages, message)

	// Flush if the batch is full
	if len(w.messages) >= w.batchSize {
		return w.flush()
	}

	return nil
}

// Flush flushes the batched writer
func (w *BatchedWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	return w.flush()
}

// flush flushes the batched writer (must be called with lock held)
func (w *BatchedWriter) flush() error {
	if len(w.messages) == 0 {
		return nil
	}

	// Reset the timer
	w.timer.Reset(w.batchInterval)

	// Set write deadline
	w.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// Write the messages
	err := w.conn.WriteJSON(w.messages)
	if err != nil {
		w.logger.Error("Failed to write WebSocket messages",
			zap.Int("count", len(w.messages)),
			zap.Error(err))
		return err
	}

	// Clear the messages
	w.messages = w.messages[:0]

	return nil
}

// Close closes the batched writer
func (w *BatchedWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Stop the timer
	w.timer.Stop()

	// Flush any remaining messages
	return w.flush()
}

// StartPinger starts a pinger for a WebSocket connection
func (o *WebSocketOptimizer) StartPinger(conn *websocket.Conn, done chan struct{}) {
	ticker := time.NewTicker(o.pingPeriod)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			// Set write deadline
			conn.SetWriteDeadline(time.Now().Add(o.writeDeadline))

			// Send ping
			if err := conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				o.logger.Error("Failed to send ping",
					zap.Error(err))
				return
			}
		case <-done:
			return
		}
	}
}
