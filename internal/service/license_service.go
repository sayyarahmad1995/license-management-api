package service

import (
	"fmt"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/pkg/utils"
)

type LicenseService interface {
	CreateLicense(req *dto.CreateLicenseDto, userID int) (*models.License, *errors.ApiError)
	GetLicenseByKey(licenseKey string) (*models.License, *errors.ApiError)
	ActivateLicense(req *dto.ActivateLicenseDto, ipAddress string) (*dto.LicenseActivationDto, *errors.ApiError)
	ValidateLicense(req *dto.LicenseValidationDto) (*dto.LicenseValidationResultDto, *errors.ApiError)
	DeactivateLicense(req *dto.DeactivateLicenseDto) *errors.ApiError
	RenewLicense(req *dto.RenewLicenseDto) *errors.ApiError
	RevokeLicense(licenseID int) *errors.ApiError
	GetUserLicenses(userID int) ([]models.License, *errors.ApiError)
	Heartbeat(licenseKey string, machineFingerprint string) *errors.ApiError
}

type licenseService struct {
	licenseRepo    repository.ILicenseRepository
	activationRepo repository.ILicenseActivationRepository
	auditSvc       AuditService
}

func NewLicenseService(licenseRepo repository.ILicenseRepository, activationRepo repository.ILicenseActivationRepository, auditSvc AuditService) LicenseService {
	return &licenseService{
		licenseRepo:    licenseRepo,
		activationRepo: activationRepo,
		auditSvc:       auditSvc,
	}
}

// CreateLicense creates a new license
func (ls *licenseService) CreateLicense(req *dto.CreateLicenseDto, userID int) (*models.License, *errors.ApiError) {
	license := &models.License{
		LicenseKey:     utils.GenerateLicenseKey(),
		UserID:         req.UserID,
		Status:         string(models.LicenseStatusActive),
		ExpiresAt:      req.ExpiresAt,
		MaxActivations: req.MaxActivations,
		CreatedAt:      time.Now().UTC(),
	}

	if err := ls.licenseRepo.Create(license); err != nil {
		return nil, errors.NewInternalError("Failed to create license")
	}

	details := fmt.Sprintf("License created for user ID %d. Max activations: %d, Expires: %s", req.UserID, req.MaxActivations, req.ExpiresAt.Format("2006-01-02"))
	ls.auditSvc.LogAction("LICENSE_CREATED", "License", license.ID, &userID, &details, nil)

	return license, nil
}

// GetLicenseByKey retrieves a license by its key
func (ls *licenseService) GetLicenseByKey(licenseKey string) (*models.License, *errors.ApiError) {
	license, err := ls.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil || license == nil {
		return nil, errors.NewNotFoundError("License not found")
	}

	return license, nil
}

// ActivateLicense activates a license on a machine
func (ls *licenseService) ActivateLicense(req *dto.ActivateLicenseDto, ipAddress string) (*dto.LicenseActivationDto, *errors.ApiError) {
	license, apiErr := ls.GetLicenseByKey(req.LicenseKey)
	if apiErr != nil {
		return nil, apiErr
	}

	if !license.CanActivate() {
		return nil, errors.NewConflictError("License cannot be activated (expired, revoked, or max activations reached)")
	}

	activation := &models.LicenseActivation{
		LicenseID:          license.ID,
		MachineFingerprint: req.MachineFingerprint,
		Hostname:           &req.Hostname,
		IpAddress:          &ipAddress,
		ActivatedAt:        time.Now().UTC(),
		LastSeenAt:         time.Now().UTC(),
	}

	if err := ls.activationRepo.Create(activation); err != nil {
		return nil, errors.NewInternalError("Failed to activate license")
	}

	details := fmt.Sprintf("License activated on machine: %s (Fingerprint: %s). IP: %s", req.Hostname, req.MachineFingerprint, ipAddress)
	ls.auditSvc.LogAction("LICENSE_ACTIVATED", "License", license.ID, &license.UserID, &details, &ipAddress)

	return &dto.LicenseActivationDto{
		ID:                 activation.ID,
		LicenseID:          activation.LicenseID,
		MachineFingerprint: activation.MachineFingerprint,
		Hostname:           activation.Hostname,
		IpAddress:          activation.IpAddress,
		ActivatedAt:        activation.ActivatedAt,
		LastSeenAt:         activation.LastSeenAt,
		IsActive:           activation.IsActive(),
	}, nil
}

// ValidateLicense validates a license
func (ls *licenseService) ValidateLicense(req *dto.LicenseValidationDto) (*dto.LicenseValidationResultDto, *errors.ApiError) {
	license, apiErr := ls.GetLicenseByKey(req.LicenseKey)
	if apiErr != nil {
		return &dto.LicenseValidationResultDto{IsValid: false, Message: "License not found"}, nil
	}

	if license.IsExpired() {
		return &dto.LicenseValidationResultDto{IsValid: false, Message: "License expired"}, nil
	}

	if license.IsRevoked() {
		return &dto.LicenseValidationResultDto{IsValid: false, Message: "License revoked"}, nil
	}

	// Check activation
	activation, err := ls.activationRepo.GetByLicenseAndMachine(license.ID, req.MachineFingerprint)
	if err != nil || activation == nil || !activation.IsActive() {
		return &dto.LicenseValidationResultDto{IsValid: false, Message: "License not activated for this machine"}, nil
	}

	// Update last seen
	activation.UpdateLastSeen()
	_ = ls.activationRepo.Update(activation)

	return &dto.LicenseValidationResultDto{IsValid: true, ExpiresAt: license.ExpiresAt}, nil
}

// DeactivateLicense deactivates a license
func (ls *licenseService) DeactivateLicense(req *dto.DeactivateLicenseDto) *errors.ApiError {
	license, apiErr := ls.GetLicenseByKey(req.LicenseKey)
	if apiErr != nil {
		return apiErr
	}

	activation, err := ls.activationRepo.GetByLicenseAndMachine(license.ID, req.MachineFingerprint)
	if err != nil || activation == nil {
		return errors.NewNotFoundError("Activation not found")
	}

	activation.Deactivate()
	if err := ls.activationRepo.Update(activation); err != nil {
		return errors.NewInternalError("Failed to deactivate license")
	}

	details := fmt.Sprintf("License deactivated from machine with fingerprint: %s", req.MachineFingerprint)
	ls.auditSvc.LogAction("LICENSE_DEACTIVATED", "License", license.ID, &license.UserID, &details, nil)

	return nil
}

// RenewLicense renews a license
func (ls *licenseService) RenewLicense(req *dto.RenewLicenseDto) *errors.ApiError {
	license, err := ls.licenseRepo.GetByID(req.LicenseID)
	if err != nil || license == nil {
		return errors.NewNotFoundError("License not found")
	}

	license.ExpiresAt = req.ExpiresAt
	if err := ls.licenseRepo.Update(license); err != nil {
		return errors.NewInternalError("Failed to renew license")
	}

	details := fmt.Sprintf("License renewed. New expiry date: %s", req.ExpiresAt.Format("2006-01-02"))
	ls.auditSvc.LogAction("LICENSE_RENEWED", "License", license.ID, &license.UserID, &details, nil)

	return nil
}

// RevokeLicense revokes a license
func (ls *licenseService) RevokeLicense(licenseID int) *errors.ApiError {
	license, err := ls.licenseRepo.GetByID(licenseID)
	if err != nil || license == nil {
		return errors.NewNotFoundError("License not found")
	}

	now := time.Now().UTC()
	license.RevokedAt = &now
	license.Status = string(models.LicenseStatusRevoked)

	if err := ls.licenseRepo.Update(license); err != nil {
		return errors.NewInternalError("Failed to revoke license")
	}

	details := "License has been revoked and is no longer usable"
	ls.auditSvc.LogAction("LICENSE_REVOKED", "License", license.ID, &license.UserID, &details, nil)

	return nil
}

// GetUserLicenses retrieves all licenses for a user
func (ls *licenseService) GetUserLicenses(userID int) ([]models.License, *errors.ApiError) {
	licenses, err := ls.licenseRepo.GetByUserID(userID)
	if err != nil {
		return nil, errors.NewInternalError("Failed to retrieve licenses")
	}

	return licenses, nil
}

// Heartbeat records a heartbeat for a license activation
func (ls *licenseService) Heartbeat(licenseKey string, machineFingerprint string) *errors.ApiError {
	// Get license by key
	license, err := ls.licenseRepo.GetByLicenseKey(licenseKey)
	if err != nil {
		return errors.NewNotFoundError("License not found")
	}

	// Check if license is valid
	if license.IsExpired() {
		return errors.NewConflictError("License has expired")
	}

	if license.IsRevoked() {
		return errors.NewConflictError("License has been revoked")
	}

	// Find the activation with this machine fingerprint
	activation, err := ls.activationRepo.GetByLicenseAndMachine(license.ID, machineFingerprint)
	if err != nil {
		return errors.NewNotFoundError("License activation not found for this machine")
	}

	// Check if activation is still active
	if !activation.IsActive() {
		return errors.NewConflictError("License activation is no longer active")
	}

	// Update LastSeenAt
	activation.UpdateLastSeen()
	if err := ls.activationRepo.Update(activation); err != nil {
		return errors.NewInternalError("Failed to update heartbeat")
	}

	// Log audit event
	details := fmt.Sprintf("License heartbeat from machine: %s\n", machineFingerprint)
	ls.auditSvc.LogAction("LICENSE_HEARTBEAT", "License", license.ID, &license.UserID, &details, nil)

	return nil
}
