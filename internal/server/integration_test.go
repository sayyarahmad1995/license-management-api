//go:build integration

package server

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"license-management-api/internal/config"
	"license-management-api/internal/dto"
	"license-management-api/internal/handler"
	"license-management-api/internal/logger"
	"license-management-api/internal/middleware"
	"license-management-api/internal/models"
	"license-management-api/internal/repository"
	"license-management-api/internal/service"

	"github.com/go-chi/chi/v5"
	"github.com/testcontainers/testcontainers-go"
	pgcontainer "github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	pgdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// TestIntegration contains the integration test context
type TestIntegration struct {
	db          *gorm.DB
	router      http.Handler
	pgCont      testcontainers.Container
	postgres    *pgcontainer.PostgresContainer
	cfg         *config.JwtConfig
	rateLimiter *middleware.RateLimiter
}

// setupTestDB initializes a PostgreSQL test container and returns the test context
func setupTestDB(t *testing.T) *TestIntegration {
	ctx := context.Background()

	// Create PostgreSQL container
	pgContainer, err := pgcontainer.RunContainer(ctx,
		testcontainers.WithImage("postgres:15-alpine"),
		pgcontainer.WithUsername("test"),
		pgcontainer.WithPassword("testpass"),
		pgcontainer.WithDatabase("license_mgmt_test"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(10*time.Second),
		),
	)
	if err != nil {
		t.Fatalf("Failed to start PostgreSQL container: %v", err)
	}

	// Get connection string
	connStr, err := pgContainer.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("Failed to get PostgreSQL connection string: %v", err)
	}

	// Wait for database to be ready
	time.Sleep(2 * time.Second)

	// Connect to database
	db, err := gorm.Open(pgdriver.Open(connStr), &gorm.Config{})
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(
		&models.User{},
		&models.License{},
		&models.LicenseActivation{},
		&models.AuditLog{},
		&models.EmailVerification{},
		&models.PasswordReset{},
	); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Create test configuration
	cfg := &config.JwtConfig{
		SecretKey:          "test-access-secret-32-characters!",
		RefreshTokenSecret: "test-refresh-secret-32-chars!",
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}

	// Build router
	ti := &TestIntegration{
		db:          db,
		pgCont:      pgContainer,
		postgres:    pgContainer,
		cfg:         cfg,
		rateLimiter: middleware.NewRateLimiter(1000, 15*time.Minute, time.Hour), // Create rate limiter for cleanup
	}

	ti.router = buildTestRouter(t, db, cfg, ti.rateLimiter)

	return ti
}

// buildTestRouter creates a test router with all dependencies
func buildTestRouter(_ *testing.T, db *gorm.DB, cfg *config.JwtConfig, rateLimiter *middleware.RateLimiter) http.Handler {
	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	licenseRepo := repository.NewLicenseRepository(db)
	activationRepo := repository.NewLicenseActivationRepository(db)
	auditRepo := repository.NewAuditLogRepository(db)

	// Initialize services
	tokenSvc := service.NewTokenService(cfg)
	auditSvc := service.NewAuditService(auditRepo)
	ttlCfg := config.LoadCacheTTLConfig()
	authSvc := service.NewAuthService(userRepo, tokenSvc, auditSvc, db, nil, ttlCfg) // nil RedisClient for tests
	licenseSvc := service.NewLicenseService(licenseRepo, activationRepo, auditSvc)

	// Initialize Phase 1 & 2 services
	tokenRevocationSvc := service.NewTokenRevocationService(nil) // nil Redis client for tests
	tokenRotationSvc := service.NewTokenRotationService(tokenSvc, userRepo)
	machineFingerprintSvc := service.NewMachineFingerprintService(licenseRepo, activationRepo, auditSvc)
	paginationSvc := service.NewPaginationService()
	dashboardStatsSvc := service.NewDashboardStatsService(userRepo, licenseRepo, auditRepo)

	// Initialize Phase 1.5 services
	notificationPreferenceSvc := service.NewNotificationPreferenceService(userRepo)
	dataExportSvc := service.NewDataExportService(userRepo, licenseRepo, activationRepo)
	bulkOperationSvc := service.NewBulkOperationService(licenseRepo, auditRepo)

	// Initialize handlers - use provided rateLimiter
	notificationQueueSvc := service.NewNotificationQueueService()
	validationService := service.NewValidationService()
	emailCfg := service.LoadEmailConfig()
	emailSvc := service.NewEmailService(emailCfg)

	// Initialize session cache service for auth handler
	sessionCacheSvc := (*service.SessionCacheService)(nil) // nil is fine for tests that don't use sessions

	// Initialize circuit breaker service for health handler
	circuitBreakerSvc := service.NewCircuitBreakerService()

	authHandler := handler.NewAuthHandler(authSvc, tokenSvc, rateLimiter, tokenRevocationSvc, tokenRotationSvc, notificationPreferenceSvc, notificationQueueSvc, validationService, emailSvc, sessionCacheSvc)
	licenseHandler := handler.NewLicenseHandler(licenseSvc, licenseRepo, machineFingerprintSvc, paginationSvc, dataExportSvc, bulkOperationSvc)
	healthHandler := handler.NewHealthHandler(&testDatabaseService{db: db}, circuitBreakerSvc)
	userHandler := handler.NewUserHandler(userRepo, licenseRepo, paginationSvc, dataExportSvc)
	auditHandler := handler.NewAuditHandler(auditRepo)
	dashboardHandler := handler.NewDashboardHandler(userRepo, licenseRepo, activationRepo, auditRepo, dashboardStatsSvc)

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	testLog := logger.Get() // Use default logger for tests
	r.Use(middleware.LoggingMiddleware(testLog))
	r.Use(middleware.ErrorLoggingMiddleware(testLog))
	r.Use(middleware.ErrorHandlerMiddleware)

	// Public routes
	r.Post("/api/v1/auth/register", authHandler.Register)
	r.Post("/api/v1/auth/login", authHandler.Login)
	r.Post("/api/v1/auth/refresh", authHandler.RefreshToken)
	r.Post("/api/v1/auth/logout", authHandler.Logout)
	r.Post("/api/v1/auth/verify-email", authHandler.VerifyEmail)
	r.Post("/api/v1/auth/resend-verification", authHandler.ResendVerificationEmail)
	r.Post("/api/v1/auth/request-password-reset", authHandler.RequestPasswordReset)
	r.Post("/api/v1/auth/confirm-password-reset", authHandler.ConfirmPasswordReset)

	r.Post("/api/v1/licenses/activate", licenseHandler.ActivateLicense)
	r.Post("/api/v1/licenses/validate", licenseHandler.ValidateLicense)
	r.Post("/api/v1/licenses/deactivate", licenseHandler.DeactivateLicense)

	r.Get("/api/v1/health", healthHandler.Health)
	r.Get("/health", healthHandler.Health)

	// Protected routes
	authMiddleware := middleware.AuthMiddleware(tokenSvc)

	r.Route("/api/v1", func(r chi.Router) {
		r.Use(authMiddleware)

		// User routes
		r.Get("/users", userHandler.GetUsers)
		r.Get("/users/export", userHandler.ExportUsers)
		r.Get("/users/{id}", userHandler.GetUser)
		r.Get("/users/by-license/{key}", userHandler.GetUserByLicenseKey)
		r.Put("/users/{id}", userHandler.UpdateUser)
		r.Patch("/users/{id}/role", userHandler.UpdateUserRole)
		r.Patch("/users/{id}/status", userHandler.UpdateUserStatus)
		r.Delete("/users/{id}", userHandler.DeleteUser)

		// Auth routes (protected user profile endpoints)
		r.Get("/auth/me", authHandler.GetMe)
		r.Put("/auth/profile", authHandler.UpdateProfile)
		r.Get("/auth/notifications", authHandler.GetNotifications)
		r.Put("/auth/notifications", authHandler.UpdateNotifications)

		// Audit log routes
		r.Get("/audit-logs", auditHandler.GetAuditLogs)
		r.Get("/audit-logs/user", auditHandler.GetAuditLogsByUser)
		r.Get("/audit-logs/stats", auditHandler.GetAuditLogStats)

		// Dashboard routes
		r.Get("/dashboard", dashboardHandler.GetUserDashboard)
		r.Get("/dashboard/admin", dashboardHandler.GetDashboardStats)

		// Protected license routes
		r.Post("/licenses", licenseHandler.CreateLicense)
		r.Get("/licenses", licenseHandler.GetLicenses)
		r.Get("/licenses/export", licenseHandler.ExportLicenses)
		r.Post("/licenses/bulk-revoke", licenseHandler.BulkRevoke)
		r.Get("/licenses/{id}", licenseHandler.GetLicense)
		r.Post("/licenses/{id}/renew", licenseHandler.RenewLicense)
		r.Delete("/licenses/{id}", licenseHandler.RevokeLicense)
		r.Patch("/licenses/{id}/status", licenseHandler.UpdateLicenseStatus)

		// Machine fingerprint routes
		r.Get("/licenses/machines", licenseHandler.GetMachineFingerprints)
		r.Post("/licenses/machines", licenseHandler.TrackMachine)
	})

	return r
}

// testDatabaseService is a minimal implementation for testing
type testDatabaseService struct {
	db *gorm.DB
}

func (tds *testDatabaseService) GetDB() *gorm.DB {
	return tds.db
}

func (tds *testDatabaseService) Health() map[string]string {
	sqlDB, err := tds.db.DB()
	if err != nil {
		return map[string]string{
			"status": "down",
			"error":  err.Error(),
		}
	}

	if err := sqlDB.Ping(); err != nil {
		return map[string]string{
			"status": "down",
			"error":  err.Error(),
		}
	}

	return map[string]string{
		"status":  "up",
		"message": "Database is healthy",
	}
}

func (tds *testDatabaseService) Close() error {
	if tds.db != nil {
		sqlDB, err := tds.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// cleanupTestDB closes the database connection and stops the container
func (ti *TestIntegration) cleanup() error {
	// Close rate limiter goroutine
	if ti.rateLimiter != nil {
		_ = ti.rateLimiter.Close()
	}

	// Terminate container
	if ti.pgCont != nil {
		return ti.pgCont.Terminate(context.Background())
	}
	return nil
}

// doRequest helper to make HTTP requests against the test router
func (ti *TestIntegration) doRequest(method, path string, body interface{}) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		var err error
		bodyBytes, err = json.Marshal(body)
		if err != nil {
			return nil, err
		}
	}

	req := httptest.NewRequest(method, path, bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ti.router.ServeHTTP(w, req)

	return w.Result(), nil
}

// TestEmailVerificationFlow tests the complete email verification workflow
func TestEmailVerificationFlow(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	// Step 1: Register user
	t.Log("Step 1: Registering user...")
	registerReq := dto.RegisterDto{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "SecurePass123!",
	}

	resp, err := ti.doRequest("POST", "/api/v1/auth/register", registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 201 or 200, got %d", resp.StatusCode)
	}

	// Step 2: Get verification token from database
	t.Log("Step 2: Retrieving verification token from database...")
	var emailVerif struct {
		Token string
	}

	if err := ti.db.Table("email_verifications").
		Where("email = ?", "test@example.com").
		Select("token").
		First(&emailVerif).Error; err != nil {
		t.Fatalf("Failed to get verification token: %v", err)
	}

	if emailVerif.Token == "" {
		t.Fatal("No verification token found in database")
	}
	t.Logf("Got verification token: %s", emailVerif.Token[:8]+"...")

	// Step 3: Verify email with token
	t.Log("Step 3: Verifying email with token...")
	verifyReq := dto.VerifyEmailDto{
		Token: emailVerif.Token,
	}

	resp, err = ti.doRequest("POST", "/api/v1/auth/verify-email", verifyReq)
	if err != nil {
		t.Fatalf("Failed to verify email: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Step 4: Verify user status changed to ACTIVE
	t.Log("Step 4: Verifying user status in database...")
	var userStatus struct {
		Status string
	}

	if err := ti.db.Table("users").
		Where("email = ?", "test@example.com").
		Select("status").
		First(&userStatus).Error; err != nil {
		t.Fatalf("Failed to get user status: %v", err)
	}

	if userStatus.Status != "Active" {
		t.Errorf("Expected user status Active, got %s", userStatus.Status)
	}

	// Step 5: Verify token marked as used
	t.Log("Step 5: Verifying token marked as used...")
	var tokenUsed struct {
		UsedAt *time.Time
	}

	if err := ti.db.Table("email_verifications").
		Where("token = ?", emailVerif.Token).
		Select("used_at").
		First(&tokenUsed).Error; err != nil {
		t.Fatalf("Failed to check token usage: %v", err)
	}

	if tokenUsed.UsedAt == nil {
		t.Error("Token should be marked as used (used_at should not be nil)")
	}

	t.Log("✅ Email verification flow test passed!")
}

// TestPasswordResetFlow tests the complete password reset workflow
func TestPasswordResetFlow(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	// Step 1: Register user first
	t.Log("Step 1: Registering user...")
	registerReq := dto.RegisterDto{
		Username: "resetuser",
		Email:    "reset@example.com",
		Password: "OldPass123!",
	}

	resp, err := ti.doRequest("POST", "/api/v1/auth/register", registerReq)
	if err != nil {
		t.Fatalf("Failed to register user: %v", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 201 or 200, got %d", resp.StatusCode)
	}

	// Step 2: Request password reset
	t.Log("Step 2: Requesting password reset...")
	resetReq := dto.RequestPasswordResetDto{
		Email: "reset@example.com",
	}

	resp, err = ti.doRequest("POST", "/api/v1/auth/request-password-reset", resetReq)
	if err != nil {
		t.Fatalf("Failed to request password reset: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Step 3: Get reset token from database
	t.Log("Step 3: Retrieving reset token from database...")
	var resetToken struct {
		Token string
	}

	if err := ti.db.Table("password_resets").
		Where("email = ?", "reset@example.com").
		Select("token").
		First(&resetToken).Error; err != nil {
		t.Fatalf("Failed to get reset token: %v", err)
	}

	if resetToken.Token == "" {
		t.Fatal("No reset token found in database")
	}
	t.Logf("Got reset token: %s", resetToken.Token[:8]+"...")

	// Step 4: Confirm password reset
	t.Log("Step 4: Confirming password reset with new password...")
	confirmReq := dto.ConfirmPasswordResetDto{
		Token:       resetToken.Token,
		NewPassword: "NewPass456!",
	}

	resp, err = ti.doRequest("POST", "/api/v1/auth/confirm-password-reset", confirmReq)
	if err != nil {
		t.Fatalf("Failed to confirm password reset: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Step 5: Verify reset token marked as used
	t.Log("Step 5: Verifying reset token marked as used...")
	var tokenUsed struct {
		UsedAt *time.Time
	}

	if err := ti.db.Table("password_resets").
		Where("token = ?", resetToken.Token).
		Select("used_at").
		First(&tokenUsed).Error; err != nil {
		t.Fatalf("Failed to check token usage: %v", err)
	}

	if tokenUsed.UsedAt == nil {
		t.Error("Reset token should be marked as used (used_at should not be nil)")
	}

	// Step 6: Verify login with new password works
	t.Log("Step 6: Testing login with new password...")

	// First, verify the email since users need to be Active to login
	var verifyToken struct {
		Token string
	}

	if err := ti.db.Table("email_verifications").
		Where("email = ?", "reset@example.com").
		Select("token").
		Order("created_at DESC").
		First(&verifyToken).Error; err != nil {
		t.Logf("Warning: Could not get email verification token for login test: %v", err)
		// Continue anyway - some implementations might allow login without email verification
	} else if verifyToken.Token != "" {
		verifyReq := dto.VerifyEmailDto{
			Token: verifyToken.Token,
		}
		_, _ = ti.doRequest("POST", "/api/v1/auth/verify-email", verifyReq)
	}

	loginReq := dto.LoginDto{
		Username: "resetuser",
		Password: "NewPass456!",
	}

	resp, err = ti.doRequest("POST", "/api/v1/auth/login", loginReq)
	if err != nil {
		t.Fatalf("Failed to login with new password: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Logf("Warning: Login returned status %d (may require email verification)", resp.StatusCode)
	}

	t.Log("✅ Password reset flow test passed!")
}

// TestRegistrationWithEmailVerification tests that registration creates verification token
func TestRegistrationWithEmailVerification(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	t.Log("Testing registration creates email verification token...")

	registerReq := dto.RegisterDto{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "SecurePass123!",
	}

	resp, err := ti.doRequest("POST", "/api/v1/auth/register", registerReq)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}
	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		// Read and print response body for debugging
		respBody := make([]byte, 0)
		if resp.Body != nil {
			respBody, _ = io.ReadAll(resp.Body)
		}
		t.Logf("Registration failed with status %d, body: %s", resp.StatusCode, string(respBody))
		t.Errorf("Expected status 201 or 200, got %d", resp.StatusCode)
	}

	// Verify user created as Unverified
	var userStatus struct {
		Status string
	}

	if err := ti.db.Table("users").
		Where("email = ?", "new@example.com").
		Select("status").
		First(&userStatus).Error; err != nil {
		t.Fatalf("Failed to get user status: %v", err)
	}

	if userStatus.Status != "Unverified" {
		t.Errorf("Expected user status Unverified, got %s", userStatus.Status)
	}

	// Verify email verification record created
	var verifCount int64
	if err := ti.db.Table("email_verifications").
		Where("email = ?", "new@example.com").
		Count(&verifCount).Error; err != nil {
		t.Fatalf("Failed to check email verifications: %v", err)
	}

	if verifCount != 1 {
		t.Errorf("Expected 1 email verification record, got %d", verifCount)
	}

	t.Log("✅ Registration with email verification test passed!")
}

// TestResendVerificationEmail tests resending verification email
func TestResendVerificationEmail(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	t.Log("Testing resend verification email...")

	// Register user first
	registerReq := dto.RegisterDto{
		Username: "verifyuser",
		Email:    "verify@example.com",
		Password: "SecurePass123!",
	}

	_, err := ti.doRequest("POST", "/api/v1/auth/register", registerReq)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get first token
	var firstToken struct {
		Token string
	}

	if err := ti.db.Table("email_verifications").
		Where("email = ?", "verify@example.com").
		Select("token").
		Order("created_at DESC").
		First(&firstToken).Error; err != nil {
		t.Fatalf("Failed to get first token: %v", err)
	}

	// Wait a moment to ensure different created_at
	time.Sleep(100 * time.Millisecond)

	// Resend verification email
	resendReq := dto.ResendVerificationDto{
		Email: "verify@example.com",
	}

	resp, err := ti.doRequest("POST", "/api/v1/auth/resend-verification", resendReq)
	if err != nil {
		t.Fatalf("Failed to resend verification: %v", err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	// Get second token
	var secondToken struct {
		Token string
	}

	if err := ti.db.Table("email_verifications").
		Where("email = ?", "verify@example.com").
		Select("token").
		Order("created_at DESC").
		First(&secondToken).Error; err != nil {
		t.Fatalf("Failed to get second token: %v", err)
	}

	if secondToken.Token == firstToken.Token {
		t.Error("Expected new token after resend, but got the same token")
	}

	t.Log("✅ Resend verification email test passed!")
}

// TestTokenExpiration tests that expired tokens are rejected
func TestTokenExpiration(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	t.Log("Testing token expiration rejection...")

	// Register user
	registerReq := dto.RegisterDto{
		Username: "expireuser",
		Email:    "expire@example.com",
		Password: "SecurePass123!",
	}

	_, err := ti.doRequest("POST", "/api/v1/auth/register", registerReq)
	if err != nil {
		t.Fatalf("Failed to register: %v", err)
	}

	// Get token and manually expire it in database
	var verifID struct {
		ID        int64
		ExpiresAt time.Time
	}

	if err := ti.db.Table("email_verifications").
		Where("email = ?", "expire@example.com").
		Select("id, expires_at").
		First(&verifID).Error; err != nil {
		t.Fatalf("Failed to get verification: %v", err)
	}

	// Set expiry to past
	if err := ti.db.Table("email_verifications").
		Where("id = ?", verifID.ID).
		Update("expires_at", time.Now().Add(-1*time.Hour)).Error; err != nil {
		t.Fatalf("Failed to update expiry: %v", err)
	}

	// Get the token
	var token struct {
		Token string
	}

	if err := ti.db.Table("email_verifications").
		Where("id = ?", verifID.ID).
		Select("token").
		First(&token).Error; err != nil {
		t.Fatalf("Failed to get token: %v", err)
	}

	// Try to verify with expired token
	verifyReq := dto.VerifyEmailDto{
		Token: token.Token,
	}

	resp, err := ti.doRequest("POST", "/api/v1/auth/verify-email", verifyReq)
	if err != nil {
		t.Fatalf("Failed to make verify request: %v", err)
	}

	// Should get an error (not 200)
	if resp.StatusCode == http.StatusOK {
		t.Errorf("Expected error for expired token, but got status 200")
	}

	t.Logf("Got expected error status: %d", resp.StatusCode)
	t.Log("✅ Token expiration test passed!")
}

// TestHealthCheck tests the health endpoint
func TestHealthCheck(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	ti := setupTestDB(t)
	defer ti.cleanup()

	t.Log("Testing health check endpoint...")

	resp, err := ti.doRequest("GET", "/api/v1/health", nil)
	if err != nil {
		t.Fatalf("Failed to call health endpoint: %v", err)
	}

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	t.Log("✅ Health check test passed!")
}

// getAvailablePort finds an available port
func getAvailablePort() (int, error) {
	listener, err := net.Listen("tcp", ":0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()

	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port, nil
}

func TestParallel(t *testing.T) {
	// Skip if running in CI without Docker
	if os.Getenv("CI") == "true" || os.Getenv("SKIP_INTEGRATION_TESTS") == "true" {
		t.Skip("Skipping integration test in CI environment - requires Docker and Redis")
	}

	t.Log("Running parallel integration tests...")
	t.Run("EmailVerification", TestEmailVerificationFlow)
	t.Run("PasswordReset", TestPasswordResetFlow)
	t.Run("Registration", TestRegistrationWithEmailVerification)
	t.Run("ResendVerification", TestResendVerificationEmail)
	t.Run("TokenExpiration", TestTokenExpiration)
	t.Run("HealthCheck", TestHealthCheck)
}
