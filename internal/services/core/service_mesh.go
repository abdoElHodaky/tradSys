// Package core provides the foundational service mesh infrastructure for TradSys v3
// This implements Plan 4: Services Architecture with unified microservices mesh
package core

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/reflection"
)

// ServiceMesh represents the unified service mesh for TradSys v3
type ServiceMesh struct {
	services         map[string]*ServiceNode
	serviceDiscovery *ServiceDiscovery
	loadBalancer     *LoadBalancer
	healthChecker    *HealthChecker
	metricsCollector *MetricsCollector
	tlsConfig        *tls.Config
	mu               sync.RWMutex
}

// ServiceNode represents a single service in the mesh
type ServiceNode struct {
	ID          string
	Name        string
	Address     string
	Port        int
	ServiceType ServiceType
	Status      ServiceStatus
	Metadata    map[string]string
	Server      *grpc.Server
	Conn        *grpc.ClientConn
	LastSeen    time.Time
	HealthCheck *grpc_health_v1.HealthClient
}

// ServiceType defines the type of service in the mesh
type ServiceType int

const (
	ServiceTypeCore ServiceType = iota
	ServiceTypeExchange
	ServiceTypeRouting
	ServiceTypeWebSocket
	ServiceTypeLicensing
	ServiceTypeIslamicFinance
	ServiceTypeAsset
	ServiceTypeAnalytics
)

// ServiceStatus represents the current status of a service
type ServiceStatus int

const (
	ServiceStatusUnknown ServiceStatus = iota
	ServiceStatusHealthy
	ServiceStatusUnhealthy
	ServiceStatusStarting
	ServiceStatusStopping
)

// NewServiceMesh creates a new service mesh instance
func NewServiceMesh() *ServiceMesh {
	return &ServiceMesh{
		services:         make(map[string]*ServiceNode),
		serviceDiscovery: NewServiceDiscovery(),
		loadBalancer:     NewLoadBalancer(),
		healthChecker:    NewHealthChecker(),
		metricsCollector: NewMetricsCollector(),
		tlsConfig:        createMTLSConfig(),
	}
}

// RegisterService registers a new service in the mesh
func (sm *ServiceMesh) RegisterService(ctx context.Context, service *ServiceNode) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	// Validate service configuration
	if err := sm.validateService(service); err != nil {
		return fmt.Errorf("service validation failed: %w", err)
	}

	// Create gRPC server with mTLS
	server := grpc.NewServer(
		grpc.Creds(credentials.NewTLS(sm.tlsConfig)),
		grpc.KeepaliveParams(keepalive.ServerParameters{
			MaxConnectionIdle:     15 * time.Second,
			MaxConnectionAge:      30 * time.Second,
			MaxConnectionAgeGrace: 5 * time.Second,
			Time:                  5 * time.Second,
			Timeout:               1 * time.Second,
		}),
		grpc.KeepaliveEnforcementPolicy(keepalive.EnforcementPolicy{
			MinTime:             5 * time.Second,
			PermitWithoutStream: true,
		}),
	)

	// Register health service
	healthServer := health.NewServer()
	grpc_health_v1.RegisterHealthServer(server, healthServer)

	// Enable reflection for development
	reflection.Register(server)

	service.Server = server
	service.Status = ServiceStatusStarting
	service.LastSeen = time.Now()

	// Start the service
	go sm.startService(ctx, service)

	// Register with service discovery
	if err := sm.serviceDiscovery.Register(service); err != nil {
		return fmt.Errorf("service discovery registration failed: %w", err)
	}

	// Add to services map
	sm.services[service.ID] = service

	log.Printf("Service registered: %s (%s) at %s:%d",
		service.Name, service.ID, service.Address, service.Port)

	return nil
}

// GetServicesByType retrieves all services of a specific type
func (sm *ServiceMesh) GetServicesByType(serviceType ServiceType) []*ServiceNode {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	var services []*ServiceNode
	for _, service := range sm.services {
		if service.ServiceType == serviceType {
			services = append(services, service)
		}
	}

	return services
}

// startService starts a service node
func (sm *ServiceMesh) startService(ctx context.Context, service *ServiceNode) {
	address := fmt.Sprintf("%s:%d", service.Address, service.Port)
	listener, err := net.Listen("tcp", address)
	if err != nil {
		log.Printf("Failed to start service %s: %v", service.Name, err)
		service.Status = ServiceStatusUnhealthy
		return
	}

	service.Status = ServiceStatusHealthy
	log.Printf("Service %s started on %s", service.Name, address)

	if err := service.Server.Serve(listener); err != nil {
		log.Printf("Service %s stopped: %v", service.Name, err)
		service.Status = ServiceStatusUnhealthy
	}
}

// validateService validates service configuration
func (sm *ServiceMesh) validateService(service *ServiceNode) error {
	if service.ID == "" {
		return fmt.Errorf("service ID is required")
	}
	if service.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if service.Address == "" {
		return fmt.Errorf("service address is required")
	}
	if service.Port <= 0 || service.Port > 65535 {
		return fmt.Errorf("invalid service port: %d", service.Port)
	}

	// Check for duplicate service ID
	if _, exists := sm.services[service.ID]; exists {
		return fmt.Errorf("service ID already exists: %s", service.ID)
	}

	return nil
}

// createMTLSConfig creates mutual TLS configuration for secure service communication
func createMTLSConfig() *tls.Config {
	// In production, load actual certificates
	// For now, using a basic TLS config
	return &tls.Config{
		MinVersion: tls.VersionTLS12,
		CipherSuites: []uint16{
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
		},
		PreferServerCipherSuites: true,
		InsecureSkipVerify:       false, // Set to false in production
	}
}

// Shutdown gracefully shuts down the service mesh
func (sm *ServiceMesh) Shutdown(ctx context.Context) error {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	log.Println("Shutting down service mesh...")

	for _, service := range sm.services {
		service.Status = ServiceStatusStopping
		if service.Server != nil {
			service.Server.GracefulStop()
		}
		if service.Conn != nil {
			service.Conn.Close()
		}
	}

	log.Println("Service mesh shutdown complete")
	return nil
}
