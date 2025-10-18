package pools

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/db/models"
)

// OrderPool manages a pool of Order objects to reduce GC pressure
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

// Get retrieves an Order from the pool
func (p *OrderPool) Get() *models.Order {
	order := p.pool.Get().(*models.Order)
	// Reset the order to ensure clean state
	order.Reset()
	return order
}

// Put returns an Order to the pool
func (p *OrderPool) Put(order *models.Order) {
	if order != nil {
		order.Reset()
		p.pool.Put(order)
	}
}

// Global order pool instance
var globalOrderPool = NewOrderPool()

// GetOrderFromPool retrieves an Order from the global pool
func GetOrderFromPool() *models.Order {
	return globalOrderPool.Get()
}

// PutOrderToPool returns an Order to the global pool
func PutOrderToPool(order *models.Order) {
	globalOrderPool.Put(order)
}

// Reset method for Order struct - extends the existing model
func (o *models.Order) Reset() {
	o.ID = ""
	o.UserID = ""
	o.Symbol = ""
	o.Side = ""
	o.Type = ""
	o.Quantity = 0
	o.Price = 0
	o.StopPrice = 0
	o.Status = ""
	o.FilledQuantity = 0
	o.AveragePrice = 0
	o.Commission = 0
	o.CreatedAt = time.Time{}
	o.UpdatedAt = time.Time{}
	o.ExecutedAt = nil
}

// OrderRequest represents a pooled order request
type OrderRequest struct {
	Symbol      string  `json:"symbol" binding:"required"`
	Side        string  `json:"side" binding:"required,oneof=buy sell"`
	Type        string  `json:"type" binding:"required,oneof=market limit stop"`
	Quantity    float64 `json:"quantity" binding:"required,gt=0"`
	Price       float64 `json:"price,omitempty"`
	StopPrice   float64 `json:"stop_price,omitempty"`
	TimeInForce string  `json:"time_in_force,omitempty"`
}

// Reset resets the OrderRequest to zero values
func (r *OrderRequest) Reset() {
	r.Symbol = ""
	r.Side = ""
	r.Type = ""
	r.Quantity = 0
	r.Price = 0
	r.StopPrice = 0
	r.TimeInForce = ""
}

// OrderRequestPool manages a pool of OrderRequest objects
type OrderRequestPool struct {
	pool sync.Pool
}

// NewOrderRequestPool creates a new order request pool
func NewOrderRequestPool() *OrderRequestPool {
	return &OrderRequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderRequest{}
			},
		},
	}
}

// Get retrieves an OrderRequest from the pool
func (p *OrderRequestPool) Get() *OrderRequest {
	req := p.pool.Get().(*OrderRequest)
	req.Reset()
	return req
}

// Put returns an OrderRequest to the pool
func (p *OrderRequestPool) Put(req *OrderRequest) {
	if req != nil {
		req.Reset()
		p.pool.Put(req)
	}
}

// Global order request pool
var globalOrderRequestPool = NewOrderRequestPool()

// GetOrderRequestFromPool retrieves an OrderRequest from the global pool
func GetOrderRequestFromPool() *OrderRequest {
	return globalOrderRequestPool.Get()
}

// PutOrderRequestToPool returns an OrderRequest to the global pool
func PutOrderRequestToPool(req *OrderRequest) {
	globalOrderRequestPool.Put(req)
}

// OrderResponse represents a pooled order response
type OrderResponse struct {
	ID              string     `json:"id"`
	Symbol          string     `json:"symbol"`
	Side            string     `json:"side"`
	Type            string     `json:"type"`
	Quantity        float64    `json:"quantity"`
	Price           float64    `json:"price,omitempty"`
	StopPrice       float64    `json:"stop_price,omitempty"`
	Status          string     `json:"status"`
	FilledQuantity  float64    `json:"filled_quantity"`
	AveragePrice    float64    `json:"average_price"`
	Commission      float64    `json:"commission"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
	ExecutedAt      *time.Time `json:"executed_at,omitempty"`
	ProcessingTime  int64      `json:"processing_time_ns,omitempty"` // For latency tracking
}

// Reset resets the OrderResponse to zero values
func (r *OrderResponse) Reset() {
	r.ID = ""
	r.Symbol = ""
	r.Side = ""
	r.Type = ""
	r.Quantity = 0
	r.Price = 0
	r.StopPrice = 0
	r.Status = ""
	r.FilledQuantity = 0
	r.AveragePrice = 0
	r.Commission = 0
	r.CreatedAt = time.Time{}
	r.UpdatedAt = time.Time{}
	r.ExecutedAt = nil
	r.ProcessingTime = 0
}

// FromOrder populates OrderResponse from Order model
func (r *OrderResponse) FromOrder(order *models.Order) {
	r.ID = order.ID
	r.Symbol = order.Symbol
	r.Side = order.Side
	r.Type = order.Type
	r.Quantity = order.Quantity
	r.Price = order.Price
	r.StopPrice = order.StopPrice
	r.Status = order.Status
	r.FilledQuantity = order.FilledQuantity
	r.AveragePrice = order.AveragePrice
	r.Commission = order.Commission
	r.CreatedAt = order.CreatedAt
	r.UpdatedAt = order.UpdatedAt
	r.ExecutedAt = order.ExecutedAt
}

// OrderResponsePool manages a pool of OrderResponse objects
type OrderResponsePool struct {
	pool sync.Pool
}

// NewOrderResponsePool creates a new order response pool
func NewOrderResponsePool() *OrderResponsePool {
	return &OrderResponsePool{
		pool: sync.Pool{
			New: func() interface{} {
				return &OrderResponse{}
			},
		},
	}
}

// Get retrieves an OrderResponse from the pool
func (p *OrderResponsePool) Get() *OrderResponse {
	resp := p.pool.Get().(*OrderResponse)
	resp.Reset()
	return resp
}

// Put returns an OrderResponse to the pool
func (p *OrderResponsePool) Put(resp *OrderResponse) {
	if resp != nil {
		resp.Reset()
		p.pool.Put(resp)
	}
}

// Global order response pool
var globalOrderResponsePool = NewOrderResponsePool()

// GetOrderResponseFromPool retrieves an OrderResponse from the global pool
func GetOrderResponseFromPool() *OrderResponse {
	return globalOrderResponsePool.Get()
}

// PutOrderResponseToPool returns an OrderResponse to the global pool
func PutOrderResponseToPool(resp *OrderResponse) {
	globalOrderResponsePool.Put(resp)
}
