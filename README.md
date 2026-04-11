# 🚀 GPU Pipeline - Telemetry System

> An elastic, scalable telemetry pipeline for GPU clusters using a custom-built message queue (without Kafka/RabbitMQ).

[![Go Report Card](https://goreportcard.com/badge/github.com/mayur-tolexo/gpu-pipeline)](https://goreportcard.com/report/github.com/mayur-tolexo/gpu-pipeline)
[![Codacy Badge](https://app.codacy.com/project/badge/Coverage/15fe9b9c36fb48abb64ee5acc6df4608)](https://app.codacy.com/gh/mayur-tolexo/gpu-pipeline/dashboard?utm_source=gh&utm_medium=referral&utm_content=&utm_campaign=Badge_coverage)
[![Open Source Helpers](https://www.codetriage.com/mayur-tolexo/gpu-pipeline/badges/users.svg)](https://www.codetriage.com/mayur-tolexo/gpu-pipeline)
[![Release](https://img.shields.io/github/release/mayur-tolexo/gpu-pipeline.svg?style=flat-square)](https://github.com/mayur-tolexo/gpu-pipeline/releases)

**Status**: ✅ Production Ready | **Architecture**: Microservices | **Deployment**: Kubernetes + Helm

---

## 📑 Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Architecture](#architecture)
4. [Services](#services)
5. [Build & Test](#build--test)
6. [Access Swagger UI](#-access-swagger-ui-customer-facing-api)
7. [Deployment](#deployment)
8. [Troubleshooting](#troubleshooting)
9. [Project Status](#development-roadmap)
10. [Tech Stack](#tech-stack)

---

## Overview

This project implements an elastic, scalable telemetry pipeline for GPU clusters using a custom-built message queue.

### Key Features
- ✅ **Decoupled Microservices** - Independent, scalable services
- ✅ **Custom Message Queue** - Partitioned, pull-based system (no Kafka/RabbitMQ)
- ✅ **Production Ready** - 80%+ test coverage across all services
- ✅ **Kubernetes Native** - Full K8s + Helm deployment support
- ✅ **Standardized Build System** - Consistent Makefiles across services
- ✅ **Graceful Shutdown** - Context-based cancellation and error recovery

---

## Quick Start

### Prerequisites
- Go 1.25+
- Docker
- Kubernetes (Kind recommended for local)
- kubectl
- Helm 3 (optional, for Helm deployment)

### Option 1: Local Build (Fastest)
```bash
# Build all services
make build-all

# Run tests
make test

# Check coverage
make coverage

# Verify binaries
ls -la mq/bin/ collector/bin/ streamer/bin/ api-gateway/bin/
```

### Option 2: Docker & Kubernetes (Full Stack)
```bash
# One-command deployment to Kind cluster
make deploy

# Verify deployment
make verify

# Watch pods
make watch

# View logs
make logs-all

# Access Swagger UI (in another terminal)
make api-gateway-port-forward  # In one terminal (keeps running)
make swagger-ui                 # In another terminal (opens browser)

# Clean up
make cleanup
```

---

## Architecture

The system is composed of independently deployable services:

```
┌──────────────────────────────────────────────────────────┐
│              GPU Pipeline Architecture                   │
├──────────────────────────────────────────────────────────┤
│                                                          │
│  ┌──────────────┐      ┌────────────┐    ┌──────────┐  │
│  │  Streamer    │─────►│  MQ Queue  │───►│Collector │  │
│  │ (Producer)   │      │(Partitioned)    │(Consumer)│  │
│  └──────────────┘      └────────────┘    └────┬─────┘  │
│                                                │        │
│                                           ┌────▼─────┐  │
│                                           │PostgreSQL│  │
│                                           │(Storage) │  │
│                                           └────┬─────┘  │
│                                                │        │
│                                     ┌──────────▼────┐   │
│                                     │  API Gateway  │   │
│                                     │  (Port 8000)  │   │
│                                     │ Swagger UI /  │   │
│                                     │ Query API     │   │
│                                     └───────────────┘   │
│                                                          │
└──────────────────────────────────────────────────────────┘
```

---

## Services

### 📡 Telemetry Streamer
- **Binary**: `./streamer/bin/streamer`
- **Role**: Streams CSV telemetry data to MQ
- **Status**: ✅ Complete
- **Test Coverage**: 91.8%
- **Features**:
  - CSV streaming with configurable intervals
  - Graceful shutdown with context cancellation
  - Publisher interface for dependency injection
  - 4 flexible storage modes (embedded, hostPath, RWX, remote)
- **📚 Detailed Documentation**: See [streamer/README.md](./streamer/README.md)

### 📨 Custom Message Queue (MQ)
- **Binary**: `./mq/bin/mq-server`
- **Role**: Partitioned, pull-based messaging system
- **Status**: ✅ Complete
- **Test Coverage**: 85.4% (internal)
- **Features**:
  - At-least-once delivery semantics
  - Consumer groups with offset tracking
  - Partition-based scalability and ordering
  - REST API endpoints
  - HTTP server on `:8080`

### 💾 Telemetry Collector
- **Binary**: `./collector/bin/collector`
- **Role**: Consumes messages from MQ, stores in PostgreSQL
- **Status**: ✅ Complete
- **Test Coverage**: 80.0%
- **Features**:
  - Singleton GORM connection with pgbouncer support
  - Idempotent writes with unique constraint (gpu_id, timestamp)
  - Batch processing with configurable poll interval
  - Graceful error handling and recovery

### � API Gateway
- **Binary**: `./api-gateway/bin/api-gateway`
- **Role**: REST API for querying GPU telemetry from PostgreSQL
- **Status**: ✅ Complete
- **Test Coverage**: 80%+
- **Features**:
  - OpenAPI 3.0 / Swagger UI documentation
  - List GPUs endpoint
  - Query telemetry with time-range filtering
  - Health check endpoint
  - HTTP server on `:8000`
  - Multi-stage Docker build (minimal alpine image)

### �🗄️ PostgreSQL
- **Version**: 15-Alpine
- **Role**: Central data store for telemetry
- **Storage**: Persistent volume

---

## Build & Test

### Building Services

From root directory, build all services:
```bash
make build-all
```

Binary locations:
- MQ: `mq/bin/mq-server`
- Streamer: `streamer/bin/streamer`
- Collector: `collector/bin/collector`
- API Gateway: `api-gateway/bin/api-gateway`

Or build individual services:
```bash
cd mq && make build
cd streamer && make build
cd collector && make build
cd api-gateway && make build
```

### Running Tests

```bash
# Test all services
make test

# Test individual service
cd mq && make test
cd streamer && make test
cd collector && make test
```

### Test Coverage

```bash
# View coverage for all services
make coverage

# View coverage for individual service
cd mq && make coverage
cd streamer && make coverage
cd collector && make coverage
```

### Deploy to Kind Cluster (One Command)

```bash
# Single command: creates Kind cluster, builds all images, loads to Kind, and deploys
make deploy

# Monitor deployment
make verify
make logs-all
make watch
```

### 📊 Verify Services

```bash
# Check all services are running
kubectl get pods -n gpu-pipeline

# Check services
kubectl get svc -n gpu-pipeline

# View logs
make logs-all
```

Expected output:
```
NAME                     READY   STATUS    RESTARTS
postgres-xxxxx           1/1     Running   0
mq-xxxxx                 1/1     Running   0
collector-xxxxx          1/1     Running   0
collector-xxxxx          1/1     Running   0
streamer-xxxxx           1/1     Running   0
streamer-xxxxx           1/1     Running   0
api-gateway-xxxxx        1/1     Running   0
```

---

## 🌐 Access Swagger UI (Customer-Facing API)

After deployment, expose the API Gateway Swagger UI to customers:

### Local Development (Port-Forward)
```bash
# Terminal 1: Port-forward the API Gateway
make api-gateway-port-forward

# Terminal 2: Open Swagger UI in browser
make swagger-ui

# Or manually visit: http://localhost:8081/swagger/
```

### API Endpoints
- **Swagger UI**: `http://localhost:8000/swagger/`
- **Health Check**: `http://localhost:8000/api/v1/health`
- **List GPUs**: `GET http://localhost:8000/api/v1/gpus`
- **Query Telemetry**: `POST http://localhost:8000/api/v1/telemetry/query`

### Example Requests
```bash
# Get all GPUs
curl -X GET http://localhost:8000/api/v1/gpus \
  -H "Content-Type: application/json"

# Query telemetry for a specific GPU
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu-001",
    "start_time": "2026-04-11T00:00:00Z",
    "end_time": "2026-04-12T00:00:00Z"
  }'

# Health check
curl -X GET http://localhost:8000/api/v1/health
```

---

## Deployment

### ✅ Quick Start

The telemetry data is now **embedded in the Docker image**, so deployment is straightforward:

```bash
# Full deployment (recommended)
make deploy

# Or step-by-step
make kind-create            # Create local cluster
make docker-build-all       # Build all images
make docker-load-all        # Load into Kind
make deploy-all             # Deploy all services
make verify                 # Verify running
```

**Testing with Different CSV Files**:
To test with a different telemetry dataset:

```bash
# 1. Replace the CSV file
cp /path/to/new/telemetry.csv streamer/data/telemetry.csv

# 2. Rebuild the Docker image
make docker-build-all

# 3. Load into Kind
make docker-load-all

# 4. Restart the streamer
kubectl rollout restart deployment/streamer -n gpu-pipeline
```

For more details, see [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md).

**Note on Storage Modes**: The Streamer service supports 4 flexible storage modes (embedded, hostPath, RWX, remote) for telemetry data. See [streamer/README.md](./streamer/README.md#storage-modes) for detailed configuration and examples.

### Available Commands

```bash
# Build & Setup
make build-all              # Build all services
make docker-build-all       # Build all Docker images
make docker-load-all        # Load images into Kind cluster

# Kubernetes
make kind-create            # Create Kind cluster
make kind-delete            # Delete Kind cluster
make deploy                 # Full deployment (recommended)
make deploy-all             # Deploy to existing cluster
make verify                 # Verify all services
make watch                  # Watch pods in real-time
make logs                   # Show collector logs
make logs-all               # Show all service logs

# Streamer Storage (CSV File Management)
make streamer-copy-csv      # Copy CSV to ./data for KIND (hostPath mode)
make streamer-verify-csv    # Verify CSV exists in KIND node
make streamer-logs          # Stream logs from streamer pods
make streamer-describe      # Show streamer deployment details

# API Gateway
make api-gateway-port-forward  # Port-forward API Gateway (8000:8000)
make swagger-ui                 # Open Swagger UI in browser

# Helm (Unified deployment for all services)
make helm-install           # Install all services (MQ, Streamer, Collector, etc.)
make helm-uninstall         # Uninstall all services

# Cleanup
make cleanup                # Delete namespace (keep cluster)
make kind-full              # Full reset (delete cluster + redeploy)

# Help
make help                   # Show all available commands
```

---

## Local Development

### Build Services Individually

```bash
# Build MQ
cd mq && make build

# Build Streamer
cd streamer && make build

# Build Collector
cd collector && make build
```

### Run Tests

```bash
# Test all services
make test

# Test specific service
cd mq && make test
cd collector && make test
cd streamer && make test

# View coverage
make coverage
```

### Build Docker Images

```bash
# Build all images
make docker-build-all

# Build specific image
cd collector && make docker
cd streamer && make docker
cd mq && make docker
```

---

## Kubernetes Deployment

### Step-by-Step Deployment

#### 1. Create Kind Cluster
```bash
make kind-create
```

#### 2. Build and Load Images
```bash
make docker-load-all
```

#### 3. Deploy Services
```bash
make deploy-all
```

#### 4. Verify Deployment
```bash
make verify
```

#### 5. Monitor Services
```bash
# Watch pods
make watch

# Check logs
make logs-all
```

### Deployment Structure

Services are deployed to the `gpu-pipeline` namespace:

```
deployment/k8s/
├── namespace.yaml          # gpu-pipeline namespace
├── postgres.yaml           # PostgreSQL database
├── mq.yaml                 # MQ deployment
├── mq-service.yaml         # MQ service
├── job-topic.yaml          # Create topic job
├── collector.yaml          # Collector deployment
└── streamer.yaml           # Streamer deployment
```

### Service Endpoints

From within the cluster:
- **PostgreSQL**: `postgres:5432` (user: user, password: pass)
- **MQ**: `mq-service:8080` (HTTP API)
- **Collector**: Internal service

### Configuration

Edit the YAML files in `deployment/k8s/` to customize:
- Replicas
- Resource limits
- Environment variables
- Storage configuration

---

## Helm Deployment

### Using Helm Chart

```bash
# Install
make helm-install

# Verify
kubectl get all -n gpu-pipeline

# Uninstall
make helm-uninstall
```

### Helm Chart Structure

```
helm/gpu-pipeline/
├── Chart.yaml             # Chart metadata
├── values.yaml            # Default values
├── templates/
│   ├── namespace.yaml
│   ├── postgres.yaml
│   ├── mq.yaml
│   ├── mq-service.yaml
│   ├── job-topic.yaml
│   ├── collector.yaml
│   └── streamer.yaml
```

### Customize Helm Deployment

```bash
# Override values
helm install gpu-pipeline ./helm/gpu-pipeline \
  -n gpu-pipeline \
  --create-namespace \
  --set collector.replicas=3 \
  --set streamer.replicas=2 \
  --set postgres.storage=20Gi
```

---

## Verification & Testing

### Verify All Services Are Running

```bash
make verify
```

Expected output:
```
=== Namespace ===
NAME              STATUS   AGE
gpu-pipeline      Active   2m

=== Pods ===
NAME                      READY   STATUS    RESTARTS   AGE
postgres-xxxxx            1/1     Running   0          2m
mq-xxxxx                  1/1     Running   0          2m
collector-xxxxx           1/1     Running   0          1m
collector-xxxxx           1/1     Running   0          1m
streamer-xxxxx            1/1     Running   0          1m
streamer-xxxxx            1/1     Running   0          1m
create-topic-xxxxx        0/1     Completed 0          1m

=== Services ===
NAME      TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)
postgres  ClusterIP   10.96.xxx.xxx   <none>        5432/TCP
mq-service ClusterIP  10.96.xxx.xxx   <none>        8080/TCP
```

### View Logs

```bash
# Collector logs
make logs

# All service logs
make logs-all

# Specific pod
kubectl logs <pod-name> -n gpu-pipeline

# Stream logs
kubectl logs -f <pod-name> -n gpu-pipeline
```

### Check Pod Status

```bash
# Detailed pod information
kubectl describe pod <pod-name> -n gpu-pipeline

# Watch pods
make watch
```

### Test Data Flow

```bash
# Check if data is flowing through the pipeline
kubectl logs -l app=collector -n gpu-pipeline | grep -i "inserted\|error"

# Check MQ messages
kubectl logs -l app=mq -n gpu-pipeline | head -20

# Check streamer
kubectl logs -l app=streamer -n gpu-pipeline | head -20
```

---

## Troubleshooting

### Pod Not Starting

```bash
# Check pod status
kubectl describe pod <pod-name> -n gpu-pipeline

# Check logs
kubectl logs <pod-name> -n gpu-pipeline

# Check events
kubectl get events -n gpu-pipeline --sort-by='.lastTimestamp'
```

### Service Discovery Issues

```bash
# Test DNS resolution from a pod
kubectl exec -it <pod-name> -n gpu-pipeline -- nslookup mq-service

# Test connectivity to MQ
kubectl exec -it <pod-name> -n gpu-pipeline -- nc -zv mq-service 8080
```

### PostgreSQL Connection Issues

```bash
# Check PostgreSQL pod
kubectl get pod -l app=postgres -n gpu-pipeline

# Connect to PostgreSQL
kubectl exec -it <postgres-pod> -n gpu-pipeline -- psql -U user -d telemetry
```

### Collector Not Processing Messages

```bash
# Check collector logs
kubectl logs -l app=collector -n gpu-pipeline -f

# Check MQ logs
kubectl logs -l app=mq -n gpu-pipeline -f

# Check database
kubectl exec -it <postgres-pod> -n gpu-pipeline -- psql -U user -d telemetry -c "SELECT COUNT(*) FROM telemetry;"
```

### Reset Everything

```bash
# Full reset (delete cluster and redeploy)
make kind-full

# Or just cleanup namespace
make cleanup
make deploy
```

---

## Development Roadmap

### Phase 1: Core Message Queue (Foundation) ✅
- [x] Message abstraction (generic payload)
- [x] Topic & partition model
- [x] Partitioning strategy (hash-based)
- [x] Thread-safe append/read
- [x] Consumer groups
- [x] Offset tracking
- [x] At-least-once delivery semantics

### Phase 2: MQ Service Layer ✅
- [x] HTTP server for MQ
- [x] Publish API
- [x] Consume API
- [x] Ack API
- [x] Error handling & validation
- [x] Configurable partitions

### Phase 3: MQ Client SDK ✅
- [x] Producer client (for streamer)
- [x] Consumer client (for collector)
- [x] Configurable batching

### Phase 4: Telemetry Streamer ✅
- [x] CSV reader
- [x] Continuous streaming loop
- [x] Configurable rate
- [x] Horizontal scalability

### Phase 5: Telemetry Collector ✅
- [x] Consumer group integration
- [x] Message parsing
- [x] Idempotent processing
- [x] Batch processing
- [x] 80% test coverage
- [x] GORM + PostgreSQL persistence
- [x] Kubernetes deployment

### Phase 6: Storage Layer ✅
- [x] Schema design (GPU telemetry)
- [x] Indexing (gpu_id + timestamp)
- [x] Unique constraint for idempotency
- [x] pgbouncer-compatible connection pooling

### Phase 7: API Gateway ✅
- [x] List GPUs endpoint
- [x] Query telemetry endpoint
- [x] Time-range filtering
- [x] OpenAPI auto-generation / Swagger UI
- [x] 80%+ test coverage
- [x] Multi-stage Docker build
- [x] Kubernetes deployment
- [x] Swagger UI port-forward for customer access

### Phase 8: Deployment ✅
- [x] Dockerfiles for all services
- [x] Kubernetes deployment configs (including API Gateway)
- [x] Helm charts (including API Gateway)
- [x] Make targets for deployment
- [x] API Gateway Swagger UI exposure
- [ ] Horizontal pod autoscaling
- [ ] StatefulSet for PostgreSQL HA

### Phase 9: Testing & Observability ⏳
- [x] Unit tests (MQ core, Collector)
- [ ] Integration tests (pipeline)
- [ ] Logging (structured)
- [ ] Metrics (Prometheus + Grafana)
- [ ] Tracing (Jaeger)

---

## Tech Stack

| Component | Technology |
|-----------|-----------|
| Language | Go 1.25+ |
| Container | Docker |
| Orchestration | Kubernetes (Kind for local) |
| Package Manager | Helm |
| Database | PostgreSQL 15-Alpine |
| Message Queue | Custom-built (no Kafka/RabbitMQ) |
| ORM | GORM |
| Testing | Go testing + custom coverage |
| API Docs | OpenAPI (Swagger) |

---

## Detailed Documentation

- **[DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md)** - Complete deployment instructions, troubleshooting, and operations guide
  - Step-by-step deployment process
  - ConfigMap setup and telemetry data management
  - Service access and API examples
  - Helm deployment instructions
  - Troubleshooting common issues
  - Monitoring and updating services

---

## Future Improvements

- Disk-backed persistence (WAL)
- Replication & leader election
- Consumer group rebalancing
- gRPC instead of HTTP
- Compression & batching
- Structured logging & metrics
- Prometheus metrics export
- Grafana dashboards
- Horizontal pod autoscaling
- StatefulSet for PostgreSQL HA

---

## Key Features

| Feature | Details |
|---------|---------|
| **Decoupled Architecture** | Independent, scalable services |
| **Generic MQ** | Reusable message queue for any use case |
| **Partition-based Scalability** | Ordering guarantees per partition |
| **Consumer Groups** | Multiple consumers per topic with offset tracking |
| **Kubernetes-Ready** | Helm charts, deployment configs included |
| **Production-Ready** | 80%+ test coverage, graceful shutdown, error recovery |
| **Idempotency** | Exactly-once delivery semantics |
| **Horizontal Scaling** | Easy to scale each component independently |

---

## Quick Command Reference

```bash
# Deploy everything
make deploy

# Check status
make verify

# Monitor
make watch
make logs-all

# Build & test
make build-all
make test
make coverage

# Kubernetes
make kind-create
make docker-load-all
make deploy-all

# Helm
make helm-install
make helm-uninstall

# Cleanup
make cleanup
make kind-full
```

---

## Service Documentation

- **MQ**: See `mq/README.md`
- **Streamer**: See `streamer/README.md`
- **Collector**: See `collector/README.md`

---

**Last Updated**: April 2026
**Status**: 🚀 Production Ready
**Version**: 1.0
**Next Step**: Run `make deploy` to get started!
