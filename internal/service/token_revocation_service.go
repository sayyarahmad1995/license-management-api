package service

import (
	"context"
	"time"

	"license-management-api/internal/errors"
	"license-management-api/internal/repository"

	"github.com/redis/go-redis/v9"
)

const RevokedTokenPrefix = "revoked_token:"

// TokenRevocationService handles token revocation
type TokenRevocationService interface {
	RevokeToken(token string, expiresAt time.Time) error
	IsTokenRevoked(token string) (bool, error)
}

type tokenRevocationService struct {
	redisClient *redis.Client
}

// NewTokenRevocationService creates a new token revocation service
func NewTokenRevocationService(redisClient *redis.Client) TokenRevocationService {
	return &tokenRevocationService{
		redisClient: redisClient,
	}
}

// RevokeToken adds a token to the revocation list
func (trs *tokenRevocationService) RevokeToken(token string, expiresAt time.Time) error {
	ctx := context.Background()
	key := RevokedTokenPrefix + token
	ttl := time.Until(expiresAt)

	if ttl <= 0 {
		ttl = 1 * time.Hour // Minimum TTL
	}

	return trs.redisClient.Set(ctx, key, "revoked", ttl).Err()
}

// IsTokenRevoked checks if a token is revoked
func (trs *tokenRevocationService) IsTokenRevoked(token string) (bool, error) {
	ctx := context.Background()
	key := RevokedTokenPrefix + token
	result, err := trs.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}
	return result > 0, nil
}

// LicenseHeartbeatService handles license heartbeats
type LicenseHeartbeatService interface {
	RecordHeartbeat(licenseKey string, machineFingerprint string) *errors.ApiError
}

type licenseHeartbeatService struct {
	licenseRepo    repository.ILicenseRepository
	activationRepo repository.ILicenseActivationRepository
	auditSvc       AuditService
	log            interface{} // Logger instance
}

// NewLicenseHeartbeatService creates a new heartbeat service
func NewLicenseHeartbeatService(licenseRepo repository.ILicenseRepository, activationRepo repository.ILicenseActivationRepository, auditSvc AuditService) LicenseHeartbeatService {
	return &licenseHeartbeatService{
		licenseRepo:    licenseRepo,
		activationRepo: activationRepo,
		auditSvc:       auditSvc,
	}
}

// RecordHeartbeat records a heartbeat for a license activation
func (lhs *licenseHeartbeatService) RecordHeartbeat(licenseKey string, machineFingerprint string) *errors.ApiError {
	// Get license by key
	license, err := lhs.licenseRepo.GetByLicenseKey(licenseKey)
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
	activation, err := lhs.activationRepo.GetByLicenseAndMachine(license.ID, machineFingerprint)
	if err != nil {
		return errors.NewNotFoundError("License activation not found for this machine")
	}

	// Check if activation is still active
	if !activation.IsActive() {
		return errors.NewConflictError("License activation is no longer active")
	}

	// Update LastSeenAt
	activation.UpdateLastSeen()
	if err := lhs.activationRepo.Update(activation); err != nil {
		return errors.NewInternalError("Failed to update heartbeat")
	}

	// Log audit event
	lhs.auditSvc.LogAction("LICENSE_HEARTBEAT", "License", license.ID, &license.UserID, nil, nil)

	return nil
}
