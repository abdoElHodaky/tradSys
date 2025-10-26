package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/abdoElHodaky/tradSys/internal/handlers"
	"github.com/abdoElHodaky/tradSys/internal/services"
	"github.com/abdoElHodaky/tradSys/pkg/config"
	"github.com/abdoElHodaky/tradSys/pkg/testing"
)

const (
	appName    = "TradSys"
	appVersion = "v3.0.0"
)

func main() {
	// Parse command line flags
	var (
		configPath = flag.String("config", "config.yaml", "Path to configuration file")
		version    = flag.Bool("version", false, "Show version information")
		health     = flag.Bool("health", false, "Perform health check")
	)
	flag.Parse()

	// Handle version flag
	if *version {
		fmt.Printf("%s %s\n", appName, appVersion)
		os.Exit(0)
	}

	// Handle health check flag
	if *health {
		performHealthCheck()
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
	}

	// Initialize dependencies
	logger := testing.NewMockLogger()
	metrics := testing.NewMockMetricsCollector()
	publisher := testing.NewMockEventPublisher()

	// Create service registry
	registry := services.NewServiceRegistry(cfg, logger, metrics, publisher)

	// Initialize services
	if err := registry.Initialize(); err != nil {
		log.Fatalf("Failed to initialize services: %v", err)
	}

	// Start services
	ctx := context.Background()
	if err := registry.Start(ctx); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}

	// Create HTTP handlers
	httpHandlers := handlers.NewHTTPHandlers(registry, logger, metrics)

	// Setup HTTP server
	mux := http.NewServeMux()
	httpHandlers.RegisterRoutes(mux)

	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:      mux,
		ReadTimeout:  cfg.Server.ReadTimeout,
		WriteTimeout: cfg.Server.WriteTimeout,
		IdleTimeout:  cfg.Server.IdleTimeout,
	}

	// Start server in a goroutine
	go func() {
		logger.Info("Starting HTTP server", "port", cfg.Server.Port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down server...")

	// Graceful shutdown
	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	// Stop services
	if err := registry.Stop(ctx); err != nil {
		log.Printf("Error stopping services: %v", err)
	}

	logger.Info("Server stopped")
}

func performHealthCheck() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	resp, err := client.Get("http://localhost:8080/api/v1/health")
	if err != nil {
		fmt.Printf("Health check failed: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		fmt.Println("Health check passed")
	} else {
		fmt.Printf("Health check failed with status: %d\n", resp.StatusCode)
		os.Exit(1)
	}
}
