package middleware

import (
	"net/http"
)

// SecurityHeadersConfig holds security headers configuration
type SecurityHeadersConfig struct {
	// HSTS (HTTP Strict-Transport-Security)
	EnableHSTS            bool
	HSTSMaxAge            int // seconds
	HSTSIncludeSubdomains bool
	HSTSPreload           bool

	// Content Security Policy
	EnableCSP      bool
	CSPHeaderValue string

	// X-Frame-Options (Clickjacking protection)
	EnableFrameOptions bool
	FrameOptions       string // DENY, SAMEORIGIN, ALLOW-FROM

	// X-Content-Type-Options (MIME type sniffing protection)
	EnableContentTypeOptions bool

	// X-XSS-Protection (XSS protection)
	EnableXSSProtection bool

	// Referrer-Policy
	EnableReferrerPolicy bool
	ReferrerPolicy       string // no-referrer, strict-origin, etc.

	// Permissions-Policy
	EnablePermissionsPolicy bool
	PermissionsPolicyValue  string
}

// SecurityHeadersMiddleware adds security headers to all responses
func SecurityHeadersMiddleware(config *SecurityHeadersConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// HSTS - Force HTTPS
			if config.EnableHSTS {
				hstsHeader := "max-age=" + string(rune(config.HSTSMaxAge))
				if config.HSTSIncludeSubdomains {
					hstsHeader += "; includeSubDomains"
				}
				if config.HSTSPreload {
					hstsHeader += "; preload"
				}
				w.Header().Set("Strict-Transport-Security", hstsHeader)
			}

			// CSP - Content Security Policy
			if config.EnableCSP && config.CSPHeaderValue != "" {
				w.Header().Set("Content-Security-Policy", config.CSPHeaderValue)
				w.Header().Set("Content-Security-Policy-Report-Only", config.CSPHeaderValue)
			}

			// X-Frame-Options - Prevent clickjacking
			if config.EnableFrameOptions {
				w.Header().Set("X-Frame-Options", config.FrameOptions)
			}

			// X-Content-Type-Options - Prevent MIME type sniffing
			if config.EnableContentTypeOptions {
				w.Header().Set("X-Content-Type-Options", "nosniff")
			}

			// X-XSS-Protection - XSS protection header
			if config.EnableXSSProtection {
				w.Header().Set("X-XSS-Protection", "1; mode=block")
			}

			// Referrer-Policy
			if config.EnableReferrerPolicy {
				w.Header().Set("Referrer-Policy", config.ReferrerPolicy)
			}

			// Permissions-Policy (formerly Feature-Policy)
			if config.EnablePermissionsPolicy && config.PermissionsPolicyValue != "" {
				w.Header().Set("Permissions-Policy", config.PermissionsPolicyValue)
			}

			// Additional security headers
			w.Header().Set("X-Permitted-Cross-Domain-Policies", "none")
			w.Header().Set("X-Powered-By", "") // Remove server identification
			w.Header().Set("Server", "")       // Remove server identification

			next.ServeHTTP(w, r)
		})
	}
}

// DefaultSecurityHeadersConfig returns a secure-by-default configuration
func DefaultSecurityHeadersConfig() *SecurityHeadersConfig {
	return &SecurityHeadersConfig{
		// HSTS - 1 year in seconds
		EnableHSTS:            true,
		HSTSMaxAge:            31536000,
		HSTSIncludeSubdomains: true,
		HSTSPreload:           true,

		// CSP - Strict policy, customize as needed
		EnableCSP: true,
		CSPHeaderValue: "default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; " +
			"img-src 'self' data: https:; font-src 'self'; connect-src 'self'; " +
			"frame-ancestors 'none'; base-uri 'self'; form-action 'self';",

		// X-Frame-Options
		EnableFrameOptions: true,
		FrameOptions:       "DENY",

		// X-Content-Type-Options
		EnableContentTypeOptions: true,

		// X-XSS-Protection
		EnableXSSProtection: true,

		// Referrer-Policy
		EnableReferrerPolicy: true,
		ReferrerPolicy:       "strict-origin-when-cross-origin",

		// Permissions-Policy - Disable powerful features
		EnablePermissionsPolicy: true,
		PermissionsPolicyValue: "geolocation=(), " +
			"microphone=(), " +
			"camera=(), " +
			"payment=(), " +
			"usb=(), " +
			"magnetometer=(), " +
			"gyroscope=(), " +
			"accelerometer=()",
	}
}

// ProductionSecurityHeadersConfig returns a config suitable for production
func ProductionSecurityHeadersConfig() *SecurityHeadersConfig {
	cfg := DefaultSecurityHeadersConfig()
	cfg.HSTSMaxAge = 31536000 // 1 year
	cfg.HSTSIncludeSubdomains = true
	cfg.HSTSPreload = true
	return cfg
}

// DevelopmentSecurityHeadersConfig returns a config suitable for development
func DevelopmentSecurityHeadersConfig() *SecurityHeadersConfig {
	cfg := DefaultSecurityHeadersConfig()
	cfg.HSTSMaxAge = 3600   // 1 hour (revertible)
	cfg.HSTSPreload = false // Don't preload in development
	// More permissive CSP for development
	cfg.CSPHeaderValue = "default-src 'self' 'unsafe-inline' 'unsafe-eval'; " +
		"img-src 'self' data: https:; " +
		"font-src 'self' data:; " +
		"connect-src 'self' http: https: ws: wss:; " +
		"frame-ancestors 'self';"
	return cfg
}
