package repository

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

type IEmailVerificationRepository interface {
	Create(verification *models.EmailVerification) error
	GetByToken(token string) (*models.EmailVerification, error)
	GetByUserID(userID int) (*models.EmailVerification, error)
	Update(verification *models.EmailVerification) error
	Delete(id int) error
	DeleteByUserID(userID int) error
}

type emailVerificationRepository struct {
	db *gorm.DB
}

func NewEmailVerificationRepository(db *gorm.DB) IEmailVerificationRepository {
	return &emailVerificationRepository{db: db}
}

// Create creates a new email verification record
func (r *emailVerificationRepository) Create(verification *models.EmailVerification) error {
	return r.db.Create(verification).Error
}

// GetByToken retrieves verification by token
func (r *emailVerificationRepository) GetByToken(token string) (*models.EmailVerification, error) {
	var verification models.EmailVerification
	err := r.db.Where("token = ?", token).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// GetByUserID retrieves verification by user ID
func (r *emailVerificationRepository) GetByUserID(userID int) (*models.EmailVerification, error) {
	var verification models.EmailVerification
	err := r.db.Where("user_id = ? AND used_at IS NULL", userID).First(&verification).Error
	if err != nil {
		return nil, err
	}
	return &verification, nil
}

// Update updates a verification record
func (r *emailVerificationRepository) Update(verification *models.EmailVerification) error {
	return r.db.Save(verification).Error
}

// Delete deletes a verification record
func (r *emailVerificationRepository) Delete(id int) error {
	return r.db.Delete(&models.EmailVerification{}, id).Error
}

// DeleteByUserID deletes all verification records for a user
func (r *emailVerificationRepository) DeleteByUserID(userID int) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.EmailVerification{}).Error
}
