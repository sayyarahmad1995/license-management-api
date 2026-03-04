package handler

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
	"github.com/testcontainers/testcontainers-go/wait"
	postgresdriver "gorm.io/driver/postgres"
	"gorm.io/gorm"
	"license-management-api/internal/models"
)

// setupHandlerDBTest starts PostgreSQL and initializes database for handler testing
func setupHandlerDBTest(_ *testing.T) (*gorm.DB, testcontainers.Container, error) {
	ctx := context.Background()

	// Start PostgreSQL container
	container, err := postgres.Run(
		ctx,
		"postgres:15-alpine",
		postgres.WithDatabase("handlertest"),
		postgres.WithUsername("testuser"),
		postgres.WithPassword("testpass"),
		testcontainers.WithWaitStrategy(
			wait.ForLog("database system is ready to accept connections").
				WithOccurrence(2).
				WithStartupTimeout(30*time.Second)),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to start postgres: %w", err)
	}

	dbHost, _ := container.Host(ctx)
	dbPort, _ := container.MappedPort(ctx, "5432/tcp")

	dsn := fmt.Sprintf("postgres://testuser:testpass@%s:%s/handlertest?sslmode=disable", dbHost, dbPort.Port())
	db, err := gorm.Open(postgresdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		container.Terminate(ctx)
		return nil, nil, fmt.Errorf("failed to connect: %w", err)
	}

	// Run migrations
	db.AutoMigrate(&models.User{}, &models.License{}, &models.LicenseActivation{}, &models.AuditLog{})

	return db, container, nil
}

// TestHandlerDatabaseUserOps verifies user persistence for handler testing
func TestHandlerDatabaseUserOps(t *testing.T) {
	db, container, err := setupHandlerDBTest(t)
	require.NoError(t, err)
	defer container.Terminate(context.Background())

	user := &models.User{
		Email:        "handler@example.com",
		Username:     "handleruser",
		PasswordHash: "hash123",
		Status:       "Active",
	}
	assert.NoError(t, db.Create(user).Error)
	assert.NotZero(t, user.ID)

	var retrieved models.User
	assert.NoError(t, db.First(&retrieved, user.ID).Error)
	assert.Equal(t, "handler@example.com", retrieved.Email)
}

// TestHandlerDatabaseLicenseOps verifies license persistence
func TestHandlerDatabaseLicenseOps(t *testing.T) {
	db, container, err := setupHandlerDBTest(t)
	require.NoError(t, err)
	defer container.Terminate(context.Background())

	user := &models.User{Email: "licowner@example.com", Username: "licowner", PasswordHash: "h", Status: "Active"}
	db.Create(user)

	license := &models.License{
		UserID:         user.ID,
		LicenseKey:     "LICENSE-001",
		Status:         "Active",
		MaxActivations: 5,
	}
	assert.NoError(t, db.Create(license).Error)

	var retrieved models.License
	assert.NoError(t, db.First(&retrieved, license.ID).Error)
	assert.Equal(t, user.ID, retrieved.UserID)
}

// TestHandlerDatabaseAuditOps verifies audit log persistence
func TestHandlerDatabaseAuditOps(t *testing.T) {
	db, container, err := setupHandlerDBTest(t)
	require.NoError(t, err)
	defer container.Terminate(context.Background())

	entityID := 1
	ipAddr := "192.168.1.1"
	details := "User logged in"

	log := &models.AuditLog{
		Action:     "USER_LOGIN",
		EntityType: "User",
		EntityID:   &entityID,
		Details:    &details,
		IpAddress:  &ipAddr,
	}
	assert.NoError(t, db.Create(log).Error)

	var retrieved models.AuditLog
	assert.NoError(t, db.Where("action = ?", "USER_LOGIN").First(&retrieved).Error)
	assert.NotNil(t, retrieved.IpAddress)
}

// TestHandlerDatabaseRelationships verifies entity relationships
func TestHandlerDatabaseRelationships(t *testing.T) {
	db, container, err := setupHandlerDBTest(t)
	require.NoError(t, err)
	defer container.Terminate(context.Background())

	user := &models.User{Email: "rel@example.com", Username: "reluser", PasswordHash: "h", Status: "Active"}
	db.Create(user)

	license := &models.License{UserID: user.ID, LicenseKey: "REL-LIC", Status: "Active"}
	db.Create(license)

	activation := &models.LicenseActivation{
		LicenseID:          license.ID,
		MachineFingerprint: "fingerprint123",
	}
	db.Create(activation)

	var verify models.License
	db.Preload("Activations").First(&verify, license.ID)
	assert.Equal(t, 1, len(verify.Activations))
}
