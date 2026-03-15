package service

import (
	"time"

	"github.com/sony/gobreaker/v2"
)

// CircuitBreakerService manages circuit breakers for different dependencies
type CircuitBreakerService struct {
	redis    *gobreaker.CircuitBreaker[interface{}]
	database *gobreaker.CircuitBreaker[interface{}]
	external *gobreaker.CircuitBreaker[interface{}]
}

// NewCircuitBreakerService creates a new circuit breaker service with configured breakers
func NewCircuitBreakerService() *CircuitBreakerService {
	// Redis circuit breaker - more aggressive (fail fast for caching)
	redisSettings := gobreaker.Settings{
		Name:        "Redis",
		MaxRequests: 3,                // Allow 3 test requests in half-open state
		Interval:    10 * time.Second, // Count failures over 10s window
		Timeout:     30 * time.Second, // Wait 30s before trying again
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if 5 failures occur
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && (counts.TotalFailures >= 5 || failureRatio >= 0.6)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			// Log state changes for monitoring
			println("Circuit breaker", name, "changed from", from.String(), "to", to.String())
		},
	}

	// Database circuit breaker - less aggressive (critical service)
	dbSettings := gobreaker.Settings{
		Name:        "Database",
		MaxRequests: 5,                // Allow more test requests
		Interval:    15 * time.Second, // Longer failure window
		Timeout:     60 * time.Second, // Wait longer before retry
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			// Trip if 10 failures or 80% failure rate
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 5 && (counts.TotalFailures >= 10 || failureRatio >= 0.8)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			println("Circuit breaker", name, "changed from", from.String(), "to", to.String())
		},
	}

	// External API circuit breaker - moderate settings
	externalSettings := gobreaker.Settings{
		Name:        "ExternalAPI",
		MaxRequests: 3,
		Interval:    10 * time.Second,
		Timeout:     45 * time.Second,
		ReadyToTrip: func(counts gobreaker.Counts) bool {
			failureRatio := float64(counts.TotalFailures) / float64(counts.Requests)
			return counts.Requests >= 3 && (counts.TotalFailures >= 7 || failureRatio >= 0.7)
		},
		OnStateChange: func(name string, from, to gobreaker.State) {
			println("Circuit breaker", name, "changed from", from.String(), "to", to.String())
		},
	}

	return &CircuitBreakerService{
		redis:    gobreaker.NewCircuitBreaker[interface{}](redisSettings),
		database: gobreaker.NewCircuitBreaker[interface{}](dbSettings),
		external: gobreaker.NewCircuitBreaker[interface{}](externalSettings),
	}
}

// ExecuteRedis wraps Redis operations with circuit breaker
func (cb *CircuitBreakerService) ExecuteRedis(operation func() (interface{}, error)) (interface{}, error) {
	return cb.redis.Execute(operation)
}

// ExecuteDatabase wraps database operations with circuit breaker
func (cb *CircuitBreakerService) ExecuteDatabase(operation func() (interface{}, error)) (interface{}, error) {
	return cb.database.Execute(operation)
}

// ExecuteExternal wraps external API calls with circuit breaker
func (cb *CircuitBreakerService) ExecuteExternal(operation func() (interface{}, error)) (interface{}, error) {
	return cb.external.Execute(operation)
}

// GetRedisState returns the current state of the Redis circuit breaker
func (cb *CircuitBreakerService) GetRedisState() gobreaker.State {
	return cb.redis.State()
}

// GetDatabaseState returns the current state of the Database circuit breaker
func (cb *CircuitBreakerService) GetDatabaseState() gobreaker.State {
	return cb.database.State()
}

// GetExternalState returns the current state of the External API circuit breaker
func (cb *CircuitBreakerService) GetExternalState() gobreaker.State {
	return cb.external.State()
}

// GetStats returns a map of circuit breaker states for health checks
func (cb *CircuitBreakerService) GetStats() map[string]string {
	return map[string]string{
		"redis":    cb.redis.State().String(),
		"database": cb.database.State().String(),
		"external": cb.external.State().String(),
	}
}
