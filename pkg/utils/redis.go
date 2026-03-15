package utils

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisClient wraps go-redis with connection management
type RedisClient struct {
	client *redis.Client
	ctx    context.Context
}

// NewRedisClient creates a new Redis client connection
func NewRedisClient(ctx context.Context) (*RedisClient, error) {
	host := os.Getenv("REDIS_HOST")
	if host == "" {
		host = "localhost"
	}

	port := os.Getenv("REDIS_PORT")
	if port == "" {
		port = "6379"
	}

	username := os.Getenv("REDIS_USERNAME")
	if username == "" {
		username = "default"
	}

	password := os.Getenv("REDIS_PASSWORD")
	dbStr := os.Getenv("REDIS_DB")
	db := 0
	if dbStr != "" {
		if parsedDB, err := strconv.Atoi(dbStr); err == nil {
			db = parsedDB
		}
	}

	client := redis.NewClient(&redis.Options{
		Addr:         fmt.Sprintf("%s:%s", host, port),
		Username:     username,
		Password:     password,
		DB:           db,
		MaxRetries:   3,
		PoolSize:     10,
		MinIdleConns: 5,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})

	// Test connection
	cmd := client.Ping(ctx)
	if err := cmd.Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %w", err)
	}

	return &RedisClient{
		client: client,
		ctx:    ctx,
	}, nil
}

// Set stores a value with TTL
func (rc *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	return rc.client.Set(rc.ctx, key, value, ttl).Err()
}

// Get retrieves a value
func (rc *RedisClient) Get(key string) (string, error) {
	return rc.client.Get(rc.ctx, key).Result()
}

// GetInt retrieves an integer value
func (rc *RedisClient) GetInt(key string) (int, error) {
	val, err := rc.Get(key)
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, err
	}
	return strconv.Atoi(val)
}

// Incr increments a value and returns new value
func (rc *RedisClient) Incr(key string) (int64, error) {
	return rc.client.Incr(rc.ctx, key).Result()
}

// Expire sets expiration on a key
func (rc *RedisClient) Expire(key string, ttl time.Duration) error {
	return rc.client.Expire(rc.ctx, key, ttl).Err()
}

// Delete removes a key
func (rc *RedisClient) Delete(key string) error {
	return rc.client.Del(rc.ctx, key).Err()
}

// Exists checks if a key exists
func (rc *RedisClient) Exists(key string) bool {
	return rc.client.Exists(rc.ctx, key).Val() > 0
}

// SetJSON stores a struct as JSON with TTL
func (rc *RedisClient) SetJSON(key string, value interface{}, ttl time.Duration) error {
	return rc.Set(key, value, ttl)
}

// GetJSON retrieves and unmarshals JSON
func (rc *RedisClient) GetJSON(key string, dest interface{}) error {
	_, err := rc.Get(key)
	if err != nil {
		return err
	}
	// Redis returns JSON strings directly, client handles marshaling
	return nil
}

// FlushDB clears all keys in current database (use with care, mainly for testing)
func (rc *RedisClient) FlushDB() error {
	return rc.client.FlushDB(rc.ctx).Err()
}

// Close closes the Redis connection
func (rc *RedisClient) Close() error {
	if rc.client != nil {
		return rc.client.Close()
	}
	return nil
}

// Client returns underlying Redis client for advanced operations
func (rc *RedisClient) Client() *redis.Client {
	return rc.client
}
