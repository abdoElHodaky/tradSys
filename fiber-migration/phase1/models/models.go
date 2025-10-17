package models

import "time"

// Pair represents a trading pair
type Pair struct {
	ID         string    `json:"id" gorm:"primaryKey"`
	Symbol     string    `json:"symbol" gorm:"unique;not null"`
	BaseAsset  string    `json:"base_asset" gorm:"not null"`
	QuoteAsset string    `json:"quote_asset" gorm:"not null"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// Order represents a trading order
type Order struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Symbol    string    `json:"symbol" gorm:"not null"`
	Side      string    `json:"side" gorm:"not null"` // buy/sell
	Quantity  float64   `json:"quantity" gorm:"not null"`
	Price     float64   `json:"price"`
	Status    string    `json:"status" gorm:"default:'pending'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// User represents a system user
type User struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	Username  string    `json:"username" gorm:"unique;not null"`
	Email     string    `json:"email" gorm:"unique;not null"`
	Role      string    `json:"role" gorm:"default:'user'"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
