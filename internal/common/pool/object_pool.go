package pool

import (
	"sync"
)

// ObjectPool provides a generic object pool implementation
type ObjectPool struct {
	pool    sync.Pool
	factory func() interface{}
}

// NewObjectPool creates a new object pool with the given factory function
func NewObjectPool(factory func() interface{}, initialSize int) *ObjectPool {
	p := &ObjectPool{
		factory: factory,
	}
	
	p.pool.New = factory
	
	// Pre-populate the pool
	for i := 0; i < initialSize; i++ {
		p.pool.Put(factory())
	}
	
	return p
}

// Get retrieves an object from the pool
func (p *ObjectPool) Get() interface{} {
	return p.pool.Get()
}

// Put returns an object to the pool
func (p *ObjectPool) Put(obj interface{}) {
	p.pool.Put(obj)
}

// Size returns the approximate size of the pool (not thread-safe)
func (p *ObjectPool) Size() int {
	// sync.Pool doesn't provide a size method, so we return -1
	// to indicate that size is not available
	return -1
}
