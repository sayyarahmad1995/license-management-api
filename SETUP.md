# Developer Setup Guide

**Last Updated**: March 4, 2026  
**Version**: 1.0

Welcome to the License Management API project! This guide will help you set up your development environment and get the application running locally.

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Quick Start](#quick-start)
3. [Detailed Setup](#detailed-setup)
4. [Project Structure](#project-structure)
5. [Development Workflow](#development-workflow)
6. [Testing](#testing)
7. [Building & Deployment](#building--deployment)
8. [Troubleshooting](#troubleshooting)
9. [Contributing](#contributing)

---

## Prerequisites

### Required Software

| Software | Version | Purpose |
|----------|---------|---------|
| **Go** | 1.21+ | Programming language |
| **Docker** | 20.10+ | Containerization |
| **Docker Compose** | 2.0+ | Multi-container orchestration |
| **PostgreSQL** | 15+ | Primary database |
| **Redis** | 7+ | Caching & sessions |
| **Git** | 2.30+ | Version control |

### Optional Tools

- **Make** - Build automation
- **kubectl** - Kubernetes deployment
- **Helm** - Kubernetes package manager
- **Postman/Insomnia** - API testing
- **TablePlus/DBeaver** - Database GUI

---

## Quick Start

### 1. Clone Repository

```bash
git clone https://github.com/your-org/license-management-api.git
cd license-management-api
```

### 2. Environment Setup

```bash
# Copy environment template
cp .env.example .env

# Generate JWT secrets
openssl rand -base64 32  # Copy to JWT_SECRET
openssl rand -base64 32  # Copy to JWT_REFRESH_SECRET

# Edit .env with your values
nano .env
```

### 3. Start Services with Docker Compose

```bash
# Start all services (API, PostgreSQL, Redis)
docker-compose up -d

# View logs
docker-compose logs -f api

# Check services
docker-compose ps
```

### 4. Verify Installation

```bash
# Health check
curl http://localhost:8080/health

# Expected response:
# {"status":"healthy","database":"connected","redis":"connected"}
```

🎉 **You're ready to develop!**

---

## Detailed Setup

### Step 1: Install Go

**macOS:**
```bash
brew install go
```

**Linux (Ubuntu/Debian):**
```bash
wget https://go.dev/dl/go1.22.0.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.22.0.linux-amd64.tar.gz
export PATH=$PATH:/usr/local/go/bin
```

**Windows:**
Download installer from [go.dev](https://go.dev/dl/)

**Verify Installation:**
```bash
go version
# Expected: go version go1.22.0 ...
```

### Step 2: Install Docker & Docker Compose

**macOS:**
```bash
brew install --cask docker
```

**Linux (Ubuntu/Debian):**
```bash
# Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh

# Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

**Windows:**
Download Docker Desktop from [docker.com](https://www.docker.com/products/docker-desktop)

**Verify Installation:**
```bash
docker --version
docker-compose --version
```

### Step 3: Clone and Configure

```bash
# Clone repository
git clone https://github.com/your-org/license-management-api.git
cd license-management-api

# Install Go dependencies
go mod download

# Verify dependencies
go mod verify
```

### Step 4: Configure Environment

**Create .env file:**
```bash
cp .env.example .env
```

**Minimum Configuration:**
```bash
# .env
PORT=8080
ENVIRONMENT=development

# Database
DATABASE_URL=postgres://postgres:password@localhost:5432/license_mgmt?sslmode=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT Secrets (generate with: openssl rand -base64 32)
JWT_SECRET=your-super-secret-key-minimum-32-characters-long
JWT_REFRESH_SECRET=another-super-secret-key-also-32-chars

# Email (console mode for development)
SMTP_USE_CONSOLE=true
FRONTEND_BASE_URL=http://localhost:3000

# CORS (allow all in development)
CORS_ALLOWED_ORIGINS=*

# Logging
LOG_LEVEL=debug
LOG_FORMAT=text
```

### Step 5: Start Infrastructure

**Option A: Docker Compose (Recommended)**
```bash
# Start PostgreSQL + Redis + API
docker-compose up -d

# View logs
docker-compose logs -f

# Stop services
docker-compose down

# Clean up volumes
docker-compose down -v
```

**Option B: Local Services**

**PostgreSQL:**
```bash
# macOS
brew install postgresql@15
brew services start postgresql@15

# Linux
sudo apt install postgresql-15
sudo systemctl start postgresql

# Create database
createdb license_mgmt
```

**Redis:**
```bash
# macOS
brew install redis
brew services start redis

# Linux
sudo apt install redis-server
sudo systemctl start redis
```

### Step 6: Run Application

**Development Mode:**
```bash
# Run directly
go run cmd/api/main.go

# With hot reload (install air first)
go install github.com/cosmtrek/air@latest
air
```

**Build and Run:**
```bash
# Build binary
go build -o bin/app cmd/api/main.go

# Run binary
./bin/app
```

**With Make:**
```bash
# Run
make run

# Build
make build

# Test
make test

# Clean
make clean
```

---

## Project Structure

```
license-management-api/
├── cmd/
│   └── api/
│       └── main.go              # Application entry point
├── internal/
│   ├── config/                  # Configuration management
│   │   └── config.go
│   ├── database/                # Database connection
│   │   ├── database.go
│   │   └── database_test.go
│   ├── dto/                     # Data Transfer Objects
│   │   ├── auth.go
│   │   ├── license.go
│   │   └── common.go
│   ├── errors/                  # Custom error types
│   │   └── errors.go
│   ├── handler/                 # HTTP handlers
│   │   ├── auth_handler.go
│   │   ├── license_handler.go
│   │   └── health_handler.go
│   ├── middleware/              # HTTP middleware
│   │   └── auth.go
│   ├── models/                  # Database models
│   │   ├── user.go
│   │   ├── license.go
│   │   └── audit_log.go
│   ├── repository/              # Data access layer
│   │   ├── user_repository.go
│   │   └── license_repository.go
│   ├── server/                  # HTTP server setup
│   │   ├── server.go
│   │   └── routes.go
│   └── service/                 # Business logic
│       ├── auth_service.go
│       └── license_service.go
├── pkg/
│   └── utils/                   # Utility functions
│       ├── crypto.go
│       └── http.go
├── deployments/
│   ├── docker/                  # Docker configurations
│   ├── kubernetes/              # K8s manifests
│   └── helm/                    # Helm charts
├── docs/                        # API documentation
├── .env.example                 # Environment template
├── docker-compose.yml           # Docker Compose config
├── Dockerfile                   # Production image
├── Makefile                     # Build automation
├── go.mod                       # Go dependencies
├── go.sum                       # Dependency checksums
├── README.md                    # Project overview
├── SECURITY.md                  # Security guidelines
├── CONFIGURATION.md             # Config reference
└── SETUP.md                     # This file
```

### Key Directories

**cmd/api/** - Application entry point and initialization

**internal/** - Private application code (not importable by other projects)

**pkg/** - Public utility packages (importable)

**deployments/** - Deployment configurations (Docker, K8s, Helm)

**docs/** - API documentation and guides

---

## Development Workflow

### Daily Development

```bash
# 1. Pull latest changes
git pull origin main

# 2. Start services
docker-compose up -d postgres redis

# 3. Run application with hot reload
air

# 4. Make changes to code

# 5. Test changes
go test ./...

# 6. Commit changes
git add .
git commit -m "feat: add new feature"
git push origin feature-branch
```

### Creating a New Feature

```bash
# 1. Create feature branch
git checkout -b feature/user-permissions

# 2. Implement feature
# - Add models (internal/models/)
# - Add repository methods (internal/repository/)
# - Add service logic (internal/service/)
# - Add handlers (internal/handler/)
# - Add routes (internal/server/routes.go)

# 3. Write tests
# - Unit tests for services
# - Integration tests for handlers

# 4. Test locally
go test ./...
go run cmd/api/main.go

# 5. Commit and push
git add .
git commit -m "feat: implement user permissions"
git push origin feature/user-permissions

# 6. Create pull request
```

### Code Style

**Follow Go conventions:**
```bash
# Format code
go fmt ./...

# Lint code
golangci-lint run

# Vet code
go vet ./...
```

**Pre-commit checklist:**
- [ ] Code formatted with `go fmt`
- [ ] No linter warnings
- [ ] All tests passing
- [ ] Documentation updated
- [ ] Environment variables documented

---

## Testing

### Running Tests

**All tests:**
```bash
go test ./...
```

**With coverage:**
```bash
go test ./... -cover
```

**Verbose output:**
```bash
go test ./... -v
```

**Specific package:**
```bash
go test ./internal/service/...
```

**Single test:**
```bash
go test ./internal/service -run TestAuthService_Register
```

### Test Structure

```go
// Example: internal/service/auth_service_test.go
package service

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/mock"
)

// Mock repository
type MockUserRepository struct {
    mock.Mock
}

func (m *MockUserRepository) Create(user *models.User) error {
    args := m.Called(user)
    return args.Error(0)
}

// Test function
func TestAuthService_Register(t *testing.T) {
    // Arrange
    mockRepo := new(MockUserRepository)
    mockRepo.On("Create", mock.Anything).Return(nil)
    
    // Act
    err := mockRepo.Create(&models.User{})
    
    // Assert
    assert.NoError(t, err)
    mockRepo.AssertExpectations(t)
}
```

### Writing Tests

**Unit Tests:**
- Test individual functions/methods
- Mock external dependencies
- Fast execution (<100ms per test)

**Integration Tests:**
- Test with real database (testcontainers)
- Test API endpoints end-to-end
- Slower execution (ok for CI/CD)

**Coverage Goals:**
- Services: 80%+
- Handlers: 70%+
- Repositories: 60%+
- Overall: 70%+

---

## Building & Deployment

### Local Build

```bash
# Build for current platform
go build -o bin/app cmd/api/main.go

# Build for Linux
GOOS=linux GOARCH=amd64 go build -o bin/app-linux cmd/api/main.go

# Build for Windows
GOOS=windows GOARCH=amd64 go build -o bin/app.exe cmd/api/main.go
```

### Docker Build

```bash
# Build image
docker build -t license-management-api:latest .

# Run container
docker run -p 8080:8080 --env-file .env license-management-api:latest

# Push to registry
docker tag license-management-api:latest registry.example.com/license-management-api:latest
docker push registry.example.com/license-management-api:latest
```

### Kubernetes Deployment

```bash
# Apply manifests
kubectl apply -f deployments/kubernetes/

# Check deployment
kubectl get pods -l app=license-mgmt-api

# View logs
kubectl logs -f deployment/license-mgmt-api

# Port forward for testing
kubectl port-forward svc/license-mgmt-api 8080:8080
```

### Helm Deployment

```bash
# Install chart
helm install license-mgmt deployments/helm/license-mgmt-chart \
  --set image.tag=latest \
  --set jwt.secret=$(openssl rand -base64 32)

# Upgrade release
helm upgrade license-mgmt deployments/helm/license-mgmt-chart

# Uninstall
helm uninstall license-mgmt
```

---

## Troubleshooting

### Common Issues

#### Database Connection Failed

**Problem:**
```
Error: failed to connect to database
```

**Solutions:**
```bash
# 1. Check PostgreSQL is running
docker-compose ps postgres

# 2. Verify connection string
echo $DATABASE_URL

# 3. Test connection manually
psql $DATABASE_URL

# 4. Check logs
docker-compose logs postgres

# 5. Reset database
docker-compose down -v
docker-compose up -d postgres
```

#### Redis Connection Failed

**Problem:**
```
Error: failed to connect to Redis
```

**Solutions:**
```bash
# 1. Check Redis is running
docker-compose ps redis

# 2. Test connection
redis-cli -h localhost -p 6379 ping

# 3. Check logs
docker-compose logs redis

# 4. Restart Redis
docker-compose restart redis
```

#### JWT Secret Error

**Problem:**
```
PANIC: JWT_SECRET must be at least 32 characters long
```

**Solution:**
```bash
# Generate secrets
openssl rand -base64 32

# Add to .env
JWT_SECRET=<generated-secret>
JWT_REFRESH_SECRET=<another-generated-secret>
```

#### Port Already in Use

**Problem:**
```
Error: bind: address already in use
```

**Solutions:**
```bash
# Find process using port
lsof -i :8080  # macOS/Linux
netstat -ano | findstr :8080  # Windows

# Kill process
kill -9 <PID>

# Or change port in .env
PORT=8081
```

#### Module Errors

**Problem:**
```
Error: package not found
```

**Solutions:**
```bash
# Download dependencies
go mod download

# Tidy modules
go mod tidy

# Vendor dependencies (optional)
go mod vendor

# Clear cache
go clean -modcache
```

### Debug Mode

**Enable verbose logging:**
```bash
# .env
LOG_LEVEL=debug
ENVIRONMENT=development
```

**Use debugger (delve):**
```bash
# Install delve
go install github.com/go-delve/delve/cmd/dlv@latest

# Start debugger
dlv debug cmd/api/main.go

# Set breakpoint
(dlv) break main.main
(dlv) continue
```

### Getting Help

**Resources:**
- 📚 [Project README](README.md)
- 🔒 [Security Guide](SECURITY.md)
- ⚙️ [Configuration Reference](CONFIGURATION.md)
- 🐛 [Issue Tracker](https://github.com/your-org/license-management-api/issues)

**Contact:**
- 💬 Slack: #api-dev
- 📧 Email: support@example.com

---

## Contributing

### Pull Request Process

1. **Fork & Branch**
   ```bash
   git checkout -b feature/amazing-feature
   ```

2. **Code & Test**
   ```bash
   # Write code
   # Add tests
   go test ./...
   ```

3. **Format & Lint**
   ```bash
   go fmt ./...
   golangci-lint run
   ```

4. **Commit**
   ```bash
   git commit -m "feat: add amazing feature"
   ```

5. **Push & PR**
   ```bash
   git push origin feature/amazing-feature
   # Create PR on GitHub
   ```

### Commit Message Convention

```
<type>(<scope>): <subject>

<body>

<footer>
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation
- `style`: Code style (formatting)
- `refactor`: Code refactoring
- `test`: Adding tests
- `chore`: Maintenance

**Examples:**
```
feat(auth): add password reset functionality

Implement password reset flow with email verification.
Tokens expire after 1 hour.

Closes #123
```

### Code Review Checklist

**Before requesting review:**
- [ ] Code follows Go style guidelines
- [ ] All tests pass
- [ ] No linter warnings
- [ ] Documentation updated
- [ ] Commit messages are clear
- [ ] PR description explains changes

**Reviewers check:**
- [ ] Code is maintainable and readable
- [ ] Tests adequately cover changes
- [ ] No security vulnerabilities
- [ ] Performance impact considered
- [ ] Documentation is accurate

---

## Development Tools

### Recommended VS Code Extensions

```json
{
  "recommendations": [
    "golang.go",
    "ms-azuretools.vscode-docker",
    "humao.rest-client",
    "eamodio.gitlens",
    "streetsidesoftware.code-spell-checker"
  ]
}
```

### Useful Commands

```bash
# Database migrations
make migrate-up
make migrate-down

# Generate mocks
go generate ./...

# Update dependencies
go get -u ./...
go mod tidy

# Security scan
gosec ./...
go list -json -m all | nancy sleuth

# Benchmark
go test -bench=. ./...
```

### API Testing

**Example requests in `api.http`:**
```http
### Register
POST http://localhost:8080/api/auth/register
Content-Type: application/json

{
  "username": "testuser",
  "email": "test@example.com",
  "password": "password123"
}

### Login
POST http://localhost:8080/api/auth/login
Content-Type: application/json

{
  "username": "testuser",
  "password": "password123"
}

### Get Profile
GET http://localhost:8080/api/auth/me
Authorization: Bearer {{access_token}}
```

---

## Next Steps

✅ **You're all set up!** Here's what to do next:

1. **Explore the Codebase**
   - Read through [README.md](README.md)
   - Review the API documentation
   - Understand the project structure

2. **Make Your First Contribution**
   - Check [open issues](https://github.com/your-org/license-management-api/issues)
   - Pick a "good first issue"
   - Follow the contribution guidelines

3. **Learn More**
   - [Go Best Practices](https://go.dev/doc/effective_go)
   - [REST API Design](https://restfulapi.net/)
   - [PostgreSQL Documentation](https://www.postgresql.org/docs/)
   - [Redis Documentation](https://redis.io/docs/)

---

**Happy Coding! 🚀**

---

**Document Maintained By**: Development Team  
**Last Updated**: March 4, 2026  
**Questions?** support@example.com
