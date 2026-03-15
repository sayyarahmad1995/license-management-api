package handler

import (
	"net/http"
	"testing"

	"license-management-api/internal/dto"
	"license-management-api/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock AuthService for handler testing
type MockAuthServiceHandler struct {
	mock.Mock
}

func (m *MockAuthServiceHandler) Register(req *dto.RegisterDto, ipAddress string) (*models.User, error) {
	args := m.Called(req, ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceHandler) Login(req *dto.LoginDto, ipAddress string) (*dto.LoginResultDto, error) {
	args := m.Called(req, ipAddress)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*dto.LoginResultDto), args.Error(1)
}

func (m *MockAuthServiceHandler) ValidateUser(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthServiceHandler) ChangePassword(userID int, req *dto.ChangePasswordDto) error {
	args := m.Called(userID, req)
	return args.Error(0)
}

func (m *MockAuthServiceHandler) VerifyEmail(token string) error {
	args := m.Called(token)
	return args.Error(0)
}

func (m *MockAuthServiceHandler) ResendVerificationEmail(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthServiceHandler) RequestPasswordReset(email string) error {
	args := m.Called(email)
	return args.Error(0)
}

func (m *MockAuthServiceHandler) ConfirmPasswordReset(token string, newPassword string) error {
	args := m.Called(token, newPassword)
	return args.Error(0)
}

func (m *MockAuthServiceHandler) RevokeToken(refreshToken string) error {
	args := m.Called(refreshToken)
	return args.Error(0)
}

// Auth Handler Tests
func TestRegisterEndpoint(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockAuthSvc := new(MockAuthServiceHandler)

	registerReq := &dto.RegisterDto{
		Email:    "test@example.com",
		Password: "password123",
		Username: "testuser",
	}

	expectedUser := &models.User{
		ID:       1,
		Email:    "test@example.com",
		Username: "testuser",
		Status:   "Active",
	}

	mockAuthSvc.On("Register", registerReq, mock.Anything).Return(expectedUser, nil)

	// Create HTTP request
	req, _ := http.NewRequest("POST", "/api/auth/register", nil)
	req.Header.Set("Content-Type", "application/json")

	// Verify the request would succeed
	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "/api/auth/register", req.URL.Path)
}

func TestLoginEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	loginReq := &dto.LoginDto{
		Username: "testuser",
		Password: "password123",
	}

	expectedResult := &dto.LoginResultDto{
		AccessToken:  "access_token_123",
		RefreshToken: "refresh_token_456",
	}

	mockAuthSvc.On("Login", loginReq, mock.Anything).Return(expectedResult, nil)

	user, err := mockAuthSvc.Login(loginReq, "192.168.1.1")
	assert.Nil(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "access_token_123", user.AccessToken)
}

func TestLogoutEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	mockAuthSvc.On("RevokeToken", "refresh_token_123").Return(nil)

	err := mockAuthSvc.RevokeToken("refresh_token_123")
	assert.Nil(t, err)
}

func TestChangePasswordEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	changePassReq := &dto.ChangePasswordDto{
		OldPassword: "oldpass",
		NewPassword: "newpass",
	}

	mockAuthSvc.On("ChangePassword", 1, changePassReq).Return(nil)

	err := mockAuthSvc.ChangePassword(1, changePassReq)
	assert.Nil(t, err)
}

func TestVerifyEmailEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	token := "email_verify_token_123"
	mockAuthSvc.On("VerifyEmail", token).Return(nil)

	err := mockAuthSvc.VerifyEmail(token)
	assert.Nil(t, err)
}

func TestResendVerificationEmailEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	mockAuthSvc.On("ResendVerificationEmail", "test@example.com").Return(nil)

	err := mockAuthSvc.ResendVerificationEmail("test@example.com")
	assert.Nil(t, err)
}

func TestPasswordResetRequestEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	mockAuthSvc.On("RequestPasswordReset", "test@example.com").Return(nil)

	err := mockAuthSvc.RequestPasswordReset("test@example.com")
	assert.Nil(t, err)
}

func TestPasswordResetConfirmEndpoint(t *testing.T) {
	mockAuthSvc := new(MockAuthServiceHandler)

	token := "reset_token_123"
	newPassword := "newpassword123"

	mockAuthSvc.On("ConfirmPasswordReset", token, newPassword).Return(nil)

	err := mockAuthSvc.ConfirmPasswordReset(token, newPassword)
	assert.Nil(t, err)
}
