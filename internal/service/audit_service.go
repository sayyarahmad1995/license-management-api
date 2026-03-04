package service

import (
	"time"

	"license-management-api/internal/models"
	"license-management-api/internal/repository"
)

type AuditService interface {
	LogAction(action string, entityType string, entityID int, userID *int, details *string, ipAddress *string)
}

type auditService struct {
	auditRepo repository.IAuditLogRepository
}

func NewAuditService(auditRepo repository.IAuditLogRepository) AuditService {
	return &auditService{
		auditRepo: auditRepo,
	}
}

// LogAction logs an action to the audit trail
func (as *auditService) LogAction(action string, entityType string, entityID int, userID *int, details *string, ipAddress *string) {
	auditLog := &models.AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   &entityID,
		UserID:     userID,
		Details:    details,
		IpAddress:  ipAddress,
		Timestamp:  time.Now().UTC(),
	}

	_ = as.auditRepo.Create(auditLog)
}
