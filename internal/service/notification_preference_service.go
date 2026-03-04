package service

import (
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/repository"
)

// NotificationPreferenceService handles user notification settings
type NotificationPreferenceService interface {
	GetPreferences(userID int) (*dto.NotificationPreferences, error)
	UpdatePreferences(userID int, prefs *dto.NotificationPreferences) error
}

type notificationPreferenceService struct {
	userRepo repository.IUserRepository
}

// NewNotificationPreferenceService creates a new notification preference service
func NewNotificationPreferenceService(userRepo repository.IUserRepository) NotificationPreferenceService {
	return &notificationPreferenceService{
		userRepo: userRepo,
	}
}

// GetPreferences retrieves notification preferences for a user
func (nps *notificationPreferenceService) GetPreferences(userID int) (*dto.NotificationPreferences, error) {
	user, err := nps.userRepo.GetByID(userID)
	if err != nil {
		return nil, errors.NewNotFoundError("User not found")
	}

	// Return default preferences if not set (we'll store these in the user model later)
	prefs := &dto.NotificationPreferences{
		UserID:                   user.ID,
		EmailOnLogin:             false,
		EmailOnLicenseExpiry:     true,
		EmailOnPasswordChange:    true,
		EmailOnSecurityAlert:     true,
		EmailOnLicenseActivation: true,
		EmailOnLicenseRevocation: true,
		UpdatedAt:                time.Now().UTC(),
	}

	return prefs, nil
}

// UpdatePreferences updates notification preferences for a user
func (nps *notificationPreferenceService) UpdatePreferences(userID int, prefs *dto.NotificationPreferences) error {
	user, err := nps.userRepo.GetByID(userID)
	if err != nil {
		return errors.NewNotFoundError("User not found")
	}

	// Store preferences (in a real system, these would be persisted to the database)
	// For now, we'll just validate the request
	_ = user
	_ = prefs

	return nil
}
