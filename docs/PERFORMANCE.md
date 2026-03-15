# Performance Optimization Guide

## Table of Contents
1. [Database Optimization](#database-optimization)
2. [Caching Strategies](#caching-strategies)
3. [Response Compression](#response-compression)
4. [HTTP Caching Headers](#http-caching-headers)
5. [Query Optimization](#query-optimization)
6. [Connection Pooling](#connection-pooling)
7. [Performance Testing & Benchmarking](#performance-testing--benchmarking)
8. [Performance Tuning Checklist](#performance-tuning-checklist)

---

## Database Optimization

### Strategic Database Indexes

The system automatically creates 25+ strategic indexes on startup via `database/performance.go`:

#### Single Column Indexes
- **Users**: email, username, status, created_at
- **Licenses**: license_key, status, user_id, expiry_date, created_at
- **License Activations**: license_id, machine_fingerprint, created_at, last_heartbeat
- **Audit Logs**: user_id, action, resource_type, created_at
- **Email Verification**: token, user_id, expires_at
- **Password Reset**: token, user_id, expires_at

#### Composite Indexes
- `idx_license_user_status`: `licenses(user_id, status)` - Fast user+ status lookups
- `idx_license_status_expiry`: `licenses(status, expiry_date)` - Fast expiry checks
- `idx_activation_license_active`: `license_activations(license_id, last_heartbeat)` - Active license check
- `idx_audit_created_user`: `audit_logs(created_at, user_id)` - Fast audit queries

### Automatic Index Creation

Indexes are created automatically when the application starts:
```go
// In database.New()
err = database.CreateIndexes(db)
```

View created indexes:
```bash
# Connect to PostgreSQL
psql -h localhost -U license_mgmt -d license_mgmt

# List indexes
\di

# Check index usage
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read, idx_tup_fetch
FROM pg_stat_user_indexes
ORDER BY idx_scan DESC;
```

### Slow Query Logging

Enabled automatically at startup:
```go
database.EnableSlowQueryLogging(db)  // Logs queries > 1000ms
```

View slow queries in PostgreSQL logs:
```sql
-- PostgreSQL log file location
SELECT setting FROM pg_settings WHERE name = 'log_directory';

-- Or check syslog
tail -f /var/log/postgresql/postgresql-*.log
```

### Database Maintenance

Run periodic maintenance to optimize performance:
```go
// In scheduled task or admin endpoint
database.VacuumDatabase(db)  // Removes dead tuples
```

Vacuum helps with:
- Reclaiming disk space from deleted rows
- Updating table statistics for query planner
- Preventing table bloat

---

## Caching Strategies

### Two-Level Caching

The system uses **Redis** for distributed caching with automatic TTL:

#### Level 1: Session Cache
```go
// Session tokens and active session lists
sessionService.CacheSession(userID, sessionData)  // TTL: configurable
```

#### Level 2: Permission Cache
```go
// User permissions are cached in memory with TTL
permissionService.GetUserPermissions(userID)  // Automatically cached
```

### Cache Configuration

Edit environment variables:
```bash
CACHE_TTL=3600                    # Cache lifetime in seconds
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=your-password
```

### Cache Invalidation Strategy

When data changes, invalidate relevant caches:
```go
// After updating user permissions
redisService.Delete(ctx, fmt.Sprintf("permissions:user:%d", userID))

// After license update
redisService.Delete(ctx, fmt.Sprintf("license:%d", licenseID))
```

### Manual Cache Operations

```go
// Clear all caches
redis-cli FLUSHALL

// Monitor cache activity
redis-cli MONITOR

// Check memory usage
redis-cli INFO memory

// Set max memory policy
redis-cli CONFIG SET maxmemory-policy "allkeys-lru"
```

---

## Response Compression

### Gzip Compression Middleware

Enabled automatically via `CompressionMiddleware()`:

```go
// In server/routes.go
router.Use(middleware.CompressionMiddleware())
```

**Compression behavior**:
- Only compresses if client sends `Accept-Encoding: gzip`
- Automatically compresses JSON, text, HTML responses
- Removes `Content-Length` header (length changes)
- Sets `Content-Encoding: gzip` header

### Compression Benefits

- **90%+ reduction** for JSON responses (typically 10KB → 1KB)
- **Network bandwidth savings** especially on mobile
- **Minimal CPU overhead** (gzip is fast)

### Test Compression

```bash
# With gzip
curl -i --compressed http://localhost:8080/api/endpoint

# Without gzip
curl -i http://localhost:8080/api/endpoint

# Compare response sizes
curl -s --compressed http://localhost:8080/api/endpoint | wc -c
curl -s http://localhost:8080/api/endpoint | wc -c
```

---

## HTTP Caching Headers

### Cache Control Strategies

Middleware automatically sets appropriate headers based on response type:

#### API Responses (JSON)
```
Cache-Control: public, max-age=300          # 5 minutes for GET requests
Cache-Control: no-cache, no-store           # Disabled for POST/PUT/DELETE
ETag: "timestamp"                            # For conditional requests
```

#### Static Assets
```
Cache-Control: public, max-age=31536000     # 1 year
Immutable: true                              # Never changes
```

#### Error Responses
```
Cache-Control: no-store, no-cache           # Never cache errors
```

#### Admin/Auth Endpoints
```
Cache-Control: no-store, no-cache           # Never cache sensitive data
Pragma: no-cache                            # HTTP/1.0 compatibility
Expires: 0                                  # Immediately expired
```

### Enabling Cache Control

Add middleware to routes:
```go
router.Use(middleware.CacheControlMiddleware())
```

### Browser Cache Validation

Use ETags for conditional requests:
```bash
# First request
curl -i http://localhost:8080/api/licenses

# Save ETag from response header
ETag: "200"

# Conditional request
curl -i -H 'If-None-Match: "200"' http://localhost:8080/api/licenses

# Returns 304 Not Modified (no body sent)
```

---

## Query Optimization

### N+1 Query Prevention

**Problem**: Loading user with licenses causes N queries (1 for user + N for licenses)

**Solution**: Use GORM eager loading:
```go
// ❌ Bad - N+1 query problem
var user User
db.First(&user, userID)
var licenses []License
db.Where("user_id = ?", userID).Find(&licenses)  // Extra query!

// ✅ Good - Single query with join
var user User
db.Preload("Licenses").First(&user, userID)

// ✅ Better - Selective fields
var user User
db.Preload("Licenses", func(db *gorm.DB) *gorm.DB {
    return db.Select("id", "license_key", "status")
}).First(&user, userID)
```

### Query Analysis

Use `EXPLAIN ANALYZE` to optimize slow queries:
```sql
-- Check query plan
EXPLAIN ANALYZE
SELECT * FROM licenses WHERE user_id = $1 AND status = $2;

-- Output shows:
-- - Seq Scan vs Index Scan
-- - Rows estimated vs actual
-- - Execution time
```

Example output:
```
Index Scan using idx_license_user_status on licenses  (cost=0.42..8.44 rows=1 width=200)
  Index Cond: ((user_id = 1) AND (status = 'ACTIVE'))
  Planning Time: 0.123 ms
  Actual Time: 0.456 ms
```

### Common Optimization Patterns

```go
// ✅ Use database limits for pagination
var licenses []License
db.Where("user_id = ?", userID).
   Limit(pageSize).
   Offset((page - 1) * pageSize).
   Find(&licenses)

// ✅ Select only needed columns
db.Select("id", "license_key", "status", "created_at").
   Where("status = ?", "ACTIVE").
   Find(&licenses)

// ✅ Batch operations
var users []User
batchSize := 1000
for i := 0; i < len(userIDs); i += batchSize {
    batch := userIDs[i:min(i+batchSize, len(userIDs))]
    db.Where("id IN ?", batch).Find(&users)
}
```

---

## Connection Pooling

### Configuration

Automatically configured in `database/performance.go`:

```go
// Connection pool settings
sqlDB.SetMaxOpenConns(100)     // Max concurrent connections
sqlDB.SetMaxIdleConns(10)      // Idle connections to keep
sqlDB.SetConnMaxLifetime(5 * 60) // Seconds - recycle old connections
```

### Monitoring Connection Pool

```bash
# Check active connections
SELECT count(*) FROM pg_stat_activity WHERE datname = 'license_mgmt';

# Monitor in real-time
watch -n 1 "psql -U license_mgmt -d license_mgmt -c \
  'SELECT count(*) as active_connections FROM pg_stat_activity WHERE datname = \"license_mgmt\"'"
```

### Tuning for Your Workload

Adjust based on concurrent users:
```go
// For 100 concurrent users
sqlDB.SetMaxOpenConns(100)
sqlDB.SetMaxIdleConns(20)

// For 1000 concurrent users
sqlDB.SetMaxOpenConns(1000)
sqlDB.SetMaxIdleConns(100)
```

Factors to consider:
- **Concurrent users**: 1 connection per user
- **Request duration**: Longer requests need more connections
- **Database resources**: PostgreSQL has limits too
- **Memory**: Each connection uses ~5-10MB

---

## Performance Testing & Benchmarking

### Load Testing with Apache Bench

```bash
# Install
choco install apache-benchmark  # Windows
apt-get install apache2-utils   # Linux

# Simple load test
ab -n 1000 -c 10 http://localhost:8080/health

# Output:
# Requests per second:    500 [#/sec]
# Time per request:       20 ms
# Failed requests:        0
```

### Load Testing with wrk

```bash
# Install wrk
git clone https://github.com/wg/wrk.git
cd wrk && make

# Run test: 4 threads, 100 connections, 30 seconds
./wrk -t4 -c100 -d30s http://localhost:8080/api/licenses

# Output shows latency percentiles (p50, p99, max)
```

### Load Testing with k6

```javascript
// load-test.js
import http from 'k6/http';
import { check } from 'k6';

export default function() {
  let response = http.get('http://localhost:8080/api/licenses');
  check(response, {
    'status is 200': (r) => r.status === 200,
    'response time < 500ms': (r) => r.timings.duration < 500,
  });
}

export const options = {
  stages: [
    { duration: '30s', target: 20 },
    { duration: '1m30s', target: 50 },
    { duration: '30s', target: 0 },
  ],
};
```

Run:
```bash
k6 run load-test.js
```

### Baseline Metrics to Track

Establish baseline performance:
```bash
# Response time: target < 200ms
# Throughput: > 100 requests/sec
# Error rate: < 0.1%
# P99 latency: < 500ms

# Record baseline
ab -n 10000 -c 50 http://localhost:8080/api/licenses > baseline.txt
```

---

## Performance Tuning Checklist

### Immediate Gains (Do First)
- [ ] Enable database indexes (`CreateIndexes()` called on startup)
- [ ] Enable gzip compression (`CompressionMiddleware()`)
- [ ] Enable HTTP caching headers (`CacheControlMiddleware()`)
- [ ] Configure connection pooling (`ConfigureConnectionPool()`)
- [ ] Enable slow query logging (`EnableSlowQueryLogging()`)

### Short-term (Week 1)
- [ ] Analyze slow queries with `EXPLAIN ANALYZE`
- [ ] Add missing indexes for slow queries
- [ ] Implement Redis caching for frequently accessed data
- [ ] Set up load testing to establish baselines
- [ ] Profile application with pprof

### Medium-term (Week 2-3)
- [ ] Optimize N+1 queries with eager loading
- [ ] Implement pagination for large result sets
- [ ] Add response compression for large payloads
- [ ] Set up Prometheus metrics monitoring
- [ ] Configure horizontal autoscaling

### Long-term (Month 2+)
- [ ] Database read replicas for reporting queries
- [ ] Full-text search indexing for audit logs
- [ ] Service mesh for distributed tracing
- [ ] CDN for static asset delivery
- [ ] Database sharding if needed

### Monitoring Metrics

Track these KPIs:
```
P50 Latency:   < 100ms
P95 Latency:   < 200ms
P99 Latency:   < 500ms
Error Rate:    < 0.1%
Throughput:    > 100 req/sec
Memory/Pod:    < 512Mi
CPU/Pod:       < 500m
Cache Hit:     > 80%
```

---

## Example Performance Report

**Before Optimization**:
- Average latency: 450ms
- P99 latency: 2000ms
- Error rate: 2%
- Memory: 800Mi/pod

**After Optimization**:
- Average latency: 120ms (73% improvement ✨)
- P99 latency: 350ms (82% improvement ✨)
- Error rate: 0.01% (99% improvement ✨)
- Memory: 280Mi/pod (65% improvement ✨)
- Throughput: 500+ req/sec (4x improvement ✨)

**Changes Made**:
1. Added 25+ database indexes
2. Enabled query result caching (5min TTL)
3. Fixed N+1 query problems (5 queries → 1 query)
4. Enabled gzip compression (90% reduction)
5. Configured connection pooling (20 idle connections)
6. Implement query pagination (5000 → 100 rows default)


