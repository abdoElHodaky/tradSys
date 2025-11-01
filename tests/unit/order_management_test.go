package unit

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOrderService_CreateOrder(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Test valid order creation
	orderReq := &orders.CreateOrderRequest{
		UserID:        "user-001",
		ClientOrderID: "client-001",
		Symbol:        "AAPL",
		Side:          orders.OrderSideBuy,
		Type:          orders.OrderTypeLimit,
		Quantity:      100,
		Price:         150.50,
		TimeInForce:   orders.TimeInForceGTC,
	}

	order, err := service.CreateOrder(ctx, orderReq)
	require.NoError(t, err)
	assert.NotNil(t, order)

	// Verify order fields
	assert.NotEmpty(t, order.ID)
	assert.Equal(t, "user-001", order.UserID)
	assert.Equal(t, "client-001", order.ClientOrderID)
	assert.Equal(t, "AAPL", order.Symbol)
	assert.Equal(t, orders.OrderSideBuy, order.Side)
	assert.Equal(t, orders.OrderTypeLimit, order.Type)
	assert.Equal(t, float64(100), order.Quantity)
	assert.Equal(t, 150.50, order.Price)
	assert.Equal(t, orders.OrderStatusNew, order.Status)
	assert.WithinDuration(t, time.Now(), order.CreatedAt, time.Minute)
}

func TestOrderService_ValidateOrder(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Test invalid order - missing required fields
	invalidOrder := &orders.CreateOrderRequest{
		UserID: "user-001",
		// Missing Symbol, Side, Type, Quantity
	}

	_, err := service.CreateOrder(ctx, invalidOrder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "symbol is required")

	// Test invalid order - negative quantity
	negativeQtyOrder := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeLimit,
		Quantity: -100, // Invalid
		Price:    150.50,
	}

	_, err = service.CreateOrder(ctx, negativeQtyOrder)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quantity must be positive")

	// Test invalid order - market order with price
	marketOrderWithPrice := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeMarket,
		Quantity: 100,
		Price:    150.50, // Invalid for market order
	}

	_, err = service.CreateOrder(ctx, marketOrderWithPrice)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "market orders cannot have price")

	// Test invalid order - limit order without price
	limitOrderWithoutPrice := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeLimit,
		Quantity: 100,
		// Missing Price
	}

	_, err = service.CreateOrder(ctx, limitOrderWithoutPrice)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "limit orders must have price")
}

func TestOrderService_UpdateOrder(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Create initial order
	orderReq := &orders.CreateOrderRequest{
		UserID:        "user-001",
		ClientOrderID: "client-001",
		Symbol:        "AAPL",
		Side:          orders.OrderSideBuy,
		Type:          orders.OrderTypeLimit,
		Quantity:      100,
		Price:         150.50,
		TimeInForce:   orders.TimeInForceGTC,
	}

	order, err := service.CreateOrder(ctx, orderReq)
	require.NoError(t, err)

	// Update order
	updateReq := &orders.UpdateOrderRequest{
		OrderID:  order.ID,
		UserID:   "user-001",
		Quantity: 200,
		Price:    151.00,
	}

	updatedOrder, err := service.UpdateOrder(ctx, updateReq)
	require.NoError(t, err)
	assert.NotNil(t, updatedOrder)

	// Verify updates
	assert.Equal(t, float64(200), updatedOrder.Quantity)
	assert.Equal(t, 151.00, updatedOrder.Price)
	assert.Equal(t, orders.OrderStatusPending, updatedOrder.Status) // Updated status constant
	assert.After(t, updatedOrder.UpdatedAt, order.UpdatedAt)
}

func TestOrderService_CancelOrder(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Create order
	orderReq := &orders.CreateOrderRequest{
		UserID:        "user-001",
		ClientOrderID: "client-001",
		Symbol:        "AAPL",
		Side:          orders.OrderSideBuy,
		Type:          orders.OrderTypeLimit,
		Quantity:      100,
		Price:         150.50,
		TimeInForce:   orders.TimeInForceGTC,
	}

	order, err := service.CreateOrder(ctx, orderReq)
	require.NoError(t, err)

	// Cancel order
	cancelReq := &orders.CancelOrderRequest{
		OrderID: order.ID,
		UserID:  "user-001",
		Reason:  "User requested cancellation",
	}

	cancelledOrder, err := service.CancelOrder(ctx, cancelReq)
	require.NoError(t, err)
	assert.NotNil(t, cancelledOrder)

	// Verify cancellation
	assert.Equal(t, orders.OrderStatusCancelled, cancelledOrder.Status)
	assert.Equal(t, "User requested cancellation", cancelledOrder.CancelReason)
	assert.After(t, cancelledOrder.UpdatedAt, order.UpdatedAt)
}

func TestOrderService_GetOrder(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Create order
	orderReq := &orders.CreateOrderRequest{
		UserID:        "user-001",
		ClientOrderID: "client-001",
		Symbol:        "AAPL",
		Side:          orders.OrderSideBuy,
		Type:          orders.OrderTypeLimit,
		Quantity:      100,
		Price:         150.50,
		TimeInForce:   orders.TimeInForceGTC,
	}

	createdOrder, err := service.CreateOrder(ctx, orderReq)
	require.NoError(t, err)

	// Get order by ID
	retrievedOrder, err := service.GetOrder(ctx, createdOrder.ID, "user-001")
	require.NoError(t, err)
	assert.NotNil(t, retrievedOrder)

	// Verify order details
	assert.Equal(t, createdOrder.ID, retrievedOrder.ID)
	assert.Equal(t, createdOrder.UserID, retrievedOrder.UserID)
	assert.Equal(t, createdOrder.Symbol, retrievedOrder.Symbol)
	assert.Equal(t, createdOrder.Quantity, retrievedOrder.Quantity)
	assert.Equal(t, createdOrder.Price, retrievedOrder.Price)

	// Test get non-existent order
	_, err = service.GetOrder(ctx, "non-existent-id", "user-001")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "order not found")

	// Test get order with wrong user
	_, err = service.GetOrder(ctx, createdOrder.ID, "wrong-user")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unauthorized")
}

func TestOrderService_ListOrders(t *testing.T) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	// Create multiple orders
	symbols := []string{"AAPL", "GOOGL", "MSFT"}
	createdOrders := make([]*orders.Order, 0, len(symbols))

	for _, symbol := range symbols {
		orderReq := &orders.CreateOrderRequest{
			UserID:        "user-001",
			ClientOrderID: "client-" + symbol,
			Symbol:        symbol,
			Side:          orders.OrderSideBuy,
			Type:          orders.OrderTypeLimit,
			Quantity:      100,
			Price:         150.50,
			TimeInForce:   orders.TimeInForceGTC,
		}

		order, err := service.CreateOrder(ctx, orderReq)
		require.NoError(t, err)
		createdOrders = append(createdOrders, order)
	}

	// List all orders for user
	listReq := &orders.ListOrdersRequest{
		UserID: "user-001",
		Limit:  10,
		Offset: 0,
	}

	orderList, err := service.ListOrders(ctx, listReq)
	require.NoError(t, err)
	assert.NotNil(t, orderList)
	assert.Len(t, orderList.Orders, 3)
	assert.Equal(t, int64(3), orderList.Total)

	// List orders with symbol filter
	listReqFiltered := &orders.ListOrdersRequest{
		UserID: "user-001",
		Symbol: "AAPL",
		Limit:  10,
		Offset: 0,
	}

	filteredList, err := service.ListOrders(ctx, listReqFiltered)
	require.NoError(t, err)
	assert.NotNil(t, filteredList)
	assert.Len(t, filteredList.Orders, 1)
	assert.Equal(t, "AAPL", filteredList.Orders[0].Symbol)

	// List orders with pagination
	listReqPaginated := &orders.ListOrdersRequest{
		UserID: "user-001",
		Limit:  2,
		Offset: 0,
	}

	paginatedList, err := service.ListOrders(ctx, listReqPaginated)
	require.NoError(t, err)
	assert.NotNil(t, paginatedList)
	assert.Len(t, paginatedList.Orders, 2)
	assert.Equal(t, int64(3), paginatedList.Total)
}

func TestOrderLifecycle_StateTransitions(t *testing.T) {
	lifecycle := orders.NewLifecycle(nil) // Updated constructor signature

	ctx := context.Background()

	// Create order in NEW state
	order := &orders.Order{
		ID:     "order-001",
		UserID: "user-001",
		Symbol: "AAPL",
		Status: orders.OrderStatusNew,
	}

	// Test valid transition: NEW -> PENDING
	err := lifecycle.TransitionTo(ctx, order, orders.OrderStatusPending, "Order submitted to exchange")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusPending, order.Status)

	// Test valid transition: PENDING -> PARTIALLY_FILLED
	err = lifecycle.TransitionTo(ctx, order, orders.OrderStatusPartiallyFilled, "Partial execution")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusPartiallyFilled, order.Status)

	// Test valid transition: PARTIALLY_FILLED -> FILLED
	err = lifecycle.TransitionTo(ctx, order, orders.OrderStatusFilled, "Order fully executed")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusFilled, order.Status)

	// Test invalid transition: FILLED -> PENDING (should fail)
	err = lifecycle.TransitionTo(ctx, order, orders.OrderStatusPending, "Invalid transition")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid state transition")
	assert.Equal(t, orders.OrderStatusFilled, order.Status) // Status should remain unchanged
}

func TestOrderLifecycle_CancellationStates(t *testing.T) {
	lifecycle := orders.NewLifecycle(nil) // Updated constructor signature

	ctx := context.Background()

	// Test cancellation from NEW state
	newOrder := &orders.Order{
		ID:     "order-001",
		UserID: "user-001",
		Symbol: "AAPL",
		Status: orders.OrderStatusNew,
	}

	err := lifecycle.TransitionTo(ctx, newOrder, orders.OrderStatusCancelled, "User cancellation")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusCancelled, newOrder.Status)

	// Test cancellation from PENDING state
	pendingOrder := &orders.Order{
		ID:     "order-002",
		UserID: "user-001",
		Symbol: "AAPL",
		Status: orders.OrderStatusPending,
	}

	err = lifecycle.TransitionTo(ctx, pendingOrder, orders.OrderStatusCancelled, "User cancellation")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusCancelled, pendingOrder.Status)

	// Test cancellation from PARTIALLY_FILLED state
	partialOrder := &orders.Order{
		ID:     "order-003",
		UserID: "user-001",
		Symbol: "AAPL",
		Status: orders.OrderStatusPartiallyFilled,
	}

	err = lifecycle.TransitionTo(ctx, partialOrder, orders.OrderStatusCancelled, "User cancellation")
	require.NoError(t, err)
	assert.Equal(t, orders.OrderStatusCancelled, partialOrder.Status)

	// Test invalid cancellation from FILLED state
	filledOrder := &orders.Order{
		ID:     "order-004",
		UserID: "user-001",
		Symbol: "AAPL",
		Status: orders.OrderStatusFilled,
	}

	err = lifecycle.TransitionTo(ctx, filledOrder, orders.OrderStatusCancelled, "Invalid cancellation")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cannot cancel filled order")
}

func TestOrderValidator_BusinessRules(t *testing.T) {
	validator := orders.NewValidator(nil) // Updated constructor signature

	ctx := context.Background()

	// Test valid order
	validOrder := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeLimit,
		Quantity: 100,
		Price:    150.50,
	}

	violations, err := validator.ValidateOrder(ctx, validOrder)
	require.NoError(t, err)
	assert.Empty(t, violations)

	// Test order exceeding value limit
	highValueOrder := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeLimit,
		Quantity: 10000,
		Price:    150.50, // 10000 * 150.50 = 1,505,000 > 1,000,000 limit
	}

	violations, err = validator.ValidateOrder(ctx, highValueOrder)
	require.NoError(t, err)
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations[0].Message, "order value exceeds maximum")

	// Test order with invalid price
	invalidPriceOrder := &orders.CreateOrderRequest{
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Type:     orders.OrderTypeLimit,
		Quantity: 100,
		Price:    0.005, // Below minimum price
	}

	violations, err = validator.ValidateOrder(ctx, invalidPriceOrder)
	require.NoError(t, err)
	assert.NotEmpty(t, violations)
	assert.Contains(t, violations[0].Message, "price below minimum")
}

func BenchmarkOrderService_CreateOrder(b *testing.B) {
	service := orders.NewService(nil, nil) // Updated constructor signature

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		orderReq := &orders.CreateOrderRequest{
			UserID:        "user-001",
			ClientOrderID: "client-" + string(rune(i)),
			Symbol:        "AAPL",
			Side:          orders.OrderSideBuy,
			Type:          orders.OrderTypeLimit,
			Quantity:      100,
			Price:         150.50 + float64(i%100)*0.01,
			TimeInForce:   orders.TimeInForceGTC,
		}

		_, err := service.CreateOrder(ctx, orderReq)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkOrderLifecycle_StateTransition(b *testing.B) {
	lifecycle := orders.NewLifecycle(nil) // Updated constructor signature

	ctx := context.Background()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		order := &orders.Order{
			ID:     "order-" + string(rune(i)),
			UserID: "user-001",
			Symbol: "AAPL",
			Status: orders.OrderStatusNew,
		}

		err := lifecycle.TransitionTo(ctx, order, orders.OrderStatusPending, "Benchmark transition")
		if err != nil {
			b.Fatal(err)
		}
	}
}
