package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
)

// User represents a user in the system
type User struct {
	ID        string    `json:"id" db:"id"`
	Username  string    `json:"username" db:"username"`
	Email     string    `json:"email" db:"email"`
	Password  string    `json:"-" db:"password_hash"` // Password hash, not exposed in JSON
	FirstName string    `json:"first_name" db:"first_name"`
	LastName  string    `json:"last_name" db:"last_name"`
	Role      string    `json:"role" db:"role"`
	Active    bool      `json:"active" db:"active"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	LastLogin time.Time `json:"last_login,omitempty" db:"last_login"`
}

// NewUser creates a new user with the given details
func NewUser(username, email, password, firstName, lastName, role string) (*User, error) {
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	return &User{
		Username:  username,
		Email:     email,
		Password:  string(hashedPassword),
		FirstName: firstName,
		LastName:  lastName,
		Role:      role,
		Active:    true,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// CheckPassword checks if the provided password matches the stored hash
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(password))
	return err == nil
}

// UpdatePassword updates the user's password
func (u *User) UpdatePassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	u.Password = string(hashedPassword)
	u.UpdatedAt = time.Now()
	return nil
}

// UserRole represents a role in the system
type UserRole string

// User roles
const (
	RoleAdmin    UserRole = "admin"
	RoleTrader   UserRole = "trader"
	RoleAnalyst  UserRole = "analyst"
	RoleReadOnly UserRole = "readonly"
)

// Permission represents a permission in the system
type Permission string

// Permissions
const (
	PermissionReadMarketData  Permission = "read:market_data"
	PermissionWriteMarketData Permission = "write:market_data"
	PermissionReadOrders      Permission = "read:orders"
	PermissionWriteOrders     Permission = "write:orders"
	PermissionReadRisk        Permission = "read:risk"
	PermissionWriteRisk       Permission = "write:risk"
	PermissionReadStrategy    Permission = "read:strategy"
	PermissionWriteStrategy   Permission = "write:strategy"
	PermissionReadUsers       Permission = "read:users"
	PermissionWriteUsers      Permission = "write:users"
	PermissionReadSystem      Permission = "read:system"
	PermissionWriteSystem     Permission = "write:system"
)

// RolePermissions maps roles to permissions
var RolePermissions = map[UserRole][]Permission{
	RoleAdmin: {
		PermissionReadMarketData, PermissionWriteMarketData,
		PermissionReadOrders, PermissionWriteOrders,
		PermissionReadRisk, PermissionWriteRisk,
		PermissionReadStrategy, PermissionWriteStrategy,
		PermissionReadUsers, PermissionWriteUsers,
		PermissionReadSystem, PermissionWriteSystem,
	},
	RoleTrader: {
		PermissionReadMarketData,
		PermissionReadOrders, PermissionWriteOrders,
		PermissionReadRisk,
		PermissionReadStrategy, PermissionWriteStrategy,
	},
	RoleAnalyst: {
		PermissionReadMarketData,
		PermissionReadOrders,
		PermissionReadRisk,
		PermissionReadStrategy,
	},
	RoleReadOnly: {
		PermissionReadMarketData,
		PermissionReadOrders,
		PermissionReadRisk,
		PermissionReadStrategy,
	},
}
