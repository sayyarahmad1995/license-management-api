# Configuration Guide

**Last Updated**: March 4, 2026  
**Version**: 1.0

This document provides comprehensive documentation for all configuration options in the License Management License Server.

---

## Table of Contents

1. [Configuration Overview](#configuration-overview)
2. [Environment Variables](#environment-variables)
3. [Application Settings](#application-settings)
4. [Database Configuration](#database-configuration)
5. [Redis Configuration](#redis-configuration)
6. [JWT & Authentication](#jwt--authentication)
7. [Email Settings](#email-settings)
8. [CORS Configuration](#cors-configuration)
9. [Rate Limiting](#rate-limiting)
10. [Logging & Monitoring](#logging--monitoring)
11. [Configuration Files](#configuration-files)
12. [Docker Configuration](#docker-configuration)
13. [Kubernetes Configuration](#kubernetes-configuration)

---

## Configuration Overview

The License Management License Server uses **Viper** for configuration management with the following priority order:

1. **Environment Variables** (highest priority)
2. **.env file** (development)
3. **Default values** (lowest priority)

### Quick Start

```bash
# Copy example configuration
cp .env.example .env

# Edit configuration
nano .env

# Verify configuration
go run cmd/api/main.go
```

---

## Environment Variables

### Complete Variable Reference

| Variable | Type | Required | Default | Description |
|----------|------|----------|---------|-------------|
| **Application** |
| `PORT` | int | No | `8080` | HTTP server port |
| `ENVIRONMENT` | string | No | `development` | Environment mode (`development`, `production`) |
| `APP_NAME` | string | No | `License Management License Server` | Application name |
| **Database** |
| `DATABASE_URL` | string | Yes | - | PostgreSQL connection string |
| `DATABASE_HOST` | string | No | `localhost` | Database host |
| `DATABASE_PORT` | int | No | `5432` | Database port |
| `DATABASE_USER` | string | No | `postgres` | Database user |
| `DATABASE_PASSWORD` | string | Yes | - | Database password |
| `DATABASE_NAME` | string | No | `License Management` | Database name |
| `DATABASE_SSLMODE` | string | No | `disable` | SSL mode (`disable`, `require`, `verify-full`) |
| `DATABASE_MAX_OPEN_CONNS` | int | No | `25` | Maximum open connections |
| `DATABASE_MAX_IDLE_CONNS` | int | No | `5` | Maximum idle connections |
| `DATABASE_CONN_MAX_LIFETIME` | duration | No | `5m` | Connection max lifetime |
| **Redis** |
| `REDIS_HOST` | string | No | `localhost` | Redis host |
| `REDIS_PORT` | int | No | `6379` | Redis port |
| `REDIS_PASSWORD` | string | No | - | Redis password (if enabled) |
| `REDIS_DB` | int | No | `0` | Redis database number |
| `REDIS_POOL_SIZE` | int | No | `10` | Connection pool size |
| `REDIS_MIN_IDLE_CONNS` | int | No | `5` | Minimum idle connections |
| `REDIS_CONN_MAX_IDLE_TIME` | duration | No | `5m` | Connection max idle time |
| **JWT Authentication** |
| `JWT_SECRET` | string | Yes | - | Access token secret (min 32 chars) |
| `JWT_REFRESH_SECRET` | string | Yes | - | Refresh token secret (min 32 chars) |
| `JWT_ACCESS_EXPIRY` | duration | No | `15m` | Access token lifetime |
| `JWT_REFRESH_EXPIRY` | duration | No | `7d` | Refresh token lifetime |
| `JWT_ISSUER` | string | No | `License Management-server` | JWT issuer claim |
| **Email** |
| `SMTP_HOST` | string | No | - | SMTP server host |
| `SMTP_PORT` | int | No | `587` | SMTP server port |
| `SMTP_USERNAME` | string | No | - | SMTP username |
| `SMTP_PASSWORD` | string | No | - | SMTP password |
| `SMTP_FROM_ADDRESS` | string | No | - | From email address |
| `SMTP_FROM_NAME` | string | No | `License Management` | From display name |
| `SMTP_ENABLE_SSL` | bool | No | `false` | Enable SSL/TLS |
| `SMTP_USE_CONSOLE` | bool | No | `true` | Console mode (dev) |
| `FRONTEND_BASE_URL` | string | No | `http://localhost:3000` | Frontend URL for links |
| **CORS** |
| `CORS_ALLOWED_ORIGINS` | string | No | `*` | Comma-separated origins |
| `CORS_ALLOWED_METHODS` | string | No | `GET,POST,PUT,DELETE,OPTIONS` | Allowed HTTP methods |
| `CORS_ALLOWED_HEADERS` | string | No | `*` | Allowed headers |
| `CORS_ALLOW_CREDENTIALS` | bool | No | `true` | Allow credentials |
| `CORS_MAX_AGE` | int | No | `3600` | Preflight cache duration |
| **Rate Limiting** |
| `RATELIMIT_AUTH_ENABLED` | bool | No | `true` | Enable auth rate limiting |
| `RATELIMIT_AUTH_REQUESTS` | int | No | `5` | Auth requests per window |
| `RATELIMIT_AUTH_WINDOW` | int | No | `60` | Auth window (seconds) |
| `RATELIMIT_API_ENABLED` | bool | No | `true` | Enable API rate limiting |
| `RATELIMIT_API_REQUESTS` | int | No | `100` | API requests per window |
| `RATELIMIT_API_WINDOW` | int | No | `60` | API window (seconds) |
| **Security** |
| `BCRYPT_COST` | int | No | `10` | Bcrypt hashing cost |
| `SESSION_TIMEOUT` | duration | No | `24h` | Session timeout |
| `MAX_LOGIN_ATTEMPTS` | int | No | `5` | Max failed login attempts |
| `LOCKOUT_DURATION` | duration | No | `15m` | Account lockout duration |
| **Logging** |
| `LOG_LEVEL` | string | No | `info` | Log level (`debug`, `info`, `warn`, `error`) |
| `LOG_FORMAT` | string | No | `text` | Log format (`text`, `json`) |
| `LOG_OUTPUT` | string | No | `stdout` | Log output (`stdout`, `file`) |
| `LOG_FILE_PATH` | string | No | `/var/log/License Management.log` | Log file path |
| **Features** |
| `EMAIL_VERIFICATION_ENABLED` | bool | No | `true` | Require email verification |
| `EMAIL_VERIFICATION_EXPIRY` | duration | No | `24h` | Verification token expiry |
| `PASSWORD_RESET_EXPIRY` | duration | No | `1h` | Password reset token expiry |
| `LICENSE_KEY_LENGTH` | int | No | `16` | Generated license key length |
| `AUDIT_LOG_ENABLED` | bool | No | `true` | Enable audit logging |
| `AUDIT_LOG_RETENTION_DAYS` | int | No | `90` | Audit log retention |

---

## Application Settings

### Basic Configuration

```bash
# Application
PORT=8080
ENVIRONMENT=production
APP_NAME=License Management License Server
```

### Environment Modes

**Development:**
```bash
ENVIRONMENT=development
LOG_LEVEL=debug
SMTP_USE_CONSOLE=true
CORS_ALLOWED_ORIGINS=*
```

**Production:**
```bash
ENVIRONMENT=production
LOG_LEVEL=info
SMTP_USE_CONSOLE=false
CORS_ALLOWED_ORIGINS=https://app.License Management.com
```

---

## Database Configuration

### PostgreSQL Connection

**Method 1: Connection String (Recommended)**
```bash
DATABASE_URL=postgres://username:password@localhost:5432/License Management?sslmode=disable
```

**Method 2: Individual Parameters**
```bash
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=License Management_user
DATABASE_PASSWORD=secure_password
DATABASE_NAME=License Management
DATABASE_SSLMODE=require
```

### Connection Pooling

```bash
# Connection pool settings
DATABASE_MAX_OPEN_CONNS=25        # Maximum open connections
DATABASE_MAX_IDLE_CONNS=5         # Maximum idle connections
DATABASE_CONN_MAX_LIFETIME=5m     # Connection lifetime
```

**Tuning Guidelines:**

| Scenario | Max Open | Max Idle | Lifetime |
|----------|----------|----------|----------|
| Low Traffic | 10 | 2 | 10m |
| Medium Traffic | 25 | 5 | 5m |
| High Traffic | 50 | 10 | 3m |
| Very High Traffic | 100 | 20 | 1m |

### SSL/TLS Configuration

**Local Development:**
```bash
DATABASE_SSLMODE=disable
```

**Production:**
```bash
DATABASE_SSLMODE=require
# Or for full verification:
DATABASE_SSLMODE=verify-full
DATABASE_SSL_CERT=/path/to/client-cert.pem
DATABASE_SSL_KEY=/path/to/client-key.pem
DATABASE_SSL_ROOT_CERT=/path/to/ca-cert.pem
```

### Database Migrations

Migrations run automatically on startup. To disable:

```go
// In database.New()
// Comment out: db.AutoMigrate(...)
```

---

## Redis Configuration

### Basic Setup

```bash
# Redis connection
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=                   # Leave empty if no auth
REDIS_DB=0                        # Database number (0-15)
```

### Connection Pooling

```bash
# Pool configuration
REDIS_POOL_SIZE=10                # Connection pool size
REDIS_MIN_IDLE_CONNS=5            # Minimum idle connections
REDIS_CONN_MAX_IDLE_TIME=5m       # Max idle time
```

### Redis Sentinel (High Availability)

```bash
REDIS_SENTINEL_ENABLED=true
REDIS_SENTINEL_MASTER=mymaster
REDIS_SENTINEL_ADDRS=sentinel1:26379,sentinel2:26379,sentinel3:26379
REDIS_SENTINEL_PASSWORD=sentinel_password
```

### Redis Cluster

```bash
REDIS_CLUSTER_ENABLED=true
REDIS_CLUSTER_ADDRS=node1:6379,node2:6379,node3:6379
REDIS_CLUSTER_PASSWORD=cluster_password
```

### Use Cases

Redis is used for:
- **Session Storage**: User sessions with TTL
- **Rate Limiting**: Request counters with expiry
- **Token Revocation**: Blacklisted tokens
- **Cache**: API response caching

---

## JWT & Authentication

### JWT Secrets

**Generate Secure Secrets:**
```bash
# Generate 256-bit secrets
openssl rand -base64 32

# Set in environment
JWT_SECRET=<generated-secret-1>
JWT_REFRESH_SECRET=<generated-secret-2>
```

⚠️ **Security Requirements:**
- Minimum 32 characters
- Must be different for access and refresh tokens
- Never commit to version control
- Rotate every 90 days in production

### Token Expiry

**Duration Format Examples:**
```bash
# Supported formats
JWT_ACCESS_EXPIRY=15m           # 15 minutes (recommended)
JWT_ACCESS_EXPIRY=900s          # 900 seconds
JWT_ACCESS_EXPIRY=0.25h         # 15 minutes

JWT_REFRESH_EXPIRY=7d           # 7 days (recommended)
JWT_REFRESH_EXPIRY=168h         # 168 hours
JWT_REFRESH_EXPIRY=7            # 7 (interpreted as days)
```

**Recommended Settings:**

| Environment | Access Token | Refresh Token |
|-------------|--------------|---------------|
| Development | 1h | 30d |
| Staging | 30m | 7d |
| Production | 15m | 7d |

### JWT Claims

**Standard Claims in Token:**
```json
{
  "sub": "user_id",
  "username": "johndoe",
  "email": "john@example.com",
  "role": "User",
  "iss": "License Management-server",
  "exp": 1709486400,
  "iat": 1709485500
}
```

---

## Email Settings

### SMTP Configuration

**Gmail Example:**
```bash
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_ADDRESS=noreply@License Management.com
SMTP_FROM_NAME=License Management License Server
SMTP_ENABLE_SSL=true
SMTP_USE_CONSOLE=false
```

**SendGrid Example:**
```bash
SMTP_HOST=smtp.sendgrid.net
SMTP_PORT=587
SMTP_USERNAME=apikey
SMTP_PASSWORD=<sendgrid-api-key>
SMTP_FROM_ADDRESS=noreply@License Management.com
SMTP_FROM_NAME=License Management
SMTP_ENABLE_SSL=true
```

**AWS SES Example:**
```bash
SMTP_HOST=email-smtp.us-east-1.amazonaws.com
SMTP_PORT=587
SMTP_USERNAME=<aws-access-key-id>
SMTP_PASSWORD=<aws-secret-access-key>
SMTP_FROM_ADDRESS=noreply@License Management.com
SMTP_ENABLE_SSL=true
```

### Frontend URLs

```bash
# Used for email links
FRONTEND_BASE_URL=https://app.License Management.com

# Email links will be:
# - {FRONTEND_BASE_URL}/verify-email?token={token}
# - {FRONTEND_BASE_URL}/reset-password?token={token}
```

### Development Mode

```bash
# Console mode (emails printed to stdout)
SMTP_USE_CONSOLE=true
```

---

## CORS Configuration

### Basic Setup

```bash
# Allow specific origin
CORS_ALLOWED_ORIGINS=https://app.License Management.com

# Multiple origins (comma-separated)
CORS_ALLOWED_ORIGINS=https://app.License Management.com,https://admin.License Management.com

# Development (allow all)
CORS_ALLOWED_ORIGINS=*
```

### Advanced Configuration

```bash
# HTTP methods
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE,OPTIONS,PATCH

# Headers
CORS_ALLOWED_HEADERS=Authorization,Content-Type,X-Requested-With

# Credentials
CORS_ALLOW_CREDENTIALS=true

# Preflight cache
CORS_MAX_AGE=3600                 # Cache for 1 hour
```

### Production Best Practices

```bash
# Strict CORS (recommended)
CORS_ALLOWED_ORIGINS=https://app.License Management.com
CORS_ALLOWED_METHODS=GET,POST,PUT,DELETE
CORS_ALLOWED_HEADERS=Authorization,Content-Type
CORS_ALLOW_CREDENTIALS=true
CORS_MAX_AGE=86400
```

---

## Rate Limiting

### Configuration

```bash
# Authentication endpoints
RATELIMIT_AUTH_ENABLED=true
RATELIMIT_AUTH_REQUESTS=5         # 5 requests
RATELIMIT_AUTH_WINDOW=60          # per 60 seconds

# General API endpoints
RATELIMIT_API_ENABLED=true
RATELIMIT_API_REQUESTS=100        # 100 requests
RATELIMIT_API_WINDOW=60           # per 60 seconds
```

### Per-Endpoint Limits

**Authentication Endpoints (5/min):**
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/password-reset-request`
- `POST /api/auth/verify-email`

**API Endpoints (100/min):**
- All other authenticated routes

### Tuning Guidelines

| User Type | Auth Limit | API Limit |
|-----------|------------|-----------|
| Public | 3/min | 50/min |
| Registered | 5/min | 100/min |
| Premium | 10/min | 500/min |
| Internal | Unlimited | Unlimited |

### Disable Rate Limiting

```bash
# For testing/development only
RATELIMIT_AUTH_ENABLED=false
RATELIMIT_API_ENABLED=false
```

---

## Logging & Monitoring

### Log Configuration

```bash
# Log level
LOG_LEVEL=info                    # debug, info, warn, error

# Log format
LOG_FORMAT=json                   # text, json

# Log output
LOG_OUTPUT=stdout                 # stdout, file

# File logging
LOG_FILE_PATH=/var/log/License Management/app.log
LOG_MAX_SIZE=100                  # MB
LOG_MAX_BACKUPS=10
LOG_MAX_AGE=30                    # days
```

### Log Levels

**Development:**
```bash
LOG_LEVEL=debug
LOG_FORMAT=text
LOG_OUTPUT=stdout
```

**Production:**
```bash
LOG_LEVEL=info
LOG_FORMAT=json
LOG_OUTPUT=stdout                 # Captured by container runtime
```

### Structured Logging

```bash
# JSON format example
LOG_FORMAT=json

# Output:
{
  "level": "info",
  "timestamp": "2026-03-04T10:30:00Z",
  "message": "User logged in",
  "user_id": 123,
  "ip": "203.0.113.45"
}
```

---

## Configuration Files

### .env File

Create from template:

```bash
cp .env.example .env
```

Example `.env`:

```bash
# Application
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/License Management?sslmode=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-super-secret-key-min-32-characters-long
JWT_REFRESH_SECRET=another-secret-key-also-min-32-chars
JWT_ACCESS_EXPIRY=15m
JWT_REFRESH_EXPIRY=7d

# Email
SMTP_USE_CONSOLE=true
FRONTEND_BASE_URL=http://localhost:3000

# CORS
CORS_ALLOWED_ORIGINS=*

# Rate Limiting
RATELIMIT_AUTH_ENABLED=true
RATELIMIT_AUTH_REQUESTS=5
RATELIMIT_AUTH_WINDOW=60
```

### config.yaml (Alternative)

Create `config/config.yaml`:

```yaml
app:
  name: License Management License Server
  port: 8080
  environment: development

database:
  url: postgres://postgres:password@localhost:5432/License Management
  max_open_conns: 25
  max_idle_conns: 5
  conn_max_lifetime: 5m

redis:
  host: localhost
  port: 6379
  db: 0
  pool_size: 10

jwt:
  access_expiry: 15m
  refresh_expiry: 7d

cors:
  allowed_origins:
    - http://localhost:3000
  allowed_methods:
    - GET
    - POST
    - PUT
    - DELETE
  allow_credentials: true
```

---

## Docker Configuration

### docker-compose.yml

```yaml
version: '3.8'

services:
  api:
    build: .
    ports:
      - "8080:8080"
    environment:
      - PORT=8080
      - ENVIRONMENT=production
      - DATABASE_URL=postgres://postgres:password@postgres:5432/License Management
      - REDIS_HOST=redis
      - REDIS_PORT=6379
      - JWT_SECRET=${JWT_SECRET}
      - JWT_REFRESH_SECRET=${JWT_REFRESH_SECRET}
    env_file:
      - .env.production
    depends_on:
      - postgres
      - redis
    restart: unless-stopped

  postgres:
    image: postgres:15-alpine
    environment:
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
      - POSTGRES_DB=License Management
    volumes:
      - postgres_data:/var/lib/postgresql/data
    restart: unless-stopped

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    restart: unless-stopped

volumes:
  postgres_data:
  redis_data:
```

### Environment Files

**.env.production:**
```bash
JWT_SECRET=production-secret-min-32-chars
JWT_REFRESH_SECRET=production-refresh-secret-32
SMTP_HOST=smtp.sendgrid.net
SMTP_USERNAME=apikey
SMTP_PASSWORD=SG.xxxxxxxxxxxxx
```

---

## Kubernetes Configuration

### ConfigMap

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: License Management-config
data:
  PORT: "8080"
  ENVIRONMENT: "production"
  DATABASE_HOST: "postgres-service"
  DATABASE_PORT: "5432"
  DATABASE_NAME: "License Management"
  DATABASE_SSLMODE: "require"
  REDIS_HOST: "redis-service"
  REDIS_PORT: "6379"
  CORS_ALLOWED_ORIGINS: "https://app.License Management.com"
  LOG_LEVEL: "info"
  LOG_FORMAT: "json"
```

### Secret

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: License Management-secrets
type: Opaque
stringData:
  jwt-secret: "generated-secret-min-32-chars"
  jwt-refresh-secret: "another-secret-min-32-chars"
  database-password: "secure-db-password"
  redis-password: "secure-redis-password"
  smtp-password: "smtp-password"
```

### Deployment

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: License Management-api
spec:
  replicas: 3
  template:
    spec:
      containers:
      - name: api
        image: License Management-server:latest
        envFrom:
        - configMapRef:
            name: License Management-config
        env:
        - name: JWT_SECRET
          valueFrom:
            secretKeyRef:
              name: License Management-secrets
              key: jwt-secret
        - name: JWT_REFRESH_SECRET
          valueFrom:
            secretKeyRef:
              name: License Management-secrets
              key: jwt-refresh-secret
        - name: DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: License Management-secrets
              key: database-password
```

---

## Configuration Validation

### Startup Checks

The application validates configuration on startup:

```
✓ JWT_SECRET is set and sufficient length
✓ JWT_REFRESH_SECRET is set and sufficient length
✓ Database connection successful
✓ Redis connection successful
✓ SMTP configuration valid (if enabled)
```

### Manual Validation

```bash
# Test database connection
psql $DATABASE_URL -c "SELECT 1"

# Test Redis connection
redis-cli -h $REDIS_HOST -p $REDIS_PORT ping

# Test SMTP
telnet $SMTP_HOST $SMTP_PORT
```

---

## Troubleshooting

### Common Issues

**Database Connection Failed:**
```bash
# Check connectivity
nc -zv localhost 5432

# Verify credentials
psql postgres://user:pass@localhost:5432/dbname

# Check logs
docker logs postgres-container
```

**Redis Connection Failed:**
```bash
# Check connectivity
redis-cli -h localhost -p 6379 ping

# With password
redis-cli -h localhost -p 6379 -a password ping
```

**JWT Errors:**
```bash
# Ensure secrets are set
echo $JWT_SECRET | wc -c          # Should be > 32

# Regenerate if needed
openssl rand -base64 32
```

### Debug Mode

```bash
# Enable verbose logging
LOG_LEVEL=debug
ENVIRONMENT=development

# Run application
go run cmd/api/main.go
```

---

## Best Practices

### Security

- ✅ Use strong, unique secrets (min 32 characters)
- ✅ Rotate secrets regularly (quarterly)
- ✅ Use environment-specific configurations
- ✅ Never commit secrets to version control
- ✅ Use secret managers in production (AWS Secrets Manager, etc.)

### Performance

- ✅ Tune database connection pool based on load
- ✅ Enable Redis connection pooling
- ✅ Use appropriate cache TTLs
- ✅ Monitor and adjust rate limits

### Monitoring

- ✅ Use JSON logging in production
- ✅ Ship logs to centralized service (ELK, Datadog)
- ✅ Set up alerts for errors and anomalies
- ✅ Monitor database and Redis performance

---

## Quick Reference

### Minimum Production Configuration

```bash
# Required
PORT=8080
ENVIRONMENT=production
DATABASE_URL=postgres://...
JWT_SECRET=min-32-chars
JWT_REFRESH_SECRET=min-32-chars

# Recommended
REDIS_HOST=redis
SMTP_HOST=smtp.example.com
CORS_ALLOWED_ORIGINS=https://app.example.com
LOG_FORMAT=json
LOG_LEVEL=info
```

### Full Example

See `.env.example` in the repository root for a complete configuration template.

---

**Document Maintained By**: License Management Development Team  
**Questions?** support@License Management.com

