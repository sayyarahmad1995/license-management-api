package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <num_requests>")
		fmt.Println("Example: go run main.go 5")
		os.Exit(1)
	}

	var numRequests int
	fmt.Sscanf(os.Args[1], "%d", &numRequests)

	fmt.Println("🔐 Testing Rate Limiting")
	fmt.Printf("Making %d failed login attempts...\n\n", numRequests)

	// Make failed login requests
	for i := 1; i <= numRequests; i++ {
		body := map[string]string{
			"email":    "nonexistent@example.com",
			"password": "wrongpassword",
		}

		jsonBody, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "http://localhost:8080/api/v1/auth/login", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")

		client := &http.Client{Timeout: 5 * time.Second}
		resp, err := client.Do(req)
		if err != nil {
			fmt.Printf("Request %d: ❌ Error - %v\n", i, err)
			continue
		}

		statusCode := resp.StatusCode
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		switch statusCode {
		case 429:
			fmt.Printf("Request %d: 🚫 Blocked (429 Too Many Requests)\n", i)
			fmt.Printf("   Response: %s\n", string(respBody))
		case 401:
			fmt.Printf("Request %d: ✅ Failed auth (401 Unauthorized) - Counter incremented\n", i)
		default:
			fmt.Printf("Request %d: Status %d - %s\n", i, statusCode, string(respBody))
		}

		// Small delay between requests
		if i < numRequests {
			time.Sleep(100 * time.Millisecond)
		}
	}

	// Check Redis for entries
	fmt.Println("\n" + strings.Repeat("=", 70))
	fmt.Println("🔍 Checking Redis for rate limit entries...")

	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})

	defer redisClient.Close()

	// Check connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		fmt.Printf("❌ Cannot connect to Redis: %v\n", err)
		os.Exit(1)
	}

	// Get all rate limit related keys
	fmt.Println("📊 Rate Limit Keys in Redis:")

	keys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()
	if len(keys) == 0 {
		fmt.Println("❌ No rate limit entries found!")
		fmt.Println("\nPossible reasons:")
		fmt.Println("  1. Client IP might be resolved differently")
		fmt.Println("  2. Rate limiter might not be enabled")
		fmt.Println("  3. Check logs for errors")

		// Try to get any cache or test keys
		allKeys, _ := redisClient.Keys(ctx, "*").Result()
		if len(allKeys) > 0 {
			fmt.Printf("\n   Found other keys: %v\n", allKeys)
		}
		os.Exit(0)
	}

	for _, key := range keys {
		ttl, _ := redisClient.TTL(ctx, key).Result()
		val, _ := redisClient.Get(ctx, key).Result()
		fmt.Printf("✅ %s\n", key)
		fmt.Printf("   Value: %s\n", val)
		fmt.Printf("   TTL: %v seconds\n\n", ttl.Seconds())
	}

	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("\n💡 Key Patterns:")
	fmt.Println("   - ratelimit:login attempts:X.X.X.X = Failed attempt count")
	fmt.Println("   - ratelimit:login attempts:X.X.X.X:locked = Active lockout")
}
