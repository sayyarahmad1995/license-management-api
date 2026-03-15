package models

import (
	"time"
)

type AuditLog struct {
	ID         int       `gorm:"primaryKey" json:"id"`
	Action     string    `gorm:"column:action;index" json:"action"`
	EntityType string    `gorm:"column:entity_type;index" json:"entityType"`
	EntityID   *int      `gorm:"column:entity_id" json:"entityId"`
	UserID     *int      `gorm:"column:user_id;index" json:"userId"`
	Details    *string   `gorm:"column:details;type:text" json:"details"`
	IpAddress  *string   `gorm:"column:ip_address" json:"ipAddress"`
	Timestamp  time.Time `gorm:"column:timestamp;default:CURRENT_TIMESTAMP" json:"timestamp"`
}

// TableName specifies the table name for AuditLog model
func (AuditLog) TableName() string {
	return "audit_logs"
}

// NewAuditLog creates a new audit log entry
func NewAuditLog(action, entityType string, entityID, userID *int, details, ipAddress *string) *AuditLog {
	return &AuditLog{
		Action:     action,
		EntityType: entityType,
		EntityID:   entityID,
		UserID:     userID,
		Details:    details,
		IpAddress:  ipAddress,
		Timestamp:  time.Now().UTC(),
	}
}
