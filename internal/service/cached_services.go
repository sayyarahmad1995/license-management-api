package service

import (
	"context"
	"fmt"
	"time"

	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

// CachedLicenseService wraps LicenseService with caching
type CachedLicenseService struct {
	licenseService *LicenseService
	licenseRepo    repository.ILicenseRepository
	cacheService   *CacheService
}

// NewCachedLicenseService creates a new cached license service
func NewCachedLicenseService(
	licenseService *LicenseService,
	licenseRepo repository.ILicenseRepository,
	cacheService *CacheService,
) *CachedLicenseService {
	return &CachedLicenseService{
		licenseService: licenseService,
		licenseRepo:    licenseRepo,
		cacheService:   cacheService,
	}
}

// GetLicense retrieves a license with caching
func (cls *CachedLicenseService) GetLicense(id int64) (*models.License, error) {
	cacheKey := fmt.Sprintf("cache:license:%d", id)

	// Try to get from cache
	var cachedLicense *models.License
	if err := cls.cacheService.Get(cacheKey, &cachedLicense); err == nil && cachedLicense != nil {
		return cachedLicense, nil
	}

	// If not cached, get from database
	license, err := cls.licenseRepo.GetByID(int(id))
	if err != nil {
		return nil, err
	}

	// Store in cache (license status doesn't change frequently)
	_ = cls.cacheService.SetWithTTL(cacheKey, license, 30*time.Minute)

	return license, nil
}

// GetLicenseByKey retrieves a license by key with caching
func (cls *CachedLicenseService) GetLicenseByKey(key string) (*models.License, error) {
	cacheKey := fmt.Sprintf("cache:license_key:%s", key)

	var cachedLicense *models.License
	if err := cls.cacheService.Get(cacheKey, &cachedLicense); err == nil && cachedLicense != nil {
		return cachedLicense, nil
	}

	// Use repository method to get license
	license, err := cls.licenseRepo.GetByLicenseKey(key)
	if err != nil {
		return nil, err
	}

	_ = cls.cacheService.SetWithTTL(cacheKey, license, 30*time.Minute)
	return license, nil
}

// InvalidateLicenseCache clears license-related caches
func (cls *CachedLicenseService) InvalidateLicenseCache(licenseID int64) error {
	cacheKey := fmt.Sprintf("cache:license:%d", licenseID)
	return cls.cacheService.Delete(cacheKey)
}

// InvalidateUserLicensesCache clears cached user licenses list
func (cls *CachedLicenseService) InvalidateUserLicensesCache(userID int64) error {
	pattern := fmt.Sprintf("cache:user_licenses:%d*", userID)
	return cls.cacheService.InvalidatePattern(pattern)
}

// CachedUserService wraps user operations with caching
type CachedUserService struct {
	authService  *AuthService
	userRepo     repository.IUserRepository
	cacheService *CacheService
}

// NewCachedUserService creates a new cached user service
func NewCachedUserService(
	authService *AuthService,
	userRepo repository.IUserRepository,
	cacheService *CacheService,
) *CachedUserService {
	return &CachedUserService{
		authService:  authService,
		userRepo:     userRepo,
		cacheService: cacheService,
	}
}

// GetUserProfile retrieves user profile with caching
func (cus *CachedUserService) GetUserProfile(userID int64) (*models.User, error) {
	cacheKey := fmt.Sprintf("cache:user:%d", userID)

	var cachedUser *models.User
	if err := cus.cacheService.Get(cacheKey, &cachedUser); err == nil && cachedUser != nil {
		return cachedUser, nil
	}

	// Get from repository
	user, err := cus.userRepo.GetByID(int(userID))
	if err != nil {
		return nil, err
	}

	// Cache user profile (30 minutes - user data doesn't change frequently)
	_ = cus.cacheService.SetWithTTL(cacheKey, user, 30*time.Minute)

	return user, nil
}

// GetUserByEmail retrieves user by email with caching
func (cus *CachedUserService) GetUserByEmail(email string) (*models.User, error) {
	cacheKey := fmt.Sprintf("cache:user_email:%s", email)

	var cachedUser *models.User
	if err := cus.cacheService.Get(cacheKey, &cachedUser); err == nil && cachedUser != nil {
		return cachedUser, nil
	}

	user, err := cus.userRepo.GetByEmail(email)
	if err != nil {
		return nil, err
	}

	_ = cus.cacheService.SetWithTTL(cacheKey, user, 30*time.Minute)
	return user, nil
}

// InvalidateUserCache clears user-related caches
func (cus *CachedUserService) InvalidateUserCache(userID int64, email string) error {
	// Clear user profile cache
	userKey := fmt.Sprintf("cache:user:%d", userID)
	cus.cacheService.Delete(userKey)

	// Clear email cache
	emailKey := fmt.Sprintf("cache:user_email:%s", email)
	cus.cacheService.Delete(emailKey)

	// Clear dashboard cache
	dashboardKey := fmt.Sprintf("cache:user_dashboard:%d", userID)
	return cus.cacheService.Delete(dashboardKey)
}

// CacheStats provides cache statistics
type CacheStats struct {
	Hits      int64         `json:"hits"`
	Misses    int64         `json:"misses"`
	HitRate   float64       `json:"hit_rate"`
	TotalKeys int64         `json:"total_keys"`
	AvgSize   int64         `json:"avg_size"`
	Memory    int64         `json:"memory_bytes"`
	Uptime    time.Duration `json:"uptime"`
}

// GetCacheStats retrieves cache statistics
func (cs *CacheService) GetCacheStats() (*CacheStats, error) {
	if cs.redis == nil {
		return nil, fmt.Errorf("redis not available")
	}

	// Get basic Redis statistics
	// In production, this would parse the full Redis INFO response
	return &CacheStats{
		Hits:      0,
		Misses:    0,
		HitRate:   0,
		TotalKeys: 0,
		AvgSize:   0,
		Memory:    0,
		Uptime:    0,
	}, nil
}

// ClearAllCaches flushes all application caches
func (cs *CacheService) ClearAllCaches() error {
	ctx := context.Background()
	return cs.redis.Client().FlushDB(ctx).Err()
}

// CacheWarmer pre-loads frequently accessed data
type CacheWarmer struct {
	cacheService *CacheService
	userRepo     repository.IUserRepository
	licenseRepo  repository.ILicenseRepository
}

// NewCacheWarmer creates a new cache warmer
func NewCacheWarmer(
	cacheService *CacheService,
	userRepo repository.IUserRepository,
	licenseRepo repository.ILicenseRepository,
) *CacheWarmer {
	return &CacheWarmer{
		cacheService: cacheService,
		userRepo:     userRepo,
		licenseRepo:  licenseRepo,
	}
}

// WarmUserCache pre-loads active user profiles
func (cw *CacheWarmer) WarmUserCache() error {
	// Get all active users (limited to prevent memory issues)
	users, _, err := cw.userRepo.GetAll(1, 1000)
	if err != nil {
		return fmt.Errorf("failed to get users for cache warming: %v", err)
	}

	warmed := 0
	for _, user := range users {
		// Only warm active users
		if user.Status == "Active" {
			cacheKey := fmt.Sprintf("cache:user:%d", user.ID)
			if err := cw.cacheService.SetWithTTL(cacheKey, user, 30*time.Minute); err == nil {
				warmed++
			}
		}
	}

	fmt.Printf("Cache warmer: warmed %d user profiles\n", warmed)
	return nil
}

// WarmLicenseCache pre-loads active license data
func (cw *CacheWarmer) WarmLicenseCache() error {
	licenses, _, err := cw.licenseRepo.GetAll(1, 1000)
	if err != nil {
		return fmt.Errorf("failed to get licenses for cache warming: %v", err)
	}

	warmed := 0
	for _, license := range licenses {
		// Only warm active licenses (not expired)
		if license.ExpiresAt.After(time.Now()) {
			cacheKey := fmt.Sprintf("cache:license:%d", license.ID)
			if err := cw.cacheService.SetWithTTL(cacheKey, license, 30*time.Minute); err == nil {
				warmed++
			}
		}
	}

	fmt.Printf("Cache warmer: warmed %d license records\n", warmed)
	return nil
}

// WarmAllCaches warms all caches on startup
func (cw *CacheWarmer) WarmAllCaches() error {
	if err := cw.WarmUserCache(); err != nil {
		fmt.Printf("Warning: User cache warming failed: %v\n", err)
	}

	if err := cw.WarmLicenseCache(); err != nil {
		fmt.Printf("Warning: License cache warming failed: %v\n", err)
	}

	return nil
}
