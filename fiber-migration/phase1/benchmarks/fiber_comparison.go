package benchmarks

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/abdoElHodaky/tradSys/internal/db/models"
)

// FiberServer represents the Fiber implementation for benchmarking
type FiberServer struct {
	app *fiber.App
}

// NewFiberServer creates a new Fiber server for benchmarking
func NewFiberServer() *FiberServer {
	app := fiber.New(fiber.Config{
		DisableStartupMessage: true,
		JSONEncoder:           json.Marshal,
		JSONDecoder:           json.Unmarshal,
	})
	
	server := &FiberServer{
		app: app,
	}
	
	server.setupRoutes()
	return server
}

func (s *FiberServer) setupRoutes() {
	// Health check endpoint
	s.app.Get("/health", s.healthCheck)
	
	// Trading pairs endpoints
	pairs := s.app.Group("/api/pairs")
	pairs.Get("/", s.getAllPairs)
	pairs.Get("/:id", s.getPair)
	pairs.Post("/", s.createPair)
	pairs.Put("/:id", s.updatePair)
	pairs.Delete("/:id", s.deletePair)
	
	// Order endpoints
	orders := s.app.Group("/api/orders")
	orders.Get("/", s.getAllOrders)
	orders.Post("/", s.createOrder)
	orders.Get("/:id", s.getOrder)
}

func (s *FiberServer) healthCheck(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "ok",
		"timestamp": time.Now().Unix(),
	})
}

func (s *FiberServer) getAllPairs(c *fiber.Ctx) error {
	// Simulate database query
	pairs := []models.Pair{
		{ID: "1", Symbol: "BTCUSD", BaseAsset: "BTC", QuoteAsset: "USD"},
		{ID: "2", Symbol: "ETHUSD", BaseAsset: "ETH", QuoteAsset: "USD"},
		{ID: "3", Symbol: "ADAUSD", BaseAsset: "ADA", QuoteAsset: "USD"},
	}
	
	return c.JSON(fiber.Map{
		"pairs": pairs,
		"count": len(pairs),
	})
}

func (s *FiberServer) getPair(c *fiber.Ctx) error {
	id := c.Params("id")
	
	// Simulate database lookup
	pair := models.Pair{
		ID:         id,
		Symbol:     "BTCUSD",
		BaseAsset:  "BTC",
		QuoteAsset: "USD",
	}
	
	return c.JSON(pair)
}

func (s *FiberServer) createPair(c *fiber.Ctx) error {
	var pair models.Pair
	if err := c.BodyParser(&pair); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	// Simulate database insert
	pair.ID = "new-id"
	
	return c.Status(fiber.StatusCreated).JSON(pair)
}

func (s *FiberServer) updatePair(c *fiber.Ctx) error {
	id := c.Params("id")
	var pair models.Pair
	
	if err := c.BodyParser(&pair); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	pair.ID = id
	return c.JSON(pair)
}

func (s *FiberServer) deletePair(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"message": "Pair deleted successfully",
		"id":      id,
	})
}

func (s *FiberServer) getAllOrders(c *fiber.Ctx) error {
	// Simulate database query with pagination
	orders := []fiber.Map{
		{"id": "1", "symbol": "BTCUSD", "side": "buy", "quantity": 1.0, "price": 50000.0},
		{"id": "2", "symbol": "ETHUSD", "side": "sell", "quantity": 10.0, "price": 3000.0},
	}
	
	return c.JSON(fiber.Map{
		"orders": orders,
		"count":  len(orders),
	})
}

func (s *FiberServer) createOrder(c *fiber.Ctx) error {
	var order map[string]interface{}
	if err := c.BodyParser(&order); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": err.Error(),
		})
	}
	
	// Simulate order processing
	order["id"] = "new-order-id"
	order["status"] = "pending"
	order["timestamp"] = time.Now().Unix()
	
	return c.Status(fiber.StatusCreated).JSON(order)
}

func (s *FiberServer) getOrder(c *fiber.Ctx) error {
	id := c.Params("id")
	
	order := fiber.Map{
		"id":       id,
		"symbol":   "BTCUSD",
		"side":     "buy",
		"quantity": 1.0,
		"price":    50000.0,
		"status":   "filled",
	}
	
	return c.JSON(order)
}

// Benchmark functions
func BenchmarkFiberHealthCheck(b *testing.B) {
	server := NewFiberServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/health", nil)
			resp, _ := server.app.Test(req, -1)
			resp.Body.Close()
		}
	})
}

func BenchmarkFiberGetAllPairs(b *testing.B) {
	server := NewFiberServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/pairs/", nil)
			resp, _ := server.app.Test(req, -1)
			resp.Body.Close()
		}
	})
}

func BenchmarkFiberGetPair(b *testing.B) {
	server := NewFiberServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/pairs/1", nil)
			resp, _ := server.app.Test(req, -1)
			resp.Body.Close()
		}
	})
}

func BenchmarkFiberCreateOrder(b *testing.B) {
	server := NewFiberServer()
	
	orderData := map[string]interface{}{
		"symbol":   "BTCUSD",
		"side":     "buy",
		"quantity": 1.0,
		"price":    50000.0,
	}
	
	jsonData, _ := json.Marshal(orderData)
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("POST", "/api/orders/", bytes.NewBuffer(jsonData))
			req.Header.Set("Content-Type", "application/json")
			resp, _ := server.app.Test(req, -1)
			resp.Body.Close()
		}
	})
}

func BenchmarkFiberJSONSerialization(b *testing.B) {
	server := NewFiberServer()
	
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/orders/", nil)
			resp, _ := server.app.Test(req, -1)
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// Memory allocation benchmarks
func BenchmarkFiberMemoryAllocation(b *testing.B) {
	server := NewFiberServer()
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/pairs/", nil)
			resp, _ := server.app.Test(req, -1)
			resp.Body.Close()
		}
	})
}

func BenchmarkGinMemoryAllocation(b *testing.B) {
	server := NewGinServer()
	
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req, _ := http.NewRequest("GET", "/api/pairs/", nil)
			resp, _ := server.router.Test(req, -1)
			resp.Body.Close()
		}
	})
}

// Comparative benchmark functions
func BenchmarkComparison(b *testing.B) {
	ginServer := NewGinServer()
	fiberServer := NewFiberServer()
	
	b.Run("Gin-HealthCheck", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("GET", "/health", nil)
				resp, _ := ginServer.router.Test(req, -1)
				resp.Body.Close()
			}
		})
	})
	
	b.Run("Fiber-HealthCheck", func(b *testing.B) {
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("GET", "/health", nil)
				resp, _ := fiberServer.app.Test(req, -1)
				resp.Body.Close()
			}
		})
	})
	
	b.Run("Gin-CreateOrder", func(b *testing.B) {
		orderData := map[string]interface{}{
			"symbol": "BTCUSD", "side": "buy", "quantity": 1.0, "price": 50000.0,
		}
		jsonData, _ := json.Marshal(orderData)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("POST", "/api/orders/", bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
				resp, _ := ginServer.router.Test(req, -1)
				resp.Body.Close()
			}
		})
	})
	
	b.Run("Fiber-CreateOrder", func(b *testing.B) {
		orderData := map[string]interface{}{
			"symbol": "BTCUSD", "side": "buy", "quantity": 1.0, "price": 50000.0,
		}
		jsonData, _ := json.Marshal(orderData)
		
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req, _ := http.NewRequest("POST", "/api/orders/", bytes.NewBuffer(jsonData))
				req.Header.Set("Content-Type", "application/json")
				resp, _ := fiberServer.app.Test(req, -1)
				resp.Body.Close()
			}
		})
	})
}
