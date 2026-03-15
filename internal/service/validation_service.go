package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"license-management-api/internal/errors"
)

// ValidationService handles input validation and sanitization
type ValidationService interface {
	ValidateEmail(email string) error
	ValidatePassword(password string) error
	ValidateUsername(username string) error
	ValidateLicenseKey(key string) error
	SanitizeInput(input string) string
	ValidateURL(url string) error
	ValidateIPAddress(ip string) error
}

// validationService implements ValidationService
type validationService struct {
	emailRegex    *regexp.Regexp
	licenseRegex  *regexp.Regexp
	urlRegex      *regexp.Regexp
	ipRegex       *regexp.Regexp
	usernameRegex *regexp.Regexp
}

// NewValidationService creates a new validation service
func NewValidationService() ValidationService {
	return &validationService{
		emailRegex:    regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`),
		licenseRegex:  regexp.MustCompile(`^[A-Z0-9]{8,32}$`),
		urlRegex:      regexp.MustCompile(`^https?://`),
		ipRegex:       regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`),
		usernameRegex: regexp.MustCompile(`^[a-zA-Z0-9_-]{3,50}$`),
	}
}

// ValidateEmail validates an email address
func (vs *validationService) ValidateEmail(email string) error {
	if email == "" {
		return errors.NewValidationError("Email is required")
	}

	email = strings.TrimSpace(email)
	if len(email) > 254 {
		return errors.NewValidationError("Email is too long (max 254 characters)")
	}

	if !vs.emailRegex.MatchString(email) {
		return errors.NewValidationError("Invalid email format")
	}

	return nil
}

// ValidatePassword validates a password
func (vs *validationService) ValidatePassword(password string) error {
	if password == "" {
		return errors.NewValidationError("Password is required")
	}

	if len(password) < 8 {
		return errors.NewValidationError("Password must be at least 8 characters long")
	}

	if len(password) > 128 {
		return errors.NewValidationError("Password is too long (max 128 characters)")
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	// Require uppercase, lowercase, digit, and special character
	if !hasUpper || !hasLower || !hasDigit || !hasSpecial {
		return errors.NewValidationError("Password must contain uppercase, lowercase, digit, and special character")
	}

	return nil
}

// ValidateUsername validates a username
func (vs *validationService) ValidateUsername(username string) error {
	if username == "" {
		return errors.NewValidationError("Username is required")
	}

	username = strings.TrimSpace(username)

	if len(username) < 3 {
		return errors.NewValidationError("Username must be at least 3 characters long")
	}

	if len(username) > 50 {
		return errors.NewValidationError("Username must not exceed 50 characters")
	}

	if !vs.usernameRegex.MatchString(username) {
		return errors.NewValidationError("Username can only contain letters, numbers, hyphens, and underscores")
	}

	return nil
}

// ValidateLicenseKey validates a license key format
func (vs *validationService) ValidateLicenseKey(key string) error {
	if key == "" {
		return errors.NewValidationError("License key is required")
	}

	key = strings.TrimSpace(strings.ToUpper(key))

	if len(key) < 8 || len(key) > 32 {
		return errors.NewValidationError("License key must be between 8 and 32 characters")
	}

	if !vs.licenseRegex.MatchString(key) {
		return errors.NewValidationError("License key must contain only alphanumeric characters")
	}

	return nil
}

// SanitizeInput removes potentially dangerous characters from input
func (vs *validationService) SanitizeInput(input string) string {
	if input == "" {
		return ""
	}

	// Trim whitespace
	sanitized := strings.TrimSpace(input)

	// Remove null bytes
	sanitized = strings.ReplaceAll(sanitized, "\x00", "")

	// Remove control characters except newline and tab
	sanitized = strings.Map(func(r rune) rune {
		if r < 32 && r != '\n' && r != '\t' {
			return -1
		}
		return r
	}, sanitized)

	// Limit length to 10000 characters
	if len(sanitized) > 10000 {
		sanitized = sanitized[:10000]
	}

	return sanitized
}

// ValidateURL validates a URL format
func (vs *validationService) ValidateURL(url string) error {
	if url == "" {
		return errors.NewValidationError("URL is required")
	}

	url = strings.TrimSpace(url)

	if len(url) > 2048 {
		return errors.NewValidationError("URL is too long (max 2048 characters)")
	}

	if !vs.urlRegex.MatchString(url) {
		return errors.NewValidationError("URL must start with http:// or https://")
	}

	return nil
}

// ValidateIPAddress validates an IP address format
func (vs *validationService) ValidateIPAddress(ip string) error {
	if ip == "" {
		return errors.NewValidationError("IP address is required")
	}

	ip = strings.TrimSpace(ip)

	// Basic IPv4 validation
	if !vs.ipRegex.MatchString(ip) {
		return errors.NewValidationError("Invalid IP address format")
	}

	// Verify octets are between 0 and 255
	octets := strings.Split(ip, ".")
	if len(octets) != 4 {
		return errors.NewValidationError("Invalid IP address: must have 4 octets")
	}

	for _, octet := range octets {
		var num int
		if _, err := fmt.Sscanf(octet, "%d", &num); err != nil {
			return errors.NewValidationError("Invalid IP address: non-numeric octet")
		}
		if num < 0 || num > 255 {
			return errors.NewValidationError("Invalid IP address: octets must be between 0 and 255")
		}
	}

	return nil
}
