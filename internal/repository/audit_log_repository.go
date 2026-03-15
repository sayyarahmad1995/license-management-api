package repository

import (
	"gorm.io/gorm"
	"license-management-api/internal/models"
)

// IAuditLogRepository defines audit log-specific operations
type IAuditLogRepository interface {
	IRepository[models.AuditLog]
	GetByUserID(userID int) ([]models.AuditLog, error)
	GetByEntityType(entityType string) ([]models.AuditLog, error)
	DeleteOlderThan(days int) error
	DeleteAll() error
}

// AuditLogRepository implements IAuditLogRepository
type AuditLogRepository struct {
	*GenericRepository[models.AuditLog]
	db *gorm.DB
}

// NewAuditLogRepository creates a new instance
func NewAuditLogRepository(db *gorm.DB) IAuditLogRepository {
	return &AuditLogRepository{
		GenericRepository: &GenericRepository[models.AuditLog]{db: db},
		db:                db,
	}
}

// GetByUserID retrieves audit logs for a specific user
func (r *AuditLogRepository) GetByUserID(userID int) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := r.db.Where("user_id = ?", userID).Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

// GetByEntityType retrieves audit logs for a specific entity type
func (r *AuditLogRepository) GetByEntityType(entityType string) ([]models.AuditLog, error) {
	var logs []models.AuditLog
	err := r.db.Where("entity_type = ?", entityType).Order("timestamp DESC").Find(&logs).Error
	return logs, err
}

// DeleteOlderThan deletes audit logs older than a specified number of days
func (r *AuditLogRepository) DeleteOlderThan(days int) error {
	return r.db.Where("timestamp < NOW() - interval '? days'", days).Delete(&models.AuditLog{}).Error
}

// DeleteAll deletes all audit logs
func (r *AuditLogRepository) DeleteAll() error {
	return r.db.Session(&gorm.Session{AllowGlobalUpdate: true}).Delete(&models.AuditLog{}).Error
}
