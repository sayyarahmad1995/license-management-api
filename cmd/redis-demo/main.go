package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	// Connect to Redis
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})

	err := client.Ping(ctx).Err()
	if err != nil {
		fmt.Printf("❌ Failed to connect to Redis: %v\n", err)
		os.Exit(1)
	}
	defer client.Close()

	fmt.Println("✅ Connected to Redis!")

	// Create sample rate limit entries to demonstrate
	fmt.Println("📝 Creating sample rate limit entries...")

	// Sample 1: User who has 2 failed login attempts
	prefixFailed := "ratelimit:login:192.168.1.100"
	client.Set(ctx, prefixFailed+":attempts", 2, 15*time.Minute)
	fmt.Printf("Created: %s = 2\n", prefixFailed+":attempts")
	fmt.Printf("TTL: 15 minutes (automatic lock on 3rd attempt)\n\n")

	// Sample 2: Locked user
	lockedKey := "ratelimit:login:192.168.1.101:locked"
	lockedUntil := time.Now().Add(10 * time.Minute)
	client.Set(ctx, lockedKey, lockedUntil.Format(time.RFC3339Nano), 10*time.Minute)
	fmt.Printf("Created: %s\n", lockedKey)
	fmt.Printf("Value: %s (locked until this time)\n", lockedUntil.Format(time.RFC3339))
	fmt.Printf("TTL: 10 minutes\n\n")

	// Sample 3: Cache entry
	cacheKey := "cache:user:123:profile"
	client.Set(ctx, cacheKey, `{"id":123,"username":"testuser"}`, 1*time.Hour)
	fmt.Printf("Created: %s\n", cacheKey)
	fmt.Printf("Value: {cache data}\n")
	fmt.Printf("TTL: 1 hour\n\n")

	// Now display all keys
	fmt.Println("================================================================================")
	fmt.Println("📊 Current Redis Database Contents:")

	keysCmd := client.Keys(ctx, "*")
	keys, err := keysCmd.Result()
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}

	dbSize := client.DBSize(ctx)
	size, _ := dbSize.Result()
	fmt.Printf("Total keys: %d\n\n", size)

	for i, key := range keys {
		ttl := client.TTL(ctx, key)
		ttlVal, _ := ttl.Result()

		val := client.Get(ctx, key)
		value, _ := val.Result()

		fmt.Printf("%d. Key: %s\n", i+1, key)
		fmt.Printf("   Type: %s\n", client.Type(ctx, key).Val())

		// Truncate long values for display
		if len(value) > 60 {
			fmt.Printf("   Value: %s... (%d chars)\n", value[:57], len(value))
		} else {
			fmt.Printf("   Value: %s\n", value)
		}

		if ttlVal > 0 {
			fmt.Printf("   TTL: %v (expires in %d seconds)\n", ttlVal, int64(ttlVal.Seconds()))
		} else {
			fmt.Printf("   TTL: No expiration\n")
		}
		fmt.Println()
	}

	fmt.Println("================================================================================")
	fmt.Println("\n💡 Key Patterns:")
	fmt.Println("   - ratelimit:login:* = Login attempt tracking")
	fmt.Println("   - ratelimit:login:*:locked = Active lockouts")
	fmt.Println("   - cache:* = Cached data")
	fmt.Println("   - test:* = Test data (used during tests)")
}
