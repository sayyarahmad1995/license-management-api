package models

import (
	"time"
)

type LicenseActivation struct {
	ID                int        `gorm:"primaryKey"`
	LicenseID         int        `gorm:"column:license_id;index"`
	License           *License   `gorm:"foreignKey:LicenseID"`
	MachineFingerprint string    `gorm:"column:machine_fingerprint"`
	Hostname          *string    `gorm:"column:hostname"`
	IpAddress         *string    `gorm:"column:ip_address"`
	ActivatedAt       time.Time  `gorm:"column:activated_at;default:CURRENT_TIMESTAMP"`
	DeactivatedAt     *time.Time `gorm:"column:deactivated_at"`
	LastSeenAt        time.Time  `gorm:"column:last_seen_at;default:CURRENT_TIMESTAMP"`
}

// TableName specifies the table name for LicenseActivation model
func (LicenseActivation) TableName() string {
	return "license_activations"
}

// IsActive checks if the activation is still active
func (la *LicenseActivation) IsActive() bool {
	return la.DeactivatedAt == nil
}

// Deactivate deactivates the license activation
func (la *LicenseActivation) Deactivate() {
	now := time.Now().UTC()
	la.DeactivatedAt = &now
}

// UpdateLastSeen updates the last seen timestamp
func (la *LicenseActivation) UpdateLastSeen() {
	la.LastSeenAt = time.Now().UTC()
}
