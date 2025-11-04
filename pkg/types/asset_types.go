// Package types provides unified type definitions for TradSys v3
package types

import (
	"fmt"
	"strings"
)

// AssetType represents different types of financial assets
type AssetType string

const (
	// Traditional Assets
	STOCK       AssetType = "STOCK"
	BOND        AssetType = "BOND"
	ETF         AssetType = "ETF"
	REIT        AssetType = "REIT"
	MUTUAL_FUND AssetType = "MUTUAL_FUND"
	CRYPTO      AssetType = "CRYPTO"
	FOREX       AssetType = "FOREX"
	COMMODITY   AssetType = "COMMODITY"

	// Islamic Assets
	SUKUK        AssetType = "SUKUK"
	ISLAMIC_FUND AssetType = "ISLAMIC_FUND"
	SHARIA_STOCK AssetType = "SHARIA_STOCK"
	ISLAMIC_ETF  AssetType = "ISLAMIC_ETF"
	ISLAMIC_REIT AssetType = "ISLAMIC_REIT"
	TAKAFUL      AssetType = "TAKAFUL"
)

// IsValid checks if the asset type is valid
func (at AssetType) IsValid() bool {
	switch at {
	case STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL:
		return true
	default:
		return false
	}
}

// IsIslamic returns true if the asset type is Islamic/Sharia-compliant
func (at AssetType) IsIslamic() bool {
	switch at {
	case SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL:
		return true
	default:
		return false
	}
}

// String returns the string representation of AssetType
func (at AssetType) String() string {
	return string(at)
}

// ParseAssetType parses a string into AssetType
func ParseAssetType(s string) (AssetType, error) {
	assetType := AssetType(strings.ToUpper(s))
	if !assetType.IsValid() {
		return "", fmt.Errorf("invalid asset type: %s", s)
	}
	return assetType, nil
}

// GetAllAssetTypes returns all valid asset types
func GetAllAssetTypes() []AssetType {
	return []AssetType{
		STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL,
	}
}

// GetTraditionalAssetTypes returns traditional (non-Islamic) asset types
func GetTraditionalAssetTypes() []AssetType {
	return []AssetType{
		STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
	}
}

// GetIslamicAssetTypes returns Islamic/Sharia-compliant asset types
func GetIslamicAssetTypes() []AssetType {
	return []AssetType{
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL,
	}
}
