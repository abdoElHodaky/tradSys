// Package assets provides asset handler registry and management for TradSys v3
package assets

import (
	"context"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/pkg/interfaces"
	"github.com/abdoElHodaky/tradSys/pkg/types"
)

// StockHandler handles stock-specific operations
type StockHandler struct {
	BaseAssetHandler
}

func NewStockHandler() *StockHandler {
	return &StockHandler{
		BaseAssetHandler: BaseAssetHandler{
			AssetType:      types.STOCK,
			MinOrderSize:   1,
			MaxOrderSize:   1000000,
			PriceStep:      0.01,
			SettlementDays: 2, // T+2
			RiskParameters: &RiskParameters{
				MaxPositionSize:    100000,
				MaxDailyVolume:     1000000,
				VolatilityLimit:    0.20,
				LiquidityThreshold: 10000,
				MarginRequirement:  0.25,
			},
			FeeStructure: &FeeStructure{
				CommissionRate: 0.001, // 0.1%
				MinCommission:  5.0,
				MaxCommission:  100.0,
				ExchangeFee:    0.0001,
				RegulatoryFee:  0.0001,
			},
		},
	}
}

// SukukHandler handles Sukuk-specific operations
type SukukHandler struct {
	BaseAssetHandler
}

func NewSukukHandler() *SukukHandler {
	return &SukukHandler{
		BaseAssetHandler: BaseAssetHandler{
			AssetType:      types.SUKUK,
			MinOrderSize:   1000, // Higher minimum for Sukuk
			MaxOrderSize:   10000000,
			PriceStep:      0.01,
			SettlementDays: 1, // T+1 for bonds
			RiskParameters: &RiskParameters{
				MaxPositionSize:    1000000,
				MaxDailyVolume:     10000000,
				VolatilityLimit:    0.10,
				LiquidityThreshold: 100000,
				MarginRequirement:  0.10,
			},
			FeeStructure: &FeeStructure{
				CommissionRate: 0.0005, // 0.05% - lower for bonds
				MinCommission:  10.0,
				MaxCommission:  500.0,
				ExchangeFee:    0.00005,
				RegulatoryFee:  0.00005,
			},
		},
	}
}

func (s *SukukHandler) ValidateOrder(ctx context.Context, order *interfaces.Order) error {
	// Call base validation first
	if err := s.BaseAssetHandler.ValidateOrder(ctx, order); err != nil {
		return err
	}

	// Sukuk-specific validations
	if time.Now().Weekday() == time.Friday {
		return fmt.Errorf("Sukuk trading not allowed on Fridays")
	}

	return nil
}

// Factory functions for other handlers (simplified implementations)
func NewBondHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.BOND,
		MinOrderSize:   1000,
		MaxOrderSize:   10000000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    1000000,
			MaxDailyVolume:     10000000,
			VolatilityLimit:    0.10,
			LiquidityThreshold: 100000,
			MarginRequirement:  0.10,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0005,
			MinCommission:  10.0,
			MaxCommission:  500.0,
			ExchangeFee:    0.00005,
			RegulatoryFee:  0.00005,
		},
	}
}

func NewETFHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ETF,
		MinOrderSize:   1,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    200000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 20000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0008,
			MinCommission:  3.0,
			MaxCommission:  80.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewREITHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.REIT,
		MinOrderSize:   1,
		MaxOrderSize:   200000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    100000,
			MaxDailyVolume:     1000000,
			VolatilityLimit:    0.18,
			LiquidityThreshold: 15000,
			MarginRequirement:  0.30,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0012,
			MinCommission:  5.0,
			MaxCommission:  120.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewMutualFundHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.MUTUAL_FUND,
		MinOrderSize:   100,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     5000000,
			VolatilityLimit:    0.12,
			LiquidityThreshold: 50000,
			MarginRequirement:  0.15,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0015,
			MinCommission:  10.0,
			MaxCommission:  200.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewCryptoHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.CRYPTO,
		MinOrderSize:   0.001,
		MaxOrderSize:   1000,
		PriceStep:      0.00000001,
		SettlementDays: 0, // Instant settlement
		RiskParameters: &RiskParameters{
			MaxPositionSize:    10000,
			MaxDailyVolume:     100000,
			VolatilityLimit:    0.50,
			LiquidityThreshold: 1000,
			MarginRequirement:  0.50,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.002,
			MinCommission:  1.0,
			MaxCommission:  50.0,
			ExchangeFee:    0.0005,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewForexHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.FOREX,
		MinOrderSize:   1000,
		MaxOrderSize:   10000000,
		PriceStep:      0.00001,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    1000000,
			MaxDailyVolume:     10000000,
			VolatilityLimit:    0.30,
			LiquidityThreshold: 100000,
			MarginRequirement:  0.02, // 2% margin for forex
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0001,
			MinCommission:  0.0,
			MaxCommission:  0.0,
			ExchangeFee:    0.00001,
			RegulatoryFee:  0.00001,
		},
	}
}

func NewCommodityHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.COMMODITY,
		MinOrderSize:   1,
		MaxOrderSize:   100000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    50000,
			MaxDailyVolume:     500000,
			VolatilityLimit:    0.25,
			LiquidityThreshold: 5000,
			MarginRequirement:  0.10,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0008,
			MinCommission:  5.0,
			MaxCommission:  100.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}

// Islamic Asset Handlers
func NewIslamicFundHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_FUND,
		MinOrderSize:   100,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    200000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.12,
			LiquidityThreshold: 20000,
			MarginRequirement:  0.15,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0012,
			MinCommission:  8.0,
			MaxCommission:  150.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewShariaStockHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.SHARIA_STOCK,
		MinOrderSize:   1,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    100000,
			MaxDailyVolume:     1000000,
			VolatilityLimit:    0.18,
			LiquidityThreshold: 10000,
			MarginRequirement:  0.25,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.001,
			MinCommission:  5.0,
			MaxCommission:  100.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewIslamicETFHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_ETF,
		MinOrderSize:   1,
		MaxOrderSize:   1000000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    500000,
			MaxDailyVolume:     5000000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 50000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0007,
			MinCommission:  4.0,
			MaxCommission:  90.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewIslamicREITHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.ISLAMIC_REIT,
		MinOrderSize:   1,
		MaxOrderSize:   500000,
		PriceStep:      0.01,
		SettlementDays: 2,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    200000,
			MaxDailyVolume:     2000000,
			VolatilityLimit:    0.18,
			LiquidityThreshold: 20000,
			MarginRequirement:  0.30,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0009,
			MinCommission:  4.0,
			MaxCommission:  130.0,
			ExchangeFee:    0.0001,
			RegulatoryFee:  0.0001,
		},
	}
}

func NewTakafulHandler() *BaseAssetHandler {
	return &BaseAssetHandler{
		AssetType:      types.TAKAFUL,
		MinOrderSize:   1,
		MaxOrderSize:   100000,
		PriceStep:      0.01,
		SettlementDays: 1,
		RiskParameters: &RiskParameters{
			MaxPositionSize:    50000,
			MaxDailyVolume:     500000,
			VolatilityLimit:    0.15,
			LiquidityThreshold: 5000,
			MarginRequirement:  0.20,
		},
		FeeStructure: &FeeStructure{
			CommissionRate: 0.0015,
			MinCommission:  5.0,
			MaxCommission:  100.0,
			ExchangeFee:    0.0002,
			RegulatoryFee:  0.0001,
		},
	}
}
