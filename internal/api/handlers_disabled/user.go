package handlers

import (
	"net/http"
	"strconv"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/user"
	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"go.uber.org/zap"
)

// UserHandler handles user-related API endpoints
type UserHandler struct {
	logger    *zap.Logger
	service   *user.Service
	validator *validator.Validate
}

// NewUserHandler creates a new user handler
func NewUserHandler(logger *zap.Logger, service *user.Service) *UserHandler {
	return &UserHandler{
		logger:    logger,
		service:   service,
		validator: validator.New(),
	}
}

// RegisterRequest represents a user registration request
type RegisterRequest struct {
	Username  string `json:"username" validate:"required,min=3,max=50"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
}

// LoginRequest represents a user login request
type LoginRequest struct {
	UsernameOrEmail string `json:"username_or_email" validate:"required"`
	Password        string `json:"password" validate:"required"`
}

// UpdateUserRequest represents a user update request
type UpdateUserRequest struct {
	FirstName string `json:"first_name" validate:"required"`
	LastName  string `json:"last_name" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
}

// ChangePasswordRequest represents a password change request
type ChangePasswordRequest struct {
	CurrentPassword string `json:"current_password" validate:"required"`
	NewPassword     string `json:"new_password" validate:"required,min=8"`
}

// ResetPasswordRequest represents a password reset request
type ResetPasswordRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	NewPassword string `json:"new_password" validate:"required,min=8"`
}

// UpdateRoleRequest represents a role update request
type UpdateRoleRequest struct {
	UserID string `json:"user_id" validate:"required"`
	Role   string `json:"role" validate:"required,oneof=admin trader analyst readonly"`
}

// Register handles user registration
func (h *UserHandler) Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Register user
	user, err := h.service.RegisterUser(c.Request.Context(), req.Username, req.Email, req.Password, req.FirstName, req.LastName, string(models.RoleReadOnly))
	if err != nil {
		h.logger.Error("Failed to register user", zap.Error(err), zap.String("username", req.Username))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": "User registered successfully",
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
		},
	})
}

// Login handles user login
func (h *UserHandler) Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Login user
	token, err := h.service.LoginUser(c.Request.Context(), req.UsernameOrEmail, req.Password)
	if err != nil {
		h.logger.Error("Failed to login user", zap.Error(err), zap.String("username_or_email", req.UsernameOrEmail))
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Login successful",
		"token":   token,
	})
}

// GetProfile handles getting the current user's profile
func (h *UserHandler) GetProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// Get user
	user, err := h.service.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("Failed to get user profile", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get user profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"createdAt": user.CreatedAt,
			"lastLogin": user.LastLogin,
		},
	})
}

// UpdateProfile handles updating the current user's profile
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req UpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get user
	user, err := h.service.GetUserByID(c.Request.Context(), userID.(string))
	if err != nil {
		h.logger.Error("Failed to get user for update", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	// Update user
	user.FirstName = req.FirstName
	user.LastName = req.LastName
	user.Email = req.Email

	if err := h.service.UpdateUser(c.Request.Context(), user); err != nil {
		h.logger.Error("Failed to update user profile", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
		},
	})
}

// ChangePassword handles changing the current user's password
func (h *UserHandler) ChangePassword(c *gin.Context) {
	// Get user ID from context (set by auth middleware)
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Change password
	if err := h.service.ChangePassword(c.Request.Context(), userID.(string), req.CurrentPassword, req.NewPassword); err != nil {
		h.logger.Error("Failed to change password", zap.Error(err), zap.String("user_id", userID.(string)))
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password changed successfully",
	})
}

// ListUsers handles listing all users (admin only)
func (h *UserHandler) ListUsers(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// List users
	users, count, err := h.service.ListUsers(c.Request.Context(), page, pageSize)
	if err != nil {
		h.logger.Error("Failed to list users", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to list users"})
		return
	}

	// Map users to response format
	var userResponses []gin.H
	for _, user := range users {
		userResponses = append(userResponses, gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"role":      user.Role,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"createdAt": user.CreatedAt,
			"lastLogin": user.LastLogin,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"users": userResponses,
		"pagination": gin.H{
			"total":     count,
			"page":      page,
			"page_size": pageSize,
			"pages":     (count + pageSize - 1) / pageSize,
		},
	})
}

// ResetPassword handles resetting a user's password (admin only)
func (h *UserHandler) ResetPassword(c *gin.Context) {
	var req ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Reset password
	if err := h.service.ResetPassword(c.Request.Context(), req.UserID, req.NewPassword); err != nil {
		h.logger.Error("Failed to reset password", zap.Error(err), zap.String("user_id", req.UserID))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to reset password"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Password reset successfully",
	})
}

// UpdateRole handles updating a user's role (admin only)
func (h *UserHandler) UpdateRole(c *gin.Context) {
	var req UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Validate request
	if err := h.validator.Struct(req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update role
	if err := h.service.UpdateUserRole(c.Request.Context(), req.UserID, req.Role); err != nil {
		h.logger.Error("Failed to update role", zap.Error(err), zap.String("user_id", req.UserID), zap.String("role", req.Role))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update role"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Role updated successfully",
	})
}

// RegisterRoutes registers user routes
func (h *UserHandler) RegisterRoutes(router *gin.Engine, authMiddleware gin.HandlerFunc) {
	// Public routes
	router.POST("/api/auth/register", h.Register)
	router.POST("/api/auth/login", h.Login)

	// Authenticated routes
	authenticated := router.Group("/api")
	authenticated.Use(authMiddleware)
	{
		authenticated.GET("/users/profile", h.GetProfile)
		authenticated.PUT("/users/profile", h.UpdateProfile)
		authenticated.POST("/users/change-password", h.ChangePassword)
	}

	// Admin routes
	admin := router.Group("/api/admin")
	admin.Use(authMiddleware, auth.RoleMiddleware(string(models.RoleAdmin)))
	{
		admin.GET("/users", h.ListUsers)
		admin.POST("/users/reset-password", h.ResetPassword)
		admin.POST("/users/update-role", h.UpdateRole)
	}
}
