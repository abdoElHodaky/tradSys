package pools

import (
	"sync"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
)

// OrderPool manages a pool of reusable order objects for high-frequency trading
type OrderPool struct {
	pool sync.Pool
}

// NewOrderPool creates a new order pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &models.Order{}
			},
		},
	}
}

// Get retrieves an order from the pool
func (p *OrderPool) Get() *models.Order {
	order := p.pool.Get().(*models.Order)
	// Reset the order to ensure clean state
	p.resetOrder(order)
	return order
}

// Put returns an order to the pool
func (p *OrderPool) Put(order *models.Order) {
	if order != nil {
		p.pool.Put(order)
	}
}

// resetOrder resets an order to its zero state
func (p *OrderPool) resetOrder(order *models.Order) {
	*order = models.Order{}
}

// MessagePool manages a pool of reusable message objects
type MessagePool struct {
	pool sync.Pool
}

// Message represents a generic message structure
type Message struct {
	Type      string
	Data      []byte
	Timestamp int64
	ID        string
}

// NewMessagePool creates a new message pool
func NewMessagePool() *MessagePool {
	return &MessagePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Message{}
			},
		},
	}
}

// Get retrieves a message from the pool
func (p *MessagePool) Get() *Message {
	msg := p.pool.Get().(*Message)
	// Reset the message to ensure clean state
	p.resetMessage(msg)
	return msg
}

// Put returns a message to the pool
func (p *MessagePool) Put(msg *Message) {
	if msg != nil {
		p.pool.Put(msg)
	}
}

// resetMessage resets a message to its zero state
func (p *MessagePool) resetMessage(msg *Message) {
	msg.Type = ""
	msg.Data = nil
	msg.Timestamp = 0
	msg.ID = ""
}

// BufferPool manages a pool of reusable byte buffers
type BufferPool struct {
	pool sync.Pool
}

// NewBufferPool creates a new buffer pool with specified buffer size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, 0, size)
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
		// Reset the buffer length but keep capacity
		buf = buf[:0]
		p.pool.Put(buf)
	}
}

// Global pool instances for convenience
var (
	GlobalOrderPool   = NewOrderPool()
	GlobalMessagePool = NewMessagePool()
	GlobalBufferPool  = NewBufferPool(4096) // 4KB default buffer size
)

// GetOrder gets an order from the global pool
func GetOrder() *models.Order {
	return GlobalOrderPool.Get()
}

// PutOrder returns an order to the global pool
func PutOrder(order *models.Order) {
	GlobalOrderPool.Put(order)
}

// GetMessage gets a message from the global pool
func GetMessage() *Message {
	return GlobalMessagePool.Get()
}

// PutMessage returns a message to the global pool
func PutMessage(msg *Message) {
	GlobalMessagePool.Put(msg)
}

// GetBuffer gets a buffer from the global pool
func GetBuffer() []byte {
	return GlobalBufferPool.Get()
}

// PutBuffer returns a buffer to the global pool
func PutBuffer(buf []byte) {
	GlobalBufferPool.Put(buf)
}
