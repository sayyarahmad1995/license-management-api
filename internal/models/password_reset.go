package models

import "time"

// PasswordReset contains password reset token data
type PasswordReset struct {
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
func (PasswordReset) TableName() string {
	return "password_resets"
}

// IsExpired checks if token is expired
func (pr *PasswordReset) IsExpired() bool {
	return time.Now().UTC().After(pr.ExpiresAt)
}

// IsUsed checks if token was already used
func (pr *PasswordReset) IsUsed() bool {
	return pr.UsedAt != nil
}

// MarkAsUsed marks token as used
func (pr *PasswordReset) MarkAsUsed() {
	now := time.Now().UTC()
	pr.UsedAt = &now
}
