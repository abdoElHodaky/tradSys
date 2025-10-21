package pools

import (
	"sync"
	"time"

	"github.com/abdoElHodaky/tradSys/proto/orders"
	"github.com/google/uuid"
)

// OrderPool provides a pool of order responses
type OrderPool struct {
	pool sync.Pool
}

// NewOrderPool creates a new order pool
func NewOrderPool() *OrderPool {
	return &OrderPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &orders.OrderResponse{}
			},
		},
	}
}

// Get gets an order response from the pool
func (p *OrderPool) Get() *orders.OrderResponse {
	return p.pool.Get().(*orders.OrderResponse)
}

// Put puts an order response back into the pool
func (p *OrderPool) Put(order *orders.OrderResponse) {
	// Reset the order
	order.Id = ""
	order.UserId = ""
	order.AccountId = ""
	order.Symbol = ""
	order.Side = orders.OrderSide_BUY
	order.Type = orders.OrderType_MARKET
	order.Quantity = 0
	order.Price = 0
	order.StopPrice = 0
	order.TrailingOffset = 0
	order.TimeInForce = orders.TimeInForce_GTC
	order.Status = orders.OrderStatus_NEW
	order.FilledQty = 0
	order.AvgPrice = 0
	order.ClientOrderId = ""
	order.ExchangeOrderId = ""
	order.StopLoss = 0
	order.TakeProfit = 0
	order.Notes = ""
	order.CreatedAt = 0
	order.UpdatedAt = 0
	order.ExpiresAt = 0

	// Put the order back into the pool
	p.pool.Put(order)
}

// NewOrderResponse creates a new order response
func (p *OrderPool) NewOrderResponse(
	userID string,
	accountID string,
	symbol string,
	side orders.OrderSide,
	orderType orders.OrderType,
	quantity float64,
	price float64,
	stopPrice float64,
	trailingOffset float64,
	timeInForce orders.TimeInForce,
	clientOrderID string,
) *orders.OrderResponse {
	order := p.Get()
	order.Id = uuid.New().String()
	order.UserId = userID
	order.AccountId = accountID
	order.Symbol = symbol
	order.Side = side
	order.Type = orderType
	order.Quantity = quantity
	order.Price = price
	order.StopPrice = stopPrice
	order.TrailingOffset = trailingOffset
	order.TimeInForce = timeInForce
	order.Status = orders.OrderStatus_NEW
	order.FilledQty = 0
	order.AvgPrice = 0
	order.ClientOrderId = clientOrderID
	order.ExchangeOrderId = ""
	order.CreatedAt = time.Now().UnixNano() / int64(time.Millisecond)
	order.UpdatedAt = order.CreatedAt
	return order
}

