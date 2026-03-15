package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	CSRFTokenHeader   = "X-CSRF-Token"
	CSRFTokenCookie   = "csrf_token"
	CSRFHeaderPrefix  = "X-"
	CSRFFormFieldName = "csrf_token"
)

// CSRFToken represents a CSRF token with metadata
type CSRFToken struct {
	Token     string
	CreatedAt time.Time
	ExpiresAt time.Time
	Used      bool
}

// SimpleCSRFStore is a simple in-memory CSRF token store
// For production, use a Redis or database-backed store
type SimpleCSRFStore struct {
	tokens map[string]*CSRFToken
	mu     sync.RWMutex
	ttl    time.Duration
}

// NewSimpleCSRFStore creates a new CSRF token store
func NewSimpleCSRFStore(ttl time.Duration) *SimpleCSRFStore {
	store := &SimpleCSRFStore{
		tokens: make(map[string]*CSRFToken),
		ttl:    ttl,
	}

	// Clean up expired tokens periodically
	go store.cleanupExpiredTokens()

	return store
}

// GenerateToken generates a new CSRF token
func (s *SimpleCSRFStore) GenerateToken() (string, error) {
	token := make([]byte, 32)
	if _, err := rand.Read(token); err != nil {
		return "", err
	}

	tokenStr := hex.EncodeToString(token)

	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	s.tokens[tokenStr] = &CSRFToken{
		Token:     tokenStr,
		CreatedAt: now,
		ExpiresAt: now.Add(s.ttl),
		Used:      false,
	}

	return tokenStr, nil
}

// ValidateToken validates a CSRF token
func (s *SimpleCSRFStore) ValidateToken(token string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	t, exists := s.tokens[token]
	if !exists {
		return false
	}

	// Check expiration
	if time.Now().After(t.ExpiresAt) {
		return false
	}

	// Check if token was already used
	if t.Used {
		return false
	}

	return true
}

// InvalidateToken marks a token as used
func (s *SimpleCSRFStore) InvalidateToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	t, exists := s.tokens[token]
	if !exists {
		return ErrInvalidCSRFToken
	}

	t.Used = true
	return nil
}

var (
	ErrInvalidCSRFToken = errors.New("invalid CSRF token")
)

// cleanupExpiredTokens removes expired tokens periodically
func (s *SimpleCSRFStore) cleanupExpiredTokens() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for k, v := range s.tokens {
			if now.After(v.ExpiresAt) || v.Used {
				delete(s.tokens, k)
			}
		}
		s.mu.Unlock()
	}
}

// CSRFConfig holds CSRF protection configuration
type CSRFConfig struct {
	Store           *SimpleCSRFStore
	Enabled         bool
	TokenLength     int
	ExpirationTime  time.Duration
	SkipURLPatterns []string // URLs to skip CSRF check
	SafeMethods     []string // HTTP methods that are safe (GET, HEAD, OPTIONS)
}

// DefaultCSRFConfig returns a secure-by-default CSRF configuration
func DefaultCSRFConfig() *CSRFConfig {
	return &CSRFConfig{
		Store:           NewSimpleCSRFStore(1 * time.Hour),
		Enabled:         true,
		TokenLength:     32,
		ExpirationTime:  1 * time.Hour,
		SkipURLPatterns: []string{"/health", "/metrics"},
		SafeMethods:     []string{http.MethodGet, http.MethodHead, http.MethodOptions},
	}
}

// CSRFMiddleware provides CSRF protection
func CSRFMiddleware(config *CSRFConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !config.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Skip CSRF check for safe methods
			if isSafeMethod(r.Method, config.SafeMethods) {
				// For GET, HEAD, OPTIONS: Generate token if not present and send in response
				generateCSRFToken(w, config.Store, r)
				next.ServeHTTP(w, r)
				return
			}

			// Skip CSRF check for specific URL patterns
			if shouldSkipCSRF(r.URL.Path, config.SkipURLPatterns) {
				next.ServeHTTP(w, r)
				return
			}

			// For state-changing requests (POST, PUT, DELETE, PATCH): Validate CSRF token
			if !validateCSRFToken(w, r, config.Store) {
				return // Response already written by validateCSRFToken
			}

			next.ServeHTTP(w, r)
		})
	}
}

// generateCSRFToken generates a new CSRF token and sets it in a cookie
func generateCSRFToken(w http.ResponseWriter, store *SimpleCSRFStore, r *http.Request) {
	// Check if token already exists in cookie
	if _, err := r.Cookie(CSRFTokenCookie); err == nil {
		return // Token already present
	}

	// Generate new token
	token, err := store.GenerateToken()
	if err != nil {
		return // Silently fail, token generation is best-effort
	}

	// Set token in cookie (HttpOnly=false to allow JS access, but Secure=true)
	http.SetCookie(w, &http.Cookie{
		Name:     CSRFTokenCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: false, // Allow JS to read for XHR requests
		Secure:   true,  // Only send over HTTPS
		SameSite: http.SameSiteLaxMode,
		MaxAge:   3600, // 1 hour
	})

	// Also set in response header for convenience
	w.Header().Set(CSRFTokenHeader, token)
}

// validateCSRFToken validates the CSRF token from the request
func validateCSRFToken(w http.ResponseWriter, r *http.Request, store *SimpleCSRFStore) bool {
	var token string

	// Try to get token from header first (XHR requests)
	if headerToken := r.Header.Get(CSRFTokenHeader); headerToken != "" {
		token = headerToken
	}

	// Fall back to form data
	if token == "" {
		if err := r.ParseForm(); err == nil {
			if formToken := r.FormValue(CSRFFormFieldName); formToken != "" {
				token = formToken
			}
		}
	}

	// Fall back to cookie (POST from form)
	if token == "" {
		if cookie, err := r.Cookie(CSRFTokenCookie); err == nil {
			token = cookie.Value
		}
	}

	// Token not found
	if token == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "BAD_REQUEST_ERROR",
				"message": "Missing CSRF token",
			},
		})
		return false
	}

	// Validate token
	if !store.ValidateToken(token) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "BAD_REQUEST_ERROR",
				"message": "Invalid or expired CSRF token",
			},
		})
		return false
	}

	// Invalidate token for one-time use
	if err := store.InvalidateToken(token); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error": map[string]interface{}{
				"code":    "BAD_REQUEST_ERROR",
				"message": "Failed to validate CSRF token",
			},
		})
		return false
	}

	return true
}

// isSafeMethod checks if an HTTP method is safe (idempotent)
func isSafeMethod(method string, safeMethods []string) bool {
	for _, safe := range safeMethods {
		if method == safe {
			return true
		}
	}
	return false
}

// shouldSkipCSRF checks if a URL should skip CSRF protection
func shouldSkipCSRF(path string, skipPatterns []string) bool {
	for _, pattern := range skipPatterns {
		if strings.HasPrefix(path, pattern) {
			return true
		}
	}
	return false
}

// GetCSRFTokenHandlerFunc returns an HTTP handler that provides CSRF tokens
// This can be called by frontend to get a fresh token
func GetCSRFTokenHandlerFunc(store *SimpleCSRFStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token, err := store.GenerateToken()
		if err != nil {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"error": map[string]interface{}{
					"code":    "INTERNAL_ERROR",
					"message": "Failed to generate CSRF token",
				},
			})
			return
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"token": token,
		})
	}
}
