package gateway

import (
	"context"
	"io"
	"net/http"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/architecture/discovery"
	"github.com/asim/go-micro/v3/registry"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// APIGateway provides API gateway functionality
type APIGateway struct {
	router      *gin.Engine
	discovery   *discovery.ServiceDiscovery
	selector    *discovery.ServiceSelector
	logger      *zap.Logger
	httpClient  *http.Client
	routes      map[string]Route
	middlewares []gin.HandlerFunc
}

// Route represents a route in the API gateway
type Route struct {
	Method      string
	Path        string
	ServiceName string
	ServicePath string
	Middlewares []gin.HandlerFunc
}

// NewAPIGateway creates a new API gateway
func NewAPIGateway(discoveryService *discovery.ServiceDiscovery, logger *zap.Logger) *APIGateway {
	// Create a new gin router
	router := gin.New()
	router.Use(gin.Recovery())

	// Create a service selector with round-robin strategy
	selector := discovery.NewServiceSelector(
		discoveryService,
		logger,
		discovery.NewRoundRobinStrategy(),
	)

	// Create an HTTP client for proxying requests
	httpClient := &http.Client{
		Timeout: 10 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 100,
			IdleConnTimeout:     90 * time.Second,
		},
	}

	return &APIGateway{
		router:      router,
		discovery:   discoveryService,
		selector:    selector,
		logger:      logger,
		httpClient:  httpClient,
		routes:      make(map[string]Route),
		middlewares: []gin.HandlerFunc{},
	}
}

// Use adds middleware to the API gateway
func (g *APIGateway) Use(middleware gin.HandlerFunc) {
	g.middlewares = append(g.middlewares, middleware)
	g.router.Use(middleware)
}

// AddRoute adds a route to the API gateway
func (g *APIGateway) AddRoute(route Route) {
	// Add the route to the map
	routeKey := route.Method + ":" + route.Path
	g.routes[routeKey] = route

	// Add the route to the router
	g.router.Handle(route.Method, route.Path, append(g.middlewares, append(route.Middlewares, g.handleRequest(route))...)...)

	g.logger.Info("Added route",
		zap.String("method", route.Method),
		zap.String("path", route.Path),
		zap.String("service", route.ServiceName),
		zap.String("servicePath", route.ServicePath))
}

// handleRequest handles an API request
func (g *APIGateway) handleRequest(route Route) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get a node for the service
		node, err := g.selector.Select(c.Request.Context(), route.ServiceName)
		if err != nil {
			g.logger.Error("Failed to select node",
				zap.String("service", route.ServiceName),
				zap.Error(err))
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"error": "Service unavailable",
			})
			return
		}

		// Create a new request to the service
		req, err := http.NewRequestWithContext(
			c.Request.Context(),
			c.Request.Method,
			"http://"+node.Address+route.ServicePath,
			c.Request.Body,
		)
		if err != nil {
			g.logger.Error("Failed to create request",
				zap.String("service", route.ServiceName),
				zap.String("node", node.Address),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}

		// Copy headers from the original request
		for key, values := range c.Request.Header {
			for _, value := range values {
				req.Header.Add(key, value)
			}
		}

		// Send the request to the service
		resp, err := g.httpClient.Do(req)
		if err != nil {
			g.logger.Error("Failed to send request",
				zap.String("service", route.ServiceName),
				zap.String("node", node.Address),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}
		defer resp.Body.Close()

		// Copy headers from the service response
		for key, values := range resp.Header {
			for _, value := range values {
				c.Header(key, value)
			}
		}

		// Copy the status code
		c.Status(resp.StatusCode)

		// Copy the response body
		_, err = io.Copy(c.Writer, resp.Body)
		if err != nil {
			g.logger.Error("Failed to copy response body",
				zap.String("service", route.ServiceName),
				zap.String("node", node.Address),
				zap.Error(err))
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": "Internal server error",
			})
			return
		}
	}
}

// Run starts the API gateway
func (g *APIGateway) Run(addr string) error {
	return g.router.Run(addr)
}

// RegisterService registers a service with the API gateway
func (g *APIGateway) RegisterService(ctx context.Context, service *registry.Service, routes []Route) error {
	// Register the service with the discovery service
	err := g.discovery.RegisterService(ctx, service)
	if err != nil {
		return err
	}

	// Add routes for the service
	for _, route := range routes {
		g.AddRoute(route)
	}

	return nil
}

// DeregisterService deregisters a service from the API gateway
func (g *APIGateway) DeregisterService(ctx context.Context, service *registry.Service) error {
	// Deregister the service from the discovery service
	return g.discovery.DeregisterService(ctx, service)
}

// LoadBalancer provides load balancing functionality
type LoadBalancer struct {
	selector *discovery.ServiceSelector
	logger   *zap.Logger
}

// NewLoadBalancer creates a new load balancer
func NewLoadBalancer(selector *discovery.ServiceSelector, logger *zap.Logger) *LoadBalancer {
	return &LoadBalancer{
		selector: selector,
		logger:   logger,
	}
}

// Forward forwards a request to a service
func (lb *LoadBalancer) Forward(ctx context.Context, serviceName string, req *http.Request) (*http.Response, error) {
	// Get a node for the service
	node, err := lb.selector.Select(ctx, serviceName)
	if err != nil {
		lb.logger.Error("Failed to select node",
			zap.String("service", serviceName),
			zap.Error(err))
		return nil, err
	}

	// Create a new request to the service
	serviceReq, err := http.NewRequestWithContext(
		ctx,
		req.Method,
		"http://"+node.Address+req.URL.Path,
		req.Body,
	)
	if err != nil {
		lb.logger.Error("Failed to create request",
			zap.String("service", serviceName),
			zap.String("node", node.Address),
			zap.Error(err))
		return nil, err
	}

	// Copy headers from the original request
	for key, values := range req.Header {
		for _, value := range values {
			serviceReq.Header.Add(key, value)
		}
	}

	// Send the request to the service
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	return client.Do(serviceReq)
}
