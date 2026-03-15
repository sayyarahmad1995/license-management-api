package dto

import "time"

// NotificationPreferences represents user notification settings
type NotificationPreferences struct {
	UserID                   int       `json:"user_id"`
	EmailOnLogin             bool      `json:"email_on_login"`
	EmailOnLicenseExpiry     bool      `json:"email_on_license_expiry"`
	EmailOnPasswordChange    bool      `json:"email_on_password_change"`
	EmailOnSecurityAlert     bool      `json:"email_on_security_alert"`
	EmailOnLicenseActivation bool      `json:"email_on_license_activation"`
	EmailOnLicenseRevocation bool      `json:"email_on_license_revocation"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// UpdateProfileRequest represents a profile update request
type UpdateProfileRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
}

// UpdateProfileResponse represents the response after profile update
type UpdateProfileResponse struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	UpdatedAt time.Time `json:"updated_at"`
}

// GetMeResponse represents the logged-in user's profile
type GetMeResponse struct {
	ID        int        `json:"id"`
	Username  string     `json:"username"`
	Email     string     `json:"email"`
	Role      string     `json:"role"`
	Status    string     `json:"status"`
	LastLogin *time.Time `json:"last_login"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// ExportDataRequest represents a data export request
type ExportDataRequest struct {
	DataType string                 `json:"data_type" binding:"required"` // "users" or "licenses"
	Format   string                 `json:"format" binding:"required"`    // "csv"
	Filters  map[string]interface{} `json:"filters,omitempty"`
}

// ExportDataResponse represents the response with export data
type ExportDataResponse struct {
	DataType  string    `json:"data_type"`
	Filename  string    `json:"filename"`
	MimeType  string    `json:"mime_type"`
	Data      []byte    `json:"data"`
	Timestamp time.Time `json:"timestamp"`
}

// BulkRevokeRequest represents a bulk revoke operation
type BulkRevokeRequest struct {
	LicenseIDs []int  `json:"license_ids" binding:"required,min=1"`
	Reason     string `json:"reason"`
}

// BulkRevokeResponse represents the result of bulk revoke
type BulkRevokeResponse struct {
	TotalRequested int       `json:"total_requested"`
	SuccessCount   int       `json:"success_count"`
	FailureCount   int       `json:"failure_count"`
	FailedIDs      []int     `json:"failed_ids,omitempty"`
	CompletedAt    time.Time `json:"completed_at"`
}

// BulkJobStatus represents the status of an async bulk operation
type BulkJobStatus struct {
	JobID          string     `json:"job_id"`
	Status         string     `json:"status"` // pending, in_progress, completed, failed
	TotalItems     int        `json:"total_items"`
	ProcessedItems int        `json:"processed_items"`
	SuccessCount   int        `json:"success_count"`
	FailureCount   int        `json:"failure_count"`
	FailedIDs      []int      `json:"failed_ids,omitempty"`
	ErrorMessage   string     `json:"error_message,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
	CompletedAt    *time.Time `json:"completed_at,omitempty"`
	ProgressPct    float64    `json:"progress_pct"`
}

// AsyncBulkRevokeResponse represents the immediate response for async bulk operations
type AsyncBulkRevokeResponse struct {
	JobID      string    `json:"job_id"`
	Status     string    `json:"status"`
	Message    string    `json:"message"`
	TotalItems int       `json:"total_items"`
	CreatedAt  time.Time `json:"created_at"`
}
