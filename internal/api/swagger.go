package api

import (
	"net/http"
	"path/filepath"

	"github.com/gin-gonic/gin"
)

// RegisterSwaggerRoutes registers routes for Swagger documentation
func RegisterSwaggerRoutes(router *gin.Engine) {
	// Serve Swagger UI
	router.GET("/swagger", func(c *gin.Context) {
		c.Redirect(http.StatusMovedPermanently, "/swagger/")
	})
	
	router.GET("/swagger/*any", func(c *gin.Context) {
		path := c.Param("any")
		if path == "/" {
			path = "/index.html"
		}
		
		// Serve from the docs/swagger-ui directory
		filePath := filepath.Join("docs/swagger-ui", path)
		c.File(filePath)
	})
	
	// Serve Swagger YAML
	router.GET("/swagger.yaml", func(c *gin.Context) {
		c.File("docs/swagger.yaml")
	})
}

