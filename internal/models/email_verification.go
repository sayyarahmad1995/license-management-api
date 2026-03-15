package models

import "time"

// EmailVerification contains verification token data
type EmailVerification struct {
	ID        int        `gorm:"primaryKey"`
	UserID    int        `gorm:"column:user_id;index"`
	Token     string     `gorm:"column:token;uniqueIndex"`
	Email     string     `gorm:"column:email"`
	ExpiresAt time.Time  `gorm:"column:expires_at;index"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	CreatedAt time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	User      User       `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name
func (EmailVerification) TableName() string {
	return "email_verifications"
}

// IsExpired checks if token is expired
func (ev *EmailVerification) IsExpired() bool {
	return time.Now().UTC().After(ev.ExpiresAt)
}

// IsUsed checks if token was already used
func (ev *EmailVerification) IsUsed() bool {
	return ev.UsedAt != nil
}

// MarkAsUsed marks token as used
func (ev *EmailVerification) MarkAsUsed() {
	now := time.Now().UTC()
	ev.UsedAt = &now
}
