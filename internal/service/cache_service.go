package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"license-management-api/pkg/utils"
)

// CacheService provides caching functionality for frequently accessed data
type CacheService struct {
	redis *utils.RedisClient
	ttl   time.Duration
}

// NewCacheService creates a new cache service
func NewCacheService(redis *utils.RedisClient, ttl time.Duration) *CacheService {
	if ttl == 0 {
		ttl = 15 * time.Minute // Default TTL
	}
	return &CacheService{
		redis: redis,
		ttl:   ttl,
	}
}

// Get retrieves a cached value
func (cs *CacheService) Get(key string, dest interface{}) error {
	val, err := cs.redis.Get(key)
	if err != nil {
		return err
	}

	if val == "" {
		return fmt.Errorf("cache miss")
	}

	return json.Unmarshal([]byte(val), dest)
}

// Set stores a value in cache
func (cs *CacheService) Set(key string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cs.redis.Set(key, string(data), cs.ttl)
}

// SetWithTTL stores a value with custom TTL
func (cs *CacheService) SetWithTTL(key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return err
	}
	return cs.redis.Set(key, string(data), ttl)
}

// Delete removes a cached value
func (cs *CacheService) Delete(key string) error {
	return cs.redis.Delete(key)
}

// Exists checks if a key exists in cache
func (cs *CacheService) Exists(key string) bool {
	return cs.redis.Exists(key)
}

// InvalidatePattern deletes all keys matching a pattern
func (cs *CacheService) InvalidatePattern(pattern string) error {
	ctx := context.Background()
	client := cs.redis.Client()

	// Use SCAN to find keys matching pattern
	iter := client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		_ = cs.redis.Delete(iter.Val())
	}
	return iter.Err()
}

// GetKeysByPattern retrieves all keys matching a pattern
func (cs *CacheService) GetKeysByPattern(pattern string) ([]string, error) {
	ctx := context.Background()
	client := cs.redis.Client()

	var keys []string
	iter := client.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		keys = append(keys, iter.Val())
	}
	return keys, iter.Err()
}

// Cache key prefixes for different entities
const (
	CacheKeyUser           = "cache:user:%d"
	CacheKeyUserEmail      = "cache:user_email:%s"
	CacheKeyLicense        = "cache:license:%d"
	CacheKeyLicenseKey     = "cache:license_key:%s"
	CacheKeyUserLicenses   = "cache:user_licenses:%d"
	CacheKeyAuditLogs      = "cache:audit_logs:%d:%d:%d"
	CacheKeyDashboardStats = "cache:dashboard_stats"
	CacheKeyUserDashboard  = "cache:user_dashboard:%d"
)

// BuildKey builds a cache key with arguments
func BuildKey(pattern string, args ...interface{}) string {
	return fmt.Sprintf(pattern, args...)
}
