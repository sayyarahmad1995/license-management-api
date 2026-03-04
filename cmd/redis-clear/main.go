package main

import (
	"context"
	"fmt"
	"github.com/redis/go-redis/v9"
)

func main() {
	ctx := context.Background()

	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Username: "default",
		Password: "Admin@123",
		DB:       0,
	})

	defer client.Close()

	// Delete all rate limit keys
	keys, _ := client.Keys(ctx, "ratelimit:*").Result()
	if len(keys) > 0 {
		for _, key := range keys {
			client.Del(ctx, key)
			fmt.Printf("Deleted: %s\n", key)
		}
	}

	fmt.Println("\n✅ Redis cleaned")
}
