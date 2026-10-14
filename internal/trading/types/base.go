package types

import (
	"errors"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// Error definitions for base types - used across packages for DRY principle
var (
	ErrInvalidSymbol   = errors.New("invalid symbol")
	ErrInvalidQuantity = errors.New("invalid quantity")
	ErrInvalidPrice    = errors.New("invalid price")
)

// OrderCarrier interface for polymorphic order operations (SOLID: Interface Segregation)
type OrderCarrier interface {
	GetOrder() *Order
	GetOrderID() string
	Validate() error
}

// GetOrder returns the Order itself (implements OrderCarrier for Order)
func (o *Order) GetOrder() *Order { return o }

// GetOrderID returns the order's ID
func (o *Order) GetOrderID() string { return o.ID }

// Validate validates the order's fields
func (o *Order) Validate() error {
	if o.ID == "" {
		return errors.New("order ID is required")
	}
	if o.Symbol == "" {
		return ErrInvalidSymbol
	}
	if o.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	if o.Price <= 0 && (o.Type == OrderTypeLimit || o.Type == OrderTypeStopLimit) {
		return ErrInvalidPrice
	}
	return nil
}

// OrderBuilder provides fluent interface for order creation (KISS: Simple, clear API)
type OrderBuilder struct {
	order *Order
}

// NewOrderBuilder creates a new OrderBuilder
func NewOrderBuilder() *OrderBuilder {
	return &OrderBuilder{
		order: &Order{
			CreatedAt: time.Now(),
			Status:    OrderStatusNew,
		},
	}
}

func (b *OrderBuilder) WithID(id string) *OrderBuilder {
	b.order.ID = id
	return b
}

func (b *OrderBuilder) WithSymbol(s string) *OrderBuilder {
	b.order.Symbol = s
	return b
}

func (b *OrderBuilder) WithSide(side OrderSide) *OrderBuilder {
	b.order.Side = side
	return b
}

func (b *OrderBuilder) WithType(t OrderType) *OrderBuilder {
	b.order.Type = t
	return b
}

func (b *OrderBuilder) WithPrice(p float64) *OrderBuilder {
	b.order.Price = p
	return b
}

func (b *OrderBuilder) WithQuantity(q float64) *OrderBuilder {
	b.order.Quantity = q
	return b
}

func (b *OrderBuilder) WithAssetType(at AssetType) *OrderBuilder {
	b.order.AssetType = at
	return b
}

func (b *OrderBuilder) WithClientOrderID(clientOrderID string) *OrderBuilder {
	b.order.ClientOrderID = clientOrderID
	return b
}

func (b *OrderBuilder) WithUserID(userID string) *OrderBuilder {
	b.order.UserID = userID
	return b
}

func (b *OrderBuilder) WithStopPrice(stopPrice float64) *OrderBuilder {
	b.order.StopPrice = stopPrice
	return b
}

func (b *OrderBuilder) WithTimeInForce(tif string) *OrderBuilder {
	b.order.TimeInForce = tif
	return b
}

func (b *OrderBuilder) WithExpiresAt(expiresAt time.Time) *OrderBuilder {
	b.order.ExpireTime = expiresAt
	return b
}

func (b *OrderBuilder) WithTags(tags map[string]string) *OrderBuilder {
	b.order.Tags = tags
	return b
}

func (b *OrderBuilder) Build() *Order {
	return b.order
}

// OrderCache provides thread-safe caching (Encapsulation pattern - KISS)
type OrderCache struct {
	mu     sync.RWMutex
	orders map[string]*Order
	cache  *cache.Cache
}

// NewOrderCache creates a new OrderCache
func NewOrderCache(defaultExpiration, cleanupInterval time.Duration) *OrderCache {
	return &OrderCache{
		orders: make(map[string]*Order),
		cache:  cache.New(defaultExpiration, cleanupInterval),
	}
}

func (oc *OrderCache) Get(orderID string) (*Order, bool) {
	oc.mu.RLock()
	defer oc.mu.RUnlock()
	order, found := oc.orders[orderID]
	return order, found
}

func (oc *OrderCache) Set(order *Order) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	if order != nil {
		oc.orders[order.ID] = order
		oc.cache.Set(order.ID, order, cache.DefaultExpiration)
	}
}

func (oc *OrderCache) Delete(orderID string) {
	oc.mu.Lock()
	defer oc.mu.Unlock()
	delete(oc.orders, orderID)
	oc.cache.Delete(orderID)
}

// BaseOrder provides foundational order operations without inheriting from Order
// This is useful for creating simple carrier implementations (KISS: Simple solutions)
type BaseOrder struct {
	ID        string
	Symbol    string
	Side      OrderSide
	Type      OrderType
	Price     float64
	Quantity  float64
	CreatedAt time.Time
}

// NewBaseOrder creates a new BaseOrder
func NewBaseOrder() *BaseOrder {
	return &BaseOrder{
		CreatedAt: time.Now(),
	}
}

// GetOrder creates a minimal Order from BaseOrder data
func (bo *BaseOrder) GetOrder() *Order {
	return &Order{
		ID:       bo.ID,
		Symbol:   bo.Symbol,
		Side:     bo.Side,
		Type:     bo.Type,
		Price:    bo.Price,
		Quantity: bo.Quantity,
	}
}

func (bo *BaseOrder) GetOrderID() string { return bo.ID }

func (bo *BaseOrder) Validate() error {
	if bo.ID == "" {
		return errors.New("base order ID is required")
	}
	if bo.Symbol == "" {
		return ErrInvalidSymbol
	}
	if bo.Quantity <= 0 {
		return ErrInvalidQuantity
	}
	return nil
}