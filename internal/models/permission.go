package models

import "time"

// RolePermission defines which permissions are granted to a role
type RolePermission struct {
	ID         int        `gorm:"primaryKey" json:"id"`
	RoleID     string     `gorm:"index" json:"role_id"` // e.g., "Admin", "Manager", "User"
	Permission string     `gorm:"index" json:"permission"`
	CreatedAt  time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`
}

// TableName specifies the table name for RolePermission
func (RolePermission) TableName() string {
	return "role_permissions"
}

// UserPermission defines custom permissions for a specific user (overrides role permissions)
type UserPermission struct {
	ID         int        `gorm:"primaryKey" json:"id"`
	UserID     int        `gorm:"index" json:"user_id"`
	Permission string     `gorm:"index" json:"permission"`
	Granted    bool       `json:"granted"` // true = grant, false = revoke (deny)
	Reason     *string    `json:"reason,omitempty"`
	GrantedBy  *int       `json:"granted_by,omitempty"` // Admin who granted/revoked
	CreatedAt  time.Time  `gorm:"column:created_at;default:CURRENT_TIMESTAMP" json:"created_at"`
	UpdatedAt  *time.Time `gorm:"column:updated_at" json:"updated_at,omitempty"`

	// Relations
	User          *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
	GrantedByUser *User `gorm:"foreignKey:GrantedBy" json:"granted_by_user,omitempty"`
}

// TableName specifies the table name for UserPermission
func (UserPermission) TableName() string {
	return "user_permissions"
}
