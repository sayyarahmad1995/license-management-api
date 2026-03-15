package config

import (
	"os"
	"strconv"
	"time"
)

// RateLimitEndpointConfig holds configuration for a specific endpoint's rate limiting
type RateLimitEndpointConfig struct {
	MaxAttempts           int
	LockoutDuration       time.Duration
	ResetWindow           time.Duration
	AuthenticatedMaxAttrs int // Different limit for authenticated users (0 = same as anonymous)
	Description           string
}

// RateLimitSystemConfig holds all rate limiting configurations
type RateLimitSystemConfig struct {
	Login                RateLimitEndpointConfig
	Register             RateLimitEndpointConfig
	PasswordReset        RateLimitEndpointConfig
	EmailVerification    RateLimitEndpointConfig
	LicenseActivation    RateLimitEndpointConfig
	GeneralAPI           RateLimitEndpointConfig
	EnableBackoff        bool
	BackoffMultiplier    int
	MaxBackoffMultiplier int
}

// LoadRateLimitConfig loads rate limit configuration from environment variables
func LoadRateLimitConfig() *RateLimitSystemConfig {
	return &RateLimitSystemConfig{
		Login: RateLimitEndpointConfig{
			MaxAttempts:           parseIntFromEnv("RATE_LIMIT_LOGIN_MAX_ATTEMPTS", 5),
			LockoutDuration:       parseDurationFromEnv("RATE_LIMIT_LOGIN_LOCKOUT_DURATION", "15 min"),
			ResetWindow:           parseDurationFromEnv("RATE_LIMIT_LOGIN_RESET_WINDOW", "15 min"),
			AuthenticatedMaxAttrs: parseIntFromEnv("RATE_LIMIT_LOGIN_AUTH_USER_MAX_ATTEMPTS", 10),
			Description:           "login attempts",
		},
		Register: RateLimitEndpointConfig{
			MaxAttempts:     parseIntFromEnv("RATE_LIMIT_REGISTER_MAX_ATTEMPTS", 3),
			LockoutDuration: parseDurationFromEnv("RATE_LIMIT_REGISTER_LOCKOUT_DURATION", "30 min"),
			ResetWindow:     parseDurationFromEnv("RATE_LIMIT_REGISTER_RESET_WINDOW", "1 hour"),
			Description:     "registration attempts",
		},
		PasswordReset: RateLimitEndpointConfig{
			MaxAttempts:     parseIntFromEnv("RATE_LIMIT_PASSWORD_RESET_MAX_ATTEMPTS", 3),
			LockoutDuration: parseDurationFromEnv("RATE_LIMIT_PASSWORD_RESET_LOCKOUT_DURATION", "30 min"),
			ResetWindow:     parseDurationFromEnv("RATE_LIMIT_PASSWORD_RESET_RESET_WINDOW", "1 hour"),
			Description:     "password reset attempts",
		},
		EmailVerification: RateLimitEndpointConfig{
			MaxAttempts:     parseIntFromEnv("RATE_LIMIT_EMAIL_VERIFY_MAX_ATTEMPTS", 5),
			LockoutDuration: parseDurationFromEnv("RATE_LIMIT_EMAIL_VERIFY_LOCKOUT_DURATION", "15 min"),
			ResetWindow:     parseDurationFromEnv("RATE_LIMIT_EMAIL_VERIFY_RESET_WINDOW", "15 min"),
			Description:     "email verification attempts",
		},
		LicenseActivation: RateLimitEndpointConfig{
			MaxAttempts:     parseIntFromEnv("RATE_LIMIT_LICENSE_ACTIVATION_MAX_ATTEMPTS", 10),
			LockoutDuration: parseDurationFromEnv("RATE_LIMIT_LICENSE_ACTIVATION_LOCKOUT_DURATION", "10 min"),
			ResetWindow:     parseDurationFromEnv("RATE_LIMIT_LICENSE_ACTIVATION_RESET_WINDOW", "1 hour"),
			Description:     "license activation attempts",
		},
		GeneralAPI: RateLimitEndpointConfig{
			MaxAttempts:     parseIntFromEnv("RATE_LIMIT_API_MAX_ATTEMPTS", 100),
			LockoutDuration: parseDurationFromEnv("RATE_LIMIT_API_LOCKOUT_DURATION", "1 min"),
			ResetWindow:     parseDurationFromEnv("RATE_LIMIT_API_RESET_WINDOW", "1 min"),
			Description:     "API requests",
		},
		EnableBackoff:        parseBoolFromEnv("RATE_LIMIT_ENABLE_BACKOFF", true),
		BackoffMultiplier:    parseIntFromEnv("RATE_LIMIT_BACKOFF_MULTIPLIER", 2),
		MaxBackoffMultiplier: parseIntFromEnv("RATE_LIMIT_MAX_BACKOFF_MULTIPLIER", 8),
	}
}

// parseIntFromEnv parses an integer from environment variable with fallback
func parseIntFromEnv(envKey string, defaultValue int) int {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	intVal, err := strconv.Atoi(value)
	if err != nil || intVal < 0 {
		return defaultValue
	}

	return intVal
}

// parseBoolFromEnv parses a boolean from environment variable with fallback
func parseBoolFromEnv(envKey string, defaultValue bool) bool {
	value := os.Getenv(envKey)
	if value == "" {
		return defaultValue
	}

	switch value {
	case "true", "1", "yes", "on":
		return true
	case "false", "0", "no", "off":
		return false
	default:
		return defaultValue
	}
}

// parseDurationFromEnv parses a duration from environment variable (delegates to cache_ttl parser)
func parseDurationFromEnv(envKey string, defaultValue string) time.Duration {
	value := os.Getenv(envKey)
	if value == "" {
		// Parse default value directly
		return mustParseDuration(defaultValue)
	}
	return mustParseDuration(value)
}

// mustParseDuration parses duration using the same flexible parser as cache TTL
func mustParseDuration(value string) time.Duration {
	// Try Go duration format first
	if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
		return duration
	}

	// Try human-readable format
	if duration, err := ParseHumanReadableDuration(value); err == nil {
		return duration
	}

	// Try numeric (minutes fallback)
	if minutes, err := strconv.Atoi(value); err == nil && minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}

	// Default to 15 minutes
	return 15 * time.Minute
}
