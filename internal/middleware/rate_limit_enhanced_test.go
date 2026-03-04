package middleware

import (
	"context"
	"os"
	"testing"
	"time"

	"license-management-api/pkg/utils"
	"github.com/stretchr/testify/assert"
)

// TestEnhancedRedisRateLimiter tests the basic rate limiting functionality
func TestEnhancedRedisRateLimiter_BasicRateLimiting(t *testing.T) {
	// Create an in-memory Redis instance for testing
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	config := RateLimitConfig{
		MaxAttempts:     3,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	}

	limiter := NewEnhancedRedisRateLimiter(redisClient, config)

	testIP := "192.168.1.1"
	attempts := 0

	// First 3 attempts should be allowed
	for i := 1; i <= 3; i++ {
		allowed, _ := limiter.IsAllowed(testIP)
		assert.True(t, allowed, "attempt %d should be allowed", i)

		// Record failure
		remaining := limiter.RecordFailure(testIP)
		assert.Equal(t, 3-i, remaining, "remaining attempts should decrease")
		attempts++
	}

	// 4th attempt should be blocked (rate limited)
	allowed, remainingTime := limiter.IsAllowed(testIP)
	assert.False(t, allowed, "4th attempt should be blocked")
	assert.Greater(t, remainingTime.Seconds(), float64(0), "remaining time should be > 0")

	// Verify metrics
	metrics := limiter.GetMetrics()
	assert.Equal(t, int64(1), metrics.TotalBlocked, "one request should be blocked")
	assert.Equal(t, int64(3), metrics.TotalAllowed, "three requests should be allowed")
}

// TestEnhancedRedisRateLimiter_Whitelist tests IP whitelist functionality
func TestEnhancedRedisRateLimiter_Whitelist(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	})

	whitelistedIP := "10.0.0.1"
	limiter.AddToWhitelist(whitelistedIP)

	// Whitelisted IP should always be allowed, even after max attempts
	for i := 1; i <= 5; i++ {
		allowed, _ := limiter.IsAllowed(whitelistedIP)
		assert.True(t, allowed, "whitelisted IP should always be allowed at attempt %d", i)
		limiter.RecordFailure(whitelistedIP)
	}

	// Remove from whitelist
	limiter.RemoveFromWhitelist(whitelistedIP)

	// Now it should be blocked
	allowed, _ := limiter.IsAllowed(whitelistedIP)
	assert.False(t, allowed, "IP should be blocked after removal from whitelist")
}

// TestEnhancedRedisRateLimiter_RecordSuccess verifies counter behavior without success reset
// Note: With the security fix, successful login attempts no longer reset the rate limit counter.
// This test verifies we can check what the counter value is and that it's maintained over time.
func TestEnhancedRedisRateLimiter_RecordSuccess(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     5,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	})

	testIP := "192.168.1.2"

	// Record 2 failures - counter increments
	limiter.RecordFailure(testIP)
	limiter.RecordFailure(testIP)

	// Verify counter is at 2
	val, err := redisClient.Get(limiter.config.KeyPrefix + ":attempts:" + testIP)
	assert.NoError(t, err, "should be able to get counter value")
	assert.Equal(t, "2", val, "counter should be 2 after 2 failures")

	// With security fix: successful logins don't reset counter
	// In real use, we don't call RecordSuccess() for login handler anymore
	// This test just verifies the counter persists as expected

	// Verify counter persists - still at 2
	val, err = redisClient.Get(limiter.config.KeyPrefix + ":attempts:" + testIP)
	assert.NoError(t, err, "should still be able to get counter value")
	assert.Equal(t, "2", val, "counter should persist at 2 (not cleared by security fix)")

	// Next check should still allow request (2/5)
	allowed, _ := limiter.IsAllowed(testIP)
	assert.True(t, allowed, "attempt should be allowed - only at 2/5 limit")
}

// TestEnhancedRedisRateLimiter_MultipleIPs tests isolation between different IPs
func TestEnhancedRedisRateLimiter_MultipleIPs(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	})

	ip1 := "192.168.1.1"
	ip2 := "192.168.1.2"

	// IP1: Record 2 failures (should be blocked on 3rd attempt)
	limiter.RecordFailure(ip1)
	limiter.RecordFailure(ip1)

	// IP2: Record 1 failure (should still be allowed)
	limiter.RecordFailure(ip2)

	// IP1 should be blocked
	allowed1, _ := limiter.IsAllowed(ip1)
	assert.False(t, allowed1, "IP1 should be blocked")

	// IP2 should still be allowed
	allowed2, _ := limiter.IsAllowed(ip2)
	assert.True(t, allowed2, "IP2 should still be allowed")
}

// TestEnhancedRedisRateLimiter_LockoutExpiration tests that lockout expires
func TestEnhancedRedisRateLimiter_LockoutExpiration(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 1 * time.Second, // Short lockout for testing
		ResetWindow:     1 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	})

	testIP := "192.168.1.3"

	// Record 2 failures to trigger lockout
	limiter.RecordFailure(testIP)
	limiter.RecordFailure(testIP)

	// Should be locked
	allowed, _ := limiter.IsAllowed(testIP)
	assert.False(t, allowed, "should be locked after max attempts")

	// Wait for lockout to expire
	time.Sleep(1100 * time.Millisecond)

	// Should be allowed again
	allowed, _ = limiter.IsAllowed(testIP)
	assert.True(t, allowed, "should be allowed after lockout expires")
}

// TestRateLimitByEndpoint tests the helper function
func TestRateLimitByEndpoint(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := RateLimitByEndpoint(redisClient, 5, 15*time.Minute, "login attempts")

	assert.NotNil(t, limiter)
	assert.Equal(t, 5, limiter.config.MaxAttempts)
	assert.Equal(t, 15*time.Minute, limiter.config.LockoutDuration)
	assert.Equal(t, "login attempts", limiter.config.Description)
}

// TestEnhancedRedisRateLimiter_Metrics tests metric tracking
func TestEnhancedRedisRateLimiter_Metrics(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit",
		Description:     "test attempts",
	})

	ip1 := "192.168.1.4"
	ip2 := "192.168.1.5"

	// Allow some requests
	limiter.IsAllowed(ip1)
	limiter.IsAllowed(ip2)

	// Record failures
	limiter.RecordFailure(ip1)
	limiter.RecordFailure(ip1)
	limiter.RecordFailure(ip2)

	// Block ip1
	limiter.IsAllowed(ip1)

	metrics := limiter.GetMetrics()
	assert.Greater(t, metrics.TotalAllowed, int64(0), "should have allowed requests")
	assert.Greater(t, metrics.TotalBlocked, int64(0), "should have blocked requests")
	assert.Greater(t, metrics.LastBlockedTime.Unix(), int64(0), "should have last blocked time")
}

// Helper function to create a test Redis client
func createTestRedisClient(t *testing.T) *utils.RedisClient {
	// Set Redis env vars for testing
	if os.Getenv("REDIS_PASSWORD") == "" {
		os.Setenv("REDIS_PASSWORD", "Admin@123")
	}
	if os.Getenv("REDIS_USERNAME") == "" {
		os.Setenv("REDIS_USERNAME", "default")
	}
	if os.Getenv("REDIS_PORT") == "" {
		os.Setenv("REDIS_PORT", "6379")
	}

	// Use local Redis for testing; fail if not available
	ctx := context.Background()
	client, err := utils.NewRedisClient(ctx)
	if err != nil {
		t.Skipf("Skipping test: Redis not available (%v)", err)
	}

	// Flush test data
	_ = client.FlushDB()

	return client
}

// TestEnhancedRedisRateLimiter_ConcurrentRequests tests thread safety
func TestEnhancedRedisRateLimiter_ConcurrentRequests(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     10,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:ratelimit:concurrent",
		Description:     "concurrent test",
	})

	testIP := "192.168.1.6"
	done := make(chan struct{}, 20)

	// Simulate 20 concurrent requests from same IP
	for i := 0; i < 20; i++ {
		go func() {
			limiter.IsAllowed(testIP)
			limiter.RecordFailure(testIP)
			done <- struct{}{}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 20; i++ {
		<-done
	}

	// Should be rate limited after 10 attempts
	allowed, _ := limiter.IsAllowed(testIP)
	assert.False(t, allowed, "should be rate limited after concurrent attempts")
}
