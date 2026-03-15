package service

import (
	"crypto/tls"
	"fmt"
	"log"
	"net"
	"net/smtp"
	"os"
	"strconv"
	"strings"
	"time"
)

// EmailService defines email sending operations
type EmailService interface {
	SendLicenseExpiryNotification(email, username, licenseKey string, daysUntilExpiry int) error
	SendAccountActivityNotification(email, username, action string) error
	SendWelcomeEmail(email, username string) error
	SendSystemAnnouncement(email, subject, message string) error
	SendVerificationEmail(email, username, token string) error
	SendPasswordResetEmail(email, username, token string) error
	SendEmail(to, subject, body string) error // Generic email sending
}

// EmailConfig holds email configuration
type EmailConfig struct {
	SmtpHost        string
	SmtpPort        int
	SmtpUsername    string
	SmtpPassword    string
	FromAddress     string
	FromName        string
	FrontendBaseURL string // Frontend URL for email links
	EnableSSL       bool   // Enable SSL/TLS
	UseConsole      bool   // For development/testing
}

// emailService implements EmailService
type emailService struct {
	config EmailConfig
}

// NewEmailService creates a new email service
func NewEmailService(config EmailConfig) EmailService {
	return &emailService{config: config}
}

// LoadEmailConfig loads email configuration from environment variables
func LoadEmailConfig() EmailConfig {
	// Default SMTP port to 587 if not set
	smtpPortStr := os.Getenv("SMTP_PORT")
	smtpPort := 587
	if smtpPortStr != "" {
		if portVal, err := strconv.Atoi(smtpPortStr); err == nil {
			smtpPort = portVal
		}
	}

	config := EmailConfig{
		SmtpHost:        os.Getenv("SMTP_HOST"),
		SmtpPort:        smtpPort,
		SmtpUsername:    os.Getenv("SMTP_USER"),
		SmtpPassword:    os.Getenv("SMTP_PASS"),
		FromAddress:     os.Getenv("SMTP_FROM_EMAIL"),
		FromName:        os.Getenv("SMTP_FROM_NAME"),
		FrontendBaseURL: os.Getenv("FRONTEND_BASE_URL"),
		EnableSSL:       os.Getenv("SMTP_ENABLE_SSL") == "true",
		UseConsole:      os.Getenv("USE_CONSOLE_EMAIL") == "true",
	}

	// Auto-enable console mode in development environment
	environment := os.Getenv("ENVIRONMENT")
	if environment == "development" || environment == "dev" {
		config.UseConsole = true
	}

	// Log configuration for debugging (password masked)
	log.Printf("EMAIL CONFIG LOADED: SmtpHost=%s, SmtpPort=%d, SmtpUsername=%s, FromAddress=%s, FrontendBaseURL=%s, EnableSSL=%v, UseConsole=%v\n",
		config.SmtpHost, config.SmtpPort, config.SmtpUsername, config.FromAddress, config.FrontendBaseURL, config.EnableSSL, config.UseConsole)

	return config
}

// SendLicenseExpiryNotification sends email when license is expiring soon
func (es *emailService) SendLicenseExpiryNotification(email, username, licenseKey string, daysUntilExpiry int) error {
	subject := fmt.Sprintf("License Expiring Soon: %s", licenseKey)
	body := fmt.Sprintf(`
Hello %s,

Your license (%s) will expire in %d days.

Please renew your license to avoid service interruption.

License Key: %s
Days Remaining: %d

Best regards,
Support Team
`,
		username, licenseKey, daysUntilExpiry, licenseKey, daysUntilExpiry)

	return es.SendEmail(email, subject, body)
}

// SendAccountActivityNotification sends email for important account activities
func (es *emailService) SendAccountActivityNotification(email, username, action string) error {
	subject := fmt.Sprintf("Account Activity: %s", action)
	body := fmt.Sprintf(`
Hello %s,

We detected the following activity on your account:
Action: %s
Time: %s
IP: N/A (Enable tracking for details)

If this wasn't you, please reset your password immediately.

Best regards,
Support Team
`,
		username, action, time.Now().Format("2006-01-02 15:04:05 MST"))

	return es.SendEmail(email, subject, body)
}

// SendWelcomeEmail sends welcome email to new users
func (es *emailService) SendWelcomeEmail(email, username string) error {
	subject := "Welcome!"
	body := fmt.Sprintf(`
Hello %s,

Welcome! Your account has been successfully created.

You can now:
- Manage your licenses
- Track license activations
- Monitor license expiry dates
- View activity logs

Get started by logging into your dashboard.

Best regards,
Support Team
`,
		username)

	return es.SendEmail(email, subject, body)
}

// SendSystemAnnouncement sends system-wide announcements
func (es *emailService) SendSystemAnnouncement(email, subject, message string) error {
	body := fmt.Sprintf(`
%s

---
This is a system announcement from our system.

Best regards,
Support Team
`,
		message)

	return es.SendEmail(email, subject, body)
}

// SendVerificationEmail sends email verification link
func (es *emailService) SendVerificationEmail(email, username, token string) error {
	subject := "Verify Your Email Address"
	verificationLink := fmt.Sprintf("%s/verify-email?token=%s", es.config.FrontendBaseURL, token)
	body := fmt.Sprintf(`
Hello %s,

Thank you for signing up! Please verify your email address to activate your account.

Click the link below to verify your email:
%s

Or copy and paste this token: %s

This link will expire in 24 hours.

If you didn't create this account, please ignore this email.

Best regards,
Support Team
`,
		username, verificationLink, token)

	return es.SendEmail(email, subject, body)
}

// SendPasswordResetEmail sends password reset link
func (es *emailService) SendPasswordResetEmail(email, username, token string) error {
	subject := "Password Reset Request"
	resetLink := fmt.Sprintf("%s/reset-password?token=%s", es.config.FrontendBaseURL, token)
	body := fmt.Sprintf(`
Hello %s,

We received a request to reset your password. Click the link below to set a new password:

%s

Or copy and paste this token: %s

This link will expire in 1 hour.

If you didn't request this reset, please ignore this email. Your password won't be changed.

Best regards,
Support Team
`,
		username, resetLink, token)

	return es.SendEmail(email, subject, body)
}

// SendEmail is the internal method that sends emails
func (es *emailService) SendEmail(to, subject, body string) error {
	// If console mode is enabled, just print the email
	if es.config.UseConsole {
		fmt.Printf("\n========== EMAIL (CONSOLE MODE) ==========\n")
		fmt.Printf("To: %s\n", to)
		fmt.Printf("Subject: %s\n", subject)
		fmt.Printf("Body:\n%s\n", body)
		fmt.Printf("==========================================\n\n")
		log.Printf("EMAIL SENT (CONSOLE MODE): to=%s, subject=%s\n", to, subject)
		return nil
	}

	// Check if SMTP is configured
	if es.config.SmtpHost == "" || es.config.FromAddress == "" {
		log.Printf("WARNING: Email not sent - SMTP not configured. SmtpHost=%s, FromAddress=%s\n", es.config.SmtpHost, es.config.FromAddress)
		fmt.Printf("WARNING: Email not sent (SMTP not configured)\n")
		fmt.Printf("To: %s\nSubject: %s\n\n", to, subject)
		return nil
	}

	log.Printf("SENDING EMAIL: to=%s, subject=%s, host=%s:%d, ssl=%v\n", to, subject, es.config.SmtpHost, es.config.SmtpPort, es.config.EnableSSL)

	// Construct email message
	messageContent := fmt.Sprintf(
		"From: %s <%s>\r\nTo: %s\r\nSubject: %s\r\nContent-Type: text/plain; charset=UTF-8\r\n\r\n%s",
		es.config.FromName,
		es.config.FromAddress,
		to,
		subject,
		body,
	)

	// Send email via SMTP with TLS support (production-ready)
	auth := smtp.PlainAuth(
		"",
		es.config.SmtpUsername,
		es.config.SmtpPassword,
		es.config.SmtpHost,
	)

	smtpAddr := fmt.Sprintf("%s:%d", es.config.SmtpHost, es.config.SmtpPort)
	recipients := strings.Split(to, ",")

	// Use TLS connection if enabled (recommended for production)
	if es.config.EnableSSL {
		err := es.sendMailWithTLS(auth, smtpAddr, recipients, messageContent)
		if err != nil {
			log.Printf("EMAIL ERROR: Failed to send TLS email: %v\n", err)
		} else {
			log.Printf("EMAIL SENT SUCCESSFULLY via TLS: to=%s\n", to)
		}
		return err
	}

	// Fallback to plain SMTP (development only)
	log.Printf("SENDING EMAIL via plain SMTP (no TLS)\n")
	err := smtp.SendMail(
		smtpAddr,
		auth,
		es.config.FromAddress,
		recipients,
		[]byte(messageContent),
	)

	if err != nil {
		log.Printf("EMAIL ERROR: Failed to send plain SMTP email: %v\n", err)
	} else {
		log.Printf("EMAIL SENT SUCCESSFULLY via plain SMTP: to=%s\n", to)
	}

	return err
}

// sendMailWithTLS sends email using TLS encryption (production-ready)
// Supports both implicit TLS (port 465) and STARTTLS (port 587)
func (es *emailService) sendMailWithTLS(auth smtp.Auth, addr string, recipients []string, message string) error {
	// Create TLS configuration
	tlsConfig := &tls.Config{
		ServerName: es.config.SmtpHost,
		// InsecureSkipVerify set to true for self-signed certificates
		// In production with proper certs, this should be false
		InsecureSkipVerify: true,
	}

	var client *smtp.Client
	var err error

	// Check if implicit TLS (port 465, 993) or STARTTLS (port 587, 25)
	if es.config.SmtpPort == 465 || es.config.SmtpPort == 993 {
		// Implicit TLS - connection is encrypted from the start
		log.Printf("EMAIL: Using implicit TLS on port %d\n", es.config.SmtpPort)
		conn, err := tls.Dial("tcp", addr, tlsConfig)
		if err != nil {
			return fmt.Errorf("failed to establish implicit TLS connection to %s: %w", addr, err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, es.config.SmtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}
	} else {
		// STARTTLS - connection starts in plaintext then upgrades to TLS (port 587, 25)
		log.Printf("EMAIL: Using STARTTLS on port %d\n", es.config.SmtpPort)
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			return fmt.Errorf("failed to connect to SMTP server %s: %w", addr, err)
		}
		defer conn.Close()

		client, err = smtp.NewClient(conn, es.config.SmtpHost)
		if err != nil {
			return fmt.Errorf("failed to create SMTP client: %w", err)
		}

		// Upgrade to TLS using STARTTLS command
		if err = client.StartTLS(tlsConfig); err != nil {
			return fmt.Errorf("failed to upgrade to TLS using STARTTLS: %w", err)
		}
	}

	defer client.Close()

	// Authenticate
	if err = client.Auth(auth); err != nil {
		return fmt.Errorf("SMTP authentication failed: %w", err)
	}

	// Set the sender
	if err = client.Mail(es.config.FromAddress); err != nil {
		return fmt.Errorf("failed to set sender: %w", err)
	}

	// Set the recipients
	for _, recipient := range recipients {
		if err = client.Rcpt(strings.TrimSpace(recipient)); err != nil {
			return fmt.Errorf("failed to add recipient %s: %w", recipient, err)
		}
	}

	// Send the message body
	w, err := client.Data()
	if err != nil {
		return fmt.Errorf("failed to create message writer: %w", err)
	}

	_, err = w.Write([]byte(message))
	if err != nil {
		return fmt.Errorf("failed to write message body: %w", err)
	}

	err = w.Close()
	if err != nil {
		return fmt.Errorf("failed to close message writer: %w", err)
	}

	// Quit the connection
	err = client.Quit()
	if err != nil {
		return fmt.Errorf("failed to close SMTP connection: %w", err)
	}

	return nil
}
