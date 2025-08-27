package auth

import (
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestJWT(t *testing.T) {
	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Load config
	cfg, err := config.LoadConfig("../../config", logger)
	assert.NoError(t, err)

	// Create JWT service with test configuration
	jwtService := NewJWTService(JWTConfig{
		SecretKey:     "test-secret-key",
		TokenDuration: 1 * time.Hour,
		Issuer:        "tradsys",
	})

	// Test token generation and validation
	userID := "user123"
	username := "testuser"
	role := "admin"

	// Generate token
	token, err := jwtService.GenerateToken(userID, username, role)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	// Validate token
	claims, err := jwtService.ValidateToken(token)
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
	_, err = jwtService.ValidateToken("invalid.token.string")
	assert.Error(t, err)

	// Test token refresh
	refreshedToken, err := jwtService.RefreshToken(token)
	assert.NoError(t, err)
	assert.NotEmpty(t, refreshedToken)
	assert.NotEqual(t, token, refreshedToken)

	// Validate refreshed token
	refreshedClaims, err := jwtService.ValidateToken(refreshedToken)
	assert.NoError(t, err)
	assert.NotNil(t, refreshedClaims)
	assert.Equal(t, userID, refreshedClaims.UserID)
	assert.Equal(t, username, refreshedClaims.Username)
	assert.Equal(t, role, refreshedClaims.Role)
}
