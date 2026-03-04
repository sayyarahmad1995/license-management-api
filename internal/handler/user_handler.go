package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/internal/service"
	"license-management-api/pkg/utils"

	"github.com/go-chi/chi/v5"
)

type UserHandler struct {
	userRepo      repository.IUserRepository
	licenseRepo   repository.ILicenseRepository
	paginationSvc service.PaginationService
	dataExportSvc service.DataExportService
}

func NewUserHandler(userRepo repository.IUserRepository, licenseRepo repository.ILicenseRepository, paginationSvc service.PaginationService, dataExportSvc service.DataExportService) *UserHandler {
	return &UserHandler{
		userRepo:      userRepo,
		licenseRepo:   licenseRepo,
		paginationSvc: paginationSvc,
		dataExportSvc: dataExportSvc,
	}
}

// GetUsers retrieves all users (Admin only)
// @Summary Get all users
// @Description Retrieve a paginated list of all users (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param pageIndex query int false "Page index (default 1)"
// @Param pageSize query int false "Page size (default 10, max 100)"
// @Success 200 {object} map[string]interface{} "List of users"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /users [get]
// @Security BearerAuth
func (h *UserHandler) GetUsers(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin (set by auth middleware)
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// Parse pagination parameters
	page := 1
	limit := 10
	if pageStr := r.URL.Query().Get("pageIndex"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}
	if limitStr := r.URL.Query().Get("pageSize"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			limit = l
		}
	}

	// Parse search parameter
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	// Get all users (we'll filter in-memory)
	users, _, err := h.userRepo.GetAll(1, 10000)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve users"))
		return
	}

	// Apply search filter if provided
	if search != "" {
		searchLower := strings.ToLower(search)
		var filtered []models.User
		for _, user := range users {
			if strings.Contains(strings.ToLower(user.Username), searchLower) ||
				strings.Contains(strings.ToLower(user.Email), searchLower) {
				filtered = append(filtered, user)
			}
		}
		users = filtered
	}

	// Get total count after filtering
	total := len(users)

	// Apply pagination
	offset := (page - 1) * limit
	if offset >= len(users) {
		users = []models.User{}
	} else if offset+limit > len(users) {
		users = users[offset:]
	} else {
		users = users[offset : offset+limit]
	}

	// Map to response DTOs (exclude passwords)
	userDtos := make([]map[string]interface{}, len(users))
	for i, user := range users {
		userDtos[i] = map[string]interface{}{
			"id":         user.ID,
			"username":   user.Username,
			"email":      user.Email,
			"role":       user.Role,
			"status":     user.Status,
			"created_at": user.CreatedAt,
			"last_login": user.LastLogin,
		}
	}

	totalPages := (total + limit - 1) / limit

	response := map[string]interface{}{
		"pageIndex":  page,
		"pageSize":   limit,
		"total":      total,
		"totalPages": totalPages,
		"count":      len(userDtos),
		"users":      userDtos,
	}

	if search != "" {
		response["search"] = search
	}

	writeJSON(w, http.StatusOK, response)
}

// ExportUsers exports all users as CSV (Admin only)
// @Summary Export all users as CSV
// @Description Download all users in CSV format (Admin only)
// @Tags Users
// @Accept json
// @Produce text/csv
// @Success 200 {file} file "CSV file"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /users/export [get]
// @Security BearerAuth
func (h *UserHandler) ExportUsers(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// Get all users
	users, _, err := h.userRepo.GetAll(1, 10000)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve users"))
		return
	}

	// Convert to CSV format
	csvUsers := make([]utils.UserCSVRow, len(users))
	for i, user := range users {
		verifiedAt := (*string)(nil)
		if user.VerifiedAt != nil {
			formatted := utils.FormatTimestamp(*user.VerifiedAt)
			verifiedAt = &formatted
		}

		lastLogin := (*string)(nil)
		if user.LastLogin != nil {
			formatted := utils.FormatTimestamp(*user.LastLogin)
			lastLogin = &formatted
		}

		csvUsers[i] = utils.UserCSVRow{
			ID:         user.ID,
			Username:   user.Username,
			Email:      user.Email,
			Role:       user.Role,
			Status:     user.Status,
			CreatedAt:  utils.FormatTimestamp(user.CreatedAt),
			VerifiedAt: verifiedAt,
			LastLogin:  lastLogin,
		}
	}

	// Generate CSV
	csvData, err := utils.GenerateUserCSV(csvUsers)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to generate CSV"))
		return
	}

	// Set response headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"users_"+time.Now().Format("20060102_150405")+".csv\"")
	w.Header().Set("Content-Length", string(rune(len(csvData))))

	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

// GetUser retrieves a specific user
// @Summary Get user by ID
// @Description Retrieve a specific user's details
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User details"
// @Failure 403 {object} errors.ApiError "Access denied"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /users/{id} [get]
// @Security BearerAuth
func (h *UserHandler) GetUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	// Check authorization: users can only view their own profile unless they're admin
	currentUserID, _ := r.Context().Value("userId").(int)
	role, _ := r.Context().Value("userRole").(string)

	if userID != currentUserID && role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("You can only view your own profile"))
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	})
}

// UpdateUser updates a user profile
// @Summary Update user profile
// @Description Update user profile information
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body dto.UpdateUserDto true "Update request"
// @Success 200 {object} map[string]interface{} "User updated successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Access denied"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /users/{id} [put]
// @Security BearerAuth
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	// Check authorization
	currentUserID, _ := r.Context().Value("userId").(int)
	role, _ := r.Context().Value("userRole").(string)

	if userID != currentUserID && role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("You can only update your own profile"))
		return
	}

	var req dto.UpdateUserDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found"))
		return
	}

	// Update allowed fields
	if req.Username != "" {
		user.Username = req.Username
	}
	if req.NotifyLicenseExpiry != nil {
		user.NotifyLicenseExpiry = *req.NotifyLicenseExpiry
	}
	if req.NotifyAccountActivity != nil {
		user.NotifyAccountActivity = *req.NotifyAccountActivity
	}
	if req.NotifySystemAnnouncements != nil {
		user.NotifySystemAnnouncements = *req.NotifySystemAnnouncements
	}

	if err := h.userRepo.Update(user); err != nil {
		writeError(w, errors.NewInternalError("Failed to update user"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message":  "User updated successfully",
		"id":       user.ID,
		"username": user.Username,
		"email":    user.Email,
	})
}

// DeleteUser deletes a user (Admin only)
// @Summary Delete a user
// @Description Delete a user account (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} map[string]interface{} "User deleted successfully"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /users/{id} [delete]
// @Security BearerAuth
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	_, err = h.userRepo.GetByID(userID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found"))
		return
	}

	if err := h.userRepo.Delete(userID); err != nil {
		writeError(w, errors.NewInternalError("Failed to delete user"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User deleted successfully",
		"id":      userID,
	})
}

// UpdateUserRole updates a user's role (Admin only)
// @Summary Update user role
// @Description Change a user's role (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body dto.UpdateUserRoleDto true "Role update request"
// @Success 200 {object} map[string]interface{} "User role updated successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /users/{id}/role [patch]
// @Security BearerAuth
func (h *UserHandler) UpdateUserRole(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	var req dto.UpdateUserRoleDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found"))
		return
	}

	user.Role = req.Role
	now := time.Now().UTC()
	user.UpdatedAt = &now

	if err := h.userRepo.Update(user); err != nil {
		writeError(w, errors.NewInternalError("Failed to update user role"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User role updated successfully",
		"id":      user.ID,
		"role":    user.Role,
	})
}

// UpdateUserStatus updates a user's status (Admin only)
// @Summary Update user status
// @Description Change a user's account status (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param request body dto.UpdateUserStatusDto true "Status update request"
// @Success 200 {object} map[string]interface{} "User status updated successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /users/{id}/status [patch]
// @Security BearerAuth
func (h *UserHandler) UpdateUserStatus(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userIDStr := chi.URLParam(r, "id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	var req dto.UpdateUserStatusDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	user, err := h.userRepo.GetByID(userID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found"))
		return
	}

	status := strings.ToLower(req.Status)
	now := time.Now().UTC()

	switch status {
	case "unverified":
		user.Status = string(models.UserStatusUnverified)
		user.VerifiedAt = nil
	case "verified":
		user.Status = string(models.UserStatusVerified)
		user.VerifiedAt = &now
	case "active":
		user.Status = string(models.UserStatusActive)
	case "blocked":
		user.Status = string(models.UserStatusBlocked)
		user.BlockedAt = &now
	default:
		writeError(w, errors.NewBadRequestError("Invalid status. Valid values: Unverified, Verified, Active, Blocked"))
		return
	}

	user.UpdatedAt = &now

	if err := h.userRepo.Update(user); err != nil {
		writeError(w, errors.NewInternalError("Failed to update user status"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "User status updated successfully",
		"id":      user.ID,
		"status":  user.Status,
	})
}

// GetUserByLicenseKey finds a user by their license key (Admin only)
// @Summary Get user by license key
// @Description Find the user associated with a specific license key (Admin only)
// @Tags Users
// @Accept json
// @Produce json
// @Param key path string true "License key"
// @Success 200 {object} map[string]interface{} "User details"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "License or user not found"
// @Router /users/by-license/{key} [get]
// @Security BearerAuth
func (h *UserHandler) GetUserByLicenseKey(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	licenseKey := chi.URLParam(r, "key")
	if licenseKey == "" {
		writeError(w, errors.NewBadRequestError("License key is required"))
		return
	}

	// Find license by key
	license, err := h.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		writeError(w, errors.NewNotFoundError("License key not found"))
		return
	}

	// Get user for this license
	user, err := h.userRepo.GetByID(license.UserID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("User not found for this license"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":         user.ID,
		"username":   user.Username,
		"email":      user.Email,
		"role":       user.Role,
		"status":     user.Status,
		"created_at": user.CreatedAt,
		"last_login": user.LastLogin,
	})
}
