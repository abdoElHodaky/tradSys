package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// App represents the HFT application
type App struct {
	router *gin.Engine
	server *http.Server
	metrics *Metrics
}

// Metrics holds Prometheus metrics
type Metrics struct {
	OrdersTotal     prometheus.CounterVec
	OrdersProcessed prometheus.CounterVec
	ResponseTime    prometheus.HistogramVec
	ActiveOrders    prometheus.GaugeVec
}

// New creates a new HFT application
func New() *App {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	metrics := initMetrics()
	
	return &App{
		router:  router,
		metrics: metrics,
	}
}

// initMetrics initializes Prometheus metrics
func initMetrics() *Metrics {
	ordersTotal := *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trading_orders_total",
			Help: "Total number of orders received",
		},
		[]string{"symbol", "side", "type"},
	)
	
	ordersProcessed := *prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "trading_orders_processed_total",
			Help: "Total number of orders processed",
		},
		[]string{"symbol", "side", "status"},
	)
	
	responseTime := *prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name: "trading_response_time_seconds",
			Help: "Response time for trading operations",
		},
		[]string{"operation"},
	)
	
	activeOrders := *prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "trading_active_orders",
			Help: "Number of active orders",
		},
		[]string{"symbol"},
	)
	
	// Register metrics
	prometheus.MustRegister(&ordersTotal)
	prometheus.MustRegister(&ordersProcessed)
	prometheus.MustRegister(&responseTime)
	prometheus.MustRegister(&activeOrders)
	
	return &Metrics{
		OrdersTotal:     ordersTotal,
		OrdersProcessed: ordersProcessed,
		ResponseTime:    responseTime,
		ActiveOrders:    activeOrders,
	}
}

// SetupRoutes sets up the application routes
func (a *App) SetupRoutes() {
	a.router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "healthy",
			"timestamp": time.Now(),
		})
	})
	
	// Prometheus metrics endpoint
	a.router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}

// Start starts the application server
func (a *App) Start(port string) error {
	a.server = &http.Server{
		Addr:    ":" + port,
		Handler: a.router,
	}
	
	log.Printf("Starting HFT server on port %s", port)
	return a.server.ListenAndServe()
}

// Stop stops the application server
func (a *App) Stop(ctx context.Context) error {
	if a.server != nil {
		return a.server.Shutdown(ctx)
	}
	return nil
}

// GetRouter returns the gin router
func (a *App) GetRouter() *gin.Engine {
	return a.router
}
