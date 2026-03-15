package models

// UserStatus represents the status of a user account
type UserStatus string

const (
	UserStatusUnverified UserStatus = "Unverified"
	UserStatusVerified   UserStatus = "Verified"
	UserStatusActive     UserStatus = "Active"
	UserStatusBlocked    UserStatus = "Blocked"
)

// LicenseStatus represents the status of a license
type LicenseStatus string

const (
	LicenseStatusActive  LicenseStatus = "Active"
	LicenseStatusRevoked LicenseStatus = "Revoked"
)

// UserRole represents user roles
type UserRole string

const (
	UserRoleUser    UserRole = "User"
	UserRoleManager UserRole = "Manager"
	UserRoleAdmin   UserRole = "Admin"
)

// Permission represents fine-grained permissions
type Permission string

const (
	// User management permissions
	PermissionViewUsers         Permission = "view_users"
	PermissionCreateUsers       Permission = "create_users"
	PermissionEditUsers         Permission = "edit_users"
	PermissionDeleteUsers       Permission = "delete_users"
	PermissionManageRoles       Permission = "manage_roles"
	PermissionManagePermissions Permission = "manage_permissions"

	// License management permissions
	PermissionViewLicenses      Permission = "view_licenses"
	PermissionCreateLicenses    Permission = "create_licenses"
	PermissionEditLicenses      Permission = "edit_licenses"
	PermissionRevokeLicenses    Permission = "revoke_licenses"
	PermissionBulkRevoke        Permission = "bulk_revoke"
	PermissionViewActivations   Permission = "view_activations"
	PermissionManageActivations Permission = "manage_activations"

	// Audit and analytics
	PermissionViewAuditLogs Permission = "view_audit_logs"
	PermissionViewAnalytics Permission = "view_analytics"
	PermissionExportData    Permission = "export_data"

	// System management
	PermissionManageSettings      Permission = "manage_settings"
	PermissionViewSystemHealth    Permission = "view_system_health"
	PermissionManageNotifications Permission = "manage_notifications"
)

// DefaultPermissionsByRole defines default permissions for each role
var DefaultPermissionsByRole = map[UserRole][]Permission{
	UserRoleUser: {
		PermissionViewLicenses,
		PermissionViewActivations,
		PermissionExportData,
	},
	UserRoleManager: {
		PermissionViewUsers,
		PermissionViewLicenses,
		PermissionCreateLicenses,
		PermissionEditLicenses,
		PermissionRevokeLicenses,
		PermissionViewActivations,
		PermissionManageActivations,
		PermissionViewAuditLogs,
		PermissionViewAnalytics,
		PermissionExportData,
	},
	UserRoleAdmin: {
		PermissionViewUsers,
		PermissionCreateUsers,
		PermissionEditUsers,
		PermissionDeleteUsers,
		PermissionManageRoles,
		PermissionManagePermissions,
		PermissionViewLicenses,
		PermissionCreateLicenses,
		PermissionEditLicenses,
		PermissionRevokeLicenses,
		PermissionBulkRevoke,
		PermissionViewActivations,
		PermissionManageActivations,
		PermissionViewAuditLogs,
		PermissionViewAnalytics,
		PermissionExportData,
		PermissionManageSettings,
		PermissionViewSystemHealth,
		PermissionManageNotifications,
	},
}
