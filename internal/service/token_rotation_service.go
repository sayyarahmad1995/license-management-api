package service

import (
	"time"

	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

// TokenRotationService handles token rotation logic
type TokenRotationService interface {
	RotateToken(refreshToken string) (*RotatedTokenPair, error)
	ValidateAndRotate(refreshToken string) (*RotatedTokenPair, error)
}

// RotatedTokenPair represents the new token pair after rotation
type RotatedTokenPair struct {
	AccessToken      string
	RefreshToken     string
	AccessExpiresAt  time.Time
	RefreshExpiresAt time.Time
}

type tokenRotationService struct {
	tokenSvc TokenService
	userRepo repository.IUserRepository
}

// NewTokenRotationService creates a new token rotation service
func NewTokenRotationService(tokenSvc TokenService, userRepo repository.IUserRepository) TokenRotationService {
	return &tokenRotationService{
		tokenSvc: tokenSvc,
		userRepo: userRepo,
	}
}

// RotateToken generates a new token pair from a valid refresh token
func (trs *tokenRotationService) RotateToken(refreshToken string) (*RotatedTokenPair, error) {
	// Validate refresh token
	claims, err := trs.tokenSvc.ValidateRefreshToken(refreshToken)
	if err != nil {
		return nil, errors.NewUnauthorizedError("Invalid or expired refresh token")
	}

	// Get user from repository
	user, err := trs.userRepo.GetByID(claims.UserID)
	if err != nil {
		return nil, errors.NewNotFoundError("User not found")
	}

	if user.Status != string(models.UserStatusActive) && user.Status != string(models.UserStatusVerified) {
		return nil, errors.NewForbiddenError("User account is inactive")
	}

	// Generate new token pair
	accessToken, _ := trs.tokenSvc.GenerateAccessToken(user.ID, user.Email, user.Role)
	newRefreshToken, _ := trs.tokenSvc.GenerateRefreshToken(user.ID, user.Email, user.Role)

	return &RotatedTokenPair{
		AccessToken:      accessToken,
		RefreshToken:     newRefreshToken,
		AccessExpiresAt:  time.Now().UTC().Add(15 * time.Minute),
		RefreshExpiresAt: time.Now().UTC().Add(7 * 24 * time.Hour),
	}, nil
}

// ValidateAndRotate validates the refresh token and generates a new pair
func (trs *tokenRotationService) ValidateAndRotate(refreshToken string) (*RotatedTokenPair, error) {
	return trs.RotateToken(refreshToken)
}
