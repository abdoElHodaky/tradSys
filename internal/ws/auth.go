package ws

import (
	"errors"
	"net/http"
	"strings"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// AuthenticatedConnection represents a WebSocket connection with authentication
type AuthenticatedConnection struct {
	*websocket.Conn
	UserID   string
	Username string
	Role     string
}

// AuthenticatedUpgrader upgrades HTTP connections to WebSocket connections with authentication
type AuthenticatedUpgrader struct {
	upgrader   websocket.Upgrader
	logger     *zap.Logger
	jwtService *auth.JWTService
}

// NewAuthenticatedUpgrader creates a new authenticated upgrader
func NewAuthenticatedUpgrader(logger *zap.Logger, jwtService *auth.JWTService) *AuthenticatedUpgrader {
	return &AuthenticatedUpgrader{
		upgrader: websocket.Upgrader{
			ReadBufferSize:  1024,
			WriteBufferSize: 1024,
			CheckOrigin: func(r *http.Request) bool {
				return true // In production, implement proper origin checks
			},
		},
		logger:     logger,
		jwtService: jwtService,
	}
}

// Upgrade upgrades an HTTP connection to a WebSocket connection with authentication
func (au *AuthenticatedUpgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*AuthenticatedConnection, error) {
	// Get token from query parameter or Authorization header
	token := r.URL.Query().Get("token")
	if token == "" {
		// Try Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			au.logger.Error("Missing authentication token")
			return nil, errors.New("missing authentication token")
		}

		// Check if the header has the correct format
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			au.logger.Error("Invalid authorization header format")
			return nil, errors.New("invalid authorization header format")
		}

		token = parts[1]
	}

	// Validate token
	claims, err := au.jwtService.ValidateToken(token)
	if err != nil {
		au.logger.Error("Invalid authentication token", zap.Error(err))
		return nil, errors.New("invalid authentication token")
	}

	// Upgrade connection
	conn, err := au.upgrader.Upgrade(w, r, nil)
	if err != nil {
		au.logger.Error("Failed to upgrade connection", zap.Error(err))
		return nil, err
	}

	au.logger.Info("WebSocket connection authenticated", 
		zap.String("user_id", claims.UserID), 
		zap.String("username", claims.Username),
		zap.String("role", claims.Role))

	return &AuthenticatedConnection{
		Conn:     conn,
		UserID:   claims.UserID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

// AuthorizeConnection checks if the connection has the required role
func AuthorizeConnection(conn *AuthenticatedConnection, requiredRoles ...string) bool {
	if conn == nil {
		return false
	}

	for _, role := range requiredRoles {
		if conn.Role == role {
			return true
		}
	}

	return false
}
