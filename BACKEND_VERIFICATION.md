# Backend Data Verification Guide

## Current Frontend Issues

Your frontend is experiencing these issues due to incorrect backend data:

1. ❌ **Admin shows "Welcome user"** - Backend returns wrong user in `/auth/me`
2. ❌ **Settings page shows username=user, email=user@example.com** - Test/placeholder data returned
3. ❌ **Admin dashboard shows generic user role badge** - Role data not properly differentiated
4. ❌ **Sessions page redirects to login** - API error when fetching sessions

**Root Cause**: Your backend's `/auth/me` endpoint and session endpoints are returning incorrect or placeholder data instead of the actual logged-in user's information.

---

## How to Test Your Backend

### 1. Test Login Endpoint

**Endpoint**:`POST /auth/login`

**Request**:
```bash
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin_user_name",
    "password": "admin_password"
  }' \
  -c cookies.txt
```

**Expected Response**:
```json
{
  "success": true,
  "data": {
    "id": "unique-admin-id",
    "username": "admin_user_name",
    "email": "admin@example.com",
    "role": "Admin",
    "isActive": true
  },
  "token": "jwt-token-here"
}
```

---

### 2. Test Current User Endpoint (CRITICAL!)

**This is where the issue likely is!**

**Endpoint**: `GET /auth/me`

**How to test**:
```bash
# After login (cookies will be sent automatically)
curl -X GET http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer your-jwt-token" \
  -b cookies.txt
```

**Expected Response for Admin User**:
```json
{
  "success": true,
  "data": {
    "id": "unique-admin-id",
    "username": "actual_admin_username",
    "email": "admin@real-email.com",
    "role": "Admin",
    "isActive": true,
    "createdAt": "2025-01-01T10:00:00Z",
    "updatedAt": "2026-03-05T10:00:00Z",
    "notifyLicenseExpiry": true,
    "notifyAccountActivity": true,
    "notifySystemAnnouncements": false
  }
}
```

**Expected Response for Regular User**:
```json
{
  "success": true,
  "data": {
    "id": "unique-user-id",
    "username": "actual_user_username",
    "email": "user@real-email.com",
    "role": "User",
    "isActive": true,
    "createdAt": "2025-02-01T10:00:00Z",
    "updatedAt": "2026-03-05T10:00:00Z",
    "notifyLicenseExpiry": true,
    "notifyAccountActivity": true,
    "notifySystemAnnouncements": false
  }
}
```

**❌ PROBLEM - Current Response**:
```json
{
  "success": true,
  "data": {
    "id": "hardcoded-id",
    "username": "user",
    "email": "user@example.com",
    "role": "User",
    ...
  }
}
```

---

### 3. Test Active Sessions Endpoint

**Endpoint**: `GET /auth/sessions`

**Request**:
```bash
curl -X GET http://localhost:8080/api/auth/sessions \
  -H "Authorization: Bearer your-jwt-token" \
  -b cookies.txt
```

**Expected Response**:
```json
{
  "success": true,
  "data": [
    {
      "id": "session-id-1",
      "userId": "user-id",
      "userAgent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
      "ipAddress": "192.168.1.100",
      "isCurrent": true,
      "createdAt": "2026-03-05T10:00:00Z",
      "lastActivity": "2026-03-05T11:30:00Z",
      "expiresAt": "2026-04-04T10:00:00Z"
    },
    {
      "id": "session-id-2",
      "userId": "user-id",
      "userAgent": "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
      "ipAddress": "192.168.1.101",
      "isCurrent": false,
      "createdAt": "2026-02-28T14:00:00Z",
      "lastActivity": "2026-03-01T10:00:00Z",
      "expiresAt": "2026-03-28T14:00:00Z"
    }
  ]
}
```

---

## Debugging Checklist

### Is `/auth/me` returning test data?
- [ ] Username is "user" (should be actual username)
- [ ] Email is "user@example.com" (should be actual email)
- [ ] Role is always "User" (should vary by actual user)
- [ ] ID is hardcoded (should be unique per user)

**If YES to any of these**, your backend is returning placeholder data.

### Check Your Backend Code

Look for these patterns in your backend:

**❌ Problem Pattern (Go/Chi)**:
```go
// BAD - Returns hardcoded test user
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
    user := User{
        ID:       "123",
        Username: "user",
        Email:    "user@example.com",
        Role:     "User",
    }
    response.JSON(w, user)
}
```

**✅ Correct Pattern**:
```go
// GOOD - Get actual user from request context/session
func GetCurrentUser(w http.ResponseWriter, r *http.Request) {
    // Get user ID from JWT token or session
    userID := ExtractUserIDFromToken(r)
    
    // Fetch actual user from database
    user, err := db.GetUserByID(userID)
    if err != nil {
        response.Error(w, "User not found", 404)
        return
    }
    
    response.JSON(w, user)
}
```

---

## What Frontend Expects

The frontend calls these endpoints and expects:

### 1. User Object Structure
```typescript
interface User {
  id: string;
  username: string;
  email: string;
  role: 'Admin' | 'Manager' | 'User';  // Check your backend for actual role names
  isActive: boolean;
  createdAt: string;
  updatedAt: string;
  notifyLicenseExpiry?: boolean;
  notifyAccountActivity?: boolean;
  notifySystemAnnouncements?: boolean;
}
```

### 2. Session Object Structure
```typescript
interface Session {
  id: string;
  userId: string;
  userAgent: string;
  ipAddress: string;
  isCurrent: boolean;
  createdAt: string;
  lastActivity: string;
  expiresAt: string;
}
```

---

## Quick Fix Steps

### Step 1: Verify JWT Claims
Make sure your backend is:
1. ✅ Storing JWT token properly after login
2. ✅ Extracting user ID from JWT in `/auth/me` endpoint
3. ✅ Fetching ACTUAL user from database using that ID
4. ✅ NOT returning hardcoded/test user data

### Step 2: Check Session Storage
1. ✅ Storing sessions in database after login
2. ✅ Fetching sessions for current user by user ID
3. ✅ Marking current session with `isCurrent: true`
4. ✅ Properly handling session expiration

### Step 3: Test with Postman/cURL
Test each endpoint with real admin and user logins to verify:
- `POST /auth/login` - Returns different data for admin vs user
- `GET /auth/me` - Returns ACTUAL logged-in user, not placeholder
- `GET /auth/sessions` - Returns real sessions for that user

### Step 4: If Still Broken
Check these common issues:
- [ ] JWT token extraction not working
- [ ] Database query not using extracted user ID
- [ ] Session endpoint queries wrong table
- [ ] CORS/Authentication headers not passed correctly
- [ ] Middleware not setting user context properly

---

## Example Test Script (Backend)

Create a test script in your backend to verify:

```go
// test-auth.go
func TestCurrentUserEndpoint(t *testing.T) {
    // 1. Login as admin
    loginResp := LoginAsAdmin(t)
    adminToken := loginResp.Token
    
    // 2. Call /auth/me with admin token
    meResp := CallAuthMe(t, adminToken)
    
    // 3. Verify response
    assert.Equal(t, "admin_username", meResp.Username)
    assert.Equal(t, "Admin", meResp.Role)
    assert.NotEqual(t, "user", meResp.Username) // Should NOT be hardcoded
    
    // 4. Login as user
    loginResp2 := LoginAsUser(t)
    userToken := loginResp2.Token
    
    // 5. Call /auth/me with user token
    meResp2 := CallAuthMe(t, userToken)
    
    // 6. Verify different response
    assert.Equal(t, "user_username", meResp2.Username)
    assert.Equal(t, "User", meResp2.Role)
    assert.NotEqual(t, meResp.ID, meResp2.ID) // Different users
}
```

---

## What To Check in Backend Logs

When testing, look for:

```
✅ Good logs:
- "[AUTH] User john_admin logged in successfully"
- "[AUTH] GET /auth/me called for user: john_admin (ID: abc123)"
- "[SESSION] Created session for user abc123 from IP 192.168.1.100"

❌ Bad logs:
- "[AUTH] GET /auth/me returning hardcoded test user"
- "[AUTH] User ID not extracted from token, using default: 123"
- "[SESSION] No sessions found - returning empty array"
```

---

## Frontend Error Messages to Watch For

If you see these in the browser console, backend is broken:

```
❌ "Failed to fetch user data"
❌ "Unexpected user format from /auth/me"
❌ "Missing required field: role"
❌ "Session fetch failed - Invalid response"
```

---

## Next Steps

1. **Run the test requests** above using cURL or Postman
2. **Check what actual response** you get from `/auth/me`
3. **Compare with expected response** - identify differences
4. **Fix backend** to return actual user data instead of placeholders
5. **Test again** - should see correct username/role in frontend

Once your backend returns the correct data, all frontend issues will be resolved automatically!

---

## Still Having Issues?

1. Check backend `/auth/me` response - **This is 90% of the problem**
2. Verify JWT extraction - Is the user ID being extracted correctly?
3. Check database - Are users being stored with correct roles?
4. Test with multiple users - Admin vs Regular user should have different responses
5. Check middleware - Is authentication working properly?

**Post the actual `/auth/ me` response you're getting**, and we can identify the exact issue.
