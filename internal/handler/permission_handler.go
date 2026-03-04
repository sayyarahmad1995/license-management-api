package handler

import (
	"encoding/json"
	"net/http"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/middleware"
	"license-management-api/internal/models"
	"license-management-api/internal/service"
)

type PermissionHandler struct {
	permissionSvc service.PermissionService
	userRepo      interface{} // Would be IUserRepository in practice
}

// NewPermissionHandler creates a new permission handler
func NewPermissionHandler(permissionSvc service.PermissionService) *PermissionHandler {
	return &PermissionHandler{
		permissionSvc: permissionSvc,
	}
}

// GetUserPermissions returns all permissions for a user
// @Summary Get user permissions
// @Description Get all permissions (role + custom) for a specific user
// @Tags Permissions
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} dto.UserPermissionsResponse "User permissions"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/users/{userId} [get]
// @Security BearerAuth
func (ph *PermissionHandler) GetUserPermissions(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// In a real implementation, parse userID from URL
	userID := 1 // Placeholder

	perms, err := ph.permissionSvc.GetUserPermissions(userID)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch user permissions"))
		return
	}

	permStrs := make([]string, len(perms))
	for i, p := range perms {
		permStrs[i] = string(p)
	}

	response := &dto.UserPermissionsResponse{
		UserID:      userID,
		TotalCount:  len(perms),
		Permissions: permStrs,
	}

	writeJSON(w, http.StatusOK, response)
}

// GetRolePermissions returns all default permissions for a role
// @Summary Get role permissions
// @Description Get all default permissions assigned to a role
// @Tags Permissions
// @Produce json
// @Param role path string true "Role (User, Manager, Admin)"
// @Success 200 {object} dto.RolePermissionsResponse "Role permissions"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/roles/{role} [get]
// @Security BearerAuth
func (ph *PermissionHandler) GetRolePermissions(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	roleStr := "Admin" // Placeholder, would get from URL param
	userRole := models.UserRole(roleStr)

	perms := ph.permissionSvc.GetRolePermissions(userRole)
	permStrs := make([]string, len(perms))
	for i, p := range perms {
		permStrs[i] = string(p)
	}

	response := &dto.RolePermissionsResponse{
		Role:        roleStr,
		TotalCount:  len(perms),
		Permissions: permStrs,
	}

	writeJSON(w, http.StatusOK, response)
}

// GrantPermission grants a permission to a user
// @Summary Grant permission to user
// @Description Grant a specific permission to a user (overrides role defaults)
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body dto.GrantPermissionRequest true "Permission grant request"
// @Success 200 {object} dto.PermissionManagementResponse "Permission granted"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/grant [post]
// @Security BearerAuth
func (ph *PermissionHandler) GrantPermission(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.GrantPermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if err := ph.permissionSvc.GrantPermissionToUser(req.UserID, models.Permission(req.Permission), userID, req.Reason); err != nil {
		writeError(w, errors.NewInternalError("Failed to grant permission"))
		return
	}

	response := &dto.PermissionManagementResponse{
		Status:     "success",
		Message:    "Permission granted successfully",
		UserID:     req.UserID,
		Permission: req.Permission,
	}

	writeJSON(w, http.StatusOK, response)
}

// RevokePermission revokes a permission from a user
// @Summary Revoke permission from user
// @Description Revoke a specific permission from a user
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body dto.RevokePermissionRequest true "Permission revoke request"
// @Success 200 {object} dto.PermissionManagementResponse "Permission revoked"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/revoke [post]
// @Security BearerAuth
func (ph *PermissionHandler) RevokePermission(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.RevokePermissionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if err := ph.permissionSvc.RevokePermissionFromUser(req.UserID, models.Permission(req.Permission), userID, req.Reason); err != nil {
		writeError(w, errors.NewInternalError("Failed to revoke permission"))
		return
	}

	response := &dto.PermissionManagementResponse{
		Status:     "success",
		Message:    "Permission revoked successfully",
		UserID:     req.UserID,
		Permission: req.Permission,
	}

	writeJSON(w, http.StatusOK, response)
}

// SetRolePermissions sets all permissions for a role
// @Summary Set role permissions
// @Description Set all permissions for a specific role (replaces existing)
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body dto.SetRolePermissionsRequest true "Role permissions request"
// @Success 200 {object} dto.RolePermissionsResponse "Role permissions updated"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/roles [post]
// @Security BearerAuth
func (ph *PermissionHandler) SetRolePermissions(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	var req dto.SetRolePermissionsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Convert strings to Permission types
	permissions := make([]models.Permission, len(req.Permissions))
	for i, pStr := range req.Permissions {
		permissions[i] = models.Permission(pStr)
	}

	userRole := models.UserRole(req.Role)
	if err := ph.permissionSvc.SetRolePermissions(userRole, permissions); err != nil {
		writeError(w, errors.NewInternalError("Failed to set role permissions"))
		return
	}

	response := &dto.RolePermissionsResponse{
		Role:        req.Role,
		TotalCount:  len(permissions),
		Permissions: req.Permissions,
	}

	writeJSON(w, http.StatusOK, response)
}

// ResetUserPermissions resets a user's custom permissions to role defaults
// @Summary Reset user permissions
// @Description Reset user's custom permissions and restore role defaults
// @Tags Permissions
// @Produce json
// @Param userId path int true "User ID"
// @Success 200 {object} dto.ResetPermissionsResponse "Permissions reset"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 403 {object} errors.ApiError "Forbidden - admin required"
// @Router /permissions/users/{userId}/reset [post]
// @Security BearerAuth
func (ph *PermissionHandler) ResetUserPermissions(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value(middleware.UserRoleContextKey).(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID := 1 // Placeholder, would get from URL param

	if err := ph.permissionSvc.ResetUserPermissionsToRole(userID); err != nil {
		writeError(w, errors.NewInternalError("Failed to reset user permissions"))
		return
	}

	response := &dto.ResetPermissionsResponse{
		Status:  "success",
		Message: "User permissions reset to role defaults",
		UserID:  userID,
		Role:    "User", // Placeholder
	}

	writeJSON(w, http.StatusOK, response)
}

// CheckPermission checks if user has required permission(s)
// @Summary Check permissions
// @Description Check if user has required permission(s) - useful for frontend
// @Tags Permissions
// @Accept json
// @Produce json
// @Param request body dto.PermissionCheckRequest true "Permission check request"
// @Success 200 {object} dto.PermissionCheckResponse "Permission check result"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /permissions/check [post]
// @Security BearerAuth
func (ph *PermissionHandler) CheckPermission(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.PermissionCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Convert to Permission types
	permissions := make([]models.Permission, len(req.Permissions))
	for i, pStr := range req.Permissions {
		permissions[i] = models.Permission(pStr)
	}

	// Get user's actual permissions
	userPerms, err := ph.permissionSvc.GetUserPermissions(userID)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to check permissions"))
		return
	}

	// Check which permissions user has
	granted := []string{}
	denied := []string{}

	for _, reqPerm := range req.Permissions {
		found := false
		for _, userPerm := range userPerms {
			if userPerm == models.Permission(reqPerm) {
				found = true
				break
			}
		}
		if found {
			granted = append(granted, reqPerm)
		} else {
			denied = append(denied, reqPerm)
		}
	}

	response := &dto.PermissionCheckResponse{
		UserID:          userID,
		Requested:       req.Permissions,
		Granted:         granted,
		Denied:          denied,
		HasAllRequested: len(denied) == 0,
		HasAnyRequested: len(granted) > 0,
	}

	writeJSON(w, http.StatusOK, response)
}
