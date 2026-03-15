package service

import (
	"fmt"
	"log"
	"sort"
	"time"

	"license-management-api/internal/config"
	"license-management-api/internal/models"
)

const (
	// MaxConcurrentSessionsPerUser limits how many active sessions a user can have
	MaxConcurrentSessionsPerUser = 5
)

// SessionCacheService handles caching of user sessions
type SessionCacheService struct {
	cache             *CacheService
	sessionRepository interface{} // Will be injected when repository is created
	cacheTTLConfig    *config.CacheTTLConfig
}

// NewSessionCacheService creates a new session cache service with optional TTL config
func NewSessionCacheService(cache *CacheService, cacheTTLConfig *config.CacheTTLConfig) *SessionCacheService {
	if cacheTTLConfig == nil {
		cacheTTLConfig = config.LoadCacheTTLConfig()
	}
	return &SessionCacheService{
		cache:          cache,
		cacheTTLConfig: cacheTTLConfig,
	}
}

// ===== USER SESSION CACHING =====

// CacheUserSession stores user session in Redis with configurable TTL
// Uses pattern: session:{userID}:{ipAddress}:{token} for granular control
// Enforces MaxConcurrentSessionsPerUser limit by evicting oldest session if exceeded
func (scs *SessionCacheService) CacheUserSession(session *models.UserSession, token string) error {
	// Use pattern session:{userID}:{ipAddress}:{token} to organize by user/IP/device
	// This allows invalidating sessions by specific IP or device without affecting others
	key := fmt.Sprintf("session:%d:%s:%s", session.UserID, session.IPAddress, token)

	// Check concurrent session limit
	activeSessions, err := scs.GetUserActiveSessions(int64(session.UserID))
	if err == nil && len(activeSessions) >= MaxConcurrentSessionsPerUser {
		// Sort by LastActivityAt to find oldest session
		sort.Slice(activeSessions, func(i, j int) bool {
			return activeSessions[i].LastActivityAt.Before(activeSessions[j].LastActivityAt)
		})

		// Remove oldest session to make room for new one
		oldestSession := activeSessions[0]
		oldestKey := fmt.Sprintf("session:%d:%s:%s", oldestSession.UserID, oldestSession.IPAddress, oldestSession.SessionID)
		if err := scs.cache.Delete(oldestKey); err == nil {
			log.Printf("CACHE: Evicted oldest session for user %d (session limit reached: %d)\n", session.UserID, MaxConcurrentSessionsPerUser)
		}
	}

	// Calculate TTL based on session expiration or use configured TTL
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = 24 * time.Hour // Fallback default
	}

	log.Printf("CACHE: Storing user session: userID=%d, IP=%s, TTL: %v\n", session.UserID, session.IPAddress, ttl)
	return scs.cache.SetWithTTL(key, session, ttl)
}

// GetUserSession retrieves user session from cache first, then DB fallback
// Requires userID, ipAddress, and token for precise lookup
func (scs *SessionCacheService) GetUserSession(userID int64, ipAddress string, token string) (*models.UserSession, error) {
	key := fmt.Sprintf("session:%d:%s:%s", userID, ipAddress, token)

	var cached *models.UserSession
	if err := scs.cache.Get(key, &cached); err == nil && cached != nil {
		log.Printf("CACHE HIT: User session found in cache: userID=%d, IP=%s\n", userID, ipAddress)
		return cached, nil
	}

	// Cache miss - would typically fetch from database
	log.Printf("CACHE MISS: User session not in cache: userID=%d, IP=%s\n", userID, ipAddress)
	return nil, fmt.Errorf("session not in cache")
}

// InvalidateUserSession removes a specific session from cache
// Invalidates session by userID, ipAddress, and token
func (scs *SessionCacheService) InvalidateUserSession(userID int64, ipAddress string, token string) error {
	key := fmt.Sprintf("session:%d:%s:%s", userID, ipAddress, token)
	log.Printf("CACHE: Invalidating user session: userID=%d, IP=%s\n", userID, ipAddress)
	return scs.cache.Delete(key)
}

// InvalidateUserSessionsByIP removes all sessions for a user from a specific IP address
// Useful when suspicious activity detected from an IP
func (scs *SessionCacheService) InvalidateUserSessionsByIP(userID int64, ipAddress string) error {
	// Use pattern session:{userID}:{ipAddress}:* to invalidate all sessions from this IP
	pattern := fmt.Sprintf("session:%d:%s:*", userID, ipAddress)
	log.Printf("CACHE: Clearing all sessions for user %d from IP %s\n", userID, ipAddress)
	return scs.cache.InvalidatePattern(pattern)
}

// InvalidateUserSessions removes all sessions for a user (logout all devices/IPs)
// Use with caution - logs out user from everywhere
func (scs *SessionCacheService) InvalidateUserSessions(userID int64) error {
	// Use pattern session:{userID}:*:* to invalidate all sessions regardless of IP
	pattern := fmt.Sprintf("session:%d:*:*", userID)
	log.Printf("CACHE: Clearing all sessions for user: %d (logout all devices)\n", userID)
	return scs.cache.InvalidatePattern(pattern)
}

// InvalidateUserSessionsExcept removes all sessions for a user except specified token
// Useful for "logout from other devices" feature
func (scs *SessionCacheService) InvalidateUserSessionsExcept(userID int64, exceptToken string) error {
	pattern := fmt.Sprintf("session:%d:*", userID)
	log.Printf("CACHE: Clear all sessions for user %d except token %s\n", userID, exceptToken)

	// Get all keys matching the pattern
	keys, err := scs.cache.GetKeysByPattern(pattern)
	if err != nil {
		return fmt.Errorf("failed to scan for sessions: %w", err)
	}

	// Delete all sessions except the one with exceptToken
	deletedCount := 0
	for _, key := range keys {
		// Parse the key to extract token (format: session:{userID}:{ipAddress}:{token})
		// If key doesn't contain the exceptToken, delete it
		if !contains(key, exceptToken) {
			if err := scs.cache.Delete(key); err == nil {
				deletedCount++
			}
		}
	}

	log.Printf("CACHE: Deleted %d sessions for user %d (kept current session)\n", deletedCount, userID)
	return nil
}

// contains checks if the key contains the token substring
func contains(key, token string) bool {
	return len(token) > 0 && len(key) > 0 && key[len(key)-len(token):] == token
}

// UpdateSessionActivity updates the last activity timestamp in cache
func (scs *SessionCacheService) UpdateSessionActivity(userID int64, ipAddress string, token string) error {
	// Retrieve current session
	key := fmt.Sprintf("session:%d:%s:%s", userID, ipAddress, token)
	var session *models.UserSession
	if err := scs.cache.Get(key, &session); err == nil && session != nil {
		// Update activity
		session.UpdateActivity()
		// Re-cache with same TTL
		ttl := time.Until(session.ExpiresAt)
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		log.Printf("CACHE: Updated activity for session: userID=%d, IP=%s\n", userID, ipAddress)
		return scs.cache.SetWithTTL(key, session, ttl)
	}

	return fmt.Errorf("session not found in cache")
}

// RevokeUserSession marks a session as revoked in cache
func (scs *SessionCacheService) RevokeUserSession(userID int64, ipAddress string, token string) error {
	key := fmt.Sprintf("session:%d:%s:%s", userID, ipAddress, token)
	var session *models.UserSession
	if err := scs.cache.Get(key, &session); err == nil && session != nil {
		session.Revoke()
		ttl := time.Until(session.ExpiresAt)
		if ttl <= 0 {
			ttl = 24 * time.Hour
		}
		log.Printf("CACHE: Revoked user session: userID=%d, IP=%s\n", userID, ipAddress)
		return scs.cache.SetWithTTL(key, session, ttl)
	}

	return fmt.Errorf("session not found in cache")
}

// GetUserActiveSessions retrieves all active sessions for a user
// Note: This requires scanning, which is expensive - use sparingly
func (scs *SessionCacheService) GetUserActiveSessions(userID int64) ([]*models.UserSession, error) {
	pattern := fmt.Sprintf("session:%d:*", userID)
	log.Printf("CACHE: Getting active sessions for user: %d (pattern: %s)\n", userID, pattern)

	// Get all keys matching the pattern
	keys, err := scs.cache.GetKeysByPattern(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to scan for sessions: %w", err)
	}

	// Retrieve each session
	var sessions []*models.UserSession
	for _, key := range keys {
		var session *models.UserSession
		if err := scs.cache.Get(key, &session); err == nil && session != nil {
			// Only include active (non-revoked, non-expired) sessions
			if session.IsActive() {
				sessions = append(sessions, session)
			}
		}
	}

	log.Printf("CACHE: Found %d active sessions for user %d\n", len(sessions), userID)
	return sessions, nil
}

// ===== TOKEN CACHING (Access & Refresh) =====

// CacheAccessToken stores access token with configurable TTL
// Access tokens are short-lived and renewed frequently
// Uses pattern: access_token:{userID}:{ipAddress} for per-IP token management
func (scs *SessionCacheService) CacheAccessToken(userID int64, ipAddress string, token string) error {
	key := fmt.Sprintf("access_token:%d:%s", userID, ipAddress)
	ttl := scs.cacheTTLConfig.AccessTokenTTL

	log.Printf("CACHE: Storing access token for user: %d, IP: %s, TTL: %v\n", userID, ipAddress, ttl)
	return scs.cache.SetWithTTL(key, token, ttl)
}

// GetAccessToken retrieves cached access token
// Returns empty string if token has expired (cache miss)
func (scs *SessionCacheService) GetAccessToken(userID int64, ipAddress string) (string, error) {
	key := fmt.Sprintf("access_token:%d:%s", userID, ipAddress)
	var token string
	if err := scs.cache.Get(key, &token); err == nil && token != "" {
		log.Printf("CACHE HIT: Access token found for user: %d, IP: %s\n", userID, ipAddress)
		return token, nil
	}

	log.Printf("CACHE MISS: Access token expired/not in cache for user: %d, IP: %s\n", userID, ipAddress)
	return "", fmt.Errorf("access token not found or expired")
}

// CacheRefreshToken stores refresh token with configurable TTL
// Refresh tokens are longer-lived and can be extended on activity
// Uses pattern: refresh_token:{userID}:{ipAddress} for per-IP token management
func (scs *SessionCacheService) CacheRefreshToken(userID int64, ipAddress string, token string, ttl time.Duration) error {
	if ttl == 0 {
		ttl = scs.cacheTTLConfig.RefreshTokenTTL
	}
	key := fmt.Sprintf("refresh_token:%d:%s", userID, ipAddress)

	log.Printf("CACHE: Storing refresh token for user: %d, IP: %s, TTL: %v\n", userID, ipAddress, ttl)
	return scs.cache.SetWithTTL(key, token, ttl)
}

// GetRefreshToken retrieves cached refresh token
// Returns empty string if token has expired
func (scs *SessionCacheService) GetRefreshToken(userID int64, ipAddress string) (string, error) {
	key := fmt.Sprintf("refresh_token:%d:%s", userID, ipAddress)
	var token string
	if err := scs.cache.Get(key, &token); err == nil && token != "" {
		log.Printf("CACHE HIT: Refresh token found for user: %d, IP: %s\n", userID, ipAddress)
		return token, nil
	}

	log.Printf("CACHE MISS: Refresh token expired/not in cache for user: %d, IP: %s\n", userID, ipAddress)
	return "", fmt.Errorf("refresh token not found or expired")
}

// UpdateUserLastSeen updates user's last activity timestamp and extends refresh token
// Called on every API request to keep session alive without interrupting user
// Uses short TTL to avoid excessive cache updates
// Uses pattern: last_seen:{userID}:{ipAddress} for per-IP activity tracking
func (scs *SessionCacheService) UpdateUserLastSeen(userID int64, ipAddress string, refreshToken string) error {
	key := fmt.Sprintf("last_seen:%d:%s", userID, ipAddress)
	lastSeen := time.Now().UTC()
	ttl := scs.cacheTTLConfig.LastSeenActivityTTL

	// Cache last_seen timestamp with short TTL for activity tracking
	if err := scs.cache.SetWithTTL(key, lastSeen, ttl); err != nil {
		log.Printf("CACHE: Failed to update last_seen for user: %d, IP: %s\n", userID, ipAddress)
		return err
	}

	// Silently extend refresh token if it exists
	// This allows the session to continue without user noticing
	_ = scs.ExtendRefreshToken(userID, ipAddress, refreshToken, scs.cacheTTLConfig.RefreshTokenTTL)

	log.Printf("CACHE: Updated last_seen for user: %d, IP: %s and extended refresh token\n", userID, ipAddress)
	return nil
}

// GetUserLastSeen retrieves the last activity timestamp from cache
// Returns zero time if not in cache (user was inactive)
func (scs *SessionCacheService) GetUserLastSeen(userID int64, ipAddress string) (time.Time, error) {
	key := fmt.Sprintf("last_seen:%d:%s", userID, ipAddress)
	var lastSeen time.Time
	if err := scs.cache.Get(key, &lastSeen); err == nil && !lastSeen.IsZero() {
		log.Printf("CACHE HIT: Last seen for user: %d, IP: %s at %v\n", userID, ipAddress, lastSeen)
		return lastSeen, nil
	}

	log.Printf("CACHE MISS: Last seen not in cache for user: %d, IP: %s\n", userID, ipAddress)
	return time.Time{}, fmt.Errorf("last_seen not found in cache")
}

// ExtendRefreshToken extends the TTL of an existing refresh token
// Called when user activity is detected to keep session alive
// Uses pattern: refresh_token:{userID}:{ipAddress} for per-IP management
func (scs *SessionCacheService) ExtendRefreshToken(userID int64, ipAddress string, token string, newTTL time.Duration) error {
	if newTTL == 0 {
		newTTL = 7 * 24 * time.Hour
	}
	key := fmt.Sprintf("refresh_token:%d:%s", userID, ipAddress)

	// Re-cache the token with extended TTL
	log.Printf("CACHE: Extended refresh token for user: %d, IP: %s, new TTL: %v\n", userID, ipAddress, newTTL)
	return scs.cache.SetWithTTL(key, token, newTTL)
}

// InvalidateTokens removes both access and refresh tokens for a user from a specific IP
// Called on logout from specific device
func (scs *SessionCacheService) InvalidateTokens(userID int64, ipAddress string) error {
	accessTokenKey := fmt.Sprintf("access_token:%d:%s", userID, ipAddress)
	refreshTokenKey := fmt.Sprintf("refresh_token:%d:%s", userID, ipAddress)
	lastSeenKey := fmt.Sprintf("last_seen:%d:%s", userID, ipAddress)

	_ = scs.cache.Delete(accessTokenKey)
	_ = scs.cache.Delete(refreshTokenKey)
	_ = scs.cache.Delete(lastSeenKey)

	log.Printf("CACHE: Invalidated all tokens for user: %d, IP: %s\n", userID, ipAddress)
	return nil
}

// InvalidateTokensAllIPs removes all tokens for a user across all IPs
// Called on password change or security-sensitive actions
func (scs *SessionCacheService) InvalidateTokensAllIPs(userID int64) error {
	// Invalidate all tokens for all IPs
	accessPattern := fmt.Sprintf("access_token:%d:*", userID)
	refreshPattern := fmt.Sprintf("refresh_token:%d:*", userID)
	lastSeenPattern := fmt.Sprintf("last_seen:%d:*", userID)

	_ = scs.cache.InvalidatePattern(accessPattern)
	_ = scs.cache.InvalidatePattern(refreshPattern)
	_ = scs.cache.InvalidatePattern(lastSeenPattern)

	log.Printf("CACHE: Invalidated all tokens for user: %d across all IPs\n", userID)
	return nil
}

// Cache key constants for tokens
const (
	CacheKeyAccessToken  = "access_token:%d:%s"  // {userID}:{ipAddress}
	CacheKeyRefreshToken = "refresh_token:%d:%s" // {userID}:{ipAddress}
	CacheKeyLastSeen     = "last_seen:%d:%s"     // {userID}:{ipAddress}
)
