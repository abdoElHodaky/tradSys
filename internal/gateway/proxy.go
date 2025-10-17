package gateway

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
	"strings"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/config"
	"github.com/gin-gonic/gin"
	"go-micro.dev/v4/registry"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// ProxyParams contains the parameters for creating a new service proxy
type ProxyParams struct {
	fx.In

	Logger   *zap.Logger
	Config   *config.Config
	Registry registry.Registry
}

// ServiceProxy handles forwarding requests to microservices
type ServiceProxy struct {
	logger   *zap.Logger
	config   *config.Config
	registry registry.Registry
	client   *http.Client
}

// NewServiceProxy creates a new service proxy with fx dependency injection
func NewServiceProxy(p ProxyParams) *ServiceProxy {
	return &ServiceProxy{
		logger:   p.Logger,
		config:   p.Config,
		registry: p.Registry,
		client: &http.Client{
			Timeout: 10 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 20,
				IdleConnTimeout:     60 * time.Second,
			},
		},
	}
}

// ForwardToService creates a handler that forwards requests to the appropriate microservice
func (p *ServiceProxy) ForwardToService(serviceName, path string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get service from registry
		services, err := p.registry.GetService(serviceName)
		if err != nil {
			p.logger.Error("Failed to get service from registry",
				zap.String("service", serviceName),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		if len(services) == 0 {
			p.logger.Error("No instances found for service",
				zap.String("service", serviceName))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		// Get the first available node from the first service
		service := services[0]
		if len(service.Nodes) == 0 {
			p.logger.Error("No nodes found for service",
				zap.String("service", serviceName))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		// Use the first node's address
		node := service.Nodes[0]
		endpoint := fmt.Sprintf("http://%s", node.Address)

		// Create target URL
		targetPath := path
		// Replace path parameters
		for _, param := range c.Params {
			targetPath = strings.Replace(targetPath, ":"+param.Key, param.Value, -1)
		}

		// Add query parameters
		if len(c.Request.URL.RawQuery) > 0 {
			targetPath += "?" + c.Request.URL.RawQuery
		}

		targetURL, err := url.Parse(endpoint + targetPath)
		if err != nil {
			p.logger.Error("Failed to parse target URL",
				zap.String("endpoint", endpoint),
				zap.String("path", targetPath),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Create proxy request
		proxyReq, err := http.NewRequest(c.Request.Method, targetURL.String(), nil)
		if err != nil {
			p.logger.Error("Failed to create proxy request", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Copy headers
		for key, values := range c.Request.Header {
			for _, value := range values {
				proxyReq.Header.Add(key, value)
			}
		}

		// Add X-Forwarded headers
		proxyReq.Header.Set("X-Forwarded-For", c.ClientIP())
		proxyReq.Header.Set("X-Forwarded-Proto", c.Request.Proto)
		proxyReq.Header.Set("X-Forwarded-Host", c.Request.Host)
		proxyReq.Header.Set("X-Forwarded-Path", c.Request.URL.Path)

		// Copy request body if present
		if c.Request.Body != nil {
			bodyBytes, err := io.ReadAll(c.Request.Body)
			if err != nil {
				p.logger.Error("Failed to read request body", zap.Error(err))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal server error",
				})
				return
			}
			proxyReq.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
			proxyReq.ContentLength = int64(len(bodyBytes))
		}

		// Execute request with circuit breaker
		start := time.Now()
		resp, err := p.client.Do(proxyReq)
		latency := time.Since(start)

		// Log request
		p.logger.Info("Proxy request",
			zap.String("service", serviceName),
			zap.String("method", c.Request.Method),
			zap.String("path", targetPath),
			zap.Duration("latency", latency))

		if err != nil {
			p.logger.Error("Failed to execute proxy request",
				zap.String("service", serviceName),
				zap.String("url", targetURL.String()),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		defer resp.Body.Close()

		// Copy response headers
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Copy response body
		bodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			p.logger.Error("Failed to read response body", zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Set response status and write body
		c.Status(resp.StatusCode)
		c.Writer.Write(bodyBytes)
	}
}

// ReverseProxy creates a reverse proxy handler for a service
func (p *ServiceProxy) ReverseProxy(serviceName string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get service from registry
		services, err := p.registry.GetService(serviceName)
		if err != nil {
			p.logger.Error("Failed to get service from registry",
				zap.String("service", serviceName),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		if len(services) == 0 {
			p.logger.Error("No instances found for service",
				zap.String("service", serviceName))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		// Get the first available node from the first service
		service := services[0]
		if len(service.Nodes) == 0 {
			p.logger.Error("No nodes found for service",
				zap.String("service", serviceName))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}
		
		// Use the first node's address
		node := service.Nodes[0]
		endpoint := fmt.Sprintf("http://%s", node.Address)

		// Parse target URL
		target, err := url.Parse(endpoint)
		if err != nil {
			p.logger.Error("Failed to parse target URL",
				zap.String("endpoint", endpoint),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Create reverse proxy
		proxy := httputil.NewSingleHostReverseProxy(target)
		
		// Set custom director to modify the request
		originalDirector := proxy.Director
		proxy.Director = func(req *http.Request) {
			originalDirector(req)
			
			// Add X-Forwarded headers
			req.Header.Set("X-Forwarded-For", c.ClientIP())
			req.Header.Set("X-Forwarded-Proto", c.Request.Proto)
			req.Header.Set("X-Forwarded-Host", c.Request.Host)
			req.Header.Set("X-Forwarded-Path", c.Request.URL.Path)
		}

		// Set error handler
		proxy.ErrorHandler = func(rw http.ResponseWriter, req *http.Request, err error) {
			p.logger.Error("Reverse proxy error",
				zap.String("service", serviceName),
				zap.String("url", req.URL.String()),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
		}

		// Log request
		p.logger.Info("Reverse proxy request",
			zap.String("service", serviceName),
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path))

		// Serve the request
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
