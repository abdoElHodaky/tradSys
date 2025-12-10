// Package testing provides common testing utilities and helpers for TradSys
package testing

import (
	"context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHelper provides common testing utilities
type TestHelper struct {
	t *testing.T
}

// NewTestHelper creates a new test helper instance
func NewTestHelper(t *testing.T) *TestHelper {
	return &TestHelper{t: t}
}

// Context creates a test context with timeout
func (h *TestHelper) Context() context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	h.t.Cleanup(cancel) // Ensure cancel is called when test completes
	return ctx
}

// ContextWithTimeout creates a test context with custom timeout
func (h *TestHelper) ContextWithTimeout(timeout time.Duration) context.Context {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	h.t.Cleanup(cancel) // Ensure cancel is called when test completes
	return ctx
}

// AssertNoError asserts that error is nil
func (h *TestHelper) AssertNoError(err error, msgAndArgs ...interface{}) {
	assert.NoError(h.t, err, msgAndArgs...)
}

// RequireNoError requires that error is nil
func (h *TestHelper) RequireNoError(err error, msgAndArgs ...interface{}) {
	require.NoError(h.t, err, msgAndArgs...)
}

// AssertEqual asserts that two values are equal
func (h *TestHelper) AssertEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	assert.Equal(h.t, expected, actual, msgAndArgs...)
}

// RequireEqual requires that two values are equal
func (h *TestHelper) RequireEqual(expected, actual interface{}, msgAndArgs ...interface{}) {
	require.Equal(h.t, expected, actual, msgAndArgs...)
}

// MockOrder creates a mock order for testing
type MockOrder struct {
	ID       string
	UserID   string
	Symbol   string
	Side     string
	Type     string
	Quantity float64
	Price    float64
}

// GenerateMockOrder creates a random mock order
func GenerateMockOrder() *MockOrder {
	symbols := []string{"AAPL", "GOOGL", "MSFT", "TSLA", "AMZN"}
	sides := []string{"BUY", "SELL"}
	types := []string{"MARKET", "LIMIT"}

	return &MockOrder{
		ID:       fmt.Sprintf("order_%d", rand.Int63()),
		UserID:   fmt.Sprintf("user_%d", rand.Int63()),
		Symbol:   symbols[rand.Intn(len(symbols))],
		Side:     sides[rand.Intn(len(sides))],
		Type:     types[rand.Intn(len(types))],
		Quantity: float64(rand.Intn(1000) + 1),
		Price:    float64(rand.Intn(10000)+1000) / 100.0,
	}
}

// MockRiskProfile creates a mock risk profile for testing
type MockRiskProfile struct {
	UserID           string
	RiskTolerance    string
	MaxPositionSize  float64
	MaxDailyLoss     float64
	AllowedAssets    []string
	IslamicCompliant bool
}

// GenerateMockRiskProfile creates a random mock risk profile
func GenerateMockRiskProfile() *MockRiskProfile {
	tolerances := []string{"LOW", "MEDIUM", "HIGH"}
	assets := []string{"STOCKS", "BONDS", "COMMODITIES", "FOREX"}

	return &MockRiskProfile{
		UserID:           fmt.Sprintf("user_%d", rand.Int63()),
		RiskTolerance:    tolerances[rand.Intn(len(tolerances))],
		MaxPositionSize:  float64(rand.Intn(100000) + 10000),
		MaxDailyLoss:     float64(rand.Intn(10000) + 1000),
		AllowedAssets:    assets[:rand.Intn(len(assets))+1],
		IslamicCompliant: rand.Float32() < 0.3, // 30% chance of Islamic compliance
	}
}

// PerformanceBenchmark provides performance testing utilities
type PerformanceBenchmark struct {
	name      string
	startTime time.Time
	endTime   time.Time
}

// NewPerformanceBenchmark creates a new performance benchmark
func NewPerformanceBenchmark(name string) *PerformanceBenchmark {
	return &PerformanceBenchmark{
		name:      name,
		startTime: time.Now(),
	}
}

// Stop stops the benchmark and returns the duration
func (pb *PerformanceBenchmark) Stop() time.Duration {
	pb.endTime = time.Now()
	return pb.Duration()
}

// Duration returns the benchmark duration
func (pb *PerformanceBenchmark) Duration() time.Duration {
	if pb.endTime.IsZero() {
		return time.Since(pb.startTime)
	}
	return pb.endTime.Sub(pb.startTime)
}

// AssertLatency asserts that the benchmark duration is within expected latency
func (pb *PerformanceBenchmark) AssertLatency(t *testing.T, maxLatency time.Duration) {
	duration := pb.Duration()
	assert.True(t, duration <= maxLatency, 
		"Benchmark %s took %v, expected <= %v", pb.name, duration, maxLatency)
}

// RequireLatency requires that the benchmark duration is within expected latency
func (pb *PerformanceBenchmark) RequireLatency(t *testing.T, maxLatency time.Duration) {
	duration := pb.Duration()
	require.True(t, duration <= maxLatency, 
		"Benchmark %s took %v, expected <= %v", pb.name, duration, maxLatency)
}

// TestFixtures provides common test data fixtures
type TestFixtures struct {
	Orders       []*MockOrder
	RiskProfiles []*MockRiskProfile
}

// NewTestFixtures creates a new set of test fixtures
func NewTestFixtures(orderCount, profileCount int) *TestFixtures {
	fixtures := &TestFixtures{
		Orders:       make([]*MockOrder, orderCount),
		RiskProfiles: make([]*MockRiskProfile, profileCount),
	}

	for i := 0; i < orderCount; i++ {
		fixtures.Orders[i] = GenerateMockOrder()
	}

	for i := 0; i < profileCount; i++ {
		fixtures.RiskProfiles[i] = GenerateMockRiskProfile()
	}

	return fixtures
}

// MockDatabase provides a simple in-memory database for testing
type MockDatabase struct {
	orders map[string]*MockOrder
	risks  map[string]*MockRiskProfile
}

// NewMockDatabase creates a new mock database
func NewMockDatabase() *MockDatabase {
	return &MockDatabase{
		orders: make(map[string]*MockOrder),
		risks:  make(map[string]*MockRiskProfile),
	}
}

// SaveOrder saves an order to the mock database
func (db *MockDatabase) SaveOrder(order *MockOrder) error {
	db.orders[order.ID] = order
	return nil
}

// GetOrder retrieves an order from the mock database
func (db *MockDatabase) GetOrder(id string) (*MockOrder, error) {
	order, exists := db.orders[id]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", id)
	}
	return order, nil
}

// SaveRiskProfile saves a risk profile to the mock database
func (db *MockDatabase) SaveRiskProfile(profile *MockRiskProfile) error {
	db.risks[profile.UserID] = profile
	return nil
}

// GetRiskProfile retrieves a risk profile from the mock database
func (db *MockDatabase) GetRiskProfile(userID string) (*MockRiskProfile, error) {
	profile, exists := db.risks[userID]
	if !exists {
		return nil, fmt.Errorf("risk profile not found: %s", userID)
	}
	return profile, nil
}

// Cleanup clears all data from the mock database
func (db *MockDatabase) Cleanup() {
	db.orders = make(map[string]*MockOrder)
	db.risks = make(map[string]*MockRiskProfile)
}

// IntegrationTestSuite provides utilities for integration testing
type IntegrationTestSuite struct {
	t        *testing.T
	helper   *TestHelper
	fixtures *TestFixtures
	db       *MockDatabase
}

// NewIntegrationTestSuite creates a new integration test suite
func NewIntegrationTestSuite(t *testing.T) *IntegrationTestSuite {
	return &IntegrationTestSuite{
		t:        t,
		helper:   NewTestHelper(t),
		fixtures: NewTestFixtures(10, 5), // Default fixtures
		db:       NewMockDatabase(),
	}
}

// Setup performs common setup for integration tests
func (suite *IntegrationTestSuite) Setup() {
	// Load fixtures into mock database
	for _, order := range suite.fixtures.Orders {
		suite.helper.RequireNoError(suite.db.SaveOrder(order))
	}
	for _, profile := range suite.fixtures.RiskProfiles {
		suite.helper.RequireNoError(suite.db.SaveRiskProfile(profile))
	}
}

// Teardown performs common cleanup for integration tests
func (suite *IntegrationTestSuite) Teardown() {
	suite.db.Cleanup()
}

// Helper returns the test helper
func (suite *IntegrationTestSuite) Helper() *TestHelper {
	return suite.helper
}

// Fixtures returns the test fixtures
func (suite *IntegrationTestSuite) Fixtures() *TestFixtures {
	return suite.fixtures
}

// Database returns the mock database
func (suite *IntegrationTestSuite) Database() *MockDatabase {
	return suite.db
}
