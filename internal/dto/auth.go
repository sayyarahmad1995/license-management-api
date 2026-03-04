package dto

import "time"

// RegisterDto for user registration
type RegisterDto struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8,max=100"`
}

// LoginDto for user login
type LoginDto struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Password string `json:"password" validate:"required"`
}

// LoginResultDto response after login
type LoginResultDto struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

// RefreshTokenDto for token refresh
type RefreshTokenDto struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

// ChangePasswordDto for password change
type ChangePasswordDto struct {
	OldPassword string `json:"oldPassword" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

// ResetPasswordDto for password reset
type ResetPasswordDto struct {
	Email string `json:"email" validate:"required,email"`
}

// UpdateUserDto for updating user profile
type UpdateUserDto struct {
	Username                  string `json:"username,omitempty" validate:"omitempty,min=3,max=50"`
	NotifyLicenseExpiry       *bool  `json:"notify_license_expiry,omitempty"`
	NotifyAccountActivity     *bool  `json:"notify_account_activity,omitempty"`
	NotifySystemAnnouncements *bool  `json:"notify_system_announcements,omitempty"`
}

// ResendVerificationDto for resending verification email
type ResendVerificationDto struct {
	Email string `json:"email" validate:"required,email"`
}

// NotificationPreferencesDto for notification settings
type NotificationPreferencesDto struct {
	NotifyLicenseExpiry       bool `json:"notifyLicenseExpiry"`
	NotifyAccountActivity     bool `json:"notifyAccountActivity"`
	NotifySystemAnnouncements bool `json:"notifySystemAnnouncements"`
}

// UserDto represents a user
type UserDto struct {
	ID                      int                        `json:"id"`
	Username                string                     `json:"username"`
	Email                   string                     `json:"email"`
	Role                    string                     `json:"role"`
	Status                  string                     `json:"status"`
	CreatedAt               time.Time                  `json:"createdAt"`
	VerifiedAt              *time.Time                 `json:"verifiedAt,omitempty"`
	UpdatedAt               *time.Time                 `json:"updatedAt,omitempty"`
	LastLogin               *time.Time                 `json:"lastLogin,omitempty"`
	BlockedAt               *time.Time                 `json:"blockedAt,omitempty"`
	NotificationPreferences NotificationPreferencesDto `json:"notificationPreferences"`
}

// CreateUserDto for admin user creation
type CreateUserDto struct {
	Username string `json:"username" validate:"required,min=3,max=50"`
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     string `json:"role" validate:"required,oneof=User Manager Admin"`
}

// UpdateUserStatusDto for updating user status
type UpdateUserStatusDto struct {
	Status string `json:"status" validate:"required,oneof=Unverified Verified Active Blocked"`
}

// UpdateUserRoleDto for updating user role
type UpdateUserRoleDto struct {
	Role string `json:"role" validate:"required,oneof=User Manager Admin"`
}

// VerifyEmailDto for verifying email with token
type VerifyEmailDto struct {
	Token string `json:"token" validate:"required"`
}

// RequestPasswordResetDto for requesting password reset
type RequestPasswordResetDto struct {
	Email string `json:"email" validate:"required,email"`
}

// ConfirmPasswordResetDto for confirming password reset
type ConfirmPasswordResetDto struct {
	Token       string `json:"token" validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8,max=100"`
}
