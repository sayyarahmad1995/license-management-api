package service

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"

	"license-management-api/internal/repository"
)

// DataExportService handles CSV export of data
type DataExportService interface {
	ExportUsers(pageSize int) ([]byte, error)
	ExportLicenses(pageSize int) ([]byte, error)
	ExportLicenseActivations(pageSize int) ([]byte, error)
}

type dataExportService struct {
	userRepo              repository.IUserRepository
	licenseRepo           repository.ILicenseRepository
	licenseActivationRepo repository.ILicenseActivationRepository
}

// NewDataExportService creates a new data export service
func NewDataExportService(
	userRepo repository.IUserRepository,
	licenseRepo repository.ILicenseRepository,
	licenseActivationRepo repository.ILicenseActivationRepository,
) DataExportService {
	return &dataExportService{
		userRepo:              userRepo,
		licenseRepo:           licenseRepo,
		licenseActivationRepo: licenseActivationRepo,
	}
}

// ExportUsers exports all users to CSV format
func (des *dataExportService) ExportUsers(pageSize int) ([]byte, error) {
	// Get all users
	users, _, err := des.userRepo.GetAll(1, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch users: %w", err)
	}

	// Create CSV buffer
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	defer writer.Flush()

	// Write headers
	headers := []string{"ID", "Username", "Email", "Role", "Status", "CreatedAt", "VerifiedAt"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write user records
	for _, user := range users {
		verifiedAt := ""
		if user.VerifiedAt != nil {
			verifiedAt = user.VerifiedAt.Format(time.RFC3339)
		}

		record := []string{
			fmt.Sprintf("%d", user.ID),
			user.Username,
			user.Email,
			user.Role,
			user.Status,
			user.CreatedAt.Format(time.RFC3339),
			verifiedAt,
		}

		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write user record: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// ExportLicenses exports all licenses to CSV format
func (des *dataExportService) ExportLicenses(pageSize int) ([]byte, error) {
	// Get all licenses
	licenses, _, err := des.licenseRepo.GetAll(1, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch licenses: %w", err)
	}

	// Create CSV buffer
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	defer writer.Flush()

	// Write headers
	headers := []string{"ID", "LicenseKey", "UserID", "Status", "CreatedAt", "ExpiresAt", "RevokedAt"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write license records
	for _, license := range licenses {
		revokedAt := ""
		if license.RevokedAt != nil {
			revokedAt = license.RevokedAt.Format(time.RFC3339)
		}

		record := []string{
			fmt.Sprintf("%d", license.ID),
			license.LicenseKey,
			fmt.Sprintf("%d", license.UserID),
			license.Status,
			license.CreatedAt.Format(time.RFC3339),
			license.ExpiresAt.Format(time.RFC3339),
			revokedAt,
		}

		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write license record: %w", err)
		}
	}

	return buf.Bytes(), nil
}

// ExportLicenseActivations exports all license activations to CSV format
func (des *dataExportService) ExportLicenseActivations(pageSize int) ([]byte, error) {
	// Get all activations
	activations, _, err := des.licenseActivationRepo.GetAll(1, pageSize)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch activations: %w", err)
	}

	// Create CSV buffer
	buf := &bytes.Buffer{}
	writer := csv.NewWriter(buf)
	defer writer.Flush()

	// Write headers
	headers := []string{"ID", "LicenseID", "MachineFingerprint", "Hostname", "ActivatedAt", "DeactivatedAt", "LastSeenAt"}
	if err := writer.Write(headers); err != nil {
		return nil, fmt.Errorf("failed to write CSV headers: %w", err)
	}

	// Write activation records
	for _, activation := range activations {
		hostname := ""
		if activation.Hostname != nil {
			hostname = *activation.Hostname
		}
		deactivatedAt := ""
		if activation.DeactivatedAt != nil {
			deactivatedAt = activation.DeactivatedAt.Format(time.RFC3339)
		}

		record := []string{
			fmt.Sprintf("%d", activation.ID),
			fmt.Sprintf("%d", activation.LicenseID),
			activation.MachineFingerprint,
			hostname,
			activation.ActivatedAt.Format(time.RFC3339),
			deactivatedAt,
			activation.LastSeenAt.Format(time.RFC3339),
		}

		if err := writer.Write(record); err != nil {
			return nil, fmt.Errorf("failed to write activation record: %w", err)
		}
	}

	return buf.Bytes(), nil
}
