package pools

import (
	"sync"
)

// BufferPool provides a pool of byte slices
// to reduce garbage collection pressure in high-frequency scenarios
type BufferPool struct {
	pool sync.Pool
	size int
}

// NewBufferPool creates a new buffer pool with the specified buffer size
func NewBufferPool(size int) *BufferPool {
	return &BufferPool{
		pool: sync.Pool{
			New: func() interface{} {
				return make([]byte, size)
			},
		},
		size: size,
	}
}

// Get retrieves a byte slice from the pool
func (p *BufferPool) Get() []byte {
	buf := p.pool.Get().([]byte)
	// Ensure the buffer is the correct size and zeroed
	if len(buf) != p.size {
		buf = make([]byte, p.size)
	} else {
		// Zero out the buffer to prevent data leakage
		for i := range buf {
			buf[i] = 0
		}
	}
	return buf
}

// Put returns a byte slice to the pool
func (p *BufferPool) Put(buf []byte) {
	if buf == nil || len(buf) != p.size {
		// Don't put back nil or wrong-sized buffers
		return
	}
	p.pool.Put(buf)
}
