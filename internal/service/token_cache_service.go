package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"license-management-api/internal/config"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

// TokenCacheService handles caching of email verification and password reset tokens
type TokenCacheService struct {
	cache                 *CacheService
	emailVerificationRepo repository.IEmailVerificationRepository
	passwordResetRepo     repository.IPasswordResetRepository
	cacheTTLConfig        *config.CacheTTLConfig
}

// NewTokenCacheService creates a new token cache service
func NewTokenCacheService(
	cache *CacheService,
	emailVerificationRepo repository.IEmailVerificationRepository,
	passwordResetRepo repository.IPasswordResetRepository,
	cacheTTLConfig *config.CacheTTLConfig,
) *TokenCacheService {
	if cacheTTLConfig == nil {
		cacheTTLConfig = config.LoadCacheTTLConfig()
	}
	return &TokenCacheService{
		cache:                 cache,
		emailVerificationRepo: emailVerificationRepo,
		passwordResetRepo:     passwordResetRepo,
		cacheTTLConfig:        cacheTTLConfig,
	}
}

// ===== EMAIL VERIFICATION TOKEN CACHING =====

// CacheEmailVerificationToken stores email verification token in Redis with configurable TTL
func (tcs *TokenCacheService) CacheEmailVerificationToken(token string, verification *models.EmailVerification) error {
	// Use pattern verification_token:{userID}:{token} to organize by user and avoid caching User object
	key := fmt.Sprintf("verification_token:%d:%s", verification.UserID, token)
	ttl := tcs.cacheTTLConfig.EmailVerificationTokenTTL

	// Create a minimal copy without User field to avoid caching unnecessary nested data
	verificationData := &models.EmailVerification{
		ID:        verification.ID,
		UserID:    verification.UserID,
		Token:     verification.Token,
		Email:     verification.Email,
		ExpiresAt: verification.ExpiresAt,
		UsedAt:    verification.UsedAt,
		CreatedAt: verification.CreatedAt,
	}

	log.Printf("CACHE: Storing email verification token: %s, TTL: %v\n", token, ttl)
	return tcs.cache.SetWithTTL(key, verificationData, ttl)
}

// GetEmailVerificationToken retrieves email verification token from cache first, then DB
func (tcs *TokenCacheService) GetEmailVerificationToken(token string, userID ...int64) (*models.EmailVerification, error) {
	// Try cache first if userID is provided
	var cached *models.EmailVerification
	if len(userID) > 0 {
		key := fmt.Sprintf("verification_token:%d:%s", userID[0], token)
		if err := tcs.cache.Get(key, &cached); err == nil && cached != nil {
			log.Printf("CACHE HIT: Email verification token found in cache: %s\n", token)
			return cached, nil
		}
	}

	// Cache miss - get from database
	log.Printf("CACHE MISS: Email verification token not in cache, fetching from DB: %s\n", token)
	verification, err := tcs.emailVerificationRepo.GetByToken(token)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	_ = tcs.CacheEmailVerificationToken(token, verification)
	return verification, nil
}

// InvalidateEmailVerificationToken removes token from cache after verification
func (tcs *TokenCacheService) InvalidateEmailVerificationToken(token string, userID int64) error {
	key := fmt.Sprintf("verification_token:%d:%s", userID, token)
	log.Printf("CACHE: Invalidating email verification token: %s\n", token)
	return tcs.cache.Delete(key)
}

// InvalidateUserEmailVerificationTokens removes all verification tokens for a user
func (tcs *TokenCacheService) InvalidateUserEmailVerificationTokens(userID int64) error {
	// Use userID in pattern for efficient user-specific cleanup
	pattern := fmt.Sprintf("verification_token:%d:*", userID)
	log.Printf("CACHE: Clearing email verification tokens for user: %d\n", userID)
	return tcs.cache.InvalidatePattern(pattern)
}

// ===== PASSWORD RESET TOKEN CACHING =====

// CachePasswordResetToken stores password reset token in Redis with configurable TTL
func (tcs *TokenCacheService) CachePasswordResetToken(token string, reset *models.PasswordReset) error {
	// Use pattern password_reset_token:{userID}:{token} to organize by user and avoid caching User object
	key := fmt.Sprintf("password_reset_token:%d:%s", reset.UserID, token)
	ttl := tcs.cacheTTLConfig.PasswordResetTokenTTL

	// Create a minimal copy without User field to avoid caching unnecessary nested data
	resetData := &models.PasswordReset{
		ID:        reset.ID,
		UserID:    reset.UserID,
		Token:     reset.Token,
		Email:     reset.Email,
		ExpiresAt: reset.ExpiresAt,
		UsedAt:    reset.UsedAt,
		CreatedAt: reset.CreatedAt,
	}

	log.Printf("CACHE: Storing password reset token: %s, TTL: %v\n", token, ttl)
	return tcs.cache.SetWithTTL(key, resetData, ttl)
}

// GetPasswordResetToken retrieves password reset token from cache first, then DB
func (tcs *TokenCacheService) GetPasswordResetToken(token string, userID ...int64) (*models.PasswordReset, error) {
	// Try cache first if userID is provided
	var cached *models.PasswordReset
	if len(userID) > 0 {
		key := fmt.Sprintf("password_reset_token:%d:%s", userID[0], token)
		if err := tcs.cache.Get(key, &cached); err == nil && cached != nil {
			log.Printf("CACHE HIT: Password reset token found in cache: %s\n", token)
			return cached, nil
		}
	}

	// Cache miss - get from database
	log.Printf("CACHE MISS: Password reset token not in cache, fetching from DB: %s\n", token)
	reset, err := tcs.passwordResetRepo.GetByToken(token)
	if err != nil {
		return nil, err
	}

	// Store in cache for future requests
	_ = tcs.CachePasswordResetToken(token, reset)
	return reset, nil
}

// InvalidatePasswordResetToken removes token from cache after password reset
func (tcs *TokenCacheService) InvalidatePasswordResetToken(token string, userID int64) error {
	key := fmt.Sprintf("password_reset_token:%d:%s", userID, token)
	log.Printf("CACHE: Invalidating password reset token: %s\n", token)
	return tcs.cache.Delete(key)
}

// InvalidateUserPasswordResetTokens removes all reset tokens for a user
func (tcs *TokenCacheService) InvalidateUserPasswordResetTokens(userID int64) error {
	// Use userID in pattern for efficient user-specific cleanup
	pattern := fmt.Sprintf("password_reset_token:%d:*", userID)
	log.Printf("CACHE: Clearing password reset tokens for user: %d\n", userID)
	return tcs.cache.InvalidatePattern(pattern)
}

// ===== LICENSE ACTIVATION CACHING =====

// CacheLicenseActivation stores active license activation with configurable TTL
func (tcs *TokenCacheService) CacheLicenseActivation(activation *models.LicenseActivation) error {
	key := fmt.Sprintf("license_activation:%d", activation.ID)
	ttl := tcs.cacheTTLConfig.LicenseActivationTTL

	log.Printf("CACHE: Storing license activation: %d, TTL: %v\n", activation.ID, ttl)
	return tcs.cache.SetWithTTL(key, activation, ttl)
}

// GetLicenseActivation retrieves license activation from cache
func (tcs *TokenCacheService) GetLicenseActivation(activationID int64) (*models.LicenseActivation, error) {
	key := fmt.Sprintf("license_activation:%d", activationID)

	var cached *models.LicenseActivation
	if err := tcs.cache.Get(key, &cached); err == nil && cached != nil {
		log.Printf("CACHE HIT: License activation found in cache: %d\n", activationID)
		return cached, nil
	}

	return nil, fmt.Errorf("activation not in cache")
}

// InvalidateLicenseActivation removes activation from cache
func (tcs *TokenCacheService) InvalidateLicenseActivation(activationID int64) error {
	key := fmt.Sprintf("license_activation:%d", activationID)
	log.Printf("CACHE: Invalidating license activation: %d\n", activationID)
	return tcs.cache.Delete(key)
}

// InvalidateLicenseActivationsByKey removes all activations for a license key
func (tcs *TokenCacheService) InvalidateLicenseActivationsByKey(licenseKey string) error {
	pattern := "license_activation:*"
	log.Printf("CACHE: Clearing license activations for key: %s\n", licenseKey)
	return tcs.cache.InvalidatePattern(pattern)
}

// ===== DASHBOARD STATS CACHING =====

// CacheDashboardStats stores dashboard statistics with configurable TTL
func (tcs *TokenCacheService) CacheDashboardStats(stats interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = tcs.cacheTTLConfig.DashboardStatsTTL
	}

	key := "dashboard_stats"
	log.Printf("CACHE: Storing dashboard stats, TTL: %v\n", ttl)
	return tcs.cache.SetWithTTL(key, stats, ttl)
}

// GetDashboardStats retrieves cached dashboard stats
func (tcs *TokenCacheService) GetDashboardStats(dest interface{}) error {
	key := "dashboard_stats"

	var cached string
	err := tcs.cache.Get(key, &cached)
	if err != nil {
		return err
	}

	if cached == "" {
		return fmt.Errorf("dashboard stats not in cache")
	}

	log.Printf("CACHE HIT: Dashboard stats found in cache\n")
	return json.Unmarshal([]byte(cached), dest)
}

// InvalidateDashboardStats removes cached dashboard stats
func (tcs *TokenCacheService) InvalidateDashboardStats() error {
	key := "dashboard_stats"
	log.Printf("CACHE: Invalidating dashboard stats\n")
	return tcs.cache.Delete(key)
}

// ===== USER LICENSE LIST CACHING =====

// CacheUserLicenses stores user's license list with 10-minute TTL
func (tcs *TokenCacheService) CacheUserLicenses(userID int64, licenses interface{}) error {
	key := fmt.Sprintf("user_licenses:%d", userID)
	ttl := 10 * time.Minute // User license list changes less frequently

	log.Printf("CACHE: Storing licenses for user: %d, TTL: %v\n", userID, ttl)
	return tcs.cache.SetWithTTL(key, licenses, ttl)
}

// GetCachedUserLicenses retrieves cached user licenses
func (tcs *TokenCacheService) GetCachedUserLicenses(userID int64, dest interface{}) error {
	key := fmt.Sprintf("user_licenses:%d", userID)

	var cached string
	err := tcs.cache.Get(key, &cached)
	if err != nil {
		return err
	}

	if cached == "" {
		return fmt.Errorf("licenses not in cache")
	}

	log.Printf("CACHE HIT: User licenses found in cache: %d\n", userID)
	return json.Unmarshal([]byte(cached), dest)
}

// InvalidateUserLicenses removes cached licenses for a user
func (tcs *TokenCacheService) InvalidateUserLicenses(userID int64) error {
	key := fmt.Sprintf("user_licenses:%d", userID)
	log.Printf("CACHE: Invalidating licenses for user: %d\n", userID)
	return tcs.cache.Delete(key)
}

// ===== REVOKED TOKENS CACHING =====

// CacheRevokedToken stores a revoked JWT token with TTL until expiration
func (tcs *TokenCacheService) CacheRevokedToken(token string, expiresAt time.Time) error {
	key := fmt.Sprintf("revoked_token:%s", token)
	ttl := time.Until(expiresAt)

	if ttl <= 0 {
		ttl = 1 * time.Hour // Default 1 hour if already expired
	}

	log.Printf("CACHE: Storing revoked token, TTL: %v\n", ttl)
	return tcs.cache.SetWithTTL(key, true, ttl)
}

// IsTokenRevoked checks if a token is revoked
func (tcs *TokenCacheService) IsTokenRevoked(token string) (bool, error) {
	key := fmt.Sprintf("revoked_token:%s", token)

	var revoked bool
	if err := tcs.cache.Get(key, &revoked); err == nil && revoked {
		log.Printf("CACHE HIT: Token is revoked: %s\n", token)
		return true, nil
	}

	log.Printf("CACHE MISS: Token not in revoked list: %s\n", token)
	return false, nil
}

// PurgeExpiredCaches clears all expired token caches (called periodically)
func (tcs *TokenCacheService) PurgeExpiredCaches() error {
	// Redis TTL automatically handles expiration, but we can add cleanup hooks if needed
	log.Printf("CACHE: Purging expired cached entries\n")
	return nil
}

// Cache key constants for token features (others defined in cache_service.go)
const (
	CacheKeyVerificationToken  = "verification_token:%s"
	CacheKeyPasswordResetToken = "password_reset_token:%s"
	CacheKeyLicenseActivation  = "license_activation:%d"
	CacheKeyRevokedToken       = "revoked_token:%s"
	CacheKeyLicenseValidation  = "license_validation:%s"
	CacheKeyUserActivity       = "user_activity:%d"
)
