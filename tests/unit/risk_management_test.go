package unit

import (
	"context"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/risk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskCalculator_VaRCalculation(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Create test positions
	positions := []*risk.Position{
		{
			ID:             "pos-001",
			UserID:         "user-001",
			Symbol:         "AAPL",
			Quantity:       1000,
			AveragePrice:   150.00,
			MarketValue:    155000,
			UnrealizedPnL:  5000,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             "pos-002",
			UserID:         "user-001",
			Symbol:         "GOOGL",
			Quantity:       100,
			AveragePrice:   2800.00,
			MarketValue:    275000,
			UnrealizedPnL:  -5000,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	// Create price map for current prices
	prices := map[string]float64{
		"AAPL":  155.00,
		"GOOGL": 2750.00,
	}

	// Calculate account risk (includes VaR calculations)
	accountRisk, err := calculator.CalculateAccountRisk(ctx, "user-001", positions, prices)
	require.NoError(t, err)
	assert.NotNil(t, accountRisk)

	// VaR should be positive (potential loss)
	assert.Greater(t, accountRisk.PortfolioVaR95, 0.0)
	assert.Greater(t, accountRisk.PortfolioVaR99, accountRisk.PortfolioVaR95, "99% VaR should be higher than 95% VaR")

	// Should have calculation timestamp
	assert.WithinDuration(t, time.Now(), accountRisk.CalculatedAt, time.Minute)

	// Should have risk level
	assert.NotEmpty(t, accountRisk.RiskLevel)
}

func TestRiskCalculator_GreeksCalculation(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Create test option position
	optionPosition := &risk.Position{
		ID:             "opt-001",
		UserID:         "user-001",
		Symbol:         "AAPL240315C00150000", // AAPL Call option
		Quantity:       10,
		AveragePrice:   5.50, // Premium paid
		MarketValue:    5500,
		UnrealizedPnL:  0,
		RealizedPnL:    0,
		InstrumentType: "option", // This triggers Greeks calculation
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	currentPrice := 155.00

	// Calculate position risk (includes Greeks for options)
	positionRisk, err := calculator.CalculatePositionRisk(ctx, optionPosition, currentPrice)
	require.NoError(t, err)
	assert.NotNil(t, positionRisk)

	// Delta should be between 0 and 1 for call options
	assert.Greater(t, positionRisk.Delta, 0.0)
	assert.Less(t, positionRisk.Delta, 1.0)

	// Gamma should be positive
	assert.Greater(t, positionRisk.Gamma, 0.0)

	// Theta should be negative (time decay)
	assert.Less(t, positionRisk.Theta, 0.0)

	// Vega should be positive
	assert.Greater(t, positionRisk.Vega, 0.0)

	// Should have risk level
	assert.NotEmpty(t, positionRisk.RiskLevel)
}

func TestRiskCalculator_ConcentrationRisk(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Create positions with high concentration in one stock
	positions := []*risk.Position{
		{
			ID:             "pos-001",
			UserID:         "user-001",
			Symbol:         "AAPL",
			Quantity:       2000,
			AveragePrice:   150.00,
			MarketValue:    310000, // 62% of portfolio
			UnrealizedPnL:  10000,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             "pos-002",
			UserID:         "user-001",
			Symbol:         "GOOGL",
			Quantity:       50,
			AveragePrice:   2800.00,
			MarketValue:    137500, // 27.5% of portfolio
			UnrealizedPnL:  -2500,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             "pos-003",
			UserID:         "user-001",
			Symbol:         "MSFT",
			Quantity:       150,
			AveragePrice:   350.00,
			MarketValue:    53250, // 10.5% of portfolio
			UnrealizedPnL:  750,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	// Create price map
	prices := map[string]float64{
		"AAPL":  155.00,
		"GOOGL": 2750.00,
		"MSFT":  355.00,
	}

	// Calculate account risk (includes concentration risk)
	accountRisk, err := calculator.CalculateAccountRisk(ctx, "user-001", positions, prices)
	require.NoError(t, err)
	assert.NotNil(t, accountRisk)

	// Should have concentration risk calculated
	assert.Greater(t, accountRisk.ConcentrationRisk, 0.0)
	
	// Should have risk level
	assert.NotEmpty(t, accountRisk.RiskLevel)
}

func TestRiskCalculator_PositionRisk(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Test position within limits
	position := &risk.Position{
		ID:             "pos-001",
		UserID:         "user-001",
		Symbol:         "AAPL",
		Quantity:       1000,
		AveragePrice:   150.00,
		MarketValue:    155000,
		UnrealizedPnL:  5000,
		RealizedPnL:    0,
		InstrumentType: "stock",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	currentPrice := 155.00

	positionRisk, err := calculator.CalculatePositionRisk(ctx, position, currentPrice)
	require.NoError(t, err)
	assert.NotNil(t, positionRisk)

	// Should have risk level
	assert.NotEmpty(t, positionRisk.RiskLevel)
	assert.Equal(t, currentPrice, positionRisk.CurrentPrice)

	// Test large position
	largePosition := &risk.Position{
		ID:             "pos-002",
		UserID:         "user-001",
		Symbol:         "AAPL",
		Quantity:       10000,
		AveragePrice:   150.00,
		MarketValue:    1550000, // Large position
		UnrealizedPnL:  50000,
		RealizedPnL:    0,
		InstrumentType: "stock",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	largePositionRisk, err := calculator.CalculatePositionRisk(ctx, largePosition, currentPrice)
	require.NoError(t, err)
	assert.NotNil(t, largePositionRisk)

	// Should have risk level (likely higher than normal position)
	assert.NotEmpty(t, largePositionRisk.RiskLevel)
	assert.Equal(t, currentPrice, largePositionRisk.CurrentPrice)
}

func TestRiskCalculator_OrderRisk(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Create existing position
	currentPosition := &risk.Position{
		ID:             "pos-001",
		UserID:         "user-001",
		Symbol:         "AAPL",
		Quantity:       500,
		AveragePrice:   150.00,
		MarketValue:    77500,
		UnrealizedPnL:  2500,
		RealizedPnL:    0,
		InstrumentType: "stock",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	// Test order that would be acceptable
	order := &orders.Order{
		ID:       "order-001",
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Quantity: 1000,
		Price:    155.00,
		Type:     orders.OrderTypeLimit,
		Status:   orders.OrderStatusNew,
	}

	currentPrice := 155.00

	orderRisk, err := calculator.CalculateOrderRisk(ctx, order, currentPosition, currentPrice)
	require.NoError(t, err)
	assert.NotNil(t, orderRisk)

	// Should have risk level
	assert.NotEmpty(t, orderRisk.RiskLevel)
	assert.Equal(t, currentPrice, orderRisk.CurrentPrice)
	assert.Equal(t, order.Quantity, orderRisk.Quantity)

	// Test large order
	largeOrder := &orders.Order{
		ID:       "order-002",
		UserID:   "user-001",
		Symbol:   "AAPL",
		Side:     orders.OrderSideBuy,
		Quantity: 10000,
		Price:    155.00,
		Type:     orders.OrderTypeLimit,
		Status:   orders.OrderStatusNew,
	}

	largeOrderRisk, err := calculator.CalculateOrderRisk(ctx, largeOrder, currentPosition, currentPrice)
	require.NoError(t, err)
	assert.NotNil(t, largeOrderRisk)

	// Should have risk level (likely higher than normal order)
	assert.NotEmpty(t, largeOrderRisk.RiskLevel)
	assert.Equal(t, currentPrice, largeOrderRisk.CurrentPrice)
	assert.Equal(t, largeOrder.Quantity, largeOrderRisk.Quantity)
}

func TestRiskCalculator_AccountRisk(t *testing.T) {
	calculator := risk.NewCalculator(nil) // Updated constructor signature

	ctx := context.Background()

	// Create multiple positions for account risk calculation
	positions := []*risk.Position{
		{
			ID:             "pos-001",
			UserID:         "user-001",
			Symbol:         "AAPL",
			Quantity:       1000,
			AveragePrice:   150.00,
			MarketValue:    155000,
			UnrealizedPnL:  5000,
			RealizedPnL:    0,
			InstrumentType: "stock",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
		{
			ID:             "pos-002",
			UserID:         "user-001",
			Symbol:         "SPY",
			Quantity:       500,
			AveragePrice:   400.00,
			MarketValue:    205000,
			UnrealizedPnL:  5000,
			RealizedPnL:    0,
			InstrumentType: "etf",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		},
	}

	// Create price map
	prices := map[string]float64{
		"AAPL": 155.00,
		"SPY":  410.00,
	}

	// Calculate account risk
	accountRisk, err := calculator.CalculateAccountRisk(ctx, "user-001", positions, prices)
	require.NoError(t, err)
	assert.NotNil(t, accountRisk)

	// Should have risk assessment
	assert.NotEmpty(t, accountRisk.RiskLevel)
	assert.Greater(t, accountRisk.PortfolioVaR95, 0.0)
	assert.Greater(t, accountRisk.PortfolioVaR99, accountRisk.PortfolioVaR95)
	assert.Greater(t, accountRisk.TotalMarketValue, 0.0)
	assert.Equal(t, "user-001", accountRisk.UserID)

	// Should have concentration risk calculated
	assert.GreaterOrEqual(t, accountRisk.ConcentrationRisk, 0.0)
}

func TestRiskCalculator_RealTimeMonitoring(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: 100 * time.Millisecond, // Fast for testing
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Create portfolio
	portfolio := &risk.Portfolio{
		UserID: "user-001",
		Positions: []risk.Position{
			{
				Symbol:        "AAPL",
				Quantity:      1000,
				AveragePrice:  150.00,
				CurrentPrice:  155.00,
				MarketValue:   155000,
				UnrealizedPnL: 5000,
			},
		},
		TotalMarketValue:   155000,
		TotalUnrealizedPnL: 5000,
		Cash:               345000,
		TotalValue:         500000,
	}

	// Start real-time monitoring
	alertChan := make(chan *risk.RiskAlert, 10)
	err := calculator.StartRealTimeMonitoring(ctx, portfolio, alertChan)
	require.NoError(t, err)

	// Simulate price change that triggers alert
	portfolio.Positions[0].CurrentPrice = 140.00 // 10% drop
	portfolio.Positions[0].MarketValue = 140000
	portfolio.Positions[0].UnrealizedPnL = -10000
	portfolio.TotalMarketValue = 140000
	portfolio.TotalUnrealizedPnL = -10000
	portfolio.TotalValue = 485000

	// Update portfolio in calculator
	err = calculator.UpdatePortfolio(ctx, portfolio)
	require.NoError(t, err)

	// Wait for alert
	select {
	case alert := <-alertChan:
		assert.NotNil(t, alert)
		assert.Equal(t, "user-001", alert.UserID)
		assert.Equal(t, risk.AlertTypePositionLoss, alert.Type)
		assert.Greater(t, alert.Severity, risk.SeverityMedium)
	case <-time.After(1 * time.Second):
		t.Fatal("Expected risk alert but none received")
	}
}

func BenchmarkRiskCalculator_VaRCalculation(b *testing.B) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  false, // Disable for benchmarking
	})

	ctx := context.Background()

	// Create large portfolio for benchmarking
	positions := make([]risk.Position, 100)
	for i := 0; i < 100; i++ {
		positions[i] = risk.Position{
			Symbol:        "STOCK" + string(rune(i)),
			Quantity:      1000,
			AveragePrice:  100.0 + float64(i),
			CurrentPrice:  105.0 + float64(i),
			MarketValue:   105000 + float64(i*1000),
			UnrealizedPnL: 5000,
		}
	}

	portfolio := &risk.Portfolio{
		UserID:             "user-001",
		Positions:          positions,
		TotalMarketValue:   10500000,
		TotalUnrealizedPnL: 500000,
		Cash:               500000,
		TotalValue:         11000000,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := calculator.CalculateVaR(ctx, portfolio)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkRiskCalculator_GreeksCalculation(b *testing.B) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  false,
	})

	ctx := context.Background()

	option := &risk.OptionPosition{
		Symbol:           "AAPL240315C00150000",
		UnderlyingSymbol: "AAPL",
		OptionType:       risk.OptionTypeCall,
		Strike:           150.00,
		Expiry:           time.Now().AddDate(0, 3, 0),
		Quantity:         10,
		Premium:          5.50,
		UnderlyingPrice:  155.00,
		Volatility:       0.25,
		RiskFreeRate:     0.05,
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := calculator.CalculateGreeks(ctx, option)
		if err != nil {
			b.Fatal(err)
		}
	}
}
