package handler

import (
	"net/http"
	"strconv"
	"strings"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

type AuditHandler struct {
	auditRepo repository.IAuditLogRepository
}

func NewAuditHandler(auditRepo repository.IAuditLogRepository) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
	}
}

// mapAuditLogToDto converts a models.AuditLog to a dto.AuditLogDto with camelCase JSON fields
func mapAuditLogToDto(log models.AuditLog) dto.AuditLogDto {
	return dto.AuditLogDto{
		ID:         log.ID,
		Action:     log.Action,
		EntityType: log.EntityType,
		EntityID:   log.EntityID,
		UserID:     log.UserID,
		Details:    log.Details,
		IpAddress:  log.IpAddress,
		Timestamp:  log.Timestamp,
	}
}

// GetAuditLogs retrieves audit logs (Admin only or user's own logs)
// @Summary Get audit logs
// @Description Retrieve audit logs (Admins see all, users see only their own)
// @Tags Audit
// @Accept json
// @Produce json
// @Param pageIndex query int false "Page index (default 1)"
// @Param pageSize query int false "Page size (default 20, max 100)"
// @Param action query string false "Filter by action"
// @Param entity_type query string false "Filter by entity type"
// @Success 200 {object} map[string]interface{} "List of audit logs"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Router /audit-logs [get]
// @Security BearerAuth
func (h *AuditHandler) GetAuditLogs(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("userRole").(string)
	userID, _ := r.Context().Value("userId").(int)

	// Parse pagination parameters
	pageIndex := 1
	pageSize := 20
	if pageStr := r.URL.Query().Get("pageIndex"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			pageIndex = p
		}
	}
	if limitStr := r.URL.Query().Get("pageSize"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			pageSize = l
		}
	}

	// Parse filter parameters
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	entityType := strings.TrimSpace(r.URL.Query().Get("entity_type"))

	// Get all logs (we'll filter in-memory)
	logs, _, err := h.auditRepo.GetAll(1, 10000)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve audit logs"))
		return
	}

	// Filter based on role
	var filteredLogs []models.AuditLog
	for _, log := range logs {
		// If not admin, only show user's own logs
		if role != string(models.UserRoleAdmin) {
			if log.UserID == nil || *log.UserID != userID {
				continue
			}
		}
		filteredLogs = append(filteredLogs, log)
	}

	// Apply action filter if provided
	if action != "" {
		var actionFiltered []models.AuditLog
		for _, log := range filteredLogs {
			if strings.Contains(strings.ToLower(log.Action), strings.ToLower(action)) {
				actionFiltered = append(actionFiltered, log)
			}
		}
		filteredLogs = actionFiltered
	}

	// Apply entity_type filter if provided
	if entityType != "" {
		var typeFiltered []models.AuditLog
		for _, log := range filteredLogs {
			if strings.Contains(strings.ToLower(log.EntityType), strings.ToLower(entityType)) {
				typeFiltered = append(typeFiltered, log)
			}
		}
		filteredLogs = typeFiltered
	}

	// Get total count after filtering
	total := len(filteredLogs)

	// Apply pagination
	offset := (pageIndex - 1) * pageSize
	if offset >= len(filteredLogs) {
		filteredLogs = []models.AuditLog{}
	} else if offset+pageSize > len(filteredLogs) {
		filteredLogs = filteredLogs[offset:]
	} else {
		filteredLogs = filteredLogs[offset : offset+pageSize]
	}

	totalPages := (total + pageSize - 1) / pageSize

	// Convert audit log models to DTOs for camelCase serialization
	auditDtos := make([]dto.AuditLogDto, len(filteredLogs))
	for i, log := range filteredLogs {
		auditDtos[i] = mapAuditLogToDto(log)
	}

	response := map[string]interface{}{
		"pageIndex":  pageIndex,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
		"count":      len(auditDtos),
		"logs":       auditDtos,
	}

	if action != "" {
		response["action"] = action
	}
	if entityType != "" {
		response["entityType"] = entityType
	}

	writeJSON(w, http.StatusOK, response)
}

// GetAuditLogsByUser retrieves audit logs for a specific user (Admin only)
// @Summary Get audit logs for a user
// @Description Retrieve audit logs specific to a user (Admin only)
// @Tags Audit
// @Accept json
// @Produce json
// @Param user_id query int true "User ID to filter logs"
// @Param pageIndex query int false "Page index (default 1)"
// @Param pageSize query int false "Page size (default 20, max 100)"
// @Success 200 {object} map[string]interface{} "List of user audit logs"
// @Failure 400 {object} errors.ApiError "Invalid user ID"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /audit-logs/user [get]
// @Security BearerAuth
func (h *AuditHandler) GetAuditLogsByUser(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, _ := r.Context().Value("userRole").(string)
	if role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// Parse pagination parameters
	pageIndex := 1
	pageSize := 20
	if pageStr := r.URL.Query().Get("pageIndex"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			pageIndex = p
		}
	}
	if limitStr := r.URL.Query().Get("pageSize"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 && l <= 100 {
			pageSize = l
		}
	}

	userIDStr := r.URL.Query().Get("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid user ID"))
		return
	}

	logs, _, err := h.auditRepo.GetAll(1, 10000) // Get all logs for filtering
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve audit logs"))
		return
	}

	// Filter for specific user
	userLogs := []models.AuditLog{}
	for _, log := range logs {
		if log.UserID != nil && *log.UserID == userID {
			userLogs = append(userLogs, log)
		}
	}

	// Get total count after filtering
	total := len(userLogs)

	// Apply pagination
	offset := (pageIndex - 1) * pageSize
	if offset >= len(userLogs) {
		userLogs = []models.AuditLog{}
	} else if offset+pageSize > len(userLogs) {
		userLogs = userLogs[offset:]
	} else {
		userLogs = userLogs[offset : offset+pageSize]
	}

	totalPages := (total + pageSize - 1) / pageSize

	// Convert audit log models to DTOs for camelCase serialization
	auditDtos := make([]dto.AuditLogDto, len(userLogs))
	for i, log := range userLogs {
		auditDtos[i] = mapAuditLogToDto(log)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"pageIndex":  pageIndex,
		"pageSize":   pageSize,
		"userId":     userID,
		"total":      total,
		"totalPages": totalPages,
		"count":      len(auditDtos),
		"logs":       auditDtos,
	})
}

// GetAuditLogStats retrieves statistics about audit logs (Admin only)
// @Summary Get audit log statistics
// @Description Get aggregated statistics about audit logs (Admin only)
// @Tags Audit
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Audit log statistics"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /audit-logs/stats [get]
// @Security BearerAuth
func (h *AuditHandler) GetAuditLogStats(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, _ := r.Context().Value("userRole").(string)
	if role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	logs, _, err := h.auditRepo.GetAll(1, 10000) // Get all logs
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve audit logs"))
		return
	}

	// Count by action
	actionCounts := make(map[string]int)
	entityCounts := make(map[string]int)

	for _, log := range logs {
		actionCounts[log.Action]++
		entityCounts[log.EntityType]++
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"total_logs":     len(logs),
		"action_summary": actionCounts,
		"entity_summary": entityCounts,
	})
}
