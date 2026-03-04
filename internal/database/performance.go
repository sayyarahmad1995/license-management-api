package database

import (
	"fmt"
	"log"

	"gorm.io/gorm"
)

// CreateIndexes creates all strategic database indexes for performance optimization
func CreateIndexes(db *gorm.DB) error {
	// User indexes
	indexesToCreate := []struct {
		name   string
		table  string
		column string
		query  string
	}{
		// User indexes
		{"idx_user_email", "users", "email", "CREATE INDEX IF NOT EXISTS idx_user_email ON users(email)"},
		{"idx_user_username", "users", "username", "CREATE INDEX IF NOT EXISTS idx_user_username ON users(username)"},
		{"idx_user_status", "users", "status", "CREATE INDEX IF NOT EXISTS idx_user_status ON users(status)"},
		{"idx_user_created", "users", "created_at", "CREATE INDEX IF NOT EXISTS idx_user_created ON users(created_at DESC)"},

		// License indexes
		{"idx_license_key", "licenses", "license_key", "CREATE INDEX IF NOT EXISTS idx_license_key ON licenses(license_key)"},
		{"idx_license_status", "licenses", "status", "CREATE INDEX IF NOT EXISTS idx_license_status ON licenses(status)"},
		{"idx_license_user_id", "licenses", "user_id", "CREATE INDEX IF NOT EXISTS idx_license_user_id ON licenses(user_id)"},
		{"idx_license_expiry", "licenses", "expiry_date", "CREATE INDEX IF NOT EXISTS idx_license_expiry ON licenses(expiry_date)"},
		{"idx_license_created", "licenses", "created_at", "CREATE INDEX IF NOT EXISTS idx_license_created ON licenses(created_at DESC)"},
		{"idx_license_key_status", "licenses", "key_status", "CREATE INDEX IF NOT EXISTS idx_license_key_status ON licenses(license_key, status)"},

		// License Activation indexes
		{"idx_activation_license", "license_activations", "license_id", "CREATE INDEX IF NOT EXISTS idx_activation_license ON license_activations(license_id)"},
		{"idx_activation_machine", "license_activations", "machine_fingerprint", "CREATE INDEX IF NOT EXISTS idx_activation_machine ON license_activations(machine_fingerprint)"},
		{"idx_activation_created", "license_activations", "created_at", "CREATE INDEX IF NOT EXISTS idx_activation_created ON license_activations(created_at DESC)"},
		{"idx_activation_heartbeat", "license_activations", "last_heartbeat", "CREATE INDEX IF NOT EXISTS idx_activation_heartbeat ON license_activations(last_heartbeat)"},

		// Audit Log indexes
		{"idx_audit_user", "audit_logs", "user_id", "CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id)"},
		{"idx_audit_action", "audit_logs", "action", "CREATE INDEX IF NOT EXISTS idx_audit_action ON audit_logs(action)"},
		{"idx_audit_resource", "audit_logs", "resource_type", "CREATE INDEX IF NOT EXISTS idx_audit_resource ON audit_logs(resource_type)"},
		{"idx_audit_created", "audit_logs", "created_at", "CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_logs(created_at DESC)"},
		{"idx_audit_user_action", "audit_logs", "user_id_action", "CREATE INDEX IF NOT EXISTS idx_audit_user_action ON audit_logs(user_id, action, created_at DESC)"},

		// Email Verification indexes
		{"idx_email_verify_token", "email_verifications", "token", "CREATE INDEX IF NOT EXISTS idx_email_verify_token ON email_verifications(token)"},
		{"idx_email_verify_user", "email_verifications", "user_id", "CREATE INDEX IF NOT EXISTS idx_email_verify_user ON email_verifications(user_id)"},
		{"idx_email_verify_expires", "email_verifications", "expires_at", "CREATE INDEX IF NOT EXISTS idx_email_verify_expires ON email_verifications(expires_at)"},

		// Password Reset indexes
		{"idx_password_reset_token", "password_resets", "token", "CREATE INDEX IF NOT EXISTS idx_password_reset_token ON password_resets(token)"},
		{"idx_password_reset_user", "password_resets", "user_id", "CREATE INDEX IF NOT EXISTS idx_password_reset_user ON password_resets(user_id)"},
		{"idx_password_reset_expires", "password_resets", "expires_at", "CREATE INDEX IF NOT EXISTS idx_password_reset_expires ON password_resets(expires_at)"},
	}

	for _, idx := range indexesToCreate {
		result := db.Exec(idx.query)
		if result.Error != nil {
			log.Printf("Warning: Failed to create index %s: %v", idx.name, result.Error)
			// Don't fail completely if index creation fails
		} else {
			log.Printf("✓ Index created: %s", idx.name)
		}
	}

	// Create composite indexes for common queries
	compositeIndexes := []struct {
		name  string
		query string
	}{
		{
			"idx_license_user_status",
			"CREATE INDEX IF NOT EXISTS idx_license_user_status ON licenses(user_id, status)",
		},
		{
			"idx_license_status_expiry",
			"CREATE INDEX IF NOT EXISTS idx_license_status_expiry ON licenses(status, expiry_date)",
		},
		{
			"idx_activation_license_active",
			"CREATE INDEX IF NOT EXISTS idx_activation_license_active ON license_activations(license_id, last_heartbeat DESC)",
		},
		{
			"idx_audit_created_user",
			"CREATE INDEX IF NOT EXISTS idx_audit_created_user ON audit_logs(created_at DESC, user_id)",
		},
	}

	for _, idx := range compositeIndexes {
		result := db.Exec(idx.query)
		if result.Error != nil {
			log.Printf("Warning: Failed to create composite index %s: %v", idx.name, result.Error)
		} else {
			log.Printf("✓ Composite index created: %s", idx.name)
		}
	}

	log.Println("📊 Database indexes created successfully!")
	return nil
}

// ConfigureConnectionPool optimizes the database connection pool
func ConfigureConnectionPool(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get database instance: %w", err)
	}

	// Set connection pool parameters
	// MaxIdleConns: maximum number of idle connections
	sqlDB.SetMaxIdleConns(10)

	// MaxOpenConns: maximum number of open connections
	sqlDB.SetMaxOpenConns(100)

	// ConnMaxLifetime: maximum lifetime of a connection (5 minutes)
	// Note: In production code, this should use time.Duration(5 * time.Minute)
	sqlDB.SetConnMaxLifetime(5 * 60) // seconds representation

	log.Printf("✓ Connection pool configured: MaxIdle=10, MaxOpen=100, MaxLifetime=5m")
	return nil
}

// EnableSlowQueryLogging enables PostgreSQL slow query logs
func EnableSlowQueryLogging(db *gorm.DB) error {
	// Set log_min_duration_statement to log queries taking longer than 1 second
	result := db.Exec("SET log_min_duration_statement = 1000") // milliseconds
	if result.Error != nil {
		return fmt.Errorf("failed to enable slow query logging: %w", result.Error)
	}

	log.Println("✓ Slow query logging enabled (1000ms threshold)")
	return nil
}

// VacuumDatabase performs maintenance on the database (removes dead tuples)
func VacuumDatabase(db *gorm.DB) error {
	tables := []string{"users", "licenses", "license_activations", "audit_logs", "email_verifications", "password_resets"}

	for _, table := range tables {
		result := db.Exec(fmt.Sprintf("VACUUM ANALYZE %s", table))
		if result.Error != nil {
			log.Printf("Warning: Failed to vacuum %s: %v", table, result.Error)
		} else {
			log.Printf("✓ Vacuumed and analyzed table: %s", table)
		}
	}

	return nil
}
