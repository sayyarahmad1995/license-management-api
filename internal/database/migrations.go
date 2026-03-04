package database

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

// MigrateEmailVerificationTable creates the email_verifications table
func MigrateEmailVerificationTable(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.EmailVerification{}) {
		return db.Migrator().CreateTable(&models.EmailVerification{})
	}
	return nil
}

// MigratePasswordResetTable creates the password_resets table
func MigratePasswordResetTable(db *gorm.DB) error {
	if !db.Migrator().HasTable(&models.PasswordReset{}) {
		return db.Migrator().CreateTable(&models.PasswordReset{})
	}
	return nil
}
