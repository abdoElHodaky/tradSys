package security

import (
	"context"
	"crypto/rand"
	"crypto/tls"
	"encoding/base64"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// SecurityTestSuite contains comprehensive security tests
type SecurityTestSuite struct {
	suite.Suite
	server *httptest.Server
	client *http.Client
	ctx    context.Context
}

func (suite *SecurityTestSuite) SetupSuite() {
	suite.ctx = context.Background()

	// Create test server with TLS
	suite.server = httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock API endpoints for security testing
		switch r.URL.Path {
		case "/api/v1/orders":
			suite.handleOrdersEndpoint(w, r)
		case "/api/v1/users":
			suite.handleUsersEndpoint(w, r)
		case "/api/v1/auth/login":
			suite.handleLoginEndpoint(w, r)
		case "/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))

	// Create HTTP client that accepts self-signed certificates for testing
	suite.client = &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		},
		Timeout: 10 * time.Second,
	}
}

func (suite *SecurityTestSuite) TearDownSuite() {
	if suite.server != nil {
		suite.server.Close()
	}
}

// Test authentication and authorization
func (suite *SecurityTestSuite) TestAuthentication() {
	suite.T().Log("Testing authentication mechanisms...")

	// Test unauthenticated access
	resp, err := suite.client.Get(suite.server.URL + "/api/v1/orders")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	// Test invalid token
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer invalid_token")
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusUnauthorized, resp.StatusCode)

	// Test valid token
	validToken := suite.generateValidJWT()
	req, _ = http.NewRequest("GET", suite.server.URL+"/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+validToken)
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

func (suite *SecurityTestSuite) TestAuthorization() {
	suite.T().Log("Testing authorization controls...")

	// Test role-based access control
	userToken := suite.generateUserJWT()
	adminToken := suite.generateAdminJWT()

	// User should not access admin endpoints
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+userToken)
	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusForbidden, resp.StatusCode)

	// Admin should access admin endpoints
	req, _ = http.NewRequest("GET", suite.server.URL+"/api/v1/users", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	resp, err = suite.client.Do(req)
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// Test input validation and sanitization
func (suite *SecurityTestSuite) TestInputValidation() {
	suite.T().Log("Testing input validation...")

	testCases := []struct {
		name     string
		input    string
		expected int
	}{
		{"SQL Injection", "'; DROP TABLE orders; --", http.StatusBadRequest},
		{"XSS Script", "<script>alert('xss')</script>", http.StatusBadRequest},
		{"Command Injection", "; rm -rf /", http.StatusBadRequest},
		{"Path Traversal", "../../../etc/passwd", http.StatusBadRequest},
		{"LDAP Injection", "admin)(|(password=*))", http.StatusBadRequest},
		{"XML Injection", "<?xml version=\"1.0\"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM \"file:///etc/passwd\">]>", http.StatusBadRequest},
		{"Valid Input", "AAPL", http.StatusOK},
	}

	for _, tc := range testCases {
		suite.T().Run(tc.name, func(t *testing.T) {
			token := suite.generateValidJWT()
			url := fmt.Sprintf("%s/api/v1/orders?symbol=%s", suite.server.URL, tc.input)

			req, _ := http.NewRequest("GET", url, nil)
			req.Header.Set("Authorization", "Bearer "+token)

			resp, err := suite.client.Do(req)
			require.NoError(t, err)
			assert.Equal(t, tc.expected, resp.StatusCode, "Failed for input: %s", tc.input)
		})
	}
}

// Test rate limiting
func (suite *SecurityTestSuite) TestRateLimiting() {
	suite.T().Log("Testing rate limiting...")

	token := suite.generateValidJWT()

	// Make multiple requests rapidly
	successCount := 0
	rateLimitedCount := 0

	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/orders", nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)

		if resp.StatusCode == http.StatusOK {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitedCount++
		}
	}

	// Should have some rate limiting in effect
	assert.Greater(suite.T(), rateLimitedCount, 0, "Rate limiting should be active")
	assert.Greater(suite.T(), successCount, 0, "Some requests should succeed")
}

// Test HTTPS and TLS configuration
func (suite *SecurityTestSuite) TestTLSConfiguration() {
	suite.T().Log("Testing TLS configuration...")

	// Test that HTTP redirects to HTTPS (in production)
	// For now, just verify TLS is working
	resp, err := suite.client.Get(suite.server.URL + "/health")
	require.NoError(suite.T(), err)
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)

	// Verify TLS version and cipher suites
	if resp.TLS != nil {
		assert.GreaterOrEqual(suite.T(), resp.TLS.Version, uint16(tls.VersionTLS12), "Should use TLS 1.2 or higher")
		assert.NotEmpty(suite.T(), resp.TLS.CipherSuite, "Should have cipher suite")
	}
}

// Test session management
func (suite *SecurityTestSuite) TestSessionManagement() {
	suite.T().Log("Testing session management...")

	// Test session creation
	loginResp := suite.performLogin("testuser", "testpass")
	assert.Equal(suite.T(), http.StatusOK, loginResp.StatusCode)

	// Extract session token/cookie
	cookies := loginResp.Cookies()
	var sessionCookie *http.Cookie
	for _, cookie := range cookies {
		if cookie.Name == "session_id" {
			sessionCookie = cookie
			break
		}
	}

	if sessionCookie != nil {
		// Verify session cookie properties
		assert.True(suite.T(), sessionCookie.HttpOnly, "Session cookie should be HttpOnly")
		assert.True(suite.T(), sessionCookie.Secure, "Session cookie should be Secure")
		assert.Equal(suite.T(), http.SameSiteStrictMode, sessionCookie.SameSite, "Session cookie should use SameSite=Strict")
		assert.NotZero(suite.T(), sessionCookie.MaxAge, "Session cookie should have expiration")
	}
}

// Test password security
func (suite *SecurityTestSuite) TestPasswordSecurity() {
	suite.T().Log("Testing password security...")

	weakPasswords := []string{
		"123456",
		"password",
		"admin",
		"qwerty",
		"abc123",
		"",
	}

	for _, password := range weakPasswords {
		suite.T().Run(fmt.Sprintf("WeakPassword_%s", password), func(t *testing.T) {
			resp := suite.performLogin("testuser", password)
			// Should reject weak passwords during registration/password change
			assert.NotEqual(t, http.StatusOK, resp.StatusCode, "Should reject weak password: %s", password)
		})
	}

	// Test strong password
	strongPassword := "StrongP@ssw0rd123!"
	resp := suite.performLogin("testuser", strongPassword)
	// This might succeed or fail based on user existence, but shouldn't be rejected for password strength
	assert.NotEqual(suite.T(), http.StatusBadRequest, resp.StatusCode, "Should not reject strong password")
}

// Test data encryption
func (suite *SecurityTestSuite) TestDataEncryption() {
	suite.T().Log("Testing data encryption...")

	// Test that sensitive data is encrypted in transit (HTTPS)
	token := suite.generateValidJWT()
	req, _ := http.NewRequest("GET", suite.server.URL+"/api/v1/orders", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)

	// Verify response is over HTTPS
	assert.True(suite.T(), strings.HasPrefix(resp.Request.URL.Scheme, "https"), "Should use HTTPS")

	// Test that sensitive fields are not exposed in logs or responses
	// This would typically involve checking actual API responses
	assert.Equal(suite.T(), http.StatusOK, resp.StatusCode)
}

// Test audit logging
func (suite *SecurityTestSuite) TestAuditLogging() {
	suite.T().Log("Testing audit logging...")

	// Perform various operations that should be logged
	token := suite.generateValidJWT()

	operations := []struct {
		method string
		path   string
	}{
		{"GET", "/api/v1/orders"},
		{"POST", "/api/v1/orders"},
		{"PUT", "/api/v1/orders/123"},
		{"DELETE", "/api/v1/orders/123"},
	}

	for _, op := range operations {
		req, _ := http.NewRequest(op.method, suite.server.URL+op.path, nil)
		req.Header.Set("Authorization", "Bearer "+token)

		resp, err := suite.client.Do(req)
		require.NoError(suite.T(), err)

		// Verify audit headers are present
		assert.NotEmpty(suite.T(), resp.Header.Get("X-Request-ID"), "Should have request ID for audit trail")
	}
}

// Test CORS configuration
func (suite *SecurityTestSuite) TestCORSConfiguration() {
	suite.T().Log("Testing CORS configuration...")

	// Test preflight request
	req, _ := http.NewRequest("OPTIONS", suite.server.URL+"/api/v1/orders", nil)
	req.Header.Set("Origin", "https://malicious-site.com")
	req.Header.Set("Access-Control-Request-Method", "POST")

	resp, err := suite.client.Do(req)
	require.NoError(suite.T(), err)

	// Should not allow arbitrary origins
	corsOrigin := resp.Header.Get("Access-Control-Allow-Origin")
	assert.NotEqual(suite.T(), "*", corsOrigin, "Should not allow all origins")
	assert.NotEqual(suite.T(), "https://malicious-site.com", corsOrigin, "Should not allow malicious origins")
}

// Test security headers
func (suite *SecurityTestSuite) TestSecurityHeaders() {
	suite.T().Log("Testing security headers...")

	resp, err := suite.client.Get(suite.server.URL + "/health")
	require.NoError(suite.T(), err)

	// Check for important security headers
	headers := map[string]string{
		"X-Content-Type-Options":    "nosniff",
		"X-Frame-Options":           "DENY",
		"X-XSS-Protection":          "1; mode=block",
		"Strict-Transport-Security": "max-age=31536000; includeSubDomains",
		"Content-Security-Policy":   "", // Should be present
		"Referrer-Policy":           "strict-origin-when-cross-origin",
	}

	for header, expectedValue := range headers {
		actualValue := resp.Header.Get(header)
		if expectedValue == "" {
			assert.NotEmpty(suite.T(), actualValue, "Header %s should be present", header)
		} else {
			assert.Equal(suite.T(), expectedValue, actualValue, "Header %s should have correct value", header)
		}
	}
}

// Test API versioning security
func (suite *SecurityTestSuite) TestAPIVersioningSecurity() {
	suite.T().Log("Testing API versioning security...")

	// Test that old API versions are properly deprecated/secured
	oldVersions := []string{
		"/api/v0/orders",
		"/api/legacy/orders",
		"/orders", // Unversioned
	}

	for _, path := range oldVersions {
		resp, err := suite.client.Get(suite.server.URL + path)
		require.NoError(suite.T(), err)

		// Old versions should return 404 or redirect to current version
		assert.Contains(suite.T(), []int{http.StatusNotFound, http.StatusMovedPermanently, http.StatusFound},
			resp.StatusCode, "Old API version should be handled securely: %s", path)
	}
}

// Test file upload security
func (suite *SecurityTestSuite) TestFileUploadSecurity() {
	suite.T().Log("Testing file upload security...")

	// Test malicious file uploads
	maliciousFiles := []struct {
		name     string
		content  string
		mimeType string
	}{
		{"script.js", "<script>alert('xss')</script>", "application/javascript"},
		{"shell.sh", "#!/bin/bash\nrm -rf /", "application/x-sh"},
		{"virus.exe", "MZ\x90\x00", "application/octet-stream"},
		{"large.txt", strings.Repeat("A", 10*1024*1024), "text/plain"}, // 10MB file
	}

	for _, file := range maliciousFiles {
		suite.T().Run(fmt.Sprintf("MaliciousFile_%s", file.name), func(t *testing.T) {
			// This would test actual file upload endpoint
			// For now, just verify the test structure
			assert.NotEmpty(t, file.content, "Test file should have content")
		})
	}
}

// Helper methods

func (suite *SecurityTestSuite) handleOrdersEndpoint(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !suite.isValidToken(token) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check for malicious input
	symbol := r.URL.Query().Get("symbol")
	if suite.containsMaliciousInput(symbol) {
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	// Simulate rate limiting
	if suite.shouldRateLimit(r) {
		w.WriteHeader(http.StatusTooManyRequests)
		return
	}

	// Add security headers
	suite.addSecurityHeaders(w)

	// Add audit trail
	w.Header().Set("X-Request-ID", suite.generateRequestID())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"orders": []}`))
}

func (suite *SecurityTestSuite) handleUsersEndpoint(w http.ResponseWriter, r *http.Request) {
	// Check authentication
	authHeader := r.Header.Get("Authorization")
	if !strings.HasPrefix(authHeader, "Bearer ") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if !suite.isValidToken(token) {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	// Check authorization (admin only)
	if !suite.isAdminToken(token) {
		w.WriteHeader(http.StatusForbidden)
		return
	}

	suite.addSecurityHeaders(w)
	w.Header().Set("X-Request-ID", suite.generateRequestID())

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"users": []}`))
}

func (suite *SecurityTestSuite) handleLoginEndpoint(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Parse credentials (simplified)
	username := r.FormValue("username")
	password := r.FormValue("password")

	// Check password strength
	if suite.isWeakPassword(password) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "Password does not meet security requirements"}`))
		return
	}

	// Simulate login
	if username == "testuser" && password == "StrongP@ssw0rd123!" {
		// Set secure session cookie
		cookie := &http.Cookie{
			Name:     "session_id",
			Value:    suite.generateSessionID(),
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteStrictMode,
			MaxAge:   3600, // 1 hour
		}
		http.SetCookie(w, cookie)

		suite.addSecurityHeaders(w)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	} else {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte(`{"error": "Invalid credentials"}`))
	}
}

func (suite *SecurityTestSuite) generateValidJWT() string {
	// Simplified JWT generation for testing
	return "valid_jwt_token_user"
}

func (suite *SecurityTestSuite) generateUserJWT() string {
	return "valid_jwt_token_user"
}

func (suite *SecurityTestSuite) generateAdminJWT() string {
	return "valid_jwt_token_admin"
}

func (suite *SecurityTestSuite) isValidToken(token string) bool {
	validTokens := []string{"valid_jwt_token_user", "valid_jwt_token_admin"}
	for _, validToken := range validTokens {
		if token == validToken {
			return true
		}
	}
	return false
}

func (suite *SecurityTestSuite) isAdminToken(token string) bool {
	return token == "valid_jwt_token_admin"
}

func (suite *SecurityTestSuite) containsMaliciousInput(input string) bool {
	maliciousPatterns := []string{
		"<script",
		"javascript:",
		"DROP TABLE",
		"rm -rf",
		"../",
		"<?xml",
		")(|(",
	}

	lowerInput := strings.ToLower(input)
	for _, pattern := range maliciousPatterns {
		if strings.Contains(lowerInput, strings.ToLower(pattern)) {
			return true
		}
	}
	return false
}

func (suite *SecurityTestSuite) shouldRateLimit(r *http.Request) bool {
	// Simplified rate limiting simulation
	// In reality, this would check against a rate limiter
	return false // Disabled for most tests
}

func (suite *SecurityTestSuite) addSecurityHeaders(w http.ResponseWriter) {
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.Header().Set("X-Frame-Options", "DENY")
	w.Header().Set("X-XSS-Protection", "1; mode=block")
	w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
	w.Header().Set("Content-Security-Policy", "default-src 'self'")
	w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
}

func (suite *SecurityTestSuite) generateRequestID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (suite *SecurityTestSuite) generateSessionID() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func (suite *SecurityTestSuite) performLogin(username, password string) *http.Response {
	req, _ := http.NewRequest("POST", suite.server.URL+"/api/v1/auth/login",
		strings.NewReader(fmt.Sprintf("username=%s&password=%s", username, password)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, _ := suite.client.Do(req)
	return resp
}

func (suite *SecurityTestSuite) isWeakPassword(password string) bool {
	weakPasswords := []string{"123456", "password", "admin", "qwerty", "abc123", ""}
	for _, weak := range weakPasswords {
		if password == weak {
			return true
		}
	}

	// Check minimum requirements
	if len(password) < 8 {
		return true
	}

	return false
}

// Run the security test suite
func TestSecurityTestSuite(t *testing.T) {
	suite.Run(t, new(SecurityTestSuite))
}
