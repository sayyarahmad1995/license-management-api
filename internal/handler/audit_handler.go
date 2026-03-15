package handler

import (
	"encoding/json"
	"fmt"
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
)

type AuditHandler struct {
	auditRepo repository.IAuditLogRepository
	userRepo  repository.IUserRepository
	auditSvc  service.AuditService
}

func NewAuditHandler(auditRepo repository.IAuditLogRepository, userRepo repository.IUserRepository, auditSvc service.AuditService) *AuditHandler {
	return &AuditHandler{
		auditRepo: auditRepo,
		userRepo:  userRepo,
		auditSvc:  auditSvc,
	}
}

// mapAuditLogToDto converts a models.AuditLog to a dto.AuditLogDto with camelCase JSON fields
func mapAuditLogToDto(log models.AuditLog, username *string) dto.AuditLogDto {
	return dto.AuditLogDto{
		ID:         log.ID,
		Action:     log.Action,
		EntityType: log.EntityType,
		EntityID:   log.EntityID,
		UserID:     log.UserID,
		Username:   username,
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

	// Get all users for username lookup
	allUsers, _, _ := h.userRepo.GetAll(1, 10000)
	userMap := make(map[int]string)
	for _, user := range allUsers {
		userMap[user.ID] = user.Username
	}

	// Convert audit log models to DTOs for camelCase serialization
	auditDtos := make([]dto.AuditLogDto, len(filteredLogs))
	for i, log := range filteredLogs {
		var username *string
		if log.UserID != nil {
			if name, exists := userMap[*log.UserID]; exists {
				username = &name
			}
		}
		auditDtos[i] = mapAuditLogToDto(log, username)
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

	// Get all users for username lookup
	allUsers, _, _ := h.userRepo.GetAll(1, 10000)
	userMap := make(map[int]string)
	for _, user := range allUsers {
		userMap[user.ID] = user.Username
	}

	// Convert audit log models to DTOs for camelCase serialization
	auditDtos := make([]dto.AuditLogDto, len(userLogs))
	for i, log := range userLogs {
		var username *string
		if log.UserID != nil {
			if name, exists := userMap[*log.UserID]; exists {
				username = &name
			}
		}
		auditDtos[i] = mapAuditLogToDto(log, username)
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

// ExportAuditLogs exports audit logs as CSV
// @Summary Export audit logs
// @Description Export audit logs in CSV format (Admin only or user's own logs)
// @Tags Audit
// @Accept json
// @Produce text/csv
// @Param format query string false "Export format (csv)"
// @Param startDate query string false "Start date (YYYY-MM-DD)"
// @Param endDate query string false "End date (YYYY-MM-DD)"
// @Param action query string false "Filter by action"
// @Param userId query int false "Filter by user ID (Admin only)"
// @Success 200 {file} file "CSV file"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /audit-logs/export [get]
// @Security BearerAuth
func (h *AuditHandler) ExportAuditLogs(w http.ResponseWriter, r *http.Request) {
	role, _ := r.Context().Value("userRole").(string)
	userID, _ := r.Context().Value("userId").(int)
	isAdmin := role == string(models.UserRoleAdmin)

	// Parse filter parameters
	action := strings.TrimSpace(r.URL.Query().Get("action"))
	userIDStr := strings.TrimSpace(r.URL.Query().Get("userId"))
	startDateStr := strings.TrimSpace(r.URL.Query().Get("startDate"))
	endDateStr := strings.TrimSpace(r.URL.Query().Get("endDate"))
	ipAddress := utils.GetClientIP(r)

	// Get all logs
	logs, _, err := h.auditRepo.GetAll(1, 10000)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve audit logs"))
		return
	}

	// Filter based on role and parameters
	var filteredLogs []models.AuditLog
	for _, log := range logs {
		// If not admin, only include user's own logs
		if !isAdmin && (log.UserID == nil || *log.UserID != userID) {
			continue
		}

		// Filter by user ID if provided (admin only)
		if userIDStr != "" && isAdmin {
			if userIDFilter, err := strconv.Atoi(userIDStr); err == nil {
				if log.UserID == nil || *log.UserID != userIDFilter {
					continue
				}
			}
		}

		// Filter by action if provided
		if action != "" && !strings.Contains(strings.ToLower(log.Action), strings.ToLower(action)) {
			continue
		}

		// Filter by date range if provided
		if startDateStr != "" || endDateStr != "" {
			var startDate, endDate time.Time
			var err error

			if startDateStr != "" {
				startDate, err = time.Parse("2006-01-02", startDateStr)
				if err == nil {
					// Set time to start of day
					startDate = time.Date(startDate.Year(), startDate.Month(), startDate.Day(), 0, 0, 0, 0, startDate.Location())
					if log.Timestamp.Before(startDate) {
						continue
					}
				}
			}

			if endDateStr != "" {
				endDate, err = time.Parse("2006-01-02", endDateStr)
				if err == nil {
					// Set time to end of day
					endDate = time.Date(endDate.Year(), endDate.Month(), endDate.Day(), 23, 59, 59, 999999999, endDate.Location())
					if log.Timestamp.After(endDate) {
						continue
					}
				}
			}
		}

		filteredLogs = append(filteredLogs, log)
	}

	// Get all users for username lookup
	allUsers, _, _ := h.userRepo.GetAll(1, 10000)
	userMap := make(map[int]string)
	for _, user := range allUsers {
		userMap[user.ID] = user.Username
	}

	// Build CSV content
	csvContent := "ID,Action,Entity Type,Entity ID,User ID,Username,Details,IP Address,Timestamp\n"
	for _, log := range filteredLogs {
		username := ""
		if log.UserID != nil {
			if name, exists := userMap[*log.UserID]; exists {
				username = name
			}
		}

		entityID := ""
		if log.EntityID != nil {
			entityID = strconv.Itoa(*log.EntityID)
		}

		userIDStr := ""
		if log.UserID != nil {
			userIDStr = strconv.Itoa(*log.UserID)
		}

		details := ""
		if log.Details != nil {
			// Escape quotes and newlines in details for CSV
			details = strings.ReplaceAll(*log.Details, "\"", "\"\"")
			details = strings.ReplaceAll(details, "\n", " ")
			details = "\"" + details + "\""
		}

		ipAddress := ""
		if log.IpAddress != nil {
			ipAddress = *log.IpAddress
		}

		csvContent += fmt.Sprintf("%d,%s,%s,%s,%s,%s,%s,%s,%s\n",
			log.ID,
			log.Action,
			log.EntityType,
			entityID,
			userIDStr,
			username,
			details,
			ipAddress,
			log.Timestamp.Format("2006-01-02 15:04:05"),
		)
	}

	// Set response headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"audit_logs_"+time.Now().Format("20060102_150405")+".csv\"")
	w.Header().Set("Content-Length", strconv.Itoa(len(csvContent)))

	// Log the audit log export
	details := map[string]interface{}{
		"exported_records": len(filteredLogs),
		"filter": map[string]interface{}{
			"action":     action,
			"user_id":    userIDStr,
			"start_date": startDateStr,
			"end_date":   endDateStr,
		},
		"exported_at": time.Now().UTC(),
	}
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)
	h.auditSvc.LogAction("AUDIT_LOG_EXPORTED", "AuditLog", 0, &userID, &detailsStr, &ipAddress)

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(csvContent))
}

// DeleteAuditLogs deletes all audit logs (Admin only)
// @Summary Delete all audit logs
// @Description Clear all audit logs from the system (Admin only)
// @Tags Audit
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Audit logs deleted successfully"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 500 {object} errors.ApiError "Internal server error"
// @Router /audit-logs [delete]
// @Security BearerAuth
func (h *AuditHandler) DeleteAuditLogs(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, _ := r.Context().Value("userRole").(string)
	if role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID, _ := r.Context().Value("userId").(int)
	ipAddress := utils.GetClientIP(r)

	// Get count before deletion for logging
	allLogs, _, _ := h.auditRepo.GetAll(1, 10000)
	deletedCount := len(allLogs)

	// Delete all audit logs
	err := h.auditRepo.DeleteAll()
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to delete audit logs"))
		return
	}

	// Log the audit log deletion (clearing)
	details := map[string]interface{}{
		"deleted_records": deletedCount,
		"deleted_at":      time.Now().UTC(),
	}
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)
	h.auditSvc.LogAction("AUDIT_LOG_CLEARED", "AuditLog", 0, &userID, &detailsStr, &ipAddress)

	response := map[string]interface{}{
		"success": true,
		"message": "All audit logs have been deleted successfully",
	}
	writeJSON(w, http.StatusOK, response)
}
