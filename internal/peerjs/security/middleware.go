package security

import (
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// AuthMiddleware is a middleware for authenticating requests
type AuthMiddleware struct {
	authenticator *PeerAuthenticator
	logger        *zap.Logger
}

// NewAuthMiddleware creates a new auth middleware
func NewAuthMiddleware(authenticator *PeerAuthenticator, logger *zap.Logger) *AuthMiddleware {
	return &AuthMiddleware{
		authenticator: authenticator,
		logger:        logger,
	}
}

// Middleware returns an http.Handler middleware
func (m *AuthMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check origin
		if !m.authenticator.CheckOrigin(r) {
			m.logger.Warn("Origin not allowed",
				zap.String("origin", r.Header.Get("Origin")),
				zap.String("remote_addr", r.RemoteAddr))
			
			http.Error(w, "Origin not allowed", http.StatusForbidden)
			return
		}
		
		// Check rate limit
		if err := m.authenticator.CheckRateLimit(r); err != nil {
			m.logger.Warn("Rate limit exceeded",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr))
			
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		
		// Authenticate the request
		claims, err := m.authenticator.AuthenticateRequest(r)
		if err != nil {
			m.logger.Warn("Authentication failed",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr))
			
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		}
		
		// Check connection count
		if err := m.authenticator.IncrementConnectionCount(r); err != nil {
			m.logger.Warn("Connection limit exceeded",
				zap.Error(err),
				zap.String("remote_addr", r.RemoteAddr))
			
			http.Error(w, err.Error(), http.StatusTooManyRequests)
			return
		}
		
		// Store the claims in the request context
		ctx := r.Context()
		ctx = WithClaims(ctx, claims)
		
		// Call the next handler
		next.ServeHTTP(w, r.WithContext(ctx))
		
		// Decrement the connection count when the request is done
		m.authenticator.DecrementConnectionCount(r)
	})
}

// WithUpgradeAuth wraps a WebSocket upgrade handler with authentication
func (m *AuthMiddleware) WithUpgradeAuth(upgrader *websocket.Upgrader) *websocket.Upgrader {
	// Create a new upgrader with the same configuration
	newUpgrader := &websocket.Upgrader{
		ReadBufferSize:  upgrader.ReadBufferSize,
		WriteBufferSize: upgrader.WriteBufferSize,
		CheckOrigin: func(r *http.Request) bool {
			return m.authenticator.CheckOrigin(r)
		},
	}
	
	return newUpgrader
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string    `json:"error"`
	Message string    `json:"message"`
	Time    time.Time `json:"time"`
}

// RespondWithError responds with an error
func RespondWithError(w http.ResponseWriter, statusCode int, err error) {
	// Create the error response
	response := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: err.Error(),
		Time:    time.Now(),
	}
	
	// Set the content type
	w.Header().Set("Content-Type", "application/json")
	
	// Set the status code
	w.WriteHeader(statusCode)
	
	// Write the response
	json.NewEncoder(w).Encode(response)
}

