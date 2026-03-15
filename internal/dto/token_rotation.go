package dto

import "time"

// TokenRotationResponse represents a token rotation response
type TokenRotationResponse struct {
	AccessToken      string    `json:"accessToken"`
	RefreshToken     string    `json:"refreshToken"`
	ExpiresAt        time.Time `json:"expiresAt"`
	RefreshExpiresAt time.Time `json:"refreshExpiresAt"`
}

// MachineFingerprint represents machine fingerprint data
type MachineFingerprint struct {
	Fingerprint   string    `json:"fingerprint" binding:"required"`
	MachineName   string    `json:"machine_name"`
	OSInfo        string    `json:"os_info"`
	LastActivated time.Time `json:"last_activated"`
	IsActive      bool      `json:"is_active"`
}

// MachineFingerprintResponse represents the response from machine fingerprint endpoint
type MachineFingerprintResponse struct {
	LicenseKey   string               `json:"license_key"`
	Fingerprints []MachineFingerprint `json:"fingerprints"`
	TotalCount   int64                `json:"total_count"`
}

// PaginationParams represents pagination parameters
type PaginationParams struct {
	Page     int64  `query:"page" json:"page"`
	PageSize int64  `query:"page_size" json:"page_size"`
	Search   string `query:"search" json:"search"`
	SortBy   string `query:"sort_by" json:"sort_by"`
	Sort     string `query:"sort" json:"sort"`
}

// PaginatedResponse represents a paginated response
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Page       int64       `json:"page"`
	PageSize   int64       `json:"page_size"`
	TotalCount int64       `json:"total_count"`
	TotalPages int64       `json:"total_pages"`
}
