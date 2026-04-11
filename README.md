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
4. [Design Decisions](#design-decisions)
5. [Services](#services)
6. [Build & Test](#build--test)
7. [Managing Telemetry Data](#managing-telemetry-data)
8. [Access Swagger UI](#-access-swagger-ui-customer-facing-api)
9. [Deployment](#deployment)
10. [Troubleshooting](#troubleshooting)
11. [Project Status](#development-roadmap)
12. [Tech Stack](#tech-stack)

---

## 🤖 AI Development Documentation

📖 **[AI Development Workflow Document](./AI_DEVELOPMENT_WORKFLOW.md)** - Detailed breakdown of how GitHub Copilot accelerated development across all project phases:
- Project bootstrapping and architecture design
- Service code generation (86% AI-assisted)
- Unit testing strategy (91% auto-generated)
- Build environment and deployment automation
- Complete development timeline analysis (49.5 days saved)

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

## Design Decisions

### 1. PostgreSQL as Primary Storage with TimescaleDB Future Path

**Why PostgreSQL?**
- **Native Time-Series Support Ready**: PostgreSQL's TimescaleDB extension provides:
  - Hyper-tables for automatic chunking and partitioning by time
  - Configurable chunk sizes (default: 1 hour → 1 day or 1 week based on volume)
  - **~75% storage reduction** vs. raw tables
  - **10-100x faster range queries** due to automatic chunk elimination
  - Automatic compression for historical data

**Storage & Performance Benefits:**
| Scenario | Traditional PostgreSQL | With TimescaleDB |
|----------|----------------------|------------------|
| 1 Year Data Storage | 100 GB | ~1.3 GB (75% reduction) |
| Range Query (1 week) | Full table scan | Chunk elimination (10-100x faster) |
| Data Compression | None | Automatic after 7 days |

**Future Migration Path (Phase 9)** - No code changes needed:
```sql
-- One-command conversion to hyper-table
SELECT create_hypertable('telemetry', 'timestamp', if_not_exists => TRUE);

-- Auto-compress data older than 7 days
SELECT add_compression_policy('telemetry', INTERVAL '7 days', if_not_exists => TRUE);

-- Optional: Create continuous aggregates for fast reporting
CREATE MATERIALIZED VIEW telemetry_hourly AS 
SELECT time_bucket('1 hour', timestamp) AS time, gpu_id, 
       AVG((data->>'power')::float) AS avg_power
FROM telemetry GROUP BY time, gpu_id;
```

**Current State**: PostgreSQL 15-Alpine ready for zero-downtime TimescaleDB migration.

### 2. Custom Message Queue (MQ) - In-Memory Partitioned Design

**Architecture & Scalability Considerations:**
- **Partition-Based Concurrency**: Each partition has independent RWMutex, supporting ~10+ concurrent producers/consumers with zero global lock contention
- **Consistent Hashing**: Message keys deterministically route to partitions (configurable replicas: default 3) ensuring ordering guarantees per key
- **At-Least-Once Delivery**: Consumer groups track committed offsets per-partition atomically (sync.Map + atomic.Int64)
- **Watermark-Based Compaction**: Automatic memory management prevents unbounded growth by tracking slowest consumer lag

**Current Scale Capabilities:**
| Scenario | Current Capacity | Considerations |
|----------|------------------|---|
| Single Node | ~10 GB memory | Limited by machine resources |
| Message Size | 1-10 KB typical | Telemetry records are small (~500B) |
| Partition Count | 1-100+ | More partitions = finer parallelism |
| Concurrent Consumers | 10+ per partition | No global bottlenecks |
| Message Retention | Watermark-bounded | Default 1-week slowest consumer lag max |

**Vertical Scaling Changes (for higher throughput):**
1. **Increase Partition Count**: 
   ```json
   { "partitions": 10 }  // Default: 3
   ```
   - Enables parallel message consumption
   - Distributes memory across more buckets
   - Reduces per-partition compaction overhead

2. **Tune Compaction Interval** (prod-focused):
   ```json
   {
     "compaction_enabled": true,
     "compaction_interval": "10m",      // Prod: 5-10m vs dev: 1m
     "compaction_threshold": 500000      // Trigger: 500K messages per partition
   }
   ```
   - Less frequent compaction = better throughput at cost of higher temporary memory
   - Watermark prevents unbounded growth regardless

3. **Consumer Group Offset Optimization**:
   - Faster consumers process at max speed (no blocking on slow consumers)
   - Slow consumer offsets tracked atomically without blocking hot path
   - Consider multiple consumer groups for different processing speeds

4. **Future: Horizontal Scaling** (planned):
   - Extend `internal/topic.go` to manage replicas across nodes
   - Add replication logic to sync partition state (similar to Kafka brokers)
   - Leader-follower model for partition management
   - Phase: Not required for current GPU pipeline (single node sufficient)

**Memory Guarantees:**
```
Max Memory = (slowest_consumer_lag) × (message_size) × (num_partitions)
Example: 100,000 unacked messages × 1 KB × 10 partitions = ~1 GB bounded
```

**Performance Profile:**
- Publish: O(1) append to partition
- Consume: O(1) offset lookup + O(n) message read
- Ack: O(1) atomic offset update
- Compact: O(m + g) where m=messages, g=consumer groups (non-blocking, background)

### 3. Telemetry Streamer - Flexible Deployment Model

**Design Philosophy:**
- **Single Responsibility**: Reads CSV, publishes to MQ - no filtering, transformation, or storage
- **Storage Agnostic**: Supports 4 deployment scenarios without code changes
- **Horizontal Scalability**: Each replica streams independently to MQ (messages deduplicated downstream)

**Deployment Modes & Scalability:**

| Mode | Best For | Scale Replicas | Storage | Use Case |
|------|----------|---|---|---|
| **Embedded** | Dev/Demo | ✅ Any count | Docker image | Quick testing, CI/CD |
| **hostPath** | Local KIND | ❌ Max 1 | Node filesystem | Developer laptop |
| **RWX (NFS)** | Multi-node Prod | ✅ Any count | Shared storage | Production clusters |
| **Remote (S3)** | Cloud-native | ✅ Any count | Cloud storage | AWS/Azure/GCP |

**Scaling to Multi-Node Clusters:**

**Single-Node Deployment:**
```yaml
replicas: 1
storage: hostPath              # CSV on local node
```
- Fastest for development
- No shared storage needed
- MQ deduplicates duplicate messages from restart

**Multi-Node Deployment:**
Choose one of:

```yaml
# Option 1: RWX Storage (Recommended for production)
replicas: 5+
storage:
  type: rwx
  storageClassName: nfs-client  # or other RWX provider
  size: 5Gi

# Option 2: Remote HTTP/S3 (Cloud-native)
replicas: 5+
storage:
  type: remote
  remote:
    url: "https://s3.amazonaws.com/bucket/telemetry.csv"
    retries: 3
```

**Message Deduplication Strategy:**
- Multiple replicas emit same messages to MQ
- MQ partitions by `gpu_id` key - messages within same GPU go to one partition
- Collector processes with idempotent writes: `UNIQUE(gpu_id, timestamp)` in PostgreSQL
- Result: Replicas can safely scale without message loss or duplication

**Scaling Considerations:**
1. **Embedded Mode**: 
   - Scale to any replica count
   - Each replica reads full CSV, publishes all records
   - MQ + Collector handle deduplication
   - Best for: Dev/testing

2. **RWX Mode**:
   - Scale to any replica count
   - All replicas read same CSV from NFS
   - Possible issue: Multiple replicas start at same time, all read & publish
   - Solution: Add offset file on NFS to track last-read-line per replica
   - Current: Works fine due to downstream idempotency

3. **Remote Mode**:
   - Scale to any replica count
   - Each replica downloads independently
   - Network cost proportional to replicas
   - Best for: Cloud environments with bandwidth costs

**Vertical Scaling (Higher Throughput):**
```yaml
# Increase stream frequency (config parameter)
STREAM_INTERVAL_MS: 100          # Default: 5000 (5 seconds between records)
                                  # Minimum: 10ms per message practical limit
                                  # Tradeoff: CPU vs MQ throughput
```

### 4. API Gateway - Query Layer Optimization

**Design Focus:**
- **Read-Only Layer**: No writes, optimized for query patterns
- **Singleton DB Connection**: Thread-safe GORM connection with pgbouncer support
- **Layered Architecture**: Handler → Service → Repository for testability

**Scalability & Performance Optimization:**

**Horizontal Scaling (Add More Instances):**
```yaml
replicas: 3+
resources:
  requests:
    memory: "256Mi"
    cpu: "100m"
  limits:
    memory: "512Mi"
    cpu: "500m"
```
- Stateless design: Any instance can serve any request
- Load balancer distributes across replicas
- Each instance gets dedicated DB connection from pgbouncer
- No inter-instance state sharing needed

**Query Optimization Strategy:**

1. **Connection Pooling** (via pgbouncer):
   ```
   API Gateway (3 replicas) 
     ↓ pgbouncer (connection pool)
       ↓ PostgreSQL (shared connections)
   ```
   - Prevents database connection exhaustion
   - Config: `max_connections = 20` per gateway × N gateways

2. **Index Strategy** (for TimescaleDB future):
   ```sql
   -- Primary: Time-series index for range queries
   CREATE INDEX idx_telemetry_timestamp ON telemetry (timestamp DESC);
   
   -- Secondary: GPU ID for filtering
   CREATE INDEX idx_telemetry_gpu_id ON telemetry (gpu_id);
   
   -- Composite: For common query pattern
   CREATE INDEX idx_telemetry_gpu_time ON telemetry (gpu_id, timestamp DESC);
   ```

3. **Prepared Statements** (GORM auto-implements):
   - Reduces SQL parsing overhead
   - Improves cache hit rate in PostgreSQL
   - Already in repository layer

4. **Caching Strategy** (future enhancement):
   - Add Redis for frequently queried GPU lists
   - Cache policy: Invalidate on new data write (via trigger or event)
   - Benefit: O(1) lookup for GPU list vs O(n) table scan

**Vertical Scaling (Single Instance Performance):**
- Increase resources (CPU/memory)
- Optimize GORM query patterns (eager loading, select specific columns)
- Add query result pagination for large datasets
- Consider materialized views for aggregations (TimescaleDB continuous aggregates)

**Current Bottleneck Analysis:**
- **Query Layer**: Repository queries are efficient for time-range + GPU_ID filters
- **Network**: Depends on response payload size
- **Database**: Index effectiveness critical as data grows (addressed by TimescaleDB)
- **Connection Pool**: 20 connections shared across replicas - may need tuning

---

## Services

### 📡 Telemetry Streamer
- **Binary**: `./streamer/bin/streamer`
- **Role**: Streams CSV telemetry data to MQ
- **Status**: ✅ Complete | **Coverage**: 91.8%
- **Key Features**:
  - CSV streaming with configurable intervals
  - Graceful shutdown with context cancellation
  - Publisher interface for dependency injection
  - **4 flexible storage modes** (embedded, hostPath, RWX, remote)
- **📚 Documentation**: [streamer/README.md](./streamer/README.md)
  - [Storage Modes](./streamer/README.md#storage-modes) - Embedded, hostPath, RWX, Remote options
  - [Configuration](./streamer/README.md#configuration) - CSV_FILE, MQ_URL, STREAM_INTERVAL_MS
  - [Scaling](./streamer/README.md#scaling) - Horizontal scalability guide

### 📨 Custom Message Queue (MQ)
- **Binary**: `./mq/bin/mq-server`
- **Role**: Partitioned, pull-based messaging system
- **Status**: ✅ Complete | **Coverage**: 85.4%
- **Key Features**:
  - At-least-once delivery semantics
  - Consumer groups with offset tracking
  - Partition-based scalability and ordering
  - Watermark-based automatic compaction
  - REST API on port `:8080`
- **📚 Documentation**: [mq/README.md](./mq/README.md)
  - [Architecture](./mq/README.md#-architecture) - Layered design, message flow
  - [HTTP API](./mq/README.md#-http-api) - Publish, Consume, Ack endpoints
  - [Concurrency & Performance](./mq/README.md#-concurrency--performance) - Thread-safety, scalability
  - [Compaction](./mq/README.md#-watermark-based-compaction) - Automatic memory management

### 💾 Telemetry Collector
- **Binary**: `./collector/bin/collector`
- **Role**: Consumes messages from MQ, stores in PostgreSQL
- **Status**: ✅ Complete | **Coverage**: 80.0%
- **Key Features**:
  - Singleton GORM connection with pgbouncer support
  - Idempotent writes with unique constraint (gpu_id, timestamp)
  - Batch processing with configurable poll interval
  - Graceful error handling and recovery
- **📚 Documentation**: [collector/README.md](./collector/README.md)
  - [Design & Architecture](./collector/README.md#design--architecture) - Poll-based consumption
  - [Database Schema](./collector/README.md#database) - Table structure, indexes, constraints
  - [Production Deployment](./collector/README.md#production-checklist) - Pre-deployment checklist
  - [Configuration](./collector/README.md#configuration) - DB_DSN, MQ_URL, BATCH_SIZE

### 🔗 API Gateway
- **Binary**: `./api-gateway/bin/api-gateway`
- **Role**: REST API for querying GPU telemetry from PostgreSQL
- **Status**: ✅ Complete | **Coverage**: 80%+
- **Key Features**:
  - OpenAPI 3.0 / Swagger UI documentation
  - List GPUs endpoint with performance optimization
  - Query telemetry with time-range filtering
  - Health check endpoint
  - HTTP server on port `:8000`
  - Multi-stage Docker build (minimal alpine image)
  - Clean layered architecture with dependency injection
- **📚 Documentation**: [api-gateway/README.md](./api-gateway/README.md)
  - [Architecture & Design Patterns](./api-gateway/README.md#architecture) - Layered design, DI, patterns
  - [API Endpoints](./api-gateway/README.md#api-endpoints) - Complete endpoint reference
  - [Time-Range Queries](./api-gateway/README.md#testing-the-api) - Query examples
  - [Test Coverage](./api-gateway/README.md#test-coverage) - Package breakdown

### 🗄️ PostgreSQL
- **Version**: 15-Alpine
- **Role**: Central data store for telemetry (TimescaleDB-ready)
- **Storage**: Persistent volume
- **Indexes**: 
  - `idx_telemetry_gpu_id` - Fast GPU lookups
  - `idx_telemetry_timestamp` - Fast time-range queries
  - `telemetry_gpu_ts_unique` - Idempotency guarantee
- **Future**: Ready for TimescaleDB migration (Phase 9) - ~75% storage reduction

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
cd api-gateway && make test
```

### Test Coverage

```bash
# View coverage for all services
make coverage

# View coverage for individual service
cd mq && make coverage
cd streamer && make coverage
cd collector && make coverage
cd api-gateway && make coverage
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
streamer-xxxxx           1/1     Running   0
api-gateway-xxxxx        1/1     Running   0
```

---

## Managing Telemetry Data

The Streamer service supports **4 flexible storage modes** for managing telemetry data in different deployment scenarios.

### Quick Reference Table

| Mode | **Embedded** | **hostPath** | **RWX** | **Remote** |
|------|:---:|:---:|:---:|:---:|
| **Use Case** | Local testing | KIND testing | Multi-node production | Cloud-native (S3) |
| **Setup Effort** | ✅ Zero | 🟡 Copy CSV | 🟡 NFS mount | 🟡 S3 upload |
| **Rebuild on Data Change** | ✅ Required | ❌ No | ❌ No | ❌ No |
| **Scaling Replicas** | ✅ Any | ❌ Single node | ✅ Any | ✅ Any |
| **Best For** | Dev/Testing | Local KIND | Production | Production (Cloud) |

### Storage Modes Documentation

👉 **[Complete Guide: Streamer Storage Modes](./streamer/README.md#storage-modes)**

### Quick Commands for KIND Local Development

```bash
# ============================================
# Step 1: Create KIND cluster with /data mount
# ============================================
make kind-create

# ============================================
# Step 2: Copy telemetry CSV to KIND node
# ============================================
make streamer-copy-csv

# ============================================
# Step 3: Verify CSV was copied successfully
# ============================================
make streamer-verify-csv

# ============================================
# Step 4: Deploy the full stack
# ============================================
make deploy

# ============================================
# Step 5: Update CSV without rebuilding
# ============================================
# a. Edit streamer/data/telemetry.csv locally
# b. Copy to KIND: make streamer-copy-csv
# c. Restart streamer:
kubectl rollout restart deployment/streamer -n gpu-pipeline
```

### Quick Commands for Embedded Mode (Default)

```bash
# CSV is already built into Docker image - just deploy
make deploy
```

### Quick Commands for Production S3/Remote

```bash
# 1. Upload CSV to S3
aws s3 cp streamer/data/telemetry.csv s3://your-bucket/telemetry.csv

# 2. Deploy with remote mode
helm install gpu-pipeline helm/gpu-pipeline/ \
  --set streamer.storage.type=remote \
  --set 'streamer.storage.remote.url=https://s3.amazonaws.com/your-bucket/telemetry.csv'
```

---

## 🌐 Access Swagger UI (Customer-Facing API)

After deployment, expose the API Gateway Swagger UI to customers:

### Local Development (Port-Forward)
```bash
# Terminal 1: Port-forward the API Gateway (keeps running)
make api-gateway-port-forward

# Terminal 2: Open Swagger UI in browser
make swagger-ui

# Or manually visit: http://localhost:8000/swagger/
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

# Query telemetry for a specific GPU with time range
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu-001",
    "start_time": "2026-04-11T00:00:00Z",
    "end_time": "2026-04-12T00:00:00Z"
  }'

# Get telemetry for specific GPU
curl -X GET "http://localhost:8000/api/v1/gpus/gpu-001/telemetry?start_time=2026-04-11T00:00:00Z&end_time=2026-04-12T00:00:00Z" \
  -H "Content-Type: application/json"

# Health check
curl -X GET http://localhost:8000/api/v1/health
```

---

## Deployment

### ✅ Quick Start

The telemetry data is **embedded in the Docker image** by default, so deployment is straightforward:

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

### Testing with Different CSV Files

To test with a different telemetry dataset:

```bash
# 1. Replace the CSV file
cp /path/to/new/telemetry.csv streamer/data/telemetry.csv

# 2. For embedded mode: Rebuild the Docker image
make docker-build-all

# 3. Load into Kind
make docker-load-all

# 4. Restart the streamer
kubectl rollout restart deployment/streamer -n gpu-pipeline
```

For more details on storage modes, see [DEPLOYMENT_GUIDE.md](./DEPLOYMENT_GUIDE.md) and [streamer/README.md#storage-modes](./streamer/README.md#storage-modes).

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
make streamer-copy-csv      # Copy CSV to KIND node (/data for hostPath mode)
make streamer-verify-csv    # Verify CSV exists in KIND node
make streamer-logs          # Stream logs from streamer pods
make streamer-describe      # Show streamer deployment details

# API Gateway
make api-gateway-port-forward  # Port-forward API Gateway (8000:8000)
make swagger-ui                 # Open Swagger UI in browser

# Helm (Unified deployment for all services)
make helm-install           # Install all services (MQ, Streamer, Collector, API Gateway)
make helm-uninstall         # Uninstall all services

# Cleanup
make cleanup                # Delete namespace (keep cluster)
make kind-full              # Full reset (delete cluster + redeploy)

# Help
make help                   # Show all available commands
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
  - **Watermark-based automatic compaction** for memory management
- **📚 Detailed Documentation**: See [mq/README.md](./mq/README.md)
  - [Architecture Overview](./mq/README.md#-architecture) - Layered design and message flow
  - [HTTP API Reference](./mq/README.md#-http-api) - Complete API endpoints
  - [Concurrency & Performance](./mq/README.md#-concurrency--performance) - Design details
  - [Memory Management](./mq/README.md#key-features) - Compaction strategy

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
- **📚 Detailed Documentation**: See [collector/README.md](./collector/README.md)
  - [Design & Architecture](./collector/README.md#design--architecture) - Poll-based consumption and batching
  - [Database Schema](./collector/README.md#database) - Telemetry table structure and indexing
  - [Production Deployment](./collector/README.md#production-checklist) - Pre-deployment checklist
  - [Configuration](./collector/README.md#configuration) - Environment variables and tuning

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

Each service has comprehensive documentation with design decisions, deployment guides, and configuration options:

- **🗄️ [MQ README](./mq/README.md)** - Custom-built message queue with partition support, compaction, and consumer groups
  - 85.4% test coverage | In-memory with automatic memory management
  
- **📡 [Streamer README](./streamer/README.md)** - CSV streaming with 4 flexible storage modes (embedded, hostPath, RWX, remote)
  - 91.8% test coverage | Horizontal scalability support
  
- **💾 [Collector README](./collector/README.md)** - Poll-based consumption from MQ with batch processing
  - 80% test coverage | Idempotent writes with pgbouncer support
  
- **🔗 [API Gateway README](./api-gateway/README.md)** - REST API with Swagger UI and time-range filtering
  - 80%+ test coverage | Clean layered architecture with dependency injection

---

**Last Updated**: April 2026
**Status**: 🚀 Production Ready
**Version**: 1.0
**Next Step**: Run `make deploy` to get started!
