package position

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Position represents a trading position
type Position struct {
	ID           string    `gorm:"primaryKey;type:varchar(36)" json:"id"`
	UserID       string    `gorm:"type:varchar(36);index" json:"user_id"`
	AccountID    string    `gorm:"type:varchar(36);index" json:"account_id"`
	Symbol       string    `gorm:"type:varchar(20);index" json:"symbol"`
	Quantity     float64   `gorm:"type:decimal(20,8)" json:"quantity"`
	EntryPrice   float64   `gorm:"type:decimal(20,8)" json:"entry_price"`
	CurrentPrice float64   `gorm:"type:decimal(20,8)" json:"current_price"`
	UnrealizedPL float64   `gorm:"type:decimal(20,8)" json:"unrealized_pl"`
	RealizedPL   float64   `gorm:"type:decimal(20,8)" json:"realized_pl"`
	OpenTime     time.Time `json:"open_time"`
	UpdateTime   time.Time `json:"update_time"`
	CloseTime    *time.Time `json:"close_time"`
	Status       string    `gorm:"type:varchar(20);index" json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// BeforeCreate is a GORM hook that runs before creating a new position
func (p *Position) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

