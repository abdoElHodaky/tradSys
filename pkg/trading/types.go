// Package trading provides shared type definitions for order operations
// This package mirrors internal/trading/types for use by services/* packages
package trading

import (
	"errors"
	"sync"
	"time"

	"github.com/patrickmn/go-cache"
)

// OrderSide represents the side of an order
type OrderSide string

const (
	OrderSideBuy  OrderSide = "buy"
	OrderSideSell OrderSide = "sell"
)

// OrderType represents the type of order
type OrderType string

const (
	OrderTypeLimit     OrderType = "limit"
	OrderTypeMarket    OrderType = "market"
	OrderTypeStop      OrderType = "stop"
	OrderTypeStopLimit OrderType = "stop_limit"
	OrderTypeStopMarket OrderType = "stop_market"
)

// OrderStatus represents the status of an order
type OrderStatus string

const (
	OrderStatusNew             OrderStatus = "new"
	OrderStatusPending         OrderStatus = "pending"
	OrderStatusPartiallyFilled OrderStatus = "partially_filled"
	OrderStatusFilled          OrderStatus = "filled"
	OrderStatusCancelled       OrderStatus = "cancelled"
	OrderStatusRejected        OrderStatus = "rejected"
	OrderStatusExpired         OrderStatus = "expired"
)

// AssetType represents different types of financial assets
type AssetType string

const (
	AssetTypeStock      AssetType = "STOCK"
	AssetTypeREIT       AssetType = "REIT"
	AssetTypeMutualFund AssetType = "MUTUAL_FUND"
	AssetTypeETF        AssetType = "ETF"
	AssetTypeBond       AssetType = "BOND"
	AssetTypeCrypto     AssetType = "CRYPTO"
	AssetTypeForex      AssetType = "FOREX"
	AssetTypeCommodity  AssetType = "COMMODITY"
	// Islamic Assets
	AssetTypeSukuk        AssetType = "SUKUK"
	AssetTypeIslamicFund  AssetType = "ISLAMIC_FUND"
	AssetTypeShariaStock  AssetType = "SHARIA_STOCK"
	AssetTypeIslamicETF   AssetType = "ISLAMIC_ETF"
	AssetTypeIslamicREIT  AssetType = "ISLAMIC_REIT"
	AssetTypeTakful       AssetType = "TAKAFUL"
)

// Order represents an order in the trading system
type Order struct {
	ID              string
	Symbol          string
	AssetType       AssetType
	Side            OrderSide
	Type            OrderType
	Price           float64
	Quantity        float64
	FilledQuantity  float64
	Status          OrderStatus
	CreatedAt       time.Time
	UpdatedAt       time.Time
	ClientOrderID   string
	UserID          string
	StopPrice       float64
	TimeInForce     string
	DisplayQuantity float64
	Tags            map[string]string
}

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
		return errors.New("invalid symbol")
	}
	if o.Quantity <= 0 {
		return errors.New("invalid quantity")
	}
	if o.Price <= 0 && (o.Type == OrderTypeLimit || o.Type == OrderTypeStopLimit) {
		return errors.New("invalid price")
	}
	return nil
}

// IsIceberg returns true if this is an iceberg order
func (o *Order) IsIceberg() bool {
	return o.DisplayQuantity > 0 && o.DisplayQuantity < o.Quantity
}

// RemainingQuantity returns the remaining quantity to be filled
func (o *Order) RemainingQuantity() float64 {
	return o.Quantity - o.FilledQuantity
}

// IsFilled returns true if the order is completely filled
func (o *Order) IsFilled() bool {
	return o.FilledQuantity >= o.Quantity
}

// IsPartiallyFilled returns true if the order is partially filled
func (o *Order) IsPartiallyFilled() bool {
	return o.FilledQuantity > 0 && o.FilledQuantity < o.Quantity
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
	b.order.CreatedAt = expiresAt
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
		CreatedAt: bo.CreatedAt,
	}
}

func (bo *BaseOrder) GetOrderID() string { return bo.ID }

func (bo *BaseOrder) Validate() error {
	if bo.ID == "" {
		return errors.New("base order ID is required")
	}
	if bo.Symbol == "" {
		return errors.New("invalid symbol")
	}
	if bo.Quantity <= 0 {
		return errors.New("invalid quantity")
	}
	return nil
}