package service

import (
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/repository"
)

// DashboardStatsService provides dashboard statistics and analytics
type DashboardStatsService interface {
	GetDashboardStats() (*dto.DashboardStats, error)
	GetLicenseExpirationForecast() (*dto.LicenseExpirationForecast, error)
	GetActivityTimeline(period string) (*dto.ActivityTimeline, error)
	GetUsageAnalytics() (*dto.UsageAnalytics, error)
	GetEnhancedDashboardStats() (*dto.EnhancedDashboardStats, error)
}

type dashboardStatsService struct {
	userRepo       repository.IUserRepository
	licenseRepo    repository.ILicenseRepository
	activationRepo repository.ILicenseActivationRepository
	auditRepo      repository.IAuditLogRepository
}

// NewDashboardStatsService creates a new dashboard stats service
func NewDashboardStatsService(
	userRepo repository.IUserRepository,
	licenseRepo repository.ILicenseRepository,
	auditRepo repository.IAuditLogRepository,
) DashboardStatsService {
	return &dashboardStatsService{
		userRepo:    userRepo,
		licenseRepo: licenseRepo,
		auditRepo:   auditRepo,
	}
}

// NewDashboardStatsServiceWithActivations creates service with activation repository
func NewDashboardStatsServiceWithActivations(
	userRepo repository.IUserRepository,
	licenseRepo repository.ILicenseRepository,
	activationRepo repository.ILicenseActivationRepository,
	auditRepo repository.IAuditLogRepository,
) DashboardStatsService {
	return &dashboardStatsService{
		userRepo:       userRepo,
		licenseRepo:    licenseRepo,
		activationRepo: activationRepo,
		auditRepo:      auditRepo,
	}
}

// GetDashboardStats retrieves basic dashboard statistics
func (dss *dashboardStatsService) GetDashboardStats() (*dto.DashboardStats, error) {
	stats := &dto.DashboardStats{
		Timestamp: time.Now().UTC(),
	}

	// Get users from repository
	users, _, _ := dss.userRepo.GetAll(1, 10000)
	stats.TotalUsers = int64(len(users))

	// Count verified users
	verifiedCount := int64(0)
	for _, user := range users {
		if user.Status == "Active" || user.Status == "Verified" {
			verifiedCount++
		}
	}
	stats.VerifiedUsers = verifiedCount
	stats.UnverifiedUsers = stats.TotalUsers - verifiedCount

	// Get licenses from repository
	licenses, _, _ := dss.licenseRepo.GetAll(1, 10000)
	stats.TotalLicenses = int64(len(licenses))

	// Count license statuses
	activeCount := int64(0)
	expiredCount := int64(0)
	revokedCount := int64(0)
	now := time.Now()

	for _, license := range licenses {
		if license.RevokedAt != nil && !license.RevokedAt.IsZero() {
			revokedCount++
		} else if license.ExpiresAt.Before(now) {
			expiredCount++
		} else if license.Status == "Active" {
			activeCount++
		}
	}
	stats.ActiveLicenses = activeCount
	stats.ExpiredLicenses = expiredCount
	stats.RevokedLicenses = revokedCount

	// Get recent audit logs
	auditLogs, _, _ := dss.auditRepo.GetAll(1, 100)
	stats.RecentAuditLogs = int64(len(auditLogs))

	// System health check
	stats.SystemHealth = "healthy"

	return stats, nil
}

// GetLicenseExpirationForecast returns upcoming license expirations
func (dss *dashboardStatsService) GetLicenseExpirationForecast() (*dto.LicenseExpirationForecast, error) {
	forecast := &dto.LicenseExpirationForecast{}

	now := time.Now().UTC()
	in7Days := now.AddDate(0, 0, 7)
	in30Days := now.AddDate(0, 0, 30)
	in90Days := now.AddDate(0, 0, 90)

	// Get all licenses
	licenses, _, _ := dss.licenseRepo.GetAll(1, 10000)

	for _, license := range licenses {
		// Skip already expired or revoked licenses
		if license.ExpiresAt.Before(now) {
			continue
		}
		if license.RevokedAt != nil && !license.RevokedAt.IsZero() {
			continue
		}

		// Count licenses expiring within different periods
		if license.ExpiresAt.Before(in7Days) {
			forecast.ExpiringIn7Days++
		}
		if license.ExpiresAt.Before(in30Days) {
			forecast.ExpiringIn30Days++
		}
		if license.ExpiresAt.Before(in90Days) {
			forecast.ExpiringIn90Days++
		}
	}

	forecast.TotalUpcoming = forecast.ExpiringIn90Days

	return forecast, nil
}

// GetActivityTimeline returns activity data over time
func (dss *dashboardStatsService) GetActivityTimeline(period string) (*dto.ActivityTimeline, error) {
	timeline := &dto.ActivityTimeline{
		Period:  period,
		Entries: make([]dto.ActivityTimelineEntry, 0),
	}

	var days int
	switch period {
	case "7d":
		days = 7
	case "30d":
		days = 30
	case "90d":
		days = 90
	default:
		days = 30
	}

	now := time.Now().UTC()

	// Get audit logs for the period
	auditLogs, _, _ := dss.auditRepo.GetAll(1, 10000)

	// Create entries for each day
	for i := days - 1; i >= 0; i-- {
		date := now.AddDate(0, 0, -i)
		dateStr := date.Format("2006-01-02")

		entry := dto.ActivityTimelineEntry{
			Date: dateStr,
		}

		// Count activities for this date
		for _, log := range auditLogs {
			if log.Timestamp.Format("2006-01-02") == dateStr {
				switch log.Action {
				case "LOGIN", "REFRESH_TOKEN":
					entry.Logins++
				case "CREATE_LICENSE":
					entry.LicenseCreated++
				case "REVOKE_LICENSE", "BULK_REVOKE_LICENSE", "BULK_REVOKE_LICENSE_ASYNC":
					entry.LicenseRevoked++
				case "ACTIVATE_LICENSE":
					entry.Activations++
				case "DEACTIVATE_LICENSE":
					entry.Deactivations++
				}
			}
		}

		timeline.Entries = append(timeline.Entries, entry)
	}

	return timeline, nil
}

// GetUsageAnalytics returns comprehensive usage metrics
func (dss *dashboardStatsService) GetUsageAnalytics() (*dto.UsageAnalytics, error) {
	analytics := &dto.UsageAnalytics{
		TopUsers: make([]dto.TopUserStat, 0),
	}

	// Get all licenses
	licenses, _, _ := dss.licenseRepo.GetAll(1, 10000)
	now := time.Now().UTC()

	activeCount := int64(0)
	expiredCount := int64(0)
	revokedCount := int64(0)
	totalActivations := int64(0)
	activeActivations := int64(0)

	// Count activations if activation repo available
	if dss.activationRepo != nil {
		activations, _, _ := dss.activationRepo.GetAll(1, 10000)
		totalActivations = int64(len(activations))

		for _, activation := range activations {
			if activation.DeactivatedAt == nil || activation.DeactivatedAt.IsZero() {
				activeActivations++
			}
		}
	}

	// Count license statuses
	for _, license := range licenses {
		if license.RevokedAt != nil && !license.RevokedAt.IsZero() {
			revokedCount++
		} else if license.ExpiresAt.Before(now) {
			expiredCount++
		} else {
			activeCount++
		}
	}

	analytics.TotalActivations = totalActivations
	analytics.ActiveActivations = activeActivations
	analytics.DeactivatedCount = totalActivations - activeActivations

	// Calculate average activations per license
	if len(licenses) > 0 {
		analytics.AvgActivationsPerLicense = float64(totalActivations) / float64(len(licenses))
	}

	// License distribution
	total := float64(activeCount + expiredCount + revokedCount)
	if total > 0 {
		analytics.LicenseDistribution = dto.LicenseDistribution{
			Active:     activeCount,
			Expired:    expiredCount,
			Revoked:    revokedCount,
			ActivePct:  float64(activeCount) / total * 100,
			ExpiredPct: float64(expiredCount) / total * 100,
			RevokedPct: float64(revokedCount) / total * 100,
		}
	}

	// Get top users by license count
	users, _, _ := dss.userRepo.GetAll(1, 10000)
	userLicenseCounts := make(map[int]int64)
	userActiveCounts := make(map[int]int64)

	for _, license := range licenses {
		userLicenseCounts[license.UserID]++
		if license.RevokedAt == nil && license.ExpiresAt.After(now) {
			userActiveCounts[license.UserID]++
		}
	}

	// Build top users list (top 10)
	for _, user := range users {
		if count, ok := userLicenseCounts[user.ID]; ok && count > 0 {
			analytics.TopUsers = append(analytics.TopUsers, dto.TopUserStat{
				UserID:       user.ID,
				Username:     user.Username,
				Email:        user.Email,
				LicenseCount: count,
				ActiveCount:  userActiveCounts[user.ID],
			})
		}
	}

	// Sort and limit to top 10
	// Simple bubble sort for top 10 (good enough for small datasets)
	for i := 0; i < len(analytics.TopUsers)-1 && i < 10; i++ {
		for j := i + 1; j < len(analytics.TopUsers); j++ {
			if analytics.TopUsers[j].LicenseCount > analytics.TopUsers[i].LicenseCount {
				analytics.TopUsers[i], analytics.TopUsers[j] = analytics.TopUsers[j], analytics.TopUsers[i]
			}
		}
	}

	// Limit to top 10
	if len(analytics.TopUsers) > 10 {
		analytics.TopUsers = analytics.TopUsers[:10]
	}

	return analytics, nil
}

// GetEnhancedDashboardStats returns all dashboard data in one call
func (dss *dashboardStatsService) GetEnhancedDashboardStats() (*dto.EnhancedDashboardStats, error) {
	enhanced := &dto.EnhancedDashboardStats{
		Timestamp: time.Now().UTC(),
	}

	// Get basic stats
	basicStats, err := dss.GetDashboardStats()
	if err != nil {
		return nil, err
	}
	enhanced.BasicStats = *basicStats

	// Get forecast
	forecast, err := dss.GetLicenseExpirationForecast()
	if err != nil {
		return nil, err
	}
	enhanced.Forecast = *forecast

	// Get activity timeline (30 days)
	timeline, err := dss.GetActivityTimeline("30d")
	if err != nil {
		return nil, err
	}
	enhanced.ActivityTimeline = *timeline

	// Get usage analytics
	usage, err := dss.GetUsageAnalytics()
	if err != nil {
		return nil, err
	}
	enhanced.UsageAnalytics = *usage

	return enhanced, nil
}
