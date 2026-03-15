package dto

import "time"

// AuditLogDto represents an audit log entry
type AuditLogDto struct {
	ID         int       `json:"id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entityType"`
	EntityID   *int      `json:"entityId,omitempty"`
	UserID     *int      `json:"userId,omitempty"`
	Username   *string   `json:"username,omitempty"`
	Details    *string   `json:"details"`
	IpAddress  *string   `json:"ipAddress,omitempty"`
	Timestamp  time.Time `json:"timestamp"`
}

// DashboardStatsDto represents dashboard statistics
type DashboardStatsDto struct {
	TotalUsers            int `json:"totalUsers"`
	ActiveUsers           int `json:"activeUsers"`
	TotalLicenses         int `json:"totalLicenses"`
	ActiveLicenses        int `json:"activeLicenses"`
	ExpiredLicenses       int `json:"expiredLicenses"`
	RevokedLicenses       int `json:"revokedLicenses"`
	TotalActivations      int `json:"totalActivations"`
	ActiveActivations     int `json:"activeActivations"`
	LicensesExpiringIn30d int `json:"licensesExpiringIn30d"`
}

// PaginationDto for pagination
type PaginationDto struct {
	PageNumber int         `json:"pageNumber"`
	PageSize   int         `json:"pageSize"`
	Total      int         `json:"total"`
	TotalPages int         `json:"totalPages"`
	Data       interface{} `json:"data"`
}

// RenderedEmail represents a rendered email with subject and body
type RenderedEmail struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
}
