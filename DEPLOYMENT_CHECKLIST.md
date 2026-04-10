# ✅ Kubernetes Deployment Checklist

## Pre-Deployment Checklist

- [ ] Kubernetes cluster available (local Kind or cloud)
- [ ] `kubectl` command working
- [ ] Navigate to: `collector/k8s/`
- [ ] Script is executable: `chmod +x deploy.sh` (if needed)
- [ ] Review: `QUICK_REFERENCE.md` (2 min read)

## Deployment Steps

### Step 1: Deploy
```bash
./deploy.sh deploy
```
**Expected**: All resources created successfully

- [ ] Namespace created
- [ ] Secrets created
- [ ] ConfigMaps created
- [ ] PVC created
- [ ] Services created
- [ ] PostgreSQL StatefulSet created
- [ ] Collector Deployment created
- [ ] RBAC created
- [ ] HPA created
- [ ] NetworkPolicy created
- [ ] PDB created

### Step 2: Wait for Readiness
```bash
./deploy.sh verify
```
**Expected**: All pods in Running state

- [ ] Namespace: telemetry
- [ ] postgres-0 pod: Running ✓
- [ ] 3 collector pods: Running ✓
- [ ] postgres-pvc: Bound ✓
- [ ] postgres service: Available ✓
- [ ] collector service: Available ✓

### Step 3: Test Database Connection
```bash
./deploy.sh connect-postgres
```
**Expected**: PostgreSQL prompt appears

- [ ] Connection successful
- [ ] Run: `SELECT COUNT(*) FROM telemetry;`
- [ ] Shows: count: 0
- [ ] Exit with: `\q`

### Step 4: Test Collector Status
```bash
./deploy.sh logs collector
```
**Expected**: Logs show polling without errors

- [ ] Logs show startup message
- [ ] Shows "consume partition=0,1,2"
- [ ] No error messages
- [ ] Shows timestamp updates

### Step 5: Test Data Insertion
```bash
./deploy.sh test-data
./deploy.sh query "SELECT COUNT(*) FROM telemetry;"
```
**Expected**: 3 rows inserted, query returns 3

- [ ] Test data inserted
- [ ] COUNT query returns: 3
- [ ] Data verification successful

---

## Post-Deployment Checklist

### Database
- [ ] PostgreSQL pod running
- [ ] Storage allocated (10Gi)
- [ ] Table created: telemetry
- [ ] Indexes created (5 total)
- [ ] Unique constraint working
- [ ] Health checks passing

### Collector Service
- [ ] 3 pods running
- [ ] All pods Ready: 1/1
- [ ] No restart loops
- [ ] Health checks passing
- [ ] Logs showing no errors

### Networking
- [ ] postgres service DNS resolving
- [ ] collector service DNS resolving
- [ ] Collector can reach PostgreSQL
- [ ] NetworkPolicy not blocking traffic

### Security
- [ ] RBAC roles created
- [ ] ServiceAccount configured
- [ ] Secrets encrypted at rest
- [ ] NetworkPolicy active
- [ ] Pod security contexts applied

### Auto-Scaling
- [ ] HPA created
- [ ] Min replicas: 3
- [ ] Max replicas: 10
- [ ] Metrics available
- [ ] CPU target: 70%
- [ ] Memory target: 80%

### High Availability
- [ ] PDB created
- [ ] Min available: 2
- [ ] Anti-affinity configured
- [ ] No pod running on same node

---

## Configuration Review

Before production, customize:

### Critical (⚠️ Must Change)
- [ ] PostgreSQL password
  - [ ] Location: postgres-deployment.yaml → Secret → POSTGRES_PASSWORD
  - [ ] Generate: `openssl rand -base64 32`
  - [ ] Change to: Your secure password

### Important (Should Review)
- [ ] Storage size
  - [ ] Current: 10Gi
  - [ ] Location: postgres-deployment.yaml → volumeClaimTemplates → storage
  - [ ] Change if: Expecting large data volume

- [ ] Collector replicas
  - [ ] Current: 3 (auto-scales 3-10)
  - [ ] Location: postgres-deployment.yaml → HPA → minReplicas/maxReplicas
  - [ ] Adjust if: Expecting different load

- [ ] Resource limits
  - [ ] PostgreSQL: 256Mi→512Mi
  - [ ] Collector: 128Mi→256Mi
  - [ ] Location: postgres-deployment.yaml → resources → limits
  - [ ] Increase if: Getting OOMKilled errors

### Optional (Nice to Have)
- [ ] Storage class
  - [ ] Current: standard
  - [ ] Location: postgres-deployment.yaml → storageClassName
  - [ ] Change for: AWS (gp3), GCP (standard), Azure (managed-premium)

- [ ] MQ URL
  - [ ] Current: http://mq-service:8080
  - [ ] Location: postgres-deployment.yaml → Deployment → env → MQ_URL
  - [ ] Change if: MQ service at different location

- [ ] Batch size
  - [ ] Current: 10
  - [ ] Location: postgres-deployment.yaml → Deployment → env → BATCH_SIZE
  - [ ] Increase for: Better throughput

- [ ] Poll interval
  - [ ] Current: 500ms
  - [ ] Location: postgres-deployment.yaml → Deployment → env → POLL_INTERVAL_MS
  - [ ] Decrease for: Lower latency

---

## Backup Strategy

### First-Time Backups
- [ ] Create initial backup: `./deploy.sh backup`
- [ ] Store backup file safely
- [ ] Document backup location
- [ ] Test restore: `./deploy.sh restore <backup-file>`

### Ongoing Backups
- [ ] Schedule daily backups
- [ ] Document backup procedure
- [ ] Test restoration monthly
- [ ] Archive old backups

### Disaster Recovery
- [ ] Have restore procedure documented
- [ ] Test restore to new cluster
- [ ] Verify data integrity after restore
- [ ] Document RTO/RPO requirements

---

## Monitoring Setup

### Logs
- [ ] Collector logs monitored: `./deploy.sh logs collector -f`
- [ ] PostgreSQL logs monitored: `kubectl logs postgres-0 -n telemetry -f`
- [ ] Log aggregation configured (ELK, Datadog, etc.)
- [ ] Alerts on error patterns

### Metrics
- [ ] Resource usage monitored
- [ ] Pod CPU/memory tracked
- [ ] Database metrics collected
- [ ] Performance alerts configured

### Health Checks
- [ ] Liveness probes working
- [ ] Readiness probes working
- [ ] Pod restart counts monitored
- [ ] Service availability tracked

---

## Security Hardening

### Before Production
- [ ] Change default PostgreSQL password ⚠️
- [ ] Enable TLS for database connections
- [ ] Use managed secrets service
- [ ] Rotate credentials regularly
- [ ] Enable audit logging
- [ ] Restrict network access
- [ ] Use pod security policies
- [ ] Implement RBAC properly
- [ ] Add image scanning
- [ ] Regular security updates

### Networking
- [ ] NetworkPolicy enabled ✓
- [ ] Ingress rules configured
- [ ] Egress rules configured
- [ ] No unnecessary ports open
- [ ] VPN/private network used if needed

### Access Control
- [ ] ServiceAccount configured ✓
- [ ] Role-based access implemented ✓
- [ ] Resource quotas set
- [ ] Network policies enforced
- [ ] Pod security standards applied

---

## Performance Tuning

### Database Optimization
- [ ] Connection pooling verified
- [ ] Index usage monitored
- [ ] Query performance tracked
- [ ] Cache settings optimized
- [ ] Buffer pools sized appropriately

### Collector Optimization
- [ ] Batch size tuned for throughput
- [ ] Poll interval tuned for latency
- [ ] Replica count based on load
- [ ] Resource limits appropriate
- [ ] No memory leaks

### Infrastructure Optimization
- [ ] Node sizing appropriate
- [ ] Storage class optimized
- [ ] Network bandwidth sufficient
- [ ] CPU resources adequate
- [ ] No bottlenecks identified

---

## Testing Checklist

### Functionality Tests
- [ ] Deploy completes successfully
- [ ] PostgreSQL initializes with schema
- [ ] Collector connects to database
- [ ] Messages are processed
- [ ] Idempotency works (no duplicates)
- [ ] Graceful shutdown works

### Integration Tests
- [ ] Collector receives from MQ
- [ ] Messages parsed correctly
- [ ] Data stored with correct schema
- [ ] Ack sent after processing
- [ ] Error handling works

### Stress Tests
- [ ] High volume message processing
- [ ] Database handles load
- [ ] Auto-scaling triggers correctly
- [ ] No data loss under stress
- [ ] Memory stays stable

### Failover Tests
- [ ] Pod restart doesn't lose data
- [ ] Database failover works
- [ ] Collector reconnects after DB restart
- [ ] Data integrity maintained

---

## Deployment Validation

### Final Verification
- [ ] All pods running: `./deploy.sh verify`
- [ ] All services accessible: `kubectl get svc -n telemetry`
- [ ] Database responsive: `./deploy.sh connect-postgres`
- [ ] Collector logs clean: `./deploy.sh logs collector`
- [ ] Data flowing: `./deploy.sh test-data`
- [ ] Auto-scaling working: `kubectl get hpa -n telemetry`
- [ ] Backups working: `./deploy.sh backup`

### Sign-Off
- [ ] Technical review completed
- [ ] Performance requirements met
- [ ] Security requirements met
- [ ] Documentation reviewed
- [ ] Runbook completed
- [ ] Team trained
- [ ] Approved for production

---

## Troubleshooting Quick Links

| Issue | Check | File |
|-------|-------|------|
| Pod not starting | Pod events | k8s/DEPLOYMENT_GUIDE.md |
| DB connection failed | Service DNS | k8s/DEPLOYMENT_GUIDE.md |
| No data appearing | Collector logs | k8s/DEPLOYMENT_GUIDE.md |
| High resource usage | Pod metrics | k8s/DEPLOYMENT_GUIDE.md |
| OOMKilled | Memory limits | k8s/DEPLOYMENT_GUIDE.md |
| Slow performance | Database queries | k8s/DEPLOYMENT_GUIDE.md |
| Network issues | NetworkPolicy | k8s/DEPLOYMENT_GUIDE.md |

---

## Documentation Links

- Quick start: `QUICK_REFERENCE.md`
- K8s guide: `k8s/README.md`
- Complete guide: `k8s/DEPLOYMENT_GUIDE.md`
- Deployment summary: `DEPLOYMENT_COMPLETE.md`
- Service documentation: `README.md`

---

## Sign-Off

| Role | Name | Date | Sign |
|------|------|------|------|
| Deployer | _____ | _____ | _____ |
| Reviewer | _____ | _____ | _____ |
| Approver | _____ | _____ | _____ |

---

**Status**: Ready for deployment
**Next Step**: Run `./deploy.sh deploy`
**Support**: See `k8s/DEPLOYMENT_GUIDE.md` for detailed help
