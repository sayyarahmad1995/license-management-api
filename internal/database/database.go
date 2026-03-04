package database

import (
	"fmt"
	"log"
	"os"

	"license-management-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

type Service interface {
	Health() map[string]string
	GetDB() *gorm.DB
	Close() error
}

type service struct {
	db *gorm.DB
}

var (
	dbInstance *service
)

// resetInstance is used for testing to clear the cached instance
func resetInstance() {
	if dbInstance != nil && dbInstance.db != nil {
		_ = dbInstance.Close()
	}
	dbInstance = nil
}

func New() Service {
	// Reuse Connection
	if dbInstance != nil {
		return dbInstance
	}

	dbUrl := os.Getenv("DATABASE_URL")
	if dbUrl == "" {
		dbUrl = buildDSN()
	}

	db, err := gorm.Open(postgres.Open(dbUrl), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Auto-migrate models
	err = db.AutoMigrate(
		&models.User{},
		&models.License{},
		&models.LicenseActivation{},
		&models.AuditLog{},
		&models.EmailVerification{},
		&models.PasswordReset{},
	)
	if err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	dbInstance = &service{
		db: db,
	}
	return dbInstance
}

func buildDSN() string {
	database := os.Getenv("DB_DATABASE")
	password := os.Getenv("DB_PASSWORD")
	username := os.Getenv("DB_USERNAME")
	port := os.Getenv("DB_PORT")
	host := os.Getenv("DB_HOST")

	return fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		host, port, username, password, database)
}

func (s *service) Health() map[string]string {
	sqlDB, err := s.db.DB()
	if err != nil {
		return map[string]string{
			"status": "down",
			"error":  fmt.Sprintf("Database connection error: %v", err),
		}
	}

	err = sqlDB.Ping()
	if err != nil {
		return map[string]string{
			"status": "down",
			"error":  fmt.Sprintf("Failed to ping database: %v", err),
		}
	}

	return map[string]string{
		"status":  "up",
		"message": "Database is healthy",
	}
}

func (s *service) GetDB() *gorm.DB {
	return s.db
}

func (s *service) Close() error {
	if s.db != nil {
		sqlDB, err := s.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}
