package auth

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Handlers provides HTTP handlers for authentication
type Handlers struct {
	service *Service
	logger  *zap.Logger
}

// NewHandlers creates new authentication handlers
func NewHandlers(service *Service, logger *zap.Logger) *Handlers {
	return &Handlers{
		service: service,
		logger:  logger,
	}
}

// Login handles user login requests
func (h *Handlers) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid login request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.Username == "" || req.Password == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Username and password are required",
		})
		return
	}

	// Attempt login
	response, err := h.service.Login(c.Request.Context(), &req)
	if err != nil {
		h.logger.Warn("Login failed",
			zap.String("username", req.Username),
			zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authentication failed",
		})
		return
	}

	// Return successful response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Login successful",
	})
}

// RefreshToken handles token refresh requests
func (h *Handlers) RefreshToken(c *gin.Context) {
	var req RefreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Warn("Invalid refresh request", zap.Error(err))
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "Invalid request format",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.RefreshToken == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Refresh token is required",
		})
		return
	}

	// Attempt token refresh
	response, err := h.service.RefreshToken(c.Request.Context(), &req)
	if err != nil {
		h.logger.Warn("Token refresh failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Token refresh failed",
		})
		return
	}

	// Return successful response
	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    response,
		"message": "Token refreshed successfully",
	})
}

// Logout handles user logout requests
func (h *Handlers) Logout(c *gin.Context) {
	// In a production system, you would:
	// 1. Invalidate the token in a blacklist/database
	// 2. Clear any server-side sessions
	// 3. Log the logout event

	// For now, we'll just return a success response
	// The client should discard the token
	h.logger.Info("User logout",
		zap.String("user_id", c.GetString("user_id")),
		zap.String("username", c.GetString("username")))

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Logout successful",
	})
}

// Profile returns the current user's profile
func (h *Handlers) Profile(c *gin.Context) {
	username := c.GetString("username")
	if username == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "User not authenticated",
		})
		return
	}

	user, err := h.service.GetUser(username)
	if err != nil {
		h.logger.Error("Failed to get user profile",
			zap.String("username", username),
			zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve user profile",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    user,
	})
}

// ChangePassword handles password change requests
func (h *Handlers) ChangePassword(c *gin.Context) {
	var req struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID := c.GetString("user_id")
	if userID == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// Password strength validation
	if len(req.NewPassword) < 8 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "New password must be at least 8 characters long"})
		return
	}

	// In production implementation:
	// 1. Verify current password against database
	// 2. Hash new password with bcrypt
	// 3. Update password in database
	// 4. Invalidate existing JWT tokens

	// Simulate successful password change
	c.JSON(http.StatusOK, gin.H{
		"message":   "Password changed successfully",
		"timestamp": time.Now().Unix(),
		"user_id":   userID,
	})
}

// ValidateToken is a middleware helper to validate JWT tokens
func (h *Handlers) ValidateToken(c *gin.Context) {
	authHeader := c.GetHeader("Authorization")
	if authHeader == "" {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Authorization header required",
		})
		c.Abort()
		return
	}

	// Extract token from "Bearer <token>" format
	const bearerPrefix = "Bearer "
	if len(authHeader) < len(bearerPrefix) || authHeader[:len(bearerPrefix)] != bearerPrefix {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid authorization header format",
		})
		c.Abort()
		return
	}

	tokenString := authHeader[len(bearerPrefix):]

	// Validate token
	claims, err := h.service.ValidateToken(tokenString)
	if err != nil {
		h.logger.Warn("Token validation failed", zap.Error(err))
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": "Invalid or expired token",
		})
		c.Abort()
		return
	}

	// Set user information in context
	c.Set("user_id", claims.UserID)
	c.Set("username", claims.Username)
	c.Set("role", claims.Role)

	c.Next()
}
