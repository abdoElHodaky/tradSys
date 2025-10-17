package auth

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Middleware provides authentication middleware
type Middleware struct {
	jwtService *JWTService
	logger     *zap.Logger
}

// NewMiddleware creates a new authentication middleware
func NewMiddleware(jwtService *JWTService, logger *zap.Logger) *Middleware {
	return &Middleware{
		jwtService: jwtService,
		logger:     logger,
	}
}

// JWTAuth is a middleware that validates JWT tokens
func (m *Middleware) JWTAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header is required"})
			c.Abort()
			return
		}

		// Check if the Authorization header has the correct format
		if !strings.HasPrefix(authHeader, "Bearer ") {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header must be in the format 'Bearer {token}'"})
			c.Abort()
			return
		}

		// Extract the token
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		// Validate the token
		claims, err := m.jwtService.ValidateToken(tokenString)
		if err != nil {
			m.logger.Error("Failed to validate token", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid or expired token"})
			c.Abort()
			return
		}

		// Set the claims in the context
		c.Set("userID", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("role", claims.Role)

		c.Next()
	}
}

// RoleAuth is a middleware that checks if the user has the required role
func (m *Middleware) RoleAuth(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "User role not found"})
			c.Abort()
			return
		}

		if role != requiredRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// LoginHandler handles user login
func (m *Middleware) LoginHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var loginRequest struct {
			Username string `json:"username" binding:"required"`
			Password string `json:"password" binding:"required"`
		}

		if err := c.ShouldBindJSON(&loginRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Implement actual user authentication logic
		// For now, this is a placeholder that accepts any credentials
		if loginRequest.Username == "" || loginRequest.Password == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
			return
		}

		// Generate JWT token
		token, err := m.jwtService.GenerateToken(loginRequest.Username, "user", "1")
		if err != nil {
			m.logger.Error("Failed to generate token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": token,
			"user": gin.H{
				"username": loginRequest.Username,
				"role":     "user",
			},
		})
	}
}

// RefreshHandler handles token refresh
func (m *Middleware) RefreshHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		var refreshRequest struct {
			RefreshToken string `json:"refresh_token" binding:"required"`
		}

		if err := c.ShouldBindJSON(&refreshRequest); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// TODO: Implement actual refresh token validation
		// For now, this is a placeholder
		claims, err := m.jwtService.ValidateToken(refreshRequest.RefreshToken)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token"})
			return
		}

		// Generate new access token
		newToken, err := m.jwtService.GenerateToken(claims.Username, claims.Role, claims.UserID)
		if err != nil {
			m.logger.Error("Failed to generate new token", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"token": newToken,
		})
	}
}

// AuthRequired is a middleware that requires authentication
func (m *Middleware) AuthRequired() gin.HandlerFunc {
	return m.JWTAuth()
}

// AdminRequired is a middleware that requires admin role
func (m *Middleware) AdminRequired() gin.HandlerFunc {
	return gin.HandlerFunc(func(c *gin.Context) {
		// First check authentication
		m.JWTAuth()(c)
		if c.IsAborted() {
			return
		}

		// Then check admin role
		m.RoleAuth("admin")(c)
	})
}
