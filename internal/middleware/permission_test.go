package middleware

import (
	"context"
	"net/http"
	"testing"
)

// Test PermissionsMiddleware

func TestGetPermissionsFromContext_EmptyContext(t *testing.T) {
	req := &http.Request{}
	ctx := context.Background()
	req = req.WithContext(ctx)

	perms := GetPermissionsFromContext(req)
	if len(perms) > 0 {
		t.Error("Expected empty permissions from empty context")
	}
}

func TestGetPermissionsFromContext_WithPermissions(t *testing.T) {
	req := &http.Request{}
	ctx := context.Background()
	req = req.WithContext(ctx)

	// This test demonstrates the function exists and can be called
	perms := GetPermissionsFromContext(req)
	if perms == nil {
		t.Error("Expected permissions slice, got nil")
	}
}

func TestHasPermissionInContext(t *testing.T) {
	req := &http.Request{}
	ctx := context.Background()
	req = req.WithContext(ctx)

	// Should return false for empty context
	has := HasPermissionInContext(req, "VIEW_USERS")
	if has {
		t.Error("Expected false for permission in empty context")
	}
}

func TestExtractPermissionsFromString(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Single permission",
			input:    "VIEW_USERS",
			expected: 1,
		},
		{
			name:     "Multiple permissions",
			input:    "VIEW_USERS,CREATE_USERS,DELETE_USERS",
			expected: 3,
		},
		{
			name:     "With spaces",
			input:    "VIEW_USERS, CREATE_USERS, DELETE_USERS",
			expected: 3,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			perms := ExtractPermissionsFromString(tt.input)
			if len(perms) != tt.expected {
				t.Errorf("Expected %d permissions, got %d", tt.expected, len(perms))
			}
		})
	}
}

func TestPermissionContextKey(t *testing.T) {
	if PermissionContextKey != "permissions" {
		t.Errorf("PermissionContextKey should be 'permissions', got '%s'", PermissionContextKey)
	}
}

func TestAdminOnlyMiddleware(t *testing.T) {
	middleware := AdminOnlyMiddleware()

	if middleware == nil {
		t.Error("AdminOnlyMiddleware should return a middleware function")
	}
}

func TestManagerOrAdminMiddleware(t *testing.T) {
	middleware := ManagerOrAdminMiddleware()

	if middleware == nil {
		t.Error("ManagerOrAdminMiddleware should return a middleware function")
	}
}
