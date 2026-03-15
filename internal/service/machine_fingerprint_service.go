package service

import (
	"fmt"
	"time"

	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

// MachineFingerprintService handles machine fingerprint tracking
type MachineFingerprintService interface {
	TrackMachine(licenseKey string, fingerprint string, machineName string, osInfo string) error
	GetMachineFingerprints(licenseKey string) ([]models.LicenseActivation, error)
	UpdateMachineActivity(licenseKey string, fingerprint string) error
	DeactivateMachine(licenseKey string, fingerprint string) error
	GetActiveMachines(licenseKey string) ([]models.LicenseActivation, error)
	IsDeviceBlacklisted(fingerprint string) bool
}

type machineFingerprintService struct {
	licenseRepo    repository.ILicenseRepository
	activationRepo repository.ILicenseActivationRepository
	auditSvc       AuditService
	blacklist      map[string]bool // In-memory blacklist (could be Redis in production)
}

// NewMachineFingerprintService creates a new machine fingerprint service
func NewMachineFingerprintService(
	licenseRepo repository.ILicenseRepository,
	activationRepo repository.ILicenseActivationRepository,
	auditSvc AuditService,
) MachineFingerprintService {
	return &machineFingerprintService{
		licenseRepo:    licenseRepo,
		activationRepo: activationRepo,
		auditSvc:       auditSvc,
		blacklist:      make(map[string]bool),
	}
}

// TrackMachine records a new machine with a fingerprint
func (mfs *machineFingerprintService) TrackMachine(licenseKey string, fingerprint string, machineName string, osInfo string) error {
	// Get license by key
	license, err := mfs.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return errors.NewNotFoundError("License not found")
	}

	// Check if machine is blacklisted
	if mfs.IsDeviceBlacklisted(fingerprint) {
		return errors.NewConflictError("Device has been blacklisted and cannot activate licenses")
	}

	// Check if this fingerprint already exists for this license
	existing, _ := mfs.activationRepo.GetByLicenseAndMachine(license.ID, fingerprint)
	if existing != nil {
		// Update existing activation
		existing.LastSeenAt = time.Now().UTC()
		existing.Hostname = &machineName
		existing.IpAddress = &osInfo
		return mfs.activationRepo.Update(existing)
	}

	// Create new activation for this machine
	activation := &models.LicenseActivation{
		LicenseID:          license.ID,
		MachineFingerprint: fingerprint,
		Hostname:           &machineName,
		IpAddress:          &osInfo,
		ActivatedAt:        time.Now().UTC(),
		LastSeenAt:         time.Now().UTC(),
		DeactivatedAt:      nil,
	}

	err = mfs.activationRepo.Create(activation)
	if err != nil {
		return errors.NewInternalError("Failed to track machine")
	}

	// Log audit event
	details := fmt.Sprintf("Machine tracked - Name: %s, Fingerprint: %s", machineName, fingerprint)
	mfs.auditSvc.LogAction("MACHINE_TRACKED", "LicenseActivation", activation.ID, &license.UserID, &details, nil)

	return nil
}

// GetMachineFingerprints retrieves all machines for a license
func (mfs *machineFingerprintService) GetMachineFingerprints(licenseKey string) ([]models.LicenseActivation, error) {
	license, err := mfs.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.NewNotFoundError("License not found")
	}

	activations, err := mfs.activationRepo.GetByLicenseID(license.ID)
	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve machine fingerprints")
	}

	return activations, nil
}

// UpdateMachineActivity updates the last seen time for a machine
func (mfs *machineFingerprintService) UpdateMachineActivity(licenseKey string, fingerprint string) error {
	license, err := mfs.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return errors.NewNotFoundError("License not found")
	}

	activation, err := mfs.activationRepo.GetByLicenseAndMachine(license.ID, fingerprint)
	if err != nil {
		return errors.NewNotFoundError("Machine activation not found")
	}

	activation.LastSeenAt = time.Now().UTC()
	err = mfs.activationRepo.Update(activation)
	if err != nil {
		return errors.NewInternalError("Failed to update machine activity")
	}

	return nil
}

// DeactivateMachine deactivates a machine for a license
func (mfs *machineFingerprintService) DeactivateMachine(licenseKey string, fingerprint string) error {
	license, err := mfs.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return errors.NewNotFoundError("License not found")
	}

	activation, err := mfs.activationRepo.GetByLicenseAndMachine(license.ID, fingerprint)
	if err != nil {
		return errors.NewNotFoundError("Machine activation not found")
	}

	now := time.Now().UTC()
	activation.DeactivatedAt = &now
	err = mfs.activationRepo.Update(activation)
	if err != nil {
		return errors.NewInternalError("Failed to deactivate machine")
	}

	// Log audit event
	details := fmt.Sprintf("Machine deactivated - Fingerprint: %s", fingerprint)
	mfs.auditSvc.LogAction("MACHINE_DEACTIVATED", "LicenseActivation", activation.ID, &license.UserID, &details, nil)

	return nil
}

// GetActiveMachines retrieves only active (non-deactivated) machines for a license
func (mfs *machineFingerprintService) GetActiveMachines(licenseKey string) ([]models.LicenseActivation, error) {
	license, err := mfs.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return nil, errors.NewNotFoundError("License not found")
	}

	activations, err := mfs.activationRepo.GetByLicenseID(license.ID)
	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve active machines")
	}

	// Filter only active machines
	var activeMachines []models.LicenseActivation
	for _, a := range activations {
		if a.IsActive() {
			activeMachines = append(activeMachines, a)
		}
	}

	return activeMachines, nil
}

// IsDeviceBlacklisted checks if a device is blacklisted
func (mfs *machineFingerprintService) IsDeviceBlacklisted(fingerprint string) bool {
	return mfs.blacklist[fingerprint]
}

// BlacklistDevice adds a device to the blacklist
func (mfs *machineFingerprintService) BlacklistDevice(fingerprint string) {
	mfs.blacklist[fingerprint] = true
	// In production, this should be persisted to Redis or database
}

// UnblacklistDevice removes a device from the blacklist
func (mfs *machineFingerprintService) UnblacklistDevice(fingerprint string) {
	delete(mfs.blacklist, fingerprint)
	// In production, this should be persisted to Redis or database
}
