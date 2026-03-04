package models

import (
	"database/sql/driver"
	"encoding/json"
	"time"
)

type User struct {
	ID                         int       `gorm:"primaryKey"`
	Username                   string    `gorm:"column:username;index"`
	Email                      string    `gorm:"column:email;uniqueIndex"`
	PasswordHash               string    `gorm:"column:password_hash"`
	Role                       string    `gorm:"column:role;default:'User'"`
	Status                     string    `gorm:"column:status;default:'Unverified'"`
	CreatedAt                  time.Time `gorm:"column:created_at;default:CURRENT_TIMESTAMP"`
	VerifiedAt                 *time.Time `gorm:"column:verified_at"`
	UpdatedAt                  *time.Time `gorm:"column:updated_at"`
	LastLogin                  *time.Time `gorm:"column:last_login"`
	BlockedAt                  *time.Time `gorm:"column:blocked_at"`
	NotifyLicenseExpiry        bool      `gorm:"column:notify_license_expiry;default:true"`
	NotifyAccountActivity      bool      `gorm:"column:notify_account_activity;default:true"`
	NotifySystemAnnouncements  bool      `gorm:"column:notify_system_announcements;default:true"`
	Licenses                   []License `gorm:"foreignKey:UserID"`
}

// TableName specifies the table name for User model
func (User) TableName() string {
	return "users"
}

// Verify marks a user as verified
func (u *User) Verify() {
	if u.Status == string(UserStatusVerified) || u.Status == string(UserStatusActive) {
		return
	}
	now := time.Now().UTC()
	u.VerifiedAt = &now
	u.Status = string(UserStatusVerified)
	u.UpdatedAt = &now
}

// Activate activates a user account
func (u *User) Activate() {
	u.Status = string(UserStatusActive)
	now := time.Now().UTC()
	u.UpdatedAt = &now
}

// Block blocks a user account
func (u *User) Block() {
	if u.Status == string(UserStatusBlocked) {
		return
	}
	now := time.Now().UTC()
	u.BlockedAt = &now
	u.Status = string(UserStatusBlocked)
	u.UpdatedAt = &now
}

// Unblock unblocks a user account
func (u *User) Unblock() {
	if u.Status != string(UserStatusBlocked) {
		return
	}
	now := time.Now().UTC()
	u.BlockedAt = nil
	u.Status = string(UserStatusActive)
	u.UpdatedAt = &now
}

// Value implements the driver.Valuer interface
func (u User) Value() (driver.Value, error) {
	return json.Marshal(u)
}

// Scan implements the sql.Scanner interface
func (u *User) Scan(value interface{}) error {
	return json.Unmarshal(value.([]byte), &u)
}
