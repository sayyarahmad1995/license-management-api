package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"license-management-api/internal/errors"
	"license-management-api/pkg/utils"
	"github.com/redis/go-redis/v9"
)

// RedisRateLimiter uses Redis for distributed rate limiting
type RedisRateLimiter struct {
	redis             *utils.RedisClient
	maxAttempts       int
	lockoutDuration   time.Duration
	resetWindow       time.Duration
	attemptKeyPrefix  string
	lockedUntilPrefix string
}

// NewRedisRateLimiter creates a new Redis-based rate limiter
func NewRedisRateLimiter(redis *utils.RedisClient, maxAttempts int, lockoutDuration, resetWindow time.Duration) *RedisRateLimiter {
	if maxAttempts == 0 {
		maxAttempts = 5
	}
	if lockoutDuration == 0 {
		lockoutDuration = 15 * time.Minute
	}
	if resetWindow == 0 {
		resetWindow = 15 * time.Minute
	}

	return &RedisRateLimiter{
		redis:             redis,
		maxAttempts:       maxAttempts,
		lockoutDuration:   lockoutDuration,
		resetWindow:       resetWindow,
		attemptKeyPrefix:  "ratelimit:attempts:",
		lockedUntilPrefix: "ratelimit:lockeduntil:",
	}
}

// IsAllowed checks if an IP is allowed to make a request
func (rl *RedisRateLimiter) IsAllowed(ip string) bool {
	lockedUntilKey := rl.lockedUntilPrefix + ip

	// Check if IP is currently locked
	lockedUntilStr, err := rl.redis.Get(lockedUntilKey)
	if err == nil && lockedUntilStr != "" {
		lockedUntil, err := time.Parse(time.RFC3339Nano, lockedUntilStr)
		if err == nil && lockedUntil.After(time.Now()) {
			return false // Still locked
		}
		// Lockout expired, delete the key
		_ = rl.redis.Delete(lockedUntilKey)
	}

	return true
}

// RecordFailure records a failed attempt and returns remaining attempts before lockout
func (rl *RedisRateLimiter) RecordFailure(ip string) int {
	attemptsKey := rl.attemptKeyPrefix + ip
	lockedUntilKey := rl.lockedUntilPrefix + ip

	// Increment failure count
	attempts, err := rl.redis.Incr(attemptsKey)
	if err != nil && err != redis.Nil {
		// If error and not key missing, still increment
		attempts = 1
		_ = rl.redis.Set(attemptsKey, 1, rl.resetWindow)
	} else if err == redis.Nil {
		// Key doesn't exist, create it
		attempts = 1
		_ = rl.redis.Set(attemptsKey, 1, rl.resetWindow)
	} else if attempts == 1 {
		// First increment, set TTL
		_ = rl.redis.Expire(attemptsKey, rl.resetWindow)
	}

	// Check if lockout threshold reached
	if int(attempts) >= rl.maxAttempts {
		lockedUntil := time.Now().Add(rl.lockoutDuration)
		_ = rl.redis.Set(lockedUntilKey, lockedUntil.Format(time.RFC3339Nano), rl.lockoutDuration)
		// Delete attempts counter when locked to clean up
		_ = rl.redis.Delete(attemptsKey)
		return 0
	}

	return rl.maxAttempts - int(attempts)
}

// RecordSuccess clears failed attempts for an IP
func (rl *RedisRateLimiter) RecordSuccess(ip string) {
	attemptsKey := rl.attemptKeyPrefix + ip
	_ = rl.redis.Delete(attemptsKey)
}

// RateLimitMiddleware wraps rate limiting for HTTP handlers
func (rl *RedisRateLimiter) RateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ip := GetClientIP(r)

		if !rl.IsAllowed(ip) {
			utils.WriteErrorResponse(w, http.StatusTooManyRequests, errors.RateLimitError, "Too many requests. Please try again later.")
			return
		}

		next.ServeHTTP(w, r)
	})
}

// GetClientIP extracts the client IP from the request
func GetClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// X-Forwarded-For can contain multiple IPs - use the first one
		return strings.TrimSpace(strings.Split(xff, ",")[0])
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}

	// Fall back to RemoteAddr and strip port if present
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		// No port present or invalid format, return as-is
		return r.RemoteAddr
	}
	return ip
}

// ===== LEGACY IN-MEMORY RATE LIMITER (DEPRECATED) =====
// Keeping for backward compatibility, but RedisRateLimiter is preferred

// RateLimitEntry tracks failed attempts and lockout status
type RateLimitEntry struct {
	FailedAttempts int
	LockedUntil    time.Time
	LastAttempt    time.Time
}

// RateLimiter enforces rate limiting on login attempts (deprecated - use RedisRateLimiter)
type RateLimiter struct {
	mu              sync.RWMutex
	attempts        map[string]*RateLimitEntry
	maxAttempts     int
	lockoutDuration time.Duration
	resetWindow     time.Duration
	done            chan struct{}
}

// NewRateLimiter creates a new in-memory rate limiter (deprecated - use NewRedisRateLimiter)
func NewRateLimiter(maxAttempts int, lockoutDuration, resetWindow time.Duration) *RateLimiter {
	if maxAttempts == 0 {
		maxAttempts = 5
	}
	if lockoutDuration == 0 {
		lockoutDuration = 15 * time.Minute
	}
	if resetWindow == 0 {
		resetWindow = 15 * time.Minute
	}

	rl := &RateLimiter{
		attempts:        make(map[string]*RateLimitEntry),
		maxAttempts:     maxAttempts,
		lockoutDuration: lockoutDuration,
		resetWindow:     resetWindow,
		done:            make(chan struct{}),
	}

	go rl.cleanupOldEntries()
	return rl
}

func (rl *RateLimiter) cleanupOldEntries() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-rl.done:
			return
		case <-ticker.C:
			rl.mu.Lock()
			now := time.Now()
			for ip, entry := range rl.attempts {
				if now.Sub(entry.LastAttempt) > rl.resetWindow && now.After(entry.LockedUntil) {
					delete(rl.attempts, ip)
				}
			}
			rl.mu.Unlock()
		}
	}
}

// Close stops the cleanup goroutine gracefully
func (rl *RateLimiter) Close() error {
	select {
	case <-rl.done:
		// Already closed
	default:
		close(rl.done)
	}
	return nil
}

func (rl *RateLimiter) IsAllowed(ip string) bool {
	rl.mu.RLock()
	entry, exists := rl.attempts[ip]
	rl.mu.RUnlock()

	if !exists {
		return true
	}

	now := time.Now()
	if entry.LockedUntil.After(now) {
		return false
	}

	if entry.FailedAttempts >= rl.maxAttempts {
		rl.mu.Lock()
		delete(rl.attempts, ip)
		rl.mu.Unlock()
		return true
	}

	return true
}

func (rl *RateLimiter) RecordSuccess(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()
	delete(rl.attempts, ip)
}

func (rl *RateLimiter) RecordFailure(ip string) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	entry, exists := rl.attempts[ip]
	now := time.Now()

	if !exists {
		entry = &RateLimitEntry{
			FailedAttempts: 1,
			LastAttempt:    now,
		}
		rl.attempts[ip] = entry
		return
	}

	if now.Sub(entry.LastAttempt) > rl.resetWindow {
		entry.FailedAttempts = 1
	} else {
		entry.FailedAttempts++
	}

	entry.LastAttempt = now

	if entry.FailedAttempts >= rl.maxAttempts {
		entry.LockedUntil = now.Add(rl.lockoutDuration)
	}
}
