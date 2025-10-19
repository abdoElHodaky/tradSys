package app

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// App represents the HFT application
type App struct {
	router *gin.Engine
	server *http.Server
}

// New creates a new HFT application
func New() *App {
	router := gin.New()
	router.Use(gin.Logger())
	router.Use(gin.Recovery())
	
	return &App{
		router: router,
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
	
	a.router.GET("/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"metrics": "placeholder",
		})
	})
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
