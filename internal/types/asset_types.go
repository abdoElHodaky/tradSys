// Package types provides unified type definitions for TradSys v3
package types

import (
	"fmt"
	"strings"
)

// AssetTypeString represents different types of financial assets (string-based)
type AssetTypeString string

const (
	// Traditional Assets
	STOCK       AssetTypeString = "STOCK"
	BOND        AssetTypeString = "BOND"
	ETF         AssetTypeString = "ETF"
	REIT        AssetTypeString = "REIT"
	MUTUAL_FUND AssetTypeString = "MUTUAL_FUND"
	CRYPTO      AssetTypeString = "CRYPTO"
	FOREX       AssetTypeString = "FOREX"
	COMMODITY   AssetTypeString = "COMMODITY"

	// Islamic Assets
	SUKUK        AssetTypeString = "SUKUK"
	ISLAMIC_FUND AssetTypeString = "ISLAMIC_FUND"
	SHARIA_STOCK AssetTypeString = "SHARIA_STOCK"
	ISLAMIC_ETF  AssetTypeString = "ISLAMIC_ETF"
	ISLAMIC_REIT AssetTypeString = "ISLAMIC_REIT"
	TAKAFUL      AssetTypeString = "TAKAFUL"
)

// IsValid checks if the asset type is valid
func (at AssetTypeString) IsValid() bool {
	switch at {
	case STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL:
		return true
	default:
		return false
	}
}

// IsIslamic returns true if the asset type is Islamic/Sharia-compliant
func (at AssetTypeString) IsIslamic() bool {
	switch at {
	case SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL:
		return true
	default:
		return false
	}
}

// String returns the string representation of AssetTypeString
func (at AssetTypeString) String() string {
	return string(at)
}

// ParseAssetType parses a string into AssetTypeString
func ParseAssetType(s string) (AssetTypeString, error) {
	assetType := AssetTypeString(strings.ToUpper(s))
	if !assetType.IsValid() {
		return "", fmt.Errorf("invalid asset type: %s", s)
	}
	return assetType, nil
}

// GetAllAssetTypes returns all valid asset types
func GetAllAssetTypes() []AssetTypeString {
	return []AssetTypeString{
		STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL,
	}
}

// GetTraditionalAssetTypes returns traditional (non-Islamic) asset types
func GetTraditionalAssetTypes() []AssetTypeString {
	return []AssetTypeString{
		STOCK, BOND, ETF, REIT, MUTUAL_FUND, CRYPTO, FOREX, COMMODITY,
	}
}

// GetIslamicAssetTypes returns Islamic/Sharia-compliant asset types
func GetIslamicAssetTypes() []AssetTypeString {
	return []AssetTypeString{
		SUKUK, ISLAMIC_FUND, SHARIA_STOCK, ISLAMIC_ETF, ISLAMIC_REIT, TAKAFUL,
	}
}
