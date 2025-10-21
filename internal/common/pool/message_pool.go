package pool

import (
	"sync"
	"time"
)

// PriceMessage represents a market price update message
type PriceMessage struct {
	Symbol    string  `json:"symbol"`
	Price     float64 `json:"price"`
	Volume    float64 `json:"volume"`
	Timestamp int64   `json:"timestamp"`
	Change    float64 `json:"change,omitempty"`
	ChangePercent float64 `json:"change_percent,omitempty"`
}

// Reset resets the PriceMessage to zero values
func (m *PriceMessage) Reset() {
	m.Symbol = ""
	m.Price = 0
	m.Volume = 0
	m.Timestamp = 0
	m.Change = 0
	m.ChangePercent = 0
}

// PriceMessagePool manages a pool of PriceMessage objects
type PriceMessagePool struct {
	pool sync.Pool
}

// NewPriceMessagePool creates a new price message pool
func NewPriceMessagePool() *PriceMessagePool {
	return &PriceMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &PriceMessage{}
			},
		},
	}
}

// Get retrieves a PriceMessage from the pool
func (p *PriceMessagePool) Get() *PriceMessage {
	msg := p.pool.Get().(*PriceMessage)
	msg.Reset()
	return msg
}

// Put returns a PriceMessage to the pool
func (p *PriceMessagePool) Put(msg *PriceMessage) {
	if msg != nil {
		msg.Reset()
		p.pool.Put(msg)
	}
}

// Global price message pool
var globalPriceMessagePool = NewPriceMessagePool()

// GetPriceMessageFromPool retrieves a PriceMessage from the global pool
func GetPriceMessageFromPool() *PriceMessage {
	return globalPriceMessagePool.Get()
}

// PutPriceMessageToPool returns a PriceMessage to the global pool
func PutPriceMessageToPool(msg *PriceMessage) {
	globalPriceMessagePool.Put(msg)
}

// OrderUpdateMessage represents an order status update message
type OrderUpdateMessage struct {
	OrderID        string     `json:"order_id"`
	Symbol         string     `json:"symbol"`
	Side           string     `json:"side"`
	Status         string     `json:"status"`
	FilledQuantity float64    `json:"filled_quantity"`
	AveragePrice   float64    `json:"average_price"`
	Timestamp      int64      `json:"timestamp"`
	ExecutedAt     *time.Time `json:"executed_at,omitempty"`
}

// Reset resets the OrderUpdateMessage to zero values
func (m *OrderUpdateMessage) Reset() {
	m.OrderID = ""
	m.Symbol = ""
	m.Side = ""
	m.Status = ""
	m.FilledQuantity = 0
	m.AveragePrice = 0
	m.Timestamp = 0
	m.ExecutedAt = nil
}

// OrderUpdateMessagePool manages a pool of OrderUpdateMessage objects
type OrderUpdateMessagePool struct {
	pool sync.Pool
}

// NewOrderUpdateMessagePool creates a new order update message pool
func NewOrderUpdateMessagePool() *OrderUpdateMessagePool {
	return &OrderUpdateMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderUpdateMessage{}
			},
		},
	}
}

// Get retrieves an OrderUpdateMessage from the pool
func (p *OrderUpdateMessagePool) Get() *OrderUpdateMessage {
	msg := p.pool.Get().(*OrderUpdateMessage)
	msg.Reset()
	return msg
}

// Put returns an OrderUpdateMessage to the pool
func (p *OrderUpdateMessagePool) Put(msg *OrderUpdateMessage) {
	if msg != nil {
		msg.Reset()
		p.pool.Put(msg)
	}
}

// Global order update message pool
var globalOrderUpdateMessagePool = NewOrderUpdateMessagePool()

// GetOrderUpdateMessageFromPool retrieves an OrderUpdateMessage from the global pool
func GetOrderUpdateMessageFromPool() *OrderUpdateMessage {
	return globalOrderUpdateMessagePool.Get()
}

// PutOrderUpdateMessageToPool returns an OrderUpdateMessage to the global pool
func PutOrderUpdateMessageToPool(msg *OrderUpdateMessage) {
	globalOrderUpdateMessagePool.Put(msg)
}

// WebSocketMessage represents a generic WebSocket message
type WebSocketMessage struct {
	Type      string      `json:"type"`
	Channel   string      `json:"channel,omitempty"`
	Data      interface{} `json:"data"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// Reset resets the WebSocketMessage to zero values
func (m *WebSocketMessage) Reset() {
	m.Type = ""
	m.Channel = ""
	m.Data = nil
	m.Timestamp = 0
	m.RequestID = ""
}

// WebSocketMessagePool manages a pool of WebSocketMessage objects
type WebSocketMessagePool struct {
	pool sync.Pool
}

// NewWebSocketMessagePool creates a new WebSocket message pool
func NewWebSocketMessagePool() *WebSocketMessagePool {
	return &WebSocketMessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &WebSocketMessage{}
			},
		},
	}
}

// Get retrieves a WebSocketMessage from the pool
func (p *WebSocketMessagePool) Get() *WebSocketMessage {
	msg := p.pool.Get().(*WebSocketMessage)
	msg.Reset()
	return msg
}

// Put returns a WebSocketMessage to the pool
func (p *WebSocketMessagePool) Put(msg *WebSocketMessage) {
	if msg != nil {
		msg.Reset()
		p.pool.Put(msg)
	}
}

// Global WebSocket message pool
var globalWebSocketMessagePool = NewWebSocketMessagePool()

// GetWebSocketMessageFromPool retrieves a WebSocketMessage from the global pool
func GetWebSocketMessageFromPool() *WebSocketMessage {
	return globalWebSocketMessagePool.Get()
}

// PutWebSocketMessageToPool returns a WebSocketMessage to the global pool
func PutWebSocketMessageToPool(msg *WebSocketMessage) {
	globalWebSocketMessagePool.Put(msg)
}

// BufferPool manages a pool of byte buffers for network operations
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool with specified buffer size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
	}
}

// Get retrieves a buffer from the pool
func (p *BufferPool) Get() []byte {
	return p.pool.Get().([]byte)
}

// Put returns a buffer to the pool
func (p *BufferPool) Put(buf []byte) {
	if buf != nil {
		// Clear the buffer before returning to pool
		for i := range buf {
			buf[i] = 0
		}
		p.pool.Put(buf)
	}
}

// Global buffer pools for different sizes
var (
	smallBufferPool  = NewBufferPool(1024)   // 1KB buffers
	mediumBufferPool = NewBufferPool(4096)   // 4KB buffers
	largeBufferPool  = NewBufferPool(16384)  // 16KB buffers
)

// GetSmallBuffer retrieves a 1KB buffer from the global pool
func GetSmallBuffer() []byte {
	return smallBufferPool.Get()
}

// PutSmallBuffer returns a 1KB buffer to the global pool
func PutSmallBuffer(buf []byte) {
	smallBufferPool.Put(buf)
}

// GetMediumBuffer retrieves a 4KB buffer from the global pool
func GetMediumBuffer() []byte {
	return mediumBufferPool.Get()
}

// PutMediumBuffer returns a 4KB buffer to the global pool
func PutMediumBuffer(buf []byte) {
	mediumBufferPool.Put(buf)
}

// GetLargeBuffer retrieves a 16KB buffer from the global pool
func GetLargeBuffer() []byte {
	return largeBufferPool.Get()
}

// PutLargeBuffer returns a 16KB buffer to the global pool
func PutLargeBuffer(buf []byte) {
	largeBufferPool.Put(buf)
}
