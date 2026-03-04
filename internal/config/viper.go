package config

import (
	"fmt"
	"log"
	"strings"

	"github.com/spf13/viper"
)

// RegisterViperConfig initializes and configures Viper for reading configurations
// from environment variables, config files, and default values
func RegisterViperConfig() *viper.Viper {
	v := viper.New()

	// Set configuration file paths and name
	v.SetConfigName(".env")
	v.SetConfigType("env")

	// Add multiple config search paths
	v.AddConfigPath(".") // Current directory
	v.AddConfigPath("./config")
	v.AddConfigPath("../config")

	// Enable reading from environment variables
	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// Set defaults for all configuration values
	setDefaults(v)

	// Try to read config file (not required)
	if err := v.ReadInConfig(); err != nil {
		// Config file not found; not critical, we'll use env vars and defaults
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			log.Printf("Warning: Error reading config file: %v\n", err)
		}
	}

	return v
}

// setDefaults sets the default values for all configuration keys
func setDefaults(v *viper.Viper) {
	// Server
	v.SetDefault("PORT", "8080")
	v.SetDefault("ENVIRONMENT", "development")
	v.SetDefault("APP_NAME", "License Management API")

	// JWT
	v.SetDefault("JWT_ACCESS_EXPIRY", "15m")
	v.SetDefault("JWT_REFRESH_EXPIRY", "7d")
	// JWT_SECRET and JWT_REFRESH_SECRET must be set manually (no defaults for security)

	// Database
	v.SetDefault("DB_HOST", "localhost")
	v.SetDefault("DB_PORT", "5432")
	v.SetDefault("DB_USER", "postgres")
	v.SetDefault("DB_NAME", "license_mgmt")
	// DB_PASSWORD should be set via environment variable

	// Redis
	v.SetDefault("REDIS_HOST", "localhost")
	v.SetDefault("REDIS_PORT", "6379")
	v.SetDefault("REDIS_DB", 0)
	// REDIS_PASSWORD optional

	// CORS
	v.SetDefault("CORS_ORIGINS", "http://localhost:3000,http://localhost:5173")

	// Admin seeding
	v.SetDefault("ADMIN_EMAIL", "")
	v.SetDefault("ADMIN_USERNAME", "")
	// ADMIN_PASSWORD should be set via environment variable

	// Email
	v.SetDefault("SMTP_HOST", "")
	v.SetDefault("SMTP_PORT", "587")
	v.SetDefault("SMTP_USERNAME", "")
	v.SetDefault("SMTP_PASSWORD", "")
	v.SetDefault("EMAIL_FROM", "noreply@example.com")
	v.SetDefault("FRONTEND_BASE_URL", "http://localhost:3000")
	v.SetDefault("EMAIL_USE_SSL", true)
	v.SetDefault("EMAIL_USE_CONSOLE", false)

	// Rate Limiting
	v.SetDefault("RATE_LIMIT_ENABLED", true)
	v.SetDefault("RATE_LIMIT_LOGIN_MAX_ATTEMPTS", 5)
	v.SetDefault("RATE_LIMIT_LOGIN_LOCKOUT_DURATION", "15m")
	v.SetDefault("RATE_LIMIT_LOGIN_RESET_WINDOW", "15m")
	v.SetDefault("RATE_LIMIT_REGISTER_MAX_ATTEMPTS", 3)
	v.SetDefault("RATE_LIMIT_REGISTER_LOCKOUT_DURATION", "1h")
	v.SetDefault("RATE_LIMIT_REGISTER_RESET_WINDOW", "24h")
	v.SetDefault("RATE_LIMIT_PASSWORD_RESET_MAX_ATTEMPTS", 5)
	v.SetDefault("RATE_LIMIT_PASSWORD_RESET_LOCKOUT_DURATION", "30m")
	v.SetDefault("RATE_LIMIT_PASSWORD_RESET_RESET_WINDOW", "24h")
	v.SetDefault("RATE_LIMIT_EMAIL_VERIFY_MAX_ATTEMPTS", 10)
	v.SetDefault("RATE_LIMIT_EMAIL_VERIFY_LOCKOUT_DURATION", "15m")
	v.SetDefault("RATE_LIMIT_EMAIL_VERIFY_RESET_WINDOW", "24h")
	v.SetDefault("RATE_LIMIT_LICENSE_ACTIVATION_MAX_ATTEMPTS", 10)
	v.SetDefault("RATE_LIMIT_LICENSE_ACTIVATION_LOCKOUT_DURATION", "1h")
	v.SetDefault("RATE_LIMIT_LICENSE_ACTIVATION_RESET_WINDOW", "24h")
	v.SetDefault("RATE_LIMIT_ENABLE_BACKOFF", true)
	v.SetDefault("RATE_LIMIT_BACKOFF_MULTIPLIER", 1.5)
	v.SetDefault("RATE_LIMIT_MAX_BACKOFF_MULTIPLIER", 10.0)

	// Security - HTTPS/TLS
	v.SetDefault("ENABLE_HTTPS", false) // Set to true in production
	v.SetDefault("TLS_CERT_FILE", "")
	v.SetDefault("TLS_KEY_FILE", "")
	v.SetDefault("ENABLE_HSTS", true)
	v.SetDefault("HSTS_MAX_AGE", "31536000") // 1 year in seconds

	// Security - CSRF Protection
	v.SetDefault("ENABLE_CSRF", true)
	v.SetDefault("CSRF_TOKEN_EXPIRY", "1h")

	// Security - Headers
	v.SetDefault("ENABLE_SECURITY_HEADERS", true)
	v.SetDefault("ENABLE_CSP", true)
	v.SetDefault("ENABLE_FRAME_OPTIONS", true)
	v.SetDefault("ENABLE_XSS_PROTECTION", true)
	v.SetDefault("ENABLE_CONTENT_TYPE_OPTIONS", true)
	v.SetDefault("ENABLE_REFERRER_POLICY", true)
	v.SetDefault("ENABLE_PERMISSIONS_POLICY", true)

	// Security - Audit & Logging
	v.SetDefault("ENABLE_AUDIT_LOGGING", true)
	v.SetDefault("ENABLE_PII_MASKING", true)
	v.SetDefault("AUDIT_LOG_RETENTION_DAYS", "90")
	v.SetDefault("SENSITIVE_FIELDS", "password,secret,token,apikey,credit_card")

	// Cache
	v.SetDefault("CACHE_TTL_SECONDS", 900) // 15 minutes
	v.SetDefault("CACHE_LICENSE_TTL", "15m")
	v.SetDefault("CACHE_USER_TTL", "10m")
	v.SetDefault("CACHE_AUDIT_TTL", "5m")

	// Logging
	v.SetDefault("LOG_LEVEL", "info")
	v.SetDefault("LOG_OUTPUT_FILE", "")
	v.SetDefault("LOG_FORMAT", "text") // text or json
}

// Config wraps Viper and provides convenient access methods
type Config struct {
	v *viper.Viper
}

// NewConfig creates a new Config instance with Viper
func NewConfig() *Config {
	return &Config{
		v: RegisterViperConfig(),
	}
}

// GetViper returns the underlying Viper instance for advanced usage
func (c *Config) GetViper() *viper.Viper {
	return c.v
}

// String retrieves a string configuration value
func (c *Config) String(key string) string {
	return c.v.GetString(key)
}

// Int retrieves an integer configuration value
func (c *Config) Int(key string) int {
	return c.v.GetInt(key)
}

// Bool retrieves a boolean configuration value
func (c *Config) Bool(key string) bool {
	return c.v.GetBool(key)
}

// Duration retrieves a duration configuration value
func (c *Config) Duration(key string) string {
	return c.v.GetString(key)
}

// IsSet checks if a configuration key is set
func (c *Config) IsSet(key string) bool {
	return c.v.IsSet(key)
}

// AllSettings returns all configuration settings
func (c *Config) AllSettings() map[string]interface{} {
	return c.v.AllSettings()
}

// Load config file from custom path
func (c *Config) LoadConfigFile(path string) error {
	c.v.SetConfigFile(path)
	return c.v.ReadInConfig()
}

// Validate checks that required configuration values are set
func (c *Config) Validate() error {
	// Check critical JWT secrets
	if !c.IsSet("JWT_SECRET") {
		return fmt.Errorf("JWT_SECRET is required")
	}
	if c.String("JWT_SECRET") == "" {
		return fmt.Errorf("JWT_SECRET cannot be empty")
	}
	if len(c.String("JWT_SECRET")) < 32 {
		return fmt.Errorf("JWT_SECRET must be at least 32 characters")
	}

	if !c.IsSet("JWT_REFRESH_SECRET") {
		return fmt.Errorf("JWT_REFRESH_SECRET is required")
	}
	if c.String("JWT_REFRESH_SECRET") == "" {
		return fmt.Errorf("JWT_REFRESH_SECRET cannot be empty")
	}
	if len(c.String("JWT_REFRESH_SECRET")) < 32 {
		return fmt.Errorf("JWT_REFRESH_SECRET must be at least 32 characters")
	}

	return nil
}

// LogConfiguration logs all configuration values (excluding sensitive data)
func (c *Config) LogConfiguration() {
	settings := c.v.AllSettings()

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Println("Configuration Loaded:")
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	sensitivKeys := map[string]bool{
		"jwt_secret": true, "jwt_refresh_secret": true,
		"db_password": true, "redis_password": true,
		"smtp_password": true, "admin_password": true,
	}

	for key, value := range settings {
		if sensitivKeys[strings.ToLower(key)] {
			log.Printf("✓ %s: *****(redacted)\n", key)
		} else {
			log.Printf("✓ %s: %v\n", key, value)
		}
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
}

// LoadConfigFromFile loads configuration from a specific file
func LoadConfigFromFile(filepath string) (*Config, error) {
	cfg := NewConfig()
	if err := cfg.LoadConfigFile(filepath); err != nil {
		return nil, err
	}
	return cfg, nil
}

// MustLoadConfig loads config and panics if it fails validation
func MustLoadConfig() *Config {
	cfg := NewConfig()
	if err := cfg.Validate(); err != nil {
		panic(fmt.Sprintf("Configuration validation failed: %v", err))
	}
	return cfg
}
