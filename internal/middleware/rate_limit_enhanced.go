package middleware

import (
	"fmt"
	"net/http"
	"sync"
	"time"

	"license-management-api/internal/errors"
	"license-management-api/pkg/utils"

	"github.com/redis/go-redis/v9"
)

// RateLimitConfig holds configuration for rate limiting
type RateLimitConfig struct {
	MaxAttempts           int
	LockoutDuration       time.Duration
	ResetWindow           time.Duration
	AuthenticatedMaxAttrs int // Different limit for authenticated users (0 = use MaxAttempts)
	KeyPrefix             string
	Description           string // e.g., "login attempts", "API calls"
	EnableBackoff         bool   // Enable gradual backoff on repeated violations
	BackoffMultiplier     int    // Multiply lockout by this on each repetition
	MaxBackoffMultiplier  int    // Cap the backoff multiplier at this value
}

// RateLimitMetrics tracks rate limiting statistics
type RateLimitMetrics struct {
	TotalBlocked    int64
	TotalAllowed    int64
	ActiveLockouts  int64
	LastBlockedTime time.Time
}

// EnhancedRedisRateLimiter provides advanced rate limiting with metrics and configuration
type EnhancedRedisRateLimiter struct {
	redis           *utils.RedisClient
	config          RateLimitConfig
	whitelist       map[string]bool // IP whitelist
	metrics         *RateLimitMetrics
	lockoutTimings  map[string]time.Time // Track lockout end times for Retry-After header
	violationCounts map[string]int       // Track violation count for backoff calculation
	mu              sync.RWMutex         // Protects whitelist, lockoutTimings, metrics, and violationCounts
}

// NewEnhancedRedisRateLimiter creates an enhanced rate limiter with default config
func NewEnhancedRedisRateLimiter(redis *utils.RedisClient, config RateLimitConfig) *EnhancedRedisRateLimiter {
	// Set defaults
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 5
	}
	if config.LockoutDuration == 0 {
		config.LockoutDuration = 15 * time.Minute
	}
	if config.ResetWindow == 0 {
		config.ResetWindow = 15 * time.Minute
	}
	if config.KeyPrefix == "" {
		config.KeyPrefix = "ratelimit"
	}

	return &EnhancedRedisRateLimiter{
		redis:           redis,
		config:          config,
		whitelist:       make(map[string]bool),
		metrics:         &RateLimitMetrics{},
		lockoutTimings:  make(map[string]time.Time),
		violationCounts: make(map[string]int),
	}
}

// AddToWhitelist adds an IP to the whitelist (bypass rate limiting)
func (erl *EnhancedRedisRateLimiter) AddToWhitelist(ips ...string) {
	erl.mu.Lock()
	defer erl.mu.Unlock()
	for _, ip := range ips {
		erl.whitelist[ip] = true
	}
}

// RemoveFromWhitelist removes an IP from the whitelist
func (erl *EnhancedRedisRateLimiter) RemoveFromWhitelist(ips ...string) {
	erl.mu.Lock()
	defer erl.mu.Unlock()
	for _, ip := range ips {
		delete(erl.whitelist, ip)
	}
}

// IsWhitelisted checks if an IP is in the whitelist
func (erl *EnhancedRedisRateLimiter) IsWhitelisted(ip string) bool {
	erl.mu.RLock()
	defer erl.mu.RUnlock()
	return erl.whitelist[ip]
}

// IsAllowed checks if an IP is allowed (considers whitelist and lockout status)
func (erl *EnhancedRedisRateLimiter) IsAllowed(ip string) (allowed bool, remainingTime time.Duration) {
	// Check whitelist first
	if erl.IsWhitelisted(ip) {
		return true, 0
	}

	lockedUntilKey := fmt.Sprintf("%s:locked:%s", erl.config.KeyPrefix, ip)
	lockedUntilStr, err := erl.redis.Get(lockedUntilKey)

	if err == nil && lockedUntilStr != "" {
		lockedUntil, parseErr := time.Parse(time.RFC3339Nano, lockedUntilStr)
		if parseErr == nil {
			now := time.Now()
			if lockedUntil.After(now) {
				remaining := lockedUntil.Sub(now)
				erl.mu.Lock()
				erl.metrics.TotalBlocked++
				erl.metrics.LastBlockedTime = now
				erl.lockoutTimings[ip] = lockedUntil
				erl.mu.Unlock()
				return false, remaining
			}
			// Lockout expired, delete the key
			_ = erl.redis.Delete(lockedUntilKey)
		}
	}

	erl.mu.Lock()
	erl.metrics.TotalAllowed++
	erl.mu.Unlock()
	return true, 0
}

// RecordFailure records a failed attempt and returns remaining attempts before lockout
func (erl *EnhancedRedisRateLimiter) RecordFailure(ip string) int {
	attemptsKey := fmt.Sprintf("%s:attempts:%s", erl.config.KeyPrefix, ip)
	lockedUntilKey := fmt.Sprintf("%s:locked:%s", erl.config.KeyPrefix, ip)
	violationKey := fmt.Sprintf("%s:violations:%s", erl.config.KeyPrefix, ip)

	// Increment failure count
	attempts, err := erl.redis.Incr(attemptsKey)
	if err != nil && err != redis.Nil {
		attempts = 1
		_ = erl.redis.Set(attemptsKey, 1, erl.config.ResetWindow)
	} else if err == redis.Nil {
		attempts = 1
		_ = erl.redis.Set(attemptsKey, 1, erl.config.ResetWindow)
	} else if attempts == 1 {
		_ = erl.redis.Expire(attemptsKey, erl.config.ResetWindow)
	}

	// Check if lockout threshold reached
	if int(attempts) >= erl.config.MaxAttempts {
		lockoutDuration := erl.config.LockoutDuration

		// Apply gradual backoff if enabled
		if erl.config.EnableBackoff && erl.config.BackoffMultiplier > 0 {
			violationCount, _ := erl.redis.Incr(violationKey)
			_ = erl.redis.Expire(violationKey, 24*time.Hour) // Track violations for 24 hours

			// Calculate backoff: duration * (multiplier ^ violations)
			multiplier := 1.0
			for i := 0; i < int(violationCount)-1; i++ {
				multiplier *= float64(erl.config.BackoffMultiplier)
				if erl.config.MaxBackoffMultiplier > 0 && multiplier > float64(erl.config.MaxBackoffMultiplier) {
					multiplier = float64(erl.config.MaxBackoffMultiplier)
					break
				}
			}
			lockoutDuration = time.Duration(float64(erl.config.LockoutDuration) * multiplier)
		}

		lockedUntil := time.Now().Add(lockoutDuration)
		_ = erl.redis.Set(lockedUntilKey, lockedUntil.Format(time.RFC3339Nano), lockoutDuration)
		_ = erl.redis.Delete(attemptsKey)
		erl.mu.Lock()
		erl.metrics.ActiveLockouts++
		erl.lockoutTimings[ip] = lockedUntil
		erl.mu.Unlock()
		return 0
	}

	return erl.config.MaxAttempts - int(attempts)
}

// RecordAttempt increments the attempt counter and triggers lockout if threshold reached
// Used by middleware to count every login attempt (success or failure)
func (erl *EnhancedRedisRateLimiter) RecordAttempt(ip string) {
	attemptsKey := fmt.Sprintf("%s:attempts:%s", erl.config.KeyPrefix, ip)
	lockedUntilKey := fmt.Sprintf("%s:locked:%s", erl.config.KeyPrefix, ip)

	// Increment attempt count
	attempts, err := erl.redis.Incr(attemptsKey)
	if err != nil && err != redis.Nil {
		attempts = 1
		_ = erl.redis.Set(attemptsKey, 1, erl.config.ResetWindow)
	} else if err == redis.Nil {
		attempts = 1
		_ = erl.redis.Set(attemptsKey, 1, erl.config.ResetWindow)
	} else if attempts == 1 {
		_ = erl.redis.Expire(attemptsKey, erl.config.ResetWindow)
	}

	// Check if lockout threshold reached
	if int(attempts) >= erl.config.MaxAttempts {
		lockedUntil := time.Now().Add(erl.config.LockoutDuration)
		_ = erl.redis.Set(lockedUntilKey, lockedUntil.Format(time.RFC3339Nano), erl.config.LockoutDuration)
		erl.mu.Lock()
		erl.metrics.ActiveLockouts++
		erl.lockoutTimings[ip] = lockedUntil
		erl.mu.Unlock()
	}
}

// RecordSuccess clears failed attempts for an IP
func (erl *EnhancedRedisRateLimiter) RecordSuccess(ip string) {
	attemptsKey := fmt.Sprintf("%s:attempts:%s", erl.config.KeyPrefix, ip)
	_ = erl.redis.Delete(attemptsKey)
	erl.mu.Lock()
	delete(erl.lockoutTimings, ip)
	erl.mu.Unlock()
}

// GetMetrics returns current rate limiting metrics
func (erl *EnhancedRedisRateLimiter) GetMetrics() *RateLimitMetrics {
	erl.mu.RLock()
	defer erl.mu.RUnlock()
	return erl.metrics
}

// IsAllowedForUser checks if a user is allowed with user-specific limits
// If isAuthenticated is true, uses AuthenticatedMaxAttrs limit (if configured)
func (erl *EnhancedRedisRateLimiter) IsAllowedForUser(ip string, isAuthenticated bool) (allowed bool, remainingTime time.Duration) {
	// Check whitelist first
	if erl.IsWhitelisted(ip) {
		return true, 0
	}

	// For authenticated users, use higher limit if configured
	originalMaxAttempts := erl.config.MaxAttempts
	if isAuthenticated && erl.config.AuthenticatedMaxAttrs > 0 {
		erl.config.MaxAttempts = erl.config.AuthenticatedMaxAttrs
		defer func() { erl.config.MaxAttempts = originalMaxAttempts }()
	}

	return erl.IsAllowed(ip)
}

// RecordFailureForUser records a failure with user-specific limits
// If isAuthenticated is true, uses AuthenticatedMaxAttrs limit (if configured)
func (erl *EnhancedRedisRateLimiter) RecordFailureForUser(ip string, isAuthenticated bool) int {
	// For authenticated users, use higher limit if configured
	originalMaxAttempts := erl.config.MaxAttempts
	if isAuthenticated && erl.config.AuthenticatedMaxAttrs > 0 {
		erl.config.MaxAttempts = erl.config.AuthenticatedMaxAttrs
		defer func() { erl.config.MaxAttempts = originalMaxAttempts }()
	}

	return erl.RecordFailure(ip)
}

// RateLimitMiddleware returns a Chi middleware for rate limiting with Retry-After header
func (erl *EnhancedRedisRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)

		allowed, remainingTime := erl.IsAllowed(ip)
		if !allowed {
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(remainingTime.Seconds())))
			w.Header().Set("X-RateLimit-Limit-Name", erl.config.Description)
			w.Header().Set("X-RateLimit-Reset", time.Now().Add(remainingTime).Format(time.RFC3339))
			utils.WriteErrorResponse(w, http.StatusTooManyRequests, errors.RateLimitError,
				fmt.Sprintf("Too many %s. Please try again in %d seconds.", erl.config.Description, int(remainingTime.Seconds())))
			return
		}

		// IMPORTANT: Record every attempt BEFORE handler executes
		// This ensures rate limiting applies to ALL requests (success or failure)
		// The handler will NOT call RecordFailure again to avoid double-counting
		erl.RecordAttempt(ip)

		next.ServeHTTP(w, r)
	})
}

// RateLimitByEndpoint creates a rate limiter for a specific endpoint with custom config
func RateLimitByEndpoint(redis *utils.RedisClient, maxAttempts int, lockoutDuration time.Duration, description string) *EnhancedRedisRateLimiter {
	config := RateLimitConfig{
		MaxAttempts:     maxAttempts,
		LockoutDuration: lockoutDuration,
		ResetWindow:     lockoutDuration,
		KeyPrefix:       fmt.Sprintf("ratelimit:%s", description),
		Description:     description,
	}
	return NewEnhancedRedisRateLimiter(redis, config)
}
