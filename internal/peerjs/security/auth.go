package security

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"go.uber.org/zap"
)

// PeerAuthConfig contains configuration for peer authentication
type PeerAuthConfig struct {
	// JWTSecret is the secret key for JWT tokens
	JWTSecret string
	
	// JWTIssuer is the issuer for JWT tokens
	JWTIssuer string
	
	// JWTAudience is the audience for JWT tokens
	JWTAudience string
	
	// TokenExpiration is the expiration time for tokens
	TokenExpiration time.Duration
	
	// AllowedOrigins is a list of allowed origins for CORS
	AllowedOrigins []string
	
	// RequireAuthentication requires authentication for all connections
	RequireAuthentication bool
	
	// EnableRateLimiting enables rate limiting for connections
	EnableRateLimiting bool
	
	// RateLimitConfig is the configuration for rate limiting
	RateLimitConfig RateLimitConfig
}

// RateLimitConfig contains configuration for rate limiting
type RateLimitConfig struct {
	// MaxRequestsPerMinute is the maximum number of requests per minute
	MaxRequestsPerMinute int
	
	// MaxConnectionsPerIP is the maximum number of connections per IP
	MaxConnectionsPerIP int
	
	// IPBlockDuration is the duration to block an IP after exceeding limits
	IPBlockDuration time.Duration
}

// DefaultPeerAuthConfig returns the default configuration
func DefaultPeerAuthConfig() PeerAuthConfig {
	return PeerAuthConfig{
		JWTSecret:            "change-me-in-production", // Should be overridden in production
		JWTIssuer:            "tradsys-peerjs",
		JWTAudience:          "tradsys-peers",
		TokenExpiration:      24 * time.Hour,
		AllowedOrigins:       []string{"*"}, // Should be restricted in production
		RequireAuthentication: true,
		EnableRateLimiting:    true,
		RateLimitConfig: RateLimitConfig{
			MaxRequestsPerMinute: 60,
			MaxConnectionsPerIP:  5,
			IPBlockDuration:      15 * time.Minute,
		},
	}
}

// PeerAuthenticator handles authentication for PeerJS connections
type PeerAuthenticator struct {
	config PeerAuthConfig
	logger *zap.Logger
	
	// Rate limiting
	requestCounts   map[string]int
	connectionCounts map[string]int
	blockedIPs      map[string]time.Time
}

// NewPeerAuthenticator creates a new peer authenticator
func NewPeerAuthenticator(config PeerAuthConfig, logger *zap.Logger) *PeerAuthenticator {
	return &PeerAuthenticator{
		config:           config,
		logger:           logger,
		requestCounts:    make(map[string]int),
		connectionCounts: make(map[string]int),
		blockedIPs:       make(map[string]time.Time),
	}
}

// PeerClaims represents the claims in a peer JWT token
type PeerClaims struct {
	jwt.RegisteredClaims
	PeerID string `json:"peer_id"`
	Role   string `json:"role,omitempty"`
}

// GenerateToken generates a JWT token for a peer
func (a *PeerAuthenticator) GenerateToken(peerID, role string) (string, error) {
	now := time.Now()
	
	claims := PeerClaims{
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    a.config.JWTIssuer,
			Subject:   peerID,
			Audience:  jwt.ClaimStrings{a.config.JWTAudience},
			ExpiresAt: jwt.NewNumericDate(now.Add(a.config.TokenExpiration)),
			NotBefore: jwt.NewNumericDate(now),
			IssuedAt:  jwt.NewNumericDate(now),
			ID:        generateTokenID(),
		},
		PeerID: peerID,
		Role:   role,
	}
	
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	
	tokenString, err := token.SignedString([]byte(a.config.JWTSecret))
	if err != nil {
		a.logger.Error("Failed to sign token", zap.Error(err))
		return "", err
	}
	
	return tokenString, nil
}

// ValidateToken validates a JWT token
func (a *PeerAuthenticator) ValidateToken(tokenString string) (*PeerClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &PeerClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate the signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		
		return []byte(a.config.JWTSecret), nil
	})
	
	if err != nil {
		a.logger.Error("Failed to parse token", zap.Error(err))
		return nil, err
	}
	
	if !token.Valid {
		return nil, errors.New("invalid token")
	}
	
	claims, ok := token.Claims.(*PeerClaims)
	if !ok {
		return nil, errors.New("invalid claims")
	}
	
	return claims, nil
}

// AuthenticateRequest authenticates an HTTP request
func (a *PeerAuthenticator) AuthenticateRequest(r *http.Request) (*PeerClaims, error) {
	// Check if authentication is required
	if !a.config.RequireAuthentication {
		// Extract peer ID from query parameters
		peerID := r.URL.Query().Get("id")
		if peerID == "" {
			return nil, errors.New("missing peer ID")
		}
		
		// Create default claims
		return &PeerClaims{
			PeerID: peerID,
			Role:   "guest",
		}, nil
	}
	
	// Get the token from the Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return nil, errors.New("missing authorization header")
	}
	
	// Check if the header starts with "Bearer "
	if !strings.HasPrefix(authHeader, "Bearer ") {
		return nil, errors.New("invalid authorization header format")
	}
	
	// Extract the token
	tokenString := strings.TrimPrefix(authHeader, "Bearer ")
	
	// Validate the token
	return a.ValidateToken(tokenString)
}

// CheckRateLimit checks if a request exceeds rate limits
func (a *PeerAuthenticator) CheckRateLimit(r *http.Request) error {
	// Check if rate limiting is enabled
	if !a.config.EnableRateLimiting {
		return nil
	}
	
	// Get the client IP
	ip := getClientIP(r)
	
	// Check if the IP is blocked
	if blockedUntil, ok := a.blockedIPs[ip]; ok {
		if time.Now().Before(blockedUntil) {
			return fmt.Errorf("IP blocked until %s", blockedUntil.Format(time.RFC3339))
		}
		
		// Remove the IP from the blocked list
		delete(a.blockedIPs, ip)
	}
	
	// Increment the request count
	a.requestCounts[ip]++
	
	// Check if the request count exceeds the limit
	if a.requestCounts[ip] > a.config.RateLimitConfig.MaxRequestsPerMinute {
		// Block the IP
		a.blockedIPs[ip] = time.Now().Add(a.config.RateLimitConfig.IPBlockDuration)
		
		// Reset the request count
		a.requestCounts[ip] = 0
		
		return fmt.Errorf("rate limit exceeded for IP %s", ip)
	}
	
	return nil
}

// IncrementConnectionCount increments the connection count for an IP
func (a *PeerAuthenticator) IncrementConnectionCount(r *http.Request) error {
	// Check if rate limiting is enabled
	if !a.config.EnableRateLimiting {
		return nil
	}
	
	// Get the client IP
	ip := getClientIP(r)
	
	// Increment the connection count
	a.connectionCounts[ip]++
	
	// Check if the connection count exceeds the limit
	if a.connectionCounts[ip] > a.config.RateLimitConfig.MaxConnectionsPerIP {
		// Block the IP
		a.blockedIPs[ip] = time.Now().Add(a.config.RateLimitConfig.IPBlockDuration)
		
		return fmt.Errorf("connection limit exceeded for IP %s", ip)
	}
	
	return nil
}

// DecrementConnectionCount decrements the connection count for an IP
func (a *PeerAuthenticator) DecrementConnectionCount(r *http.Request) {
	// Check if rate limiting is enabled
	if !a.config.EnableRateLimiting {
		return
	}
	
	// Get the client IP
	ip := getClientIP(r)
	
	// Decrement the connection count
	if a.connectionCounts[ip] > 0 {
		a.connectionCounts[ip]--
	}
}

// CheckOrigin checks if the origin is allowed
func (a *PeerAuthenticator) CheckOrigin(r *http.Request) bool {
	// If no allowed origins are specified, allow all
	if len(a.config.AllowedOrigins) == 0 || (len(a.config.AllowedOrigins) == 1 && a.config.AllowedOrigins[0] == "*") {
		return true
	}
	
	// Get the origin
	origin := r.Header.Get("Origin")
	if origin == "" {
		// No origin header, deny the request
		return false
	}
	
	// Check if the origin is allowed
	for _, allowedOrigin := range a.config.AllowedOrigins {
		if allowedOrigin == origin {
			return true
		}
	}
	
	return false
}

// ResetRateLimits resets the rate limits
func (a *PeerAuthenticator) ResetRateLimits() {
	a.requestCounts = make(map[string]int)
	
	// Clean up expired blocked IPs
	now := time.Now()
	for ip, blockedUntil := range a.blockedIPs {
		if now.After(blockedUntil) {
			delete(a.blockedIPs, ip)
		}
	}
}

// StartRateLimitCleanup starts a goroutine to periodically clean up rate limits
func (a *PeerAuthenticator) StartRateLimitCleanup(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			a.ResetRateLimits()
		}
	}()
}

// getClientIP gets the client IP from a request
func getClientIP(r *http.Request) string {
	// Check for X-Forwarded-For header
	forwardedFor := r.Header.Get("X-Forwarded-For")
	if forwardedFor != "" {
		// X-Forwarded-For can contain multiple IPs, use the first one
		ips := strings.Split(forwardedFor, ",")
		return strings.TrimSpace(ips[0])
	}
	
	// Check for X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}
	
	// Use the remote address
	return strings.Split(r.RemoteAddr, ":")[0]
}

// generateTokenID generates a unique token ID
func generateTokenID() string {
	// Generate a random token ID
	tokenID := make([]byte, 16)
	for i := range tokenID {
		tokenID[i] = byte(time.Now().Nanosecond() % 256)
	}
	
	return base64.URLEncoding.EncodeToString(tokenID)
}

// GenerateSignature generates an HMAC signature for a message
func (a *PeerAuthenticator) GenerateSignature(message string) string {
	h := hmac.New(sha256.New, []byte(a.config.JWTSecret))
	h.Write([]byte(message))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// VerifySignature verifies an HMAC signature for a message
func (a *PeerAuthenticator) VerifySignature(message, signature string) bool {
	expectedSignature := a.GenerateSignature(message)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

