package types

import (
	"fmt"
	"strings"
)

// AssetType represents different types of financial assets
type AssetType string

const (
	AssetTypeStock      AssetType = "STOCK"
	AssetTypeREIT       AssetType = "REIT"
	AssetTypeMutualFund AssetType = "MUTUAL_FUND"
	AssetTypeETF        AssetType = "ETF"
	AssetTypeBond       AssetType = "BOND"
	AssetTypeCrypto     AssetType = "CRYPTO"
	AssetTypeForex      AssetType = "FOREX"
	AssetTypeCommodity  AssetType = "COMMODITY"
)

// String returns the string representation of AssetType
func (at AssetType) String() string {
	return string(at)
}

// IsValid checks if the asset type is valid
func (at AssetType) IsValid() bool {
	switch at {
	case AssetTypeStock, AssetTypeREIT, AssetTypeMutualFund, AssetTypeETF,
		AssetTypeBond, AssetTypeCrypto, AssetTypeForex, AssetTypeCommodity:
		return true
	default:
		return false
	}
}

// FromString converts a string to AssetType
func AssetTypeFromString(s string) (AssetType, error) {
	at := AssetType(strings.ToUpper(s))
	if !at.IsValid() {
		return "", fmt.Errorf("invalid asset type: %s", s)
	}
	return at, nil
}

// GetAllAssetTypes returns all valid asset types
func GetAllAssetTypes() []AssetType {
	return []AssetType{
		AssetTypeStock,
		AssetTypeREIT,
		AssetTypeMutualFund,
		AssetTypeETF,
		AssetTypeBond,
		AssetTypeCrypto,
		AssetTypeForex,
		AssetTypeCommodity,
	}
}

// AssetMetadata represents metadata for different asset types
type AssetMetadata struct {
	// Common fields
	Sector   string `json:"sector,omitempty"`
	Industry string `json:"industry,omitempty"`
	Country  string `json:"country,omitempty"`
	Currency string `json:"currency,omitempty"`
	Exchange string `json:"exchange,omitempty"`

	// Asset-specific metadata stored as key-value pairs
	Attributes map[string]interface{} `json:"attributes,omitempty"`
}

// AssetInfo combines asset type with its metadata
type AssetInfo struct {
	Symbol   string        `json:"symbol"`
	Type     AssetType     `json:"type"`
	Metadata AssetMetadata `json:"metadata"`
}

// Asset-specific attribute keys
const (
	// REIT attributes
	AttrREITType       = "reit_type"       // Equity, Mortgage, Hybrid
	AttrPropertySector = "property_sector" // Residential, Commercial, Industrial
	AttrDividendYield  = "dividend_yield"
	AttrFFO            = "ffo" // Funds From Operations
	AttrNAVPerShare    = "nav_per_share"

	// Mutual Fund attributes
	AttrFundFamily    = "fund_family"
	AttrExpenseRatio  = "expense_ratio"
	AttrMinInvestment = "min_investment"
	AttrFundManager   = "fund_manager"
	AttrInceptionDate = "inception_date"
	AttrAssetClass    = "asset_class" // Growth, Value, Blend

	// ETF attributes
	AttrUnderlyingIndex = "underlying_index"
	AttrTrackingError   = "tracking_error"
	AttrCreationUnit    = "creation_unit"
	AttrPremiumDiscount = "premium_discount"

	// Bond attributes
	AttrMaturityDate    = "maturity_date"
	AttrCouponRate      = "coupon_rate"
	AttrYieldToMaturity = "yield_to_maturity"
	AttrCreditRating    = "credit_rating"
	AttrIssuer          = "issuer"
	AttrBondType        = "bond_type" // Government, Corporate, Municipal

	// Stock attributes
	AttrMarketCap    = "market_cap"
	AttrPERatio      = "pe_ratio"
	AttrDividendRate = "dividend_rate"
	AttrBeta         = "beta"

	// Crypto attributes
	AttrBlockchain        = "blockchain"
	AttrConsensus         = "consensus_mechanism"
	AttrMaxSupply         = "max_supply"
	AttrCirculatingSupply = "circulating_supply"
)

// GetAssetSpecificAttributes returns the relevant attributes for an asset type
func (at AssetType) GetRelevantAttributes() []string {
	switch at {
	case AssetTypeREIT:
		return []string{AttrREITType, AttrPropertySector, AttrDividendYield, AttrFFO, AttrNAVPerShare}
	case AssetTypeMutualFund:
		return []string{AttrFundFamily, AttrExpenseRatio, AttrMinInvestment, AttrFundManager, AttrInceptionDate, AttrAssetClass}
	case AssetTypeETF:
		return []string{AttrUnderlyingIndex, AttrTrackingError, AttrCreationUnit, AttrPremiumDiscount}
	case AssetTypeBond:
		return []string{AttrMaturityDate, AttrCouponRate, AttrYieldToMaturity, AttrCreditRating, AttrIssuer, AttrBondType}
	case AssetTypeStock:
		return []string{AttrMarketCap, AttrPERatio, AttrDividendRate, AttrBeta}
	case AssetTypeCrypto:
		return []string{AttrBlockchain, AttrConsensus, AttrMaxSupply, AttrCirculatingSupply}
	default:
		return []string{}
	}
}

// RequiresSpecialHandling returns true if the asset type requires special trading logic
func (at AssetType) RequiresSpecialHandling() bool {
	switch at {
	case AssetTypeMutualFund, AssetTypeREIT:
		return true // NAV-based pricing, dividend distributions
	default:
		return false
	}
}

// GetTradingHours returns typical trading hours for the asset type
func (at AssetType) GetTradingHours() string {
	switch at {
	case AssetTypeStock, AssetTypeETF, AssetTypeREIT:
		return "09:30-16:00 EST" // US market hours
	case AssetTypeMutualFund:
		return "16:00 EST" // End of day pricing
	case AssetTypeCrypto:
		return "24/7"
	case AssetTypeForex:
		return "24/5" // 24 hours, 5 days a week
	case AssetTypeBond:
		return "08:00-17:00 EST" // Bond market hours
	default:
		return "Market dependent"
	}
}
