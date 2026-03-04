package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

// TestRateLimitMiddleware_HTTP tests the HTTP middleware functionality
func TestRateLimitMiddleware_HTTP(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     3,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:http",
		Description:     "HTTP test",
	})

	testIP := "192.168.1.100"
	var middleware http.Handler

	// Simulate auth handler that processes login requests
	requestCount := 0
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		// Middleware already recorded the attempt via RecordAttempt()
		// Handler processes login without modifying counter to avoid double-counting
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	middleware = limiter.RateLimitMiddleware(nextHandler)

	// First 3 requests should pass middleware and reach handler
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.RemoteAddr = testIP + ":8080"
		w := httptest.NewRecorder()

		middleware.ServeHTTP(w, req)

		// Should pass middleware and reach handler
		assert.Equal(t, http.StatusOK, w.Code, "request %d should reach handler (attempt %d/3)", i+1, i+1)
	}

	// 4th request should be rate limited by middleware (no more attempts allowed)
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = testIP + ":8080"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	// 4th request should be blocked by middleware - rate limit enforced
	// Even with valid credentials, total attempt count is limited
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "4th request should be rate limited (max attempts exceeded)")
}

// TestRateLimitMiddleware_RetryAfterHeader verifies Retry-After header
func TestRateLimitMiddleware_RetryAfterHeader(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     1,
		LockoutDuration: 30 * time.Second,
		ResetWindow:     30 * time.Second,
		KeyPrefix:       "test:header",
		Description:     "header test",
	})

	testIP := "192.168.1.101"

	// Create middleware with a handler that records failure
	failureCount := 0
	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)
		failureCount++
		limiter.RecordFailure(ip)
		w.WriteHeader(http.StatusUnauthorized)
	})
	middleware := limiter.RateLimitMiddleware(nextHandler)

	// First request: Should pass middleware, go to handler, handler records failure
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = testIP + ":8080"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)
	assert.Equal(t, http.StatusUnauthorized, w.Code, "first request should reach handler")

	// Second request: Should be rate limited by middleware (because of lockout from first failure)
	req = httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = testIP + ":8080"
	w = httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusTooManyRequests, w.Code, "second request should be rate limited")
	assert.NotEmpty(t, w.Header().Get("Retry-After"), "Retry-After header should be set")
	retryAfter := w.Header().Get("Retry-After")
	assert.Greater(t, len(retryAfter), 0, "Retry-After should have value")
}

// TestRateLimitMiddleware_ClientIPExtraction tests IP extraction from different sources
func TestRateLimitMiddleware_ClientIPExtraction(t *testing.T) {
	tests := []struct {
		name      string
		remoteAddr string
		xForwardedFor string
		expectedIP string
	}{
		{
			name:      "Direct RemoteAddr",
			remoteAddr: "192.168.1.1:8080",
			expectedIP: "192.168.1.1",
		},
		{
			name:      "X-Forwarded-For header",
			remoteAddr: "127.0.0.1:8080",
			xForwardedFor: "203.0.113.1, 198.51.100.1",
			expectedIP: "203.0.113.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("POST", "/test", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xForwardedFor != "" {
				req.Header.Set("X-Forwarded-For", tt.xForwardedFor)
			}

			ip := GetClientIP(req)
			assert.Equal(t, tt.expectedIP, ip)
		})
	}
}

// TestRateLimitMiddleware_WhitelistBypass tests that whitelisted IPs bypass rate limiting
func TestRateLimitMiddleware_WhitelistBypass(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     1,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:whitelist",
		Description:     "whitelist test",
	})

	whitelistedIP := "10.0.0.1"
	limiter.AddToWhitelist(whitelistedIP)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})
	middleware := limiter.RateLimitMiddleware(nextHandler)

	// Record multiple failures for whitelisted IP
	for i := 0; i < 5; i++ {
		limiter.RecordFailure(whitelistedIP)
	}

	// Request from whitelisted IP should still be allowed
	req := httptest.NewRequest("POST", "/test", nil)
	req.RemoteAddr = whitelistedIP + ":8080"
	w := httptest.NewRecorder()

	middleware.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code, "whitelisted IP should bypass rate limiting")
}

// TestEnhancedRedisRateLimiter_RedisPersistence tests that rate limits persist in Redis
func TestEnhancedRedisRateLimiter_RedisPersistence(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	testIP := "192.168.1.102"

	// Create first limiter and record failures
	limiter1 := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 30 * time.Second,
		ResetWindow:     30 * time.Second,
		KeyPrefix:       "test:persistence",
		Description:     "persistence test",
	})

	limiter1.RecordFailure(testIP)
	limiter1.RecordFailure(testIP)

	// Create second limiter (simulating server restart)
	limiter2 := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     2,
		LockoutDuration: 30 * time.Second,
		ResetWindow:     30 * time.Second,
		KeyPrefix:       "test:persistence",
		Description:     "persistence test",
	})

	// Should still be locked from first limiter's failures
	allowed, _ := limiter2.IsAllowed(testIP)
	assert.False(t, allowed, "rate limit should persist in Redis across restarts")
}

// TestEnhancedRedisRateLimiter_PartialFailuresRecovery tests recovery from partial failures
func TestEnhancedRedisRateLimiter_PartialFailuresRecovery(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     3,
		LockoutDuration: 10 * time.Second,
		ResetWindow:     10 * time.Second,
		KeyPrefix:       "test:recovery",
		Description:     "recovery test",
	})

	testIP := "192.168.1.103"

	// Record 2 failures
	limiter.RecordFailure(testIP)
	limiter.RecordFailure(testIP)

	allowed, _ := limiter.IsAllowed(testIP)
	assert.True(t, allowed, "should still be allowed after 2 failures")

	// Record success to reset counter
	limiter.RecordSuccess(testIP)

	// Should be able to fail again from the beginning
	limiter.RecordFailure(testIP)
	allowed, _ = limiter.IsAllowed(testIP)
	assert.True(t, allowed, "should be allowed and reset after success")
}

// TestEnhancedRedisRateLimiter_HighLoad simulates high load scenario
func TestEnhancedRedisRateLimiter_HighLoad(t *testing.T) {
	redisClient := createTestRedisClient(t)
	defer redisClient.Close()

	limiter := NewEnhancedRedisRateLimiter(redisClient, RateLimitConfig{
		MaxAttempts:     100,
		LockoutDuration: 1 * time.Second,
		ResetWindow:     1 * time.Second,
		KeyPrefix:       "test:load",
		Description:     "load test",
	})

	// Simulate 50 different IPs making requests
	for ipNum := 1; ipNum <= 50; ipNum++ {
		ip := "192.168." + string(rune(ipNum/256)) + "." + string(rune(ipNum%256))
		
		// Each IP makes 5 requests
		for i := 0; i < 5; i++ {
			allowed, _ := limiter.IsAllowed(ip)
			assert.True(t, allowed, "IP %s request %d should be allowed", ip, i)
			limiter.RecordFailure(ip)
		}
	}

	// All IPs should still be allowed (under 100 limit)
	for ipNum := 1; ipNum <= 50; ipNum++ {
		ip := "192.168." + string(rune(ipNum/256)) + "." + string(rune(ipNum%256))
		allowed, _ := limiter.IsAllowed(ip)
		assert.True(t, allowed, "IP %s should still be allowed", ip)
	}

	metrics := limiter.GetMetrics()
	assert.Greater(t, metrics.TotalAllowed, int64(0), "should have allowed requests in high load test")
}

// TestGetClientIP tests the client IP extraction function
func TestGetClientIP(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(*http.Request)
		expected  string
	}{
		{
			name: "X-Forwarded-For with multiple IPs",
			setup: func(r *http.Request) {
				r.Header.Set("X-Forwarded-For", "203.0.113.1, 198.51.100.1, 192.0.2.1")
			},
			expected: "203.0.113.1",
		},
		{
			name: "X-Real-IP fallback",
			setup: func(r *http.Request) {
				r.Header.Set("X-Real-IP", "203.0.113.2")
			},
			expected: "203.0.113.2",
		},
		{
			name: "RemoteAddr direct",
			setup: func(r *http.Request) {
				r.RemoteAddr = "192.168.1.1:8080"
			},
			expected: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			tt.setup(req)
			actual := GetClientIP(req)
			assert.Equal(t, tt.expected, actual)
		})
	}
}
