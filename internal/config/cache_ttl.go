package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"
)

// CacheTTLConfig holds all cache TTL configurations
type CacheTTLConfig struct {
	EmailVerificationTokenTTL time.Duration
	PasswordResetTokenTTL     time.Duration
	LicenseActivationTTL      time.Duration
	DashboardStatsTTL         time.Duration
	UserLicenseCacheTTL       time.Duration
	UserSessionTTL            time.Duration
	AccessTokenTTL            time.Duration
	RefreshTokenTTL           time.Duration
	LastSeenActivityTTL       time.Duration
}

// LoadCacheTTLConfig loads cache TTL configuration from environment variables
func LoadCacheTTLConfig() *CacheTTLConfig {
	return &CacheTTLConfig{
		EmailVerificationTokenTTL: parseTTLFromEnv("CACHE_TTL_EMAIL_VERIFICATION", 24*60), // 24 hours default
		PasswordResetTokenTTL:     parseTTLFromEnv("CACHE_TTL_PASSWORD_RESET", 60),        // 1 hour default
		LicenseActivationTTL:      parseTTLFromEnv("CACHE_TTL_LICENSE_ACTIVATION", 15),    // 15 minutes default
		DashboardStatsTTL:         parseTTLFromEnv("CACHE_TTL_DASHBOARD_STATS", 5),        // 5 minutes default
		UserLicenseCacheTTL:       parseTTLFromEnv("CACHE_TTL_USER_LICENSES", 10),         // 10 minutes default
		UserSessionTTL:            parseTTLFromEnv("CACHE_TTL_USER_SESSION", 24*60),       // 24 hours default
		AccessTokenTTL:            parseJWTAccessExpiry(),                                 // Uses JWT_ACCESS_EXPIRY
		RefreshTokenTTL:           parseJWTRefreshExpiry(),                                // Uses JWT_REFRESH_EXPIRY
		LastSeenActivityTTL:       parseTTLFromEnv("CACHE_TTL_LAST_SEEN_ACTIVITY", 5),     // 5 minutes default
	}
}

// parseTTLFromEnv parses a TTL value from environment variable with flexible format support
// Supports formats like:
//   - "15m" or "15 min" or "15 minutes" (minutes)
//   - "300s" or "300 sec" or "300 seconds" (seconds)
//   - "2h" or "2 hours" (hours)
//   - "7days" or "7 days" (days)
//   - "1440" (numeric, treated as minutes for backwards compatibility)
//
// Falls back to defaultMinutes if not set or invalid
func parseTTLFromEnv(envKey string, defaultMinutes int) time.Duration {
	value := os.Getenv(envKey)
	if value == "" {
		return time.Duration(defaultMinutes) * time.Minute
	}

	value = strings.TrimSpace(value)

	// Try parsing as Go duration format first (e.g., "15m", "300s", "2h")
	if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
		return duration
	}

	// Try parsing human-readable format (e.g., "15 min", "7 days", "300 sec")
	if duration, err := ParseHumanReadableDuration(value); err == nil {
		return duration
	}

	// Try parsing as plain number (backwards compatibility, treat as minutes)
	if minutes, err := strconv.Atoi(value); err == nil && minutes > 0 {
		return time.Duration(minutes) * time.Minute
	}

	// Fallback to default
	return time.Duration(defaultMinutes) * time.Minute
}

// parseJWTAccessExpiry reads JWT_ACCESS_EXPIRY and falls back to CACHE_TTL_ACCESS_TOKEN for backwards compatibility
func parseJWTAccessExpiry() time.Duration {
	// First try JWT_ACCESS_EXPIRY (primary source)
	if value := os.Getenv("JWT_ACCESS_EXPIRY"); value != "" {
		if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
			return duration
		}
		if duration, err := ParseHumanReadableDuration(value); err == nil {
			return duration
		}
	}
	// Fallback to 15 minutes
	return 15 * time.Minute
}

// parseJWTRefreshExpiry reads JWT_REFRESH_EXPIRY and falls back to CACHE_TTL_REFRESH_TOKEN for backwards compatibility
func parseJWTRefreshExpiry() time.Duration {
	// First try JWT_REFRESH_EXPIRY (primary source)
	if value := os.Getenv("JWT_REFRESH_EXPIRY"); value != "" {
		if duration, err := time.ParseDuration(value); err == nil && duration > 0 {
			return duration
		}
		if duration, err := ParseHumanReadableDuration(value); err == nil {
			return duration
		}
	}
	// Fallback to 7 days
	return 7 * 24 * time.Hour
}

// ParseHumanReadableDuration parses human-readable time formats like "15 min", "7 days", "300 sec"
func ParseHumanReadableDuration(value string) (time.Duration, error) {
	value = strings.ToLower(strings.TrimSpace(value))

	// Split into number and unit parts
	parts := strings.Fields(value)
	if len(parts) < 2 {
		// Try without space: "15min", "7days", "300sec"
		for i, char := range value {
			if char >= 'a' && char <= 'z' {
				numberPart := value[:i]
				unitPart := value[i:]

				num, err := strconv.ParseInt(numberPart, 10, 64)
				if err != nil || num <= 0 {
					return 0, fmt.Errorf("invalid duration format")
				}

				return ParseDurationUnit(int(num), unitPart)
			}
		}
		return 0, fmt.Errorf("invalid duration format")
	}

	// Parse with space: "15 min", "7 days", "300 sec"
	num, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil || num <= 0 {
		return 0, fmt.Errorf("invalid duration number")
	}

	unitPart := strings.Join(parts[1:], " ")
	return ParseDurationUnit(int(num), unitPart)
}

// ParseDurationUnit converts a number and unit string to time.Duration
func ParseDurationUnit(num int, unit string) (time.Duration, error) {
	unit = strings.ToLower(strings.TrimSpace(unit))

	switch {
	case strings.HasPrefix(unit, "nanosec"):
		return time.Duration(num) * time.Nanosecond, nil
	case strings.HasPrefix(unit, "microsec") || strings.HasPrefix(unit, "µs"):
		return time.Duration(num) * time.Microsecond, nil
	case strings.HasPrefix(unit, "millisec") || strings.HasPrefix(unit, "ms"):
		return time.Duration(num) * time.Millisecond, nil
	case strings.HasPrefix(unit, "sec"):
		return time.Duration(num) * time.Second, nil
	case strings.HasPrefix(unit, "min"):
		return time.Duration(num) * time.Minute, nil
	case strings.HasPrefix(unit, "hour") || strings.HasPrefix(unit, "h"):
		return time.Duration(num) * time.Hour, nil
	case strings.HasPrefix(unit, "day") || unit == "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case strings.HasPrefix(unit, "week"):
		return time.Duration(num) * 7 * 24 * time.Hour, nil
	default:
		return 0, fmt.Errorf("unknown duration unit: %s", unit)
	}
}
