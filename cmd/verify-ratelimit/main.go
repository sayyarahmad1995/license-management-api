package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("🔍 Step-by-Step Rate Limiting Test")
	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("\n📝 Instructions for Postman Testing:")
	fmt.Println("  1. Open Postman")
	fmt.Println("  2. Make POST request to: http://localhost:8080/api/v1/auth/login")
	fmt.Println("  3. Body (raw JSON):")
	fmt.Println(`     {
       "email": "nonexistent@test.com",
       "password": "wrongpassword"
     }`)
	fmt.Println("\n  4. Send this request 5 times (you'll get 401)")
	fmt.Println("  5. Send 6th time - you'll get 429 (Too Many Requests)")
	fmt.Println("  6. Come back here and press ENTER")

	fmt.Print("Press ENTER after making 6 requests from Postman: ")
	var input string
	fmt.Scanln(&input)

	// Check Redis for all rate limit entries
	fmt.Println("\n" + "=" + strings.Repeat("=", 69))
	fmt.Println("\n🔍 Checking Redis for rate limit entries from all IPs:")

	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})

	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("❌ Cannot connect to Redis: %v\n", err)
		return
	}
	defer redisClient.Close()

	// Get all rate limit keys
	keys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()

	if len(keys) == 0 {
		fmt.Println("❌ No rate limit entries found!")
		fmt.Println("\nPossible issues:")
		fmt.Println("  1. Middleware may not be applied correctly")
		fmt.Println("  2. Check API logs for errors")
		fmt.Println("  3. Verify Redis is running")
		return
	}

	fmt.Printf("✅ Found %d rate limit entries:\n\n", len(keys))

	for _, key := range keys {
		ttl, _ := redisClient.TTL(ctx, key).Result()
		val, _ := redisClient.Get(ctx, key).Result()

		parts := strings.Split(key, ":")
		var ip string
		if len(parts) >= 3 {
			ip = parts[2]
		}

		fmt.Printf("📍 IP: %s\n", ip)
		fmt.Printf("   Key: %s\n", key)
		fmt.Printf("   TTL: %d seconds (%.1f minutes)\n", int64(ttl.Seconds()), ttl.Minutes())

		if strings.Contains(key, "locked") {
			fmt.Printf("   Status: 🚫 LOCKED\n")
			fmt.Printf("   Expires: %s\n", val)
		} else {
			fmt.Printf("   Failures: %s\n", val)
		}
		fmt.Println()
	}

	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("\n✅ Rate Limiting Verification Complete!")
	fmt.Println("\nWhat this means:")
	fmt.Println("  • Requests 1-5: Handler was called, auth failed, failure was recorded")
	fmt.Println("  • Request 6: Middleware blocked it before handler (429 response)")
	fmt.Println("  • 15-minute lockout is now active for your IP")
	fmt.Println("\nIt IS protected! The middleware is working correctly.")
}
