package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/trading/types"
	"gorm.io/gorm"
)

// AssetMetadata represents metadata for different asset types in the database
type AssetMetadata struct {
	gorm.Model
	Symbol      string                 `gorm:"uniqueIndex;not null;size:50" json:"symbol"`
	AssetType   types.AssetType        `gorm:"not null;size:20;index" json:"asset_type"`
	Sector      string                 `gorm:"size:100;index" json:"sector,omitempty"`
	Industry    string                 `gorm:"size:100" json:"industry,omitempty"`
	Country     string                 `gorm:"size:10" json:"country,omitempty"`
	Currency    string                 `gorm:"size:10" json:"currency,omitempty"`
	Exchange    string                 `gorm:"size:50;index" json:"exchange,omitempty"`
	Attributes  AssetAttributes        `gorm:"type:text" json:"attributes,omitempty"`
	IsActive    bool                   `gorm:"default:true;index" json:"is_active"`
	LastUpdated time.Time              `gorm:"autoUpdateTime" json:"last_updated"`
}

// AssetAttributes is a custom type for storing asset-specific attributes as JSON
type AssetAttributes map[string]interface{}

// Value implements the driver.Valuer interface for database storage
func (a AssetAttributes) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

// Scan implements the sql.Scanner interface for database retrieval
func (a *AssetAttributes) Scan(value interface{}) error {
	if value == nil {
		*a = make(AssetAttributes)
		return nil
	}

	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return fmt.Errorf("cannot scan %T into AssetAttributes", value)
	}
}

// GetAttribute retrieves a specific attribute value
func (a AssetAttributes) GetAttribute(key string) (interface{}, bool) {
	if a == nil {
		return nil, false
	}
	value, exists := a[key]
	return value, exists
}

// SetAttribute sets a specific attribute value
func (a AssetAttributes) SetAttribute(key string, value interface{}) {
	if a == nil {
		a = make(AssetAttributes)
	}
	a[key] = value
}

// GetStringAttribute retrieves a string attribute
func (a AssetAttributes) GetStringAttribute(key string) (string, bool) {
	if value, exists := a.GetAttribute(key); exists {
		if str, ok := value.(string); ok {
			return str, true
		}
	}
	return "", false
}

// GetFloatAttribute retrieves a float64 attribute
func (a AssetAttributes) GetFloatAttribute(key string) (float64, bool) {
	if value, exists := a.GetAttribute(key); exists {
		switch v := value.(type) {
		case float64:
			return v, true
		case float32:
			return float64(v), true
		case int:
			return float64(v), true
		case int64:
			return float64(v), true
		}
	}
	return 0, false
}

// GetTimeAttribute retrieves a time.Time attribute
func (a AssetAttributes) GetTimeAttribute(key string) (time.Time, bool) {
	if value, exists := a.GetAttribute(key); exists {
		if timeStr, ok := value.(string); ok {
			if t, err := time.Parse(time.RFC3339, timeStr); err == nil {
				return t, true
			}
		}
	}
	return time.Time{}, false
}

// AssetConfiguration represents configuration settings for different asset types
type AssetConfiguration struct {
	gorm.Model
	AssetType           types.AssetType `gorm:"uniqueIndex;not null;size:20" json:"asset_type"`
	TradingEnabled      bool            `gorm:"default:true" json:"trading_enabled"`
	MinOrderSize        float64         `gorm:"type:decimal(20,8)" json:"min_order_size"`
	MaxOrderSize        float64         `gorm:"type:decimal(20,8)" json:"max_order_size"`
	PriceIncrement      float64         `gorm:"type:decimal(20,8)" json:"price_increment"`
	QuantityIncrement   float64         `gorm:"type:decimal(20,8)" json:"quantity_increment"`
	TradingHours        string          `gorm:"size:100" json:"trading_hours"`
	SettlementDays      int             `gorm:"default:2" json:"settlement_days"`
	RequiresApproval    bool            `gorm:"default:false" json:"requires_approval"`
	RiskMultiplier      float64         `gorm:"type:decimal(10,4);default:1.0" json:"risk_multiplier"`
	Configuration       AssetAttributes `gorm:"type:text" json:"configuration,omitempty"`
}

// AssetPricing represents pricing information for assets
type AssetPricing struct {
	gorm.Model
	Symbol          string          `gorm:"index;not null;size:50" json:"symbol"`
	AssetType       types.AssetType `gorm:"not null;size:20;index" json:"asset_type"`
	Price           float64         `gorm:"type:decimal(20,8);not null" json:"price"`
	BidPrice        float64         `gorm:"type:decimal(20,8)" json:"bid_price,omitempty"`
	AskPrice        float64         `gorm:"type:decimal(20,8)" json:"ask_price,omitempty"`
	Volume          float64         `gorm:"type:decimal(20,8)" json:"volume,omitempty"`
	High24h         float64         `gorm:"type:decimal(20,8)" json:"high_24h,omitempty"`
	Low24h          float64         `gorm:"type:decimal(20,8)" json:"low_24h,omitempty"`
	Change24h       float64         `gorm:"type:decimal(20,8)" json:"change_24h,omitempty"`
	ChangePercent24h float64        `gorm:"type:decimal(10,4)" json:"change_percent_24h,omitempty"`
	MarketCap       float64         `gorm:"type:decimal(30,8)" json:"market_cap,omitempty"`
	Timestamp       time.Time       `gorm:"not null;index" json:"timestamp"`
	Source          string          `gorm:"size:50" json:"source,omitempty"`
}

// AssetDividend represents dividend information for dividend-paying assets
type AssetDividend struct {
	gorm.Model
	Symbol        string          `gorm:"index;not null;size:50" json:"symbol"`
	AssetType     types.AssetType `gorm:"not null;size:20;index" json:"asset_type"`
	ExDate        time.Time       `gorm:"not null;index" json:"ex_date"`
	PayDate       time.Time       `gorm:"not null" json:"pay_date"`
	RecordDate    time.Time       `json:"record_date,omitempty"`
	Amount        float64         `gorm:"type:decimal(20,8);not null" json:"amount"`
	Currency      string          `gorm:"size:10" json:"currency,omitempty"`
	DividendType  string          `gorm:"size:20" json:"dividend_type,omitempty"` // Regular, Special, etc.
	Frequency     string          `gorm:"size:20" json:"frequency,omitempty"`     // Monthly, Quarterly, etc.
	YieldPercent  float64         `gorm:"type:decimal(10,4)" json:"yield_percent,omitempty"`
}

// TableName specifies the table name for AssetMetadata
func (AssetMetadata) TableName() string {
	return "asset_metadata"
}

// TableName specifies the table name for AssetConfiguration
func (AssetConfiguration) TableName() string {
	return "asset_configurations"
}

// TableName specifies the table name for AssetPricing
func (AssetPricing) TableName() string {
	return "asset_pricing"
}

// TableName specifies the table name for AssetDividend
func (AssetDividend) TableName() string {
	return "asset_dividends"
}

// BeforeCreate hook for AssetMetadata
func (am *AssetMetadata) BeforeCreate(tx *gorm.DB) error {
	if am.Attributes == nil {
		am.Attributes = make(AssetAttributes)
	}
	return nil
}

// BeforeCreate hook for AssetConfiguration
func (ac *AssetConfiguration) BeforeCreate(tx *gorm.DB) error {
	if ac.Configuration == nil {
		ac.Configuration = make(AssetAttributes)
	}
	return nil
}

// IsREIT returns true if the asset is a REIT
func (am *AssetMetadata) IsREIT() bool {
	return am.AssetType == types.AssetTypeREIT
}

// IsMutualFund returns true if the asset is a mutual fund
func (am *AssetMetadata) IsMutualFund() bool {
	return am.AssetType == types.AssetTypeMutualFund
}

// IsETF returns true if the asset is an ETF
func (am *AssetMetadata) IsETF() bool {
	return am.AssetType == types.AssetTypeETF
}

// IsBond returns true if the asset is a bond
func (am *AssetMetadata) IsBond() bool {
	return am.AssetType == types.AssetTypeBond
}

// RequiresSpecialHandling returns true if the asset requires special trading logic
func (am *AssetMetadata) RequiresSpecialHandling() bool {
	return am.AssetType.RequiresSpecialHandling()
}

// GetTradingHours returns the trading hours for this asset type
func (am *AssetMetadata) GetTradingHours() string {
	return am.AssetType.GetTradingHours()
}
