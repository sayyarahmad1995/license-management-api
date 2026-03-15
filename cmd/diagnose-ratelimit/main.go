package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

func main() {
	fmt.Println("🔬 COMPREHENSIVE RATE LIMITING DIAGNOSTIC")
	fmt.Println(strings.Repeat("=", 70))

	// Step 1: Clear Redis
	fmt.Println("\n[STEP 1] Clearing Redis...")
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})
	defer redisClient.Close()

	keys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()
	for _, key := range keys {
		redisClient.Del(ctx, key)
	}
	fmt.Printf("✅ Cleared %d existing rate limit entries\n", len(keys))

	// Step 2: Make first request
	fmt.Println("\n[STEP 2] Sending 1st login request...")
	statusCode1, _ := makeLoginRequest()
	fmt.Printf("Status: %d\n", statusCode1)

	// Step 3: Check Redis after first request
	fmt.Println("\n[STEP 3] Checking Redis after 1st request...")
	printRedisState(redisClient, ctx)

	// Step 4: Make more requests until blocked
	fmt.Println("\n[STEP 4] Sending requests #2-6...")
	for i := 2; i <= 6; i++ {
		statusCode, body := makeLoginRequest()
		fmt.Printf("Request %d: Status %d", i, statusCode)
		if statusCode == 429 {
			fmt.Printf(" - BLOCKED!\n")
			fmt.Printf("Response: %s\n", body)
			break
		} else {
			fmt.Println(" - Allowed")
		}
		time.Sleep(50 * time.Millisecond)
	}

	// Step 5: Final Redis check
	fmt.Println("\n[STEP 5] Final Redis state...")
	printRedisState(redisClient, ctx)

	// Step 6: Analysis
	fmt.Println("\n[ANALYSIS]")
	allKeys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()
	if len(allKeys) == 0 {
		fmt.Println("❌ NO REDIS ENTRIES - Rate limiting NOT working at middleware level")
		fmt.Println("\nPossible causes:")
		fmt.Println("  1. Middleware is not being applied correctly")
		fmt.Println("  2. Redis connection is failing")
		fmt.Println("  3. Check API logs for errors")
	} else {
		fmt.Println("✅ Redis entries found - Middleware IS working")
	}
}

func makeLoginRequest() (int, string) {
	body := map[string]string{
		"email":    "test@example.com",
		"password": "wrongpass",
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8080/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err.Error()
	}
	defer resp.Body.Close()

	respBody, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(respBody)
}

func printRedisState(redisClient *redis.Client, ctx context.Context) {
	keys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()

	if len(keys) == 0 {
		fmt.Println("📊 No rate limit entries in Redis")
		return
	}

	fmt.Printf("📊 Found %d entries:\n", len(keys))
	for _, key := range keys {
		val, _ := redisClient.Get(ctx, key).Result()
		ttl, _ := redisClient.TTL(ctx, key).Result()
		fmt.Printf("  • %s = %s (TTL: %d sec)\n", key, val, int64(ttl.Seconds()))
	}
}
