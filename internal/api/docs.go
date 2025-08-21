package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
)

//go:embed docs/swagger.yaml
var swaggerFS embed.FS

// RegisterSwaggerRoutes registers the Swagger documentation routes
func RegisterSwaggerRoutes(router *gin.Engine) {
	// Serve Swagger UI
	router.GET("/swagger/*any", func(c *gin.Context) {
		// Redirect to Swagger UI
		c.Redirect(http.StatusMovedPermanently, "/swagger/index.html")
	})

	// Serve Swagger YAML file
	router.GET("/swagger.yaml", func(c *gin.Context) {
		yamlFile, err := swaggerFS.ReadFile("docs/swagger.yaml")
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read Swagger file"})
			return
		}
		c.Data(http.StatusOK, "application/yaml", yamlFile)
	})

	// Serve Swagger UI static files
	swaggerFiles, err := fs.Sub(swaggerFS, "docs")
	if err != nil {
		panic(err)
	}
	router.StaticFS("/swagger", http.FS(swaggerFiles))
}

