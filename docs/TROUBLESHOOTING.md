# Troubleshooting Guide

**Last Updated**: March 4, 2026  
**Version**: 1.0

Common issues, solutions, and debugging techniques for the License Management API.

---

## Table of Contents

1. [Installation & Setup](#installation--setup)
2. [Database Issues](#database-issues)
3. [Redis Issues](#redis-issues)
4. [Authentication & JWT](#authentication--jwt)
5. [API & Handler Issues](#api--handler-issues)
6. [Email & Notifications](#email--notifications)
7. [Performance & Scaling](#performance--scaling)
8. [Docker & Containers](#docker--containers)
9. [Kubernetes Deployment](#kubernetes-deployment)
10. [Testing Issues](#testing-issues)
11. [Debugging Techniques](#debugging-techniques)

---

## Installation & Setup

### Issue: Go modules not found

**Symptoms:**
```
Error: package not found
Error: import cycle not allowed
```

**Solutions:**
```bash
# Download all dependencies
go mod download

# Tidy up unused imports
go mod tidy

# Verify checksums
go mod verify

# Clear cache if persistent
go clean -modcache
rm go.sum
go mod download
```

### Issue: Binary won't build

**Symptoms:**
```
Error: undefined: [function name]
Build failed: unexpected end of file
```

**Solutions:**
```bash
# Check for syntax errors
go build -v ./cmd/api

# Run linter
golangci-lint run

# Fix formatting
go fmt ./...

# Vet for suspicious code
go vet ./...
```

### Issue: Port already in use

**Symptoms:**
```
Error: listen tcp :8080: bind: address already in use
```

**Solutions (Windows):**
```powershell
# Find process using port 8080
netstat -ano | findstr :8080

# Kill process (e.g., PID 1234)
taskkill /PID 1234 /F

# Change port in .env
PORT=8081
```

**Solutions (Linux/macOS):**
```bash
# Find process
lsof -i :8080

# Kill process
kill -9 <PID>

# Or use fuser
fuser -k 8080/tcp
```

---

## Database Issues

### Issue: Database connection refused

**Symptoms:**
```
Error: failed to connect to postgres://localhost:5432
Error: connection refused
```

**Solutions:**

1. **Check PostgreSQL is running:**
```bash
# Docker
docker-compose ps postgres

# macOS
brew services list | grep postgres

# Linux
sudo systemctl status postgresql
```

2. **Verify connection string:**
```bash
# Test manual connection
psql $DATABASE_URL

# Check .env file
cat .env | grep DATABASE
```

3. **Restart database:**
```bash
# Docker
docker-compose restart postgres
docker-compose logs postgres

# Local
sudo systemctl restart postgresql
```

### Issue: Database migrations fail

**Symptoms:**
```
Error: migration: file does not exist
Error: table already exists
```

**Solutions:**

1. **Manual migration check:**
```sql
-- Connect to database
psql $DATABASE_URL

-- Check tables
\dt

-- List migrations
SELECT * FROM schema_migrations;
```

2. **Reset database (development only):**
```bash
# Drop and recreate
docker-compose down -v
docker-compose up -d postgres

# Or PostgreSQL CLI
dropdb license_mgmt
createdb license_mgmt
```

3. **Skip auto-migration temporarily:**
```go
// In database.New() temporarily comment out:
// db.AutoMigrate(...)
```

### Issue: Slow queries

**Symptoms:**
```
Database queries taking > 1 second
N+1 query patterns in logs
```

**Solutions:**

1. **Enable slow query logging:**
```bash
DATABASE_SLOW_QUERY_THRESHOLD=1s
```

2. **Check indexes:**
```sql
-- List indexes
SELECT * FROM pg_indexes WHERE tablename = 'users';

-- Verify index usage
EXPLAIN ANALYZE SELECT * FROM licenses WHERE status = 'Active';
```

3. **Optimize queries:**
```go
// Use JOIN instead of N+1
db.Preload("User").Where("status = ?", "Active").Find(&licenses)

// Use Select to limit columns
db.Select("id", "license_key", "status").Find(&licenses)
```

### Issue: Race conditions in tests

**Symptoms:**
```
Flaky tests, intermittent failures
Tests pass individually but fail together
```

**Solutions:**

```bash
# Run tests with race detector
go test -race ./...

# Serial execution (fixes race conditions)
go test -p 1 ./...

# Run specific test multiple times
go test -count 100 -run TestName ./package
```

---

## Redis Issues

### Issue: Redis connection failed

**Symptoms:**
```
Error: dial tcp localhost:6379: connection refused
Error: WRONGPASS invalid username-password pair
```

**Solutions:**

1. **Check Redis is running:**
```bash
# Docker
docker-compose ps redis

# Manual test
redis-cli -h localhost -p 6379 ping
# Expected: PONG
```

2. **Verify configuration:**
```bash
# Check connection string
echo $REDIS_HOST
echo $REDIS_PORT
echo $REDIS_PASSWORD

# Test with credentials
redis-cli -h localhost -p 6379 -a $REDIS_PASSWORD ping
```

3. **Restart Redis:**
```bash
# Docker
docker-compose restart redis

# Check logs
docker-compose logs redis
```

### Issue: Redis memory full

**Symptoms:**
```
MISCONF Redis is configured to save RDB snapshots
OOM command not allowed when used memory > maxmemory
```

**Solutions:**

```bash
# Check memory usage
redis-cli info memory

# Clear old keys
redis-cli FLUSHDB

# Increase maxmemory
redis-cli CONFIG SET maxmemory 512mb

# Check rate limit keys
redis-cli KEYS "ratelimit:*" | wc -l
redis-cli DEL $(redis-cli KEYS "ratelimit:*")
```

### Issue: Sessions/cache not persisting

**Symptoms:**
```
User sessions expire immediately
Cache not working
```

**Solutions:**

1. **Verify schema:**
```bash
# Check Redis key prefix
redis-cli KEYS "session:*"
redis-cli KEYS "token:*"
redis-cli KEYS "cache:*"
```

2. **Check TTL:**
```bash
# Get TTL for key
redis-cli TTL "session:user123"

# Should be > 0 (seconds remaining)
# -1 = no expiry set
# -2 = key doesn't exist
```

3. **Reset sessions:**
```bash
# Clear all sessions/tokens
redis-cli DEL $(redis-cli KEYS "session:*")
redis-cli DEL $(redis-cli KEYS "token:*")
```

---

## Authentication & JWT

### Issue: JWT token errors

**Symptoms:**
```
Error: JWT_SECRET must be at least 32 characters
Error: token is invalid or expired
```

**Solutions:**

1. **Verify secret is set:**
```bash
# Check environment
echo $JWT_SECRET | wc -c
# Should be > 32

# Generate new secret
openssl rand -base64 32

# Set in .env
JWT_SECRET=<generated-secret>
```

2. **Check token expiry:**
```bash
# Adjust timeouts in .env
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Test token generation
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"admin","password":"password"}'
```

3. **Decode JWT to inspect:**
```bash
# Use jwt.io or decode CLI
jwt decode <token> --secret="$JWT_SECRET"

# Or check claims programmatically
```

### Issue: Authentication fails for valid credentials

**Symptoms:**
```
401 Unauthorized for correct username/password
Invalid credentials error
```

**Solutions:**

1. **Verify user exists:**
```sql
-- Check database
psql $DATABASE_URL -c "SELECT id, username, email, status FROM users WHERE username='admin';"

# User should exist and have status='Active'
```

2. **Check password:**
```bash
# Test password hashing locally
# Try resetting password
curl -X POST http://localhost:8080/api/auth/password-reset-request \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com"}'
```

3. **Review auth middleware:**
```bash
# Enable debug logging
LOG_LEVEL=debug
ENVIRONMENT=development

# Check logs for auth flow
docker-compose logs api | grep -i auth
```

### Issue: CORS errors on login

**Symptoms:**
```
Access to XMLHttpRequest blocked by CORS policy
```

**Solutions:**

1. **Verify CORS configuration:**
```bash
# Check .env
echo $CORS_ALLOWED_ORIGINS

# Should include frontend URL
CORS_ALLOWED_ORIGINS=https://app.example.com
```

2. **Test preflight:**
```bash
# OPTIONS request should succeed
curl -X OPTIONS http://api:8080/api/auth/login \
  -H "Origin: https://app.example.com" \
  -v
```

---

## API & Handler Issues

### Issue: 404 Not Found on valid endpoint

**Symptoms:**
```
404 error on endpoints that should exist
Route not found
```

**Solutions:**

1. **Verify routes are registered:**
```bash
# Check internal/server/routes.go
# Ensure handler is attached to router

# Restart API
docker-compose restart api
```

2. **Test endpoint:**
```bash
# Check exact path
curl http://localhost:8080/api/auth/me \
  -H "Authorization: Bearer $TOKEN"

# Check method (GET vs POST)
curl -X GET http://localhost:8080/api/licenses
```

3. **Check prefix paths:**
```go
// Routes should use full path like:
router.HandleFunc("/api/auth/login", authHandler.Login).Methods("POST")
// Not just: router.HandleFunc("/login", ...)
```

### Issue: 500 Internal Server Error

**Symptoms:**
```
500 Internal Server Error
Handler panic
```

**Solutions:**

1. **Check logs:**
```bash
# View real-time logs
docker-compose logs -f api

# Look for stack trace and error message
LOG_LEVEL=debug
```

2. **Test endpoint directly:**
```bash
# Use curl with verbose
curl -v -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"username":"testuser","email":"test@example.com","password":"password123"}'
```

3. **Check dependencies:**
- Database connection working?
- Redis connection working?
- Required middlewares in place?

### Issue: Request timeout

**Symptoms:**
```
Request takes > 30 seconds
Timeout error
```

**Solutions:**

1. **Check database performance:**
```sql
-- Enable slow query log
SET log_min_duration_statement = 1000; -- 1 second

-- Check actual query time
EXPLAIN ANALYZE SELECT * FROM licenses WHERE user_id = 123;
```

2. **Check response compression:**
```bash
# Ensure gzip is enabled
# Response should have Content-Encoding: gzip
curl -I http://localhost:8080/api/licenses \
  -H "Accept-Encoding: gzip"
```

3. **Add request timeout logging:**
```go
// In middleware, log request duration
start := time.Now()
// ... handle request
duration := time.Since(start)
if duration > 5*time.Second {
    log.Warnf("Slow request: %s took %v", r.URL.Path, duration)
}
```

---

## Email & Notifications

### Issue: Emails not sending

**Symptoms:**
```
Email service errors
Verification emails not received
```

**Solutions:**

1. **Check SMTP configuration:**
```bash
# Verify settings are loaded
LOG_LEVEL=debug
go run cmd/api/main.go

# Should show: "Email config loaded: SmtpHost=..., SmtpPort=587"
```

2. **Test email manually:**
```bash
# Using curl to test SMTP
telnet $SMTP_HOST $SMTP_PORT
# Should connect

# Or use Go email package test
go test ./internal/service -run TestEmailService_SendEmail -v
```

3. **Use console mode for development:**
```bash
SMTP_USE_CONSOLE=true
# Emails will print to stdout instead of sending
```

4. **Check email queue:**
```bash
# Verify queue service is running
redis-cli LLEN "notification_queue"

# Check queued emails
redis-cli LRANGE "notification_queue" 0 -1
```

### Issue: Email verification tokens not working

**Symptoms:**
```
Verification link returns error
Token expired immediately
```

**Solutions:**

1. **Check token expiry:**
```bash
# Set longer expiry for testing
EMAIL_VERIFICATION_EXPIRY=24h
PASSWORD_RESET_EXPIRY=1h
```

2. **Verify token is sent:**
```bash
# Check logs for token generation
LOG_LEVEL=debug SMTP_USE_CONSOLE=true go run cmd/api/main.go

# Manually test token endpoint
curl -X POST http://localhost:8080/api/auth/verify-email \
  -H "Content-Type: application/json" \
  -d '{"token":"<token-from-email>"}'
```

---

## Performance & Scaling

### Issue: High memory usage

**Symptoms:**
```
Memory constantly growing
OOM killer triggers
```

**Solutions:**

1. **Profile memory:**
```bash
# Use pprof
curl http://localhost:6060/debug/pprof/heap > heap.prof
go tool pprof heap.prof
```

2. **Check connection pools:**
```bash
# Verify settings
DATABASE_MAX_OPEN_CONNS=25
DATABASE_MAX_IDLE_CONNS=5
REDIS_POOL_SIZE=10

# Reduce if memory-constrained
DATABASE_MAX_OPEN_CONNS=10
```

3. **Monitor goroutines:**
```bash
# Check goroutine count
curl http://localhost:6060/debug/pprof/goroutine?debug=1
```

### Issue: Slow response times

**Symptoms:**
```
P95 latency > 1 second
API sluggish
```

**Solutions:**

1. **Enable caching:**
```bash
# Verify cache headers are sent
curl -I http://localhost:8080/api/licenses
# Should see: Cache-Control: max-age=300

# Check Redis cache hit rate
redis-cli INFO stats | grep keyspace_hits
```

2. **Add database indexes:**
```sql
-- Check existing indexes
SELECT * FROM pg_stat_user_indexes;

-- Create missing indexes for common queries
CREATE INDEX idx_licenses_user_status ON licenses(user_id, status);
CREATE INDEX idx_audit_created_user ON audit_logs(created_at, user_id);
```

3. **Batch operations:**
```bash
# Instead of N requests, use pagination
curl http://localhost:8080/api/licenses?page=1&pageSize=50
```

---

## Docker & Containers

### Issue: Container won't start

**Symptoms:**
```
Container exits immediately
Exit code 1
```

**Solutions:**

```bash
# Check logs
docker-compose logs api

# rebuild image
docker-compose build --no-cache api

# Check disk space
docker system df

# Verify environment file
cat .env | grep -E "^[A-Z]" | wc -l
# Should be > 10
```

### Issue: Database migrations not running

**Symptoms:**
```
"relation does not exist" errors
Tables not created
```

**Solutions:**

```bash
# Check migration logs
docker-compose logs postgres

# Manually run migrations
docker-compose exec api go run cmd/api/main.go

# Verify tables
docker-compose exec postgres psql -U postgres -d license_mgmt -c "\dt"
```

### Issue: Docker network issues

**Symptoms:**
```
api can't reach postgres
Connection refused between services
```

**Solutions:**

```bash
# Check network
docker network ls
docker network inspect license-management-api_default

# Use service names (not localhost)
DATABASE_URL=postgres://user:pass@postgres:5432/license_mgmt
REDIS_HOST=redis
REDIS_PORT=6379
```

---

## Kubernetes Deployment

### Issue: Pod pending/not starting

**Symptoms:**
```
kubectl get pods shows Pending
ImagePullBackOff
```

**Solutions:**

```bash
# Check events
kubectl describe pod <pod-name>

# Check logs
kubectl logs <pod-name>

# Check image exists
kubectl get images

# Delete and redeploy
kubectl delete pod <pod-name>
kubectl apply -f deployment.yaml
```

### Issue: Service unreachable

**Symptoms:**
```
Can't reach API from outside cluster
Port forwarding fails
```

**Solutions:**

```bash
# Check service
kubectl get svc license_mgmt-api
kubectl describe svc license_mgmt-api

# Port forward for testing
kubectl port-forward svc/license_mgmt-api 8080:8080

# Check endpoints
kubectl get endpoints license_mgmt-api

# Check network policy if blocking
kubectl describe networkpolicy
```

### Issue: Persistent volume issues

**Symptoms:**
```
PVC pending
Mount fails
```

**Solutions:**

```bash
# Check PVC status
kubectl get pvc

# Check storage classes
kubectl get sc

# If using local storage, ensure directory exists
kubectl describe pvc <pvc-name>
```

---

## Testing Issues

### Issue: Tests hang/timeout

**Symptoms:**
```
Tests never complete
go test times out after 10m
```

**Solutions:**

```bash
# Run with timeout override
go test -timeout 30m ./...

# Run single test
go test -timeout 5m -run TestName ./package

# Check for deadlocks
go test -race ./...
```

### Issue: Coverage not improving

**Symptoms:**
```
New tests don't increase coverage %
Coverage stuck at previous level
```

**Solutions:**

```bash
# Generate coverage report
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Find uncovered lines
go tool cover -html=coverage.out -o coverage.html

# Add tests for uncovered functions
# Focus on high-impact areas first (services, handlers)
```

### Issue: Mock tests pass but integration fails

**Symptoms:**
```
Unit tests pass, API broken in production
Mock expectations != real behavior
```

**Solutions:**

1. **Convert mocks to integration tests:**
```bash
# Run integration tests with real DB/Redis
go test -tags=integration ./...
```

2. **Test with real dependencies:**
```go
// Use testcontainers for real PostgreSQL
postgres, _ := setupTestDB()
defer postgres.Terminate()
```

3. **Load test against staging:**
```bash
# Use k6 or similar
k6 run loadtest.js --vus 50 --duration 30s https://staging-api.example.com
```

---

## Debugging Techniques

### Enable Verbose Logging

```bash
# Maximum debug logging
LOG_LEVEL=debug
ENVIRONMENT=development
LOG_FORMAT=text

go run cmd/api/main.go
```

### Use Debugger (Delve)

```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Start debugger
dlv debug cmd/api/main.go

# Set breakpoint
(dlv) break main.main

# Continue execution
(dlv) continue
```

### Request Inspection

```bash
# Log all requests with curl
curl -v -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"test","password":"pass"}' \
  2>&1 | tee request.log
```

### Database Query Inspection

```bash
# Connect to database directly
psql $DATABASE_URL

# Enable query logging
SET log_statement = 'all';
SET log_min_duration_statement = 0;

# Check active queries
SELECT pid, usename, application_name, query FROM pg_stat_activity;
```

### Redis Inspection

```bash
# Monitor all commands
redis-cli MONITOR

# Check memory usage
redis-cli INFO memory

# List all keys
redis-cli KEYS "*"

# Inspect specific key
redis-cli GET "session:user123"
redis-cli HGETALL "token:abc123"
```

---

## Getting Help

**If issue not listed above:**

1. **Check logs for actual error:**
```bash
docker-compose logs -f api | grep -i error
LOG_LEVEL=debug go run cmd/api/main.go 2>&1 | grep -A 5 error
```

2. **Verify configuration:**
```bash
# Print all environment variables loaded
go run -ldflags "-X main.debug=1" cmd/api/main.go

# Check .env file is correct
cat .env | sort
```

3. **Search documentation:**
- [SETUP.md](../SETUP.md) - Installation
- [CONFIGURATION.md](../CONFIGURATION.md) - Settings reference
- [SECURITY.md](../SECURITY.md) - Security issues
- [DEPLOYMENT.md](./DEPLOYMENT.md) - Deployment problems

4. **Contact support:**
- 📧 Email: dev-team@license_mgmt.com
- 💬 Slack: #license_mgmt-dev
- 🐛 GitHub Issues: [Issue Tracker](https://github.com/your-org/license-management-api/issues)

---

**Document Maintained By**: license_mgmt Development Team  
**Last Updated**: March 4, 2026  
**Version**: 1.0


