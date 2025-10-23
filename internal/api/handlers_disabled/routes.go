package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// SetupRoutes sets up the API routes
func SetupRoutes(router *gin.RouterGroup, tradingSystem interface{}) {
	// Health check
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"message": "Trading system API is running",
		})
	})

	// Placeholder routes
	router.GET("/orders", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Orders endpoint - implementation pending",
		})
	})

	router.GET("/trades", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Trades endpoint - implementation pending",
		})
	})

	router.GET("/positions", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "Positions endpoint - implementation pending",
		})
	})
}
