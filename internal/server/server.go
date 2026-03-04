package server

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"license-management-api/internal/config"
	"license-management-api/internal/database"
	"license-management-api/internal/handler"
	"license-management-api/internal/logger"
	"license-management-api/internal/middleware"
	"license-management-api/internal/repository"
	"license-management-api/internal/service"
	"license-management-api/pkg/utils"

	"github.com/go-chi/chi/v5"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// API version constants for clean, DRY route definitions
const (
	APIv1 = "/api/v1"
)

// Package-level server instance for graceful shutdown access
var serverInstance *Server

// RouteConfig holds configuration for a single route with optional rate limiting
type RouteConfig struct {
	Method  string
	Path    string
	Handler http.HandlerFunc
	Limiter *middleware.EnhancedRedisRateLimiter
}

// applyRouteWithOptionalRateLimit registers a route with optional rate limiting
func applyRouteWithOptionalRateLimit(r chi.Router, config RouteConfig) {
	method := config.Method
	path := config.Path
	handler := config.Handler

	if config.Limiter != nil {
		wrappedHandler := config.Limiter.RateLimitMiddleware(http.HandlerFunc(handler))
		switch method {
		case "POST":
			r.Post(path, func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, r)
			})
		case "GET":
			r.Get(path, func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, r)
			})
		case "PUT":
			r.Put(path, func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, r)
			})
		case "DELETE":
			r.Delete(path, func(w http.ResponseWriter, r *http.Request) {
				wrappedHandler.ServeHTTP(w, r)
			})
		}
	} else {
		switch method {
		case "POST":
			r.Post(path, handler)
		case "GET":
			r.Get(path, handler)
		case "PUT":
			r.Put(path, handler)
		case "DELETE":
			r.Delete(path, handler)
		}
	}
}

// ServiceDependencies holds all initialized services and repositories
type ServiceDependencies struct {
	// Repositories
	UserRepo       repository.IUserRepository
	LicenseRepo    repository.ILicenseRepository
	ActivationRepo repository.ILicenseActivationRepository
	AuditRepo      repository.IAuditLogRepository

	// Core services
	TokenSvc                  service.TokenService
	AuthSvc                   service.AuthService
	LicenseSvc                service.LicenseService
	AuditSvc                  service.AuditService
	TokenRevocationSvc        service.TokenRevocationService
	TokenRotationSvc          service.TokenRotationService
	MachineFingerprintSvc     service.MachineFingerprintService
	PaginationSvc             service.PaginationService
	DashboardStatsSvc         service.DashboardStatsService
	NotificationPreferenceSvc service.NotificationPreferenceService
	DataExportSvc             service.DataExportService
	BulkOperationSvc          service.BulkOperationService
	EmailSvc                  service.EmailService
	NotificationQueueSvc      service.NotificationQueueService
	ValidationSvc             service.ValidationService
	MetricsSvc                *service.MetricsService
	CircuitBreakerSvc         *service.CircuitBreakerService

	// Rate limiters
	LegacyRateLimiter        *middleware.RateLimiter
	LoginLimiter             *middleware.EnhancedRedisRateLimiter
	RegisterLimiter          *middleware.EnhancedRedisRateLimiter
	PasswordResetLimiter     *middleware.EnhancedRedisRateLimiter
	EmailVerifyLimiter       *middleware.EnhancedRedisRateLimiter
	LicenseActivationLimiter *middleware.EnhancedRedisRateLimiter

	// Handlers
	AuthHandler       *handler.AuthHandler
	LicenseHandler    *handler.LicenseHandler
	HealthHandler     *handler.HealthHandler
	UserHandler       *handler.UserHandler
	AuditHandler      *handler.AuditHandler
	DashboardHandler  *handler.DashboardHandler
	PermissionHandler *handler.PermissionHandler

	// Permission service
	PermissionSvc service.PermissionService
}

type Server struct {
	port                 int
	db                   database.Service
	cfg                  *config.AppConfig
	log                  *logger.Logger
	notificationQueueSvc service.NotificationQueueService
}

func NewServer() *http.Server {
	portStr := os.Getenv("PORT")
	if portStr == "" {
		portStr = "8080"
	}
	port, _ := strconv.Atoi(portStr)

	// Initialize Viper configuration management
	config.InitConfig()

	// Initialize configurations
	appCfg := config.LoadAppConfig()
	dbService := database.New()

	// Initialize structured logger
	appLog := logger.FromConfig(appCfg)
	appLog.Info("Initializing API Server", "port", port, "environment", appCfg.Env)

	// Seed admin user at startup
	adminEmail := config.GetConfigString("ADMIN_EMAIL")
	adminUsername := config.GetConfigString("ADMIN_USERNAME")
	adminPassword := config.GetConfigString("ADMIN_PASSWORD")

	if adminEmail != "" && adminUsername != "" && adminPassword != "" {
		if err := database.SeedAdminUser(dbService.GetDB(), adminEmail, adminUsername, adminPassword); err != nil {
			appLog.Warn("Failed to seed admin user", "error", err)
		}
	}

	server := &Server{
		port: port,
		db:   dbService,
		cfg:  appCfg,
		log:  appLog,
	}

	// Store server instance for graceful shutdown
	serverInstance = server

	// Declare Server config
	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", server.port),
		Handler:      server.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	appLog.Info("API Server configured successfully", "address", httpServer.Addr)
	return httpServer
}

// initializeDependencies initializes all services, repositories, and handlers
func (s *Server) initializeDependencies(userRepo repository.IUserRepository, licenseRepo repository.ILicenseRepository, activationRepo repository.ILicenseActivationRepository, auditRepo repository.IAuditLogRepository, legacyRL *middleware.RateLimiter, loginLimiter, registerLimiter, passwordResetLimiter, emailVerifyLimiter, licenseActivationLimiter *middleware.EnhancedRedisRateLimiter, redisClient interface{}) *ServiceDependencies {
	// Initialize core services
	tokenSvc := service.NewTokenService(s.cfg.JwtCfg)
	auditSvc := service.NewAuditService(auditRepo)

	// Load cache TTL configuration
	ttlCfg := config.LoadCacheTTLConfig()

	// Extract RedisClient for AuthService
	var rc *utils.RedisClient
	if redisClient != nil {
		if r, ok := redisClient.(*utils.RedisClient); ok {
			rc = r
		}
	}
	authSvc := service.NewAuthService(userRepo, tokenSvc, auditSvc, s.db.GetDB(), rc, ttlCfg)
	licenseSvc := service.NewLicenseService(licenseRepo, activationRepo, auditSvc)

	// Initialize token revocation service
	var tokenRevocationSvc service.TokenRevocationService
	if redisClient != nil {
		if rc, ok := redisClient.(*utils.RedisClient); ok {
			tokenRevocationSvc = service.NewTokenRevocationService(rc.Client())
		}
	}

	// Initialize remaining services
	tokenRotationSvc := service.NewTokenRotationService(tokenSvc, userRepo)
	machineFingerprintSvc := service.NewMachineFingerprintService(licenseRepo, activationRepo, auditSvc)
	paginationSvc := service.NewPaginationService()
	dashboardStatsSvc := service.NewDashboardStatsServiceWithActivations(userRepo, licenseRepo, activationRepo, auditRepo)
	notificationPreferenceSvc := service.NewNotificationPreferenceService(userRepo)
	dataExportSvc := service.NewDataExportService(userRepo, licenseRepo, activationRepo)
	bulkOperationSvc := service.NewBulkOperationService(licenseRepo, auditRepo)

	emailCfg := service.LoadEmailConfig()
	emailSvc := service.NewEmailService(emailCfg)

	notificationQueueSvc := service.NewNotificationQueueService()
	notificationQueueSvc.SetEmailService(emailSvc)
	metricsSvc := service.NewMetricsService()
	validationSvc := service.NewValidationService()
	circuitBreakerSvc := service.NewCircuitBreakerService()

	// Initialize cache and session cache services
	var sessionCacheSvc *service.SessionCacheService
	if rc != nil {
		cacheService := service.NewCacheService(rc, 15*time.Minute)
		sessionCacheSvc = service.NewSessionCacheService(cacheService, ttlCfg)
	}

	notificationQueueSvc.Start()

	// Store notification queue for cleanup during shutdown
	s.notificationQueueSvc = notificationQueueSvc

	// Initialize handlers
	var authHandler *handler.AuthHandler
	if loginLimiter != nil {
		authHandler = handler.NewAuthHandlerWithEnhancedRateLimiter(authSvc, tokenSvc, loginLimiter, tokenRevocationSvc, tokenRotationSvc, notificationPreferenceSvc, notificationQueueSvc, validationSvc, emailSvc, sessionCacheSvc)
	} else {
		authHandler = handler.NewAuthHandler(authSvc, tokenSvc, legacyRL, tokenRevocationSvc, tokenRotationSvc, notificationPreferenceSvc, notificationQueueSvc, validationSvc, emailSvc, sessionCacheSvc)
	}

	licenseHandler := handler.NewLicenseHandler(licenseSvc, licenseRepo, machineFingerprintSvc, paginationSvc, dataExportSvc, bulkOperationSvc)
	healthHandler := handler.NewHealthHandler(s.db, circuitBreakerSvc)
	userHandler := handler.NewUserHandler(userRepo, licenseRepo, paginationSvc, dataExportSvc)
	auditHandler := handler.NewAuditHandler(auditRepo)
	dashboardHandler := handler.NewDashboardHandler(userRepo, licenseRepo, activationRepo, auditRepo, dashboardStatsSvc)

	// Initialize permission service
	permissionSvc := service.NewPermissionService(userRepo)
	permissionHandler := handler.NewPermissionHandler(permissionSvc)

	// Return all dependencies
	return &ServiceDependencies{
		UserRepo:                  userRepo,
		LicenseRepo:               licenseRepo,
		ActivationRepo:            activationRepo,
		AuditRepo:                 auditRepo,
		TokenSvc:                  tokenSvc,
		AuthSvc:                   authSvc,
		LicenseSvc:                licenseSvc,
		AuditSvc:                  auditSvc,
		TokenRevocationSvc:        tokenRevocationSvc,
		TokenRotationSvc:          tokenRotationSvc,
		MachineFingerprintSvc:     machineFingerprintSvc,
		PaginationSvc:             paginationSvc,
		DashboardStatsSvc:         dashboardStatsSvc,
		NotificationPreferenceSvc: notificationPreferenceSvc,
		DataExportSvc:             dataExportSvc,
		BulkOperationSvc:          bulkOperationSvc,
		EmailSvc:                  emailSvc,
		NotificationQueueSvc:      notificationQueueSvc,
		ValidationSvc:             validationSvc,
		MetricsSvc:                metricsSvc,
		CircuitBreakerSvc:         circuitBreakerSvc,
		LegacyRateLimiter:         legacyRL,
		LoginLimiter:              loginLimiter,
		RegisterLimiter:           registerLimiter,
		PasswordResetLimiter:      passwordResetLimiter,
		EmailVerifyLimiter:        emailVerifyLimiter,
		LicenseActivationLimiter:  licenseActivationLimiter,
		AuthHandler:               authHandler,
		LicenseHandler:            licenseHandler,
		HealthHandler:             healthHandler,
		UserHandler:               userHandler,
		AuditHandler:              auditHandler,
		DashboardHandler:          dashboardHandler,
		PermissionHandler:         permissionHandler,
		PermissionSvc:             permissionSvc,
	}
}

func (s *Server) RegisterRoutes() http.Handler {
	// Initialize repositories
	userRepo := repository.NewUserRepository(s.db.GetDB())
	licenseRepo := repository.NewLicenseRepository(s.db.GetDB())
	activationRepo := repository.NewLicenseActivationRepository(s.db.GetDB())
	auditRepo := repository.NewAuditLogRepository(s.db.GetDB())

	// Initialize Redis client
	ctx := context.Background()
	redisClient, err := utils.NewRedisClient(ctx)
	if err != nil {
		s.log.Warn("Redis connection failed. Falling back to in-memory rate limiter.",
			"error", err)
		// Fallback to in-memory rate limiter if Redis is unavailable
		rateLimiter := middleware.NewRateLimiter(5, 15*time.Minute, 15*time.Minute)
		deps := s.initializeDependencies(userRepo, licenseRepo, activationRepo, auditRepo, rateLimiter, nil, nil, nil, nil, nil, nil)
		return s.setupRoutes(deps)
	}

	s.log.Info("Redis connection established successfully")

	// Load rate limit configuration from environment
	rateLimitCfg := config.LoadRateLimitConfig()

	// Initialize enhanced Redis-based rate limiters for different endpoints
	loginLimiter := middleware.NewEnhancedRedisRateLimiter(redisClient, middleware.RateLimitConfig{
		MaxAttempts:           rateLimitCfg.Login.MaxAttempts,
		LockoutDuration:       rateLimitCfg.Login.LockoutDuration,
		ResetWindow:           rateLimitCfg.Login.ResetWindow,
		AuthenticatedMaxAttrs: rateLimitCfg.Login.AuthenticatedMaxAttrs,
		KeyPrefix:             "ratelimit:login",
		Description:           rateLimitCfg.Login.Description,
		EnableBackoff:         rateLimitCfg.EnableBackoff,
		BackoffMultiplier:     rateLimitCfg.BackoffMultiplier,
		MaxBackoffMultiplier:  rateLimitCfg.MaxBackoffMultiplier,
	})

	registerLimiter := middleware.NewEnhancedRedisRateLimiter(redisClient, middleware.RateLimitConfig{
		MaxAttempts:          rateLimitCfg.Register.MaxAttempts,
		LockoutDuration:      rateLimitCfg.Register.LockoutDuration,
		ResetWindow:          rateLimitCfg.Register.ResetWindow,
		KeyPrefix:            "ratelimit:register",
		Description:          rateLimitCfg.Register.Description,
		EnableBackoff:        rateLimitCfg.EnableBackoff,
		BackoffMultiplier:    rateLimitCfg.BackoffMultiplier,
		MaxBackoffMultiplier: rateLimitCfg.MaxBackoffMultiplier,
	})

	passwordResetLimiter := middleware.NewEnhancedRedisRateLimiter(redisClient, middleware.RateLimitConfig{
		MaxAttempts:          rateLimitCfg.PasswordReset.MaxAttempts,
		LockoutDuration:      rateLimitCfg.PasswordReset.LockoutDuration,
		ResetWindow:          rateLimitCfg.PasswordReset.ResetWindow,
		KeyPrefix:            "ratelimit:password_reset",
		Description:          rateLimitCfg.PasswordReset.Description,
		EnableBackoff:        rateLimitCfg.EnableBackoff,
		BackoffMultiplier:    rateLimitCfg.BackoffMultiplier,
		MaxBackoffMultiplier: rateLimitCfg.MaxBackoffMultiplier,
	})

	emailVerifyLimiter := middleware.NewEnhancedRedisRateLimiter(redisClient, middleware.RateLimitConfig{
		MaxAttempts:          rateLimitCfg.EmailVerification.MaxAttempts,
		LockoutDuration:      rateLimitCfg.EmailVerification.LockoutDuration,
		ResetWindow:          rateLimitCfg.EmailVerification.ResetWindow,
		KeyPrefix:            "ratelimit:email_verify",
		Description:          rateLimitCfg.EmailVerification.Description,
		EnableBackoff:        rateLimitCfg.EnableBackoff,
		BackoffMultiplier:    rateLimitCfg.BackoffMultiplier,
		MaxBackoffMultiplier: rateLimitCfg.MaxBackoffMultiplier,
	})

	licenseActivationLimiter := middleware.NewEnhancedRedisRateLimiter(redisClient, middleware.RateLimitConfig{
		MaxAttempts:          rateLimitCfg.LicenseActivation.MaxAttempts,
		LockoutDuration:      rateLimitCfg.LicenseActivation.LockoutDuration,
		ResetWindow:          rateLimitCfg.LicenseActivation.ResetWindow,
		KeyPrefix:            "ratelimit:license_activation",
		Description:          rateLimitCfg.LicenseActivation.Description,
		EnableBackoff:        rateLimitCfg.EnableBackoff,
		BackoffMultiplier:    rateLimitCfg.BackoffMultiplier,
		MaxBackoffMultiplier: rateLimitCfg.MaxBackoffMultiplier,
	})

	// Initialize dependencies using factory method
	deps := s.initializeDependencies(userRepo, licenseRepo, activationRepo, auditRepo, nil, loginLimiter, registerLimiter, passwordResetLimiter, emailVerifyLimiter, licenseActivationLimiter, redisClient)

	return s.setupRoutes(deps)
}

// setupRoutes initializes all routes with the given services
func (s *Server) setupRoutes(deps *ServiceDependencies) http.Handler {

	// Setup router
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.CorrelationIDMiddleware)
	r.Use(middleware.LoggingMiddleware(s.log))
	r.Use(middleware.ErrorLoggingMiddleware(s.log))
	r.Use(middleware.ErrorHandlerMiddleware)
	r.Use(middleware.MetricsMiddleware(deps.MetricsSvc))

	// Security headers middleware
	enableSecurityHeaders := config.GetConfigString("ENABLE_SECURITY_HEADERS")
	if enableSecurityHeaders == "true" || enableSecurityHeaders == "" {
		env := s.cfg.Env
		var securityConfig *middleware.SecurityHeadersConfig
		if env == "production" {
			securityConfig = middleware.ProductionSecurityHeadersConfig()
		} else {
			securityConfig = middleware.DevelopmentSecurityHeadersConfig()
		}
		r.Use(middleware.SecurityHeadersMiddleware(securityConfig))
		s.log.Info("Security headers enabled", "environment", env)
	}

	// CSRF protection middleware
	var csrfConfig *middleware.CSRFConfig
	enableCSRF := config.GetConfigString("ENABLE_CSRF")
	if enableCSRF == "true" || enableCSRF == "" {
		csrfConfig = middleware.DefaultCSRFConfig()
		csrfConfig.SkipURLPatterns = []string{"/health", "/livez", "/readyz", "/startup", "/metrics", "/api/v1/auth/login", "/api/v1/auth/register", "/csrf-token"}
		r.Use(middleware.CSRFMiddleware(csrfConfig))
		s.log.Info("CSRF protection enabled")
	}

	// CORS middleware
	corsOriginsStr := config.GetConfigString("CORS_ORIGINS")
	corsOrigins := middleware.ParseCORSOrigins(corsOriginsStr)
	if len(corsOrigins) > 0 {
		corsConfig := &middleware.CORSConfig{
			AllowedOrigins: corsOrigins,
		}
		r.Use(middleware.CORSMiddleware(corsConfig))
		s.log.Info("CORS enabled for origins", "origins", corsOrigins)
	} else {
		s.log.Warn("CORS not configured - no origins allowed for cross-origin requests")
	}

	// CSRF token endpoint (public, before all other routes)
	if csrfConfig != nil {
		r.Get("/csrf-token", middleware.GetCSRFTokenHandlerFunc(csrfConfig.Store))
	}

	// All API routes under /api/v1
	r.Route(APIv1, func(r chi.Router) {
		// ===== AUTH ROUTES (combined public + protected) =====
		r.Route("/auth", func(r chi.Router) {
			// PUBLIC auth endpoints
			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/register",
				Handler: deps.AuthHandler.Register,
				Limiter: deps.RegisterLimiter,
			})

			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/login",
				Handler: deps.AuthHandler.Login,
				Limiter: deps.LoginLimiter,
			})

			r.Post("/refresh", deps.AuthHandler.RefreshToken)
			r.Post("/rotate", deps.AuthHandler.RotateToken)
			r.Post("/logout", deps.AuthHandler.Logout)
			r.Get("/sessions", deps.AuthHandler.ListActiveSessions)
			r.Post("/logout-all-others", deps.AuthHandler.LogoutAllOtherSessions)

			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/verify-email",
				Handler: deps.AuthHandler.VerifyEmail,
				Limiter: deps.EmailVerifyLimiter,
			})

			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/resend-verification",
				Handler: deps.AuthHandler.ResendVerificationEmail,
				Limiter: deps.EmailVerifyLimiter,
			})

			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/request-password-reset",
				Handler: deps.AuthHandler.RequestPasswordReset,
				Limiter: deps.PasswordResetLimiter,
			})

			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/confirm-password-reset",
				Handler: deps.AuthHandler.ConfirmPasswordReset,
				Limiter: deps.PasswordResetLimiter,
			})

			r.Post("/revoke", deps.AuthHandler.Revoke)

			// PROTECTED auth endpoints (profile, notifications)
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuthMiddleware(deps.TokenSvc))

				r.Get("/me", deps.AuthHandler.GetMe)
				r.Put("/profile", deps.AuthHandler.UpdateProfile)
				r.Get("/notifications", deps.AuthHandler.GetNotifications)
				r.Put("/notifications", deps.AuthHandler.UpdateNotifications)

				// Email notification routes
				r.Post("/send-email", deps.AuthHandler.SendEmailNotification)
				r.Get("/email-templates", deps.AuthHandler.GetEmailTemplates)
				r.Post("/test-email", deps.AuthHandler.TestEmailSend)
				r.Get("/email-queue-status", deps.AuthHandler.GetEmailQueueStatus)
			})
		})

		// ===== LICENSE ROUTES (combined public + protected) =====
		r.Route("/licenses", func(r chi.Router) {
			// PUBLIC license endpoints
			applyRouteWithOptionalRateLimit(r, RouteConfig{
				Method:  "POST",
				Path:    "/activate",
				Handler: deps.LicenseHandler.ActivateLicense,
				Limiter: deps.LicenseActivationLimiter,
			})

			r.Post("/validate", deps.LicenseHandler.ValidateLicense)
			r.Post("/deactivate", deps.LicenseHandler.DeactivateLicense)
			r.Post("/heartbeat", deps.LicenseHandler.Heartbeat)

			// PROTECTED license endpoints (admin)
			r.Group(func(r chi.Router) {
				r.Use(middleware.AuthMiddleware(deps.TokenSvc))

				r.Post("/", deps.LicenseHandler.CreateLicense)
				r.Get("/", deps.LicenseHandler.GetLicenses)
				r.Get("/export", deps.LicenseHandler.ExportLicenses)
				r.Post("/bulk-revoke", deps.LicenseHandler.BulkRevoke)
				r.Post("/bulk-revoke-async", deps.LicenseHandler.BulkRevokeAsync)
				r.Get("/bulk-jobs/{jobId}", deps.LicenseHandler.GetBulkJobStatus)
				r.Post("/bulk-jobs/{jobId}/cancel", deps.LicenseHandler.CancelBulkJob)
				r.Get("/{id}", deps.LicenseHandler.GetLicense)
				r.Post("/{id}/renew", deps.LicenseHandler.RenewLicense)
				r.Delete("/{id}", deps.LicenseHandler.RevokeLicense)
				r.Patch("/{id}/status", deps.LicenseHandler.UpdateLicenseStatus)

				// Machine fingerprint routes
				r.Get("/machines", deps.LicenseHandler.GetMachineFingerprints)
				r.Post("/machines", deps.LicenseHandler.TrackMachine)
			})
		})

		// ===== HEALTH ENDPOINT (PUBLIC) =====
		r.Get("/health", deps.HealthHandler.Health)

		// ===== OTHER PROTECTED ROUTES =====
		r.Group(func(r chi.Router) {
			r.Use(middleware.AuthMiddleware(deps.TokenSvc))

			// User routes
			r.Route("/users", func(r chi.Router) {
				r.Get("/", deps.UserHandler.GetUsers)
				r.Get("/export", deps.UserHandler.ExportUsers)
				r.Get("/{id}", deps.UserHandler.GetUser)
				r.Get("/by-license/{key}", deps.UserHandler.GetUserByLicenseKey)
				r.Put("/{id}", deps.UserHandler.UpdateUser)
				r.Patch("/{id}/role", deps.UserHandler.UpdateUserRole)
				r.Patch("/{id}/status", deps.UserHandler.UpdateUserStatus)
				r.Delete("/{id}", deps.UserHandler.DeleteUser)
			})

			// Audit log routes
			r.Route("/audit-logs", func(r chi.Router) {
				r.Get("/", deps.AuditHandler.GetAuditLogs)
				r.Get("/user", deps.AuditHandler.GetAuditLogsByUser)
				r.Get("/stats", deps.AuditHandler.GetAuditLogStats)
			})

			// Dashboard routes
			r.Route("/dashboard", func(r chi.Router) {
				r.Get("/", deps.DashboardHandler.GetUserDashboard)
				r.Get("/admin", deps.DashboardHandler.GetDashboardStats)
				r.Get("/stats", deps.DashboardHandler.GetDashboardStats)
				r.Get("/forecast", deps.DashboardHandler.GetLicenseExpirationForecast)
				r.Get("/activity-timeline", deps.DashboardHandler.GetActivityTimeline)
				r.Get("/analytics", deps.DashboardHandler.GetUsageAnalytics)
				r.Get("/enhanced", deps.DashboardHandler.GetEnhancedDashboardStats)
			})

			// Permission routes (admin-only)
			r.Route("/permissions", func(r chi.Router) {
				r.Use(middleware.AdminOnlyMiddleware())

				r.Get("/users/{userId}", deps.PermissionHandler.GetUserPermissions)
				r.Get("/roles/{role}", deps.PermissionHandler.GetRolePermissions)
				r.Post("/grant", deps.PermissionHandler.GrantPermission)
				r.Post("/revoke", deps.PermissionHandler.RevokePermission)
				r.Post("/roles", deps.PermissionHandler.SetRolePermissions)
				r.Post("/users/{userId}/reset", deps.PermissionHandler.ResetUserPermissions)
				r.Post("/check", deps.PermissionHandler.CheckPermission)
			})

			r.Get("/stats", deps.DashboardHandler.GetDashboardStats)
		})
	})

	// Metrics endpoint - Prometheus
	r.Get("/metrics", func(w http.ResponseWriter, r *http.Request) {
		// Use Prometheus promhttp handler to serve metrics
		// For now, return simple JSON response with queue info
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":     "metrics collection active",
			"queue_size": deps.NotificationQueueSvc.GetQueueSize(),
		})
	})

	// Backward compatibility for old health endpoint
	r.Get("/health", deps.HealthHandler.Health)

	// Kubernetes probe endpoints
	r.Get("/livez", deps.HealthHandler.Liveness)   // Liveness probe: service is running
	r.Get("/readyz", deps.HealthHandler.Readiness) // Readiness probe: ready for traffic
	r.Get("/startup", deps.HealthHandler.Startup)  // Startup probe: bootstrap complete

	// Metrics endpoint for Prometheus scraping
	r.Handle("/metrics", promhttp.Handler())

	// Swagger documentation routes
	r.Get("/swagger", swaggerUIHandler)
	r.Get("/swagger/swagger.json", swaggerJSONHandler)
	r.Get("/api/docs", swaggerUIHandler)

	return r
}

// swaggerUIHandler serves the Swagger UI HTML
func swaggerUIHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(`
		<!DOCTYPE html>
		<html>
		<head>
			<title>License Management API</title>
			<meta charset="utf-8"/>
			<meta name="viewport" content="width=device-width, initial-scale=1">
			<link rel="stylesheet" href="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui.css">
			<style>
				html{
					box-sizing: border-box;
					overflow: -moz-scrollbars-vertical;
					overflow-y: scroll;
				}
				*,
				*:before,
				*:after{
					box-sizing: inherit;
				}
				body{
					margin:0;
					background: #fafafa;
				}
			</style>
		</head>
		<body>
			<div id="swagger-ui"></div>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-bundle.js"></script>
			<script src="https://cdn.jsdelivr.net/npm/swagger-ui-dist@3/swagger-ui-standalone-preset.js"></script>
			<script>
				window.onload = function() {
					const ui = SwaggerUIBundle({
						url: "/swagger/swagger.json",
						dom_id: '#swagger-ui',
						deepLinking: true,
						presets: [
							SwaggerUIBundle.presets.apis,
							SwaggerUIStandalonePreset
						],
						plugins: [
							SwaggerUIBundle.plugins.DownloadUrl
						],
						layout: "BaseLayout"
					})
					window.ui = ui
				}
			</script>
		</body>
		</html>
	`))
}

// swaggerJSONHandler serves the Swagger specification JSON
func swaggerJSONHandler(w http.ResponseWriter, r *http.Request) {
	// Read the generated swagger.json file
	swaggerJSON, err := os.ReadFile("./docs/swagger.json")
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":"swagger.json not found"}`))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(swaggerJSON)
}

// Shutdown gracefully stops the server and cleans up resources
func Shutdown(ctx context.Context) error {
	if serverInstance == nil {
		return nil
	}

	serverInstance.log.Info("Starting graceful shutdown...")

	// Stop notification queue processor
	if serverInstance.notificationQueueSvc != nil {
		serverInstance.log.Info("Stopping notification queue...")
		serverInstance.notificationQueueSvc.Stop()
	}

	// Close database connections
	if serverInstance.db != nil {
		serverInstance.log.Info("Closing database connections...")
		if err := serverInstance.db.Close(); err != nil {
			serverInstance.log.Error("Failed to close database", err)
			return err
		}
	}

	serverInstance.log.Info("Graceful shutdown complete")
	return nil
}
