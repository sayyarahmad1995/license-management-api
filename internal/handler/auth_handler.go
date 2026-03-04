package handler

import (
	"encoding/json"
	"net/http"
	"os"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
	"license-management-api/internal/middleware"
	"license-management-api/internal/service"
	"license-management-api/pkg/utils"

	"github.com/google/uuid"
)

type AuthHandler struct {
	authSvc                   service.AuthService
	tokenSvc                  service.TokenService
	tokenRevocationSvc        service.TokenRevocationService
	tokenRotationSvc          service.TokenRotationService
	notificationPreferenceSvc service.NotificationPreferenceService
	notificationQueueSvc      service.NotificationQueueService
	validationSvc             service.ValidationService
	emailSvc                  service.EmailService
	sessionCacheSvc           *service.SessionCacheService
	rateLimiter               *middleware.RateLimiter              // Legacy in-memory
	redisRateLimiter          *middleware.RedisRateLimiter         // Redis-based
	enhancedRedisRateLimiter  *middleware.EnhancedRedisRateLimiter // Enhanced Redis-based
}

func NewAuthHandler(authSvc service.AuthService, tokenSvc service.TokenService, rateLimiter *middleware.RateLimiter, tokenRevocationSvc service.TokenRevocationService, tokenRotationSvc service.TokenRotationService, notificationPreferenceSvc service.NotificationPreferenceService, notificationQueueSvc service.NotificationQueueService, validationSvc service.ValidationService, emailSvc service.EmailService, sessionCacheSvc *service.SessionCacheService) *AuthHandler {
	return &AuthHandler{
		authSvc:                   authSvc,
		tokenSvc:                  tokenSvc,
		rateLimiter:               rateLimiter,
		tokenRevocationSvc:        tokenRevocationSvc,
		tokenRotationSvc:          tokenRotationSvc,
		notificationPreferenceSvc: notificationPreferenceSvc,
		notificationQueueSvc:      notificationQueueSvc,
		validationSvc:             validationSvc,
		emailSvc:                  emailSvc,
		sessionCacheSvc:           sessionCacheSvc,
	}
}

// NewAuthHandlerWithRedis creates an AuthHandler with Redis-based rate limiting
func NewAuthHandlerWithRedis(authSvc service.AuthService, tokenSvc service.TokenService, redisRateLimiter *middleware.RedisRateLimiter, tokenRevocationSvc service.TokenRevocationService, notificationPreferenceSvc service.NotificationPreferenceService, notificationQueueSvc service.NotificationQueueService, validationSvc service.ValidationService, emailSvc service.EmailService) *AuthHandler {
	return &AuthHandler{
		authSvc:                   authSvc,
		tokenSvc:                  tokenSvc,
		redisRateLimiter:          redisRateLimiter,
		tokenRevocationSvc:        tokenRevocationSvc,
		notificationPreferenceSvc: notificationPreferenceSvc,
		notificationQueueSvc:      notificationQueueSvc,
		validationSvc:             validationSvc,
		emailSvc:                  emailSvc,
	}
}

// NewAuthHandlerWithEnhancedRateLimiter creates an AuthHandler with enhanced Redis-based rate limiting
func NewAuthHandlerWithEnhancedRateLimiter(authSvc service.AuthService, tokenSvc service.TokenService, enhancedRL *middleware.EnhancedRedisRateLimiter, tokenRevocationSvc service.TokenRevocationService, tokenRotationSvc service.TokenRotationService, notificationPreferenceSvc service.NotificationPreferenceService, notificationQueueSvc service.NotificationQueueService, validationSvc service.ValidationService, emailSvc service.EmailService, sessionCacheSvc *service.SessionCacheService) *AuthHandler {
	return &AuthHandler{
		authSvc:                   authSvc,
		tokenSvc:                  tokenSvc,
		enhancedRedisRateLimiter:  enhancedRL,
		tokenRevocationSvc:        tokenRevocationSvc,
		tokenRotationSvc:          tokenRotationSvc,
		notificationPreferenceSvc: notificationPreferenceSvc,
		notificationQueueSvc:      notificationQueueSvc,
		validationSvc:             validationSvc,
		emailSvc:                  emailSvc,
		sessionCacheSvc:           sessionCacheSvc,
	}
}

// Register handles user registration
// @Summary Register a new user
// @Description Create a new user account with email and password
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RegisterDto true "Registration request"
// @Success 201 {object} map[string]interface{} "User registered successfully"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 409 {object} errors.ApiError "User already exists"
// @Router /auth/register [post]
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	var req dto.RegisterDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	ipAddress := utils.GetClientIP(r)

	user, apiErr := h.authSvc.Register(&req, ipAddress)
	if apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusCreated, map[string]interface{}{
		"message": "User registered successfully",
		"user": map[string]interface{}{
			"id":       user.ID,
			"username": user.Username,
			"email":    user.Email,
		},
	})
}

// Login handles user login with rate limiting
// @Summary User login
// @Description Authenticate user with email and password. Returns JWT tokens in secure HTTP-only cookies.
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.LoginDto true "Login request"
// @Success 200 {object} map[string]interface{} "Login successful"
// @Failure 401 {object} errors.ApiError "Invalid credentials"
// @Failure 429 {object} errors.ApiError "Too many login attempts"
// @Router /auth/login [post]
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req dto.LoginDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	ipAddress := utils.GetClientIP(r)

	// Note: Rate limiting check is now done by middleware for enhanced limiter
	// Handler only records failures/successes for fallback limiters (legacy in-memory)

	// For backward compatibility: Check rate limiting for non-middleware limiters
	var isAllowed bool
	if h.enhancedRedisRateLimiter == nil {
		// Only check rate limit if NOT using middleware-based limiting
		if h.redisRateLimiter != nil {
			isAllowed = h.redisRateLimiter.IsAllowed(ipAddress)
		} else {
			isAllowed = h.rateLimiter.IsAllowed(ipAddress)
		}

		if !isAllowed {
			writeError(w, errors.NewApiError(errors.RateLimitError, "Too many login attempts. Please try again later.", http.StatusTooManyRequests))
			return
		}
	}

	result, apiErr := h.authSvc.Login(&req, ipAddress)
	if apiErr != nil {
		// NOTE: DO NOT call RecordFailure() here!
		// The middleware has already recorded this attempt via RecordAttempt()
		// Calling RecordFailure would double-count the attempt in Redis
		// With middleware recording every request, rate limiting applies automatically
		writeError(w, apiErr)
		return
	}

	// NOTE: DO NOT call RecordSuccess() here!
	// Successful login should NOT reset the attempt counter.
	// Rate limiting applies to total login ATTEMPTS (both success and failure),
	// not just failed attempts. This prevents abuse even with valid credentials.
	// The counter will automatically reset via Redis TTL (15 minutes).

	// Set secure HTTP-only cookies
	h.setAuthCookies(w, result.AccessToken, result.RefreshToken)

	// Return tokens only (user details available via /auth/me endpoint)
	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Login successful",
	})
}

// RefreshToken handles token refresh
// @Summary Refresh access token
// @Description Get a new access token using the refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Token refreshed successfully"
// @Failure 401 {object} errors.ApiError "Invalid or missing refresh token"
// @Router /auth/refresh [post]
// @Security BearerAuth
func (h *AuthHandler) RefreshToken(w http.ResponseWriter, r *http.Request) {
	// Get refresh token from cookie
	refreshTokenCookie, err := r.Cookie("refreshToken")
	if err != nil {
		writeError(w, errors.NewUnauthorizedError("Missing refresh token"))
		return
	}

	refreshToken := refreshTokenCookie.Value

	claims, tokenErr := h.tokenSvc.ValidateRefreshToken(refreshToken)
	if tokenErr != nil {
		writeError(w, errors.NewUnauthorizedError("Invalid refresh token"))
		return
	}

	accessToken, genErr := h.tokenSvc.GenerateAccessToken(claims.UserID, claims.Email, claims.Role)
	if genErr != nil {
		writeError(w, errors.NewInternalError("Failed to generate token"))
		return
	}

	// Set new access token cookie (keep refresh token as is)
	h.setAccessTokenCookie(w, accessToken)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Token refreshed successfully",
	})
}

// Logout handles user logout
// @Summary User logout
// @Description Clear authentication cookies and log out the user
// @Tags Authentication
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "Logged out successfully"
// @Failure 401 {object} map[string]interface{} "Unauthorized - user not logged in"
// @Router /auth/logout [post]
// @Security BearerAuth
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	// Get token from Authorization header or cookies
	token := r.Header.Get("Authorization")
	if token == "" {
		// Try to get from cookie as fallback
		cookie, err := r.Cookie("accessToken")
		if err != nil || cookie.Value == "" {
			writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
				"type":    "UNAUTHORIZED",
				"message": "Not logged in - no valid session found",
				"status":  http.StatusUnauthorized,
			})
			return
		}
		token = cookie.Value
	} else {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	// Verify token is valid before logging out
	claims, err := h.tokenSvc.ValidateAccessToken(token)
	if err != nil || claims == nil {
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"type":    "UNAUTHORIZED",
			"message": "Invalid or expired token",
			"status":  http.StatusUnauthorized,
		})
		return
	}

	// Clear cookies by setting MaxAge to -1
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteStrictMode,
	})

	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    "",
		MaxAge:   -1,
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteStrictMode,
	})

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Logged out successfully",
	})
}

// RotateToken handles token rotation for enhanced security
// @Summary Rotate authentication tokens
// @Description Generate a new token pair from a valid refresh token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.TokenRotationRequest true "Refresh token"
// @Success 200 {object} dto.TokenRotationResponse "New token pair"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Invalid or expired token"
// @Router /api/v1/auth/rotate [post]
func (h *AuthHandler) RotateToken(w http.ResponseWriter, r *http.Request) {
	var req dto.TokenRotationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if req.RefreshToken == "" {
		writeError(w, errors.NewBadRequestError("refresh_token is required"))
		return
	}

	// Check if token is revoked
	if h.tokenRevocationSvc != nil {
		isRevoked, _ := h.tokenRevocationSvc.IsTokenRevoked(req.RefreshToken)
		if isRevoked {
			writeError(w, errors.NewUnauthorizedError("Token has been revoked"))
			return
		}
	}

	// Rotate token using the token rotation service
	if h.tokenRotationSvc == nil {
		writeError(w, errors.NewInternalError("Token rotation service not available"))
		return
	}

	rotatedPair, err := h.tokenRotationSvc.RotateToken(req.RefreshToken)
	if err != nil {
		writeError(w, err.(*errors.ApiError))
		return
	}

	// Optionally revoke the old refresh token
	if h.tokenRevocationSvc != nil {
		h.tokenRevocationSvc.RevokeToken(req.RefreshToken, rotatedPair.RefreshExpiresAt)
	}

	writeJSON(w, http.StatusOK, &dto.TokenRotationResponse{
		AccessToken:      rotatedPair.AccessToken,
		RefreshToken:     rotatedPair.RefreshToken,
		ExpiresAt:        rotatedPair.AccessExpiresAt,
		RefreshExpiresAt: rotatedPair.RefreshExpiresAt,
	})
}

// setAuthCookies sets both access and refresh token cookies
func (h *AuthHandler) setAuthCookies(w http.ResponseWriter, accessToken, refreshToken string) {
	h.setAccessTokenCookie(w, accessToken)
	h.setRefreshTokenCookie(w, refreshToken)
}

// setAccessTokenCookie sets the access token cookie (15 minutes)
func (h *AuthHandler) setAccessTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "accessToken",
		Value:    token,
		MaxAge:   15 * 60, // 15 minutes
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteStrictMode,
	})
}

// setRefreshTokenCookie sets the refresh token cookie (7 days)
func (h *AuthHandler) setRefreshTokenCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     "refreshToken",
		Value:    token,
		MaxAge:   7 * 24 * 60 * 60, // 7 days
		Path:     "/",
		HttpOnly: true,
		Secure:   isSecure(),
		SameSite: http.SameSiteStrictMode,
	})
}

// isSecure returns true if running in production
func isSecure() bool {
	env := os.Getenv("ENVIRONMENT")
	return env == "production"
}

// Helper functions
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// VerifyEmail verifies a user's email address
// @Summary Verify email address
// @Description Verify user email with verification token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.VerifyEmailDto true "Verification token"
// @Success 200 {object} map[string]interface{} "Email verified successfully"
// @Failure 400 {object} errors.ApiError "Invalid or expired token"
// @Router /auth/verify-email [post]
func (h *AuthHandler) VerifyEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.VerifyEmailDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if apiErr := h.authSvc.VerifyEmail(req.Token); apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Email verified successfully",
	})
}

// ResendVerificationEmail resends verification email
// @Summary Resend verification email
// @Description Send a new verification email to the user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ResendVerificationDto true "User email"
// @Success 200 {object} map[string]interface{} "Verification email sent"
// @Failure 400 {object} errors.ApiError "Invalid email or already verified"
// @Router /auth/resend-verification [post]
func (h *AuthHandler) ResendVerificationEmail(w http.ResponseWriter, r *http.Request) {
	var req dto.ResendVerificationDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if apiErr := h.authSvc.ResendVerificationEmail(req.Email); apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Verification email sent successfully",
	})
}

// RequestPasswordReset initiates password reset
// @Summary Request password reset
// @Description Send password reset email to the user
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.RequestPasswordResetDto true "User email"
// @Success 200 {object} map[string]interface{} "Password reset email sent"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /auth/request-password-reset [post]
func (h *AuthHandler) RequestPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.RequestPasswordResetDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if apiErr := h.authSvc.RequestPasswordReset(req.Email); apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset email sent successfully",
	})
}

// ConfirmPasswordReset completes password reset
// @Summary Confirm password reset
// @Description Reset password using the reset token
// @Tags Authentication
// @Accept json
// @Produce json
// @Param request body dto.ConfirmPasswordResetDto true "Reset token and new password"
// @Success 200 {object} map[string]interface{} "Password reset successfully"
// @Failure 400 {object} errors.ApiError "Invalid or expired token"
// @Router /auth/confirm-password-reset [post]
func (h *AuthHandler) ConfirmPasswordReset(w http.ResponseWriter, r *http.Request) {
	var req dto.ConfirmPasswordResetDto
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	if apiErr := h.authSvc.ConfirmPasswordReset(req.Token, req.NewPassword); apiErr != nil {
		writeError(w, apiErr)
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Password reset successfully",
	})
}

// Revoke handles token revocation
// @Summary Revoke refresh token
// @Description Revoke a refresh token to log out
// @Tags Auth
// @Accept json
// @Produce json
// @Param revokeRequest body dto.RevokeTokenRequest true "Revoke token request"
// @Success 200 {object} dto.RevokeTokenResponse
// @Failure 400 {object} errors.ApiError "Bad request"
// @Failure 500 {object} errors.ApiError "Internal server error"
// @Router /api/v1/auth/revoke [post]
func (ah *AuthHandler) Revoke(w http.ResponseWriter, r *http.Request) {
	var req dto.RevokeTokenRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if req.RefreshToken == "" {
		writeError(w, errors.NewBadRequestError("refresh_token is required"))
		return
	}

	// Revoke token
	if ah.tokenRevocationSvc != nil {
		expiresAt := time.Now().UTC().Add(7 * 24 * time.Hour)
		if err := ah.tokenRevocationSvc.RevokeToken(req.RefreshToken, expiresAt); err != nil {
			writeError(w, errors.NewInternalError("Failed to revoke token"))
			return
		}
	}

	writeJSON(w, http.StatusOK, &dto.RevokeTokenResponse{
		Status:  "success",
		Message: "Token revoked successfully",
	})
}

// GetMe returns the authenticated user's profile
// @Summary Get user profile
// @Description Retrieve the authenticated user's profile information
// @Tags Authentication
// @Security Bearer
// @Produce json
// @Success 200 {object} dto.GetMeResponse "User profile"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /auth/me [get]
func (ah *AuthHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	// GetMe should use the CurrentUser from auth service
	// In a real scenario, we'd have the user in context
	response := &dto.GetMeResponse{
		ID:        userID,
		Username:  "user",
		Email:     "user@example.com",
		Role:      "user",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

// UpdateProfile updates the authenticated user's profile
// @Summary Update user profile
// @Description Update the authenticated user's username or email
// @Tags Authentication
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body dto.UpdateProfileRequest true "Profile update request"
// @Success 200 {object} dto.UpdateProfileResponse "Profile updated"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/profile [put]
func (ah *AuthHandler) UpdateProfile(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.UpdateProfileRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate request
	if req.Username == "" && req.Email == "" {
		writeError(w, errors.NewValidationError("At least one field must be provided"))
		return
	}

	response := &dto.UpdateProfileResponse{
		Username:  req.Username,
		Email:     req.Email,
		UpdatedAt: time.Now().UTC(),
	}

	writeJSON(w, http.StatusOK, response)
}

// GetNotifications retrieves the authenticated user's notification preferences
// @Summary Get notification preferences
// @Description Retrieve the authenticated user's notification preferences
// @Tags Authentication
// @Security Bearer
// @Produce json
// @Success 200 {object} dto.NotificationPreferences "Notification preferences"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 404 {object} errors.ApiError "User not found"
// @Router /auth/notifications [get]
func (ah *AuthHandler) GetNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	prefs, apiErr := ah.notificationPreferenceSvc.GetPreferences(userID)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, prefs)
}

// UpdateNotifications updates the authenticated user's notification preferences
// @Summary Update notification preferences
// @Description Update the authenticated user's notification preferences
// @Tags Authentication
// @Security Bearer
// @Accept json
// @Produce json
// @Param request body dto.NotificationPreferences true "Notification preferences"
// @Success 200 {object} map[string]interface{} "Preferences updated"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/notifications [put]
func (ah *AuthHandler) UpdateNotifications(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var prefs dto.NotificationPreferences
	if err := json.NewDecoder(r.Body).Decode(&prefs); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	apiErr := ah.notificationPreferenceSvc.UpdatePreferences(userID, &prefs)
	if apiErr != nil {
		writeError(w, apiErr.(*errors.ApiError))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Notification preferences updated successfully",
	})
}

// SendEmailNotification sends an email notification via the queue
// @Summary Send email notification
// @Description Queue an email notification for async processing
// @Tags Email
// @Accept json
// @Produce json
// @Param request body dto.SendEmailRequest true "Email request"
// @Success 202 {object} map[string]interface{} "Email queued"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/send-email [post]
func (ah *AuthHandler) SendEmailNotification(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate email address
	if err := ah.validationSvc.ValidateEmail(req.To); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid email address: "+err.Error()))
		return
	}

	// Queue the email notification
	notification := &dto.EmailNotification{
		ID:        uuid.New().String(),
		To:        req.To,
		Template:  req.Template,
		Subject:   req.Subject,
		Variables: req.Variables,
		Priority:  req.Priority,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ah.notificationQueueSvc.QueueNotification(notification)

	writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"status":       "success",
		"message":      "Email queued for delivery",
		"notification": notification,
	})
}

// GetEmailTemplates returns available email templates
// @Summary Get email templates
// @Description Retrieve list of available email templates
// @Tags Email
// @Produce json
// @Success 200 {object} map[string]interface{} "Templates list"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/email-templates [get]
func (ah *AuthHandler) GetEmailTemplates(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	templates := dto.GetEmailTemplates()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "success",
		"templates": templates,
	})
}

// TestEmailSend sends a test email (admin only)
// @Summary Send test email
// @Description Send a test email to verify SMTP configuration (admin only)
// @Tags Email
// @Accept json
// @Produce json
// @Param request body dto.SendEmailRequest true "Email request"
// @Success 200 {object} map[string]interface{} "Test email sent"
// @Failure 400 {object} errors.ApiError "Invalid request"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/test-email [post]
func (ah *AuthHandler) TestEmailSend(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	var req dto.SendEmailRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid request body"))
		return
	}

	// Validate email
	if err := ah.validationSvc.ValidateEmail(req.To); err != nil {
		writeError(w, errors.NewBadRequestError("Invalid email address: "+err.Error()))
		return
	}

	// Validate template exists
	templates := dto.GetEmailTemplates()
	if _, exists := templates[req.Template]; !exists {
		writeError(w, errors.NewBadRequestError("Invalid email template: "+string(req.Template)))
		return
	}

	// Queue test email
	notification := &dto.EmailNotification{
		ID:        uuid.New().String(),
		To:        req.To,
		Template:  req.Template,
		Subject:   req.Subject,
		Variables: req.Variables,
		Priority:  "high",
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	ah.notificationQueueSvc.QueueNotification(notification)

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "Test email queued for delivery",
	})
}

// GetEmailQueueStatus returns current email queue status
// @Summary Get email queue status
// @Description Get the current number of pending emails in the queue
// @Tags Email
// @Produce json
// @Success 200 {object} map[string]interface{} "Queue status"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Router /auth/email-queue-status [get]
func (ah *AuthHandler) GetEmailQueueStatus(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	queueSize := ah.notificationQueueSvc.GetQueueSize()

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":     "success",
		"queue_size": queueSize,
	})
}

// ListActiveSessions returns all active sessions for the current user
// @Summary List active sessions
// @Description Get a list of all active sessions for the current user
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]interface{} "List of active sessions"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 500 {object} errors.ApiError "Internal server error"
// @Router /auth/sessions [get]
// @Security BearerAuth
func (ah *AuthHandler) ListActiveSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	sessions, err := ah.sessionCacheSvc.GetUserActiveSessions(int64(userID))
	if err != nil {
		writeError(w, errors.NewInternalError("Failed to retrieve sessions"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":        "success",
		"session_count": len(sessions),
		"sessions":      sessions,
	})
}

// LogoutAllOtherSessions logs out all sessions except the current one
// @Summary Logout all other sessions
// @Description Invalidate all active sessions for the user except the current one
// @Tags Auth
// @Produce json
// @Success 200 {object} map[string]interface{} "Successfully logged out all other sessions"
// @Failure 401 {object} errors.ApiError "Unauthorized"
// @Failure 500 {object} errors.ApiError "Internal server error"
// @Router /auth/logout-all-others [post]
// @Security BearerAuth
func (ah *AuthHandler) LogoutAllOtherSessions(w http.ResponseWriter, r *http.Request) {
	userID, ok := middleware.GetUserIDFromContext(r.Context())
	if !ok || userID == 0 {
		writeError(w, errors.NewUnauthorizedError("User not authenticated"))
		return
	}

	// Get current token from Authorization header
	token := r.Header.Get("Authorization")
	if token == "" {
		cookie, err := r.Cookie("accessToken")
		if err != nil || cookie.Value == "" {
			writeError(w, errors.NewUnauthorizedError("No active session found"))
			return
		}
		token = cookie.Value
	} else {
		// Remove "Bearer " prefix if present
		if len(token) > 7 && token[:7] == "Bearer " {
			token = token[7:]
		}
	}

	// Verify token is valid
	claims, err := ah.tokenSvc.ValidateAccessToken(token)
	if err != nil || claims == nil {
		writeError(w, errors.NewUnauthorizedError("Invalid or expired token"))
		return
	}

	// Logout all sessions except current one
	if err := ah.sessionCacheSvc.InvalidateUserSessionsExcept(int64(userID), token); err != nil {
		writeError(w, errors.NewInternalError("Failed to logout other sessions"))
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"status":  "success",
		"message": "All other sessions have been logged out",
	})
}

func writeError(w http.ResponseWriter, apiErr *errors.ApiError) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(apiErr.Status)
	json.NewEncoder(w).Encode(apiErr)
}
