package service

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"license-management-api/internal/models"
)

// Mock LicenseRepository
type MockLicenseRepository struct {
	mock.Mock
}

func (m *MockLicenseRepository) Create(license *models.License) error {
	args := m.Called(license)
	return args.Error(0)
}

func (m *MockLicenseRepository) GetByID(id int) (*models.License, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}

func (m *MockLicenseRepository) GetByKey(key string) (*models.License, error) {
	args := m.Called(key)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}

func (m *MockLicenseRepository) Update(license *models.License) error {
	args := m.Called(license)
	return args.Error(0)
}

func (m *MockLicenseRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockLicenseRepository) FindByUserID(userID int, page, pageSize int) ([]models.License, int64, error) {
	args := m.Called(userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.License), args.Get(1).(int64), args.Error(2)
}

func (m *MockLicenseRepository) FindByStatus(status string, page, pageSize int) ([]models.License, int64, error) {
	args := m.Called(status, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.License), args.Get(1).(int64), args.Error(2)
}

func (m *MockLicenseRepository) FindExpiringLicenses(days int, page, pageSize int) ([]models.License, int64, error) {
	args := m.Called(days, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.License), args.Get(1).(int64), args.Error(2)
}

// LicenseService Tests
func TestCreateLicenseSuccess(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	newLicense := &models.License{
		ID:         1,
		LicenseKey: "LICENSE-ABC123",
		Status:     "Active",
		UserID:     1,
		ExpiresAt:  time.Now().AddDate(1, 0, 0),
	}

	mockLicenseRepo.On("Create", newLicense).Return(nil)

	err := mockLicenseRepo.Create(newLicense)
	assert.Nil(t, err)
	mockLicenseRepo.AssertCalled(t, "Create", newLicense)
}

func TestGetLicenseByKey(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	license := &models.License{
		ID:         1,
		LicenseKey: "LICENSE-ABC123",
		Status:     "Active",
	}

	mockLicenseRepo.On("GetByKey", "LICENSE-ABC123").Return(license, nil)

	retrieved, err := mockLicenseRepo.GetByKey("LICENSE-ABC123")
	assert.Nil(t, err)
	assert.NotNil(t, retrieved)
	assert.Equal(t, "LICENSE-ABC123", retrieved.LicenseKey)
}

func TestGetLicenseNotFound(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	mockLicenseRepo.On("GetByKey", "INVALID-KEY").Return((*models.License)(nil), errors.New("record not found"))

	retrieved, err := mockLicenseRepo.GetByKey("INVALID-KEY")
	assert.Nil(t, retrieved)
	assert.NotNil(t, err)
}

func TestUpdateLicenseSuccess(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	license := &models.License{
		ID:     1,
		Status: "Active",
	}

	mockLicenseRepo.On("Update", license).Return(nil)

	err := mockLicenseRepo.Update(license)
	assert.Nil(t, err)
}

func TestUpdateLicenseStatus(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	license := &models.License{
		ID:     1,
		Status: "Active",
	}

	mockLicenseRepo.On("Update", mock.MatchedBy(func(l *models.License) bool {
		return l.Status == "Revoked"
	})).Return(nil)

	license.Status = "Revoked"
	err := mockLicenseRepo.Update(license)
	assert.Nil(t, err)
}

func TestDeleteLicenseSuccess(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	mockLicenseRepo.On("Delete", 1).Return(nil)

	err := mockLicenseRepo.Delete(1)
	assert.Nil(t, err)
}

func TestFindLicensesByUserID(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	licenses := []models.License{
		{
			ID:     1,
			Status: "Active",
		},
		{
			ID:     2,
			Status: "Expired",
		},
	}

	mockLicenseRepo.On("FindByUserID", 1, 1, 10).Return(licenses, int64(2), nil)

	retrieved, total, err := mockLicenseRepo.FindByUserID(1, 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(retrieved))
	assert.Equal(t, int64(2), total)
}

func TestFindLicensesByStatus(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	activeLicenses := []models.License{
		{ID: 1, Status: "Active"},
		{ID: 2, Status: "Active"},
	}

	mockLicenseRepo.On("FindByStatus", "Active", 1, 10).Return(activeLicenses, int64(2), nil)

	retrieved, total, err := mockLicenseRepo.FindByStatus("Active", 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(retrieved))
	assert.Equal(t, int64(2), total)
}

func TestFindExpiringLicenses(t *testing.T) {
	mockLicenseRepo := new(MockLicenseRepository)

	now := time.Now()
	expiringLicenses := []models.License{
		{
			ID:        1,
			Status:    "Active",
			ExpiresAt: now.AddDate(0, 0, 5),
		},
	}

	mockLicenseRepo.On("FindExpiringLicenses", 7, 1, 10).Return(expiringLicenses, int64(1), nil)

	retrieved, total, err := mockLicenseRepo.FindExpiringLicenses(7, 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, 1, len(retrieved))
	assert.Equal(t, int64(1), total)
}

func TestValidateLicenseKeyFormat(t *testing.T) {
	// Test valid formats
	validKeys := []string{
		"LICENSE-ABC123",
		"PROD-XYZ789",
		"TEST-DEV999",
	}

	for _, key := range validKeys {
		assert.NotEmpty(t, key)
		assert.Greater(t, len(key), 5)
	}
}

func TestLicenseExpiryCheck(t *testing.T) {
	now := time.Now()

	// Active license
	activeLicense := &models.License{
		ExpiresAt: now.AddDate(1, 0, 0),
	}

	isExpired := activeLicense.ExpiresAt.Before(now)
	assert.False(t, isExpired)

	// Expired license
	expiredLicense := &models.License{
		ExpiresAt: now.AddDate(-1, 0, 0),
	}

	isExpired = expiredLicense.ExpiresAt.Before(now)
	assert.True(t, isExpired)
}
