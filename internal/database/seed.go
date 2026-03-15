package database

import (
	"fmt"
	"log"
	"time"

	"license-management-api/internal/models"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// SeedAdminUser creates a default admin user if one doesn't exist
func SeedAdminUser(db *gorm.DB, email, username, password string) error {
	// Check if admin user already exists
	var existingUser models.User
	result := db.Where("email = ? OR username = ?", email, username).First(&existingUser)

	if result.Error == nil {
		// User already exists
		log.Printf("Admin user already exists: %s", email)
		return nil
	}

	if result.Error != gorm.ErrRecordNotFound {
		// Some other error occurred
		return fmt.Errorf("error checking for existing admin user: %w", result.Error)
	}

	// Hash the password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("error hashing password: %w", err)
	}

	// Create admin user
	now := time.Now().UTC()
	adminUser := models.User{
		Username:                  username,
		Email:                     email,
		PasswordHash:              string(hashedPassword),
		Role:                      string(models.UserRoleAdmin),
		Status:                    string(models.UserStatusActive),
		CreatedAt:                 now,
		VerifiedAt:                &now,
		NotifyLicenseExpiry:       true,
		NotifyAccountActivity:     true,
		NotifySystemAnnouncements: true,
	}

	if err := db.Create(&adminUser).Error; err != nil {
		return fmt.Errorf("error creating admin user: %w", err)
	}

	log.Printf("Admin user created successfully: %s (%s)", adminUser.Username, adminUser.Email)
	return nil
}
