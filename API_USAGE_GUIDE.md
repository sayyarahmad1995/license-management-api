# License Management API - User Guide

A comprehensive guide for end-users and developers to interact with the License Management API.

## Table of Contents

1. [Getting Started](#getting-started)
2. [Authentication](#authentication)
3. [Core Features](#core-features)
4. [Dashboard & Statistics](#dashboard--statistics)
5. [License Management](#license-management)
6. [API Endpoints Reference](#api-endpoints-reference)
7. [Common Use Cases](#common-use-cases)

---

## Getting Started

### Base URL

```
http://localhost:8080/api/v1
```

### Required Headers

All requests (except authentication) require:

```
Authorization: Bearer <your_access_token>
Content-Type: application/json
```

### API Response Format

All responses follow a consistent JSON structure:

```json
{
  "message": "Operation successful",
  "data": { ... }
}
```

---

## Authentication

### 1. Register a New Account

**Endpoint:** `POST /auth/register`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!",
  "fullName": "John Doe"
}
```

**Response:**
```json
{
  "message": "Account created successfully",
  "userId": 1,
  "email": "user@example.com"
}
```

### 2. Login

**Endpoint:** `POST /auth/login`

**Request Body:**
```json
{
  "email": "user@example.com",
  "password": "SecurePassword123!"
}
```

**Response:**
```json
{
  "message": "Login successful",
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresIn": 3600,
  "user": {
    "id": 1,
    "email": "user@example.com",
    "fullName": "John Doe",
    "role": "USER"
  }
}
```

### 3. Logout

**Endpoint:** `POST /auth/logout`

**Headers:**
```
Authorization: Bearer <access_token>
```

**Response:**
```json
{
  "message": "Logged out successfully"
}
```

### 4. Refresh Access Token

**Endpoint:** `POST /auth/refresh`

**Note:** The refresh token is automatically read from the `refreshToken` cookie. No request body needed.

**Response:**
```json
{
  "message": "Token refreshed successfully"
}
```

The new `accessToken` is automatically set in the `accessToken` cookie.

### 5. Rotate Tokens (Enhanced Security)

**Endpoint:** `POST /auth/rotate`

**Description:** Generates a new token pair and revokes the old refresh token. Useful for periodic token refresh or after sensitive operations.

**Note:** The refresh token is automatically read from the `refreshToken` cookie. No request body needed.

**Response:**
```json
{
  "accessToken": "eyJhbGciOiJIUzI1NiIs...",
  "refreshToken": "eyJhbGciOiJIUzI1NiIs...",
  "expiresAt": "2026-03-05T15:22:00Z",
  "refreshExpiresAt": "2026-03-12T14:22:00Z"
}
```

Both new `accessToken` and `refreshToken` are automatically set in their respective cookies. The old refresh token is automatically revoked.

---

## Core Features

### User Profile

**Get Current User Info**

**Endpoint:** `GET /auth/me`

**Response:**
```json
{
  "message": "User information retrieved",
  "id": 1,
  "email": "user@example.com",
  "fullName": "John Doe",
  "role": "USER",
  "createdAt": "2026-03-01T10:30:00Z",
  "lastLogin": "2026-03-05T14:22:00Z"
}
```

### Change Password

**Endpoint:** `POST /auth/change-password`

**Request Body:**
```json
{
  "currentPassword": "OldPassword123!",
  "newPassword": "NewPassword456!",
  "confirmPassword": "NewPassword456!"
}
```

**Response:**
```json
{
  "message": "Password changed successfully"
}
```

---

## Dashboard & Statistics

### User's Personal Dashboard

**Endpoint:** `GET /dashboard`

Returns aggregated statistics for the authenticated user.

**Response:**
```json
{
  "userId": 1,
  "licenses": {
    "total": 5,        // Total licenses owned
    "active": 3,       // Currently active
    "expired": 1,      // Expired licenses
    "expiringSoon": 1  // Expiring within 30 days
  },
  "activations": {
    "total": 8         // Total license activations
  },
  "recent_activity": [
    {
      "id": 1,
      "userId": 1,
      "action": "LOGIN",
      "description": "User logged in",
      "createdAt": "2026-03-05T14:22:00Z"
    },
    ...
  ]
}
```

### License Expiration Forecast

**Endpoint:** `GET /dashboard/forecast`

**Response:**
```json
{
  "forecast": [
    {
      "period": "Next 7 days",
      "expiringCount": 2
    },
    {
      "period": "Next 30 days",
      "expiringCount": 5
    },
    {
      "period": "Next 90 days",
      "expiringCount": 8
    }
  ]
}
```

### Activity Timeline

**Endpoint:** `GET /dashboard/activity-timeline`

Shows chronological history of user actions.

**Response:**
```json
{
  "timeline": [
    {
      "id": 1,
      "action": "LICENSE_CREATED",
      "description": "Created license ABC-123-XYZ",
      "timestamp": "2026-03-05T14:20:00Z"
    },
    {
      "id": 2,
      "action": "LICENSE_ACTIVATED",
      "description": "Activated license on machine PC-001",
      "timestamp": "2026-03-05T14:21:00Z"
    }
  ]
}
```

### Usage Analytics

**Endpoint:** `GET /dashboard/analytics`

**Response:**
```json
{
  "analyticsData": {
    "licensesPerMonth": [
      { "month": "January", "count": 2 },
      { "month": "February", "count": 3 }
    ],
    "activationsPerMonth": [
      { "month": "January", "count": 5 },
      { "month": "February", "count": 8 }
    ],
    "averageActivationTime": "2.5 hours"
  }
}
```

### Enhanced Dashboard Statistics

**Endpoint:** `GET /dashboard/enhanced`

Comprehensive dashboard with all metrics combined.

**Response:**
```json
{
  "summary": {
    "totalLicenses": 5,
    "activeLicenses": 3,
    "totalActivations": 8
  },
  "details": {
    "licenses": {...},
    "forecast": {...},
    "analytics": {...}
  }
}
```

### Admin Dashboard (Admin Only)

**Endpoint:** `GET /dashboard/admin` or `GET /dashboard/stats`

System-wide statistics (requires Admin role).

**Response:**
```json
{
  "totalUsers": 50,
  "totalLicenses": 250,
  "totalActivations": 500,
  "activeUsers": 45,
  "licensesAboutToExpire": 12,
  "averageActivationsPerLicense": 2.5
}
```

---

## License Management

### Create a License

**Endpoint:** `POST /licenses`

**Request Body:**
```json
{
  "licenseKey": "ABC-123-XYZ-789",
  "name": "Enterprise License",
  "description": "Annual enterprise subscription",
  "expiresAt": "2027-03-05T00:00:00Z",
  "maxActivations": 5,
  "features": ["feature1", "feature2"]
}
```

**Response:**
```json
{
  "message": "License created successfully",
  "licenseId": 1,
  "licenseKey": "ABC-123-XYZ-789",
  "status": "ACTIVE"
}
```

### Get License Details

**Endpoint:** `GET /licenses/{licenseId}`

**Response:**
```json
{
  "id": 1,
  "licenseKey": "ABC-123-XYZ-789",
  "name": "Enterprise License",
  "description": "Annual enterprise subscription",
  "status": "ACTIVE",
  "createdAt": "2026-01-15T10:30:00Z",
  "expiresAt": "2027-03-05T00:00:00Z",
  "maxActivations": 5,
  "currentActivations": 3,
  "features": ["feature1", "feature2"]
}
```

### List All Licenses

**Endpoint:** `GET /licenses?page=1&limit=10`

**Response:**
```json
{
  "message": "Licenses retrieved",
  "data": [
    { ... },
    { ... }
  ],
  "pagination": {
    "total": 25,
    "page": 1,
    "limit": 10,
    "pages": 3
  }
}
```

### Validate License

**Endpoint:** `POST /licenses/validate`

**Request Body:**
```json
{
  "licenseKey": "ABC-123-XYZ-789"
}
```

**Response:**
```json
{
  "message": "License is valid",
  "isValid": true,
  "licenseKey": "ABC-123-XYZ-789",
  "status": "ACTIVE",
  "expiresAt": "2027-03-05T00:00:00Z",
  "daysRemaining": 365
}
```

### Activate License

**Endpoint:** `POST /licenses/activate`

**Request Body:**
```json
{
  "licenseKey": "ABC-123-XYZ-789",
  "machineId": "PC-001",
  "machineName": "Developer Workstation"
}
```

**Response:**
```json
{
  "message": "License activated successfully",
  "activationId": 1,
  "licenseKey": "ABC-123-XYZ-789",
  "machineId": "PC-001",
  "activatedAt": "2026-03-05T14:22:00Z"
}
```

### Deactivate License

**Endpoint:** `POST /licenses/deactivate`

**Request Body:**
```json
{
  "licenseKey": "ABC-123-XYZ-789",
  "machineId": "PC-001"
}
```

**Response:**
```json
{
  "message": "License deactivated successfully"
}
```

### Delete License

**Endpoint:** `DELETE /licenses/{licenseId}`

**Response:**
```json
{
  "message": "License deleted successfully"
}
```

---

## API Endpoints Reference

### Authentication Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|-----------------|
| POST | `/auth/register` | Create new account | ❌ |
| POST | `/auth/login` | Authenticate user | ❌ |
| POST | `/auth/logout` | Logout user | ✅ |
| GET | `/auth/me` | Get current user info | ✅ |
| POST | `/auth/change-password` | Change password | ✅ |
| POST | `/auth/refresh` | Refresh access token | ❌ |
| POST | `/auth/rotate` | Rotate refresh token | ✅ |

### Dashboard Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|-----------------|
| GET | `/dashboard` | User's personal dashboard | ✅ |
| GET | `/dashboard/admin` | System-wide stats (Admin) | ✅ (Admin) |
| GET | `/dashboard/stats` | System-wide stats (Admin) | ✅ (Admin) |
| GET | `/dashboard/forecast` | License expiration forecast | ✅ |
| GET | `/dashboard/activity-timeline` | User activity history | ✅ |
| GET | `/dashboard/analytics` | Usage analytics | ✅ |
| GET | `/dashboard/enhanced` | Enhanced dashboard stats | ✅ |

### License Endpoints

| Method | Endpoint | Description | Auth Required |
|--------|----------|-------------|-----------------|
| POST | `/licenses` | Create license | ✅ |
| GET | `/licenses` | List all licenses | ✅ |
| GET | `/licenses/{id}` | Get license details | ✅ |
| POST | `/licenses/validate` | Validate license key | ✅ |
| POST | `/licenses/activate` | Activate license | ✅ |
| POST | `/licenses/deactivate` | Deactivate license | ✅ |
| DELETE | `/licenses/{id}` | Delete license | ✅ |

---

## Common Use Cases

### Use Case 1: New User Registration and First Login

```bash
# 1. Register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!",
    "fullName": "John Doe"
  }'

# 2. Login
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "user@example.com",
    "password": "SecurePassword123!"
  }'

# 3. Get dashboard
curl -X GET http://localhost:8080/api/v1/dashboard \
  -H "Authorization: Bearer <access_token>"
```

### Use Case 2: Create and Activate a License

```bash
# 1. Create license
curl -X POST http://localhost:8080/api/v1/licenses \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "licenseKey": "ABC-123-XYZ-789",
    "name": "Professional License",
    "expiresAt": "2027-03-05T00:00:00Z",
    "maxActivations": 3
  }'

# 2. Validate license
curl -X POST http://localhost:8080/api/v1/licenses/validate \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "licenseKey": "ABC-123-XYZ-789"
  }'

# 3. Activate on a machine
curl -X POST http://localhost:8080/api/v1/licenses/activate \
  -H "Authorization: Bearer <access_token>" \
  -H "Content-Type: application/json" \
  -d '{
    "licenseKey": "ABC-123-XYZ-789",
    "machineId": "DEV-PC-001",
    "machineName": "Developer Workstation"
  }'
```

### Use Case 3: Monitor License Health

```bash
# 1. Check personal dashboard for overview
curl -X GET http://localhost:8080/api/v1/dashboard \
  -H "Authorization: Bearer <access_token>"

# 2. View expiration forecast
curl -X GET http://localhost:8080/api/v1/dashboard/forecast \
  -H "Authorization: Bearer <access_token>"

# 3. Check recent activity
curl -X GET http://localhost:8080/api/v1/dashboard/activity-timeline \
  -H "Authorization: Bearer <access_token>"
```

### Use Case 4: Admin System Monitoring

```bash
# Get system-wide statistics (requires Admin role)
curl -X GET http://localhost:8080/api/v1/dashboard/admin \
  -H "Authorization: Bearer <admin_access_token>"
```

### Use Case 5: Rotate Tokens for Enhanced Security

```bash
# Refresh access token (reads refreshToken from cookies automatically)
curl -X POST http://localhost:8080/api/v1/auth/refresh \
  -H "Cookie: refreshToken=<your_refresh_token>" \
  -c cookies.txt

# Rotate tokens (generates new pair, revokes old refresh token)
curl -X POST http://localhost:8080/api/v1/auth/rotate \
  -H "Cookie: refreshToken=<your_refresh_token>" \
  -c cookies.txt

# Both endpoints automatically set new tokens in cookies
# No request body needed - tokens come from cookies
```

---

## Error Handling

All errors follow this format:

```json
{
  "error": "Error code",
  "message": "Human-readable error description",
  "status": 400
}
```

### Common Error Codes

| Status | Error | Meaning |
|--------|-------|---------|
| 400 | InvalidInput | Invalid request parameters |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | NotFound | Resource not found |
| 429 | RateLimited | Too many requests |
| 500 | InternalError | Server error |

---

## Rate Limiting

The API implements rate limiting per endpoint:

- **Login/Register:** 5 attempts per 15 minutes
- **General endpoints:** 100 requests per 15 minutes
- **License activation:** 10 attempts per 15 minutes

Rate limit headers are included in responses:

```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 2026-03-05T15:00:00Z
```

---

## Best Practices

1. **Token Management**
   - Store tokens securely (use httpOnly cookies in web applications)
   - Refresh tokens before expiration
   - Revoke tokens on logout

2. **License Validation**
   - Always validate licenses before use
   - Check expiration dates regularly
   - Monitor activation count vs. maximum activations

3. **Error Handling**
   - Implement proper error handling for all API calls
   - Log errors for debugging
   - Provide user-friendly error messages

4. **Performance**
   - Use pagination for list endpoints (default limit: 10, max: 100)
   - Cache frequently accessed data
   - Monitor API response times

5. **Security**
   - Never expose tokens in logs or URLs
   - Use HTTPS in production
   - Implement proper input validation
   - Keep refresh tokens separate from access tokens

---

## Support

For issues or questions:
- Check the API documentation
- Review the error messages
- Contact the support team

---

## Changelog

**Version 1.0.0** - March 5, 2026
- Initial release
- Core authentication
- License management
- Dashboard statistics
- Audit logging
