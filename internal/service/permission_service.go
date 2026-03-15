package service

import (
	"fmt"
	"sync"
	"time"

	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

// PermissionService provides permission checking and management
type PermissionService interface {
	// Permission checking
	HasPermission(userID int, permission models.Permission) bool
	HasAnyPermission(userID int, permissions ...models.Permission) bool
	HasAllPermissions(userID int, permissions ...models.Permission) bool
	GetUserPermissions(userID int) ([]models.Permission, error)
	GetRolePermissions(role models.UserRole) []models.Permission

	// Permission management
	GrantPermissionToUser(userID int, permission models.Permission, grantedBy int, reason string) error
	RevokePermissionFromUser(userID int, permission models.Permission, revokedBy int, reason string) error
	SetRolePermissions(role models.UserRole, permissions []models.Permission) error
	GetUserCustomPermissions(userID int) ([]models.UserPermission, error)
	ResetUserPermissionsToRole(userID int) error

	// Cache management
	InvalidateUserPermissionCache(userID int)
	InvalidateRolePermissionCache(role models.UserRole)
}

type permissionService struct {
	userRepo  repository.IUserRepository
	mu        sync.RWMutex
	userCache map[int][]models.Permission             // userID -> permissions
	roleCache map[models.UserRole][]models.Permission // role -> permissions
}

// NewPermissionService creates a new permission service
func NewPermissionService(userRepo repository.IUserRepository) PermissionService {
	ps := &permissionService{
		userRepo:  userRepo,
		userCache: make(map[int][]models.Permission),
		roleCache: make(map[models.UserRole][]models.Permission),
	}

	// Initialize role cache with default permissions
	for role, perms := range models.DefaultPermissionsByRole {
		ps.roleCache[role] = perms
	}

	return ps
}

// HasPermission checks if a user has a specific permission
func (ps *permissionService) HasPermission(userID int, permission models.Permission) bool {
	return ps.HasAnyPermission(userID, permission)
}

// HasAnyPermission checks if user has any of the given permissions
func (ps *permissionService) HasAnyPermission(userID int, permissions ...models.Permission) bool {
	userPerms := ps.getPermissionsWithCache(userID)

	for _, requiredPerm := range permissions {
		for _, userPerm := range userPerms {
			if userPerm == requiredPerm {
				return true
			}
		}
	}
	return false
}

// HasAllPermissions checks if user has all given permissions
func (ps *permissionService) HasAllPermissions(userID int, permissions ...models.Permission) bool {
	userPerms := ps.getPermissionsWithCache(userID)

	for _, requiredPerm := range permissions {
		found := false
		for _, userPerm := range userPerms {
			if userPerm == requiredPerm {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

// GetUserPermissions returns all permissions for a user (role + custom)
func (ps *permissionService) GetUserPermissions(userID int) ([]models.Permission, error) {
	return ps.getPermissionsWithCache(userID), nil
}

// GetRolePermissions returns all default permissions for a role
func (ps *permissionService) GetRolePermissions(role models.UserRole) []models.Permission {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if perms, ok := ps.roleCache[role]; ok {
		return perms
	}

	// Fallback to model defaults
	if perms, ok := models.DefaultPermissionsByRole[role]; ok {
		return perms
	}

	return []models.Permission{}
}

// GrantPermissionToUser grants a permission to a user (override)
func (ps *permissionService) GrantPermissionToUser(userID int, permission models.Permission, grantedBy int, reason string) error {
	// In a real implementation, this would save to database
	// For now, we invalidate the cache so permissions are recalculated
	ps.InvalidateUserPermissionCache(userID)
	return nil
}

// RevokePermissionFromUser revokes a permission from a user (override)
func (ps *permissionService) RevokePermissionFromUser(userID int, permission models.Permission, revokedBy int, reason string) error {
	// In a real implementation, this would save to database
	ps.InvalidateUserPermissionCache(userID)
	return nil
}

// SetRolePermissions sets all permissions for a role
func (ps *permissionService) SetRolePermissions(role models.UserRole, permissions []models.Permission) error {
	ps.mu.Lock()
	ps.roleCache[role] = permissions
	ps.mu.Unlock()

	// Invalidate all user caches since role permissions changed
	ps.mu.Lock()
	ps.userCache = make(map[int][]models.Permission)
	ps.mu.Unlock()

	return nil
}

// GetUserCustomPermissions returns custom overrides for a user
func (ps *permissionService) GetUserCustomPermissions(userID int) ([]models.UserPermission, error) {
	// In a real implementation, query from database
	return []models.UserPermission{}, nil
}

// ResetUserPermissionsToRole removes all custom permissions and restores role defaults
func (ps *permissionService) ResetUserPermissionsToRole(userID int) error {
	ps.InvalidateUserPermissionCache(userID)
	return nil
}

// InvalidateUserPermissionCache clears cached permissions for a user
func (ps *permissionService) InvalidateUserPermissionCache(userID int) {
	ps.mu.Lock()
	delete(ps.userCache, userID)
	ps.mu.Unlock()
}

// InvalidateRolePermissionCache clears cached permissions for a role
func (ps *permissionService) InvalidateRolePermissionCache(role models.UserRole) {
	ps.mu.Lock()
	delete(ps.roleCache, role)
	ps.mu.Unlock()

	// Re-initialize with defaults
	if perms, ok := models.DefaultPermissionsByRole[role]; ok {
		ps.roleCache[role] = perms
	}
}

// getPermissionsWithCache returns user permissions with caching
func (ps *permissionService) getPermissionsWithCache(userID int) []models.Permission {
	ps.mu.RLock()
	if perms, ok := ps.userCache[userID]; ok {
		ps.mu.RUnlock()
		return perms
	}
	ps.mu.RUnlock()

	// Get user's role and build permission set
	permissions := ps.buildUserPermissions(userID)

	// Cache the result
	ps.mu.Lock()
	ps.userCache[userID] = permissions
	ps.mu.Unlock()

	return permissions
}

// buildUserPermissions constructs full permission set for user
func (ps *permissionService) buildUserPermissions(userID int) []models.Permission {
	// Get user's role from repository
	users, _, err := ps.userRepo.GetAll(1, 10000)
	if err != nil || len(users) == 0 {
		return []models.Permission{}
	}

	var userRole models.UserRole
	for _, user := range users {
		if user.ID == userID {
			userRole = models.UserRole(user.Role)
			break
		}
	}

	if userRole == "" {
		userRole = models.UserRoleUser // default to User role
	}

	// Get role permissions
	rolePerms := ps.GetRolePermissions(userRole)

	// In a full implementation, would also apply custom user overrides here
	// For now, just return role permissions
	return rolePerms
}

// PermissionChecker provides helper for permission validation
type PermissionChecker struct {
	permSvc PermissionService
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(permSvc PermissionService) *PermissionChecker {
	return &PermissionChecker{
		permSvc: permSvc,
	}
}

// RequirePermission checks permission and returns formatted error message
func (pc *PermissionChecker) RequirePermission(userID int, permission models.Permission) error {
	if !pc.permSvc.HasPermission(userID, permission) {
		return fmt.Errorf("user does not have required permission: %s", permission)
	}
	return nil
}

// RequireAnyPermission checks if user has any of the permissions
func (pc *PermissionChecker) RequireAnyPermission(userID int, permissions ...models.Permission) error {
	if !pc.permSvc.HasAnyPermission(userID, permissions...) {
		return fmt.Errorf("user does not have any of the required permissions: %v", permissions)
	}
	return nil
}

// RequireAllPermissions checks if user has all permissions
func (pc *PermissionChecker) RequireAllPermissions(userID int, permissions ...models.Permission) error {
	if !pc.permSvc.HasAllPermissions(userID, permissions...) {
		return fmt.Errorf("user does not have all required permissions: %v", permissions)
	}
	return nil
}

// PermissionStats tracks permission usage
type PermissionStats struct {
	Permission string    `json:"permission"`
	UserCount  int       `json:"user_count"`
	LastUsed   time.Time `json:"last_used"`
}

// RequireAdmin is a helper that checks for admin role
func RequireAdmin(userRole string) error {
	if userRole != string(models.UserRoleAdmin) {
		return fmt.Errorf("admin role required")
	}
	return nil
}

// RequireManager is a helper that checks for manager or admin
func RequireManager(userRole string) error {
	if userRole != string(models.UserRoleManager) && userRole != string(models.UserRoleAdmin) {
		return fmt.Errorf("manager role or higher required")
	}
	return nil
}
