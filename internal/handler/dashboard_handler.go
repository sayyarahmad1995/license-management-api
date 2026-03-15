package handler

import (
	"net/http"
	"time"

	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/internal/service"
)

type DashboardHandler struct {
	userRepo          repository.IUserRepository
	licenseRepo       repository.ILicenseRepository
	activationRepo    repository.ILicenseActivationRepository
	auditRepo         repository.IAuditLogRepository
	dashboardStatsSvc service.DashboardStatsService
}

func NewDashboardHandler(
	userRepo repository.IUserRepository,
	licenseRepo repository.ILicenseRepository,
	activationRepo repository.ILicenseActivationRepository,
	auditRepo repository.IAuditLogRepository,
	dashboardStatsSvc service.DashboardStatsService,
) *DashboardHandler {
	return &DashboardHandler{
		userRepo:          userRepo,
		licenseRepo:       licenseRepo,
		activationRepo:    activationRepo,
		auditRepo:         auditRepo,
		dashboardStatsSvc: dashboardStatsSvc,
	}
}

// GetDashboardStats returns aggregated statistics (Admin only)
// @Summary Get dashboard statistics
// @Description Get system-wide dashboard statistics (Admin only)
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} dto.DashboardStats "Dashboard statistics"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /dashboard/stats [get]
// @Security BearerAuth
func (h *DashboardHandler) GetDashboardStats(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	// Get dashboard stats from service
	stats, err := h.dashboardStatsSvc.GetDashboardStats()
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch dashboard statistics"))
		return
	}

	writeJSON(w, http.StatusOK, stats)
}

// GetUserDashboard returns user-specific statistics
// @Summary Get user dashboard
// @Description Get dashboard statistics specific to the authenticated user
// @Tags Dashboard
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "User dashboard statistics"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Router /dashboard [get]
// @Security BearerAuth
func (h *DashboardHandler) GetUserDashboard(w http.ResponseWriter, r *http.Request) {
	userID, ok := r.Context().Value("userId").(int)
	if !ok {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	// Get user's licenses
	licenses, _, _ := h.licenseRepo.GetAll(1, 10000)
	userLicenses := []models.License{}
	for _, license := range licenses {
		if license.UserID == userID {
			userLicenses = append(userLicenses, license)
		}
	}

	activeLicenses := 0
	expiredLicenses := 0
	expiringLicenses := 0
	totalActivations := 0
	thirtyDaysFromNow := time.Now().AddDate(0, 0, 30)

	for _, license := range userLicenses {
		if license.Status == string(models.LicenseStatusActive) {
			activeLicenses++
			if license.ExpiresAt.Before(time.Now()) {
				expiredLicenses++
			} else if license.ExpiresAt.Before(thirtyDaysFromNow) {
				expiringLicenses++
			}
		}
	}

	// Get user's activations
	allActivations, _, _ := h.activationRepo.GetAll(1, 10000)
	for _, activation := range allActivations {
		if activation.License != nil && activation.License.UserID == userID {
			totalActivations++
		}
	}

	// Get user's recent activity
	allLogs, _, _ := h.auditRepo.GetAll(1, 10000)
	userLogs := []models.AuditLog{}
	for _, log := range allLogs {
		if log.UserID != nil && *log.UserID == userID {
			userLogs = append(userLogs, log)
		}
	}

	// Get last 5 logs
	recentLogs := []models.AuditLog{}
	if len(userLogs) > 5 {
		recentLogs = userLogs[len(userLogs)-5:]
	} else {
		recentLogs = userLogs
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"userId": userID,
		"licenses": map[string]interface{}{
			"total":            len(userLicenses),
			"active":           activeLicenses,
			"expired":          expiredLicenses,
			"expiringLikewise": expiringLicenses,
		},
		"activations": map[string]interface{}{
			"total": totalActivations,
		},
		"recentActivity": recentLogs,
	})
}

// GetLicenseExpirationForecast returns license expiration predictions
// @Summary Get license expiration forecast
// @Description Get upcoming license expirations (7, 30, 90 days) - Admin only
// @Tags Dashboard
// @Produce json
// @Success 200 {object} dto.LicenseExpirationForecast "Expiration forecast"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /dashboard/forecast [get]
// @Security BearerAuth
func (h *DashboardHandler) GetLicenseExpirationForecast(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	forecast, err := h.dashboardStatsSvc.GetLicenseExpirationForecast()
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch forecast"))
		return
	}

	writeJSON(w, http.StatusOK, forecast)
}

// GetActivityTimeline returns activity data over time
// @Summary Get activity timeline
// @Description Get activity timeline for specified period (7d, 30d, 90d) - Admin only
// @Tags Dashboard
// @Produce json
// @Param period query string false "Period (7d, 30d, 90d)" default(30d)
// @Success 200 {object} dto.ActivityTimeline "Activity timeline"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /dashboard/activity-timeline [get]
// @Security BearerAuth
func (h *DashboardHandler) GetActivityTimeline(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	period := r.URL.Query().Get("period")
	if period == "" {
		period = "30d"
	}

	// Validate period
	if period != "7d" && period != "30d" && period != "90d" {
		writeError(w, errors.NewBadRequestError("Invalid period. Must be 7d, 30d, or 90d"))
		return
	}

	timeline, err := h.dashboardStatsSvc.GetActivityTimeline(period)
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch activity timeline"))
		return
	}

	writeJSON(w, http.StatusOK, timeline)
}

// GetUsageAnalytics returns comprehensive usage metrics
// @Summary Get usage analytics
// @Description Get comprehensive system usage analytics - Admin only
// @Tags Dashboard
// @Produce json
// @Success 200 {object} dto.UsageAnalytics "Usage analytics"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /dashboard/analytics [get]
// @Security BearerAuth
func (h *DashboardHandler) GetUsageAnalytics(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	analytics, err := h.dashboardStatsSvc.GetUsageAnalytics()
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch usage analytics"))
		return
	}

	writeJSON(w, http.StatusOK, analytics)
}

// GetEnhancedDashboardStats returns all analytics in one call
// @Summary Get enhanced dashboard statistics
// @Description Get comprehensive dashboard with all analytics - Admin only
// @Tags Dashboard
// @Produce json
// @Success 200 {object} dto.EnhancedDashboardStats "Enhanced dashboard data"
// @Failure 401 {object} errors.ApiError "User not authenticated"
// @Failure 403 {object} errors.ApiError "Admin access required"
// @Router /dashboard/enhanced [get]
// @Security BearerAuth
func (h *DashboardHandler) GetEnhancedDashboardStats(w http.ResponseWriter, r *http.Request) {
	// Check if user is admin
	role, ok := r.Context().Value("userRole").(string)
	if !ok || role != string(models.UserRoleAdmin) {
		writeError(w, errors.NewForbiddenError("Admin access required"))
		return
	}

	enhanced, err := h.dashboardStatsSvc.GetEnhancedDashboardStats()
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to fetch enhanced dashboard"))
		return
	}

	writeJSON(w, http.StatusOK, enhanced)
}
