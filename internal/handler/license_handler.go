package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"license-management-api/internal/cache"
	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/middleware"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/internal/service"
	"license-management-api/pkg/utils"

	"github.com/go-chi/chi/v5"
)

type LicenseHandler struct {
	licenseSvc            service.LicenseService
	licenseRepo           repository.ILicenseRepository
	machineFingerprintSvc service.MachineFingerprintService
	paginationSvc         service.PaginationService
	dataExportSvc         service.DataExportService
	bulkOperationSvc      service.BulkOperationService
	licenseCache          *cache.LicenseCache
	auditSvc              service.AuditService
}

func NewLicenseHandler(licenseSvc service.LicenseService, licenseRepo repository.ILicenseRepository, machineFingerprintSvc service.MachineFingerprintService, paginationSvc service.PaginationService, dataExportSvc service.DataExportService, bulkOperationSvc service.BulkOperationService, licenseCache *cache.LicenseCache, auditSvc service.AuditService) *LicenseHandler {
	return &LicenseHandler{
		licenseSvc:            licenseSvc,
		licenseRepo:           licenseRepo,
		machineFingerprintSvc: machineFingerprintSvc,
		paginationSvc:         paginationSvc,
		dataExportSvc:         dataExportSvc,
		bulkOperationSvc:      bulkOperationSvc,
		licenseCache:          licenseCache,
		auditSvc:              auditSvc,
	}
}

// GetLicenses retrieves all licenses (Admin only) with pagination and filtering
// @Summary Get all licenses
// @Description Retrieve a paginated list of all licenses (Admin only)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param pageIndex query int false "Page index (default 1)"
// @Param pageSize query int false "Page size (default 10, max 100)"
// @Param status query string false "Filter by license status"
// @Success 200 {object} map[string]interface{} "List of licenses"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /licenses [get]
// @Security BearerAuth
func (h *LicenseHandler) GetLicenses(w http.ResponseWriter, r *http.Request) {
	// Get user context
	currentUserID, _ := r.Context().Value("userId").(int)
	role, ok := r.Context().Value("userRole").(string)
	isAdmin := ok && role == string(models.UserRoleAdmin)

	// Parse pagination parameters
	pageIndex := 1
	pageSize := 10
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
	status := strings.TrimSpace(r.URL.Query().Get("status"))
	userIDStr := strings.TrimSpace(r.URL.Query().Get("user_id"))
	search := strings.TrimSpace(r.URL.Query().Get("search"))

	// Try to get licenses from cache
	var licenses []models.License
	var cacheHit bool
	licenses, cacheHit = h.licenseCache.Get(currentUserID, isAdmin)

	// If cache miss, fetch from database and cache
	if !cacheHit {
		var err error
		licenses, _, err = h.licenseRepo.GetAll(1, 10000)
		if err != nil {
			writeError(w, errors.NewInternalError("Failed to retrieve licenses"))
			return
		}
		h.licenseCache.Set(currentUserID, isAdmin, licenses)
	}

	// Apply filters
	var filtered []models.License

	for _, license := range licenses {
		// If user is NOT admin, only show their own licenses
		if !isAdmin && license.UserID != currentUserID {
			continue
		}

		// Filter by status if provided
		if status != "" && !strings.EqualFold(license.Status, status) {
			continue
		}

		// Filter by user_id if provided (admin only can filter by other user IDs)
		if userIDStr != "" {
			if userID, err := strconv.Atoi(userIDStr); err == nil {
				// Non-admin users cannot filter by other user IDs
				if !isAdmin && userID != currentUserID {
					continue
				}
				if license.UserID != userID {
					continue
				}
			}
		}

		// Filter by license key if search provided
		if search != "" && !strings.Contains(strings.ToLower(license.LicenseKey), strings.ToLower(search)) {
			continue
		}

		filtered = append(filtered, license)
	}

	// Get total count after filtering
	total := len(filtered)

	// Apply pagination
	offset := (pageIndex - 1) * pageSize
	if offset >= len(filtered) {
		filtered = []models.License{}
	} else if offset+pageSize > len(filtered) {
		filtered = filtered[offset:]
	} else {
		filtered = filtered[offset : offset+pageSize]
	}

	// Map to response DTOs
	licenseDtos := make([]map[string]interface{}, len(filtered))
	for i, license := range filtered {
		licenseDtos[i] = map[string]interface{}{
			"id":              license.ID,
			"licenseKey":      license.LicenseKey,
			"userId":          license.UserID,
			"status":          license.Status,
			"createdAt":       license.CreatedAt,
			"expiresAt":       license.ExpiresAt,
			"revokedAt":       license.RevokedAt,
			"maxActivations":  license.MaxActivations,
			"activationCount": len(license.Activations),
			"isExpired":       license.IsExpired(),
			"isRevoked":       license.IsRevoked(),
			"daysUntilExpiry": calculateDaysUntilExpiry(license.ExpiresAt),
		}
	}

	totalPages := (total + pageSize - 1) / pageSize

	response := map[string]interface{}{
		"pageIndex":  pageIndex,
		"pageSize":   pageSize,
		"total":      total,
		"totalPages": totalPages,
		"count":      len(licenseDtos),
		"licenses":   licenseDtos,
	}

	if status != "" {
		response["status"] = status
	}
	if userIDStr != "" {
		response["userId"] = userIDStr
	}
	if search != "" {
		response["search"] = search
	}

	writeJSON(w, http.StatusOK, response)
}

// GetLicense retrieves a specific license by ID
// @Summary Get license by ID
// @Description Retrieve a specific license with all its details
// @Tags Licenses
// @Accept json
// @Produce json
// @Param id path int true "License ID"
// @Success 200 {object} map[string]interface{} "License details"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /licenses/{id} [get]
// @Security BearerAuth
func (h *LicenseHandler) GetLicense(w http.ResponseWriter, r *http.Request) {
	// Get user context
	currentUserID, _ := r.Context().Value("userId").(int)
	role, _ := r.Context().Value("userRole").(string)

	// Parse license ID from URL
	licenseIDStr := chi.URLParam(r, "id")
	licenseID, err := strconv.Atoi(licenseIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid license ID"))
		return
	}

	license, err := h.licenseRepo.GetByID(licenseID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("License not found"))
		return
	}

	// Authorization: user can view own licenses, admin can view any
	if role != string(models.UserRoleAdmin) && license.UserID != currentUserID {
		writeError(w, errors.NewForbiddenError("You can only view your own licenses"))
		return
	}

	// Build response with activation details
	activationDtos := make([]map[string]interface{}, len(license.Activations))
	for i, activation := range license.Activations {
		activationDtos[i] = map[string]interface{}{
			"id":                 activation.ID,
			"machineFingerprint": activation.MachineFingerprint,
			"hostname":           activation.Hostname,
			"ipAddress":          activation.IpAddress,
			"activatedAt":        activation.ActivatedAt,
			"deactivatedAt":      activation.DeactivatedAt,
			"lastSeenAt":         activation.LastSeenAt,
			"isActive":           activation.IsActive(),
		}
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":              license.ID,
		"licenseKey":      license.LicenseKey,
		"userId":          license.UserID,
		"status":          license.Status,
		"createdAt":       license.CreatedAt,
		"expiresAt":       license.ExpiresAt,
		"revokedAt":       license.RevokedAt,
		"maxActivations":  license.MaxActivations,
		"activationCount": len(license.Activations),
		"activations":     activationDtos,
		"isExpired":       license.IsExpired(),
		"isRevoked":       license.IsRevoked(),
		"daysUntilExpiry": calculateDaysUntilExpiry(license.ExpiresAt),
	})
}

// Helper function to calculate days until expiry
func calculateDaysUntilExpiry(expiresAt time.Time) int {
	daysLeft := time.Until(expiresAt).Hours() / 24
	if daysLeft < 0 {
		return 0
	}
	return int(daysLeft)
}

// ExportLicenses exports all licenses as CSV (Admin only)
// @Summary Export all licenses as CSV
// @Description Download all licenses in CSV format (Admin only)
// @Tags Licenses
// @Accept json
// @Produce text/csv
// @Success 200 {file} file "CSV file"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /licenses/export [get]
// @Security BearerAuth
func (h *LicenseHandler) ExportLicenses(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// Get all licenses
	licenses, _, err := h.licenseRepo.GetAll(1, 10000)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve licenses"))
		return
	}

	// Convert to CSV format
	csvLicenses := make([]utils.LicenseCSVRow, len(licenses))
	for i, license := range licenses {
		revokedAt := (*string)(nil)
		if license.RevokedAt != nil {
			formatted := utils.FormatTimestamp(*license.RevokedAt)
			revokedAt = &formatted
		}

		// Count active activations
		activeCount := 0
		for _, activation := range license.Activations {
			if activation.IsActive() {
				activeCount++
			}
		}

		csvLicenses[i] = utils.LicenseCSVRow{
			ID:             license.ID,
			LicenseKey:     license.LicenseKey,
			UserID:         license.UserID,
			Status:         license.Status,
			CreatedAt:      utils.FormatTimestamp(license.CreatedAt),
			ExpiresAt:      utils.FormatTimestamp(license.ExpiresAt),
			RevokedAt:      revokedAt,
			MaxActivations: license.MaxActivations,
			ActiveCount:    activeCount,
		}
	}

	// Generate CSV
	csvData, err := utils.GenerateLicenseCSV(csvLicenses)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to generate CSV"))
		return
	}

	// Set response headers for file download
	w.Header().Set("Content-Type", "text/csv")
	w.Header().Set("Content-Disposition", "attachment; filename=\"licenses_"+time.Now().Format("20060102_150405")+".csv\"")
	w.Header().Set("Content-Length", string(rune(len(csvData))))

	w.WriteHeader(http.StatusOK)
	w.Write(csvData)
}

// CreateLicense creates a new license (Admin only)
// @Summary Create a new license
// @Description Create a new license for a user (Admin only)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.CreateLicenseDto true "License creation request"
// @Success 201 {object} map[string]interface{} "License created successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /licenses [post]
// @Security BearerAuth
func (h *LicenseHandler) CreateLicense(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateLicenseDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Get user ID from context (set by auth middleware)
	userID, ok := r.Context().Value("userId").(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	license, apiErr := h.licenseSvc.CreateLicense(&req, userID)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	// Invalidate cache when license is created
	h.licenseCache.InvalidateLicense(license)

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "License created successfully",
		"license": map[string]interface{}{
			"id":              license.ID,
			"licenseKey":      license.LicenseKey,
			"userId":          license.UserID,
			"status":          license.Status,
			"createdAt":       license.CreatedAt,
			"expiresAt":       license.ExpiresAt,
			"revokedAt":       license.RevokedAt,
			"maxActivations":  license.MaxActivations,
			"activationCount": len(license.Activations),
			"isExpired":       license.IsExpired(),
			"isRevoked":       license.IsRevoked(),
		},
	})
}

// ActivateLicense activates a license on a machine
// @Summary Activate a license
// @Description Activate a license on a specific machine using fingerprint
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.ActivateLicenseDto true "License activation request"
// @Success 200 {object} map[string]interface{} "License activated successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /licenses/activate [post]
func (h *LicenseHandler) ActivateLicense(w http.ResponseWriter, r *http.Request) {
	var req dto.ActivateLicenseDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	ipAddress := utils.GetClientIP(r)

	activation, apiErr := h.licenseSvc.ActivateLicense(&req, ipAddress)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	// Invalidate cache when license activation is created
	if license, err := h.licenseRepo.GetByLicenseKey(req.LicenseKey); err == nil && license != nil {
		h.licenseCache.InvalidateLicense(license)
	}

	writeJSON(w, http.StatusCreated, activation)
}

// ValidateLicense validates a license
// @Summary Validate a license
// @Description Check if a license is valid and not expired or revoked
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.LicenseValidationDto true "License validation request"
// @Success 200 {object} dto.LicenseValidationResultDto "Validation result"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Router /licenses/validate [post]
func (h *LicenseHandler) ValidateLicense(w http.ResponseWriter, r *http.Request) {
	var req dto.LicenseValidationDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	result, apiErr := h.licenseSvc.ValidateLicense(&req)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, result)
}

// DeactivateLicense deactivates a license
// @Summary Deactivate a license
// @Description Remove a license activation from a specific machine
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.DeactivateLicenseDto true "License deactivation request"
// @Success 200 {object} map[string]interface{} "License deactivated successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Router /licenses/deactivate [post]
func (h *LicenseHandler) DeactivateLicense(w http.ResponseWriter, r *http.Request) {
	var req dto.DeactivateLicenseDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	apiErr := h.licenseSvc.DeactivateLicense(&req)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	// Invalidate cache when license is deactivated
	if license, err := h.licenseRepo.GetByLicenseKey(req.LicenseKey); err == nil && license != nil {
		h.licenseCache.InvalidateLicense(license)
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "License deactivated successfully",
	})
}

// RenewLicense renews a license's expiration date (Admin only)
// @Summary Renew a license
// @Description Extend the expiration date of a license (Admin only)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param id path int true "License ID"
// @Param request body dto.RenewLicenseDto true "License renewal request"
// @Success 200 {object} map[string]interface{} "License renewed successfully"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /licenses/{id}/renew [post]
// @Security BearerAuth
func (h *LicenseHandler) RenewLicense(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	licenseIDStr := chi.URLParam(r, "id")
	licenseID, err := strconv.Atoi(licenseIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid license ID"))
		return
	}

	var req dto.RenewLicenseDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Get license
	license, err := h.licenseRepo.GetByID(licenseID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("License not found"))
		return
	}

	// Update expiry date
	license.ExpiresAt = req.ExpiresAt

	if updateErr := h.licenseRepo.Update(license); updateErr != nil {
		writeError(w, errors.NewInternalError("Failed to renew license"))
		return
	}

	// Invalidate cache when license is renewed
	h.licenseCache.InvalidateLicense(license)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "License renewed successfully",
		"license": map[string]interface{}{
			"id":              license.ID,
			"licenseKey":      license.LicenseKey,
			"userId":          license.UserID,
			"status":          license.Status,
			"createdAt":       license.CreatedAt,
			"expiresAt":       license.ExpiresAt,
			"revokedAt":       license.RevokedAt,
			"maxActivations":  license.MaxActivations,
			"activationCount": len(license.Activations),
			"isExpired":       license.IsExpired(),
			"isRevoked":       license.IsRevoked(),
		},
	})
}

// RevokeLicense revokes a license (Admin only)
// @Summary Revoke a license
// @Description Mark a license as revoked, preventing further use (Admin only)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param id path int true "License ID"
// @Success 200 {object} map[string]interface{} "License revoked successfully"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /licenses/{id} [delete]
// @Security BearerAuth
func (h *LicenseHandler) RevokeLicense(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	licenseIDStr := chi.URLParam(r, "id")
	licenseID, err := strconv.Atoi(licenseIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid license ID"))
		return
	}

	// Get license
	license, err := h.licenseRepo.GetByID(licenseID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("License not found"))
		return
	}

	// Check if already revoked
	if license.IsRevoked() {
		writeError(w, errors.NewBadRequestError("License is already revoked"))
		return
	}

	// Revoke the license
	license.Status = string(models.LicenseStatusRevoked)
	now := time.Now().UTC()
	license.RevokedAt = &now

	if updateErr := h.licenseRepo.Update(license); updateErr != nil {
		writeError(w, errors.NewInternalError("Failed to revoke license"))
		return
	}

	// Invalidate cache when license is revoked
	h.licenseCache.InvalidateLicense(license)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "License revoked successfully",
		"license": map[string]interface{}{
			"id":              license.ID,
			"licenseKey":      license.LicenseKey,
			"userId":          license.UserID,
			"status":          license.Status,
			"createdAt":       license.CreatedAt,
			"expiresAt":       license.ExpiresAt,
			"revokedAt":       license.RevokedAt,
			"maxActivations":  license.MaxActivations,
			"activationCount": len(license.Activations),
			"isExpired":       license.IsExpired(),
			"isRevoked":       license.IsRevoked(),
		},
	})
}

// UpdateLicenseStatus updates the status of a license (Admin only)
// @Summary Update license status
// @Description Change the status of a license (Admin only)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param id path int true "License ID"
// @Param request body dto.UpdateLicenseStatusDto true "Status update request"
// @Success 200 {object} map[string]interface{} "License status updated successfully"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /licenses/{id}/status [patch]
// @Security BearerAuth
func (h *LicenseHandler) UpdateLicenseStatus(w http.ResponseWriter, r *http.Request) {
	// Check admin access
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	licenseIDStr := chi.URLParam(r, "id")
	licenseID, err := strconv.Atoi(licenseIDStr)
	if err != nil {
		writeError(w, errors.NewBadRequestError("Invalid license ID"))
		return
	}

	var req dto.UpdateLicenseStatusDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate status value
	if req.Status != string(models.LicenseStatusActive) && req.Status != string(models.LicenseStatusRevoked) {
		writeError(w, errors.NewBadRequestError("Invalid status. Must be 'Active' or 'Revoked'"))
		return
	}

	// Get license
	license, err := h.licenseRepo.GetByID(licenseID)
	if err != nil {
		writeError(w, errors.NewNotFoundError("License not found"))
		return
	}

	oldStatus := license.Status // Capture old status for logging
	now := time.Now().UTC()

	// Update status based on value
	if req.Status == string(models.LicenseStatusRevoked) {
		license.Status = req.Status
		license.RevokedAt = &now
	} else if req.Status == string(models.LicenseStatusActive) {
		license.Status = req.Status
		license.RevokedAt = nil
	}

	if updateErr := h.licenseRepo.Update(license); updateErr != nil {
		writeError(w, errors.NewInternalError("Failed to update license status"))
		return
	}

	// Invalidate cache when license status is updated
	h.licenseCache.InvalidateLicense(license)

	// Log the license status change
	currentUserID, _ := r.Context().Value("userId").(int)
	details := map[string]interface{}{
		"old_status":  oldStatus,
		"new_status":  license.Status,
		"license_key": license.LicenseKey,
		"user_id":     license.UserID,
		"changed_at":  now,
	}
	detailsJSON, _ := json.Marshal(details)
	detailsStr := string(detailsJSON)
	ipAddress := utils.GetClientIP(r)
	h.auditSvc.LogAction("LICENSE_STATUS_CHANGED", "License", license.ID, &currentUserID, &detailsStr, &ipAddress)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "License status updated successfully",
		"license": license,
	})
}

// Heartbeat handles license heartbeat/ping
// @Summary License heartbeat
// @Description Record a heartbeat for an active license activation
// @Tags Licenses
// @Accept json
// @Produce json
// @Param heartbeatRequest body dto.HeartbeatRequest true "Heartbeat request"
// @Success 200 {object} dto.HeartbeatResponse
// @Failure 400 {object} errors.ApiError "Bad request"
// @Failure 404 {object} errors.ApiError "License or activation not found"
// @Failure 409 {object} errors.ApiError "License not valid"
// @Failure 500 {object} errors.ApiError "Internal server error"
// @Router /api/v1/licenses/heartbeat [post]
func (h *LicenseHandler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	var req dto.HeartbeatRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if req.LicenseKey == "" || req.MachineFingerprint == "" {
		writeError(w, errors.NewBadRequestError("license_key and machine_fingerprint are required"))
		return
	}

	// Record heartbeat
	if apiErr := h.licenseSvc.Heartbeat(req.LicenseKey, req.MachineFingerprint); apiErr != nil {
		writeError(w, apiErr)
		return
	}

	// Get the updated activation to return last_seen
	license, _ := h.licenseRepo.GetByLicenseKey(req.LicenseKey)
	if license != nil && len(license.Activations) > 0 {
		for _, activation := range license.Activations {
			if activation.MachineFingerprint == req.MachineFingerprint {
				writeJSON(w, http.StatusOK, &dto.HeartbeatResponse{
					Status:   "success",
					Message:  "Heartbeat recorded successfully",
					LastSeen: activation.LastSeenAt,
				})
				return
			}
		}
	}

	writeJSON(w, http.StatusOK, &dto.HeartbeatResponse{
		Status:   "success",
		Message:  "Heartbeat recorded successfully",
		LastSeen: time.Now().UTC(),
	})
}

// GetMachineFingerprints returns all machine fingerprints for a license
// @Summary Get machine fingerprints
// @Description Retrieve all machines that have activated a license
// @Tags Licenses
// @Accept json
// @Produce json
// @Param license_key query string true "License key"
// @Success 200 {object} dto.MachineFingerprintResponse "Machine fingerprints"
// @Failure 400 {object} errors.ApiError "Missing license key"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /api/v1/licenses/machines [get]
// @Security BearerAuth
func (h *LicenseHandler) GetMachineFingerprints(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("license_key")
	if licenseKey == "" {
		writeError(w, errors.NewBadRequestError("license_key parameter is required"))
		return
	}

	machines, err := h.machineFingerprintSvc.GetMachineFingerprints(licenseKey)
	if err != nil {
		writeError(w, err.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, &dto.MachineFingerprintResponse{
		LicenseKey:   licenseKey,
		Fingerprints: convertToFingerprints(machines),
		TotalCount:   int64(len(machines)),
	})
}

// TrackMachine registers a new machine with a license
// @Summary Track machine
// @Description Register a new machine or update existing machine info for a license
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.MachineFingerprint true "Machine info"
// @Success 201 {object} map[string]interface{} "Machine tracked"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 404 {object} errors.ApiError "License not found"
// @Router /api/v1/licenses/machines [post]
// @Security BearerAuth
func (h *LicenseHandler) TrackMachine(w http.ResponseWriter, r *http.Request) {
	licenseKey := r.URL.Query().Get("license_key")
	if licenseKey == "" {
		writeError(w, errors.NewBadRequestError("license_key parameter is required"))
		return
	}

	var machineInfo dto.MachineFingerprint
	if err := json.NewDecoder(r.Body).Decode(&machineInfo); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if machineInfo.Fingerprint == "" {
		writeError(w, errors.NewBadRequestError("fingerprint is required"))
		return
	}

	if err := h.machineFingerprintSvc.TrackMachine(licenseKey, machineInfo.Fingerprint, machineInfo.MachineName, machineInfo.OSInfo); err != nil {
		writeError(w, err.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message":     "Machine tracked successfully",
		"fingerprint": machineInfo.Fingerprint,
	})
}

// Helper function to convert license activations to fingerprint DTOs
func convertToFingerprints(activations []models.LicenseActivation) []dto.MachineFingerprint {
	var fingerprints []dto.MachineFingerprint
	for _, a := range activations {
		hostname := ""
		if a.Hostname != nil {
			hostname = *a.Hostname
		}
		ipAddress := ""
		if a.IpAddress != nil {
			ipAddress = *a.IpAddress
		}
		fingerprints = append(fingerprints, dto.MachineFingerprint{
			Fingerprint:   a.MachineFingerprint,
			MachineName:   hostname,
			OSInfo:        ipAddress,
			LastActivated: a.LastSeenAt,
			IsActive:      a.IsActive(),
		})
	}
	return fingerprints
}

// BulkRevoke revokes multiple licenses in a single operation
// @Summary Bulk revoke licenses
// @Description Revoke multiple licenses at once (Admin only)
// @Tags Licenses
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body dto.BulkRevokeRequest true "Bulk revoke request"
// @Success 200 {object} dto.BulkRevokeResponse "Revoke operation results"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /licenses/bulk-revoke [post]
func (h *LicenseHandler) BulkRevoke(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.BulkRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if len(req.LicenseIDs) == 0 {
		writeError(w, errors.NewValidationError("At least one license ID is required"))
		return
	}

	response, apiErr := h.bulkOperationSvc.BulkRevokeLicenses(req.LicenseIDs, req.Reason, userID)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, response)
}

// BulkRevokeAsync handles async bulk license revocation
// @Summary Bulk revoke licenses asynchronously
// @Description Revoke multiple licenses in background job (for large batches)
// @Tags Licenses
// @Accept json
// @Produce json
// @Param request body dto.BulkRevokeRequest true "Bulk revoke request"
// @Success 202 {object} dto.AsyncBulkRevokeResponse "Job started"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /licenses/bulk-revoke-async [post]
func (h *LicenseHandler) BulkRevokeAsync(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	userID, ok := r.Context().Value(middleware.UserIDContextKey).(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.BulkRevokeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if len(req.LicenseIDs) == 0 {
		writeError(w, errors.NewValidationError("At least one license ID is required"))
		return
	}

	response, apiErr := h.bulkOperationSvc.BulkRevokeLicensesAsync(req.LicenseIDs, req.Reason, userID)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusAccepted, response)
}

// GetBulkJobStatus retrieves the status of a bulk operation job
// @Summary Get bulk job status
// @Description Retrieve the current status and progress of a bulk operation job
// @Tags Licenses
// @Produce json
// @Param jobId path string true "Job ID"
// @Success 200 {object} dto.BulkJobStatus "Job status"
// @Failure 404 {object} errors.ApiError "Job not found"
// @Router /licenses/bulk-jobs/{jobId} [get]
func (h *LicenseHandler) GetBulkJobStatus(w http.ResponseWriter, r *http.Request) {
	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeError(w, errors.NewBadRequestError("Job ID is required"))
		return
	}

	status, apiErr := h.bulkOperationSvc.GetJobStatus(jobID)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, status)
}

// CancelBulkJob cancels a running bulk operation job
// @Summary Cancel bulk job
// @Description Cancel a pending or in-progress bulk operation job
// @Tags Licenses
// @Produce json
// @Param jobId path string true "Job ID"
// @Success 200 {object} map[string]interface{} "Job cancelled"
// @Failure 400 {object} errors.ApiError "Cannot cancel job"
// @Failure 404 {object} errors.ApiError "Job not found"
// @Router /licenses/bulk-jobs/{jobId}/cancel [post]
func (h *LicenseHandler) CancelBulkJob(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	jobID := chi.URLParam(r, "jobId")
	if jobID == "" {
		writeError(w, errors.NewBadRequestError("Job ID is required"))
		return
	}

	apiErr := h.bulkOperationSvc.CancelJob(jobID)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Job cancelled successfully",
		"job_id":  jobID,
	})
}
