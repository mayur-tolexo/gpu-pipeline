# Telemetry Streamer

The Telemetry Streamer reads telemetry data from a CSV file and continuously publishes it to the Message Queue (MQ). It simulates real-time telemetry ingestion and is designed to scale horizontally in Kubernetes.

---

## 📋 Table of Contents

1. [Overview](#overview)
2. [Configuration](#configuration)
3. [Local Development](#local-development)
4. [Docker](#docker)
5. [Kubernetes Deployment](#kubernetes-deployment)
6. [Storage Modes](#storage-modes)
7. [Scaling](#scaling)
8. [Design Decisions](#design-decisions)

---

## Overview

**Purpose:** Stream GPU telemetry data from CSV files to a message queue

**Data Flow:**
```
CSV File → Streamer → Message Queue → Collector → PostgreSQL → API Gateway
```

**Key Features:**
- Reads telemetry CSV files
- Publishes to partitioned message queue
- Scales to multiple replicas
- Supports flexible storage modes
- Kubernetes-native deployment

---

## Configuration

Environment variables control streamer behavior:

| Variable | Description | Default |
|----------|-------------|---------|
| `CSV_FILE` | Path to telemetry CSV file | `/data/telemetry.csv` |
| `MQ_URL` | Message Queue service URL | `http://mq-service` |
| `TOPIC` | MQ topic name | `telemetry` |
| `STREAM_INTERVAL_MS` | Delay between messages (ms) | `5000` |

Example:
```bash
export CSV_FILE=/data/telemetry.csv
export MQ_URL=http://mq-service:8080
export TOPIC=telemetry
export STREAM_INTERVAL_MS=500
```

---

## Local Development

### Build Binary

```bash
make build
```

Outputs: `bin/streamer`

### Run Locally

```bash
make run
```

Requires:
- CSV file at `./data/telemetry.csv`
- MQ running at `http://localhost:8080`

### Tests

```bash
make test           # Run all tests
make coverage       # Show coverage
make lint           # Format code
```

---

## Docker

### Build Image

```bash
make docker
```

Builds image: `streamer:latest`

**Image details:**
- Builder stage: `golang:1.25-alpine`
- Runtime stage: `gcr.io/distroless/base-debian12`
- Includes telemetry CSV data at `/data/telemetry.csv`
- Size: ~30-50MB

### Run Container

```bash
make docker-run
```

Or manually:
```bash
docker run -it \
  -e CSV_FILE=/data/telemetry.csv \
  -e MQ_URL=http://host.docker.internal:8080 \
  streamer:latest
```

---

## Kubernetes Deployment

### Deployment via Helm (Recommended)

All services including streamer are deployed together via Helm:

```bash
# From root directory
make deploy

# Or use Helm directly
helm install gpu-pipeline helm/gpu-pipeline/ \
  --namespace gpu-pipeline \
  --create-namespace
```

### Storage Modes

The streamer supports **4 storage modes** for telemetry data. Configure via `helm/gpu-pipeline/values.yaml`:

#### Mode 1: Embedded (Default)

CSV data built into Docker image. No external setup needed.

```yaml
streamer:
  storage:
    type: embedded
```

**Pros:**
- ✅ Works anywhere
- ✅ No external setup
- ✅ Scales to any replicas

**Cons:**
- ❌ Rebuild needed to change data

#### Mode 2: hostPath (Local KIND Testing)

CSV loaded from host filesystem via KIND volume mount.

```yaml
streamer:
  storage:
    type: hostPath
    hostPath:
      enabled: true
      path: /data
      nodeName: kind-control-plane
```

**Setup:**
```bash
# 1. Create KIND cluster with /data mount
make kind-create

# 2. Copy CSV to host
make streamer-copy-csv

# 3. Verify
make streamer-verify-csv

# 4. Deploy with hostPath mode
helm install gpu-pipeline helm/gpu-pipeline/ \
  --set streamer.storage.type=hostPath
```

**Pros:**
- ✅ Update CSV without rebuild
- ✅ Fast iteration

**Cons:**
- ❌ Single node only
- ❌ Don't scale replicas

#### Mode 3: RWX (Multi-node Production)

CSV on shared NFS or RWX storage for multi-node clusters.

```yaml
streamer:
  storage:
    type: rwx
    rwx:
      storageClassName: nfs-client
      size: 5Gi
```

**Setup:**
```bash
# 1. Install NFS provisioner (one-time)
helm install nfs-provisioner nfs-subdir-external-provisioner/...

# 2. Upload CSV to NFS share

# 3. Deploy with RWX mode
helm install gpu-pipeline helm/gpu-pipeline/ \
  --set streamer.storage.type=rwx
```

**Pros:**
- ✅ Multi-node support
- ✅ Scale to many replicas
- ✅ Production-ready

**Cons:**
- ❌ Requires NFS setup
- ❌ Storage costs

#### Mode 4: Remote (HTTP/S3)

CSV downloaded from remote URL on pod startup.

```yaml
streamer:
  storage:
    type: remote
    remote:
      url: "https://s3.amazonaws.com/bucket/data.csv"
      retries: 3
```

**Setup:**
```bash
# Upload to S3/cloud storage
aws s3 cp streamer/data/telemetry.csv s3://bucket/data.csv

# Deploy with remote mode
helm install gpu-pipeline helm/gpu-pipeline/ \
  --set streamer.storage.type=remote \
  --set 'streamer.storage.remote.url=https://s3.amazonaws.com/bucket/data.csv'
```

**Pros:**
- ✅ Cloud-native
- ✅ Scale to any replicas
- ✅ No cluster storage needed

**Cons:**
- ❌ Network dependency
- ❌ Download latency

### Verify Deployment

```bash
# Check pod status
kubectl get pods -n gpu-pipeline -l app=streamer

# View logs
kubectl logs -n gpu-pipeline -l app=streamer -f

# Verify CSV in pod
kubectl exec -it -n gpu-pipeline deployment/streamer -- \
  head -3 /data/telemetry.csv
```

### Troubleshooting

**Pod stuck in Init:**
```bash
# Check init container logs
kubectl logs -n gpu-pipeline deployment/streamer -c wait-for-csv

# For hostPath: verify file on node
make streamer-verify-csv

# For remote: check download logs
kubectl logs -n gpu-pipeline deployment/streamer -c fetch-csv
```

**CSV not found:**
```bash
# Verify in pod
kubectl exec -it deployment/streamer -n gpu-pipeline -- ls -la /data/

# For hostPath
make streamer-copy-csv
```

---

## Scaling

### With Embedded Mode

Scale to any number of replicas:

```bash
kubectl scale deployment streamer --replicas=10 -n gpu-pipeline
# ✅ All replicas have data from image
```

### With hostPath Mode

Don't scale beyond 1 (other nodes don't have mount):

```bash
kubectl scale deployment streamer --replicas=1 -n gpu-pipeline
# ⚠️ Only works on single node
```

### With RWX Mode

Scale to any number of replicas:

```bash
kubectl scale deployment streamer --replicas=10 -n gpu-pipeline
# ✅ All replicas share NFS volume
```

### With Remote Mode

Scale to any number of replicas (concurrent downloads):

```bash
kubectl scale deployment streamer --replicas=10 -n gpu-pipeline
# ✅ Each pod downloads independently
```

---

## Design Decisions

| Decision | Rationale |
|----------|-----------|
| **Partition by gpu_id** | Ensures all events from same GPU go to same partition, maintaining order |
| **Multiple storage modes** | Flexible for dev, test, and production scenarios |
| **CSV file format** | Human-readable, easy to test with different datasets |
| **ENV-based config** | Standard Kubernetes practice, easy to override |
| **Timestamp override** | Ensures records appear as real-time during replay |

### Storage Mode Comparison

| Aspect | Embedded | hostPath | RWX | Remote |
|--------|----------|----------|-----|--------|
| **Setup** | None | KIND mount | NFS install | URL/S3 |
| **Data updates** | Rebuild | Copy file | Upload | Re-deploy |
| **Scaling** | Unlimited | Single node | Unlimited | Unlimited |
| **Performance** | Fastest | Fast | Medium | Slowest |
| **Cloud-native** | No | No | Yes | Yes |
| **Dev use** | ✅ | ✅ | ❌ | ❌ |
| **Production** | ✅ (static) | ❌ | ✅ | ✅ |

### Future Enhancements

- **Dynamic data loading** - Support CSV reload without pod restart
- **Compression** - Reduce network overhead when streaming
- **Batching** - Group events for efficient publishing
- **Metrics** - Prometheus export of throughput, latency, errors
- **S3 integration** - Native S3 SDK instead of HTTP downloads

---

## Data Formats

### CSV File Format

Required columns:
```
timestamp,metric_name,gpu_id,device,uuid,modelName,Hostname,container,pod,namespace,value,labels_raw
```

Example:
```csv
"2025-07-18T20:42:34Z","DCGM_FI_DEV_GPU_UTIL","0","nvidia0","GPU-xxxxx","NVIDIA H100 80GB HBM3","host1","","","","45","..."
```

### Message Queue Format

Published to topic as JSON:
```json
{
  "timestamp": "2025-07-18T20:42:34Z",
  "metric_name": "DCGM_FI_DEV_GPU_UTIL",
  "gpu_id": "0",
  "value": 45
}
```

Partition key: `gpu_id` (ensures same GPU always goes to same partition)

---

## Common Workflows

### Change Telemetry Data (Embedded Mode)

```bash
# 1. Update CSV file
cp /path/to/new/data.csv streamer/data/telemetry.csv

# 2. Rebuild Docker image
make docker-build-all

# 3. Load into KIND
make docker-load-all

# 4. Restart streamer
kubectl rollout restart deployment/streamer -n gpu-pipeline

# 5. Verify
kubectl logs -l app=streamer -n gpu-pipeline | tail -10
```

### Change Telemetry Data (hostPath Mode)

```bash
# 1. Copy new CSV to host
cp /path/to/new/data.csv ./data/telemetry.csv

# 2. Restart streamer (picks up new file immediately)
kubectl rollout restart deployment/streamer -n gpu-pipeline

# 3. Verify
make streamer-verify-csv
```

### Scale Streamer

```bash
# Scale to 10 replicas (works with all modes except hostPath)
kubectl scale deployment streamer --replicas=10 -n gpu-pipeline

# Watch scaling progress
kubectl get pods -l app=streamer -n gpu-pipeline -w
```

### Monitor Performance

```bash
# Stream logs from all streamer pods
kubectl logs -l app=streamer -n gpu-pipeline -f

# Check pod resource usage
kubectl top pods -l app=streamer -n gpu-pipeline

# Check message queue depth
kubectl exec -it mq-pod -- \
  curl -s http://localhost:8080/topics/telemetry/stats | jq .
```

---

## Makefile Commands

| Command | Purpose |
|---------|---------|
| `make build` | Build binary |
| `make run` | Run locally |
| `make test` | Run tests |
| `make coverage` | Show coverage |
| `make lint` | Format code |
| `make docker` | Build Docker image |
| `make docker-run` | Run Docker container |

---

## Related Documentation

- **Root README**: `../README.md` - Overall project architecture
- **Helm Chart**: `../helm/gpu-pipeline/` - Deployment configuration
- **Kubernetes**: `../deployment/k8s/streamer.yaml` - K8s manifest
- **KIND Config**: `../kind-config.yaml` - Local cluster setup

---

## Summary

| Aspect | Details |
|--------|---------|
| **Language** | Go 1.25+ |
| **Entry Point** | `streamer/cmd/main.go` |
| **Configuration** | Environment variables |
| **Storage Modes** | 4 (embedded, hostPath, RWX, remote) |
| **Deployment** | Kubernetes via Helm |
| **Scaling** | Horizontal (except hostPath) |
| **Data Source** | CSV file at `/data/telemetry.csv` |
| **Output** | Message Queue (partitioned by gpu_id) |

---

## Quick Reference

```bash
# Development
make build && make run

# Testing
make test && make coverage

# Docker
make docker

# Kubernetes (all services including streamer)
make deploy

# Check streamer specifically
kubectl logs -l app=streamer -n gpu-pipeline -f

# Change data and redeploy (embedded mode)
cp new-data.csv streamer/data/telemetry.csv
make docker-build-all && make docker-load-all
kubectl rollout restart deployment/streamer -n gpu-pipeline
```
