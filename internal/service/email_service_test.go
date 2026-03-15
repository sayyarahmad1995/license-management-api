package service

import (
	"testing"

	"license-management-api/internal/dto"
	"github.com/stretchr/testify/assert"
)

// Test NotificationQueueService
func TestNotificationQueueService_QueueNotification_Success(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()
	notification := &dto.EmailNotification{
		ID:       "test-1",
		To:       "test@example.com",
		Template: dto.EmailTemplateWelcome,
		Subject:  "Welcome!",
	}

	// Execute
	err := queueSvc.QueueNotification(notification)

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, queueSvc.GetQueueSize())
}

func TestNotificationQueueService_QueueSize_Multiple(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()

	// Queue multiple notifications
	for i := 0; i < 5; i++ {
		notification := &dto.EmailNotification{
			ID:       "test-" + string(rune(i)),
			To:       "test@example.com",
			Template: dto.EmailTemplateWelcome,
			Subject:  "Welcome!",
		}
		queueSvc.QueueNotification(notification)
	}

	// Assert
	assert.Equal(t, 5, queueSvc.GetQueueSize())
}

func TestNotificationQueueService_GetQueueSize_Empty(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()

	// Execute
	size := queueSvc.GetQueueSize()

	// Assert
	assert.Equal(t, 0, size)
}

func TestNotificationQueueService_QueueEmailWithTemplate_Success(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()

	variables := map[string]interface{}{
		"username": "John Doe",
	}

	// Execute
	err := queueSvc.QueueEmailWithTemplate("john@example.com", dto.EmailTemplateWelcome, variables, "normal")

	// Assert
	assert.Nil(t, err)
	assert.Equal(t, 1, queueSvc.GetQueueSize())
}

// Test EmailService
func TestEmailService_SendEmail_ConsoleMode(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true // Force console mode for testing
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendEmail("test@example.com", "Test Subject", "Test Body")

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendWelcomeEmail(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendWelcomeEmail("test@example.com", "John Doe")

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendVerificationEmail(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendVerificationEmail("test@example.com", "John Doe", "verification-token-123")

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendPasswordResetEmail(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendPasswordResetEmail("test@example.com", "John Doe", "reset-token-456")

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendLicenseExpiryNotification(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendLicenseExpiryNotification("test@example.com", "John Doe", "ABC-123-XYZ", 5)

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendAccountActivityNotification(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendAccountActivityNotification("test@example.com", "John Doe", "Login from new device")

	// Assert
	assert.Nil(t, err)
}

func TestEmailService_SendSystemAnnouncement(t *testing.T) {
	// Setup
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)

	// Execute
	err := emailSvc.SendSystemAnnouncement("test@example.com", "System Maintenance", "System will be down for maintenance")

	// Assert
	assert.Nil(t, err)
}

// Test template rendering
func TestRenderTemplate_SimpleSubstitution(t *testing.T) {
	// Setup
	template := "Hello {{username}}"
	variables := map[string]interface{}{
		"username": "John Doe",
	}

	// Execute
	result := renderTemplate(template, variables)

	// Assert
	assert.Equal(t, "Hello John Doe", result)
}

func TestRenderTemplate_MultiplePlaceholders(t *testing.T) {
	// Setup
	template := "Hello {{firstName}} {{lastName}}, your license {{licenseKey}} expires in {{daysUntilExpiry}} days"
	variables := map[string]interface{}{
		"firstName":       "John",
		"lastName":        "Doe",
		"licenseKey":      "ABC-123-XYZ",
		"daysUntilExpiry": 30,
	}

	// Execute
	result := renderTemplate(template, variables)

	// Assert
	assert.Equal(t, "Hello John Doe, your license ABC-123-XYZ expires in 30 days", result)
}

func TestRenderTemplate_MissingVariable(t *testing.T) {
	// Setup
	template := "Hello {{username}}, your license is {{licenseKey}}"
	variables := map[string]interface{}{
		"username": "John Doe",
	}

	// Execute
	result := renderTemplate(template, variables)

	// Assert
	// Missing variable should remain as placeholder
	assert.Contains(t, result, "John Doe")
	assert.Contains(t, result, "{{licenseKey}}")
}

func TestRenderTemplate_NoVariables(t *testing.T) {
	// Setup
	template := "Hello World"
	variables := make(map[string]interface{})

	// Execute
	result := renderTemplate(template, variables)

	// Assert
	assert.Equal(t, "Hello World", result)
}

func TestRenderTemplate_NumericVariables(t *testing.T) {
	// Setup
	template := "Activation {{activationNumber}} of {{maxActivations}}"
	variables := map[string]interface{}{
		"activationNumber": 3,
		"maxActivations":   5,
	}

	// Execute
	result := renderTemplate(template, variables)

	// Assert
	assert.Equal(t, "Activation 3 of 5", result)
}

// Test NotificationQueueService with email service integration
func TestNotificationQueueService_ProcessQueue_WithEmailService(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)
	queueSvc.SetEmailService(emailSvc)

	notification := &dto.EmailNotification{
		ID:       "test-1",
		To:       "test@example.com",
		Template: dto.EmailTemplateWelcome,
		Subject:  "Welcome!",
		Variables: map[string]interface{}{
			"username":      "John Doe",
			"dashboardLink": "https://example.com/dashboard",
		},
		Status: "pending",
	}

	// Queue the notification
	queueSvc.QueueNotification(notification)

	// Execute
	err := queueSvc.ProcessQueue()

	// Assert
	assert.Nil(t, err)
}

func TestNotificationQueueService_StartStop(t *testing.T) {
	// Setup
	queueSvc := NewNotificationQueueService()
	cfg := LoadEmailConfig()
	cfg.UseConsole = true
	emailSvc := NewEmailService(cfg)
	queueSvc.SetEmailService(emailSvc)

	// Execute
	queueSvc.Start()
	queueSvc.Stop()

	// Assert - no panics should occur
	assert.NotNil(t, queueSvc)
}
