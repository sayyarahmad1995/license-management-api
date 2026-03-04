# Security Guide

**Last Updated**: March 4, 2026  
**Version**: 1.0

This document outlines security best practices, configurations, and guidelines for deploying and maintaining the License Management API.

---

## Table of Contents

1. [Authentication & Authorization](#authentication--authorization)
2. [Data Protection](#data-protection)
3. [Network Security](#network-security)
4. [Secrets Management](#secrets-management)
5. [Input Validation](#input-validation)
6. [Rate Limiting](#rate-limiting)
7. [Audit & Monitoring](#audit--monitoring)
8. [Deployment Security](#deployment-security)
9. [Security Headers](#security-headers)
10. [Incident Response](#incident-response)

---

## Authentication & Authorization

### JWT Token Security

**Token Configuration:**
```yaml
JWT_SECRET: <256-bit random secret>
JWT_EXPIRY: 15m              # Access token lifetime
JWT_REFRESH_EXPIRY: 7d       # Refresh token lifetime
```

**Best Practices:**
- ✅ Use strong, randomly generated JWT secrets (minimum 256 bits)
- ✅ Rotate JWT secrets periodically (every 90 days recommended)
- ✅ Keep access token expiry short (15 minutes)
- ✅ Store refresh tokens securely in HTTP-only cookies or secure storage
- ✅ Implement token revocation for logout and security events

**Token Revocation:**
```bash
# Revoke token via Redis
redis-cli DEL "revoked:token:<token_id>"

# Clear all user sessions
redis-cli KEYS "session:*" | xargs redis-cli DEL
```

### Password Security

**Current Implementation:**
- Bcrypt hashing with cost factor 10
- Minimum 8 characters enforced at API level
- Password reset tokens expire in 1 hour
- Email verification tokens expire in 24 hours

**Recommended Policies:**
```go
// Enforce strong password requirements
- Minimum 12 characters
- At least 1 uppercase letter
- At least 1 lowercase letter
- At least 1 number
- At least 1 special character
```

**Password Reset Flow:**
1. User requests reset via email
2. Server generates cryptographically secure token
3. Token stored in database with 1-hour expiry
4. Link sent to verified email address
5. Token invalidated after use
6. Audit log records password change

### Role-Based Access Control (RBAC)

**Available Roles:**
- `Admin` - Full system access
- `Manager` - License management, user viewing
- `User` - Own license viewing only

**Permission Matrix:**
| Endpoint | Admin | Manager | User |
|----------|-------|---------|------|
| Create License | ✅ | ✅ | ❌ |
| View All Licenses | ✅ | ✅ | ❌ |
| View Own Licenses | ✅ | ✅ | ✅ |
| Delete License | ✅ | ❌ | ❌ |
| Manage Users | ✅ | ❌ | ❌ |
| View Audit Logs | ✅ | ✅ | ❌ |

**Securing Endpoints:**
```go
// Use auth middleware with role requirements
router.Use(middleware.AuthMiddleware(deps))
router.Use(middleware.RequireRole("Admin"))
```

---

## Data Protection

### Database Security

**Connection Security:**
```yaml
# Use SSL/TLS for database connections
DATABASE_URL: postgres://user:pass@host:5432/db?sslmode=require

# Connection pooling limits
DATABASE_MAX_OPEN_CONNS: 25
DATABASE_MAX_IDLE_CONNS: 5
DATABASE_CONN_MAX_LIFETIME: 5m
```

**Data Encryption:**
- ✅ Passwords: Bcrypt hashed (never stored in plaintext)
- ✅ Tokens: Stored as cryptographic hashes
- ⚠️ License Keys: Stored in plaintext (consider encryption at rest)
- ✅ Audit Logs: Tamper-evident via timestamps and user tracking

**Backup Security:**
```bash
# Encrypt database backups
pg_dump -h localhost -U postgres license_mgmt | gpg --encrypt -r admin@example.com > backup.sql.gpg

# Automated encrypted backups (cron)
0 2 * * * /scripts/backup-db.sh | gpg --encrypt -r backup@example.com > /backups/$(date +\%Y\%m\%d).sql.gpg
```

### Sensitive Data Handling

**PII Protection:**
- Email addresses stored but not exposed in public APIs
- IP addresses logged in audit trail (consider GDPR implications)
- User data deletions must cascade to audit logs (configurable)

**Data Retention:**
```yaml
AUDIT_LOG_RETENTION_DAYS: 90        # Keep audit logs for 90 days
SESSION_RETENTION_DAYS: 30          # Clean up old sessions
TOKEN_RETENTION_DAYS: 7             # Clean up expired tokens
```

---

## Network Security

### HTTPS/TLS Configuration

**Production Requirements:**
- ✅ Always use HTTPS in production
- ✅ TLS 1.2 or higher
- ✅ Strong cipher suites only
- ❌ Never use self-signed certificates in production

**Nginx/Load Balancer Configuration:**
```nginx
server {
    listen 443 ssl http2;
    server_name api.example.com;

    ssl_certificate /etc/letsencrypt/live/api.example.com/fullchain.pem;
    ssl_certificate_key /etc/letsencrypt/live/api.example.com/privkey.pem;
    ssl_protocols TLSv1.2 TLSv1.3;
    ssl_ciphers HIGH:!aNULL:!MD5;
    ssl_prefer_server_ciphers on;

    # Security headers
    add_header Strict-Transport-Security "max-age=31536000; includeSubDomains" always;
    add_header X-Frame-Options "SAMEORIGIN" always;
    add_header X-Content-Type-Options "nosniff" always;
    add_header X-XSS-Protection "1; mode=block" always;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### CORS Configuration

**Current Settings:**
```yaml
CORS_ALLOWED_ORIGINS: https://app.example.com
CORS_ALLOWED_METHODS: GET,POST,PUT,DELETE,OPTIONS
CORS_ALLOWED_HEADERS: Authorization,Content-Type
CORS_ALLOW_CREDENTIALS: true
CORS_MAX_AGE: 3600
```

**Production Checklist:**
- ✅ Restrict origins to known domains (no wildcards)
- ✅ Use HTTPS origins only
- ✅ Minimize allowed methods
- ✅ Validate Origin header on server side

### Firewall Rules

**Recommended Firewall Configuration:**
```bash
# Allow HTTPS
ufw allow 443/tcp

# Allow SSH (restrict to known IPs)
ufw allow from 203.0.113.0/24 to any port 22

# Block direct database access
ufw deny 5432/tcp

# Block direct Redis access
ufw deny 6379/tcp

# Enable firewall
ufw enable
```

---

## Secrets Management

### Environment Variables

**Never commit secrets to version control!**

**Secure Storage Options:**

1. **Local Development:**
   ```bash
   # Use .env file (git-ignored)
   cp .env.example .env
   # Edit .env with local values
   ```

2. **Docker/Compose:**
   ```yaml
   # Use Docker secrets or env_file
   services:
     api:
       env_file: .env.production
       secrets:
         - jwt_secret
         - db_password
   ```

3. **Kubernetes:**
   ```yaml
   # Use Kubernetes Secrets
   apiVersion: v1
   kind: Secret
   metadata:
     name: license-mgmt-secrets
   type: Opaque
   data:
     jwt-secret: <base64-encoded>
     db-password: <base64-encoded>
   ```

4. **Cloud Providers:**
   - AWS: AWS Secrets Manager / Parameter Store
   - Azure: Azure Key Vault
   - GCP: Google Secret Manager

### Secret Rotation

**JWT Secret Rotation:**
```bash
# Generate new secret
openssl rand -base64 32

# Update environment
export JWT_SECRET="new_secret_here"

# Restart service
docker-compose restart api
```

**Database Password Rotation:**
1. Create new password in database
2. Update environment variable
3. Restart application
4. Revoke old password

---

## Input Validation

### Request Validation

**Implemented Validations:**
- Email format validation (RFC 5322)
- Password strength (minimum 8 chars)
- License key format validation
- Pagination bounds (page >= 1, pageSize <= 100)
- Date/time parsing with timezone handling

**Protection Against Common Attacks:**

**SQL Injection:**
- ✅ GORM prepared statements used throughout
- ✅ Parameterized queries (no string concatenation)

**XSS (Cross-Site Scripting):**
- ✅ Content-Type headers enforced
- ✅ Input sanitization on user-provided fields
- ✅ No direct HTML rendering in responses

**Path Traversal:**
- ✅ File paths validated and sanitized
- ✅ No direct file system access from user input

**Command Injection:**
- ✅ No shell command execution from user input
- ✅ External commands properly escaped

### Rate Limiting

See dedicated section below.

---

## Rate Limiting

### Configuration

**Current Limits:**
```yaml
RATELIMIT_AUTH_ENABLED: true
RATELIMIT_AUTH_REQUESTS: 5           # 5 requests
RATELIMIT_AUTH_WINDOW: 60            # per 60 seconds
RATELIMIT_API_REQUESTS: 100          # 100 requests
RATELIMIT_API_WINDOW: 60             # per 60 seconds
```

### Protected Endpoints

**Authentication Endpoints** (5 req/min):
- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/password-reset-request`
- `POST /api/auth/verify-email`

**General API Endpoints** (100 req/min):
- All other authenticated endpoints

### Redis-Based Rate Limiting

**Storage:**
- Rate limit counters stored in Redis
- Keys: `ratelimit:ip:<ip_address>:<endpoint>`
- TTL: Automatically expires after window

**Monitoring:**
```bash
# Check rate limit keys
redis-cli KEYS "ratelimit:*"

# Check specific IP
redis-cli GET "ratelimit:ip:192.168.1.100:/api/auth/login"

# Clear rate limits (emergency)
redis-cli DEL $(redis-cli KEYS "ratelimit:*")
```

### DDoS Protection

**Additional Protections:**
1. Use Cloudflare or similar CDN with DDoS protection
2. Implement IP blacklisting for repeat offenders
3. Use connection limiting at nginx/load balancer level
4. Monitor for unusual traffic patterns

```nginx
# Nginx rate limiting
limit_req_zone $binary_remote_addr zone=api:10m rate=10r/s;
limit_req zone=api burst=20 nodelay;
```

---

## Audit & Monitoring

### Audit Logging

**What's Logged:**
- User registration and login attempts
- License creation, modification, deletion
- Password changes and resets
- Email verification events
- License activations/deactivations
- Role changes

**Audit Log Format:**
```json
{
  "id": 12345,
  "action": "LICENSE_CREATE",
  "entity_type": "License",
  "entity_id": 42,
  "user_id": 7,
  "ip_address": "203.0.113.25",
  "details": "Created license ABC-123-XYZ",
  "timestamp": "2026-03-04T10:30:00Z"
}
```

**Querying Audit Logs:**
```sql
-- Recent login attempts
SELECT * FROM audit_logs 
WHERE action = 'LOGIN' 
ORDER BY timestamp DESC 
LIMIT 100;

-- Failed login attempts by IP
SELECT ip_address, COUNT(*) as attempts 
FROM audit_logs 
WHERE action = 'LOGIN_FAILED' 
  AND timestamp > NOW() - INTERVAL '1 hour'
GROUP BY ip_address 
ORDER BY attempts DESC;

-- User activity
SELECT * FROM audit_logs 
WHERE user_id = 123 
ORDER BY timestamp DESC;
```

### Application Logging

**Log Levels:**
- `DEBUG`: Development debugging (disable in production)
- `INFO`: Normal operations, requests
- `WARN`: Unexpected but handled situations
- `ERROR`: Errors requiring attention

**Centralized Logging:**
```yaml
# Recommended: Ship logs to centralized service
LOG_FORMAT: json
LOG_OUTPUT: stdout  # Captured by container runtime

# Examples:
# - ELK Stack (Elasticsearch, Logstash, Kibana)
# - Datadog
# - Splunk
# - New Relic
```

### Health Monitoring

**Health Check Endpoints:**
- `GET /health` - Overall health status
- `GET /health/liveness` - Kubernetes liveness probe
- `GET /health/readiness` - Kubernetes readiness probe

**Monitoring Checklist:**
- ✅ Database connectivity
- ✅ Redis connectivity
- ✅ Disk space
- ✅ Memory usage
- ✅ Response times
- ✅ Error rates

**Alerting Thresholds:**
```yaml
# Example Prometheus alerts
- alert: HighErrorRate
  expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
  
- alert: SlowResponses
  expr: histogram_quantile(0.95, http_request_duration_seconds) > 1
  
- alert: DatabaseDown
  expr: up{job="postgres"} == 0
```

---

## Deployment Security

### Container Security

**Docker Best Practices:**
- ✅ Use multi-stage builds (implemented)
- ✅ Run as non-root user
- ✅ Minimal base images (alpine)
- ✅ No secrets in Dockerfile
- ✅ Scan images for vulnerabilities

**Image Scanning:**
```bash
# Scan with Trivy
trivy image license-management-api:latest

# Scan with Snyk
snyk container test license-management-api:latest
```

### Kubernetes Security

**Pod Security:**
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  capabilities:
    drop:
      - ALL
  readOnlyRootFilesystem: true
  allowPrivilegeEscalation: false
```

**Network Policies:**
```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: license-mgmt-api-policy
spec:
  podSelector:
    matchLabels:
      app: license-mgmt-api
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - podSelector:
        matchLabels:
          app: nginx-ingress
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - podSelector:
        matchLabels:
          app: postgres
    ports:
    - protocol: TCP
      port: 5432
  - to:
    - podSelector:
        matchLabels:
          app: redis
    ports:
    - protocol: TCP
      port: 6379
```

**Secrets Management:**
```yaml
# Use external secrets operator
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secrets-manager
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
```

### Cloud Deployment

**AWS Security:**
- Use VPC with private subnets
- Security groups with least privilege
- Enable GuardDuty
- Use IAM roles (no access keys)
- Enable CloudTrail logging

**GCP Security:**
- Use VPC with firewall rules
- Enable Cloud Security Command Center
- Use service accounts with minimal permissions
- Enable Cloud Logging and Monitoring

---

## Security Headers

### Implemented Headers

The application sets the following security headers:

```go
// Content Security Policy
Content-Security-Policy: default-src 'self'

// XSS Protection
X-XSS-Protection: 1; mode=block

// Frame Options
X-Frame-Options: SAMEORIGIN

// Content Type Sniffing
X-Content-Type-Options: nosniff

// Referrer Policy
Referrer-Policy: strict-origin-when-cross-origin

// HSTS (set by reverse proxy)
Strict-Transport-Security: max-age=31536000; includeSubDomains
```

### Verifying Headers

```bash
# Check security headers
curl -I https://api.example.com/health

# Use securityheaders.com
# https://securityheaders.com/?q=https://api.example.com
```

---

## Incident Response

### Security Incident Playbook

**1. Detection:**
- Monitor logs for suspicious activity
- Set up alerts for anomalies
- Regular security audits

**2. Containment:**
```bash
# Immediately revoke compromised tokens
redis-cli KEYS "token:*" | xargs redis-cli DEL

# Block malicious IPs
ufw deny from <malicious_ip>

# Disable compromised accounts
psql -c "UPDATE users SET status='Blocked' WHERE id=<user_id>"
```

**3. Investigation:**
```sql
-- Review audit logs
SELECT * FROM audit_logs 
WHERE timestamp > NOW() - INTERVAL '24 hours'
  AND (action LIKE '%FAILED%' OR ip_address = '<suspicious_ip>')
ORDER BY timestamp DESC;

-- Check for unauthorized access
SELECT * FROM audit_logs 
WHERE user_id = <compromised_user_id>
ORDER BY timestamp DESC;
```

**4. Recovery:**
- Rotate all secrets (JWT, database passwords)
- Force password resets for affected users
- Review and patch vulnerabilities
- Restore from clean backups if necessary

**5. Post-Incident:**
- Document incident timeline
- Update security policies
- Implement additional security controls
- Notify affected parties (if PII compromised)

### Emergency Contacts

```
Security Team Lead: security@example.com
On-Call Engineer: +1-XXX-XXX-XXXX
Cloud Provider Support: [Provider-specific]
```

### Vulnerability Disclosure

**Reporting Security Issues:**
- Email: security@example.com
- PGP Key: [Public key fingerprint]
- Response Time: 48 hours
- Coordinated disclosure timeline: 90 days

---

## Security Checklist

### Pre-Production Deployment

- [ ] All secrets stored securely (no hardcoded values)
- [ ] HTTPS/TLS enabled with valid certificates
- [ ] Strong JWT secret generated and rotated
- [ ] Database connection uses SSL
- [ ] Redis requires password authentication
- [ ] CORS restricted to known origins
- [ ] Rate limiting enabled on all endpoints
- [ ] Security headers configured
- [ ] Audit logging enabled
- [ ] Health checks functional
- [ ] Backups configured and encrypted
- [ ] Monitoring and alerting set up
- [ ] Firewall rules configured (least privilege)
- [ ] Container images scanned for vulnerabilities
- [ ] Kubernetes security contexts applied
- [ ] Network policies implemented
- [ ] Incident response plan documented

### Regular Maintenance

- [ ] Weekly: Review audit logs for suspicious activity
- [ ] Weekly: Check for security updates (Go, dependencies)
- [ ] Monthly: Rotate JWT secrets
- [ ] Monthly: Review and update firewall rules
- [ ] Quarterly: Security audit and penetration testing
- [ ] Quarterly: Rotate database passwords
- [ ] Quarterly: Review access control policies
- [ ] Annually: Security training for development team

---

## Additional Resources

- [OWASP Top 10](https://owasp.org/www-project-top-ten/)
- [OWASP API Security Top 10](https://owasp.org/www-project-api-security/)
- [Go Security Checklist](https://github.com/Checkmarx/Go-SCP)
- [Docker Security Best Practices](https://docs.docker.com/develop/security-best-practices/)
- [Kubernetes Security](https://kubernetes.io/docs/concepts/security/)

---

**Document Maintained By**: Security Team  
**Questions?** security@example.com
