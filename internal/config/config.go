package config

import (
	"fmt"
	"strconv"
	"time"

	"github.com/spf13/viper"
)

// Global Viper instance
var globalViper *viper.Viper

// InitConfig initializes the global Viper configuration
func InitConfig() {
	globalViper = RegisterViperConfig()
}

// GetConfigValue retrieves a configuration value from Viper
func GetConfigValue(key string) interface{} {
	if globalViper == nil {
		InitConfig()
	}
	return globalViper.Get(key)
}

// GetConfigString retrieves a string config value
func GetConfigString(key string) string {
	if globalViper == nil {
		InitConfig()
	}
	return globalViper.GetString(key)
}

// GetConfigInt retrieves an int config value
func GetConfigInt(key string) int {
	if globalViper == nil {
		InitConfig()
	}
	return globalViper.GetInt(key)
}

// GetConfigBool retrieves a bool config value
func GetConfigBool(key string) bool {
	if globalViper == nil {
		InitConfig()
	}
	return globalViper.GetBool(key)
}

type JwtConfig struct {
	SecretKey          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
	RefreshTokenSecret string
}

func LoadJwtConfig() *JwtConfig {
	// Validate and load access token expiry using flexible duration parser
	accessExpiryStr := GetConfigString("JWT_ACCESS_EXPIRY")
	if accessExpiryStr == "" {
		accessExpiryStr = "15m" // default
	}
	var accessExpiry time.Duration
	if dur, err := time.ParseDuration(accessExpiryStr); err == nil && dur > 0 {
		accessExpiry = dur
	} else if dur, err := ParseHumanReadableDuration(accessExpiryStr); err == nil {
		accessExpiry = dur
	} else if val, err := strconv.Atoi(accessExpiryStr); err == nil && val > 0 {
		// Fallback to numeric (minutes)
		accessExpiry = time.Duration(val) * time.Minute
	} else {
		panic("FATAL: JWT_ACCESS_EXPIRY must be a valid duration (e.g., '15m', '15 min', '900s', or '15' for minutes)")
	}

	// Validate and load refresh token expiry using flexible duration parser
	refreshExpiryStr := GetConfigString("JWT_REFRESH_EXPIRY")
	if refreshExpiryStr == "" {
		refreshExpiryStr = "7d" // default
	}
	var refreshExpiry time.Duration
	if dur, err := time.ParseDuration(refreshExpiryStr); err == nil && dur > 0 {
		refreshExpiry = dur
	} else if dur, err := ParseHumanReadableDuration(refreshExpiryStr); err == nil {
		refreshExpiry = dur
	} else if val, err := strconv.Atoi(refreshExpiryStr); err == nil && val > 0 {
		// Fallback to numeric (days)
		refreshExpiry = time.Duration(val) * 24 * time.Hour
	} else {
		panic("FATAL: JWT_REFRESH_EXPIRY must be a valid duration (e.g., '7d', '7 days', '168h', or '7' for days)")
	}

	// CRITICAL: JWT_SECRET must be set - no defaults allowed
	secret := GetConfigString("JWT_SECRET")
	if secret == "" {
		panic("FATAL: JWT_SECRET environment variable must be set (minimum 32 characters)")
	}
	if len(secret) < 32 {
		panic("FATAL: JWT_SECRET must be at least 32 characters long for security")
	}

	// CRITICAL: JWT_REFRESH_SECRET must be set - no defaults allowed
	refreshSecret := GetConfigString("JWT_REFRESH_SECRET")
	if refreshSecret == "" {
		panic("FATAL: JWT_REFRESH_SECRET environment variable must be set (minimum 32 characters)")
	}
	if len(refreshSecret) < 32 {
		panic("FATAL: JWT_REFRESH_SECRET must be at least 32 characters long for security")
	}

	return &JwtConfig{
		SecretKey:          secret,
		AccessTokenExpiry:  accessExpiry,
		RefreshTokenExpiry: refreshExpiry,
		RefreshTokenSecret: refreshSecret,
	}
}

type AppConfig struct {
	AppName string
	Port    string
	Env     string
	JwtCfg  *JwtConfig
}

func LoadAppConfig() *AppConfig {
	// Initialize Viper if not already done
	if globalViper == nil {
		InitConfig()
	}

	// Validate PORT
	port := GetConfigString("PORT")
	if port == "" {
		port = "8080"
	}

	// Validate ENVIRONMENT
	env := GetConfigString("ENVIRONMENT")
	if env == "" {
		env = "development"
	}
	if env != "development" && env != "production" {
		panic(fmt.Sprintf("FATAL: ENVIRONMENT must be 'development' or 'production', got '%s'", env))
	}

	return &AppConfig{
		AppName: "license-management-api",
		Port:    port,
		Env:     env,
		JwtCfg:  LoadJwtConfig(),
	}
}
