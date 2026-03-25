package dto

import "time"

// CreateLicenseDto for creating a license
type CreateLicenseDto struct {
	UserID         int       `json:"userId" validate:"required"`
	ExpiresAt      time.Time `json:"expiresAt" validate:"required"`
	MaxActivations int       `json:"maxActivations" validate:"min=0"`
}

// LicenseDto represents a license
type LicenseDto struct {
	ID             int                    `json:"id"`
	LicenseKey     string                 `json:"licenseKey"`
	UserID         int                    `json:"userId"`
	Status         string                 `json:"status"`
	CreatedAt      time.Time              `json:"createdAt"`
	ExpiresAt      time.Time              `json:"expiresAt"`
	RevokedAt      *time.Time             `json:"revokedAt,omitempty"`
	MaxActivations int                    `json:"maxActivations"`
	Activations    []LicenseActivationDto `json:"activations,omitempty"`
}

// LicenseActivationDto represents a license activation
type LicenseActivationDto struct {
	ID                 int        `json:"id"`
	LicenseID          int        `json:"licenseId"`
	MachineFingerprint string     `json:"machineFingerprint"`
	Hostname           *string    `json:"hostname,omitempty"`
	IpAddress          *string    `json:"ipAddress,omitempty"`
	ActivatedAt        time.Time  `json:"activatedAt"`
	DeactivatedAt      *time.Time `json:"deactivatedAt,omitempty"`
	LastSeenAt         time.Time  `json:"lastSeenAt"`
	IsActive           bool       `json:"isActive"`
}

// ActivateLicenseDto for activating a license
type ActivateLicenseDto struct {
	LicenseKey         string `json:"licenseKey" validate:"required"`
	MachineFingerprint string `json:"machineFingerprint" validate:"required"`
	Hostname           string `json:"hostname"`
	IpAddress          string `json:"ipAddress"`
}

// LicenseValidationDto for validating a license
type LicenseValidationDto struct {
	LicenseKey         string `json:"licenseKey" validate:"required"`
	MachineFingerprint string `json:"machineFingerprint" validate:"required"`
}

// LicenseValidationResultDto response for license validation
type LicenseValidationResultDto struct {
	IsValid   bool      `json:"isValid"`
	ExpiresAt time.Time `json:"expiresAt,omitempty"`
	Message   string    `json:"message,omitempty"`
}

// DeactivateLicenseDto for deactivating a license
type DeactivateLicenseDto struct {
	LicenseKey         string `json:"licenseKey" validate:"required"`
	MachineFingerprint string `json:"machineFingerprint" validate:"required"`
}

// RenewLicenseDto for renewing a license
type RenewLicenseDto struct {
	LicenseID int       `json:"licenseId" validate:"required"`
	ExpiresAt time.Time `json:"expiresAt" validate:"required"`
}

// BulkLicenseRevokeDto for bulk revoking licenses
type BulkLicenseRevokeDto struct {
	LicenseIds []int `json:"licenseIds" validate:"required,min=1"`
}

// BulkStatusUpdateDto for bulk updating license status
type BulkStatusUpdateDto struct {
	LicenseIds []int  `json:"licenseIds" validate:"required,min=1"`
	Status     string `json:"status" validate:"required,oneof=Active Revoked"`
}

// RevokeLicenseDto for revoking a license
type RevokeLicenseDto struct {
	LicenseID int `json:"licenseId" validate:"required"`
}

// UpdateLicenseStatusDto for updating license status
type UpdateLicenseStatusDto struct {
	Status string `json:"status" validate:"required,oneof=Active Revoked Suspended"`
}
