package models

import (
	"time"
)

type License struct {
	ID             int                  `gorm:"primaryKey"`
	LicenseKey     string               `gorm:"column:license_key;uniqueIndex"`
	UserID         int                  `gorm:"column:user_id;index"`
	User           *User                `gorm:"foreignKey:UserID"`
	Status         string               `gorm:"column:status;default:'Active'"`
	CreatedAt      time.Time            `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	ExpiresAt      time.Time            `gorm:"column:expires_at"`
	RevokedAt      *time.Time           `gorm:"column:revoked_at"`
	MaxActivations int                  `gorm:"column:max_activations;default:1"`
	Activations    []LicenseActivation  `gorm:"foreignKey:LicenseID"`
}

// TableName specifies the table name for License model
func (License) TableName() string {
	return "licenses"
}

// IsExpired checks if license is expired
func (l *License) IsExpired() bool {
	return time.Now().UTC().After(l.ExpiresAt)
}

// IsRevoked checks if license is revoked
func (l *License) IsRevoked() bool {
	return l.RevokedAt != nil
}

// CanActivate checks if license can be activated
func (l *License) CanActivate() bool {
	if l.IsExpired() || l.IsRevoked() {
		return false
	}
	if l.MaxActivations == 0 {
		return true // Unlimited activations
	}
	activeCount := 0
	for _, activation := range l.Activations {
		if activation.IsActive() {
			activeCount++
		}
	}
	return activeCount < l.MaxActivations
}
