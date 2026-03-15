package repository

import (
	"license-management-api/internal/models"
	"gorm.io/gorm"
)

// ILicenseActivationRepository defines license activation-specific operations
type ILicenseActivationRepository interface {
	IRepository[models.LicenseActivation]
	GetByLicenseID(licenseID int) ([]models.LicenseActivation, error)
	GetActiveByLicenseID(licenseID int) ([]models.LicenseActivation, error)
	GetByLicenseAndMachine(licenseID int, fingerprint string) (*models.LicenseActivation, error)
}

// LicenseActivationRepository implements ILicenseActivationRepository
type LicenseActivationRepository struct {
	*GenericRepository[models.LicenseActivation]
	db *gorm.DB
}

// NewLicenseActivationRepository creates a new instance
func NewLicenseActivationRepository(db *gorm.DB) ILicenseActivationRepository {
	return &LicenseActivationRepository{
		GenericRepository: &GenericRepository[models.LicenseActivation]{db: db},
		db:                db,
	}
}

// GetByLicenseID retrieves all activations for a license
func (r *LicenseActivationRepository) GetByLicenseID(licenseID int) ([]models.LicenseActivation, error) {
	var activations []models.LicenseActivation
	err := r.db.Where("license_id = ?", licenseID).Find(&activations).Error
	return activations, err
}

// GetActiveByLicenseID retrieves active activations for a license
func (r *LicenseActivationRepository) GetActiveByLicenseID(licenseID int) ([]models.LicenseActivation, error) {
	var activations []models.LicenseActivation
	err := r.db.Where("license_id = ? AND deactivated_at IS NULL", licenseID).Find(&activations).Error
	return activations, err
}

// GetByLicenseAndMachine retrieves activation by license and machine fingerprint
func (r *LicenseActivationRepository) GetByLicenseAndMachine(licenseID int, fingerprint string) (*models.LicenseActivation, error) {
	var activation models.LicenseActivation
	err := r.db.Where("license_id = ? AND machine_fingerprint = ?", licenseID, fingerprint).First(&activation).Error
	if err != nil {
		return nil, err
	}
	return &activation, nil
}
