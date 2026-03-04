package models

import "time"

// UserSession represents an active user session
type UserSession struct {
	ID             int64  `gorm:"primaryKey"`
	UserID         int    `gorm:"index"`
	SessionID      string `gorm:"index;unique"`
	IPAddress      string
	UserAgent      string
	DeviceInfo     string
	ExpiresAt      time.Time `gorm:"index"`
	LastActivityAt time.Time
	IsRevoked      bool
	CreatedAt      time.Time
	UpdatedAt      *time.Time
}

// TableName specifies the table name for UserSession
func (UserSession) TableName() string {
	return "user_sessions"
}

// IsExpired checks if the session has expired
func (us *UserSession) IsExpired() bool {
	return time.Now().UTC().After(us.ExpiresAt)
}

// IsActive checks if the session is active and not revoked
func (us *UserSession) IsActive() bool {
	return !us.IsExpired() && !us.IsRevoked
}

// UpdateActivity updates the last activity timestamp
func (us *UserSession) UpdateActivity() {
	now := time.Now().UTC()
	us.LastActivityAt = now
	us.UpdatedAt = &now
}

// Revoke marks the session as revoked
func (us *UserSession) Revoke() {
	us.IsRevoked = true
	now := time.Now().UTC()
	us.UpdatedAt = &now
}
