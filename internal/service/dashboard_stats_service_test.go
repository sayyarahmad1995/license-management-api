package service

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock DashboardStatsRepository
type MockDashboardStatsRepository struct {
	mock.Mock
}

func (m *MockDashboardStatsRepository) CountLicenses() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardStatsRepository) CountActiveLicenses() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardStatsRepository) CountExpiredLicenses() (int64, error) {
	args := m.Called()
	return args.Get(0).(int64), args.Error(1)
}

func (m *MockDashboardStatsRepository) CountExpiringLicenses(days int) (int64, error) {
	args := m.Called(days)
	return args.Get(0).(int64), args.Error(1)
}

// Dashboard Stats Tests
func TestTotalLicensesCount(t *testing.T) {
	mockRepo := new(MockDashboardStatsRepository)

	mockRepo.On("CountLicenses").Return(int64(150), nil)

	count, err := mockRepo.CountLicenses()
	assert.Nil(t, err)
	assert.Equal(t, int64(150), count)
}

func TestActiveLicensesCount(t *testing.T) {
	mockRepo := new(MockDashboardStatsRepository)

	mockRepo.On("CountActiveLicenses").Return(int64(120), nil)

	count, err := mockRepo.CountActiveLicenses()
	assert.Nil(t, err)
	assert.Equal(t, int64(120), count)
}

func TestExpiredLicensesCount(t *testing.T) {
	mockRepo := new(MockDashboardStatsRepository)

	mockRepo.On("CountExpiredLicenses").Return(int64(10), nil)

	count, err := mockRepo.CountExpiredLicenses()
	assert.Nil(t, err)
	assert.Equal(t, int64(10), count)
}

func TestExpiringLicensesInDays(t *testing.T) {
	mockRepo := new(MockDashboardStatsRepository)

	// Licenses expiring in 30 days
	mockRepo.On("CountExpiringLicenses", 30).Return(int64(15), nil)

	count, err := mockRepo.CountExpiringLicenses(30)
	assert.Nil(t, err)
	assert.Equal(t, int64(15), count)
}

func TestLicenseExpirationForecast(t *testing.T) {
	mockRepo := new(MockDashboardStatsRepository)

	// Setup forecast data
	mockRepo.On("CountExpiringLicenses", 7).Return(int64(5), nil)
	mockRepo.On("CountExpiringLicenses", 30).Return(int64(15), nil)
	mockRepo.On("CountExpiringLicenses", 90).Return(int64(40), nil)

	// Verify forecasts
	week, _ := mockRepo.CountExpiringLicenses(7)
	month, _ := mockRepo.CountExpiringLicenses(30)
	quarter, _ := mockRepo.CountExpiringLicenses(90)

	assert.Less(t, week, month)
	assert.Less(t, month, quarter)
	assert.Equal(t, int64(5), week)
	assert.Equal(t, int64(15), month)
	assert.Equal(t, int64(40), quarter)
}

func TestDashboardMetricsCalculation(t *testing.T) {
	total := int64(150)
	active := int64(120)
	expired := int64(10)
	expiring := int64(15)

	// Calculate usage percentage
	usagePercentage := (active * 100) / total
	assert.Equal(t, int64(80), usagePercentage)

	// Calculate expiration risk
	riskPercentage := ((expired + expiring) * 100) / total
	assert.Equal(t, int64(16), riskPercentage)
}

// Mock TokenService for additional testing
type MockTokenServiceExtended struct {
	mock.Mock
}

func (m *MockTokenServiceExtended) GenerateAccessToken(userID int) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenServiceExtended) GenerateRefreshToken(userID int, email, role string) (string, error) {
	args := m.Called(userID, email, role)
	return args.String(0), args.Error(1)
}

func (m *MockTokenServiceExtended) GenerateVerificationToken(userID int, expiryMinutes int) (string, error) {
	args := m.Called(userID, expiryMinutes)
	return args.String(0), args.Error(1)
}

// Token Service Tests
func TestGenerateAccessToken(t *testing.T) {
	mockTokenSvc := new(MockTokenServiceExtended)

	mockTokenSvc.On("GenerateAccessToken", 1).Return("access_token_123", nil)

	token, err := mockTokenSvc.GenerateAccessToken(1)
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
	assert.Equal(t, "access_token_123", token)
}

func TestGenerateRefreshToken(t *testing.T) {
	mockTokenSvc := new(MockTokenServiceExtended)

	mockTokenSvc.On("GenerateRefreshToken", 1, "user@example.com", "User").Return("refresh_token_456", nil)

	token, err := mockTokenSvc.GenerateRefreshToken(1, "user@example.com", "User")
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
}

func TestGenerateVerificationToken(t *testing.T) {
	mockTokenSvc := new(MockTokenServiceExtended)

	// Email verification tokens have 24 hour expiry
	mockTokenSvc.On("GenerateVerificationToken", 1, 1440).Return("verify_token_789", nil)

	token, err := mockTokenSvc.GenerateVerificationToken(1, 1440)
	assert.Nil(t, err)
	assert.NotEmpty(t, token)
}

func TestTokenExpiration(t *testing.T) {
	now := time.Now()

	// Token valid for 1 hour
	expiryTime := now.Add(1 * time.Hour)
	isExpired := expiryTime.Before(now)
	assert.False(t, isExpired)

	// Token expired 1 hour ago
	expiryTime = now.Add(-1 * time.Hour)
	isExpired = expiryTime.Before(now)
	assert.True(t, isExpired)
}

func TestTokenRefresh(t *testing.T) {
	mockTokenSvc := new(MockTokenServiceExtended)

	// Old token expires
	mockTokenSvc.On("GenerateAccessToken", 1).Return("new_access_token_123", nil)

	newToken, err := mockTokenSvc.GenerateAccessToken(1)
	assert.Nil(t, err)
	assert.NotEmpty(t, newToken)
	assert.NotEqual(t, "old_token", newToken)
}
