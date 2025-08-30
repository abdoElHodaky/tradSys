package trade

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Trade represents a trade in the trading system
type Trade struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	OrderID      string    `gorm:"type:varchar(36);index" json:"order_id"`
	UserID       string    `gorm:"type:varchar(36);index" json:"user_id"`
	AccountID    string    `gorm:"type:varchar(36);index" json:"account_id"`
	Symbol       string    `gorm:"type:varchar(20);index" json:"symbol"`
	Side         string    `gorm:"type:varchar(10);index" json:"side"`
	Quantity     float64   `gorm:"type:decimal(20,8)" json:"quantity"`
	Price        float64   `gorm:"type:decimal(20,8)" json:"price"`
	Fee          float64   `gorm:"type:decimal(20,8)" json:"fee"`
	FeeCurrency  string    `gorm:"type:varchar(10)" json:"fee_currency"`
	Timestamp    time.Time `gorm:"index" json:"timestamp"`
	ExchangeID   string    `gorm:"type:varchar(50);index" json:"exchange_id"`
	ExternalID   string    `gorm:"type:varchar(50);index" json:"external_id"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a new trade
func (t *Trade) BeforeCreate(tx *gorm.DB) error {
	if t.ID == "" {
		t.ID = uuid.New().String()
	}
	return nil
}

