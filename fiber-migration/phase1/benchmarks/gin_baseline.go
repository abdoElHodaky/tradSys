package benchmarks

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/abdoElHodaky/tradSys/internal/db/models"
)

// GinServer represents the baseline Gin implementation for benchmarking
type GinServer struct {
	router *gin.Engine
}

// NewGinServer creates a new Gin server for benchmarking
func NewGinServer() *GinServer {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	
	server := &GinServer{
		router: router,
	}
	
	server.setupRoutes()
	return server
}

func (s *GinServer) setupRoutes() {
	// Health check endpoint
	s.router.GET("/health", s.healthCheck)
	
	// Trading pairs endpoints
	pairs := s.router.Group("/api/pairs")
	{
		pairs.GET("/", s.getAllPairs)
		pairs.GET("/:id", s.getPair)
		pairs.POST("/", s.createPair)
		pairs.PUT("/:id", s.updatePair)
		pairs.DELETE("/:id", s.deletePair)
	}
	
	// Order endpoints
	orders := s.router.Group("/api/orders")
	{
		orders.GET("/", s.getAllOrders)
		orders.POST("/", s.createOrder)
		orders.GET("/:id", s.getOrder)
	}
}

func (s *GinServer) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
		"timestamp": time.Now().Unix(),
	})
}

func (s *GinServer) getAllPairs(c *gin.Context) {
	// Simulate database query
	pairs := []models.Pair{
		{ID: "1", Symbol: "BTCUSD", BaseAsset: "BTC", QuoteAsset: "USD"},
		{ID: "2", Symbol: "ETHUSD", BaseAsset: "ETH", QuoteAsset: "USD"},
		{ID: "3", Symbol: "ADAUSD", BaseAsset: "ADA", QuoteAsset: "USD"},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"pairs": pairs,
		"count": len(pairs),
	})
}

func (s *GinServer) getPair(c *gin.Context) {
	id := c.Param("id")
	
	// Simulate database lookup
	pair := models.Pair{
		ID: id,
		Symbol: "BTCUSD",
		BaseAsset: "BTC",
		QuoteAsset: "USD",
	}
	
	c.JSON(http.StatusOK, pair)
}

func (s *GinServer) createPair(c *gin.Context) {
	var pair models.Pair
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Simulate database insert
	pair.ID = "new-id"
	
	c.JSON(http.StatusCreated, pair)
}

func (s *GinServer) updatePair(c *gin.Context) {
	id := c.Param("id")
	var pair models.Pair
	
	if err := c.ShouldBindJSON(&pair); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	pair.ID = id
	c.JSON(http.StatusOK, pair)
}

func (s *GinServer) deletePair(c *gin.Context) {
	id := c.Param("id")
	c.JSON(http.StatusOK, gin.H{
		"message": "Pair deleted successfully",
		"id": id,
	})
}

func (s *GinServer) getAllOrders(c *gin.Context) {
	// Simulate database query with pagination
	orders := []map[string]interface{}{
		{"id": "1", "symbol": "BTCUSD", "side": "buy", "quantity": 1.0, "price": 50000.0},
		{"id": "2", "symbol": "ETHUSD", "side": "sell", "quantity": 10.0, "price": 3000.0},
	}
	
	c.JSON(http.StatusOK, gin.H{
		"orders": orders,
		"count": len(orders),
	})
}

func (s *GinServer) createOrder(c *gin.Context) {
	var order map[string]interface{}
	if err := c.ShouldBindJSON(&order); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	
	// Simulate order processing
	order["id"] = "new-order-id"
	order["status"] = "pending"
	order["timestamp"] = time.Now().Unix()
	
	c.JSON(http.StatusCreated, order)
}

func (s *GinServer) getOrder(c *gin.Context) {
	id := c.Param("id")
	
	order := map[string]interface{}{
		"id": id,
		"symbol": "BTCUSD",
		"side": "buy",
		"quantity": 1.0,
		"price": 50000.0,
		"status": "filled",
	}
	
	c.JSON(http.StatusOK, order)
}

// Benchmark functions
func BenchmarkGinHealthCheck(b *testing.B) {
	server := NewGinServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkGinGetAllPairs(b *testing.B) {
	server := NewGinServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/pairs/", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkGinGetPair(b *testing.B) {
	server := NewGinServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/pairs/1", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkGinCreateOrder(b *testing.B) {
	server := NewGinServer()
	
	orderData := map[string]interface{}{
		"symbol": "BTCUSD",
		"side": "buy",
		"quantity": 1.0,
		"price": 50000.0,
	}
	
	jsonData, _ := json.Marshal(orderData)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/api/orders/", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}

func BenchmarkGinJSONSerialization(b *testing.B) {
	server := NewGinServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("GET", "/api/orders/", nil)
			w := httptest.NewRecorder()
			server.router.ServeHTTP(w, req)
		}
	})
}
