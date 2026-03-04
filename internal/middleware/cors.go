package middleware

import (
	"net/http"
	"strings"
)

// CORSConfig holds CORS configuration
type CORSConfig struct {
	AllowedOrigins []string
}

// CORSMiddleware creates a CORS middleware that validates incoming requests against allowed origins
func CORSMiddleware(config *CORSConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			// Check if origin is allowed
			if isOriginAllowed(origin, config.AllowedOrigins) {
				// Set CORS headers for allowed origins
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, PATCH, OPTIONS")
				w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Accept, X-Requested-With")
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Max-Age", "3600") // 1 hour
			}

			// Handle preflight requests
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isOriginAllowed checks if the given origin is in the allowed list
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	// If no origins are configured, deny all cross-origin requests
	if len(allowedOrigins) == 0 {
		return false
	}

	// Check for wildcard (allow all) - use with caution in production
	for _, allowed := range allowedOrigins {
		if allowed == "*" {
			return true
		}
		if allowed == origin {
			return true
		}
	}

	return false
}

// ParseCORSOrigins parses comma-separated origins from a string
// Example: "http://localhost:3000,http://localhost:5173" -> []string{"http://localhost:3000", "http://localhost:5173"}
func ParseCORSOrigins(originsString string) []string {
	if originsString == "" {
		return []string{}
	}

	origins := strings.Split(originsString, ",")
	trimmedOrigins := make([]string, 0, len(origins))

	for _, origin := range origins {
		trimmed := strings.TrimSpace(origin)
		if trimmed != "" {
			trimmedOrigins = append(trimmedOrigins, trimmed)
		}
	}

	return trimmedOrigins
}
