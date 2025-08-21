package resilience

import (
	"context"
	"errors"
	"time"

	"github.com/sony/gobreaker"
	"go.uber.org/zap"
)

// This file provides example usage of the circuit breaker components
// It is not meant to be used in production, but rather to demonstrate
// how to use the circuit breaker components in a way that follows Fx benefits

// ExampleService demonstrates how to use the circuit breaker in a service
type ExampleService struct {
	logger            *zap.Logger
	circuitBreaker    *CircuitBreakerFactory
	externalAPIClient *ExternalAPIClient
}

// NewExampleService creates a new example service
// This follows Fx's dependency injection pattern
func NewExampleService(
	logger *zap.Logger,
	circuitBreaker *CircuitBreakerFactory,
	externalAPIClient *ExternalAPIClient,
) *ExampleService {
	return &ExampleService{
		logger:            logger,
		circuitBreaker:    circuitBreaker,
		externalAPIClient: externalAPIClient,
	}
}

// GetUserData demonstrates how to use the circuit breaker to protect an external API call
func (s *ExampleService) GetUserData(ctx context.Context, userID string) (interface{}, error) {
	// Create a custom circuit breaker for this specific API
	cbConfig := CircuitBreakerConfig{
		Name:        "get-user-data",
		MaxRequests: 2,
		Interval:    time.Minute,
		Timeout:     10 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip when error rate is over 50% with at least 5 requests
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && failureRatio >= 0.5
		},
	}
	s.circuitBreaker.CreateCustomCircuitBreaker(cbConfig)

	// Execute the API call with the circuit breaker and fallback
	result := s.circuitBreaker.ExecuteContextWithFallback(
		ctx,
		"get-user-data",
		func(ctx context.Context) (interface{}, error) {
			// This is the primary function that will be protected by the circuit breaker
			return s.externalAPIClient.GetUserData(ctx, userID)
		},
		func(ctx context.Context, err error) (interface{}, error) {
			// This is the fallback function that will be called if the primary function fails
			s.logger.Warn("Falling back to cached user data",
				zap.String("user_id", userID),
				zap.Error(err))
			
			// Return cached data or a default value
			return map[string]interface{}{
				"user_id": userID,
				"name":    "Unknown",
				"email":   "unknown@example.com",
				"cached":  true,
			}, nil
		},
	)

	// Check the result
	if result.Error != nil {
		s.logger.Error("Failed to get user data",
			zap.String("user_id", userID),
			zap.Error(result.Error))
		return nil, result.Error
	}

	s.logger.Info("Successfully got user data",
		zap.String("user_id", userID),
		zap.Duration("duration", result.Duration),
		zap.String("breaker_state", stateToString(result.State)))

	return result.Value, nil
}

// ExternalAPIClient is a mock external API client
type ExternalAPIClient struct {
	logger *zap.Logger
}

// NewExternalAPIClient creates a new external API client
func NewExternalAPIClient(logger *zap.Logger) *ExternalAPIClient {
	return &ExternalAPIClient{
		logger: logger,
	}
}

// GetUserData simulates an external API call to get user data
func (c *ExternalAPIClient) GetUserData(ctx context.Context, userID string) (interface{}, error) {
	// Simulate a slow or failing API call
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(100 * time.Millisecond):
		// Simulate a random failure
		if time.Now().UnixNano()%5 == 0 {
			return nil, errors.New("external API error")
		}

		// Return mock user data
		return map[string]interface{}{
			"user_id": userID,
			"name":    "John Doe",
			"email":   "john.doe@example.com",
		}, nil
	}
}

