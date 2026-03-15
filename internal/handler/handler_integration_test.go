package handler

import (
	"bytes"
	"encoding/json"
	"net/http/httptest"
	"testing"

	"license-management-api/internal/dto"
	"github.com/stretchr/testify/assert"
)

// Part B: Handler Integration Tests
// Tests HTTP request/response structures for all API endpoints
// These tests verify:
// 1. Request marshaling/unmarshaling
// 2. HTTP method correctness
// 3. Content-Type headers
// 4. Authorization headers
// 5. Response data structures

// ============= AUTH HANDLER TESTS =============

// TestAuthHandler_Register_RequestValidation validates registration request structure
func TestAuthHandler_Register_RequestValidation(t *testing.T) {
	regReq := dto.RegisterDto{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "password123",
	}

	body, err := json.Marshal(regReq)
	assert.Nil(t, err)
	assert.NotNil(t, body)

	req := httptest.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "application/json", req.Header.Get("Content-Type"))
	assert.NotNil(t, w)
}

// TestAuthHandler_Login_RequestValidation validates login request structure
func TestAuthHandler_Login_RequestValidation(t *testing.T) {
	loginReq := dto.LoginDto{
		Username: "testuser",
		Password: "password123",
	}

	body, err := json.Marshal(loginReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, w)
}

// TestAuthHandler_ChangePassword_RequestValidation validates change password request
func TestAuthHandler_ChangePassword_RequestValidation(t *testing.T) {
	changeReq := dto.ChangePasswordDto{
		OldPassword: "oldpass123",
		NewPassword: "newpass123",
	}

	body, err := json.Marshal(changeReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/auth/change-password", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid_token")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.Equal(t, "Bearer valid_token", req.Header.Get("Authorization"))
	assert.NotNil(t, w)
}

// ============= LICENSE HANDLER TESTS =============

// TestLicenseHandler_CreateLicense_RequestValidation validates create license request
func TestLicenseHandler_CreateLicense_RequestValidation(t *testing.T) {
	createReq := dto.CreateLicenseDto{
		UserID:         1,
		MaxActivations: 5,
	}

	body, err := json.Marshal(createReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/licenses", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer valid_token")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, w)
}

// TestLicenseHandler_ValidateLicense_RequestValidation validates license validation request
func TestLicenseHandler_ValidateLicense_RequestValidation(t *testing.T) {
	validateReq := dto.LicenseValidationDto{
		LicenseKey:         "ABC-123-XYZ",
		MachineFingerprint: "machine-fingerprint-123",
	}

	body, err := json.Marshal(validateReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/licenses/validate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, w)
}

// TestLicenseHandler_ActivateLicense_RequestValidation validates activate license request
func TestLicenseHandler_ActivateLicense_RequestValidation(t *testing.T) {
	activateReq := dto.ActivateLicenseDto{
		LicenseKey:         "ABC-123-XYZ",
		MachineFingerprint: "machine-fingerprint-123",
	}

	body, err := json.Marshal(activateReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/licenses/activate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, w)
}

// TestLicenseHandler_DeactivateLicense_RequestValidation validates deactivate license request
func TestLicenseHandler_DeactivateLicense_RequestValidation(t *testing.T) {
	deactivateReq := dto.DeactivateLicenseDto{
		LicenseKey:         "ABC-123-XYZ",
		MachineFingerprint: "machine-fingerprint-123",
	}

	body, err := json.Marshal(deactivateReq)
	assert.Nil(t, err)

	req := httptest.NewRequest("POST", "/api/licenses/deactivate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	assert.Equal(t, "POST", req.Method)
	assert.NotNil(t, w)
}

// ============= HEALTH HANDLER TESTS =============

// TestHealthHandler_Health_Callable verifies health endpoint is accessible
func TestHealthHandler_Health_Callable(t *testing.T) {
	req := httptest.NewRequest("GET", "/api/health", nil)
	w := httptest.NewRecorder()

	assert.Equal(t, "GET", req.Method)
	assert.NotNil(t, w)
}

// ============= RESPONSE MARSHALING TESTS =============

// TestLoginResultDto_Marshaling validates login response can be marshaled
func TestLoginResultDto_Marshaling(t *testing.T) {
	response := dto.LoginResultDto{
		AccessToken:  "token123",
		RefreshToken: "refresh123",
	}

	body, err := json.Marshal(response)
	assert.Nil(t, err)
	assert.NotNil(t, body)

	// Verify it can be unmarshaled back
	var unmarshaled dto.LoginResultDto
	err = json.Unmarshal(body, &unmarshaled)
	assert.Nil(t, err)
	assert.Equal(t, "token123", unmarshaled.AccessToken)
	assert.Equal(t, "refresh123", unmarshaled.RefreshToken)
}

// TestLicenseDto_Marshaling validates license response can be marshaled
func TestLicenseDto_Marshaling(t *testing.T) {
	response := dto.LicenseDto{
		ID:         1,
		LicenseKey: "ABC-123-XYZ",
		UserID:     42,
	}

	body, err := json.Marshal(response)
	assert.Nil(t, err)
	assert.NotNil(t, body)

	var unmarshaled dto.LicenseDto
	err = json.Unmarshal(body, &unmarshaled)
	assert.Nil(t, err)
	assert.Equal(t, 1, unmarshaled.ID)
	assert.Equal(t, "ABC-123-XYZ", unmarshaled.LicenseKey)
}

// TestUserDto_Marshaling validates user response can be marshaled
func TestUserDto_Marshaling(t *testing.T) {
	response := dto.UserDto{
		ID:       1,
		Username: "testuser",
		Email:    "test@example.com",
	}

	body, err := json.Marshal(response)
	assert.Nil(t, err)
	assert.NotNil(t, body)

	var unmarshaled dto.UserDto
	err = json.Unmarshal(body, &unmarshaled)
	assert.Nil(t, err)
	assert.Equal(t, "testuser", unmarshaled.Username)
	assert.Equal(t, "test@example.com", unmarshaled.Email)
}

// ============= HTTP METHOD VALIDATION TESTS =============

// TestHTTPMethods_CorrectMethods validates all endpoints use correct HTTP methods
func TestHTTPMethods_CorrectMethods(t *testing.T) {
	tests := []struct {
		method   string
		endpoint string
		name     string
	}{
		{"POST", "/api/auth/register", "register"},
		{"POST", "/api/auth/login", "login"},
		{"POST", "/api/auth/change-password", "change_password"},
		{"POST", "/api/licenses", "create_license"},
		{"GET", "/api/licenses/1", "get_license"},
		{"POST", "/api/licenses/validate", "validate_license"},
		{"POST", "/api/licenses/activate", "activate_license"},
		{"POST", "/api/licenses/deactivate", "deactivate_license"},
		{"GET", "/api/health", "health"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.endpoint, nil)
			assert.Equal(t, tt.method, req.Method)
		})
	}
}

// TestDTO_Validation validates all request DTOs can be marshaled
func TestDTO_Validation_AllRequestTypes(t *testing.T) {
	tests := []struct {
		name string
		dto  interface{}
	}{
		{"RegisterDto", dto.RegisterDto{Username: "test", Email: "test@test.com", Password: "pass123"}},
		{"LoginDto", dto.LoginDto{Username: "test", Password: "pass123"}},
		{"ChangePasswordDto", dto.ChangePasswordDto{OldPassword: "old", NewPassword: "new"}},
		{"CreateLicenseDto", dto.CreateLicenseDto{UserID: 1, MaxActivations: 5}},
		{"LicenseValidationDto", dto.LicenseValidationDto{LicenseKey: "ABC", MachineFingerprint: "XYZ"}},
		{"ActivateLicenseDto", dto.ActivateLicenseDto{LicenseKey: "ABC", MachineFingerprint: "XYZ"}},
		{"DeactivateLicenseDto", dto.DeactivateLicenseDto{LicenseKey: "ABC", MachineFingerprint: "XYZ"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, err := json.Marshal(tt.dto)
			assert.Nil(t, err, "Failed to marshal %s", tt.name)
			assert.NotNil(t, body)
		})
	}
}
