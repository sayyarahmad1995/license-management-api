package middleware

import (
	"context"
	"net/http"
	"strings"

	"license-management-api/internal/models"
	"license-management-api/internal/service"
)

const PermissionContextKey = "permissions"

// RequirePermission middleware checks if user has required permission
func RequirePermission(permSvc service.PermissionService, requiredPerm models.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDContextKey).(int)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !permSvc.HasPermission(userID, requiredPerm) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAnyPermission middleware checks if user has any of the given permissions
func RequireAnyPermission(permSvc service.PermissionService, permissions ...models.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDContextKey).(int)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !permSvc.HasAnyPermission(userID, permissions...) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequireAllPermissions middleware checks if user has all given permissions
func RequireAllPermissions(permSvc service.PermissionService, permissions ...models.Permission) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDContextKey).(int)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				return
			}

			if !permSvc.HasAllPermissions(userID, permissions...) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// PermissionsMiddleware attaches user permissions to context
func PermissionsMiddleware(permSvc service.PermissionService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			userID, ok := r.Context().Value(UserIDContextKey).(int)
			if !ok {
				next.ServeHTTP(w, r)
				return
			}

			// Get user permissions
			perms, err := permSvc.GetUserPermissions(userID)
			if err != nil {
				// Continue without permissions
				next.ServeHTTP(w, r)
				return
			}

			// Add permissions to context
			ctx := context.WithValue(r.Context(), PermissionContextKey, perms)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetPermissionsFromContext retrieves permissions from request context
func GetPermissionsFromContext(r *http.Request) []models.Permission {
	if perms, ok := r.Context().Value(PermissionContextKey).([]models.Permission); ok {
		return perms
	}
	return []models.Permission{}
}

// HasPermissionInContext checks if a permission exists in context
func HasPermissionInContext(r *http.Request, permission models.Permission) bool {
	perms := GetPermissionsFromContext(r)
	for _, p := range perms {
		if p == permission {
			return true
		}
	}
	return false
}

// AdminOnlyMiddleware ensures user is admin
func AdminOnlyMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(UserRoleContextKey).(string)
			if !ok || role != string(models.UserRoleAdmin) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// ManagerOrAdminMiddleware ensures user is manager or admin
func ManagerOrAdminMiddleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			role, ok := r.Context().Value(UserRoleContextKey).(string)
			if !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			if role != string(models.UserRoleManager) && role != string(models.UserRoleAdmin) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractPermissionsFromString converts comma-separated permission string to models.Permission slice
func ExtractPermissionsFromString(permsStr string) []models.Permission {
	var permissions []models.Permission
	parts := strings.Split(permsStr, ",")
	for _, part := range parts {
		trimmed := strings.TrimSpace(part)
		if trimmed != "" {
			permissions = append(permissions, models.Permission(trimmed))
		}
	}
	return permissions
}
