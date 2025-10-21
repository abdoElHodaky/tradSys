package auth

import (
	"context"
	"crypto/bcrypt"
	"errors"
	"fmt"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// User represents a user in the system
type User struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Role     string `json:"role"`
	Password string `json:"-"` // Never serialize password
	Active   bool   `json:"active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// LoginRequest represents a login request
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse represents a login response
type LoginResponse struct {
	Token        string    `json:"token"`
	RefreshToken string    `json:"refresh_token"`
	User         *User     `json:"user"`
	ExpiresAt    time.Time `json:"expires_at"`
}

// RefreshRequest represents a token refresh request
type RefreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// ServiceParams contains the parameters for creating an auth service
type ServiceParams struct {
	fx.In

	Logger *zap.Logger
	Config *config.Config
}

// Service provides authentication operations
type Service struct {
	logger     *zap.Logger
	config     *config.Config
	jwtService *JWTService
	
	// In-memory user storage for demo purposes
	// In production, this would be a database
	users map[string]*User
}

// NewService creates a new authentication service with fx dependency injection
func NewService(p ServiceParams) *Service {
	// Initialize JWT service
	jwtConfig := JWTConfig{
		SecretKey:     p.Config.JWT.SecretKey,
		TokenDuration: p.Config.JWT.TokenDuration,
		Issuer:        p.Config.JWT.Issuer,
	}
	
	service := &Service{
		logger:     p.Logger,
		config:     p.Config,
		jwtService: NewJWTService(jwtConfig),
		users:      make(map[string]*User),
	}
	
	// Initialize with default users for demo
	service.initializeDefaultUsers()
	
	return service
}

// initializeDefaultUsers creates default users for demo purposes
func (s *Service) initializeDefaultUsers() {
	// Hash default password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte("admin123"), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash default password", zap.Error(err))
		return
	}
	
	// Create default admin user
	adminUser := &User{
		ID:       "admin-001",
		Username: "admin",
		Email:    "admin@tradsys.com",
		Role:     "admin",
		Password: string(hashedPassword),
		Active:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	s.users[adminUser.Username] = adminUser
	
	// Create default trader user
	hashedTraderPassword, err := bcrypt.GenerateFromPassword([]byte("trader123"), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Error("Failed to hash trader password", zap.Error(err))
		return
	}
	
	traderUser := &User{
		ID:       "trader-001",
		Username: "trader",
		Email:    "trader@tradsys.com",
		Role:     "trader",
		Password: string(hashedTraderPassword),
		Active:   true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	
	s.users[traderUser.Username] = traderUser
	
	s.logger.Info("Initialized default users", 
		zap.Int("user_count", len(s.users)),
		zap.Strings("usernames", []string{"admin", "trader"}))
}

// Login authenticates a user and returns a JWT token
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*LoginResponse, error) {
	s.logger.Info("Login attempt", zap.String("username", req.Username))
	
	// Find user
	user, exists := s.users[req.Username]
	if !exists {
		s.logger.Warn("Login failed: user not found", zap.String("username", req.Username))
		return nil, errors.New("invalid credentials")
	}
	
	// Check if user is active
	if !user.Active {
		s.logger.Warn("Login failed: user inactive", zap.String("username", req.Username))
		return nil, errors.New("account is inactive")
	}
	
	// Verify password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		s.logger.Warn("Login failed: invalid password", zap.String("username", req.Username))
		return nil, errors.New("invalid credentials")
	}
	
	// Generate JWT token
	token, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate token", zap.Error(err))
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}
	
	// Generate refresh token (simplified - in production, store in database)
	refreshToken, err := s.jwtService.GenerateToken(user.ID, user.Username, "refresh")
	if err != nil {
		s.logger.Error("Failed to generate refresh token", zap.Error(err))
		return nil, fmt.Errorf("failed to generate refresh token: %w", err)
	}
	
	// Create response
	response := &LoginResponse{
		Token:        token,
		RefreshToken: refreshToken,
		User: &User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Active:   user.Active,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		ExpiresAt: time.Now().Add(s.config.JWT.TokenDuration),
	}
	
	s.logger.Info("Login successful", 
		zap.String("username", req.Username),
		zap.String("user_id", user.ID),
		zap.String("role", user.Role))
	
	return response, nil
}

// RefreshToken refreshes an existing JWT token
func (s *Service) RefreshToken(ctx context.Context, req *RefreshRequest) (*LoginResponse, error) {
	s.logger.Info("Token refresh attempt")
	
	// Validate refresh token
	claims, err := s.jwtService.ValidateToken(req.RefreshToken)
	if err != nil {
		s.logger.Warn("Token refresh failed: invalid token", zap.Error(err))
		return nil, errors.New("invalid refresh token")
	}
	
	// Check if it's a refresh token
	if claims.Role != "refresh" {
		s.logger.Warn("Token refresh failed: not a refresh token")
		return nil, errors.New("invalid refresh token")
	}
	
	// Find user
	user, exists := s.users[claims.Username]
	if !exists || !user.Active {
		s.logger.Warn("Token refresh failed: user not found or inactive", zap.String("username", claims.Username))
		return nil, errors.New("user not found or inactive")
	}
	
	// Generate new tokens
	newToken, err := s.jwtService.GenerateToken(user.ID, user.Username, user.Role)
	if err != nil {
		s.logger.Error("Failed to generate new token", zap.Error(err))
		return nil, fmt.Errorf("failed to generate new token: %w", err)
	}
	
	newRefreshToken, err := s.jwtService.GenerateToken(user.ID, user.Username, "refresh")
	if err != nil {
		s.logger.Error("Failed to generate new refresh token", zap.Error(err))
		return nil, fmt.Errorf("failed to generate new refresh token: %w", err)
	}
	
	// Create response
	response := &LoginResponse{
		Token:        newToken,
		RefreshToken: newRefreshToken,
		User: &User{
			ID:       user.ID,
			Username: user.Username,
			Email:    user.Email,
			Role:     user.Role,
			Active:   user.Active,
			CreatedAt: user.CreatedAt,
			UpdatedAt: user.UpdatedAt,
		},
		ExpiresAt: time.Now().Add(s.config.JWT.TokenDuration),
	}
	
	s.logger.Info("Token refresh successful", 
		zap.String("username", claims.Username),
		zap.String("user_id", user.ID))
	
	return response, nil
}

// ValidateToken validates a JWT token and returns the claims
func (s *Service) ValidateToken(tokenString string) (*JWTClaims, error) {
	return s.jwtService.ValidateToken(tokenString)
}

// GetUser returns a user by username (without password)
func (s *Service) GetUser(username string) (*User, error) {
	user, exists := s.users[username]
	if !exists {
		return nil, errors.New("user not found")
	}
	
	// Return user without password
	return &User{
		ID:       user.ID,
		Username: user.Username,
		Email:    user.Email,
		Role:     user.Role,
		Active:   user.Active,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

