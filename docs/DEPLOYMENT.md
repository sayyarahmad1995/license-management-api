# License Management API - Deployment Guide

**Last Updated**: March 4, 2026

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Local Development](#local-development)
3. [Docker Deployment](#docker-deployment)
4. [Production Deployment](#production-deployment)
5. [Environment Configuration](#environment-configuration)
6. [Database Setup](#database-setup)
7. [Security Checklist](#security-checklist)
8. [Monitoring & Logging](#monitoring--logging)
9. [Troubleshooting](#troubleshooting)

---

## Quick Start

### Prerequisites
- Go 1.21+
- PostgreSQL 13+
- Redis 7+
- Docker & Docker Compose (for containerized deployment)

### Development Quick Start (5 minutes)

```bash
# 1. Clone repository
git clone <repo-url>
cd license-management-api

# 2. Install dependencies
go mod download

# 3. Copy environment template
cp .env.example .env

# 4. Start services with Docker Compose
docker-compose up -d

# 5. Run migrations (automatic via Docker entrypoint)
go run ./cmd/api/main.go

# 6. API available at http://localhost:8080
# Swagger UI at http://localhost:8080/swagger
```

---

## Local Development

### Setup Steps

#### 1. Prerequisites Installation

**macOS/Linux:**
```bash
# Go installation
brew install go@1.21

# PostgreSQL
brew install postgresql

# Redis
brew install redis

# Start services
brew services start postgresql
brew services start redis
```

**Windows (with Chocolatey):**
```bash
choco install golang postgresql redis

# Start services (or use Docker)
```

#### 2. Environment Configuration

```bash
# Copy environment file
cp .env.example .env

# Edit .env with your settings
nano .env
```

**Minimal .env for development:**
```env
# Server
PORT=8080
ENV=development

# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=license_mgmt_dev
DB_USER=postgres
DB_PASSWORD=postgres

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET_KEY=dev-secret-key-change-in-production

# Admin user
ADMIN_EMAIL=admin@example.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=Admin123!
```

#### 3. Database Setup

```bash
# Create database
createdb license_mgmt_dev

# Run migrations (automatic on startup via GORM auto-migration)
# Or manually run migrations if needed
go run ./cmd/api/main.go
```

#### 4. Start Development Server

```bash
# Run with hot reload (install air: github.com/cosmtrek/air)
air

# Or run directly
go run ./cmd/api/main.go
```

#### 5. Verify Setup

```bash
# Health check
curl http://localhost:8080/health

# Swagger UI
open http://localhost:8080/swagger

# Try register
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "username": "testuser",
    "password": "Test123!@#"
  }'
```

---

## Docker Deployment

### Docker Compose (Recommended for Development)

```bash
# Start all services
docker-compose up -d

# View logs
docker-compose logs -f api

# Stop services
docker-compose down

# Rebuild images
docker-compose build --no-cache api
```

### Docker Compose Services

The `docker-compose.yml` includes:
- **API**: `http://localhost:8080`
- **PostgreSQL**: Database at `localhost:5432`
- **Redis**: Cache at `localhost:6379` (optional)
- **Swagger UI**: `http://localhost:8080/swagger`

### Manual Docker Build

```bash
# Build image
docker build -t license-management-api:latest .

# Run container
docker run -d \
  -p 8080:8080 \
  -e DB_HOST=postgres \
  -e REDIS_HOST=redis \
  -e JWT_SECRET_KEY=your-secret \
  --network license-mgmt-net \
  --name license-mgmt-api \
  license-management-api:latest

# View logs
docker logs -f license-mgmt-api
```

---

## Production Deployment

### Pre-Deployment Checklist

- [ ] Generate strong JWT secret key (`openssl rand -hex 32`)
- [ ] Configure prod database with backups enabled
- [ ] Set up Redis with persistence
- [ ] Configure SMTP for email (or use console mode)
- [ ] Enable HTTPS/TLS
- [ ] Configure firewall rules
- [ ] Set up monitoring/alerting
- [ ] Enable audit logging
- [ ] Configure CORS properly
- [ ] Set up log aggregation

### Environment Variables (Production)

```env
# Server
PORT=8080
ENV=production

# Database (use managed service if possible)
DB_HOST=prod-postgres.example.com
DB_PORT=5432
DB_NAME=license_mgmt_prod
DB_USER=license_mgmt_app
DB_PASSWORD=<STRONG_PASSWORD>
DB_SSL_MODE=require
DB_MAX_OPEN_CONNS=25
DB_MAX_IDLE_CONNS=5

# Redis
REDIS_HOST=prod-redis.example.com
REDIS_PORT=6379
REDIS_PASSWORD=<STRONG_PASSWORD>

# Security
JWT_SECRET_KEY=<GENERATE_NEW>
ADMIN_EMAIL=admin@company.com
ADMIN_USERNAME=admin
ADMIN_PASSWORD=<STRONG_PASSWORD>

# Email
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=noreply@company.com
SMTP_PASSWORD=<APP_PASSWORD>
SMTP_FROM=noreply@company.com

# CORS
CORS_ORIGINS=https://app.example.com,https://www.example.com

# Security Headers
ENABLE_SECURITY_HEADERS=true
ENABLE_CSRF=true

# Rate Limiting  
RATE_LIMIT_ENABLED=true
LOGIN_MAX_ATTEMPTS=5
LOGIN_LOCKOUT_DURATION=15m

# Monitoring
ENABLE_METRICS=true
LOG_LEVEL=info
```

### Docker Production Deployment

```bash
# Build production image
docker build -t license-management-api:v1.0.0 .

# Push to registry
docker tag license-management-api:v1.0.0 registry.example.com/license-management-api:v1.0.0
docker push registry.example.com/license-management-api:v1.0.0

# Run with environment file
docker run -d \
  -p 8080:8080 \
  --env-file .env.prod \
  --restart unless-stopped \
  --health-cmd='curl -f http://localhost:8080/health || exit 1' \
  --health-interval=30s \
  --health-timeout=5s \
  --health-retries=3 \
  --name license_mgmt-api \
  registry.example.com/license-management-api:v1.0.0
```

### Kubernetes Deployment

```bash
# Create namespace
kubectl create namespace license-mgmt

# Create secrets
kubectl create secret generic license_mgmt-secrets \
  --from-literal=jwt-secret=$(openssl rand -hex 32) \
  --from-literal=db-password=<PASSWORD> \
  -n license-mgmt

# Deploy application
kubectl apply -f k8s/deployment.yaml
kubectl apply -f k8s/service.yaml

# Check deployment
kubectl get pods -n license-mgmt
kubectl logs -n license-mgmt deployment/license-mgmt-api
```

---

## Environment Configuration

### Core Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | 8080 | Server port |
| `ENV` | development | Environment (development/production) |
| `LOG_LEVEL` | info | Logging level (debug/info/warn/error) |

### Database Settings

| Variable | Default | Required |
|----------|---------|----------|
| `DB_HOST` | localhost | ✅ Database host |
| `DB_PORT` | 5432 | ✅ Database port |
| `DB_NAME` | license_mgmt | ✅ Database name |
| `DB_USER` | postgres | ✅ Database user |
| `DB_PASSWORD` | | ✅ Database password |
| `DB_SSL_MODE` | disable | SSL mode (disable/require) |
| `DB_MAX_OPEN_CONNS` | 25 | Max open connections |
| `DB_MAX_IDLE_CONNS` | 5 | Max idle connections |

### Redis Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `REDIS_HOST` | localhost | Redis host |
| `REDIS_PORT` | 6379 | Redis port |
| `REDIS_PASSWORD` | | Redis password |
| `REDIS_DB` | 0 | Redis database number |

### Security Settings

| Variable | Default | Description |
|----------|---------|-------------|
| `JWT_SECRET_KEY` | | ✅ JWT signing key (min 32 chars) |
| `JWT_EXPIRY` | 3600 | Token expiry (seconds) |
| `REFRESH_TOKEN_EXPIRY` | 604800 | Refresh token expiry (seconds) |
| `EMAIL_VERIFY_TOKEN_EXPIRY` | 86400 | Email verification expiry |
| `PASSWORD_RESET_TOKEN_EXPIRY` | 3600 | Password reset expiry |

### CORS Settings

```env
# Space or comma-separated origins
CORS_ORIGINS=https://app.example.com,https://www.example.com
```

---

## Database Setup

### PostgreSQL

#### Create User and Database

```sql
-- Create user
CREATE USER license_mgmt_app WITH PASSWORD 'strong_password_here';

-- Create database
CREATE DATABASE license_mgmt_prod OWNER license_mgmt_app;

-- Grant privileges
GRANT CONNECT ON DATABASE license_mgmt_prod TO license_mgmt_app;
GRANT USAGE ON SCHEMA public TO license_mgmt_app;
GRANT CREATE ON SCHEMA public TO license_mgmt_app;
```

#### Backup & Recovery

```bash
# Backup
pg_dump -U license_mgmt_app -h localhost license_mgmt_prod > backup.sql

# Restore
psql -U license_mgmt_app -h localhost license_mgmt_prod < backup.sql

# Automated backup (cron job)
0 2 * * * pg_dump -U license_mgmt_app -h localhost license_mgmt_prod > /backups/license_mgmt_$(date +\%Y\%m\%d).sql
```

### Database Indexes

The application creates necessary indexes automatically via GORM. Key indexes:
- License key (unique)
- User ID on licenses/activations
- Status on licenses
- Created/Updated timestamps
- Audit log indexes

---

## Security Checklist

### Before Going Live

- [ ] **Secrets Management**
  - [ ] Generate strong JWT secret key
  - [ ] Store secrets in secure vault (not in code)
  - [ ] Rotate secrets regularly
  - [ ] Never commit secrets to git

- [ ] **HTTPS/TLS**
  - [ ] Enable HTTPS with valid certificate
  - [ ] Enforce HTTPS redirect
  - [ ] Use strong TLS version (1.2+)
  - [ ] Configure HSTS header

- [ ] **Authentication**
  - [ ] Enforce strong password requirements
  - [ ] Enable email verification
  - [ ] Test password reset flow
  - [ ] Configure session timeout

- [ ] **Database**
  - [ ] Enable SSL connections
  - [ ] Restrict database access
  - [ ] Configure backups
  - [ ] Set up monitoring
  - [ ] Use strong credentials

- [ ] **API Security**
  - [ ] Enable rate limiting
  - [ ] Enable CORS properly
  - [ ] Enable CSRF protection
  - [ ] Validate all inputs
  - [ ] Implement audit logging

- [ ] **Operations**
  - [ ] Configure log aggregation
  - [ ] Set up monitoring/alerts
  - [ ] Test graceful shutdown
  - [ ] Document runbooks
  - [ ] Set up backup schedule

---

## Monitoring & Logging

### Metrics Endpoint

```bash
curl http://localhost:8080/metrics
```

Returns Prometheus metrics for:
- HTTP request latency/count
- Database query performance
- Cache hit rates
- Active connections
- Error rates

### Logging

Configure log levels and formats:

```env
LOG_LEVEL=info          # debug, info, warn, error
LOG_FORMAT=json         # json, text
LOG_OUTPUT=stdout       # stdout, file, both
LOG_FILE=/var/log/license-mgmt/app.log
```

### Health Checks

```bash
# Liveness (service is running)
curl http://localhost:8080/livez

# Readiness (ready for traffic)
curl http://localhost:8080/readyz

# Startup (bootstrap complete)
curl http://localhost:8080/startup
```

### Alerting Recommendations

- Error rate > 1%
- P95 request latency > 1s
- Database connection limit > 80%
- Redis memory usage > 80%
- Disk space < 10%
- Backup failure
- Expired license count

---

## Troubleshooting

### Common Issues

#### Connection Refused
```bash
# Check if service is running
docker ps

# Check logs
docker logs license_mgmt-api

# Verify ports
netstat -an | grep 8080
```

#### Database Connection Error
```bash
# Test PostgreSQL connection
psql -h localhost -U postgres -d license_mgmt_dev

# Check credentials in .env
cat .env | grep DB_

# Verify database exists
psql -h localhost -U postgres -l
```

#### JWT Secret Missing
```bash
# Generate secret
openssl rand -hex 32

# Set in .env
JWT_SECRET_KEY=<generated-value>
```

#### Email Not Sending
```bash
# Verify SMTP config
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your-email@gmail.com

# Use app password for Gmail (not main password)
# Enable console mode for development
```

### Performance Tuning

```env
# Connection pooling
DB_MAX_OPEN_CONNS=50
DB_MAX_IDLE_CONNS=10

# Cache TTL (seconds)
CACHE_TTL=900

# Rate limiting
RATE_LIMIT_ENABLED=true
LOGIN_MAX_ATTEMPTS=5

# Pagination defaults
PAGE_SIZE_DEFAULT=20
PAGE_SIZE_MAX=100
```

---

## Support

For deployment issues:
1. Check logs: `docker logs license-mgmt-api`
2. Verify environment: `docker exec license-mgmt-api env | grep DB_`
3. Test connectivity: `curl http://localhost:8080/health`
4. Check database: `psql -h localhost -U postgres -d license_mgmt_dev`
5. Contact: support@example.com

---

## Additional Resources

- [API Reference](./API_REFERENCE.md)
- [Security Guide](./SECURITY.md)
- [Configuration Guide](./CONFIGURATION.md)
- [Troubleshooting Guide](./TROUBLESHOOTING.md)


