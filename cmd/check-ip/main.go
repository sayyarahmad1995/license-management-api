package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

func main() {
	fmt.Println("🔎 What IP does your machine appear as from Docker?")

	// Make a request to get our IP as seen by the API
	body := map[string]string{
		"email":    "test@example.com",
		"password": "test",
	}

	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "http://localhost:8080/api/v1/auth/login", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Printf("Error making request: %v\n", err)
		return
	}
	defer resp.Body.Close()

	_, _ = io.ReadAll(resp.Body)

	fmt.Printf("Response Status: %d\n", resp.StatusCode)
	fmt.Printf("Response Headers:\n")
	for key, values := range resp.Header {
		fmt.Printf("  %s: %v\n", key, values)
	}

	fmt.Println("\n📊 To find your IP in Redis:")
	fmt.Println("   1. Run: go run ./cmd/redis-check/main.go")
	fmt.Println("   2. Look for key like: ratelimit:login attempts:XXXX")
	fmt.Println("   3. The XXXX is your IP as seen by the API")

	fmt.Println("💡 Common IPs from Docker:")
	fmt.Println("   • 127.0.0.1 - localhost")
	fmt.Println("   • 172.17.0.1 - Docker gateway (host machine)")
	fmt.Println("   • 172.18.0.1 - Docker network IP (inside container)")
}
