# License Management API - API Reference

**Version**: 1.0.0  
**Base URL**: `/api/v1`  
**Last Updated**: March 4, 2026

---

## Table of Contents

1. [Authentication](#authentication)
2. [Authorization](#authorization)
3. [Endpoints](#endpoints)
   - [Auth](#auth-endpoints)
   - [Licenses](#license-endpoints)
   - [Permissions](#permission-endpoints)
   - [Users](#user-endpoints)
   - [Dashboard](#dashboard-endpoints)
   - [Audit](#audit-endpoints)
4. [Error Handling](#error-handling)
5. [Rate Limiting](#rate-limiting)
6. [Best Practices](#best-practices)

---

## Authentication

All endpoints (except public ones) require JWT authentication via:
- **Cookie**: `accessToken` (preferred)
- **Bearer Token**: `Authorization: Bearer <token>`

### Public Endpoints
- `POST /api/v1/auth/login` - Login
- `POST /api/v1/auth/register` - Register
- `POST /api/v1/auth/verify-email` - Verify email
- `POST /api/v1/auth/resend-verification` - Resend verification
- `POST /api/v1/auth/request-password-reset` - Request password reset
- `POST /api/v1/auth/confirm-password-reset` - Confirm password reset
- `POST /api/v1/licenses/validate` - Validate license
- `POST /api/v1/licenses/activate` - Activate license
- `POST /api/v1/health` - Health check

---

## Authorization

### Role-Based Access Control

Three roles with hierarchical permissions:

| Role | Permissions | Use Case |
|------|---|---|
| **User** | 3 permissions (view licenses, activations, export) | End users, customers |
| **Manager** | 14 permissions (user/license view, audit access, analytics) | Team leads, support staff |
| **Admin** | 19 permissions (all except custom user permission mgmt) | System administrators |

### Permission System

20+ granular permissions across 5 categories:

**User Management**
- `VIEW_USERS` - View user list and details
- `CREATE_USERS` - Create new users
- `EDIT_USERS` - Edit user information
- `DELETE_USERS` - Delete users
- `MANAGE_ROLES` - Assign/modify roles
- `MANAGE_PERMISSIONS` - Grant/revoke custom permissions

**License Management**
- `VIEW_LICENSES` - View license list and details
- `CREATE_LICENSES` - Create new licenses
- `EDIT_LICENSES` - Edit license information
- `REVOKE_LICENSES` - Revoke licenses
- `BULK_REVOKE_LICENSES` - Revoke multiple licenses
- `VIEW_LICENSE_ACTIVATIONS` - View activation records
- `MANAGE_LICENSE_ACTIVATIONS` - Create/modify activations

**Audit & Analytics**
- `VIEW_AUDIT_LOGS` - View audit logs
- `VIEW_ANALYTICS` - View analytics dashboard
- `EXPORT_DATA` - Export user/license data

**System Management**
- `MANAGE_SYSTEM_SETTINGS` - Configure system settings
- `VIEW_SYSTEM_HEALTH` - View health metrics
- `MANAGE_NOTIFICATIONS` - Configure notifications

---

## Endpoints

### Auth Endpoints

#### Register
```
POST /api/v1/auth/register
Content-Type: application/json

{
  "email": "user@example.com",
  "username": "username",
  "password": "SecurePassword123!",
  "confirmPassword": "SecurePassword123!"
}

Response: 201 Created
{
  "message": "Registration successful",
  "user": {
    "id": 1,
    "email": "user@example.com",
    "username": "username",
    "role": "User"
  }
}
```

#### Login
```
POST /api/v1/auth/login
Content-Type: application/json

{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}

Response: 200 OK
{
  "accessToken": "eyJhbGc...",
  "refreshToken": "eyJhbGc...",
  "expiresIn": 3600,
  "user": {
    "id": 1,
    "email": "user@example.com",
    "role": "User"
  }
}
```

#### Get Current User
```
GET /api/v1/auth/me
Authorization: Bearer <token>

Response: 200 OK
{
  "id": 1,
  "email": "user@example.com",
  "username": "username",
  "role": "User",
  "createdAt": "2026-03-04T10:00:00Z",
  "emailVerified": true
}
```

#### Refresh Token
```
POST /api/v1/auth/refresh
Content-Type: application/json

{
  "refreshToken": "eyJhbGc..."
}

Response: 200 OK
{
  "accessToken": "eyJhbGc...",
  "expiresIn": 3600
}
```

#### Logout
```
POST /api/v1/auth/logout
Authorization: Bearer <token>

Response: 200 OK
{
  "message": "Logout successful"
}
```

#### List Active Sessions
```
GET /api/v1/auth/sessions
Authorization: Bearer <token>

Response: 200 OK
{
  "sessions": [
    {
      "sessionId": "sess_123",
      "userAgent": "Mozilla/5.0...",
      "ipAddress": "192.168.1.1",
      "createdAt": "2026-03-04T10:00:00Z",
      "lastActivity": "2026-03-04T10:30:00Z",
      "isCurrent": true
    }
  ],
  "totalCount": 1
}
```

#### Logout All Other Sessions
```
POST /api/v1/auth/logout-all-others
Authorization: Bearer <token>

Response: 200 OK
{
  "message": "All other sessions revoked",
  "sessionsRevoked": 2
}
```

### License Endpoints

#### Create License
```
POST /api/v1/licenses
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": 1,
  "expiresAt": "2027-03-04T00:00:00Z",
  "maxActivations": 5,
  "notes": "Enterprise license"
}

Response: 201 Created
{
  "id": 1,
  "licenseKey": "KEY-ABC123XYZ",
  "status": "active",
  "expiresAt": "2027-03-04T00:00:00Z",
  "createdAt": "2026-03-04T10:00:00Z"
}
```

#### Get All Licenses
```
GET /api/v1/licenses?page=1&pageSize=20&status=active
Authorization: Bearer <token>

Response: 200 OK
{
  "data": [
    {
      "id": 1,
      "licenseKey": "KEY-ABC123XYZ",
      "userId": 1,
      "status": "active",
      "expiresAt": "2027-03-04T00:00:00Z",
      "maxActivations": 5,
      "activeActivations": 2,
      "createdAt": "2026-03-04T10:00:00Z"
    }
  ],
  "pagination": {
    "totalCount": 42,
    "page": 1,
    "pageSize": 20,
    "totalPages": 3
  }
}
```

#### Validate License
```
POST /api/v1/licenses/validate
Content-Type: application/json

{
  "licenseKey": "KEY-ABC123XYZ"
}

Response: 200 OK
{
  "valid": true,
  "licenseId": 1,
  "userId": 1,
  "status": "active",
  "expiresAt": "2027-03-04T00:00:00Z",
  "canActivate": true
}
```

#### Activate License
```
POST /api/v1/licenses/activate
Content-Type: application/json
Rate Limited: YES (5 attempts per 15 minutes)

{
  "licenseKey": "KEY-ABC123XYZ",
  "machineId": "MACHINE-UUID",
  "machineName": "Developer Laptop",
  "fingerprint": "..."
}

Response: 200 OK OR 429 Too Many Requests
{
  "activationId": 1,
  "active": true,
  "activatedAt": "2026-03-04T10:00:00Z",
  "expiresAt": "2027-03-04T00:00:00Z"
}
```

#### Revoke License
```
DELETE /api/v1/licenses/{id}
Authorization: Bearer <token>

Response: 200 OK
{
  "message": "License revoked",
  "revokedAt": "2026-03-04T10:00:00Z"
}
```

#### Bulk Revoke Licenses (Sync)
```
POST /api/v1/licenses/bulk-revoke
Authorization: Bearer <token>
Content-Type: application/json

{
  "licenseIds": [1, 2, 3],
  "reason": "Contract ended"
}

Response: 200 OK
{
  "totalProcessed": 3,
  "successful": 3,
  "failed": 0,
  "results": [
    {
      "licenseId": 1,
      "status": "success",
      "message": "License revoked"
    }
  ]
}
```

#### Bulk Revoke Licenses (Async)
```
POST /api/v1/licenses/bulk-revoke-async
Authorization: Bearer <token>
Content-Type: application/json

{
  "licenseIds": [1, 2, 3, ...],
  "reason": "Contract ended"
}

Response: 202 Accepted
{
  "jobId": "job_abc123",
  "status": "processing",
  "message": "Bulk revoke operation started",
  "createdAt": "2026-03-04T10:00:00Z"
}
```

#### Get Bulk Job Status
```
GET /api/v1/licenses/bulk-jobs/{jobId}
Authorization: Bearer <token>

Response: 200 OK
{
  "jobId": "job_abc123",
  "status": "processing",
  "progress": {
    "processed": 50,
    "total": 100,
    "percentage": 50
  },
  "createdAt": "2026-03-04T10:00:00Z",
  "updatedAt": "2026-03-04T10:05:00Z"
}
```

#### Cancel Bulk Job
```
POST /api/v1/licenses/bulk-jobs/{jobId}/cancel
Authorization: Bearer <token>

Response: 200 OK
{
  "message": "Job cancelled",
  "jobId": "job_abc123"
}
```

### Permission Endpoints

All permission endpoints require **Admin role**.

#### List User Permissions
```
GET /api/v1/permissions/users/{userId}
Authorization: Bearer <token> (Admin only)

Response: 200 OK
{
  "userId": 1,
  "role": "Manager",
  "totalCount": 14,
  "permissions": [
    "VIEW_USERS",
    "VIEW_LICENSES",
    "VIEW_AUDIT_LOGS",
    ...
  ],
  "customGrants": [
    {
      "permission": "DELETE_USERS",
      "grantedBy": 2,
      "grantedAt": "2026-01-01T00:00:00Z",
      "reason": "Needed for cleanup"
    }
  ],
  "customRevokes": [
    {
      "permission": "CREATE_LICENSES",
      "revokedBy": 2,
      "revokedAt": "2026-01-15T00:00:00Z",
      "reason": "Temporary restriction"
    }
  ]
}
```

#### List Role Permissions
```
GET /api/v1/permissions/roles/{role}
Authorization: Bearer <token> (Admin only)

Response: 200 OK
{
  "role": "Manager",
  "totalCount": 14,
  "permissions": [
    "VIEW_USERS",
    "VIEW_LICENSES",
    "VIEW_AUDIT_LOGS",
    ...
  ],
  "lastUpdated": "2026-02-01T00:00:00Z"
}
```

#### Grant Permission to User
```
POST /api/v1/permissions/grant
Authorization: Bearer <token> (Admin only)
Content-Type: application/json

{
  "userId": 1,
  "permission": "DELETE_USERS",
  "reason": "Needed for cleanup task"
}

Response: 200 OK
{
  "status": "success",
  "message": "Permission granted successfully",
  "userId": 1,
  "permission": "DELETE_USERS",
  "timestamp": "2026-03-04T10:00:00Z"
}
```

#### Revoke Permission from User
```
POST /api/v1/permissions/revoke
Authorization: Bearer <token> (Admin only)
Content-Type: application/json

{
  "userId": 1,
  "permission": "CREATE_LICENSES",
  "reason": "Temporary security lockdown"
}

Response: 200 OK
{
  "status": "success",
  "message": "Permission revoked successfully",
  "userId": 1,
  "permission": "CREATE_LICENSES",
  "timestamp": "2026-03-04T10:00:00Z"
}
```

#### Set Role Permissions
```
POST /api/v1/permissions/roles
Authorization: Bearer <token> (Admin only)
Content-Type: application/json

{
  "role": "Manager",
  "permissions": [
    "VIEW_USERS",
    "VIEW_LICENSES",
    "VIEW_AUDIT_LOGS"
  ]
}

Response: 200 OK
{
  "role": "Manager",
  "totalCount": 3,
  "permissions": [
    "VIEW_USERS",
    "VIEW_LICENSES",
    "VIEW_AUDIT_LOGS"
  ]
}
```

#### Reset User Permissions to Role Defaults
```
POST /api/v1/permissions/users/{userId}/reset
Authorization: Bearer <token> (Admin only)

Response: 200 OK
{
  "status": "success",
  "message": "User permissions reset to role defaults",
  "userId": 1,
  "role": "Manager"
}
```

#### Check Permissions
```
POST /api/v1/permissions/check
Authorization: Bearer <token>
Content-Type: application/json

{
  "userId": 1,
  "permissions": ["VIEW_USERS", "CREATE_USERS"],
  "requireAll": false
}

Response: 200 OK
{
  "userId": 1,
  "requested": ["VIEW_USERS", "CREATE_USERS"],
  "granted": ["VIEW_USERS"],
  "denied": ["CREATE_USERS"],
  "hasAllRequested": false,
  "hasAnyRequested": true
}
```

### Dashboard Endpoints

#### Get License Expiration Forecast
```
GET /api/v1/dashboard/forecast
Authorization: Bearer <token>

Response: 200 OK
{
  "forecast": {
    "next7Days": 2,
    "next30Days": 15,
    "next90Days": 45,
    "details": [
      {
        "period": "7days",
        "count": 2,
        "licenses": [
          {
            "id": 1,
            "licenseKey": "KEY-ABC",
            "expiresAt": "2026-03-08T00:00:00Z",
            "userId": 1
          }
        ]
      }
    ]
  }
}
```

#### Get Activity Timeline
```
GET /api/v1/dashboard/activity-timeline
Authorization: Bearer <token>

Response: 200 OK
{
  "timeline": {
    "period": "7days",
    "data": [
      {
        "date": "2026-03-04",
        "activations": 5,
        "deactivations": 1,
        "revocations": 0,
        "newLicenses": 2
      }
    ]
  }
}
```

#### Get Usage Analytics
```
GET /api/v1/dashboard/analytics
Authorization: Bearer <token>

Response: 200 OK
{
  "analytics": {
    "totalLicenses": 100,
    "activeLicenses": 85,
    "expiredLicenses": 10,
    "revokedLicenses": 5,
    "activeActivations": 120,
    "totalActivations": 150,
    "topUsers": [
      {
        "userId": 1,
        "username": "john.doe",
        "licenseCount": 10,
        "activCount": 15
      }
    ],
    "distribution": {
      "byStatus": {
        "active": 85,
        "expired": 10,
        "revoked": 5
      },
      "byUser": {
        "1": 10,
        "2": 8,
        "3": 5
      }
    }
  }
}
```

### User Endpoints

#### List Users
```
GET /api/v1/users?page=1&pageSize=20
Authorization: Bearer <token>

Response: 200 OK
{
  "data": [
    {
      "id": 1,
      "email": "user@example.com",
      "username": "username",
      "role": "User",
      "status": "active",
      "createdAt": "2026-03-04T10:00:00Z",
      "emailVerified": true
    }
  ],
  "pagination": {
    "totalCount": 50,
    "page": 1,
    "pageSize": 20,
    "totalPages": 3
  }
}
```

#### Get User
```
GET /api/v1/users/{id}
Authorization: Bearer <token>

Response: 200 OK
{
  "id": 1,
  "email": "user@example.com",
  "username": "username",
  "role": "User",
  "status": "active",
  "createdAt": "2026-03-04T10:00:00Z",
  "emailVerified": true,
  "licenseCount": 5,
  "lastLogin": "2026-03-04T15:00:00Z"
}
```

#### Update User Role
```
PATCH /api/v1/users/{id}/role
Authorization: Bearer <token> (Admin only)
Content-Type: application/json

{
  "role": "Manager"
}

Response: 200 OK
{
  "id": 1,
  "email": "user@example.com",
  "role": "Manager",
  "updatedAt": "2026-03-04T10:00:00Z"
}
```

### Audit Endpoints

#### List Audit Logs
```
GET /api/v1/audit-logs?page=1&pageSize=50&action=CREATE_LICENSE
Authorization: Bearer <token>

Response: 200 OK
{
  "data": [
    {
      "id": 1,
      "action": "CREATE_LICENSE",
      "userId": 1,
      "resourceType": "License",
      "resourceId": "123",
      "details": {
        "licenseKey": "KEY-ABC123"
      },
      "ipAddress": "192.168.1.1",
      "userAgent": "Mozilla/5.0...",
      "change": {
        "before": null,
        "after": {
          "status": "active"
        }
      },
      "severity": "INFO",
      "timestamp": "2026-03-04T10:00:00Z"
    }
  ],
  "pagination": {
    "totalCount": 500,
    "page": 1,
    "pageSize": 50,
    "totalPages": 10
  }
}
```

---

## Error Handling

All errors follow this format:

```json
{
  "error": "Error Title",
  "message": "Detailed error message",
  "code": "ERROR_CODE",
  "timestamp": "2026-03-04T10:00:00Z",
  "requestId": "req_abc123"
}
```

### Common HTTP Status Codes

| Status | Meaning | Example |
|--------|---------|---------|
| 200 | OK | Successful request |
| 201 | Created | Resource created |
| 202 | Accepted | Async operation started |
| 400 | Bad Request | Invalid input |
| 401 | Unauthorized | Missing/invalid token |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource not found |
| 409 | Conflict | Resource already exists |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |

---

## Rate Limiting

Rate limits protect endpoints from abuse:

| Endpoint | Limit | Window |
|----------|-------|--------|
| Login | 5 attempts | 15 minutes |
| Register | 3 attempts | 1 hour |
| Email Verify | 5 attempts | 15 minutes |
| Password Reset | 3 attempts | 1 hour |
| License Activate | 5 attempts | 15 minutes |

When rate limited, you'll receive:
```
429 Too Many Requests
Retry-After: 600

{
  "error": "Rate Limit Exceeded",
  "message": "Too many requests. Try again in 10 minutes.",
  "retryAfter": 600
}
```

---

## Best Practices

### Authentication
- Always use HTTPS in production
- Store tokens securely (httpOnly cookies preferred)
- Refresh tokens before expiration
- Never expose tokens in URLs

### Permissions
- Check required permissions before API calls
- Use `/permissions/check` on frontend for conditional UI
- Request minimal required permissions
- Review custom permissions regularly

### Pagination
- Always request reasonable page sizes (10-100 items)
- Handle `totalCount` for pagination UI
- Don't assume page availability (check `totalPages`)

### Error Handling
- Always check response status codes
- Extract `requestId` for support debugging
- Implement exponential backoff for retries
- Don't retry on 4xx errors

### Performance
- Use pagination for list endpoints
- Filter/search server-side when possible
- Batch operations (bulk license revoke)
- Cache permission checks (~1 hour TTL)

---

## Support

For API issues, include:
- Request ID (`requestId` in error response)
- Endpoint and method
- Request/response bodies
- Error message and code
- Any custom header extensions

Contact: support@example.com


