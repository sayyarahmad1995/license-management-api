package utils

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"time"
)

// UserCSVRow represents a user in CSV format
type UserCSVRow struct {
	ID       int
	Username string
	Email    string
	Role     string
	Status   string
	CreatedAt string
	VerifiedAt *string
	LastLogin *string
}

// LicenseCSVRow represents a license in CSV format
type LicenseCSVRow struct {
	ID             int
	LicenseKey     string
	UserID         int
	Status         string
	CreatedAt      string
	ExpiresAt      string
	RevokedAt      *string
	MaxActivations int
	ActiveCount    int
}

// GenerateUserCSV generates a CSV from users
func GenerateUserCSV(users []UserCSVRow) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write header
	headers := []string{"ID", "Username", "Email", "Role", "Status", "Created At", "Verified At", "Last Login"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// Write data rows
	for _, user := range users {
		verifiedAt := ""
		if user.VerifiedAt != nil {
			verifiedAt = *user.VerifiedAt
		}
		lastLogin := ""
		if user.LastLogin != nil {
			lastLogin = *user.LastLogin
		}

		row := []string{
			fmt.Sprintf("%d", user.ID),
			user.Username,
			user.Email,
			user.Role,
			user.Status,
			user.CreatedAt,
			verifiedAt,
			lastLogin,
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// GenerateLicenseCSV generates a CSV from licenses
func GenerateLicenseCSV(licenses []LicenseCSVRow) ([]byte, error) {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)

	// Write header
	headers := []string{"ID", "License Key", "User ID", "Status", "Created At", "Expires At", "Revoked At", "Max Activations", "Active Activations"}
	if err := writer.Write(headers); err != nil {
		return nil, err
	}

	// Write data rows
	for _, license := range licenses {
		revokedAt := ""
		if license.RevokedAt != nil {
			revokedAt = *license.RevokedAt
		}

		row := []string{
			fmt.Sprintf("%d", license.ID),
			license.LicenseKey,
			fmt.Sprintf("%d", license.UserID),
			license.Status,
			license.CreatedAt,
			license.ExpiresAt,
			revokedAt,
			fmt.Sprintf("%d", license.MaxActivations),
			fmt.Sprintf("%d", license.ActiveCount),
		}
		if err := writer.Write(row); err != nil {
			return nil, err
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		return nil, err
	}

	return buffer.Bytes(), nil
}

// FormatTimestamp formats a time.Time for CSV export
func FormatTimestamp(t time.Time) string {
	return t.UTC().Format("2006-01-02 15:04:05")
}
