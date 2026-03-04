package main

import (
	"context"
	"fmt"
	"os"

	"github.com/redis/go-redis/v9"
)

func main() {
	// Set environment variables for Redis
	os.Setenv("REDIS_HOST", "localhost")
	os.Setenv("REDIS_PORT", "6379")
	os.Setenv("REDIS_USERNAME", "default")
	os.Setenv("REDIS_PASSWORD", "Admin@123")

	ctx := context.Background()

	// Connect to Redis directly
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})

	// Test connection
	err := client.Ping(ctx).Err()
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Redis!")

	// Get database size
	dbSize := client.DBSize(ctx)
	size, err := dbSize.Result()
	if err != nil {
		fmt.Printf("Error getting database size: %v\n", err)
	} else {
		fmt.Printf("📊 Total keys in database: %d\n\n", size)
	}

	// Get all keys
	keysCmd := client.Keys(ctx, "*")
	keys, err := keysCmd.Result()
	if err != nil {
		fmt.Printf("Error scanning keys: %v\n", err)
		os.Exit(1)
	}

	if len(keys) == 0 {
		fmt.Println("❌ No keys found in Redis database")
		fmt.Println("\nPossible reasons:")
		fmt.Println("  1. Tests haven't been run yet")
		fmt.Println("  2. Data has been flushed")
		fmt.Println("  3. TTL has expired (old rate limit entries)")
		os.Exit(0)
	}

	fmt.Println("🔑 Keys in Redis:")
	for i, key := range keys {
		ttl := client.TTL(ctx, key)
		ttlVal, _ := ttl.Result()

		val := client.Get(ctx, key)
		value, _ := val.Result()

		fmt.Printf("%d. Key: %s\n", i+1, key)
		fmt.Printf("   Value: %s\n", value)
		fmt.Printf("   TTL: %v\n\n", ttlVal)
	}
}
