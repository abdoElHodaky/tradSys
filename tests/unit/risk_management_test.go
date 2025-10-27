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
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Create test portfolio
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
			{
				Symbol:        "GOOGL",
				Quantity:      100,
				AveragePrice:  2800.00,
				CurrentPrice:  2750.00,
				MarketValue:   275000,
				UnrealizedPnL: -5000,
			},
		},
		TotalMarketValue:   430000,
		TotalUnrealizedPnL: 0,
		Cash:               70000,
		TotalValue:         500000,
	}

	// Calculate VaR
	varResult, err := calculator.CalculateVaR(ctx, portfolio)
	require.NoError(t, err)
	assert.NotNil(t, varResult)

	// VaR should be positive (potential loss)
	assert.Greater(t, varResult.VaR95, 0.0)
	assert.Greater(t, varResult.VaR99, varResult.VaR95, "99% VaR should be higher than 95% VaR")

	// Expected Shortfall should be higher than VaR
	assert.Greater(t, varResult.ExpectedShortfall, varResult.VaR95)

	// Confidence level should match
	assert.Equal(t, 0.95, varResult.Confidence)

	// Should have calculation timestamp
	assert.WithinDuration(t, time.Now(), varResult.CalculatedAt, time.Minute)
}

func TestRiskCalculator_GreeksCalculation(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Create test option position
	option := &risk.OptionPosition{
		Symbol:           "AAPL240315C00150000", // AAPL Call option
		UnderlyingSymbol: "AAPL",
		OptionType:       risk.OptionTypeCall,
		Strike:           150.00,
		Expiry:           time.Now().AddDate(0, 3, 0), // 3 months
		Quantity:         10,
		Premium:          5.50,
		UnderlyingPrice:  155.00,
		Volatility:       0.25,
		RiskFreeRate:     0.05,
	}

	// Calculate Greeks
	greeks, err := calculator.CalculateGreeks(ctx, option)
	require.NoError(t, err)
	assert.NotNil(t, greeks)

	// Delta should be between 0 and 1 for call options
	assert.Greater(t, greeks.Delta, 0.0)
	assert.Less(t, greeks.Delta, 1.0)

	// Gamma should be positive
	assert.Greater(t, greeks.Gamma, 0.0)

	// Theta should be negative (time decay)
	assert.Less(t, greeks.Theta, 0.0)

	// Vega should be positive
	assert.Greater(t, greeks.Vega, 0.0)

	// Rho should be positive for call options
	assert.Greater(t, greeks.Rho, 0.0)
}

func TestRiskCalculator_ConcentrationRisk(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3, // 30% max concentration
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Create portfolio with high concentration in one stock
	portfolio := &risk.Portfolio{
		UserID: "user-001",
		Positions: []risk.Position{
			{
				Symbol:        "AAPL",
				Quantity:      2000,
				AveragePrice:  150.00,
				CurrentPrice:  155.00,
				MarketValue:   310000, // 62% of portfolio
				UnrealizedPnL: 10000,
			},
			{
				Symbol:        "GOOGL",
				Quantity:      50,
				AveragePrice:  2800.00,
				CurrentPrice:  2750.00,
				MarketValue:   137500, // 27.5% of portfolio
				UnrealizedPnL: -2500,
			},
			{
				Symbol:        "MSFT",
				Quantity:      150,
				AveragePrice:  350.00,
				CurrentPrice:  355.00,
				MarketValue:   53250, // 10.5% of portfolio
				UnrealizedPnL: 750,
			},
		},
		TotalMarketValue:   500750,
		TotalUnrealizedPnL: 8250,
		Cash:               -750,
		TotalValue:         500000,
	}

	// Calculate concentration risk
	concentration, err := calculator.CalculateConcentrationRisk(ctx, portfolio)
	require.NoError(t, err)
	assert.NotNil(t, concentration)

	// Should identify AAPL as highest concentration
	assert.Equal(t, "AAPL", concentration.HighestConcentration.Symbol)
	assert.Greater(t, concentration.HighestConcentration.Percentage, 0.6) // > 60%

	// Should exceed concentration limit
	assert.True(t, concentration.ExceedsLimit)
	assert.Greater(t, concentration.ConcentrationRatio, 0.3) // > 30% limit

	// Should have risk level
	assert.Equal(t, risk.RiskLevelHigh, concentration.RiskLevel)
}

func TestRiskCalculator_PositionRisk(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Test position within limits
	position := &risk.Position{
		Symbol:        "AAPL",
		Quantity:      1000,
		AveragePrice:  150.00,
		CurrentPrice:  155.00,
		MarketValue:   155000,
		UnrealizedPnL: 5000,
	}

	positionRisk, err := calculator.CalculatePositionRisk(ctx, position)
	require.NoError(t, err)
	assert.NotNil(t, positionRisk)

	// Should be within limits
	assert.False(t, positionRisk.ExceedsLimit)
	assert.Equal(t, risk.RiskLevelLow, positionRisk.RiskLevel)

	// Test position exceeding limits
	largePosition := &risk.Position{
		Symbol:        "AAPL",
		Quantity:      10000,
		AveragePrice:  150.00,
		CurrentPrice:  155.00,
		MarketValue:   1550000, // Exceeds 1M limit
		UnrealizedPnL: 50000,
	}

	largePositionRisk, err := calculator.CalculatePositionRisk(ctx, largePosition)
	require.NoError(t, err)
	assert.NotNil(t, largePositionRisk)

	// Should exceed limits
	assert.True(t, largePositionRisk.ExceedsLimit)
	assert.Equal(t, risk.RiskLevelHigh, largePositionRisk.RiskLevel)
}

func TestRiskCalculator_OrderRisk(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Create existing portfolio
	portfolio := &risk.Portfolio{
		UserID: "user-001",
		Positions: []risk.Position{
			{
				Symbol:        "AAPL",
				Quantity:      500,
				AveragePrice:  150.00,
				CurrentPrice:  155.00,
				MarketValue:   77500,
				UnrealizedPnL: 2500,
			},
		},
		TotalMarketValue:   77500,
		TotalUnrealizedPnL: 2500,
		Cash:               422500,
		TotalValue:         500000,
	}

	// Test order that would be acceptable
	order := &risk.OrderRisk{
		UserID:    "user-001",
		Symbol:    "AAPL",
		Side:      "buy",
		Quantity:  1000,
		Price:     155.00,
		OrderType: "limit",
	}

	orderRisk, err := calculator.CalculateOrderRisk(ctx, order, portfolio)
	require.NoError(t, err)
	assert.NotNil(t, orderRisk)

	// Should be acceptable
	assert.True(t, orderRisk.IsAcceptable)
	assert.Equal(t, risk.RiskLevelMedium, orderRisk.RiskLevel)

	// Test order that would exceed limits
	largeOrder := &risk.OrderRisk{
		UserID:    "user-001",
		Symbol:    "AAPL",
		Side:      "buy",
		Quantity:  10000,
		Price:     155.00,
		OrderType: "limit",
	}

	largeOrderRisk, err := calculator.CalculateOrderRisk(ctx, largeOrder, portfolio)
	require.NoError(t, err)
	assert.NotNil(t, largeOrderRisk)

	// Should be rejected
	assert.False(t, largeOrderRisk.IsAcceptable)
	assert.Equal(t, risk.RiskLevelHigh, largeOrderRisk.RiskLevel)
	assert.NotEmpty(t, largeOrderRisk.Violations)
}

func TestRiskCalculator_AccountRisk(t *testing.T) {
	calculator := risk.NewCalculator(&risk.Config{
		VaRConfidence:       0.95,
		CalculationInterval: time.Second,
		MaxPositionSize:     1000000,
		ConcentrationLimit:  0.3,
		EnableRealTimeCalc:  true,
	})

	ctx := context.Background()

	// Create account with multiple portfolios
	account := &risk.Account{
		UserID: "user-001",
		Portfolios: map[string]*risk.Portfolio{
			"main": {
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
			},
			"retirement": {
				UserID: "user-001",
				Positions: []risk.Position{
					{
						Symbol:        "SPY",
						Quantity:      500,
						AveragePrice:  400.00,
						CurrentPrice:  410.00,
						MarketValue:   205000,
						UnrealizedPnL: 5000,
					},
				},
				TotalMarketValue:   205000,
				TotalUnrealizedPnL: 5000,
				Cash:               95000,
				TotalValue:         300000,
			},
		},
		TotalValue:         800000,
		TotalUnrealizedPnL: 10000,
		MarginUsed:         0,
		AvailableMargin:    200000,
	}

	// Calculate account risk
	accountRisk, err := calculator.CalculateAccountRisk(ctx, account)
	require.NoError(t, err)
	assert.NotNil(t, accountRisk)

	// Should have overall risk assessment
	assert.Equal(t, risk.RiskLevelLow, accountRisk.OverallRiskLevel)
	assert.Greater(t, accountRisk.TotalVaR, 0.0)
	assert.Greater(t, accountRisk.DiversificationRatio, 0.0)
	assert.Less(t, accountRisk.DiversificationRatio, 1.0)

	// Should have portfolio-level risks
	assert.Len(t, accountRisk.PortfolioRisks, 2)
	assert.Contains(t, accountRisk.PortfolioRisks, "main")
	assert.Contains(t, accountRisk.PortfolioRisks, "retirement")
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
