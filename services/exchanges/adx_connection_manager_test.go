package exchanges

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap/zaptest"
)

// ADXConfig represents ADX configuration (mock for testing)
type ADXConfig struct {
	MaxConnections int
	Timeout        time.Duration
}

func TestADXConnectionManager_NewADXConnectionManager(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}

	manager := NewADXConnectionManager(config, logger)

	if manager == nil {
		t.Fatal("Expected connection manager to be created")
	}

	if manager.config != config {
		t.Error("Expected config to be set")
	}

	if manager.connections == nil {
		t.Error("Expected connections map to be initialized")
	}

	if manager.connectionPool == nil {
		t.Error("Expected connection pool to be initialized")
	}
}

func TestADXConnectionManager_Connect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Test connecting market data connection
	connection, err := manager.Connect(ctx, ConnectionTypeMarketData)
	if err != nil {
		t.Errorf("Expected no error on connect, got %v", err)
	}

	if connection == nil {
		t.Fatal("Expected connection to be created")
	}

	if connection.Type != ConnectionTypeMarketData {
		t.Errorf("Expected connection type %v, got %v", ConnectionTypeMarketData, connection.Type)
	}

	if connection.Status != ConnectionStatusConnected {
		t.Errorf("Expected connection status %v, got %v", ConnectionStatusConnected, connection.Status)
	}

	// Verify connection is stored
	storedConnection, err := manager.GetConnection(connection.ID)
	if err != nil {
		t.Errorf("Expected no error getting connection, got %v", err)
	}

	if storedConnection.ID != connection.ID {
		t.Error("Expected stored connection to match created connection")
	}
}

func TestADXConnectionManager_ConnectMultipleTypes(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	connectionTypes := []ConnectionType{
		ConnectionTypeMarketData,
		ConnectionTypeTrading,
		ConnectionTypeReference,
		ConnectionTypeCompliance,
	}

	connections := make([]*ADXConnection, 0, len(connectionTypes))

	// Connect all types
	for _, connType := range connectionTypes {
		connection, err := manager.Connect(ctx, connType)
		if err != nil {
			t.Errorf("Expected no error connecting %v, got %v", connType, err)
			continue
		}
		connections = append(connections, connection)
	}

	// Verify all connections
	if len(connections) != len(connectionTypes) {
		t.Errorf("Expected %d connections, got %d", len(connectionTypes), len(connections))
	}

	// Test GetConnectionsByType
	for _, connType := range connectionTypes {
		typeConnections := manager.GetConnectionsByType(connType)
		if len(typeConnections) != 1 {
			t.Errorf("Expected 1 connection of type %v, got %d", connType, len(typeConnections))
		}
	}
}

func TestADXConnectionManager_Disconnect(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Connect first
	connection, err := manager.Connect(ctx, ConnectionTypeMarketData)
	if err != nil {
		t.Fatalf("Failed to connect: %v", err)
	}

	// Disconnect
	err = manager.Disconnect(ctx, connection.ID)
	if err != nil {
		t.Errorf("Expected no error on disconnect, got %v", err)
	}

	// Verify connection is removed
	_, err = manager.GetConnection(connection.ID)
	if err == nil {
		t.Error("Expected error getting disconnected connection")
	}

	// Test disconnecting non-existent connection
	err = manager.Disconnect(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error disconnecting non-existent connection")
	}
}

func TestADXConnectionManager_GetHealthyConnections(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Connect multiple connections
	numConnections := 3
	for i := 0; i < numConnections; i++ {
		_, err := manager.Connect(ctx, ConnectionTypeMarketData)
		if err != nil {
			t.Errorf("Failed to connect: %v", err)
		}
	}

	// Get healthy connections
	healthyConnections := manager.GetHealthyConnections()
	if len(healthyConnections) != numConnections {
		t.Errorf("Expected %d healthy connections, got %d", numConnections, len(healthyConnections))
	}

	// Verify all are connected
	for _, conn := range healthyConnections {
		if conn.Status != ConnectionStatusConnected {
			t.Errorf("Expected connection status %v, got %v", ConnectionStatusConnected, conn.Status)
		}
	}
}

func TestADXConnectionManager_GetMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Initial metrics
	metrics := manager.GetMetrics()
	if metrics["total_connections"].(int64) != 0 {
		t.Error("Expected initial total connections to be 0")
	}

	if metrics["active_connections"].(int64) != 0 {
		t.Error("Expected initial active connections to be 0")
	}

	// Connect some connections
	numConnections := 2
	for i := 0; i < numConnections; i++ {
		_, err := manager.Connect(ctx, ConnectionTypeMarketData)
		if err != nil {
			t.Errorf("Failed to connect: %v", err)
		}
	}

	// Check updated metrics
	metrics = manager.GetMetrics()
	if metrics["total_connections"].(int64) != int64(numConnections) {
		t.Errorf("Expected total connections %d, got %v", numConnections, metrics["total_connections"])
	}

	if metrics["active_connections"].(int64) != int64(numConnections) {
		t.Errorf("Expected active connections %d, got %v", numConnections, metrics["active_connections"])
	}

	// Check connection types
	connectionTypes := metrics["connection_types"].(map[string]int)
	if connectionTypes["market_data"] != numConnections {
		t.Errorf("Expected %d market data connections, got %d", numConnections, connectionTypes["market_data"])
	}
}

func TestADXConnectionManager_Close(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 10,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Connect multiple connections
	numConnections := 3
	connectionIDs := make([]string, 0, numConnections)
	for i := 0; i < numConnections; i++ {
		connection, err := manager.Connect(ctx, ConnectionTypeMarketData)
		if err != nil {
			t.Errorf("Failed to connect: %v", err)
			continue
		}
		connectionIDs = append(connectionIDs, connection.ID)
	}

	// Close manager
	err := manager.Close(ctx)
	if err != nil {
		t.Errorf("Expected no error on close, got %v", err)
	}

	// Verify all connections are closed
	for _, connectionID := range connectionIDs {
		_, err := manager.GetConnection(connectionID)
		if err == nil {
			t.Error("Expected connection to be closed")
		}
	}

	// Check metrics
	metrics := manager.GetMetrics()
	if metrics["active_connections"].(int64) != 0 {
		t.Error("Expected no active connections after close")
	}
}

func TestConnectionType_String(t *testing.T) {
	tests := []struct {
		connType ConnectionType
		expected string
	}{
		{ConnectionTypeMarketData, "market_data"},
		{ConnectionTypeTrading, "trading"},
		{ConnectionTypeReference, "reference"},
		{ConnectionTypeCompliance, "compliance"},
		{ConnectionType(999), "unknown"},
	}

	for _, test := range tests {
		if test.connType.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.connType.String())
		}
	}
}

func TestConnectionStatus_String(t *testing.T) {
	tests := []struct {
		status   ConnectionStatus
		expected string
	}{
		{ConnectionStatusDisconnected, "disconnected"},
		{ConnectionStatusConnecting, "connecting"},
		{ConnectionStatusConnected, "connected"},
		{ConnectionStatusReconnecting, "reconnecting"},
		{ConnectionStatusError, "error"},
		{ConnectionStatus(999), "unknown"},
	}

	for _, test := range tests {
		if test.status.String() != test.expected {
			t.Errorf("Expected %s, got %s", test.expected, test.status.String())
		}
	}
}

func TestADXConnectionManager_ConcurrentConnections(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := &ADXConfig{
		MaxConnections: 100,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	numGoroutines := 10
	numConnectionsPerGoroutine := 5
	done := make(chan bool, numGoroutines)

	// Connect concurrently
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer func() { done <- true }()
			for j := 0; j < numConnectionsPerGoroutine; j++ {
				_, err := manager.Connect(ctx, ConnectionTypeMarketData)
				if err != nil {
					t.Errorf("Failed to connect: %v", err)
				}
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify total connections
	expectedTotal := numGoroutines * numConnectionsPerGoroutine
	metrics := manager.GetMetrics()
	if metrics["total_connections"].(int64) != int64(expectedTotal) {
		t.Errorf("Expected %d total connections, got %v", expectedTotal, metrics["total_connections"])
	}
}

func BenchmarkADXConnectionManager_Connect(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := &ADXConfig{
		MaxConnections: 1000,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.Connect(ctx, ConnectionTypeMarketData)
		if err != nil {
			b.Errorf("Failed to connect: %v", err)
		}
	}
}

func BenchmarkADXConnectionManager_GetConnection(b *testing.B) {
	logger := zaptest.NewLogger(b)
	config := &ADXConfig{
		MaxConnections: 1000,
		Timeout:        30 * time.Second,
	}
	manager := NewADXConnectionManager(config, logger)
	ctx := context.Background()

	// Create a connection
	connection, err := manager.Connect(ctx, ConnectionTypeMarketData)
	if err != nil {
		b.Fatalf("Failed to connect: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := manager.GetConnection(connection.ID)
		if err != nil {
			b.Errorf("Failed to get connection: %v", err)
		}
	}
}
