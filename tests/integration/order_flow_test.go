package integration

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/orders"
	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/abdoElHodaky/tradSys/pkg/matching"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type OrderFlowTestSuite struct {
	suite.Suite
	orderService   *orders.Service
	riskCalculator *risk.Calculator
	matchingEngine *matching.Engine
	ctx            context.Context
}

func (suite *OrderFlowTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Initialize order service
	suite.orderService = orders.NewService(&orders.Config{
		MaxOrdersPerUser: 1000,
		MaxOrderValue:    1000000,
		EnableRiskChecks: true,
		EnableCompliance: true,
		OrderTimeout:     30 * time.Minute,
	})

	// Initialize risk calculator
	suite.riskCalculator = risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	// Initialize matching engine
	suite.matchingEngine = matching.NewEngine(&matching.Config{
		Symbol:           "AAPL",
		LatencyTargetNS:  100000,
		MaxOrdersPerSec:  100000,
		OrderBookDepth:   1000,
		EnableHFTMode:    true,
	})
}

func (suite *OrderFlowTestSuite) TestCompleteOrderFlow() {
	// Test complete order flow: Create -> Risk Check -> Match -> Execute

	// Step 1: Create buy order
	buyOrderReq := &orders.CreateOrderRequest{
		UserID:      "user-001",
		ClientOrderID: "buy-001",
		Symbol:      "AAPL",
		Side:        orders.SideBuy,
		Type:        orders.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: orders.TimeInForceGTC,
	}

	buyOrder, err := suite.orderService.CreateOrder(suite.ctx, buyOrderReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), orders.StatusNew, buyOrder.Status)

	// Step 2: Risk check for buy order
	portfolio := &risk.Portfolio{
		UserID: "user-001",
		Positions: []risk.Position{
			{
				Symbol:        "AAPL",
				Quantity:      500,
				AveragePrice:  145.00,
				CurrentPrice:  150.00,
				MarketValue:   75000,
				UnrealizedPnL: 2500,
			},
		},
		TotalMarketValue:   75000,
		TotalUnrealizedPnL: 2500,
		Cash:              425000,
		TotalValue:        500000,
	}

	orderRisk := &risk.OrderRisk{
		UserID:    "user-001",
		Symbol:    "AAPL",
		Side:      "buy",
		Quantity:  100,
		Price:     150.50,
		OrderType: "limit",
	}

	riskResult, err := suite.riskCalculator.CalculateOrderRisk(suite.ctx, orderRisk, portfolio)
	require.NoError(suite.T(), err)
	assert.True(suite.T(), riskResult.IsAcceptable, "Buy order should pass risk checks")

	// Step 3: Submit to matching engine
	matchingOrder := &matching.Order{
		ID:          buyOrder.ID,
		UserID:      buyOrder.UserID,
		Symbol:      buyOrder.Symbol,
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    buyOrder.Quantity,
		Price:       buyOrder.Price,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   buyOrder.CreatedAt,
	}

	trades, err := suite.matchingEngine.ProcessOrder(suite.ctx, matchingOrder)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), trades, "No trades expected for single buy order")

	// Step 4: Create matching sell order
	sellOrderReq := &orders.CreateOrderRequest{
		UserID:      "user-002",
		ClientOrderID: "sell-001",
		Symbol:      "AAPL",
		Side:        orders.SideSell,
		Type:        orders.TypeLimit,
		Quantity:    100,
		Price:       150.50,
		TimeInForce: orders.TimeInForceGTC,
	}

	sellOrder, err := suite.orderService.CreateOrder(suite.ctx, sellOrderReq)
	require.NoError(suite.T(), err)

	// Step 5: Submit sell order to matching engine
	sellMatchingOrder := &matching.Order{
		ID:          sellOrder.ID,
		UserID:      sellOrder.UserID,
		Symbol:      sellOrder.Symbol,
		Side:        matching.SideSell,
		Type:        matching.TypeLimit,
		Quantity:    sellOrder.Quantity,
		Price:       sellOrder.Price,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   sellOrder.CreatedAt,
	}

	trades, err = suite.matchingEngine.ProcessOrder(suite.ctx, sellMatchingOrder)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), trades, 1, "Should generate one trade")

	// Step 6: Verify trade details
	trade := trades[0]
	assert.Equal(suite.T(), buyOrder.ID, trade.BuyOrderID)
	assert.Equal(suite.T(), sellOrder.ID, trade.SellOrderID)
	assert.Equal(suite.T(), float64(100), trade.Quantity)
	assert.Equal(suite.T(), 150.50, trade.Price)
	assert.WithinDuration(suite.T(), time.Now(), trade.Timestamp, time.Minute)
}

func (suite *OrderFlowTestSuite) TestPartialFillFlow() {
	// Test partial fill scenario

	// Create large buy order
	buyOrderReq := &orders.CreateOrderRequest{
		UserID:      "user-003",
		ClientOrderID: "buy-large-001",
		Symbol:      "AAPL",
		Side:        orders.SideBuy,
		Type:        orders.TypeLimit,
		Quantity:    1000,
		Price:       151.00,
		TimeInForce: orders.TimeInForceGTC,
	}

	buyOrder, err := suite.orderService.CreateOrder(suite.ctx, buyOrderReq)
	require.NoError(suite.T(), err)

	// Submit to matching engine
	matchingOrder := &matching.Order{
		ID:          buyOrder.ID,
		UserID:      buyOrder.UserID,
		Symbol:      buyOrder.Symbol,
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    buyOrder.Quantity,
		Price:       buyOrder.Price,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   buyOrder.CreatedAt,
	}

	trades, err := suite.matchingEngine.ProcessOrder(suite.ctx, matchingOrder)
	require.NoError(suite.T(), err)
	assert.Empty(suite.T(), trades)

	// Create smaller sell orders for partial fills
	sellQuantities := []float64{300, 200, 150}
	totalFilled := float64(0)

	for i, qty := range sellQuantities {
		sellOrderReq := &orders.CreateOrderRequest{
			UserID:      "user-004",
			ClientOrderID: "sell-partial-" + string(rune(i)),
			Symbol:      "AAPL",
			Side:        orders.SideSell,
			Type:        orders.TypeLimit,
			Quantity:    qty,
			Price:       151.00,
			TimeInForce: orders.TimeInForceGTC,
		}

		sellOrder, err := suite.orderService.CreateOrder(suite.ctx, sellOrderReq)
		require.NoError(suite.T(), err)

		sellMatchingOrder := &matching.Order{
			ID:          sellOrder.ID,
			UserID:      sellOrder.UserID,
			Symbol:      sellOrder.Symbol,
			Side:        matching.SideSell,
			Type:        matching.TypeLimit,
			Quantity:    sellOrder.Quantity,
			Price:       sellOrder.Price,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   sellOrder.CreatedAt,
		}

		trades, err = suite.matchingEngine.ProcessOrder(suite.ctx, sellMatchingOrder)
		require.NoError(suite.T(), err)
		assert.Len(suite.T(), trades, 1)

		trade := trades[0]
		assert.Equal(suite.T(), buyOrder.ID, trade.BuyOrderID)
		assert.Equal(suite.T(), sellOrder.ID, trade.SellOrderID)
		assert.Equal(suite.T(), qty, trade.Quantity)
		assert.Equal(suite.T(), 151.00, trade.Price)

		totalFilled += qty
	}

	// Verify remaining quantity in order book
	orderBook := suite.matchingEngine.GetOrderBook()
	buyLevels := orderBook.GetBuyLevels()
	assert.NotEmpty(suite.T(), buyLevels)

	topBuyLevel := buyLevels[0]
	assert.Equal(suite.T(), 151.00, topBuyLevel.Price)
	assert.Equal(suite.T(), 1000-totalFilled, topBuyLevel.Quantity)
}

func (suite *OrderFlowTestSuite) TestRiskRejectionFlow() {
	// Test order rejection due to risk limits

	// Create portfolio with high concentration
	portfolio := &risk.Portfolio{
		UserID: "user-005",
		Positions: []risk.Position{
			{
				Symbol:        "AAPL",
				Quantity:      5000,
				AveragePrice:  150.00,
				CurrentPrice:  155.00,
				MarketValue:   775000, // 77.5% of portfolio
				UnrealizedPnL: 25000,
			},
		},
		TotalMarketValue:   775000,
		TotalUnrealizedPnL: 25000,
		Cash:              225000,
		TotalValue:        1000000,
	}

	// Try to create order that would exceed concentration limits
	orderRisk := &risk.OrderRisk{
		UserID:    "user-005",
		Symbol:    "AAPL",
		Side:      "buy",
		Quantity:  2000,
		Price:     155.00, // Would add 310k, making AAPL 108.5% of portfolio
		OrderType: "limit",
	}

	riskResult, err := suite.riskCalculator.CalculateOrderRisk(suite.ctx, orderRisk, portfolio)
	require.NoError(suite.T(), err)
	assert.False(suite.T(), riskResult.IsAcceptable, "Order should be rejected due to concentration risk")
	assert.NotEmpty(suite.T(), riskResult.Violations)

	// Verify violation details
	found := false
	for _, violation := range riskResult.Violations {
		if violation.Type == "concentration_limit" {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Should have concentration limit violation")
}

func (suite *OrderFlowTestSuite) TestMarketOrderFlow() {
	// Test market order execution

	// First, create some limit orders to provide liquidity
	sellPrices := []float64{152.00, 152.25, 152.50}
	for i, price := range sellPrices {
		sellOrderReq := &orders.CreateOrderRequest{
			UserID:      "user-liquidity",
			ClientOrderID: "sell-liquidity-" + string(rune(i)),
			Symbol:      "AAPL",
			Side:        orders.SideSell,
			Type:        orders.TypeLimit,
			Quantity:    100,
			Price:       price,
			TimeInForce: orders.TimeInForceGTC,
		}

		sellOrder, err := suite.orderService.CreateOrder(suite.ctx, sellOrderReq)
		require.NoError(suite.T(), err)

		matchingOrder := &matching.Order{
			ID:          sellOrder.ID,
			UserID:      sellOrder.UserID,
			Symbol:      sellOrder.Symbol,
			Side:        matching.SideSell,
			Type:        matching.TypeLimit,
			Quantity:    sellOrder.Quantity,
			Price:       sellOrder.Price,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   sellOrder.CreatedAt,
		}

		_, err = suite.matchingEngine.ProcessOrder(suite.ctx, matchingOrder)
		require.NoError(suite.T(), err)
	}

	// Now create market buy order
	marketOrderReq := &orders.CreateOrderRequest{
		UserID:      "user-006",
		ClientOrderID: "market-buy-001",
		Symbol:      "AAPL",
		Side:        orders.SideBuy,
		Type:        orders.TypeMarket,
		Quantity:    250, // Should match against all three sell orders
		TimeInForce: orders.TimeInForceIOC,
	}

	marketOrder, err := suite.orderService.CreateOrder(suite.ctx, marketOrderReq)
	require.NoError(suite.T(), err)

	// Submit market order to matching engine
	marketMatchingOrder := &matching.Order{
		ID:          marketOrder.ID,
		UserID:      marketOrder.UserID,
		Symbol:      marketOrder.Symbol,
		Side:        matching.SideBuy,
		Type:        matching.TypeMarket,
		Quantity:    marketOrder.Quantity,
		TimeInForce: matching.TimeInForceIOC,
		Timestamp:   marketOrder.CreatedAt,
	}

	trades, err := suite.matchingEngine.ProcessOrder(suite.ctx, marketMatchingOrder)
	require.NoError(suite.T(), err)
	assert.Len(suite.T(), trades, 3, "Should generate three trades")

	// Verify trades are at correct prices
	expectedPrices := []float64{152.00, 152.25, 152.50}
	for i, trade := range trades {
		assert.Equal(suite.T(), marketOrder.ID, trade.BuyOrderID)
		assert.Equal(suite.T(), float64(100), trade.Quantity)
		assert.Equal(suite.T(), expectedPrices[i], trade.Price)
	}
}

func (suite *OrderFlowTestSuite) TestOrderCancellationFlow() {
	// Test order cancellation flow

	// Create order
	orderReq := &orders.CreateOrderRequest{
		UserID:      "user-007",
		ClientOrderID: "cancel-test-001",
		Symbol:      "AAPL",
		Side:        orders.SideBuy,
		Type:        orders.TypeLimit,
		Quantity:    100,
		Price:       149.00,
		TimeInForce: orders.TimeInForceGTC,
	}

	order, err := suite.orderService.CreateOrder(suite.ctx, orderReq)
	require.NoError(suite.T(), err)

	// Submit to matching engine
	matchingOrder := &matching.Order{
		ID:          order.ID,
		UserID:      order.UserID,
		Symbol:      order.Symbol,
		Side:        matching.SideBuy,
		Type:        matching.TypeLimit,
		Quantity:    order.Quantity,
		Price:       order.Price,
		TimeInForce: matching.TimeInForceGTC,
		Timestamp:   order.CreatedAt,
	}

	_, err = suite.matchingEngine.ProcessOrder(suite.ctx, matchingOrder)
	require.NoError(suite.T(), err)

	// Verify order is in the book
	orderBook := suite.matchingEngine.GetOrderBook()
	buyLevels := orderBook.GetBuyLevels()
	found := false
	for _, level := range buyLevels {
		if level.Price == 149.00 {
			found = true
			break
		}
	}
	assert.True(suite.T(), found, "Order should be in order book")

	// Cancel order
	cancelReq := &orders.CancelOrderRequest{
		OrderID: order.ID,
		UserID:  "user-007",
		Reason:  "User cancellation",
	}

	cancelledOrder, err := suite.orderService.CancelOrder(suite.ctx, cancelReq)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), orders.StatusCancelled, cancelledOrder.Status)

	// Cancel in matching engine
	err = suite.matchingEngine.CancelOrder(suite.ctx, order.ID, order.UserID)
	require.NoError(suite.T(), err)

	// Verify order is removed from book
	orderBook = suite.matchingEngine.GetOrderBook()
	buyLevels = orderBook.GetBuyLevels()
	found = false
	for _, level := range buyLevels {
		if level.Price == 149.00 {
			found = true
			break
		}
	}
	assert.False(suite.T(), found, "Order should be removed from order book")
}

func (suite *OrderFlowTestSuite) TestHighFrequencyOrderFlow() {
	// Test high-frequency order processing

	startTime := time.Now()
	orderCount := 1000
	trades := make([]*matching.Trade, 0)

	// Create alternating buy and sell orders rapidly
	for i := 0; i < orderCount; i++ {
		var side orders.Side
		var matchingSide matching.Side
		var price float64

		if i%2 == 0 {
			side = orders.SideBuy
			matchingSide = matching.SideBuy
			price = 150.00 + float64(i%10)*0.01
		} else {
			side = orders.SideSell
			matchingSide = matching.SideSell
			price = 150.00 + float64(i%10)*0.01
		}

		orderReq := &orders.CreateOrderRequest{
			UserID:      "user-hft",
			ClientOrderID: "hft-" + string(rune(i)),
			Symbol:      "AAPL",
			Side:        side,
			Type:        orders.TypeLimit,
			Quantity:    100,
			Price:       price,
			TimeInForce: orders.TimeInForceGTC,
		}

		order, err := suite.orderService.CreateOrder(suite.ctx, orderReq)
		require.NoError(suite.T(), err)

		matchingOrder := &matching.Order{
			ID:          order.ID,
			UserID:      order.UserID,
			Symbol:      order.Symbol,
			Side:        matchingSide,
			Type:        matching.TypeLimit,
			Quantity:    order.Quantity,
			Price:       order.Price,
			TimeInForce: matching.TimeInForceGTC,
			Timestamp:   order.CreatedAt,
		}

		orderTrades, err := suite.matchingEngine.ProcessOrder(suite.ctx, matchingOrder)
		require.NoError(suite.T(), err)
		trades = append(trades, orderTrades...)
	}

	duration := time.Since(startTime)
	ordersPerSecond := float64(orderCount) / duration.Seconds()

	suite.T().Logf("Processed %d orders in %v", orderCount, duration)
	suite.T().Logf("Orders per second: %.2f", ordersPerSecond)
	suite.T().Logf("Generated %d trades", len(trades))

	// Assert performance targets
	assert.Greater(suite.T(), ordersPerSecond, 10000.0, "Should process at least 10,000 orders per second")
	assert.Less(suite.T(), duration.Nanoseconds()/int64(orderCount), int64(100000), "Average order processing should be under 100μs")
}

func TestOrderFlowTestSuite(t *testing.T) {
	suite.Run(t, new(OrderFlowTestSuite))
}
