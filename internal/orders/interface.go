package orders

import (
	"context"

	"github.com/abdoElHodaky/tradSys/proto/orders"
)

// Service defines the interface for order management operations
type Service interface {
	// CreateOrder creates a new order
	CreateOrder(ctx context.Context, symbol string, orderType orders.OrderType, side orders.OrderSide, quantity, price, stopPrice float64, clientOrderID string) (*orders.OrderResponse, error)
	
	// GetOrder retrieves an order by ID
	GetOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error)
	
	// CancelOrder cancels an existing order
	CancelOrder(ctx context.Context, orderID string) (*orders.OrderResponse, error)
	
	// GetOrders retrieves a list of orders
	GetOrders(ctx context.Context, symbol string, status orders.OrderStatus, startTime, endTime int64, limit int32) ([]*orders.OrderResponse, error)
}
