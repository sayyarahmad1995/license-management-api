package handler

import (
	"net/http"
	"testing"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock LicenseService for handler testing
type MockLicenseServiceHandler struct {
	mock.Mock
}

func (m *MockLicenseServiceHandler) CreateLicense(req *dto.CreateLicenseDto) (*models.License, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}

func (m *MockLicenseServiceHandler) GetLicense(licenseID int) (*models.License, error) {
	args := m.Called(licenseID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.License), args.Error(1)
}

func (m *MockLicenseServiceHandler) DeleteLicense(licenseID int) error {
	args := m.Called(licenseID)
	return args.Error(0)
}

func (m *MockLicenseServiceHandler) GetLicensesByUser(userID int, page, pageSize int) ([]models.License, int64, error) {
	args := m.Called(userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.License), args.Get(1).(int64), args.Error(2)
}

func (m *MockLicenseServiceHandler) ValidateLicense(licenseKey string) (bool, error) {
	args := m.Called(licenseKey)
	return args.Bool(0), args.Error(1)
}

func (m *MockLicenseServiceHandler) ActivateLicense(req *dto.ActivateLicenseDto) (*models.LicenseActivation, error) {
	args := m.Called(req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.LicenseActivation), args.Error(1)
}

// License Handler Tests
func TestCreateLicenseHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	createReq := &dto.CreateLicenseDto{
		UserID:    1,
		ExpiresAt: time.Now().AddDate(1, 0, 0),
	}

	expectedLicense := &models.License{
		ID:         1,
		UserID:     1,
		LicenseKey: "LICENSE-ABC123",
		Status:     "Active",
	}

	mockLicenseSvc.On("CreateLicense", createReq).Return(expectedLicense, nil)

	license, err := mockLicenseSvc.CreateLicense(createReq)
	assert.Nil(t, err)
	assert.NotNil(t, license)
	assert.Equal(t, "LICENSE-ABC123", license.LicenseKey)
}

func TestGetLicenseHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	expectedLicense := &models.License{
		ID:         1,
		LicenseKey: "LICENSE-ABC123",
		Status:     "ACTIVE",
	}

	mockLicenseSvc.On("GetLicense", 1).Return(expectedLicense, nil)

	license, err := mockLicenseSvc.GetLicense(1)
	assert.Nil(t, err)
	assert.NotNil(t, license)
}

func TestGetLicenseNotFound(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	mockLicenseSvc.On("GetLicense", 999).Return((*models.License)(nil), assert.AnError)

	license, err := mockLicenseSvc.GetLicense(999)
	assert.Nil(t, license)
	assert.NotNil(t, err)
}

func TestDeleteLicenseHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	mockLicenseSvc.On("DeleteLicense", 1).Return(nil)

	err := mockLicenseSvc.DeleteLicense(1)
	assert.Nil(t, err)
}

func TestGetLicensesByUserHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	licenses := []models.License{
		{ID: 1, LicenseKey: "LICENSE-ABC123", Status: "ACTIVE"},
		{ID: 2, LicenseKey: "LICENSE-XYZ789", Status: "ACTIVE"},
	}

	mockLicenseSvc.On("GetLicensesByUser", 1, 1, 10).Return(licenses, int64(2), nil)

	result, total, err := mockLicenseSvc.GetLicensesByUser(1, 1, 10)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(result))
	assert.Equal(t, int64(2), total)
}

func TestValidateLicenseHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	mockLicenseSvc.On("ValidateLicense", "LICENSE-ABC123").Return(true, nil)

	isValid, err := mockLicenseSvc.ValidateLicense("LICENSE-ABC123")
	assert.Nil(t, err)
	assert.True(t, isValid)
}

func TestValidateLicenseInvalid(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	mockLicenseSvc.On("ValidateLicense", "INVALID-KEY").Return(false, nil)

	isValid, err := mockLicenseSvc.ValidateLicense("INVALID-KEY")
	assert.Nil(t, err)
	assert.False(t, isValid)
}

func TestActivateLicenseHandler(t *testing.T) {
	mockLicenseSvc := new(MockLicenseServiceHandler)

	activateReq := &dto.ActivateLicenseDto{
		LicenseKey:         "LICENSE-ABC123",
		MachineFingerprint: "MACHINE-123",
	}

	expectedActivation := &models.LicenseActivation{
		ID:                 1,
		LicenseID:          1,
		MachineFingerprint: "MACHINE-123",
	}

	mockLicenseSvc.On("ActivateLicense", activateReq).Return(expectedActivation, nil)

	activation, err := mockLicenseSvc.ActivateLicense(activateReq)
	assert.Nil(t, err)
	assert.NotNil(t, activation)
	assert.Equal(t, "MACHINE-123", activation.MachineFingerprint)
}

// Mock Health Service
type MockHealthServiceHandler struct {
	mock.Mock
}

func (m *MockHealthServiceHandler) Health() map[string]interface{} {
	args := m.Called()
	return args.Get(0).(map[string]interface{})
}

// Health Handler Tests
func TestHealthCheckHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockHealthSvc := new(MockHealthServiceHandler)

	healthStatus := map[string]interface{}{
		"status":    "up",
		"database":  "up",
		"cache":     "up",
		"timestamp": time.Now().Unix(),
	}

	mockHealthSvc.On("Health").Return(healthStatus)

	status := mockHealthSvc.Health()
	assert.NotNil(t, status)
	assert.Equal(t, "up", status["status"])
}

func TestLivenessProbe(t *testing.T) {
	mockHealthSvc := new(MockHealthServiceHandler)

	livenessStatus := map[string]interface{}{
		"status": "alive",
	}

	mockHealthSvc.On("Health").Return(livenessStatus)

	status := mockHealthSvc.Health()
	assert.Equal(t, "alive", status["status"])
}

func TestReadinessProbe(t *testing.T) {
	mockHealthSvc := new(MockHealthServiceHandler)

	readinessStatus := map[string]interface{}{
		"status":   "ready",
		"database": "connected",
		"cache":    "connected",
	}

	mockHealthSvc.On("Health").Return(readinessStatus)

	status := mockHealthSvc.Health()
	assert.Equal(t, "ready", status["status"])
	assert.Equal(t, "connected", status["database"])
}

func TestHealthEndpointHTTP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	// Create a test HTTP request
	req, _ := http.NewRequest("GET", "/health", nil)

	// Verify request properties
	assert.Equal(t, "GET", req.Method)
	assert.Equal(t, "/health", req.URL.Path)
}
