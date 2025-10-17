// This file has been consolidated into marketDataRepository.go
// The GORM-based implementation is now the single source of truth for market data operations
// This file is kept as a placeholder to avoid breaking imports during the transition

package repositories

import (
	"go.uber.org/fx"
)

// MarketDataRepositoryModule provides the market data repository module for fx
// This now uses the unified GORM-based implementation
var MarketDataRepositoryModule = fx.Options(
	fx.Provide(NewMarketDataRepository),
)
