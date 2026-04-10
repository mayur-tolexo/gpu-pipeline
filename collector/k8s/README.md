# Kubernetes Deployment - PostgreSQL + Collector

> Production-ready Kubernetes deployment with PostgreSQL StatefulSet and Collector microservice

**Status**: ✅ **PRODUCTION READY**

---

## Table of Contents

1. [Quick Start](#quick-start)
2. [Architecture](#architecture)
3. [Files & Components](#files--components)
4. [Deployment](#deployment)
5. [Verification](#verification)
6. [Operations](#operations)
7. [Database Operations](#database-operations)
8. [Scaling](#scaling)
9. [Monitoring](#monitoring)
10. [Troubleshooting](#troubleshooting)
11. [Security](#security)
12. [Configuration](#configuration)
13. [Backup & Restore](#backup--restore)
14. [Cleanup](#cleanup)

---

## Quick Start

### Prerequisites
- Kubernetes 1.24+ (local Kind or cloud cluster)
- `kubectl` configured to access cluster
- Docker (for building images)

### Deploy in 3 Commands

```bash
# 1. Build and deploy
make k8s-deploy

# 2. Verify deployment
make k8s-verify

# 3. Test it works
make k8s-test-data
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"
```

**Done!** PostgreSQL and Collector running on Kubernetes ✅

---

## Architecture

```
┌──────────────────────────────────────────────────────────┐
│               Kubernetes Cluster                          │
│                                                           │
│  Namespace: telemetry                                    │
│                                                           │
│  ┌────────────────────┐      ┌──────────────────────┐   │
│  │  PostgreSQL        │◄─────│  Collector           │   │
│  │  StatefulSet       │      │  Deployment          │   │
│  │  • v15-Alpine      │      │  • 3 replicas        │   │
│  │  • 10Gi storage    │      │  • HPA: 3-10         │   │
│  │  • 1 replica       │      │  • Auto-connect      │   │
│  │  • Persistent      │      │  • Poll MQ topic     │   │
│  └────────────────────┘      └──────────────────────┘   │
│       ▲                              ▲                    │
│       │ Headless Service             │ ClusterIP Service │
│       │ postgres:5432                │ collector:8081    │
│       └──────────────────────────────┘                   │
│                                                           │
│  Supporting Components:                                   │
│  • RBAC: ServiceAccount + Role                          │
│  • Secrets: postgres-secret                             │
│  • HPA: Auto-scaling (3-10 replicas)                    │
│  • PDB: High availability (2+ pods)                     │
│  • NetworkPolicy: Traffic restrictions                  │
│  • PVC: 10Gi persistent storage                         │
│                                                           │
└──────────────────────────────────────────────────────────┘
```

---

## Files & Components

### `postgres-deployment.yaml` (MAIN FILE)
**Complete all-in-one Kubernetes deployment manifest** (~450 lines)

Includes:
- **Namespace** (`telemetry`) - Isolates all components
- **Secret** - PostgreSQL credentials
- **PVC** - Persistent storage (10Gi)
- **Services** - postgres (headless), collector (ClusterIP)
- **StatefulSet** - PostgreSQL 15-Alpine (1 replica)
- **Deployment** - Collector (3 replicas)
- **ServiceAccount & RBAC** - Minimal permissions
- **HPA** - Auto-scales 3-10 replicas
- **PDB** - Min 2 pods always available
- **NetworkPolicy** - Restrict traffic

**Status**: ✅ Single file, everything included

### Makefile Commands
All operations run from `../Makefile` with `make k8s-*` targets

**Build & Deploy**:
```bash
make k8s-deploy              # Full deployment
make kind-deploy            # Deploy to local Kind cluster
```

**Monitor & Verify**:
```bash
make k8s-verify             # Check all resources
make k8s-watch              # Watch pods in real-time
make k8s-logs POD=<name>    # View pod logs
make k8s-logs-all           # View all collector logs
```

**Database Operations**:
```bash
make k8s-db-connect         # PostgreSQL shell
make k8s-query SQL="..."    # Execute SQL
make k8s-test-data          # Insert test data
make k8s-backup             # Backup database
make k8s-restore BACKUP=... # Restore backup
```

**Scaling & Management**:
```bash
make k8s-scale REPLICAS=5   # Scale collector replicas
make k8s-hpa                # View HPA status
make k8s-resources          # Check resource usage
make k8s-cleanup            # Delete deployment
```

---

## Deployment

### Option 1: Using Kind (Local Development)

```bash
# Create Kind cluster and deploy
make kind-deploy

# Verify
make k8s-verify

# Port forward
make port-forward
```

Access: `http://localhost:8081/health`

### Option 2: Using kubectl Directly

```bash
# Deploy
kubectl apply -f k8s/postgres-deployment.yaml

# Verify
kubectl get all -n telemetry
```

### Option 3: Using Makefile

```bash
# Build, load to Kind, and deploy
make k8s-deploy

# For Kind cluster
make kind-deploy
```

### What Gets Deployed

| Component | Details |
|-----------|---------|
| **Namespace** | `telemetry` |
| **PostgreSQL** | StatefulSet, 1 replica, 15-Alpine, 10Gi storage |
| **Collector** | Deployment, 3 replicas, auto-scales 3-10 |
| **Storage** | PersistentVolumeClaim 10Gi, standard class |
| **Services** | postgres (headless), collector (ClusterIP) |
| **RBAC** | ServiceAccount + Role with minimal permissions |
| **Networking** | NetworkPolicy restricts to telemetry namespace |
| **HA** | PDB ensures 2+ pods, anti-affinity spreading |
| **Scaling** | HPA auto-scales on CPU>70% or Memory>80% |

---

## Verification

### 1. Check All Resources

```bash
make k8s-verify
```

Output will show:
- ✅ Namespace created
- ✅ Secrets configured
- ✅ PVC allocated
- ✅ Services running
- ✅ StatefulSet ready
- ✅ Deployment ready
- ✅ Pods running
- ✅ HPA active

### 2. Check Pods

```bash
make k8s-watch
```

Wait for all pods to be `Ready`:
```
NAME                      READY   STATUS    RESTARTS
postgres-0                1/1     Running   0
collector-abc123-def456   1/1     Running   0
collector-ghi789-jkl012   1/1     Running   0
collector-mno345-pqr678   1/1     Running   0
```

### 3. Test PostgreSQL Connection

```bash
make k8s-db-connect
```

Inside PostgreSQL shell:
```sql
postgres=# SELECT COUNT(*) FROM telemetry;
postgres=# \dt
postgres=# \q
```

### 4. Check Collector Logs

```bash
make k8s-logs-all
```

Should show:
```
collector: starting for topic=telemetry...
collector: consume partition=0...
collector: consume partition=1...
collector: consume partition=2...
```

### 5. Insert Test Data

```bash
make k8s-test-data
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"
```

---

## Operations

### View Logs

```bash
# All collector logs (streaming)
make k8s-logs-all

# Specific pod logs
make k8s-logs POD=collector-abc123-def456

# PostgreSQL logs
make k8s-logs POD=postgres-0

# Follow specific pod
make k8s-logs POD=<pod_name>
```

### Watch Pods

```bash
make k8s-watch
```

Continuously monitor pod status.

### Connect to PostgreSQL

```bash
make k8s-db-connect
```

Opens interactive PostgreSQL shell on pod.

### Execute SQL

```bash
make k8s-query SQL="SELECT * FROM telemetry LIMIT 10;"
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"
make k8s-query SQL="\d telemetry"
```

### Connect to Collector Pod

```bash
POD=$(kubectl get pod -l app=collector -n telemetry -o jsonpath='{.items[0].metadata.name}')
kubectl exec -it $POD -n telemetry -- sh
```

---

## Database Operations

### Insert Test Data

```bash
make k8s-test-data
```

Inserts 3 test records (using hardcoded values).

For production, use application to insert via MQ.

### Query Data

```bash
# Count records
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"

# Recent records
make k8s-query SQL="SELECT * FROM telemetry ORDER BY created_at DESC LIMIT 10;"

# Records per GPU
make k8s-query SQL="SELECT gpu_id, COUNT(*) FROM telemetry GROUP BY gpu_id;"

# Check for duplicates (idempotency)
make k8s-query SQL="SELECT gpu_id, timestamp, COUNT(*) FROM telemetry GROUP BY gpu_id, timestamp HAVING COUNT(*) > 1;"

# Database size
make k8s-query SQL="SELECT pg_size_pretty(pg_total_relation_size('telemetry'));"
```

### Backup Database

```bash
# Auto-named backup (telemetry-backup-<timestamp>.sql)
make k8s-backup

# Custom filename
make k8s-backup BACKUP_FILE=my-backup.sql
```

Creates backup in current directory.

### Restore Database

```bash
make k8s-restore BACKUP=telemetry-backup-123456.sql
```

Restores from backup file.

### Connect Directly

```bash
make k8s-db-connect
```

Opens PostgreSQL shell:
```sql
postgres=# \c telemetry
telemetry=# \dt
telemetry=# SELECT * FROM telemetry;
```

---

## Scaling

### View Current Scale

```bash
make k8s-verify | grep -A5 Deployments
```

### Manual Scale

```bash
# Scale to 5 replicas
make k8s-scale REPLICAS=5

# Scale to 10 replicas
make k8s-scale REPLICAS=10

# Scale down to 3
make k8s-scale REPLICAS=3
```

### Auto-Scaling (HPA)

Current configuration:
- **Min replicas**: 3
- **Max replicas**: 10
- **CPU threshold**: 70% utilization
- **Memory threshold**: 80% utilization

View HPA status:
```bash
make k8s-hpa
```

To modify HPA, edit `postgres-deployment.yaml` and reapply:
```bash
kubectl apply -f k8s/postgres-deployment.yaml
```

### Scale PostgreSQL (HA)

For high availability, modify `postgres-deployment.yaml`:
```yaml
spec:
  replicas: 3  # Change from 1
```

Then apply with streaming replication setup.

---

## Monitoring

### Pod Resource Usage

```bash
make k8s-resources
```

Shows CPU/Memory usage per pod (requires metrics-server).

### Check HPA Status

```bash
make k8s-hpa
```

Shows:
- Current replicas
- Desired replicas
- CPU/Memory metrics
- Last scale action

### View Events

```bash
make k8s-verify | grep -A10 "Pod Events"
```

Shows recent pod events and issues.

### Port Forward for Debugging

```bash
make port-forward
```

Access Collector health check:
```bash
curl http://localhost:8081/health
curl http://localhost:8081/ready
```

---

## Troubleshooting

### Pods Not Starting

```bash
# Check pod status
make k8s-verify

# Describe pod for events
kubectl describe pod <pod_name> -n telemetry

# Check logs
make k8s-logs POD=<pod_name>
```

### PostgreSQL Connection Refused

```bash
# Verify PostgreSQL pod is running
make k8s-watch

# Check PostgreSQL service
kubectl get svc -n telemetry

# Test DNS resolution
POD=$(kubectl get pod -l app=collector -n telemetry -o jsonpath='{.items[0].metadata.name}')
kubectl exec $POD -n telemetry -- nslookup postgres.telemetry.svc.cluster.local

# Test connection
kubectl exec $POD -n telemetry -- nc -zv postgres.telemetry.svc.cluster.local 5432
```

### No Data in Database

```bash
# Check Collector logs
make k8s-logs-all | grep -i "error\|failed"

# Verify table exists
make k8s-query SQL="\d telemetry"

# Check if data is being inserted
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"

# Insert test data
make k8s-test-data
```

### High Memory Usage

```bash
# Check pod resource usage
make k8s-resources

# Increase resource limits in postgres-deployment.yaml
# Edit limits for PostgreSQL container, then apply
kubectl apply -f k8s/postgres-deployment.yaml
```

### Pods Crashing (CrashLoopBackOff)

```bash
# Check logs
make k8s-logs POD=<pod_name>

# Check events
kubectl describe pod <pod_name> -n telemetry

# Common causes:
# 1. PostgreSQL not ready - check postgres pod
# 2. Connection string wrong - check DB_DSN env var
# 3. Memory limit too low - increase in YAML
# 4. Readiness probe failing - check /ready endpoint
```

### Storage Issues

```bash
# Check PVC
kubectl get pvc -n telemetry
kubectl describe pvc postgres-pvc -n telemetry

# Check storage class
kubectl get storageclass

# Increase storage size in postgres-deployment.yaml:
# spec.resources.requests.storage: 50Gi
# (Note: Cannot decrease, only increase)
```

---

## Security

### Change PostgreSQL Password

**⚠️ IMPORTANT**: Default password must be changed for production!

```bash
# Generate secure password
NEW_PASS=$(openssl rand -base64 32)
echo $NEW_PASS

# Update secret
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_USER=postgres \
  --from-literal=POSTGRES_PASSWORD=$NEW_PASS \
  --from-literal=POSTGRES_DB=telemetry \
  -n telemetry --dry-run=client -o yaml | kubectl apply -f -

# Restart PostgreSQL
kubectl delete pod postgres-0 -n telemetry
```

### Enable TLS for Database Connections

Edit `postgres-deployment.yaml`:
```yaml
env:
- name: DB_DSN
  value: "postgres://postgres:password@postgres.telemetry.svc.cluster.local:5432/telemetry?sslmode=require"
```

### Network Policy

Deployment includes NetworkPolicy restricting traffic:
```bash
kubectl get networkpolicy -n telemetry
kubectl describe networkpolicy telemetry-network-policy -n telemetry
```

### Pod Security

Implemented:
- ✅ Non-root user (uid 1000 for collector, 999 for postgres)
- ✅ Read-only root filesystem (where possible)
- ✅ Resource limits (prevent resource exhaustion)
- ✅ Security context with dropped capabilities
- ✅ RBAC with minimal permissions

---

## Configuration

### PostgreSQL Credentials

Edit `postgres-deployment.yaml`:
```yaml
stringData:
  POSTGRES_USER: postgres
  POSTGRES_PASSWORD: YOUR_SECURE_PASSWORD
  POSTGRES_DB: telemetry
```

Then apply:
```bash
kubectl apply -f k8s/postgres-deployment.yaml
```

### Collector Environment Variables

Edit `postgres-deployment.yaml` collector deployment:
```yaml
env:
- name: MQ_URL
  value: "http://mq-service:8080"
- name: TOPIC
  value: "telemetry"
- name: BATCH_SIZE
  value: "10"
- name: POLL_INTERVAL_MS
  value: "500"
- name: DB_DSN
  value: "postgres://user:pass@host:port/db"
```

### Storage Class

Default: `standard`

For cloud providers:
- **AWS EKS**: Use `gp3` or `io1`
- **GCP GKE**: Use `standard-rwo` or `premium-rwo`
- **Azure AKS**: Use `default` or `managed-premium`

Edit in `postgres-deployment.yaml`:
```yaml
storageClassName: gp3  # Change to your provider's class
```

### Resource Requests/Limits

PostgreSQL:
```yaml
resources:
  requests:
    memory: "256Mi"
    cpu: "250m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```

Collector:
```yaml
resources:
  requests:
    memory: "128Mi"
    cpu: "100m"
  limits:
    memory: "256Mi"
    cpu: "500m"
```

---

## Backup & Restore

### Create Backup

```bash
make k8s-backup
```

Creates `telemetry-backup-<timestamp>.sql`

### List Backups

```bash
ls -lah telemetry-backup-*.sql
```

### Restore from Backup

```bash
make k8s-restore BACKUP=telemetry-backup-123456.sql
```

### Manual Backup Command

```bash
POD=$(kubectl get pod -l app=postgres -n telemetry -o jsonpath='{.items[0].metadata.name}')
kubectl exec $POD -n telemetry -- pg_dump -U postgres telemetry > backup.sql
```

### Manual Restore Command

```bash
POD=$(kubectl get pod -l app=postgres -n telemetry -o jsonpath='{.items[0].metadata.name}')
kubectl exec -i $POD -n telemetry -- psql -U postgres telemetry < backup.sql
```

### Backup Strategy

- Backup after major data loads
- Store backups outside cluster
- Test restore periodically
- Use automated backup solution for production

---

## Cleanup

### Delete Deployment

```bash
make k8s-cleanup
```

Deletes namespace and all resources.

### Delete Specific Components

```bash
# Delete only collector
kubectl delete deployment collector -n telemetry
kubectl delete service collector -n telemetry

# Delete only PostgreSQL
kubectl delete statefulset postgres -n telemetry
kubectl delete pvc postgres-data-postgres-0 -n telemetry

# Delete namespace (deletes everything)
kubectl delete namespace telemetry
```

### Delete Kind Cluster

```bash
make kind-delete
```

---

## Common Commands Reference

```bash
# Deploy & verify
make k8s-deploy
make k8s-verify

# Monitor
make k8s-watch
make k8s-logs-all
make k8s-hpa
make k8s-resources

# Database
make k8s-db-connect
make k8s-query SQL="SELECT COUNT(*) FROM telemetry;"
make k8s-test-data

# Backup/restore
make k8s-backup
make k8s-restore BACKUP=telemetry-backup-123456.sql

# Scale
make k8s-scale REPLICAS=5

# Cleanup
make k8s-cleanup
```

---

## Makefile Targets

All `make k8s-*` targets available:

| Target | Purpose |
|--------|---------|
| `k8s-deploy` | Build and deploy everything |
| `k8s-verify` | Check all resources |
| `k8s-logs` | View pod logs |
| `k8s-logs-all` | View all collector logs |
| `k8s-watch` | Watch pods |
| `k8s-db-connect` | Connect to PostgreSQL |
| `k8s-query` | Execute SQL query |
| `k8s-test-data` | Insert test data |
| `k8s-backup` | Backup database |
| `k8s-restore` | Restore from backup |
| `k8s-scale` | Scale replicas |
| `k8s-hpa` | View HPA status |
| `k8s-resources` | Check resource usage |
| `k8s-cleanup` | Delete deployment |

---

## Support

For issues or questions:
1. Check logs: `make k8s-logs POD=<pod_name>`
2. Verify deployment: `make k8s-verify`
3. Check events: `make k8s-verify | tail -20`
4. Review this README's Troubleshooting section

---

**Status**: ✅ Production Ready
**Last Updated**: April 2026
**Kubernetes**: 1.24+
**PostgreSQL**: 15-Alpine

---

## 📋 What Gets Deployed

### Namespace
- **Name**: `telemetry`
- **Purpose**: Isolates all components

### PostgreSQL (StatefulSet)
- **Image**: `postgres:15-alpine`
- **Replicas**: 1 (single instance)
- **Storage**: 10Gi persistent volume
- **Health**: Liveness & readiness probes
- **Resources**: 256Mi RAM request, 512Mi limit
- **Features**:
  - Auto-initializes telemetry table with schema
  - Unique index on (gpu_id, timestamp) for idempotency
  - pgbouncer-compatible connection pooling

### Collector (Deployment)
- **Image**: `collector:latest`
- **Replicas**: 3 (managed by HPA)
- **Scaling**: 3-10 replicas based on CPU/memory
- **Health**: HTTP health checks on port 8081
- **Resources**: 128Mi RAM request, 256Mi limit
- **Features**:
  - Waits for PostgreSQL via init container
  - Auto-connects to PostgreSQL
  - Polls MQ topic "telemetry"
  - Inserts with idempotency

### Networking
- **PostgreSQL Service**: Headless (DNS: `postgres.telemetry.svc.cluster.local:5432`)
- **Collector Service**: ClusterIP (DNS: `collector.telemetry.svc.cluster.local:8081`)

### Security
- **ServiceAccount**: `collector` with minimal RBAC
- **Secrets**: `postgres-secret` with credentials
- **NetworkPolicy**: Restrict traffic to telemetry namespace
- **Pod Security**: Non-root users, resource limits

### Auto-Scaling
- **HPA**: 3-10 replicas based on CPU (70%) and Memory (80%)

### Reliability
- **PDB**: Minimum 2 pods running (for maintenance)
- **Anti-Affinity**: Spread replicas across nodes

---

## ⚙️ Configuration

### PostgreSQL Credentials

```yaml
# In postgres-deployment.yaml, edit Secret
stringData:
  POSTGRES_USER: postgres
  POSTGRES_PASSWORD: YOUR_SECURE_PASSWORD
  POSTGRES_DB: telemetry
```

**⚠️ IMPORTANT**: Change password before production deployment!

### Collector Environment Variables

```yaml
# In postgres-deployment.yaml, edit Deployment.env
- name: MQ_URL
  value: "http://mq-service:8080"
- name: TOPIC
  value: "telemetry"
- name: BATCH_SIZE
  value: "10"  # Messages per fetch
- name: POLL_INTERVAL_MS
  value: "500"  # ms between polls
```

### Storage

```yaml
# In StatefulSet.volumeClaimTemplates
resources:
  requests:
    storage: 10Gi  # Change as needed
storageClassName: standard  # Use cloud provider class
```

For different cloud providers:
- **AWS EKS**: `gp3`, `io1`
- **GCP GKE**: `standard`, `premium-rwo`
- **Azure AKS**: `default`, `managed-premium`
- **Kind**: `standard` (local storage)

### Resource Limits

Adjust based on your load:

```yaml
# PostgreSQL
resources:
  requests:
    memory: "256Mi"  # Increase for large databases
    cpu: "250m"
  limits:
    memory: "512Mi"

# Collector
resources:
  requests:
    memory: "128Mi"  # Increase if processing many messages
    cpu: "100m"
  limits:
    memory: "256Mi"
```

---

## ✅ Verification

### 1. Check Resources Created

```bash
./deploy.sh verify
# or
kubectl get all -n telemetry
```

### 2. Test PostgreSQL Connection

```bash
./deploy.sh connect-postgres

# Inside PostgreSQL shell:
postgres=# \dt
postgres=# SELECT COUNT(*) FROM telemetry;
postgres=# \q
```

### 3. Check Collector Status

```bash
./deploy.sh logs collector

# Should see:
# collector: starting for topic=telemetry...
# collector: consume partition=0...
# collector: consume partition=1...
# collector: consume partition=2...
```

### 4. Insert Test Data

```bash
./deploy.sh test-data

# Verify data was inserted
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"
```

---

## 🐛 Troubleshooting

### PostgreSQL Pod Won't Start

```bash
# Check pod status
kubectl describe pod postgres-0 -n telemetry

# Check logs
kubectl logs postgres-0 -n telemetry

# Check storage
kubectl get pvc -n telemetry
kubectl describe pvc postgres-data-postgres-0 -n telemetry
```

### Collector Can't Connect to PostgreSQL

```bash
# Check DNS resolution
kubectl exec -it <collector-pod> -n telemetry -- nslookup postgres.telemetry.svc.cluster.local

# Check PostgreSQL service
kubectl get svc postgres -n telemetry

# Test connection
kubectl exec -it <collector-pod> -n telemetry -- nc -zv postgres.telemetry.svc.cluster.local 5432

# Check Collector logs
./deploy.sh logs collector
```

### No Data in Database

```bash
# Check Collector logs for errors
./deploy.sh logs collector | grep -i "error\|failed"

# Verify table exists
./deploy.sh query "\d telemetry"

# Insert test data manually
./deploy.sh test-data

# Query data
./deploy.sh query "SELECT * FROM telemetry LIMIT 5;"
```

---

## 🔍 Monitoring

### View Pod Status

```bash
# All pods
kubectl get pods -n telemetry

# Watch for changes
kubectl get pods -n telemetry -w

# Detailed status
kubectl describe pod <pod-name> -n telemetry
```

### View Logs

```bash
# PostgreSQL
kubectl logs postgres-0 -n telemetry -f

# Collector (all pods)
kubectl logs -l app=collector -n telemetry -f

# Specific pod
kubectl logs <collector-pod> -n telemetry
```

### View Resource Usage

```bash
# Pod usage
kubectl top pods -n telemetry

# Node usage
kubectl top nodes
```

### Check HPA Status

```bash
kubectl get hpa collector-hpa -n telemetry
kubectl describe hpa collector-hpa -n telemetry
```

---

## 🔐 Security

### Change PostgreSQL Password

```bash
# Generate new password
NEW_PASS=$(openssl rand -base64 32)
echo $NEW_PASS

# Update secret
kubectl create secret generic postgres-secret \
  --from-literal=POSTGRES_USER=postgres \
  --from-literal=POSTGRES_PASSWORD=$NEW_PASS \
  --from-literal=POSTGRES_DB=telemetry \
  -n telemetry --dry-run=client -o yaml | kubectl apply -f -

# Restart PostgreSQL to use new password
kubectl delete pod postgres-0 -n telemetry
```

### Enable TLS for PostgreSQL

Edit `postgres-deployment.yaml`:

```yaml
# In Collector Deployment env
- name: DB_DSN
  value: "postgres://postgres:password@postgres.telemetry.svc.cluster.local:5432/telemetry?sslmode=require"
```

### Enable Network Policies

The deployment includes NetworkPolicy to restrict traffic. Verify:

```bash
kubectl get networkpolicy -n telemetry
kubectl describe networkpolicy telemetry-network-policy -n telemetry
```

---

## 📦 Backup & Restore

### Backup PostgreSQL

```bash
# Automatic backup to dated file
./deploy.sh backup

# Backup to specific file
./deploy.sh backup /path/to/telemetry-backup.sql
```

### Restore from Backup

```bash
./deploy.sh restore /path/to/telemetry-backup.sql
```

### Manual Backup

```bash
kubectl exec -i postgres-0 -n telemetry -- pg_dump -U postgres telemetry > backup.sql
```

### Manual Restore

```bash
kubectl exec -i postgres-0 -n telemetry -- psql -U postgres telemetry < backup.sql
```

---

## 🔄 Scaling

### Scale Collector Replicas

```bash
# Manual scale to 5 replicas
kubectl scale deployment collector --replicas=5 -n telemetry

# Verify
kubectl get deployment collector -n telemetry

# Check HPA status (if HPA is managing replicas)
kubectl get hpa collector-hpa -n telemetry
```

### Increase PostgreSQL Storage

```yaml
# Edit postgres-deployment.yaml
# In volumeClaimTemplates:
resources:
  requests:
    storage: 50Gi  # Increase from 10Gi
```

Apply changes:
```bash
kubectl apply -f postgres-deployment.yaml
```

### Increase Resource Limits

```yaml
# Edit postgres-deployment.yaml
# In containers.resources
resources:
  limits:
    memory: "1Gi"  # Increase from 512Mi
    cpu: "1000m"
```

---

## 🧹 Cleanup

### Delete Everything

```bash
./deploy.sh cleanup
# or
kubectl delete namespace telemetry
```

### Delete Only Collector (Keep PostgreSQL Data)

```bash
kubectl delete deployment collector -n telemetry
kubectl delete service collector -n telemetry
# Data persists in PVC
```

### Delete Only Collector Service

```bash
kubectl delete svc collector -n telemetry
```

---

## 📊 Database Queries

### Common Queries

```bash
# Count records
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"

# Recent records
./deploy.sh query "SELECT * FROM telemetry ORDER BY created_at DESC LIMIT 10;"

# Records per GPU
./deploy.sh query "SELECT gpu_id, COUNT(*) FROM telemetry GROUP BY gpu_id;"

# Check for duplicates (should be 0)
./deploy.sh query "SELECT COUNT(*) FROM (SELECT gpu_id, timestamp FROM telemetry GROUP BY gpu_id, timestamp HAVING COUNT(*) > 1);"

# Database size
./deploy.sh query "SELECT pg_size_pretty(pg_total_relation_size('telemetry'));"

# Active connections
./deploy.sh query "SELECT COUNT(*) FROM pg_stat_activity;"
```

---

## 🚨 Common Issues

| Issue | Solution |
|-------|----------|
| Pod stuck in pending | Check node resources: `kubectl describe nodes` |
| PostgreSQL OOMKilled | Increase memory limits in YAML |
| Connection refused | Verify PostgreSQL pod is running: `kubectl get pods -n telemetry` |
| No data appearing | Check Collector logs: `./deploy.sh logs collector` |
| High memory usage | Check PostgreSQL settings: `./deploy.sh connect-postgres` |
| Slow queries | Add indexes or increase PostgreSQL cache |

---

## 📚 Additional Resources

- **Deployment Guide**: See `DEPLOYMENT_GUIDE.md` for detailed documentation
- **README**: See `../README.md` for service documentation
- **Makefile**: See `../Makefile` for local Kind deployment

---

## 📞 Quick Commands

```bash
# Deploy
./deploy.sh deploy

# Monitor
./deploy.sh verify

# Logs
./deploy.sh logs collector
./deploy.sh logs postgres

# Database
./deploy.sh connect-postgres
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"

# Backup
./deploy.sh backup
./deploy.sh restore backup.sql

# Cleanup
./deploy.sh cleanup

# Help
./deploy.sh help
```

---

**Last Updated**: April 2026
**Kubernetes**: 1.24+
**PostgreSQL**: 15-Alpine
**Status**: ✅ Production Ready
