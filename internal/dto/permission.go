package dto

import "time"

// GrantPermissionRequest represents a request to grant a permission
type GrantPermissionRequest struct {
	UserID     int    `json:"user_id" validate:"required"`
	Permission string `json:"permission" validate:"required"`
	Reason     string `json:"reason"`
}

// RevokePermissionRequest represents a request to revoke a permission
type RevokePermissionRequest struct {
	UserID     int    `json:"user_id" validate:"required"`
	Permission string `json:"permission" validate:"required"`
	Reason     string `json:"reason"`
}

// SetRolePermissionsRequest represents a request to set role permissions
type SetRolePermissionsRequest struct {
	Role        string   `json:"role" validate:"required,oneof=User Manager Admin"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
}

// PermissionResponse represents permission information
type PermissionResponse struct {
	Permission string    `json:"permission"`
	Granted    bool      `json:"granted"`
	GrantedAt  time.Time `json:"granted_at,omitempty"`
	GrantedBy  int       `json:"granted_by,omitempty"`
}

// UserPermissionsResponse represents all permissions for a user
type UserPermissionsResponse struct {
	UserID        int      `json:"user_id"`
	Role          string   `json:"role"`
	TotalCount    int      `json:"total_count"`
	Permissions   []string `json:"permissions"`
	CustomGrants  []string `json:"custom_grants,omitempty"`
	CustomRevokes []string `json:"custom_revokes,omitempty"`
}

// RolePermissionsResponse represents all permissions for a role
type RolePermissionsResponse struct {
	Role        string    `json:"role"`
	TotalCount  int       `json:"total_count"`
	Permissions []string  `json:"permissions"`
	LastUpdated time.Time `json:"last_updated"`
}

// PermissionManagementResponse represents the result of a permission operation
type PermissionManagementResponse struct {
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	UserID     int       `json:"user_id,omitempty"`
	Permission string    `json:"permission,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// ResetPermissionsResponse represents the result of resetting user permissions
type ResetPermissionsResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	UserID    int       `json:"user_id"`
	Role      string    `json:"role"`
	Timestamp time.Time `json:"timestamp"`
}

// PermissionCheckRequest checks if user has permission(s)
type PermissionCheckRequest struct {
	UserID      int      `json:"user_id" validate:"required"`
	Permissions []string `json:"permissions" validate:"required,min=1"`
	RequireAll  bool     `json:"require_all"` // true = AND logic, false = OR logic
}

// PermissionCheckResponse returns permission check result
type PermissionCheckResponse struct {
	UserID          int      `json:"user_id"`
	Requested       []string `json:"requested"`
	Granted         []string `json:"granted"`
	Denied          []string `json:"denied"`
	HasAllRequested bool     `json:"has_all_requested"`
	HasAnyRequested bool     `json:"has_any_requested"`
}
