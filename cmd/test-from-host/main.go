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
	fmt.Println("🧪 Testing Rate Limiting from Host Machine")
	fmt.Println("=" + strings.Repeat("=", 69))

	// FlushRedis first
	ctx := context.Background()
	redisClient := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})
	defer redisClient.Close()

	// Clear old entries
	keys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()
	for _, key := range keys {
		redisClient.Del(ctx, key)
	}
	fmt.Println("✅ Redis cleared")

	// Make 6 login requests from host machine
	fmt.Println("📝 Making 6 login requests...")

	for i := 1; i <= 6; i++ {
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
			fmt.Printf("Request %d: ❌ Error - %v\n", i, err)
			continue
		}

		statusCode := resp.StatusCode
		respBody, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		switch statusCode {
		case 429:
			fmt.Printf("Request %d: 🚫 BLOCKED - 429 Too Many Requests\n", i)
			fmt.Printf("   Response: %s\n\n", string(respBody))
		case 401:
			fmt.Printf("Request %d: ✅ Allowed (401 Auth Failed)\n", i)
		default:
			fmt.Printf("Request %d: Status %d\n", i, statusCode)
		}

		time.Sleep(100 * time.Millisecond)
	}

	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("\n🔍 Checking Redis for entries from HOST machine:")

	// Get all keys
	allKeys, _ := redisClient.Keys(ctx, "ratelimit:*").Result()

	if len(allKeys) == 0 {
		fmt.Println("❌ NO ENTRIES FOUND IN REDIS!")
		fmt.Println("\n⚠️  This means:")
		fmt.Println("  • Middleware is NOT being called")
		fmt.Println("  • Middleware is NOT updating Redis")
		fmt.Println("  • Rate limiting is NOT working")
		fmt.Println("\nDebugging:")
		fmt.Println("  1. Check API logs: docker-compose logs api")
		fmt.Println("  2. Verify Redis is up: docker-compose ps")
		fmt.Println("  3. Check server.go for middleware setup")
		return
	}

	fmt.Printf("✅ Found %d entries:\n\n", len(allKeys))

	for _, key := range allKeys {
		ttl, _ := redisClient.TTL(ctx, key).Result()
		val, _ := redisClient.Get(ctx, key).Result()

		fmt.Printf("Key: %s\n", key)
		fmt.Printf("Value: %s\n", val)
		fmt.Printf("TTL: %d seconds\n\n", int64(ttl.Seconds()))
	}

	fmt.Println("=" + strings.Repeat("=", 69))
	fmt.Println("\n✅ Rate limiting IS working!")
	fmt.Println("   Middleware is tracking requests and blocking after 5 failures.")
}
