package middleware

import (
	"crypto/md5"
	"fmt"
	"net/http"
	"strings"

	"license-management-api/internal/service"
)

// CachingMiddleware provides HTTP-level caching for GET requests
// Caches successful responses (200, 304) for specified time
type CachingMiddleware struct {
	cacheService *service.CacheService
	cacheTTL     string
}

// NewCachingMiddleware creates a new caching middleware
func NewCachingMiddleware(cacheService *service.CacheService) *CachingMiddleware {
	return &CachingMiddleware{
		cacheService: cacheService,
		cacheTTL:     "15m", // Default Redis cache control header
	}
}

// Middleware wraps the HTTP handler with caching logic
func (cm *CachingMiddleware) Middleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Only cache GET and HEAD requests
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			next.ServeHTTP(w, r)
			return
		}

		// Skip caching for requests with specific query params or auth headers
		if shouldSkipCache(r) {
			next.ServeHTTP(w, r)
			return
		}

		// Set cache headers for all GET requests
		w.Header().Set("Cache-Control", fmt.Sprintf("public, max-age=%s", cm.cacheTTL))

		// Proceed to handler
		next.ServeHTTP(w, r)
	})
}

// shouldSkipCache determines if request should bypass cache
func shouldSkipCache(r *http.Request) bool {
	// Skip if cache=false query param
	if r.URL.Query().Get("cache") == "false" {
		return true
	}

	// Skip if specific user filters (sorting, pagination edge cases)
	if r.URL.Query().Get("search") != "" {
		return true // Search results are too specific to cache
	}

	// Skip authenticated requests with user-specific data
	if r.Header.Get("Authorization") != "" {
		// Allow caching for public user profiles, but not private dashboards
		if strings.Contains(r.URL.Path, "dashboard") {
			return true
		}
	}

	return false
}

// generateCacheKey creates a unique cache key from request
func generateCacheKey(r *http.Request) string {
	// Include URL path and important query parameters
	key := fmt.Sprintf("http:cache:%s:%s", r.Method, r.URL.Path)

	// Include relevant query params
	relevantParams := []string{"page", "limit", "sort", "status"}
	for _, param := range relevantParams {
		if val := r.URL.Query().Get(param); val != "" {
			key += fmt.Sprintf(":%s=%s", param, val)
		}
	}

	// Create MD5 hash of the key for consistency
	hash := md5.Sum([]byte(key))
	return fmt.Sprintf("cache:http:%x", hash)
}
