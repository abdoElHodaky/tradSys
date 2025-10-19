package gateway_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/auth"
	"github.com/abdoElHodaky/tradSys/internal/unified-config"
	"github.com/abdoElHodaky/tradSys/internal/gateway"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
	"go.uber.org/fx/fxtest"
	"go.uber.org/zap"
)

// ErrTestError is a test error
var ErrTestError = errors.New("test error")

func TestHealthEndpoint(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a server variable to be populated
	var server *gateway.Server

	// Create a test app with fx
	app := fxtest.New(t,
		fx.Supply(logger),
		fx.Provide(func() (*config.Config, error) {
			cfg := &config.Config{}
			cfg.Service.Name = "api-gateway-test"
			cfg.Service.Version = "1.0.0"
			cfg.Service.Address = ":8000"
			cfg.Service.Environment = "test"
			cfg.Gateway.Address = ":8000"
			cfg.Auth.JWTSecret = "test-secret"
			return cfg, nil
		}),
		fx.Provide(func() *auth.Middleware {
			return &auth.Middleware{}
		}),
		gateway.Module,
		fx.Populate(&server),
	)
	defer app.RequireStart().RequireStop()

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	// Serve the request
	server.Router().ServeHTTP(w, req)

	// Check the response
	assert.Equal(t, http.StatusOK, w.Code)

	// Parse the response body
	var response map[string]string
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)

	// Check the response content
	assert.Equal(t, "ok", response["status"])
}

func TestRateLimiting(t *testing.T) {
	// Set Gin to test mode
	gin.SetMode(gin.TestMode)

	// Create a test logger
	logger, _ := zap.NewDevelopment()

	// Create a test middleware
	middleware := gateway.NewMiddleware(gateway.MiddlewareParams{
		Logger: logger,
		Config: &config.Config{},
	})

	// Create a test router
	router := gin.New()
	router.Use(middleware.RateLimitByIP(1, 1)) // 1 request per second, burst of 1
	router.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Create a test HTTP request
	req := httptest.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:1234" // Set a remote address for IP-based rate limiting

	// First request should succeed
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req)
	assert.Equal(t, http.StatusOK, w1.Code)

	// Second request should be rate limited
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req)
	assert.Equal(t, http.StatusTooManyRequests, w2.Code)
}

func TestCircuitBreaker(t *testing.T) {
	// Create a new circuit breaker
	cb := gateway.NewCircuitBreaker("test", 2, time.Second) // 2 failures, 1s timeout

	// First execution should succeed
	err := cb.Execute(func() error {
		return nil
	})
	assert.NoError(t, err)

	// Second execution with error
	err = cb.Execute(func() error {
		return ErrTestError
	})
	assert.Error(t, err)
	assert.Equal(t, ErrTestError, err)

	// Third execution with error should trip the circuit
	err = cb.Execute(func() error {
		return ErrTestError
	})
	assert.Error(t, err)
	assert.Equal(t, ErrTestError, err)

	// Fourth execution should return circuit open error
	err = cb.Execute(func() error {
		return nil
	})
	assert.Error(t, err)
	assert.Equal(t, gateway.ErrCircuitOpen, err)
}

