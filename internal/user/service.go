package user

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/db/models"
	"github.com/abdoElHodaky/tradSys/internal/db/repositories"
	"go.uber.org/zap"
)

// Service handles user-related operations
type Service struct {
	logger     *zap.Logger
	repository *repositories.UserRepository
}

// NewService creates a new user service
func NewService(logger *zap.Logger, repository *repositories.UserRepository) *Service {
	return &Service{
		logger:     logger,
		repository: repository,
	}
}

// RegisterUser registers a new user
func (s *Service) RegisterUser(ctx context.Context, username, email, password, firstName, lastName, role string) (*models.User, error) {
	// Validate input
	if username == "" {
		return nil, errors.New("username is required")
	}
	if email == "" {
		return nil, errors.New("email is required")
	}
	if password == "" {
		return nil, errors.New("password is required")
	}
	if role == "" {
		role = string(models.RoleReadOnly) // Default role
	}

	// Check if username already exists
	existingUser, err := s.repository.GetByUsername(ctx, username)
	if err == nil && existingUser != nil {
		return nil, errors.New("username already exists")
	}

	// Check if email already exists
	existingUser, err = s.repository.GetByEmail(ctx, email)
	if err == nil && existingUser != nil {
		return nil, errors.New("email already exists")
	}

	// Create user
	user, err := models.NewUser(username, email, password, firstName, lastName, role)
	if err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Save user to database
	if err := s.repository.Create(ctx, user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("User registered", zap.String("username", username), zap.String("role", role))

	return user, nil
}

// LoginUser authenticates a user and returns a JWT token
func (s *Service) LoginUser(ctx context.Context, usernameOrEmail, password string) (string, error) {
	// Validate input
	if usernameOrEmail == "" {
		return "", errors.New("username or email is required")
	}
	if password == "" {
		return "", errors.New("password is required")
	}

	// Find user by username or email
	var user *models.User
	var err error

	// Try username first
	user, err = s.repository.GetByUsername(ctx, usernameOrEmail)
	if err != nil {
		// Try email
		user, err = s.repository.GetByEmail(ctx, usernameOrEmail)
		if err != nil {
			return "", errors.New("invalid username/email or password")
		}
	}

	// Check password
	if !user.CheckPassword(password) {
		return "", errors.New("invalid username/email or password")
	}

	// Update last login
	if err := s.repository.UpdateLastLogin(ctx, user.ID); err != nil {
		s.logger.Warn("Failed to update last login", zap.Error(err), zap.String("user_id", user.ID))
	}

	// Create JWT service
	jwtConfig := auth.JWTConfig{
		SecretKey:     "your-secret-key", // In production, use environment variable
		TokenDuration: 24 * time.Hour,
		Issuer:        "tradsys-api",
	}
	jwtService := auth.NewJWTService(jwtConfig)
	
	// Generate JWT token
	token, err := jwtService.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	s.logger.Info("User logged in", zap.String("username", user.Username))

	return token, nil
}

// GetUserByID gets a user by ID
func (s *Service) GetUserByID(ctx context.Context, id string) (*models.User, error) {
	return s.repository.GetByID(ctx, id)
}

// GetUserByUsername gets a user by username
func (s *Service) GetUserByUsername(ctx context.Context, username string) (*models.User, error) {
	return s.repository.GetByUsername(ctx, username)
}

// UpdateUser updates a user
func (s *Service) UpdateUser(ctx context.Context, user *models.User) error {
	return s.repository.Update(ctx, user)
}

// DeleteUser deletes a user
func (s *Service) DeleteUser(ctx context.Context, id string) error {
	return s.repository.Delete(ctx, id)
}

// ListUsers lists all users with pagination
func (s *Service) ListUsers(ctx context.Context, page, pageSize int) ([]*models.User, int, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}

	offset := (page - 1) * pageSize

	users, err := s.repository.List(ctx, offset, pageSize)
	if err != nil {
		return nil, 0, err
	}

	count, err := s.repository.Count(ctx)
	if err != nil {
		return nil, 0, err
	}

	return users, count, nil
}

// ChangePassword changes a user's password
func (s *Service) ChangePassword(ctx context.Context, id, currentPassword, newPassword string) error {
	// Get user
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Check current password
	if !user.CheckPassword(currentPassword) {
		return errors.New("invalid current password")
	}

	// Update password
	if err := user.UpdatePassword(newPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Save user
	if err := s.repository.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("User password changed", zap.String("user_id", id))

	return nil
}

// ResetPassword resets a user's password (admin only)
func (s *Service) ResetPassword(ctx context.Context, id, newPassword string) error {
	// Get user
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Update password
	if err := user.UpdatePassword(newPassword); err != nil {
		return fmt.Errorf("failed to update password: %w", err)
	}

	// Save user
	if err := s.repository.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("User password reset", zap.String("user_id", id))

	return nil
}

// UpdateUserRole updates a user's role (admin only)
func (s *Service) UpdateUserRole(ctx context.Context, id, role string) error {
	// Get user
	user, err := s.repository.GetByID(ctx, id)
	if err != nil {
		return err
	}

	// Update role
	user.Role = role
	user.UpdatedAt = time.Now()

	// Save user
	if err := s.repository.Update(ctx, user); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	s.logger.Info("User role updated", zap.String("user_id", id), zap.String("role", role))

	return nil
}
