package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"license-management-api/internal/models"

	"github.com/redis/go-redis/v9"
)

type LicenseCache struct {
	client *redis.Client
	ttl    time.Duration
	ctx    context.Context
}

// NewLicenseCache creates a new license cache backed by Redis
func NewLicenseCache(redisAddr string, redisPassword string, redisDB int, ttl time.Duration) *LicenseCache {
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: redisPassword,
		DB:       redisDB,
	})

	ctx := context.Background()

	// Test connection
	if err := client.Ping(ctx).Err(); err != nil {
		// Log error but don't fail - cache is optional for functionality
		fmt.Printf("Warning: Redis connection failed: %v\n", err)
	}

	return &LicenseCache{
		client: client,
		ttl:    ttl,
		ctx:    ctx,
	}
}

// getCacheKey generates a cache key for a user's licenses
func (c *LicenseCache) getCacheKey(userID int, isAdmin bool) string {
	if isAdmin {
		return "license:cache:admin:all"
	}
	return fmt.Sprintf("license:cache:user:%d", userID)
}

// Get retrieves cached licenses for a user
func (c *LicenseCache) Get(userID int, isAdmin bool) ([]models.License, bool) {
	key := c.getCacheKey(userID, isAdmin)

	val, err := c.client.Get(c.ctx, key).Result()
	if err == redis.Nil {
		// Cache miss
		return nil, false
	}
	if err != nil {
		// Error accessing cache, treat as miss
		return nil, false
	}

	// Unmarshal JSON
	var licenses []models.License
	if err := json.Unmarshal([]byte(val), &licenses); err != nil {
		// Corrupted cache, treat as miss
		return nil, false
	}

	return licenses, true
}

// Set stores licenses in cache for a user
func (c *LicenseCache) Set(userID int, isAdmin bool, licenses []models.License) {
	key := c.getCacheKey(userID, isAdmin)

	// Marshal to JSON
	data, err := json.Marshal(licenses)
	if err != nil {
		// Silently fail if marshal fails
		return
	}

	// Set in Redis with TTL
	c.client.Set(c.ctx, key, data, c.ttl)
}

// InvalidateUser invalidates cache for a specific user
func (c *LicenseCache) InvalidateUser(userID int) {
	key := c.getCacheKey(userID, false)
	c.client.Del(c.ctx, key)
}

// InvalidateAdmin invalidates admin cache
func (c *LicenseCache) InvalidateAdmin() {
	c.client.Del(c.ctx, "license:cache:admin:all")
}

// InvalidateAll invalidates all caches
func (c *LicenseCache) InvalidateAll() {
	// Use pattern matching to delete all license cache keys
	pattern := "license:cache:*"
	iter := c.client.Scan(c.ctx, 0, pattern, 0).Iterator()
	for iter.Next(c.ctx) {
		c.client.Del(c.ctx, iter.Val())
	}
}

// InvalidateLicense invalidates cache entries affected by a license change
func (c *LicenseCache) InvalidateLicense(license *models.License) {
	// Invalidate the specific user's cache
	c.InvalidateUser(license.UserID)

	// Invalidate admin cache (since admin can see all)
	c.InvalidateAdmin()
}

// Close closes the Redis connection
func (c *LicenseCache) Close() error {
	return c.client.Close()
}

// HealthCheck checks if Redis is reachable
func (c *LicenseCache) HealthCheck(ctx context.Context) error {
	if c.client == nil {
		return fmt.Errorf("redis client not initialized")
	}
	return c.client.Ping(ctx).Err()
}
