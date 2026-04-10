# PostgreSQL + Collector Deployment Guide

> Complete Kubernetes deployment with stateful PostgreSQL and Collector service connected together.

## Overview

This deployment provides:

- **PostgreSQL StatefulSet** with persistent storage and automatic initialization
- **Collector Deployment** with 3 replicas connected to PostgreSQL
- **Kubernetes-native components**: Services, ConfigMaps, Secrets, RBAC, NetworkPolicy
- **High availability features**: Pod anti-affinity, HPA, PodDisruptionBudget
- **Security**: Resource limits, security contexts, network policies
- **Monitoring**: Health checks, liveness/readiness probes

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────┐
│                    Kubernetes Cluster                        │
│                                                               │
│  ┌──────────────────────────────────────────────────────┐  │
│  │            telemetry Namespace                       │  │
│  │                                                      │  │
│  │  ┌──────────────────┐        ┌─────────────────┐   │  │
│  │  │  PostgreSQL      │        │   Collector     │   │  │
│  │  │  StatefulSet     │◄───────│   Deployment    │   │  │
│  │  │                  │        │   (3 replicas)  │   │  │
│  │  │  • telemetry tbl │        │                 │   │  │
│  │  │  • Persistent    │        └─────────────────┘   │  │
│  │  │    Storage (10Gi)│                               │  │
│  │  └──────────────────┘        ┌─────────────────┐   │  │
│  │         ▲                    │   HPA (3-10)    │   │  │
│  │         │                    └─────────────────┘   │  │
│  │    Service                                          │  │
│  │   (postgres)                                        │  │
│  └──────────────────────────────────────────────────────┘  │
│                                                               │
└─────────────────────────────────────────────────────────────┘
                          │
                    ┌─────▼──────┐
                    │  MQ Topic   │
                    │(telemetry)  │
                    └─────────────┘
```

## File Organization

```
k8s/
├── postgres-deployment.yaml      # Complete deployment (THIS FILE)
│   ├── Namespace: telemetry
│   ├── Secret: postgres-secret
│   ├── ConfigMap: postgres-init (schema + indexes)
│   ├── PVC: postgres-pvc
│   ├── Service: postgres (headless)
│   ├── StatefulSet: postgres (PostgreSQL 15-alpine)
│   ├── Service: collector
│   ├── Deployment: collector (3 replicas)
│   ├── ServiceAccount: collector
│   ├── Role + RoleBinding: RBAC for collector
│   ├── HPA: Auto-scaling (3-10 replicas)
│   ├── NetworkPolicy: Network isolation
│   └── PodDisruptionBudget: Availability SLA
├── deployment.yaml               # Original Collector deployment (kept for reference)
└── configmap.yaml               # (optional) If using separate ConfigMap
```

## Quick Start

### Deploy Everything

```bash
# 1. Create and switch to telemetry namespace
kubectl create namespace telemetry
kubectl config set-context --current --namespace=telemetry

# 2. Apply deployment (creates PostgreSQL + Collector)
kubectl apply -f k8s/postgres-deployment.yaml

# 3. Verify resources
kubectl get all -n telemetry
kubectl get pvc -n telemetry
kubectl get secrets -n telemetry

# 4. Wait for PostgreSQL to be ready
kubectl wait --for=condition=Ready pod -l app=postgres -n telemetry --timeout=300s

# 5. Check Collector pods
kubectl get pods -l app=collector -n telemetry
kubectl logs -l app=collector -n telemetry -f
```

### Verify PostgreSQL is Connected

```bash
# Connect to PostgreSQL pod
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry

# Inside psql:
\dt                    # List tables (should show telemetry)
\d telemetry          # Show telemetry table structure
SELECT COUNT(*) FROM telemetry;  # Check data count
\q                    # Exit
```

### Verify Collector is Running

```bash
# Check pod status
kubectl describe pod <collector-pod-name> -n telemetry

# View logs
kubectl logs <collector-pod-name> -n telemetry

# Port-forward to access (optional)
kubectl port-forward svc/collector 8081:8081 -n telemetry
```

---

## Component Details

### 1. Namespace

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: telemetry
```

Isolates all telemetry components in a dedicated namespace.

### 2. Secret (PostgreSQL Credentials)

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: postgres-secret
  namespace: telemetry
stringData:
  POSTGRES_USER: postgres
  POSTGRES_PASSWORD: telemetry-secure-password-change-in-production
  POSTGRES_DB: telemetry
```

**⚠️ IMPORTANT**: Change the password in production!

```bash
# Generate a secure password
openssl rand -base64 32

# Create secret with secure password
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_USER=postgres \
  --from-literal=POSTGRES_PASSWORD=$(openssl rand -base64 32) \
  --from-literal=POSTGRES_DB=telemetry \
  -n telemetry
```

### 3. ConfigMap (PostgreSQL Initialization)

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: postgres-init
  namespace: telemetry
data:
  init.sql: |
    CREATE TABLE IF NOT EXISTS telemetry (...)
    CREATE UNIQUE INDEX idx_telemetry_gpu_ts_unique ...
```

Automatically initializes the database schema on first run.

### 4. PersistentVolumeClaim

```yaml
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: postgres-pvc
  namespace: telemetry
spec:
  resources:
    requests:
      storage: 10Gi
  storageClassName: standard
```

**Storage Size**: 10Gi (change based on your needs)

For Kind clusters, use `standard` storage class. For cloud providers:
- AWS EKS: `gp2`, `gp3`, `io1`
- GCP GKE: `standard`, `premium-rwo`
- Azure AKS: `default`, `managed-premium`

### 5. PostgreSQL Service (Headless)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: postgres
  namespace: telemetry
spec:
  type: ClusterIP
  clusterIP: None  # Headless for StatefulSet
  ports:
  - port: 5432
    targetPort: 5432
```

Headless service provides stable DNS names for StatefulSet pods:
- `postgres-0.postgres.telemetry.svc.cluster.local:5432`

### 6. PostgreSQL StatefulSet

**Key Features**:

- **Image**: `postgres:15-alpine` (lightweight, ~100MB)
- **Replicas**: 1 (single instance) - can be increased for HA with replication
- **Health Checks**:
  - Liveness: `pg_isready` every 10s
  - Readiness: `pg_isready` every 5s
- **Resource Limits**:
  - Requests: 256Mi RAM, 250m CPU
  - Limits: 512Mi RAM, 500m CPU
- **Security**: Non-root user (999), read-only root filesystem option available
- **Initialization**: Auto-creates telemetry table with indexes

**Database Configuration** (configurable):
```
shared_buffers=256MB
max_connections=200
effective_cache_size=1GB
```

**Verify PostgreSQL**:
```bash
# Check StatefulSet
kubectl get statefulset -n telemetry

# Check pod
kubectl get pods -l app=postgres -n telemetry

# Check PVC
kubectl get pvc -n telemetry

# Check logs
kubectl logs postgres-0 -n telemetry
```

### 7. Collector Service

```yaml
apiVersion: v1
kind: Service
metadata:
  name: collector
  namespace: telemetry
spec:
  type: ClusterIP
  ports:
  - port: 8081
    targetPort: 8081
    name: http
  selector:
    app: collector
```

Exposes Collector on port 8081 (ClusterIP only - internal access).

### 8. Collector Deployment

**Configuration**:
- **Replicas**: 3 (managed by HPA, can scale to 10)
- **Image**: `collector:latest`
- **Strategy**: RollingUpdate with maxSurge=1, maxUnavailable=0

**Environment Variables**:
```bash
MQ_URL=http://mq-service:8080
TOPIC=telemetry
GROUP=collector-group
PARTITIONS=3
BATCH_SIZE=10
POLL_INTERVAL_MS=500
DB_DSN=postgres://postgres:password@postgres.telemetry.svc.cluster.local:5432/telemetry?sslmode=disable
```

**Health Checks**:
- Liveness: HTTP GET `/health` on port 8081
- Readiness: HTTP GET `/ready` on port 8081

**Init Container**: Waits for PostgreSQL to be ready before starting Collector

**Resource Limits**:
- Requests: 128Mi RAM, 100m CPU
- Limits: 256Mi RAM, 500m CPU

**Security**:
- Non-root user (1000)
- Read-only root filesystem
- No privilege escalation
- Dropped all capabilities

**Affinity**:
- Pod anti-affinity prefers spreading replicas across different nodes

### 9. ServiceAccount & RBAC

```yaml
serviceAccountName: collector
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: collector
rules:
- apiGroups: [""]
  resources: ["pods"]
  verbs: ["get", "list", "watch"]
- apiGroups: [""]
  resources: ["configmaps"]
  verbs: ["get", "list"]
```

Grants minimal permissions for Collector to read pods and configs.

### 10. HorizontalPodAutoscaler

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: collector-hpa
spec:
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        averageUtilization: 80
```

**Scaling Logic**:
- Scale up when CPU > 70% or Memory > 80%
- Scale down when CPU < 70% and Memory < 80%
- Min pods: 3 (ensures availability)
- Max pods: 10 (prevents runaway scaling)

### 11. PodDisruptionBudget

```yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: collector-pdb
spec:
  minAvailable: 2
```

Ensures at least 2 Collector pods are always running (for cluster maintenance).

### 12. NetworkPolicy

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: telemetry-network-policy
spec:
  policyTypes:
  - Ingress
  - Egress
```

Restricts network traffic within the telemetry namespace.

---

## Deployment Instructions

### Local Kind Cluster

```bash
# 1. Create kind cluster (if not exists)
kind create cluster --name telemetry

# 2. Load collector image
docker build -t collector:latest ../..
kind load docker-image collector:latest --name telemetry

# 3. Apply deployment
kubectl apply -f postgres-deployment.yaml

# 4. Monitor
kubectl get pods -n telemetry -w
kubectl logs -l app=collector -n telemetry -f
```

### Production Cluster (AWS/GCP/Azure)

```bash
# 1. Update storage class
# Edit postgres-deployment.yaml and change storageClassName
sed -i 's/storageClassName: standard/storageClassName: gp3/g' postgres-deployment.yaml

# 2. Update PostgreSQL password
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_USER=postgres \
  --from-literal=POSTGRES_PASSWORD=$(openssl rand -base64 32) \
  --from-literal=POSTGRES_DB=telemetry \
  -n telemetry

# 3. Push collector image to registry
docker build -t your-registry/collector:latest ../..
docker push your-registry/collector:latest

# Edit image in postgres-deployment.yaml:
sed -i 's|image: collector:latest|image: your-registry/collector:latest|g' postgres-deployment.yaml

# 4. Apply deployment
kubectl apply -f postgres-deployment.yaml

# 5. Verify
kubectl get all -n telemetry
kubectl get pvc -n telemetry
```

---

## Verification Steps

### Check All Resources

```bash
# Namespace
kubectl get namespace telemetry

# Secrets
kubectl get secrets -n telemetry

# PVC
kubectl get pvc -n telemetry
kubectl describe pvc postgres-pvc -n telemetry

# Services
kubectl get svc -n telemetry
kubectl describe svc postgres -n telemetry
kubectl describe svc collector -n telemetry

# StatefulSet
kubectl get statefulset -n telemetry
kubectl describe statefulset postgres -n telemetry

# Pods
kubectl get pods -n telemetry -o wide
kubectl get pods -l app=postgres -n telemetry
kubectl get pods -l app=collector -n telemetry

# Deployment
kubectl get deployment -n telemetry
kubectl describe deployment collector -n telemetry

# HPA
kubectl get hpa -n telemetry
kubectl describe hpa collector-hpa -n telemetry

# PDB
kubectl get pdb -n telemetry

# Events
kubectl get events -n telemetry --sort-by='.lastTimestamp'
```

### Test PostgreSQL Connection

```bash
# Execute psql inside PostgreSQL pod
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry

# Inside psql:
postgres=# \dt
                 List of relations
 Schema |    Name    | Type  |     Owner
--------+------------+-------+---------------
 public | telemetry  | table | postgres
(1 row)

postgres=# \d telemetry
                           Table "public.telemetry"
     Column     |           Type           | Collation | Nullable | Default
----------------+--------------------------+-----------+----------+---------
 id             | integer                  |           | not null | nextval(...)
 gpu_id         | text                     |           | not null |
 timestamp      | timestamp with time zone |           | not null |
 data           | jsonb                    |           | not null |
 created_at     | timestamp with time zone |           |          | CURRENT_TIMESTAMP

Indexes:
    "telemetry_pkey" PRIMARY KEY, btree (id)
    "idx_telemetry_created_at" btree (created_at)
    "idx_telemetry_gpu_id" btree (gpu_id)
    "idx_telemetry_gpu_timestamp" btree (gpu_id, timestamp DESC)
    "idx_telemetry_gpu_ts_unique" UNIQUE, btree (gpu_id, timestamp)
    "idx_telemetry_timestamp" btree (timestamp)

postgres=# SELECT COUNT(*) FROM telemetry;
 count
-------
     0
(1 row)

postgres=# \q
```

### Test Collector Connection

```bash
# Check Collector pod logs
kubectl logs <collector-pod-name> -n telemetry

# Expected output:
# collector: starting for topic=telemetry group=collector-group partitions=3
# collector: consume partition=0 ...
# collector: consume partition=1 ...
# collector: consume partition=2 ...

# Describe pod for errors
kubectl describe pod <collector-pod-name> -n telemetry

# Check events for startup issues
kubectl get events -n telemetry --field-selector involvedObject.name=<collector-pod-name>
```

### Test Data Flow

```bash
# 1. Send telemetry message to MQ (from external)
curl -X POST http://localhost:8080/produce \
  -H "Content-Type: application/json" \
  -d '{
    "topic": "telemetry",
    "partition": 0,
    "value": {
      "gpu_id": "gpu-001",
      "timestamp": "2026-04-10T15:30:45Z",
      "utilization": 85.5,
      "memory": 24576,
      "temperature": 72,
      "power": 250
    }
  }'

# 2. Wait a few seconds for Collector to process

# 3. Check if data is in PostgreSQL
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry -c "SELECT COUNT(*) FROM telemetry;"

# 4. Query the data
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry -c "SELECT * FROM telemetry LIMIT 1;"
```

---

## Monitoring

### Pod Status

```bash
# Watch pods
kubectl get pods -n telemetry -w

# Check specific pod
kubectl describe pod <pod-name> -n telemetry

# View events
kubectl get events -n telemetry --sort-by='.lastTimestamp'
```

### Logs

```bash
# PostgreSQL logs
kubectl logs postgres-0 -n telemetry

# Collector logs (current pod)
kubectl logs -l app=collector -n telemetry --tail=100

# Collector logs (all pods, follow)
kubectl logs -l app=collector -n telemetry -f

# Previous logs (if pod restarted)
kubectl logs <pod-name> -n telemetry --previous
```

### Metrics

```bash
# Pod resource usage
kubectl top pods -n telemetry

# Node resource usage
kubectl top nodes

# HPA status
kubectl get hpa collector-hpa -n telemetry
kubectl describe hpa collector-hpa -n telemetry
```

### Database Monitoring

```bash
# Connect to PostgreSQL
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry

# Check active connections
SELECT * FROM pg_stat_activity;

# Check table stats
SELECT schemaname, tablename, seq_scan, seq_tup_read, idx_scan
FROM pg_stat_user_tables;

# Check index efficiency
SELECT schemaname, tablename, indexname, idx_scan, idx_tup_read
FROM pg_stat_user_indexes;

# Check disk usage
SELECT pg_size_pretty(pg_total_relation_size('telemetry'));

# Exit
\q
```

---

## Troubleshooting

### PostgreSQL Pod Won't Start

```bash
# 1. Check pod events
kubectl describe pod postgres-0 -n telemetry

# 2. Check logs
kubectl logs postgres-0 -n telemetry

# 3. Check PVC
kubectl get pvc -n telemetry
kubectl describe pvc postgres-data-postgres-0 -n telemetry

# 4. Check storage availability
kubectl get storageclass
kubectl get pv

# 5. Delete and recreate
kubectl delete statefulset postgres -n telemetry
kubectl apply -f postgres-deployment.yaml
```

### Collector Pod Can't Connect to PostgreSQL

```bash
# 1. Check DNS resolution
kubectl exec -it <collector-pod> -n telemetry -- nslookup postgres.telemetry.svc.cluster.local

# 2. Check PostgreSQL service
kubectl get svc postgres -n telemetry

# 3. Check network connectivity
kubectl exec -it <collector-pod> -n telemetry -- nc -zv postgres.telemetry.svc.cluster.local 5432

# 4. Check environment variables
kubectl exec -it <collector-pod> -n telemetry -- env | grep DB_DSN

# 5. Check logs
kubectl logs <collector-pod> -n telemetry

# 6. Port-forward to test
kubectl port-forward svc/postgres 5432:5432 -n telemetry
# From another terminal:
psql -h localhost -U postgres -d telemetry
```

### Data Not Appearing in Database

```bash
# 1. Check Collector logs
kubectl logs -l app=collector -n telemetry | grep -i "error\|failed\|insert"

# 2. Check if messages exist in MQ
# (depends on your MQ client)

# 3. Verify PostgreSQL is accepting connections
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry -c "SELECT 1;"

# 4. Check for unique constraint violations (expected for duplicates)
kubectl logs -l app=collector -n telemetry | grep -i "conflict\|duplicate"

# 5. Manually check table
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry -c "SELECT * FROM telemetry LIMIT 10;"
```

### High Memory Usage

```bash
# 1. Check pod memory
kubectl top pods -n telemetry

# 2. Increase memory limits
# Edit postgres-deployment.yaml:
# resources:
#   limits:
#     memory: "1Gi"  # Increase from 512Mi

kubectl apply -f postgres-deployment.yaml

# 3. Check PostgreSQL settings
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry -c "SELECT name, setting FROM pg_settings WHERE name LIKE '%buffers' OR name LIKE '%cache';"
```

### Collector Pods Crashing

```bash
# 1. Check previous logs
kubectl logs <pod-name> -n telemetry --previous

# 2. Describe pod
kubectl describe pod <pod-name> -n telemetry

# 3. Check events
kubectl get events -n telemetry --field-selector involvedObject.name=<pod-name>

# 4. Check resource availability
kubectl describe node <node-name>

# 5. Check for OOMKilled
kubectl describe pod <pod-name> -n telemetry | grep -i "oom\|killed"

# If OOMKilled, increase limits:
# resources:
#   limits:
#     memory: "512Mi"  # Increase from 256Mi
```

---

## Customization

### Scaling PostgreSQL to HA (3 replicas with streaming replication)

```yaml
# Change replicas in StatefulSet
spec:
  replicas: 3  # Was 1

# Add streaming replication parameters
- name: POSTGRES_INITDB_ARGS
  value: "-c max_wal_senders=3 -c max_replication_slots=3"
```

### Increasing Storage

```yaml
# In volumeClaimTemplates
resources:
  requests:
    storage: 50Gi  # Increase from 10Gi
```

### Adjusting Collector Replicas

```bash
# Set manual scale
kubectl scale deployment collector --replicas=5 -n telemetry

# Check HPA limits (don't exceed maxReplicas: 10)
```

### Changing PostgreSQL Version

```yaml
# In StatefulSet
- image: postgres:16-alpine  # Update version
  imagePullPolicy: IfNotPresent
```

### Adding Backup Strategy

```bash
# Create backup pod (manual example)
kubectl exec -it postgres-0 -n telemetry -- pg_dump -U postgres telemetry > telemetry-backup.sql

# Restore
kubectl exec -i postgres-0 -n telemetry -- psql -U postgres telemetry < telemetry-backup.sql
```

---

## Cleanup

### Delete Everything

```bash
# Delete all telemetry namespace resources
kubectl delete namespace telemetry

# Or delete specific resources
kubectl delete -f postgres-deployment.yaml
```

### Keep PostgreSQL Data (Delete Cluster Only)

```bash
# Delete deployment but keep PVC
kubectl delete deployment collector -n telemetry
kubectl delete statefulset postgres -n telemetry
# But keep:
kubectl get pvc -n telemetry  # Data persisted
```

---

## Security Best Practices

✅ **Implemented in this deployment**:
- Database credentials in Kubernetes Secrets (not environment variables in pods)
- Pod security contexts (non-root users, read-only filesystems where possible)
- Network policies for traffic control
- RBAC for minimal permissions
- Resource limits to prevent resource exhaustion
- Health checks for pod restart on failures
- Pod disruption budgets for availability

⚠️ **Additional recommendations for production**:
- Use HashiCorp Vault or AWS Secrets Manager for credential rotation
- Enable Kubernetes audit logging
- Use NetworkPolicy to restrict egress to required services only
- Implement OPA/Gatekeeper for policy enforcement
- Enable Pod Security Policies or Pod Security Standards
- Use TLS for PostgreSQL connections (`sslmode=require`)
- Implement regular backups and disaster recovery
- Monitor and alert on security events

---

## Performance Tuning

### PostgreSQL Optimization

```bash
# Connect to PostgreSQL
kubectl exec -it postgres-0 -n telemetry -- psql -U postgres -d telemetry

# Check current settings
SHOW shared_buffers;
SHOW max_connections;
SHOW effective_cache_size;

# Modify settings (persistent requires rebuild)
ALTER SYSTEM SET max_connections = 300;
SELECT pg_reload_conf();
```

### Collector Tuning

```bash
# Increase batch size for higher throughput
BATCH_SIZE=50  # Default: 10

# Decrease poll interval for lower latency
POLL_INTERVAL_MS=100  # Default: 500

# Increase replicas for parallel processing
kubectl scale deployment collector --replicas=10 -n telemetry
```

---

## Support

For issues or questions:
1. Check logs: `kubectl logs -l app=collector -n telemetry`
2. Verify connectivity: `kubectl exec <pod> -n telemetry -- nc -zv postgres 5432`
3. Check events: `kubectl get events -n telemetry --sort-by='.lastTimestamp'`
4. Review PostgreSQL status: `kubectl describe statefulset postgres -n telemetry`

---

**Last Updated**: April 2026
**Kubernetes Version**: 1.24+
**PostgreSQL Version**: 15
**Collector**: Latest
