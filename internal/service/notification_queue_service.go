package service

import (
	"fmt"
	"log/slog"
	"strings"
	"sync"
	"time"

	"license-management-api/internal/dto"
	"license-management-api/internal/errors"
)

// NotificationQueueService handles async email notifications
type NotificationQueueService interface {
	QueueNotification(notification *dto.EmailNotification) error
	QueueEmailWithTemplate(to string, template dto.EmailTemplate, variables map[string]interface{}, priority string) error
	ProcessQueue() error
	GetQueueSize() int
	Start()
	Stop()
	SetEmailService(emailService EmailService)
}

// notificationQueueService implements NotificationQueueService
type notificationQueueService struct {
	queue          []*dto.EmailNotification
	mu             sync.RWMutex
	queueSize      int
	maxRetries     int
	ticker         *time.Ticker
	stopChan       chan bool
	logger         *slog.Logger
	isRunning      bool
	emailService   EmailService
	retryBackoff   float64 // exponential backoff multiplier
	baseRetryDelay time.Duration
}

// NewNotificationQueueService creates a new notification queue service
func NewNotificationQueueService() NotificationQueueService {
	return &notificationQueueService{
		queue:          make([]*dto.EmailNotification, 0),
		queueSize:      100,
		maxRetries:     3,
		stopChan:       make(chan bool),
		logger:         slog.Default(),
		isRunning:      false,
		emailService:   nil, // Will be set via SetEmailService
		retryBackoff:   2.0, // exponential backoff factor
		baseRetryDelay: 5 * time.Second,
	}
}

// SetEmailService sets the email service dependency
func (nqs *notificationQueueService) SetEmailService(emailService EmailService) {
	nqs.mu.Lock()
	defer nqs.mu.Unlock()
	nqs.emailService = emailService
}

// QueueNotification adds a notification to the queue
func (nqs *notificationQueueService) QueueNotification(notification *dto.EmailNotification) error {
	if notification == nil {
		return errors.NewValidationError("Notification cannot be nil")
	}

	if notification.To == "" {
		return errors.NewValidationError("Notification email address is required")
	}

	nqs.mu.Lock()
	defer nqs.mu.Unlock()

	// Check queue size
	if len(nqs.queue) >= nqs.queueSize {
		return errors.NewInternalError("Notification queue is full")
	}

	notification.CreatedAt = time.Now().UTC()
	notification.Status = "pending"
	nqs.queue = append(nqs.queue, notification)

	nqs.logger.Debug("Notification queued", "to", notification.To, "template", notification.Template)

	return nil
}

// QueueEmailWithTemplate queues an email with a template
func (nqs *notificationQueueService) QueueEmailWithTemplate(to string, template dto.EmailTemplate, variables map[string]interface{}, priority string) error {
	if to == "" {
		return errors.NewValidationError("Email address is required")
	}

	templateContent := dto.GetEmailTemplates()[template]
	if templateContent == nil {
		return errors.NewValidationError(fmt.Sprintf("Unknown email template: %s", template))
	}

	notification := &dto.EmailNotification{
		ID:        fmt.Sprintf("notif_%d_%s", time.Now().UnixNano(), to),
		To:        to,
		Template:  template,
		Subject:   templateContent.Subject,
		Variables: variables,
		Priority:  priority,
		Status:    "pending",
	}

	return nqs.QueueNotification(notification)
}

// ProcessQueue processes all pending notifications in the queue
func (nqs *notificationQueueService) ProcessQueue() error {
	nqs.mu.Lock()
	defer nqs.mu.Unlock()

	if len(nqs.queue) == 0 {
		return nil
	}

	// If no email service configured, just log and skip
	if nqs.emailService == nil {
		nqs.logger.Warn("Email service not configured, skipping email sending")
		return nil
	}

	nqs.logger.Info("Processing notification queue", "size", len(nqs.queue))

	for i, notification := range nqs.queue {
		if notification.Status != "pending" {
			continue
		}

		// Try to send email with retry mechanism
		err := nqs.sendNotificationWithRetry(notification)

		if err == nil {
			// Email sent successfully
			notification.Status = "sent"
			now := time.Now().UTC()
			notification.SentAt = &now
			notification.UpdatedAt = now
			notification.RetryCount = 0
			nqs.logger.Info("Email sent successfully",
				"to", notification.To,
				"template", notification.Template,
				"subject", notification.Subject)
		} else {
			// Increment retry count
			notification.RetryCount++

			if notification.RetryCount >= nqs.maxRetries {
				// Max retries exceeded
				notification.Status = "failed"
				now := time.Now().UTC()
				notification.UpdatedAt = now
				nqs.logger.Error("Email delivery failed after max retries",
					"to", notification.To,
					"template", notification.Template,
					"retries", notification.RetryCount,
					"error", err)
			} else {
				// Still pending, will retry next cycle
				now := time.Now().UTC()
				notification.UpdatedAt = now
				nqs.logger.Warn("Email delivery failed, will retry",
					"to", notification.To,
					"template", notification.Template,
					"retries", notification.RetryCount,
					"maxRetries", nqs.maxRetries,
					"error", err)
			}
		}

		nqs.queue[i] = notification
	}

	// Clean up sent and failed notifications from queue
	var remainingQueue []*dto.EmailNotification
	for _, notification := range nqs.queue {
		if notification.Status == "pending" {
			remainingQueue = append(remainingQueue, notification)
		}
	}
	nqs.queue = remainingQueue

	return nil
}

// sendNotificationWithRetry sends a notification with inline retry logic
func (nqs *notificationQueueService) sendNotificationWithRetry(notification *dto.EmailNotification) error {
	// Render template with variables if defined
	body := ""
	subject := notification.Subject

	// Use template if specified
	if notification.Template != "" {
		templates := dto.GetEmailTemplates()
		if templateContent, exists := templates[notification.Template]; exists {
			body = renderTemplate(templateContent.HTML, notification.Variables)
			if notification.Subject == "" {
				subject = templateContent.Subject
			}
		}
	}

	// If no template was used, check if there's a specific email body content
	// For now, body remains empty if no template is provided
	if body == "" && notification.Subject != "" {
		// Fall back to subject as body if no template
		body = notification.Subject
	}

	// Send email through email service
	err := nqs.emailService.SendEmail(notification.To, subject, body)
	return err
}

// GetQueueSize returns the current queue size
func (nqs *notificationQueueService) GetQueueSize() int {
	nqs.mu.RLock()
	defer nqs.mu.RUnlock()
	return len(nqs.queue)
}

// Start starts the notification queue processor
func (nqs *notificationQueueService) Start() {
	if nqs.isRunning {
		return
	}

	nqs.isRunning = true
	nqs.ticker = time.NewTicker(30 * time.Second) // Process queue every 30 seconds

	go func() {
		for {
			select {
			case <-nqs.ticker.C:
				if err := nqs.ProcessQueue(); err != nil {
					nqs.logger.Error("Failed to process notification queue", "error", err)
				}
			case <-nqs.stopChan:
				nqs.ticker.Stop()
				nqs.isRunning = false
				nqs.logger.Info("Notification queue processor stopped")
				return
			}
		}
	}()

	nqs.logger.Info("Notification queue processor started")
}

// Stop stops the notification queue processor
func (nqs *notificationQueueService) Stop() {
	if !nqs.isRunning {
		return
	}
	nqs.stopChan <- true
}

// renderEmailTemplate renders an email template with variable substitution
func renderEmailTemplate(templateName dto.EmailTemplate, subject string, variables map[string]interface{}) *dto.RenderedEmail {
	if variables == nil {
		variables = make(map[string]interface{})
	}

	templates := dto.GetEmailTemplates()
	if templateContent, exists := templates[templateName]; exists {
		// Template substitution - replace {{variable}} with actual values
		body := renderTemplate(templateContent.HTML, variables)
		renderedSubject := renderTemplate(subject, variables)

		return &dto.RenderedEmail{
			Subject: renderedSubject,
			Body:    body,
		}
	}

	return nil
}

// renderTemplate performs simple {{variable}} substitution
func renderTemplate(template string, variables map[string]interface{}) string {
	result := template
	for key, value := range variables {
		placeholder := fmt.Sprintf("{{%s}}", key)
		replacement := fmt.Sprintf("%v", value)
		result = strings.ReplaceAll(result, placeholder, replacement)
	}
	return result
}
