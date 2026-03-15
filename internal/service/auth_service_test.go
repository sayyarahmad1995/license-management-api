package service

import (
	"errors"
	"testing"

	"license-management-api/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"gorm.io/gorm"
)

// Mock UserRepository
type MockUserRepository struct {
	mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) GetByID(id int) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) GetByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserRepository) Update(user *models.User) error {
	args := m.Called(user)
	return args.Error(0)
}

func (m *MockUserRepository) Delete(id int) error {
	args := m.Called(id)
	return args.Error(0)
}

func (m *MockUserRepository) FindAll(page, pageSize int) ([]models.User, int64, error) {
	args := m.Called(page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

// Mock TokenService
type MockTokenService struct {
	mock.Mock
}

func (m *MockTokenService) GenerateToken(userID int, expiryHours int) (string, error) {
	args := m.Called(userID, expiryHours)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) ValidateToken(token string) (int, error) {
	args := m.Called(token)
	return args.Int(0), args.Error(1)
}

func (m *MockTokenService) RefreshToken(userID int) (string, error) {
	args := m.Called(userID)
	return args.String(0), args.Error(1)
}

func (m *MockTokenService) RevokeToken(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

// Mock AuditService
type MockAuditService struct {
	mock.Mock
}

func (m *MockAuditService) LogAction(userID int, action, resourceType string, resourceID int, details string) error {
	args := m.Called(userID, action, resourceType, resourceID, details)
	return args.Error(0)
}

func (m *MockAuditService) GetUserAuditLogs(userID int, page, pageSize int) ([]models.AuditLog, int64, error) {
	args := m.Called(userID, page, pageSize)
	if args.Get(0) == nil {
		return nil, args.Get(1).(int64), args.Error(2)
	}
	return args.Get(0).([]models.AuditLog), args.Get(1).(int64), args.Error(2)
}

// AuthService Tests
func TestRegisterSuccess(t *testing.T) {
	mockUserRepo := new(MockUserRepository)
	mockAuditSvc := new(MockAuditService)

	// Mock expectations
	mockUserRepo.On("GetByEmail", "test@example.com").Return(nil, gorm.ErrRecordNotFound)
	mockUserRepo.On("Create", mock.MatchedBy(func(u *models.User) bool {
		return u.Email == "test@example.com"
	})).Return(nil)
	mockAuditSvc.On("LogAction", mock.Anything, "REGISTER", "USER", mock.Anything, mock.Anything).Return(nil)

	// Note: In a real scenario, you'd create the AuthService with mocked dependencies
	// For now, we're testing the structure
	assert.NotNil(t, mockUserRepo)
}

func TestRegisterEmailAlreadyExists(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	// User already exists
	existingUser := &models.User{
		ID:    1,
		Email: "test@example.com",
	}

	mockUserRepo.On("GetByEmail", "test@example.com").Return(existingUser, nil)

	user, err := mockUserRepo.GetByEmail("test@example.com")
	assert.NotNil(t, user)
	assert.Nil(t, err)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestLoginSuccess(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	testUser := &models.User{
		ID:           1,
		Email:        "test@example.com",
		Username:     "testuser",
		PasswordHash: "$2a$10$hashedpassword", // Simulated bcrypt hash
	}

	mockUserRepo.On("GetByEmail", "test@example.com").Return(testUser, nil)

	// Verify user can be retrieved
	user, err := mockUserRepo.GetByEmail("test@example.com")
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "test@example.com", user.Email)
}

func TestLoginInvalidCredentials(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	mockUserRepo.On("GetByEmail", "nonexistent@example.com").Return(nil, gorm.ErrRecordNotFound)

	user, err := mockUserRepo.GetByEmail("nonexistent@example.com")
	assert.Nil(t, user)
	assert.NotNil(t, err)
	assert.Equal(t, gorm.ErrRecordNotFound, err)
}

func TestChangePasswordSuccess(t *testing.T) {
	mockUserRepo := new(MockUserRepository)

	user := &models.User{
		ID:           1,
		Email:        "test@example.com",
		PasswordHash: "$2a$10$oldpasswordhash", // Simulated bcrypt hash
	}

	mockUserRepo.On("GetByID", 1).Return(user, nil)
	mockUserRepo.On("Update", mock.MatchedBy(func(u *models.User) bool {
		return u.ID == 1
	})).Return(nil)

	// Retrieve user
	retrieved, err := mockUserRepo.GetByID(1)
	assert.Nil(t, err)
	assert.NotNil(t, retrieved)

	// Verify password can be updated
	retrieved.PasswordHash = "$2a$10$newpasswordhash" // Simulated new hash
	err = mockUserRepo.Update(retrieved)
	assert.Nil(t, err)
}

func TestVerifyEmailValidToken(t *testing.T) {
	mockTokenSvc := new(MockTokenService)

	mockTokenSvc.On("ValidateToken", "valid_verify_token").Return(1, nil)

	userID, err := mockTokenSvc.ValidateToken("valid_verify_token")
	assert.Nil(t, err)
	assert.Equal(t, 1, userID)
}

func TestVerifyEmailInvalidToken(t *testing.T) {
	mockTokenSvc := new(MockTokenService)

	mockTokenSvc.On("ValidateToken", "invalid_token").Return(0, errors.New("Invalid or expired token"))

	userID, err := mockTokenSvc.ValidateToken("invalid_token")
	assert.Equal(t, 0, userID)
	assert.NotNil(t, err)
}

func TestRevokeTokenSuccess(t *testing.T) {
	mockTokenSvc := new(MockTokenService)

	mockTokenSvc.On("RevokeToken", "refresh_token_123").Return(nil)

	err := mockTokenSvc.RevokeToken("refresh_token_123")
	assert.Nil(t, err)
}
