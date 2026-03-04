package middleware

import (
	"context"
	"net/http"
	"strings"

	"license-management-api/internal/errors"
	"license-management-api/internal/service"
)

const (
	UserIDContextKey = "userId"
	EmailContextKey  = "email"
	UserRoleContextKey = "userRole"
)

// AuthMiddleware validates JWT tokens and sets user info in context
// It checks for tokens in cookies first, then falls back to Authorization header
func AuthMiddleware(ts service.TokenService) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var token string

			// First, try to get token from cookie
			accessTokenCookie, err := r.Cookie("accessToken")
			if err == nil && accessTokenCookie.Value != "" {
				token = accessTokenCookie.Value
			} else {
				// Fall back to Authorization header (Bearer token)
				authHeader := r.Header.Get("Authorization")
				if authHeader == "" {
					writeError(w, errors.NewUnauthorizedError("Missing authorization credentials"))
					return
				}

				parts := strings.Split(authHeader, " ")
				if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
					writeError(w, errors.NewUnauthorizedError("Invalid authorization header format"))
					return
				}

				token = parts[1]
			}

			claims, err := ts.ValidateAccessToken(token)
			if err != nil {
				writeError(w, errors.NewUnauthorizedError("Invalid or expired token"))
				return
			}

			// Add user info to context
			ctx := context.WithValue(r.Context(), UserIDContextKey, claims.UserID)
			ctx = context.WithValue(ctx, EmailContextKey, claims.Email)
			ctx = context.WithValue(ctx, UserRoleContextKey, claims.Role)

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext extracts user ID from context
func GetUserIDFromContext(ctx context.Context) (int, bool) {
	userID, ok := ctx.Value(UserIDContextKey).(int)
	return userID, ok
}

// GetEmailFromContext extracts email from context
func GetEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value(EmailContextKey).(string)
	return email, ok
}
func ErrorHandlerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				writeError(w, errors.NewInternalError("Internal server error"))
			}
		}()
		next.ServeHTTP(w, r)
	})
}

func writeError(w http.ResponseWriter, apiErr *errors.ApiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	// Simple JSON write (in production use json.Marshal)
	w.Write([]byte(`{"type":"` + string(apiErr.Type) + `","message":"` + apiErr.Message + `","status":` + string(rune(apiErr.Status)) + `}`))
}
