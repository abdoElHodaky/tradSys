package api

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/swaggo/files"
	"github.com/swaggo/gin-swagger"
	"go.uber.org/zap"
)

// DocsConfig represents the configuration for API documentation
type DocsConfig struct {
	// BasePath is the base path for the API
	BasePath string
	// Title is the title of the API documentation
	Title string
	// Description is the description of the API documentation
	Description string
	// Version is the version of the API documentation
	Version string
	// Host is the host of the API
	Host string
	// Schemes is the list of schemes supported by the API
	Schemes []string
	// EnableSwagger enables Swagger documentation
	EnableSwagger bool
	// SwaggerPath is the path to the Swagger documentation
	SwaggerPath string
	// CustomDocsPath is the path to custom documentation
	CustomDocsPath string
}

// DefaultDocsConfig returns the default configuration for API documentation
func DefaultDocsConfig() DocsConfig {
	return DocsConfig{
		BasePath:      "/api/v1",
		Title:         "TradSys API",
		Description:   "Trading System API",
		Version:       "1.0.0",
		Host:          "localhost:8080",
		Schemes:       []string{"http", "https"},
		EnableSwagger: true,
		SwaggerPath:   "/swagger/*any",
		CustomDocsPath: "",
	}
}

// SetupDocs sets up API documentation
func SetupDocs(router *gin.Engine, config DocsConfig, logger *zap.Logger) error {
	// Initialize Swagger documentation
	if config.EnableSwagger {
		// Initialize Swagger info
		docs := swaggerFiles.NewHandler()
		
		// Set up Swagger endpoint
		router.GET(config.SwaggerPath, ginSwagger.WrapHandler(docs))
		
		logger.Info("Swagger documentation enabled",
			zap.String("path", config.SwaggerPath))
	}
	
	// Set up custom documentation if provided
	if config.CustomDocsPath != "" {
		if _, err := os.Stat(config.CustomDocsPath); err != nil {
			logger.Error("Custom documentation path does not exist",
				zap.String("path", config.CustomDocsPath),
				zap.Error(err))
			return fmt.Errorf("custom documentation path does not exist: %w", err)
		}
		
		// Serve custom documentation
		docsPath := filepath.Clean(config.CustomDocsPath)
		router.StaticFS("/docs", http.Dir(docsPath))
		
		logger.Info("Custom documentation enabled",
			zap.String("path", "/docs"),
			zap.String("source", docsPath))
	}
	
	return nil
}

