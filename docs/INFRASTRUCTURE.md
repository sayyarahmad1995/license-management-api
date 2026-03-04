# Infrastructure & Scaling Guide

## Table of Contents
1. [Kubernetes Deployment](#kubernetes-deployment)
2. [Helm Charts](#helm-charts)
3. [Scaling Strategies](#scaling-strategies)
4. [High Availability Setup](#high-availability-setup)
5. [Backup & Recovery](#backup--recovery)
6. [Monitoring & Observability](#monitoring--observability)

---

## Kubernetes Deployment

### Prerequisites
- Kubernetes cluster (1.24+)
- kubectl configured
- Docker images built and pushed to registry
- PostgreSQL and Redis available (or use Helm to deploy)

### Quick Start

Deploy using raw manifests:
```bash
# Create namespace
kubectl apply -f k8s/namespace.yaml

# Create secrets (update values first!)
kubectl apply -f k8s/secrets.yaml

# Create ConfigMap
kubectl apply -f k8s/configmap.yaml

# Deploy application
kubectl apply -f k8s/deployment.yaml

# Create services
kubectl apply -f k8s/service.yaml

# Apply network policies
kubectl apply -f k8s/network-policy.yaml

# Configure HPA
kubectl apply -f k8s/hpa.yaml

# Configure PDB
kubectl apply -f k8s/pdb.yaml
```

### Manifest Files Overview

#### `namespace.yaml`
- Creates isolated namespace `license_mgmt` for all resources
- Good practice for resource organization and RBAC

#### `secrets.yaml` 
- Stores sensitive data (credentials, API keys)
- **CRITICAL**: Update before deploying to production
- Should use external secret management (Vault, Sealed Secrets)

#### `configmap.yaml`
- Non-sensitive configuration
- 15+ environment variables for application settings

#### `deployment.yaml`
- 3 replicas by default (adjust as needed)
- Rolling update strategy (maxSurge: 1, maxUnavailable: 0)
- Pod anti-affinity spreads replicas across nodes
- Security context: read-only filesystem, non-root user
- Health checks: liveness, readiness, startup probes
- Resource limits: 512Mi memory, 500m CPU per pod

#### `service.yaml`
- ClusterIP service for internal communication
- LoadBalancer service for external access (optional)
- Port 8080 HTTP

#### `network-policy.yaml`
- Restricts ingress from ingress-nginx only
- Restricts egress to DNS, PostgreSQL, Redis only
- Enhance security posture

#### `hpa.yaml`
- Scales from 3 to 10 replicas
- Based on CPU (70%) and memory (80%) utilization
- Aggressive scale-up (100% within 15s), gradual scale-down

#### `pdb.yaml`
- Ensures minimum 2 pods available during disruptions
- Prevents complete downtime during node maintenance

---

## Helm Charts

### Structure
```
helm/license-mgmt/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default values
├── NOTES.txt              # Post-install notes
└── templates/
    ├── _helpers.tpl       # Reusable template helpers
    ├── deployment.yaml    # Deployment template
    ├── service.yaml       # Service template
    ├── configmap.yaml     # ConfigMap template
    ├── secrets.yaml       # Secrets template
    ├── serviceaccount.yaml# ServiceAccount template
    ├── hpa.yaml          # HPA template
    └── pdb.yaml          # PDB template
```

### Installation

**Add repository** (if published):
```bash
helm repo add license_mgmt https://your-registry.com/charts
helm repo update
```

**Install from local chart**:
```bash
helm install license_mgmt ./helm/license-mgmt \
  -n license_mgmt \
  --create-namespace \
  -f helm/license-mgmt/values.yaml
```

**Install with custom values**:
```bash
helm install license_mgmt ./helm/license-mgmt \
  -n license_mgmt \
  --create-namespace \
  -f helm/license-mgmt/values.yaml \
  -f custom-values.yaml \
  --set secrets.dbPassword=YOUR_SECURE_PASSWORD \
  --set secrets.jwtSecretKey=YOUR_SECRET_KEY
```

### Customization

Edit `values.yaml` to customize:
- Replica count
- Resource limits/requests
- Autoscaling thresholds
- Health check parameters
- Ingress configuration
- Database/Redis settings

Example custom values:
```yaml
# custom-values.yaml
replicaCount: 5

resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 500m
    memory: 512Mi

autoscaling:
  enabled: true
  minReplicas: 5
  maxReplicas: 20
  targetCPUUtilizationPercentage: 60

ingress:
  enabled: true
  className: nginx
  hosts:
    - host: api.yourdomain.com
      paths:
        - path: /
          pathType: Prefix

secrets:
  dbPassword: "your-secure-password"
  jwtSecretKey: "your-long-secret-key-min-32-chars"
```

### Common Helm Commands

```bash
# Check values
helm values license_mgmt

# Dry run
helm install license_mgmt ./helm/license-mgmt --dry-run --debug

# Upgrade release
helm upgrade license_mgmt ./helm/license-mgmt -f values.yaml

# Rollback to previous version
helm rollback license_mgmt 1

# Uninstall
helm uninstall license_mgmt -n license_mgmt

# Release history
helm history license_mgmt
```

---

## Scaling Strategies

### Horizontal Pod Autoscaling (HPA)

Currently configured metrics:
- **CPU**: Scale when utilization > 70%
- **Memory**: Scale when utilization > 80%

**Behavior**:
- Scale up: 100% increase every 15 seconds (max)
- Scale down: 50% decrease every 60 seconds

**Adjust thresholds**:
```bash
kubectl patch hpa license_mgmt-api-hpa -p '{"spec":{"minReplicas":5,"maxReplicas":20}}'
```

### Vertical Pod Autoscaling (Optional)

For automatic resource recommendation:
```bash
# Install VPA
helm repo add autoscaling https://kubernetes.github.io/autoscaler
helm install vpa autoscaling/vertical-pod-autoscaler -n kube-system

# Create VPA for license_mgmt
kubectl apply -f - <<EOF
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: license_mgmt-vpa
  namespace: license_mgmt
spec:
  targetRef:
    apiVersion: "apps/v1"
    kind: Deployment
    name: license_mgmt-api
  updatePolicy:
    updateMode: "Auto"
EOF
```

### Database Connection Pooling

Connection pool configuration in `performance.go`:
- **MaxOpenConns**: 100 (adjust based on load)
- **MaxIdleConns**: 10 (connections kept alive)
- **ConnMaxLifetime**: 5 minutes (recycle connections)

Adjust in code:
```go
// For high concurrent load
sqlDB.SetMaxOpenConns(200)

// For higher idle needs
sqlDB.SetMaxIdleConns(20)
```

---

## High Availability Setup

### Multi-Zone Deployment

Edit deployment affinity rules:
```yaml
affinity:
  podAntiAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app
              operator: In
              values:
                - license_mgmt-api
        topologyKey: topology.kubernetes.io/zone
```

### Health Checks Tuning

Adjust probe settings in deployment:

```yaml
livenessProbe:
  initialDelaySeconds: 30    # Wait before first check
  periodSeconds: 10          # Check every 10 seconds
  timeoutSeconds: 5          # Timeout if no response in 5s
  failureThreshold: 3        # Restart after 3 failures

readinessProbe:
  initialDelaySeconds: 10    # Wait before first check
  periodSeconds: 5           # Check every 5 seconds
  timeoutSeconds: 3          # Timeout if no response in 3s
  failureThreshold: 2        # Remove from traffic after 2 failures

startupProbe:
  initialDelaySeconds: 0
  periodSeconds: 5
  timeoutSeconds: 3
  failureThreshold: 30       # Wait up to 2.5 minutes for startup
```

### Load Balancing

Use Kubernetes Service `type: LoadBalancer`:
```yaml
apiVersion: v1
kind: Service
metadata:
  name: license_mgmt-api-lb
spec:
  type: LoadBalancer
  selector:
    app: license_mgmt-api
  ports:
    - port: 80
      targetPort: 8080
```

Or use Ingress for advanced routing:
```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: license_mgmt-ingress
spec:
  rules:
  - host: api.yourdomain.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: license_mgmt-api
            port:
              number: 8080
```

---

## Backup & Recovery

### PostgreSQL Backup

```bash
# Backup database
pg_dump -h postgres.license_mgmt -U license_mgmt -d license_mgmt > backup.sql

# Restore
psql -h postgres.license_mgmt -U license_mgmt -d license_mgmt < backup.sql

# In Kubernetes pod
kubectl exec -it postgres-pod -n license_mgmt -- \
  pg_dump -U license_mgmt license_mgmt > backup.sql
```

### Data Recovery in Kubernetes

```bash
# Create backup before deletion
kubectl exec -it postgres-pod -n license_mgmt -- \
  pg_dump -U license_mgmt license_mgmt > pre-delete-backup.sql

# Restore to new database
kubectl cp pre-delete-backup.sql postgres-pod:/backup.sql -n license_mgmt
kubectl exec -it postgres-pod -n license_mgmt -- \
  psql -U license_mgmt license_mgmt < /backup.sql
```

### Certificate Backup (for TLS)

```bash
# Export certificate
kubectl get secret license_mgmt-tls -n license_mgmt -o yaml > tls-backup.yaml

# Restore certificate
kubectl apply -f tls-backup.yaml
```

---

## Monitoring & Observability

### Prometheus Metrics

Metrics available at `http://localhost:8080/metrics`:
- **HTTP requests**: `http_requests_total`, `http_request_duration_seconds`
- **Database**: `db_query_duration_seconds`, `db_query_errors_total`
- **Cache**: `cache_hits_total`, `cache_misses_total`
- **Authentication**: `auth_failures_total`
- **Active connections**: `db_active_connections`

### Install Prometheus + Grafana

```bash
# Add Prometheus Helm repo
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts

# Install Prometheus
helm install prometheus prometheus-community/kube-prometheus-stack \
  -n prometheus \
  --create-namespace

# Port forward
kubectl port-forward svc/prometheus-operated 9090:9090 -n prometheus
```

### Create Grafana Dashboards

Import community dashboards:
1. Access Grafana: `http://localhost:3000`
2. Create data source pointing to Prometheus
3. Import dashboard JSON from community

Key metrics to monitor:
- CPU/Memory usage
- Request latency (p50, p95, p99)
- Error rates
- Database connection count
- Cache hit ratio

---

## Troubleshooting

### Pod not starting
```bash
kubectl describe pod license_mgmt-api-xxx -n license_mgmt
kubectl logs license_mgmt-api-xxx -n license_mgmt
```

### Readiness probe failing
```bash
# Check health endpoint
kubectl exec license_mgmt-api-xxx -n license_mgmt -- \
  curl -v http://localhost:8080/readyz
```

### Database connection issues
```bash
# Check database connectivity
kubectl run -it --rm debug --image=postgres --restart=Never \
  -- psql -h postgres.license_mgmt -U license_mgmt -d license_mgmt -c "SELECT 1"
```

### High memory usage
```bash
# Check pod metrics
kubectl top pods -n license_mgmt

# Increase resource limits
kubectl patch deployment license_mgmt-api -n license_mgmt \
  -p '{"spec":{"template":{"spec":{"containers":[{"name":"license_mgmt-api","resources":{"limits":{"memory":"1Gi"}}}]}}}}'
```

---

## Production Checklist

- [ ] Update all secrets in `secrets.yaml` or use external secret manager
- [ ] Configure TLS/HTTPS with valid certificates
- [ ] Enable network policies for security
- [ ] Set up Prometheus monitoring
- [ ] Configure log aggregation (ELK, Loki)
- [ ] Set up alerts for high CPU/memory
- [ ] Configure backup strategy
- [ ] Test disaster recovery procedures
- [ ] Document runbooks for common issues
- [ ] Implement circuit breakers for external APIs
- [ ] Set up distributed tracing (Jaeger)
- [ ] Configure rate limiting per client


