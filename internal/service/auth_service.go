package service

import (
	"time"

	"license-management-api/internal/config"
	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/pkg/utils"

	"gorm.io/gorm"
)

type AuthService interface {
	Register(req *dto.RegisterDto, ipAddress string) (*models.User, *errors.ApiError)
	Login(req *dto.LoginDto, ipAddress string) (*dto.LoginResultDto, *errors.ApiError)
	ValidateUser(email string) (*models.User, *errors.ApiError)
	ChangePassword(userID int, req *dto.ChangePasswordDto) *errors.ApiError
	VerifyEmail(token string) *errors.ApiError
	ResendVerificationEmail(email string) *errors.ApiError
	RequestPasswordReset(email string) *errors.ApiError
	ConfirmPasswordReset(token string, newPassword string) *errors.ApiError
	RevokeToken(refreshToken string) *errors.ApiError
}

type authService struct {
	userRepo              repository.IUserRepository
	emailVerificationRepo repository.IEmailVerificationRepository
	passwordResetRepo     repository.IPasswordResetRepository
	tokenSvc              TokenService
	auditSvc              AuditService
	emailSvc              EmailService
	tokenCacheSvc         *TokenCacheService
}

func NewAuthService(userRepo repository.IUserRepository, tokenSvc TokenService, auditSvc AuditService, db interface{}, redisClient *utils.RedisClient, ttlCfg *config.CacheTTLConfig) AuthService {
	// Extract *gorm.DB from db interface
	var gormDB interface{} = db

	// Initialize repositories
	emailVerificationRepo := repository.NewEmailVerificationRepository(gormDB.(*gorm.DB))
	passwordResetRepo := repository.NewPasswordResetRepository(gormDB.(*gorm.DB))

	// Initialize cache service with Redis client
	cacheService := NewCacheService(redisClient, 15*time.Minute)

	// Initialize token cache service with TTL configuration
	tokenCacheSvc := NewTokenCacheService(cacheService, emailVerificationRepo, passwordResetRepo, ttlCfg)

	return &authService{
		userRepo:              userRepo,
		emailVerificationRepo: emailVerificationRepo,
		passwordResetRepo:     passwordResetRepo,
		tokenSvc:              tokenSvc,
		auditSvc:              auditSvc,
		emailSvc:              NewEmailService(LoadEmailConfig()),
		tokenCacheSvc:         tokenCacheSvc,
	}
}

// Register creates a new user
func (as *authService) Register(req *dto.RegisterDto, ipAddress string) (*models.User, *errors.ApiError) {
	// Check if user already exists
	existingUser, _ := as.userRepo.GetByEmail(req.Email)
	if existingUser != nil {
		return nil, errors.NewConflictError("User with this email already exists")
	}

	// Hash password
	hashedPassword, err := utils.HashPassword(req.Password)
	if err != nil {
		return nil, errors.NewInternalError("Failed to hash password")
	}

	// Create user
	user := &models.User{
		Username:                  req.Username,
		Email:                     req.Email,
		PasswordHash:              hashedPassword,
		Role:                      string(models.UserRoleUser),
		Status:                    string(models.UserStatusUnverified),
		NotifyLicenseExpiry:       true,
		NotifyAccountActivity:     true,
		NotifySystemAnnouncements: true,
	}

	if err := as.userRepo.Create(user); err != nil {
		return nil, errors.NewInternalError("Failed to create user")
	}

	// Create email verification token
	token := utils.GenerateRandomString(32)
	verification := &models.EmailVerification{
		UserID:    user.ID,
		Token:     token,
		Email:     user.Email,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour),
	}

	if err := as.emailVerificationRepo.Create(verification); err != nil {
		// Log but don't fail registration if verification record creation fails
	}

	// Cache the verification token (24-hour TTL)
	_ = as.tokenCacheSvc.CacheEmailVerificationToken(token, verification)

	// Log audit
	as.auditSvc.LogAction("USER_REGISTERED", "User", user.ID, &user.ID, nil, &ipAddress)

	// Send welcome email and verification email asynchronously
	go func() {
		if err := as.emailSvc.SendWelcomeEmail(user.Email, user.Username); err != nil {
			// Log but don't fail registration if email fails
		}
		if err := as.emailSvc.SendVerificationEmail(user.Email, user.Username, token); err != nil {
			// Log but don't fail registration if email fails
		}
	}()

	return user, nil
}

// Login authenticates a user
func (as *authService) Login(req *dto.LoginDto, ipAddress string) (*dto.LoginResultDto, *errors.ApiError) {
	user, err := as.userRepo.GetByUsername(req.Username)
	if err != nil || user == nil {
		return nil, errors.NewUnauthorizedError("Invalid username or password")
	}

	// Verify password
	if !utils.CheckPassword(req.Password, user.PasswordHash) {
		return nil, errors.NewUnauthorizedError("Invalid username or password")
	}

	// Check if user is active
	if user.Status != string(models.UserStatusActive) {
		return nil, errors.NewUnauthorizedError("User account is not active")
	}

	// Generate tokens
	accessToken, errAT := as.tokenSvc.GenerateAccessToken(user.ID, user.Email, user.Role)
	if errAT != nil {
		return nil, errors.NewInternalError("Failed to generate access token")
	}

	refreshToken, errRT := as.tokenSvc.GenerateRefreshToken(user.ID)
	if errRT != nil {
		return nil, errors.NewInternalError("Failed to generate refresh token")
	}

	// Log audit
	as.auditSvc.LogAction("USER_LOGIN", "User", user.ID, &user.ID, nil, &ipAddress)

	return &dto.LoginResultDto{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}, nil
}

// ValidateUser retrieves and validates a user
func (as *authService) ValidateUser(email string) (*models.User, *errors.ApiError) {
	user, err := as.userRepo.GetByEmail(email)
	if err != nil || user == nil {
		return nil, errors.NewNotFoundError("User not found")
	}

	return user, nil
}

// ChangePassword changes a user's password
func (as *authService) ChangePassword(userID int, req *dto.ChangePasswordDto) *errors.ApiError {
	user, err := as.userRepo.GetByID(userID)
	if err != nil || user == nil {
		return errors.NewNotFoundError("User not found")
	}

	// Verify old password
	if !utils.CheckPassword(req.OldPassword, user.PasswordHash) {
		return errors.NewUnauthorizedError("Invalid current password")
	}

	// Hash new password
	newHash, hashErr := utils.HashPassword(req.NewPassword)
	if hashErr != nil {
		return errors.NewInternalError("Failed to hash new password")
	}

	user.PasswordHash = newHash
	if err := as.userRepo.Update(user); err != nil {
		return errors.NewInternalError("Failed to update password")
	}

	return nil
}

// VerifyEmail verifies a user's email with a token
func (as *authService) VerifyEmail(token string) *errors.ApiError {
	// Try cache first
	verification, err := as.tokenCacheSvc.GetEmailVerificationToken(token)
	if err != nil {
		// Cache miss, try database
		verification, err = as.emailVerificationRepo.GetByToken(token)
		if err != nil {
			return errors.NewNotFoundError("Invalid or expired verification token")
		}
	}

	if verification.IsExpired() {
		return errors.NewBadRequestError("Verification token has expired")
	}

	if verification.IsUsed() {
		return errors.NewBadRequestError("Verification token has already been used")
	}

	user, err := as.userRepo.GetByID(verification.UserID)
	if err != nil || user == nil {
		return errors.NewNotFoundError("User not found")
	}

	// Mark email as verified
	user.Verify()
	user.Activate()
	if err := as.userRepo.Update(user); err != nil {
		return errors.NewInternalError("Failed to verify email")
	}

	// Mark token as used
	verification.MarkAsUsed()
	if err := as.emailVerificationRepo.Update(verification); err != nil {
		// Log but don't fail
	}

	// Invalidate cached token
	_ = as.tokenCacheSvc.InvalidateEmailVerificationToken(token, int64(verification.UserID))

	// Log audit
	as.auditSvc.LogAction("EMAIL_VERIFIED", "User", user.ID, &user.ID, nil, nil)

	return nil
}

// ResendVerificationEmail sends a new verification email
func (as *authService) ResendVerificationEmail(email string) *errors.ApiError {
	user, err := as.userRepo.GetByEmail(email)
	if err != nil || user == nil {
		return errors.NewNotFoundError("User not found")
	}

	if user.Status == string(models.UserStatusActive) || user.Status == string(models.UserStatusVerified) {
		return errors.NewBadRequestError("User email is already verified")
	}

	// Generate new verification token
	token := utils.GenerateRandomString(32)
	verification := &models.EmailVerification{
		UserID:    user.ID,
		Token:     token,
		Email:     user.Email,
		ExpiresAt: time.Now().UTC().Add(24 * time.Hour), // 24 hour expiry
	}

	if err := as.emailVerificationRepo.Create(verification); err != nil {
		return errors.NewInternalError("Failed to create verification token")
	}

	// Send verification email asynchronously
	go func() {
		as.emailSvc.SendVerificationEmail(user.Email, user.Username, token)
	}()

	return nil
}

// RequestPasswordReset initiates a password reset
func (as *authService) RequestPasswordReset(email string) *errors.ApiError {
	user, err := as.userRepo.GetByEmail(email)
	if err != nil || user == nil {
		return errors.NewNotFoundError("User not found")
	}

	// Generate reset token
	token := utils.GenerateRandomString(32)
	reset := &models.PasswordReset{
		UserID:    user.ID,
		Token:     token,
		Email:     user.Email,
		ExpiresAt: time.Now().UTC().Add(1 * time.Hour), // 1 hour expiry
	}

	if err := as.passwordResetRepo.Create(reset); err != nil {
		return errors.NewInternalError("Failed to create reset token")
	}

	// Cache the password reset token (1-hour TTL)
	_ = as.tokenCacheSvc.CachePasswordResetToken(token, reset)

	// Send reset email asynchronously
	go func() {
		as.emailSvc.SendPasswordResetEmail(user.Email, user.Username, token)
	}()

	// Log audit
	as.auditSvc.LogAction("PASSWORD_RESET_REQUESTED", "User", user.ID, &user.ID, nil, nil)

	return nil
}

// ConfirmPasswordReset completes the password reset process
func (as *authService) ConfirmPasswordReset(token string, newPassword string) *errors.ApiError {
	// Try cache first
	reset, err := as.tokenCacheSvc.GetPasswordResetToken(token)
	if err != nil {
		// Cache miss, try database
		reset, err = as.passwordResetRepo.GetByToken(token)
		if err != nil {
			return errors.NewNotFoundError("Invalid or expired reset token")
		}
	}

	if reset.IsExpired() {
		return errors.NewBadRequestError("Reset token has expired")
	}

	if reset.IsUsed() {
		return errors.NewBadRequestError("Reset token has already been used")
	}

	user, err := as.userRepo.GetByID(reset.UserID)
	if err != nil || user == nil {
		return errors.NewNotFoundError("User not found")
	}

	// Hash new password
	hashedPassword, err := utils.HashPassword(newPassword)
	if err != nil {
		return errors.NewInternalError("Failed to hash password")
	}

	user.PasswordHash = hashedPassword
	if err := as.userRepo.Update(user); err != nil {
		return errors.NewInternalError("Failed to update password")
	}

	// Mark token as used
	reset.MarkAsUsed()
	if err := as.passwordResetRepo.Update(reset); err != nil {
		// Log but don't fail
	}

	// Invalidate cached token
	_ = as.tokenCacheSvc.InvalidatePasswordResetToken(token, int64(reset.UserID))

	// Log audit
	as.auditSvc.LogAction("PASSWORD_RESET_CONFIRMED", "User", user.ID, &user.ID, nil, nil)

	return nil
}

// RevokeToken revokes a refresh token
func (as *authService) RevokeToken(refreshToken string) *errors.ApiError {
	// Validate token format (basic check)
	if refreshToken == "" {
		return errors.NewBadRequestError("Refresh token is required")
	}

	// Store revoked token in Redis with expiration
	trs := NewTokenRevocationService(nil)                 // Will be injected from server
	expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour) // Standard refresh token expiry

	if err := trs.RevokeToken(refreshToken, expiresAt); err != nil {
		return errors.NewInternalError("Failed to revoke token")
	}

	return nil
}
