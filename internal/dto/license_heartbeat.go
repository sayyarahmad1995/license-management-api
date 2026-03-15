package dto

import "time"

// HeartbeatRequest represents a license heartbeat request
type HeartbeatRequest struct {
	LicenseKey         string `json:"license_key" binding:"required"`
	MachineFingerprint string `json:"machine_fingerprint" binding:"required"`
}

// HeartbeatResponse represents the heartbeat response
type HeartbeatResponse struct {
	Status   string    `json:"status"`
	Message  string    `json:"message"`
	LastSeen time.Time `json:"last_seen"`
}

// RevokeTokenRequest represents a token revocation request
type RevokeTokenRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// RevokeTokenResponse represents the token revocation response
type RevokeTokenResponse struct {
	Status  string `json:"status"`
	Message string `json:"message"`
}

// DashboardStats represents dashboard statistics
type DashboardStats struct {
	TotalUsers      int64     `json:"total_users"`
	TotalLicenses   int64     `json:"total_licenses"`
	ActiveLicenses  int64     `json:"active_licenses"`
	ExpiredLicenses int64     `json:"expired_licenses"`
	RevokedLicenses int64     `json:"revoked_licenses"`
	VerifiedUsers   int64     `json:"verified_users"`
	UnverifiedUsers int64     `json:"unverified_users"`
	RecentAuditLogs int64     `json:"recent_audit_logs"`
	SystemHealth    string    `json:"system_health"`
	Timestamp       time.Time `json:"timestamp"`
}

// LicenseExpirationForecast represents license expiration predictions
type LicenseExpirationForecast struct {
	ExpiringIn7Days  int64 `json:"expiring_in_7_days"`
	ExpiringIn30Days int64 `json:"expiring_in_30_days"`
	ExpiringIn90Days int64 `json:"expiring_in_90_days"`
	TotalUpcoming    int64 `json:"total_upcoming"`
}

// ActivityTimelineEntry represents a single activity data point
type ActivityTimelineEntry struct {
	Date           string `json:"date"`
	Logins         int64  `json:"logins"`
	LicenseCreated int64  `json:"license_created"`
	LicenseRevoked int64  `json:"license_revoked"`
	Activations    int64  `json:"activations"`
	Deactivations  int64  `json:"deactivations"`
}

// ActivityTimeline represents activity over time
type ActivityTimeline struct {
	Period  string                  `json:"period"` // "7d", "30d", "90d"
	Entries []ActivityTimelineEntry `json:"entries"`
}

// UsageAnalytics represents system usage metrics
type UsageAnalytics struct {
	TotalActivations         int64               `json:"total_activations"`
	ActiveActivations        int64               `json:"active_activations"`
	DeactivatedCount         int64               `json:"deactivated_count"`
	AvgActivationsPerLicense float64             `json:"avg_activations_per_license"`
	TopUsers                 []TopUserStat       `json:"top_users"`
	LicenseDistribution      LicenseDistribution `json:"license_distribution"`
}

// TopUserStat represents top user statistics
type TopUserStat struct {
	UserID       int    `json:"user_id"`
	Username     string `json:"username"`
	Email        string `json:"email"`
	LicenseCount int64  `json:"license_count"`
	ActiveCount  int64  `json:"active_count"`
}

// LicenseDistribution represents license status distribution
type LicenseDistribution struct {
	Active     int64   `json:"active"`
	Expired    int64   `json:"expired"`
	Revoked    int64   `json:"revoked"`
	ActivePct  float64 `json:"active_pct"`
	ExpiredPct float64 `json:"expired_pct"`
	RevokedPct float64 `json:"revoked_pct"`
}

// EnhancedDashboardStats represents comprehensive dashboard data
type EnhancedDashboardStats struct {
	BasicStats       DashboardStats            `json:"basic_stats"`
	Forecast         LicenseExpirationForecast `json:"forecast"`
	ActivityTimeline ActivityTimeline          `json:"activity_timeline"`
	UsageAnalytics   UsageAnalytics            `json:"usage_analytics"`
	Timestamp        time.Time                 `json:"timestamp"`
}
