package dto

import "time"

// EmailTemplate represents available email templates
type EmailTemplate string

const (
	EmailTemplateWelcome           EmailTemplate = "welcome"
	EmailTemplateVerification      EmailTemplate = "verification"
	EmailTemplatePasswordReset     EmailTemplate = "password_reset"
	EmailTemplateLicenseExpiry     EmailTemplate = "license_expiry"
	EmailTemplateSecurityAlert     EmailTemplate = "security_alert"
	EmailTemplateLicenseActivation EmailTemplate = "license_activation"
	EmailTemplateLicenseRevocation EmailTemplate = "license_revocation"
	EmailTemplatePasswordChanged   EmailTemplate = "password_changed"
	EmailTemplateAccountSuspended  EmailTemplate = "account_suspended"
	EmailTemplateUnusualActivity   EmailTemplate = "unusual_activity"
)

// EmailNotification represents an email to be sent
type EmailNotification struct {
	ID         string                 `json:"id"`
	To         string                 `json:"to"`
	Template   EmailTemplate          `json:"template"`
	Subject    string                 `json:"subject"`
	Variables  map[string]interface{} `json:"variables"`
	Priority   string                 `json:"priority"` // "low", "normal", "high"
	Status     string                 `json:"status"`   // "pending", "sent", "failed"
	RetryCount int                    `json:"retry_count"`
	LastError  *string                `json:"last_error,omitempty"`
	CreatedAt  time.Time              `json:"created_at"`
	UpdatedAt  time.Time              `json:"updated_at"`
	SentAt     *time.Time             `json:"sent_at,omitempty"`
}

// SendEmailRequest represents a request to send an email
type SendEmailRequest struct {
	To        string                 `json:"to" binding:"required,email"`
	Template  EmailTemplate          `json:"template" binding:"required"`
	Subject   string                 `json:"subject,omitempty"`
	Variables map[string]interface{} `json:"variables,omitempty"`
	Priority  string                 `json:"priority"` // "low", "normal", "high"
}

// SendEmailResponse represents the response after email send attempt
type SendEmailResponse struct {
	Status    string    `json:"status"`
	Message   string    `json:"message"`
	MessageID string    `json:"message_id,omitempty"`
	SentAt    time.Time `json:"sent_at,omitempty"`
}

// EmailConfiguration holds SMTP configuration
type EmailConfiguration struct {
	SMTPHost     string `json:"smtp_host"`
	SMTPPort     int    `json:"smtp_port"`
	SMTPUsername string `json:"smtp_username"`
	SMTPPassword string `json:"smtp_password"`
	FromEmail    string `json:"from_email"`
	FromName     string `json:"from_name"`
	ReplyTo      string `json:"reply_to,omitempty"`
}

// EmailTemplate content for mapping
type EmailTemplateContent struct {
	Subject string
	HTML    string
	Text    string
}

// GetEmailTemplates returns the email template map
func GetEmailTemplates() map[EmailTemplate]*EmailTemplateContent {
	return map[EmailTemplate]*EmailTemplateContent{
		EmailTemplateWelcome: {
			Subject: "Welcome!",
			HTML:    `<h1>Welcome, {{.Username}}!</h1><p>Thank you for signing up to our License Management API.</p>`,
			Text:    "Welcome!",
		},
		EmailTemplateVerification: {
			Subject: "Verify Your Email Address",
			HTML:    `<h1>Verify Your Email</h1><p>Click the link below to verify your email address.</p><a href="{{.VerificationLink}}">Verify Email</a>`,
			Text:    "Verify your email address",
		},
		EmailTemplatePasswordReset: {
			Subject: "Reset Your Password",
			HTML:    `<h1>Password Reset</h1><p>Click the link below to reset your password.</p><a href="{{.ResetLink}}">Reset Password</a>`,
			Text:    "Reset your password",
		},
		EmailTemplateLicenseExpiry: {
			Subject: "Your License is Expiring Soon",
			HTML:    `<h1>License Expiry Notice</h1><p>Your license {{.LicenseKey}} will expire on {{.ExpiryDate}}.</p>`,
			Text:    "Your license is expiring soon",
		},
		EmailTemplateSecurityAlert: {
			Subject: "Security Alert - Unusual Login Activity",
			HTML:    `<h1>Security Alert</h1><p>Unusual login activity detected on your account from {{.Location}}.</p>`,
			Text:    "Security alert: Unusual login activity",
		},
		EmailTemplateLicenseActivation: {
			Subject: "License Activated Successfully",
			HTML:    `<h1>License Activated</h1><p>Your license {{.LicenseKey}} has been activated on {{.ActivationDate}}.</p>`,
			Text:    "License activated successfully",
		},
		EmailTemplateLicenseRevocation: {
			Subject: "License Revoked",
			HTML:    `<h1>License Revoked</h1><p>Your license {{.LicenseKey}} has been revoked.</p>`,
			Text:    "License revoked",
		},
		EmailTemplatePasswordChanged: {
			Subject: "Password Changed Successfully",
			HTML:    `<h1>Password Changed</h1><p>Your password was successfully changed on {{.ChangeDate}}.</p>`,
			Text:    "Password changed successfully",
		},
		EmailTemplateAccountSuspended: {
			Subject: "Account Suspended",
			HTML:    `<h1>Account Suspended</h1><p>Your account has been suspended due to security concerns.</p>`,
			Text:    "Account suspended",
		},
		EmailTemplateUnusualActivity: {
			Subject: "Unusual Account Activity Detected",
			HTML:    `<h1>Unusual Activity</h1><p>Unusual activity detected on your account.</p>`,
			Text:    "Unusual account activity detected",
		},
	}
}
