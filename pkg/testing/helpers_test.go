package testing

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTestHelper(t *testing.T) {
	helper := NewTestHelper(t)

	t.Run("Context creation", func(t *testing.T) {
		ctx := helper.Context()
		assert.NotNil(t, ctx)

		// Test context with custom timeout
		ctx2 := helper.ContextWithTimeout(5 * time.Second)
		assert.NotNil(t, ctx2)
	})

	t.Run("Assertions", func(t *testing.T) {
		helper.AssertNoError(nil)
		helper.AssertEqual("test", "test")
		helper.RequireNoError(nil)
		helper.RequireEqual(123, 123)
	})
}

func TestMockOrder(t *testing.T) {
	order := GenerateMockOrder()
	
	assert.NotEmpty(t, order.ID)
	assert.NotEmpty(t, order.UserID)
	assert.NotEmpty(t, order.Symbol)
	assert.Contains(t, []string{"BUY", "SELL"}, order.Side)
	assert.Contains(t, []string{"MARKET", "LIMIT"}, order.Type)
	assert.Greater(t, order.Quantity, 0.0)
	assert.Greater(t, order.Price, 0.0)
}

func TestMockRiskProfile(t *testing.T) {
	profile := GenerateMockRiskProfile()
	
	assert.NotEmpty(t, profile.UserID)
	assert.Contains(t, []string{"LOW", "MEDIUM", "HIGH"}, profile.RiskTolerance)
	assert.Greater(t, profile.MaxPositionSize, 0.0)
	assert.Greater(t, profile.MaxDailyLoss, 0.0)
	assert.NotEmpty(t, profile.AllowedAssets)
}

func TestPerformanceBenchmark(t *testing.T) {
	benchmark := NewPerformanceBenchmark("test_operation")
	
	// Simulate some work
	time.Sleep(10 * time.Millisecond)
	
	duration := benchmark.Stop()
	assert.Greater(t, duration, 10*time.Millisecond)
	assert.Less(t, duration, 100*time.Millisecond)
	
	// Test latency assertion
	benchmark.AssertLatency(t, 100*time.Millisecond)
	benchmark.RequireLatency(t, 100*time.Millisecond)
}

func TestTestFixtures(t *testing.T) {
	fixtures := NewTestFixtures(5, 3)
	
	assert.Len(t, fixtures.Orders, 5)
	assert.Len(t, fixtures.RiskProfiles, 3)
	
	// Verify all orders are valid
	for _, order := range fixtures.Orders {
		assert.NotEmpty(t, order.ID)
		assert.NotEmpty(t, order.UserID)
	}
	
	// Verify all risk profiles are valid
	for _, profile := range fixtures.RiskProfiles {
		assert.NotEmpty(t, profile.UserID)
		assert.NotEmpty(t, profile.RiskTolerance)
	}
}

func TestMockDatabase(t *testing.T) {
	db := NewMockDatabase()
	
	// Test order operations
	order := GenerateMockOrder()
	err := db.SaveOrder(order)
	require.NoError(t, err)
	
	retrievedOrder, err := db.GetOrder(order.ID)
	require.NoError(t, err)
	assert.Equal(t, order.ID, retrievedOrder.ID)
	assert.Equal(t, order.UserID, retrievedOrder.UserID)
	
	// Test risk profile operations
	profile := GenerateMockRiskProfile()
	err = db.SaveRiskProfile(profile)
	require.NoError(t, err)
	
	retrievedProfile, err := db.GetRiskProfile(profile.UserID)
	require.NoError(t, err)
	assert.Equal(t, profile.UserID, retrievedProfile.UserID)
	assert.Equal(t, profile.RiskTolerance, retrievedProfile.RiskTolerance)
	
	// Test cleanup
	db.Cleanup()
	_, err = db.GetOrder(order.ID)
	assert.Error(t, err)
	_, err = db.GetRiskProfile(profile.UserID)
	assert.Error(t, err)
}

func TestIntegrationTestSuite(t *testing.T) {
	suite := NewIntegrationTestSuite(t)
	
	// Test setup
	suite.Setup()
	
	// Verify fixtures are loaded
	assert.NotNil(t, suite.Helper())
	assert.NotNil(t, suite.Fixtures())
	assert.NotNil(t, suite.Database())
	
	// Test that fixtures are accessible through database
	for _, order := range suite.Fixtures().Orders {
		retrievedOrder, err := suite.Database().GetOrder(order.ID)
		suite.Helper().RequireNoError(err)
		suite.Helper().AssertEqual(order.ID, retrievedOrder.ID)
	}
	
	// Test teardown
	suite.Teardown()
	
	// Verify cleanup worked
	for _, order := range suite.Fixtures().Orders {
		_, err := suite.Database().GetOrder(order.ID)
		assert.Error(t, err)
	}
}

// Benchmark tests
func BenchmarkGenerateMockOrder(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateMockOrder()
	}
}

func BenchmarkGenerateMockRiskProfile(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = GenerateMockRiskProfile()
	}
}

func BenchmarkMockDatabaseOperations(b *testing.B) {
	db := NewMockDatabase()
	orders := make([]*MockOrder, 1000)
	for i := 0; i < 1000; i++ {
		orders[i] = GenerateMockOrder()
	}
	
	b.ResetTimer()
	
	for i := 0; i < b.N; i++ {
		order := orders[i%1000]
		_ = db.SaveOrder(order)
		_, _ = db.GetOrder(order.ID)
	}
}
