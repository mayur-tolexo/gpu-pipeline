# GPU Pipeline - Complete Deployment Guide

This guide covers deploying the entire GPU Pipeline system to Kubernetes, including all 4 services (MQ, Streamer, Collector, API Gateway) and supporting infrastructure.

---

## 📋 Prerequisites

### Required Tools
- Go 1.25+
- Docker
- Kubernetes (Kind recommended for local, EKS/GKE for cloud)
- kubectl
- make
- Helm 3 (optional, for Helm deployments)

### Verify Installation
```bash
go version          # Go 1.25+
docker --version    # Latest stable
kubectl version     # v1.28+
kind version        # Latest
helm version        # 3.0+
```

---

## 🚀 Quick Deploy

### 1-Command Full Deployment (Recommended)

```bash
# Creates Kind cluster, builds all images, deploys all services
make kind-full
```

This automatically:
- Creates a local Kind cluster
- Builds all Docker images
- Creates the namespace and ConfigMap
- Deploys all services (MQ, Streamer, Collector, API Gateway)
- Initializes PostgreSQL database

### Verify Deployment

```bash
# Check all services are running
make verify

# Watch pods in real-time
make watch

# View logs from all services
make logs-all
```

---

## 📊 Telemetry Data Setup

### ✅ No ConfigMap Required

The telemetry data is now **embedded in the Docker image** for the Streamer service. No ConfigMap setup is needed!

**How it works:**
- The CSV file (`streamer/data/telemetry.csv`) is copied into the Docker image during build
- When the pod starts, the file is automatically available at `/data/telemetry.csv`
- All streamer replicas have the same consistent data

### Using Different Telemetry Data

To test with a different CSV file:

```bash
# 1. Replace the telemetry file
cp /path/to/your/telemetry.csv streamer/data/telemetry.csv

# 2. Rebuild the Docker image
make docker-build-all

# 3. Load into Kind (if using local cluster)
make docker-load-all

# 4. Restart streamer to pick up new image
kubectl rollout restart deployment/streamer -n gpu-pipeline

# 5. Verify new data loaded
kubectl exec -it deployment/streamer -n gpu-pipeline -- \
    head /data/telemetry.csv
```

### CSV File Format

The CSV file must contain GPU telemetry metrics with these columns:
```
timestamp,metric_name,gpu_id,device,uuid,modelName,Hostname,container,pod,namespace,value,labels_raw
```

Example:
```
"2025-07-18T20:42:34Z","DCGM_FI_DEV_GPU_UTIL","0","nvidia0","GPU-xxxxx","NVIDIA H100 80GB HBM3","host1","","","","45","..."
```

### Future: PersistentVolume Support

In the future, to support **dynamic data updates without rebuilding the image**, we should implement PersistentVolume storage:

```yaml
# Future enhancement (not yet implemented)
volumes:
  - name: telemetry-data
    persistentVolumeClaim:
      claimName: telemetry-data-pvc
```

See `streamer/README.md` for details on the future PV implementation plan.

---

## 🏗️ Deployment Stages

### Stage 1: Preparation

```bash
# Navigate to project root
cd /path/to/gpu-pipeline

# Verify all services are present
ls -la mq/ streamer/ collector/ api-gateway/

# Verify telemetry data exists
ls -la streamer/data/telemetry.csv
```

### Stage 2: Build

```bash
# Build all services
make build-all

# Verify binaries created
make verify-build
```

### Stage 3: Docker Images

```bash
# Build Docker images
make docker-build-all

# Load into Kind (if using local cluster)
make docker-load-all

# Or push to registry for cloud deployments
docker push your-registry/gpu-pipeline/mq:latest
docker push your-registry/gpu-pipeline/streamer:latest
docker push your-registry/gpu-pipeline/collector:latest
docker push your-registry/gpu-pipeline/api-gateway:latest
```

### Stage 4: Kubernetes Setup

```bash
# Create Kind cluster (local development)
make kind-create

# Or use existing cloud cluster
kubectl config use-context <your-cluster>

# Verify cluster access
kubectl cluster-info
```

### Stage 5: Deploy

```bash
# Full deployment (all services)
make deploy-all

# Or step-by-step:

# 1. Create namespace
kubectl create namespace gpu-pipeline

# 2. Create ConfigMap from telemetry data
bash scripts/create-configmap.sh gpu-pipeline

# 3. Deploy services
kubectl apply -n gpu-pipeline -f deployment/k8s/mq.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/mq-service.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/postgres.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/streamer.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/collector.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/api-gateway.yaml
kubectl apply -n gpu-pipeline -f deployment/k8s/api-gateway-service.yaml
```

### Stage 6: Verification

```bash
# Check all pods are running
kubectl get pods -n gpu-pipeline

# Check services
kubectl get svc -n gpu-pipeline

# Check ConfigMap
kubectl get configmap -n gpu-pipeline

# View logs
kubectl logs -n gpu-pipeline deployment/streamer
kubectl logs -n gpu-pipeline deployment/collector
kubectl logs -n gpu-pipeline deployment/api-gateway
```

---

## 🔍 Accessing Services

### API Gateway Swagger UI (Customer Access)

```bash
# Port-forward API Gateway
make api-gateway-port-forward

# In another terminal, open Swagger UI
make swagger-ui

# Or manually open browser to:
http://localhost:8000/swagger/
```

### Available API Endpoints

- `GET /api/v1/health` - Service health check
- `GET /api/v1/gpus` - List all GPU IDs
- `POST /api/v1/telemetry/query` - Query telemetry data (with time-range filtering)
- `GET /swagger/` - API documentation (Swagger UI)

### Example API Queries

```bash
# Get health
curl http://localhost:8000/api/v1/health

# List GPUs
curl http://localhost:8000/api/v1/gpus

# Query telemetry (last 1 hour)
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "start_time": "2025-07-18T19:42:34Z",
    "end_time": "2025-07-18T20:42:34Z",
    "gpu_id": "0",
    "metric": "DCGM_FI_DEV_GPU_UTIL"
  }'
```

### Message Queue Service

The MQ service is internal-only (not exposed):
- Port: 8080 (ClusterIP service, not accessible outside cluster)
- Used by: Streamer (producer) and Collector (consumer)

### PostgreSQL Database

The database is internal-only:
- Port: 5432 (ClusterIP service)
- Credentials: Set in `deployment/k8s/*.yaml` files
- Used by: Collector (storage) and API Gateway (queries)

---

## 📦 Helm Deployment

### Install via Helm

```bash
# Create ConfigMap from telemetry data first
bash scripts/create-configmap.sh gpu-pipeline

# Install Helm chart
make helm-install

# Or manually:
helm install gpu-pipeline helm/gpu-pipeline/ \
  --namespace gpu-pipeline \
  --create-namespace \
  --values helm/gpu-pipeline/values.yaml
```

**Note**: The ConfigMap must be created separately (before helm install) because Helm templates cannot reference external files at deployment time. The script handles this automatically.

### Upgrade Helm Release

```bash
helm upgrade gpu-pipeline helm/gpu-pipeline/ \
  --namespace gpu-pipeline \
  --values helm/gpu-pipeline/values.yaml
```

### Uninstall Helm Release

```bash
make helm-uninstall

# Or manually:
helm uninstall gpu-pipeline -n gpu-pipeline
```

---

## 🔧 Troubleshooting

### Pods Not Running

```bash
# Check pod status
kubectl get pods -n gpu-pipeline

# Describe problematic pod
kubectl describe pod <pod-name> -n gpu-pipeline

# View pod logs
kubectl logs <pod-name> -n gpu-pipeline
```

### ConfigMap Not Found

```bash
# Verify ConfigMap exists
kubectl get configmap -n gpu-pipeline

# If missing, create it
bash scripts/create-configmap.sh gpu-pipeline

# Verify data loaded
kubectl get configmap telemetry-csv -n gpu-pipeline -o yaml | head -50
```

### Database Connection Issues

```bash
# Check PostgreSQL pod
kubectl get pod -n gpu-pipeline | grep postgres

# View database logs
kubectl logs deployment/postgres -n gpu-pipeline

# Verify environment variables in deployments
kubectl get deployment -n gpu-pipeline -o yaml | grep -A 20 DATABASE_URL
```

### Service Discovery Issues

```bash
# Test MQ connectivity from collector pod
kubectl exec -it deployment/collector -n gpu-pipeline -- \
  curl -s http://mq-service:8080/health

# Test PostgreSQL connectivity
kubectl exec -it deployment/collector -n gpu-pipeline -- \
  pg_isready -h postgres.gpu-pipeline -p 5432
```

### API Gateway Not Responding

```bash
# Check API Gateway pod
kubectl get pod -n gpu-pipeline | grep api-gateway

# Check service
kubectl get svc -n gpu-pipeline api-gateway-service

# Test internal endpoint
kubectl exec -it deployment/api-gateway -n gpu-pipeline -- \
  curl -s http://localhost:8000/api/v1/health

# Verify port-forward
kubectl port-forward -n gpu-pipeline svc/api-gateway-service 8000:8000
```

---

## 🧹 Cleanup

### Delete All Deployments

```bash
# Delete namespace (removes all resources)
make cleanup

# Or manually:
kubectl delete namespace gpu-pipeline
```

### Full Reset (Including Cluster)

```bash
# Delete cluster and redeploy
make kind-full
```

### Delete Specific Service

```bash
# Delete just API Gateway
kubectl delete deployment,service -n gpu-pipeline -l app=api-gateway
```

---

## 📊 Monitoring

### Real-Time Pod Monitoring

```bash
# Watch all pods
make watch

# Specific service logs
make logs-all
make logs                    # Collector logs
```

### Resource Usage

```bash
# Check pod resource usage
kubectl top pods -n gpu-pipeline

# Check node resource usage
kubectl top nodes
```

### Service Status

```bash
# Verify all services
make verify

# Custom verification
kubectl get deployments,services,configmaps -n gpu-pipeline
```

---

## 🔄 Updating Services

### Update Single Service

```bash
# 1. Build new image
cd <service> && make docker-build && cd ..

# 2. Load into Kind
kind load docker-image gpu-pipeline/<service>:latest

# 3. Restart deployment
kubectl rollout restart deployment/<service> -n gpu-pipeline

# 4. Watch rollout
kubectl rollout status deployment/<service> -n gpu-pipeline
```

### Update With New Telemetry Data

```bash
# 1. Replace CSV file
cp /path/to/new/telemetry.csv streamer/data/telemetry.csv

# 2. Recreate ConfigMap
bash scripts/create-configmap.sh gpu-pipeline

# 3. Restart streamer
kubectl rollout restart deployment/streamer -n gpu-pipeline

# 4. Verify new data
kubectl exec -it deployment/streamer -n gpu-pipeline -- \
  head -5 /data/telemetry.csv
```

---

## 📚 Reference

### Key Files

- **Makefile** - Root orchestration targets
- **deployment/k8s/*.yaml** - Kubernetes manifests
- **helm/gpu-pipeline/** - Helm chart templates
- **scripts/create-configmap.sh** - ConfigMap generation script
- **streamer/data/telemetry.csv** - Sample telemetry data

### Service Ports

| Service | Port | Type | Access |
|---------|------|------|--------|
| MQ | 8080 | ClusterIP (Internal) | Within cluster only |
| PostgreSQL | 5432 | ClusterIP (Internal) | Within cluster only |
| API Gateway | 8000 | ClusterIP (Exposed via port-forward) | Customer-facing |

### Environment Variables

All services configured via environment variables in deployment YAML:
- `PORT` - Service port
- `DATABASE_URL` - PostgreSQL connection string (Collector/API Gateway)
- `MQ_HOST` - Message Queue hostname (Streamer/Collector)

---

## ✅ Deployment Checklist

- [ ] All prerequisites installed and verified
- [ ] Telemetry CSV file exists at `streamer/data/telemetry.csv`
- [ ] All services build successfully: `make build-all`
- [ ] All tests pass: `make test`
- [ ] Docker images build successfully: `make docker-build-all`
- [ ] Kubernetes cluster created/accessible
- [ ] ConfigMap created: `bash scripts/create-configmap.sh gpu-pipeline`
- [ ] All services deployed: `make deploy-all`
- [ ] All pods running: `kubectl get pods -n gpu-pipeline`
- [ ] API Gateway accessible: port-forward and test Swagger UI
- [ ] Data flowing: Check logs and query API Gateway

---

## 📞 Support

For issues or questions:
1. Check logs: `make logs-all`
2. Verify resources: `kubectl get all -n gpu-pipeline`
3. Review this guide's Troubleshooting section
4. Check individual service documentation in each service folder
