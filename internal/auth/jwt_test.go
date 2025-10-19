package auth

import (
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/unified-config"
	"github.com/stretchr/testify/assert"
)

func TestJWT(t *testing.T) {
	// Load config
	_, err := config.LoadConfig("../../config")
	assert.NoError(t, err)

	// Test token generation and validation
	userID := "user123"
	username := "testuser"
	role := "admin"

	// Generate token
	token, err := GenerateToken(userID, username, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := ValidateToken(token)
	assert.NoError(t, err)
	assert.NotNil(t, claims)
	assert.Equal(t, userID, claims.UserID)
	assert.Equal(t, username, claims.Username)
	assert.Equal(t, role, claims.Role)
	assert.Equal(t, userID, claims.Subject)
	assert.Equal(t, "tradsys", claims.Issuer)

	// Check expiration
	assert.True(t, claims.ExpiresAt.Time.After(time.Now()))
	assert.True(t, claims.IssuedAt.Time.Before(time.Now()) || claims.IssuedAt.Time.Equal(time.Now()))
	assert.True(t, claims.NotBefore.Time.Before(time.Now()) || claims.NotBefore.Time.Equal(time.Now()))

	// Test invalid token
	_, err = ValidateToken("invalid.token.string")
	assert.Error(t, err)
}
