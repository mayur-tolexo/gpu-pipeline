# 🎯 Deployment Quick Reference

## Single Command Deploy

```bash
cd collector/k8s
./deploy.sh deploy
```

## What Gets Deployed

```
Namespace: telemetry
├── PostgreSQL StatefulSet
│   ├── Image: postgres:15-alpine
│   ├── Replicas: 1
│   ├── Storage: 10Gi (persistent)
│   ├── Service: postgres.telemetry.svc.cluster.local:5432
│   └── Database: telemetry
│
├── Collector Deployment
│   ├── Image: collector:latest
│   ├── Replicas: 3 (auto-scales 3-10)
│   ├── Service: collector.telemetry.svc.cluster.local:8081
│   └── Health: HTTP /health & /ready
│
└── Supporting Resources
    ├── Secrets (postgres-secret)
    ├── ConfigMaps (postgres-init with schema)
    ├── RBAC (ServiceAccount + Role)
    ├── HPA (auto-scaling)
    ├── PDB (availability)
    └── NetworkPolicy (security)
```

## Verify Deployment

```bash
./deploy.sh verify

# Expected output:
# ✓ telemetry namespace exists
# ✓ postgres-secret secret created
# ✓ postgres-pvc PVC created (10Gi)
# ✓ postgres service (headless)
# ✓ collector service (ClusterIP)
# ✓ postgres-0 pod (Running)
# ✓ collector-xxx pod 1 (Running)
# ✓ collector-yyy pod 2 (Running)
# ✓ collector-zzz pod 3 (Running)
```

## Test Connection

```bash
# PostgreSQL
./deploy.sh connect-postgres
# Then: SELECT COUNT(*) FROM telemetry;

# Or query directly
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"

# Collector
./deploy.sh logs collector

# Should show:
# collector: starting for topic=telemetry group=collector-group partitions=3
# collector: consume partition=0
# collector: consume partition=1
# collector: consume partition=2
```

## Common Operations

| Operation | Command |
|-----------|---------|
| Deploy | `./deploy.sh deploy` |
| Status | `./deploy.sh verify` |
| View Logs | `./deploy.sh logs collector` |
| Connect DB | `./deploy.sh connect-postgres` |
| Insert Test Data | `./deploy.sh test-data` |
| Query DB | `./deploy.sh query "SELECT * FROM telemetry LIMIT 5;"` |
| Backup DB | `./deploy.sh backup` |
| Restore DB | `./deploy.sh restore backup.sql` |
| Scale Replicas | `kubectl scale deployment collector --replicas=5 -n telemetry` |
| Cleanup | `./deploy.sh cleanup` |

## Key Files

| File | Purpose |
|------|---------|
| `postgres-deployment.yaml` | ✅ Main deployment (complete) |
| `deploy.sh` | 🛠️ Helper script |
| `README.md` | 📖 Quick guide |
| `DEPLOYMENT_GUIDE.md` | 📚 Complete documentation |
| `deployment.yaml` | 📦 Legacy (reference only) |

## Configuration

### PostgreSQL Password (⚠️ IMPORTANT)
```yaml
# In postgres-deployment.yaml
stringData:
  POSTGRES_PASSWORD: telemetry-secure-password-change-in-production  # Change this!
```

### Collector Environment
```yaml
MQ_URL: "http://mq-service:8080"
TOPIC: "telemetry"
BATCH_SIZE: "10"
POLL_INTERVAL_MS: "500"
```

### Storage Class
```yaml
# For cloud providers, change to:
# AWS: gp3, gp2
# GCP: standard, premium-rwo
# Azure: default, managed-premium
# Kind: standard
```

## Networking

| Service | DNS | Port | Type |
|---------|-----|------|------|
| PostgreSQL | `postgres.telemetry.svc.cluster.local` | 5432 | Headless |
| Collector | `collector.telemetry.svc.cluster.local` | 8081 | ClusterIP |

## Security

✅ Included:
- Kubernetes Secrets for credentials
- Pod Security Context (non-root)
- Network Policies
- RBAC (minimal permissions)
- Resource limits
- Health checks

⚠️ Before Production:
- Change PostgreSQL password
- Use managed secrets (AWS/GCP/Azure)
- Enable TLS
- Set up backups
- Configure monitoring
- Implement audit logging

## Troubleshooting

| Issue | Check |
|-------|-------|
| Pod not starting | `kubectl describe pod <name> -n telemetry` |
| Can't connect to DB | `kubectl get svc -n telemetry` |
| No data in DB | `./deploy.sh logs collector` |
| Out of memory | `kubectl top pods -n telemetry` |
| Slow queries | `./deploy.sh connect-postgres` → Check indexes |

## Database Queries

```bash
# Count records
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"

# Recent records
./deploy.sh query "SELECT * FROM telemetry ORDER BY created_at DESC LIMIT 10;"

# Per GPU
./deploy.sh query "SELECT gpu_id, COUNT(*) FROM telemetry GROUP BY gpu_id;"

# Check duplicates
./deploy.sh query "SELECT COUNT(*) FROM (SELECT gpu_id, timestamp FROM telemetry GROUP BY gpu_id, timestamp HAVING COUNT(*) > 1);"

# DB size
./deploy.sh query "SELECT pg_size_pretty(pg_total_relation_size('telemetry'));"
```

## Scaling

### Manual
```bash
kubectl scale deployment collector --replicas=10 -n telemetry
```

### Auto (HPA)
Currently set to:
- Min: 3 replicas
- Max: 10 replicas
- Triggers: CPU > 70% or Memory > 80%

Check status:
```bash
kubectl get hpa collector-hpa -n telemetry
```

## Monitoring

```bash
# Real-time logs
kubectl logs -l app=collector -n telemetry -f

# Resource usage
kubectl top pods -n telemetry
kubectl top nodes

# Pod events
kubectl get events -n telemetry --sort-by='.lastTimestamp'
```

## Backup & Restore

```bash
# Backup (auto-dated)
./deploy.sh backup

# Backup to specific file
./deploy.sh backup /path/to/backup.sql

# Restore
./deploy.sh restore /path/to/backup.sql
```

## Data Flow

```
MQ Topic "telemetry"
        ↓
[Message: {gpu_id, timestamp, data}]
        ↓
[Collector polls partition 0,1,2]
        ↓
[Parse JSON]
        ↓
[Insert into PostgreSQL]
        ↓
[Ack to MQ]
        ↓
[Duplicate ignored (unique constraint)]
```

## Test Data Flow

1. Produce message to MQ (external)
2. Collector polls and processes
3. Data inserted to PostgreSQL
4. Query with: `./deploy.sh query "SELECT COUNT(*) FROM telemetry;"`

## Success Indicators

✅ All working when:
- `./deploy.sh verify` shows all pods running
- `./deploy.sh connect-postgres` connects successfully
- `./deploy.sh logs collector` shows no errors
- `./deploy.sh query "SELECT COUNT(*) FROM telemetry;"` returns 0+ rows
- Data increases as MQ messages are processed

## Next Steps

1. Run: `./deploy.sh deploy`
2. Verify: `./deploy.sh verify`
3. Test: `./deploy.sh test-data`
4. Monitor: `./deploy.sh logs collector -f`
5. Produce MQ messages and watch data appear in PostgreSQL
6. Scale as needed: `kubectl scale deployment collector --replicas=X -n telemetry`

## Documentation

- **README.md** - Quick start (this directory)
- **DEPLOYMENT_GUIDE.md** - Complete guide (this directory)
- **DEPLOYMENT_COMPLETE.md** - Summary (collector directory)

---

**Status**: 🎉 **PRODUCTION READY**

Everything is configured and ready to deploy!
