package security

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/time/rate"
)

// HFTSecurityManager manages security for HFT applications
type HFTSecurityManager struct {
	jwtSecret   []byte
	tokenExpiry time.Duration
	rateLimiter *rate.Limiter
	enableTLS   bool
	tlsCertFile string
	tlsKeyFile  string
}

// SecurityConfig contains security configuration
type SecurityConfig struct {
	JWTSecret   string          `yaml:"jwt_secret"`
	TokenExpiry time.Duration   `yaml:"token_expiry" default:"24h"`
	EnableTLS   bool            `yaml:"enable_tls" default:"false"`
	TLSCertFile string          `yaml:"tls_cert_file"`
	TLSKeyFile  string          `yaml:"tls_key_file"`
	RateLimit   RateLimitConfig `yaml:"rate_limit"`
}

// RateLimitConfig contains rate limiting configuration
type RateLimitConfig struct {
	RequestsPerSecond int `yaml:"requests_per_second" default:"1000"`
	BurstSize         int `yaml:"burst_size" default:"100"`
}

// Claims represents JWT claims
type Claims struct {
	UserID   string   `json:"user_id"`
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// User represents a user in the system
type User struct {
	ID           string    `json:"id"`
	Username     string    `json:"username"`
	PasswordHash string    `json:"-"` // Never serialize password hash
	Roles        []string  `json:"roles"`
	CreatedAt    time.Time `json:"created_at"`
	LastLogin    time.Time `json:"last_login"`
	IsActive     bool      `json:"is_active"`
}

// NewHFTSecurityManager creates a new security manager
func NewHFTSecurityManager(config *SecurityConfig) (*HFTSecurityManager, error) {
	if config.JWTSecret == "" {
		return nil, fmt.Errorf("JWT secret is required")
	}

	// Create rate limiter
	rateLimiter := rate.NewLimiter(
		rate.Limit(config.RateLimit.RequestsPerSecond),
		config.RateLimit.BurstSize,
	)

	return &HFTSecurityManager{
		jwtSecret:   []byte(config.JWTSecret),
		tokenExpiry: config.TokenExpiry,
		rateLimiter: rateLimiter,
		enableTLS:   config.EnableTLS,
		tlsCertFile: config.TLSCertFile,
		tlsKeyFile:  config.TLSKeyFile,
	}, nil
}

// GenerateToken generates a JWT token for a user
func (sm *HFTSecurityManager) GenerateToken(user *User) (string, error) {
	now := time.Now()
	claims := &Claims{
		UserID:   user.ID,
		Username: user.Username,
		Roles:    user.Roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(now.Add(sm.tokenExpiry)),
			IssuedAt:  jwt.NewNumericDate(now),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    "hft-trading-system",
			Subject:   user.ID,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(sm.jwtSecret)
}

// ValidateToken validates a JWT token and returns claims
func (sm *HFTSecurityManager) ValidateToken(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return sm.jwtSecret, nil
	})

	if err != nil {
		return nil, err
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// HashPassword hashes a password using bcrypt
func (sm *HFTSecurityManager) HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func (sm *HFTSecurityManager) VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateSecureToken generates a cryptographically secure random token
func (sm *HFTSecurityManager) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// AuthMiddleware provides JWT authentication middleware
func (sm *HFTSecurityManager) AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		// Check Bearer prefix
		tokenParts := strings.SplitN(authHeader, " ", 2)
		if len(tokenParts) != 2 || tokenParts[0] != "Bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid authorization header format"})
			c.Abort()
			return
		}

		// Validate token
		claims, err := sm.ValidateToken(tokenParts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		// Store claims in context
		c.Set("user_id", claims.UserID)
		c.Set("username", claims.Username)
		c.Set("roles", claims.Roles)
		c.Set("claims", claims)

		c.Next()
	}
}

// RoleMiddleware provides role-based authorization
func (sm *HFTSecurityManager) RoleMiddleware(requiredRoles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get user roles from context
		rolesInterface, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusForbidden, gin.H{"error": "No roles found"})
			c.Abort()
			return
		}

		userRoles, ok := rolesInterface.([]string)
		if !ok {
			c.JSON(http.StatusForbidden, gin.H{"error": "Invalid roles format"})
			c.Abort()
			return
		}

		// Check if user has any of the required roles
		hasRole := false
		for _, requiredRole := range requiredRoles {
			for _, userRole := range userRoles {
				if userRole == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitMiddleware provides rate limiting
func (sm *HFTSecurityManager) RateLimitMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.rateLimiter.Allow() {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"error":   "rate_limit_exceeded",
				"message": "Too many requests",
			})
			c.Abort()
			return
		}
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func (sm *HFTSecurityManager) SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Security headers
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")

		// Remove server information
		c.Header("Server", "")

		c.Next()
	}
}

// InputValidationMiddleware provides input validation and sanitization
func (sm *HFTSecurityManager) InputValidationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check content length
		if c.Request.ContentLength > 10*1024*1024 { // 10MB limit
			c.JSON(http.StatusRequestEntityTooLarge, gin.H{
				"error": "request_too_large",
			})
			c.Abort()
			return
		}

		// Validate content type for POST/PUT requests
		if c.Request.Method == "POST" || c.Request.Method == "PUT" {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" && !strings.Contains(contentType, "application/json") &&
				!strings.Contains(contentType, "application/x-www-form-urlencoded") &&
				!strings.Contains(contentType, "multipart/form-data") {
				c.JSON(http.StatusUnsupportedMediaType, gin.H{
					"error": "unsupported_media_type",
				})
				c.Abort()
				return
			}
		}

		c.Next()
	}
}

// AuditMiddleware provides audit logging
func (sm *HFTSecurityManager) AuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// Process request
		c.Next()

		// Log audit information
		duration := time.Since(start)
		userID, _ := c.Get("user_id")

		// In production, this would write to an audit log
		fmt.Printf("[AUDIT] %s %s %d %v user=%v ip=%s\n",
			c.Request.Method,
			c.Request.URL.Path,
			c.Writer.Status(),
			duration,
			userID,
			c.ClientIP(),
		)
	}
}

// ValidateInput validates and sanitizes input strings
func (sm *HFTSecurityManager) ValidateInput(input string, maxLength int) (string, error) {
	// Check length
	if len(input) > maxLength {
		return "", fmt.Errorf("input too long")
	}

	// Basic sanitization - remove null bytes and control characters
	sanitized := strings.Map(func(r rune) rune {
		if r == 0 || (r < 32 && r != '\t' && r != '\n' && r != '\r') {
			return -1
		}
		return r
	}, input)

	return sanitized, nil
}

// SecureCompare performs constant-time string comparison
func (sm *HFTSecurityManager) SecureCompare(a, b string) bool {
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

// IsValidRole checks if a role is valid
func (sm *HFTSecurityManager) IsValidRole(role string) bool {
	validRoles := map[string]bool{
		"admin":  true,
		"trader": true,
		"viewer": true,
	}
	return validRoles[role]
}

// GetUserFromContext extracts user information from Gin context
func (sm *HFTSecurityManager) GetUserFromContext(c *gin.Context) (*User, error) {
	userID, exists := c.Get("user_id")
	if !exists {
		return nil, fmt.Errorf("user not found in context")
	}

	username, _ := c.Get("username")
	roles, _ := c.Get("roles")

	user := &User{
		ID:       userID.(string),
		Username: username.(string),
		Roles:    roles.([]string),
		IsActive: true,
	}

	return user, nil
}

// RequireHTTPS middleware redirects HTTP to HTTPS
func (sm *HFTSecurityManager) RequireHTTPS() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sm.enableTLS {
			c.Next()
			return
		}

		if c.Request.Header.Get("X-Forwarded-Proto") != "https" &&
			c.Request.TLS == nil {
			httpsURL := "https://" + c.Request.Host + c.Request.RequestURI
			c.Redirect(http.StatusMovedPermanently, httpsURL)
			c.Abort()
			return
		}

		c.Next()
	}
}

// Global security manager instance
var GlobalSecurityManager *HFTSecurityManager

// InitSecurityManager initializes the global security manager
func InitSecurityManager(config *SecurityConfig) error {
	var err error
	GlobalSecurityManager, err = NewHFTSecurityManager(config)
	return err
}
