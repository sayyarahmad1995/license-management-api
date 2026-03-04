package repository

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

// ILicenseRepository defines license-specific repository operations
type ILicenseRepository interface {
	IRepository[models.License]
	GetByLicenseKey(licenseKey string) (*models.License, error)
	GetByUserID(userID int) ([]models.License, error)
	GetActiveByUserID(userID int) ([]models.License, error)
}

// LicenseRepository implements ILicenseRepository
type LicenseRepository struct {
	*GenericRepository[models.License]
	db *gorm.DB
}

// NewLicenseRepository creates a new instance of LicenseRepository
func NewLicenseRepository(db *gorm.DB) ILicenseRepository {
	return &LicenseRepository{
		GenericRepository: &GenericRepository[models.License]{db: db},
		db:                db,
	}
}

// GetByLicenseKey retrieves a license by license key
func (r *LicenseRepository) GetByLicenseKey(licenseKey string) (*models.License, error) {
	var license models.License
	err := r.db.Where("license_key = ?", licenseKey).First(&license).Error
	if err != nil {
		return nil, err
	}
	return &license, nil
}

// GetByUserID retrieves all licenses for a user
func (r *LicenseRepository) GetByUserID(userID int) ([]models.License, error) {
	var licenses []models.License
	err := r.db.Where("user_id = ?", userID).Find(&licenses).Error
	return licenses, err
}

// GetActiveByUserID retrieves active licenses for a user
func (r *LicenseRepository) GetActiveByUserID(userID int) ([]models.License, error) {
	var licenses []models.License
	err := r.db.Where("user_id = ? AND status = ?", userID, models.LicenseStatusActive).Find(&licenses).Error
	return licenses, err
}
