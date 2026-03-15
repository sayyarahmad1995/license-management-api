package repository

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

type IPasswordResetRepository interface {
	Create(reset *models.PasswordReset) error
	GetByToken(token string) (*models.PasswordReset, error)
	GetByUserID(userID int) (*models.PasswordReset, error)
	Update(reset *models.PasswordReset) error
	Delete(id int) error
	DeleteByUserID(userID int) error
}

type passwordResetRepository struct {
	db *gorm.DB
}

func NewPasswordResetRepository(db *gorm.DB) IPasswordResetRepository {
	return &passwordResetRepository{db: db}
}

// Create creates a new password reset record
func (r *passwordResetRepository) Create(reset *models.PasswordReset) error {
	return r.db.Create(reset).Error
}

// GetByToken retrieves reset by token
func (r *passwordResetRepository) GetByToken(token string) (*models.PasswordReset, error) {
	var reset models.PasswordReset
	err := r.db.Where("token = ?", token).First(&reset).Error
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

// GetByUserID retrieves reset by user ID
func (r *passwordResetRepository) GetByUserID(userID int) (*models.PasswordReset, error) {
	var reset models.PasswordReset
	err := r.db.Where("user_id = ? AND used_at IS NULL", userID).First(&reset).Error
	if err != nil {
		return nil, err
	}
	return &reset, nil
}

// Update updates a reset record
func (r *passwordResetRepository) Update(reset *models.PasswordReset) error {
	return r.db.Save(reset).Error
}

// Delete deletes a reset record
func (r *passwordResetRepository) Delete(id int) error {
	return r.db.Delete(&models.PasswordReset{}, id).Error
}

// DeleteByUserID deletes all reset records for a user
func (r *passwordResetRepository) DeleteByUserID(userID int) error {
	return r.db.Where("user_id = ?", userID).Delete(&models.PasswordReset{}).Error
}
