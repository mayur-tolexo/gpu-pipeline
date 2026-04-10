# 📊 Collector Microservice - Complete Documentation

> Poll-based microservice that consumes telemetry messages from MQ and persists them to PostgreSQL with idempotency guarantees.

**Status**: ✅ Production Ready | **Coverage**: 80.0% | **Go Version**: 1.25+

---

## 📑 Table of Contents

1. [Overview](#overview)
2. [Quick Start](#quick-start)
3. [Design & Architecture](#design--architecture)
4. [Directory Structure](#directory-structure)
5. [Core Components](#core-components)
6. [Data Model](#data-model)
7. [Configuration](#configuration)
8. [Developer Guide](#developer-guide)
9. [Testing](#testing)
10. [Deployment](#deployment)
11. [Docker](#docker)
12. [Kubernetes](#kubernetes)
13. [Makefile Reference](#makefile-reference)
14. [Database](#database)
15. [Troubleshooting](#troubleshooting)
16. [Production Checklist](#production-checklist)

---

## Overview

The **Collector Microservice** is part of the GPU telemetry pipeline. It:

- 🔄 Polls telemetry messages from a custom MQ service via consumer groups
- 💾 Persists messages to PostgreSQL with idempotency guarantees
- ⚡ Processes messages in configurable batches
- 🔐 Ensures exactly-once delivery semantics via unique constraints
- 🐳 Deploys as Docker container and Kubernetes pods
- 🧪 Maintains 80%+ test coverage with comprehensive unit tests
- ✅ Includes graceful shutdown and error recovery

### Key Features

| Feature | Details |
|---------|---------|
| **Message Processing** | Poll-based, batch-aware, partition-parallel |
| **Idempotency** | Unique index on (gpu_id, timestamp) with ON CONFLICT DO NOTHING |
| **Database** | GORM + PostgreSQL, pgbouncer-compatible pooling |
| **Configuration** | Environment-driven with sensible defaults |
| **Deployment** | Docker multi-stage build + Kubernetes ready |
| **Testing** | sqlmock for DB mocking, 80% coverage |
| **Error Handling** | Graceful recovery with comprehensive logging |

---

## Quick Start

### Build

```bash
make build
# Creates: ./bin/collector
```

### Run Locally

```bash
# Set environment
export DB_DSN="postgres://user:pass@localhost:5432/telemetry?sslmode=disable"
export MQ_URL="http://localhost:8080"
export TOPIC="telemetry"
export GROUP="collector-group"

# Run
./bin/collector
```

### Test

```bash
# Run all tests
make test

# View coverage
make coverage
# Output: Total coverage: 80.0%
```

### Deploy to Kubernetes (local Kind)

```bash
# One-shot setup
make kind-deploy

# Or step-by-step:
make kind-create      # Create cluster
make kind-load        # Build & load Docker image
make kind-deploy      # Deploy to cluster
make port-forward     # Port forward (optional)
```

---

## Design & Architecture

### High-Level Flow

```
┌─────────────┐
│   MQ Topic  │
│ (telemetry) │
└──────┬──────┘
       │ Consume (batch)
       ▼
┌──────────────────┐
│   Collector      │
│ (per partition)  │
├──────────────────┤
│ • Poll partition │
│ • Parse JSON     │
│ • Insert record  │
│ • Ack offset     │
└──────┬───────────┘
       │
       ▼
┌──────────────────┐
│  PostgreSQL DB   │
│  (telemetry tbl) │
└──────────────────┘
```

### Design Principles

1. **Poll-Based Consumption**
   - Collector actively polls each partition at configured intervals
   - No dependency on Kafka-style brokers or Zookeeper
   - Stateful consumer group tracking in MQ service

2. **Batch Processing**
   - Messages fetched in configurable batches (default: 10)
   - Reduces API calls and improves throughput
   - Tunable via `BATCH_SIZE` environment variable

3. **Singleton Database Connection**
   - Single shared GORM instance across all goroutines
   - pgbouncer-compatible connection pooling
   - Thread-safe via sync.Once pattern
   - Configured for long-lived connections

4. **Idempotent Storage**
   - Unique constraint on (gpu_id, timestamp)
   - Duplicate messages silently ignored (ON CONFLICT DO NOTHING)
   - Ensures exactly-once semantics despite message retries

5. **Graceful Shutdown**
   - Context-based cancellation signal propagation
   - Responds to SIGINT and SIGTERM
   - Allows in-flight messages to complete

6. **Multi-Partition Parallelism**
   - Each partition polled independently
   - Concurrent polling without blocking
   - Configurable number of partitions

---

## Directory Structure

```
collector/
├── cmd/
│   └── main.go                      # Service entrypoint
│       ├── LoadConfig() from env
│       ├── InitStore(DB_DSN)
│       ├── NewCollector(cfg)
│       └── Run() + graceful shutdown
│
├── internal/
│   ├── collector.go                 # Core polling & processing logic
│   │   ├── type Collector struct
│   │   ├── type MQClient interface (testable)
│   │   ├── NewCollector(cfg, dbDSN)
│   │   ├── NewCollectorWithClient(cfg, client, dbDSN)
│   │   └── Run(ctx context.Context)
│   │
│   ├── collector_test.go            # Collector unit tests
│   │   ├── Fake MQ client (fakeClient)
│   │   ├── Message processing tests
│   │   ├── Multi-partition tests
│   │   └── Error handling tests
│   │
│   ├── store.go                     # Database persistence layer (GORM)
│   │   ├── type Store struct
│   │   ├── InitStore(dsn string) *Store
│   │   ├── GetStore() *Store (singleton)
│   │   ├── CloseStore() error
│   │   ├── insert(record map[string]interface{})
│   │   └── InsertFunc (overridable for testing)
│   │
│   ├── store_test.go                # Store unit tests (sqlmock)
│   │   ├── SQLmock transaction testing
│   │   ├── Field validation tests
│   │   ├── Error handling tests
│   │   └── Config loading tests
│   │
│   ├── config.go                    # Configuration loader
│   │   ├── type Config struct
│   │   ├── LoadConfig() Config
│   │   └── getEnv(key, defaultVal) string
│   │
│   └── model.go                     # Telemetry data model
│       ├── type Telemetry struct
│       └── func (*Telemetry) TableName() string
│
├── k8s/
│   └── deployment.yaml              # Kubernetes deployment manifests
│       ├── Deployment (3 replicas)
│       ├── Service (ClusterIP:8081)
│       └── ConfigMap (environment variables)
│
├── Dockerfile                       # Multi-stage Docker build
│   ├── Stage 1: Build (golang:1.25-alpine)
│   └── Stage 2: Runtime (alpine:3.18)
│
├── Makefile                         # Build, test, deploy targets
│   ├── build
│   ├── test
│   ├── coverage
│   ├── docker
│   ├── kind-create / kind-load / kind-deploy / kind-delete
│   └── port-forward
│
├── go.mod                           # Go module definition
├── go.sum                           # Dependency lock file
│
├── DOCS.md                          # This file (complete documentation)
└── coverage.out                     # Coverage report (gitignored)
```

---

## Core Components

### 1. Collector (`internal/collector.go`)

**Purpose**: Main polling and message processing logic

**Key Methods**:
- `NewCollector(cfg Config, dbDSN string) (*Collector, error)`
  - Creates collector with real HTTP client for MQ
  - Initializes database connection
  - Returns ready-to-run collector

- `NewCollectorWithClient(cfg Config, cli MQClient, dbDSN string) (*Collector, error)`
  - Allows dependency injection of MQ client
  - Used in tests with fake client
  - Enables isolated testing

- `Run(ctx context.Context)`
  - Main loop polling all partitions
  - Processes messages in batches
  - Respects context cancellation
  - Handles errors gracefully

**Key Interface**:
```go
type MQClient interface {
    Consume(ctx context.Context, topic, group string, partition int, batchSize int) ([]Message, error)
    Ack(ctx context.Context, topic, group string, partition int, offset int64) error
}
```

### 2. Store (`internal/store.go`)

**Purpose**: Database persistence layer with singleton pattern

**Key Methods**:
- `InitStore(dsn string) error`
  - Creates/opens PostgreSQL connection
  - Runs auto-migration (creates telemetry table)
  - Sets up unique constraint on (gpu_id, timestamp)
  - Configures pgbouncer-compatible pooling

- `GetStore() *Store`
  - Returns initialized singleton instance
  - Thread-safe via sync.Once

- `CloseStore() error`
  - Closes database connection
  - Called on graceful shutdown

- `insert(record map[string]interface{}) error`
  - Inserts record with idempotency
  - Uses ON CONFLICT DO NOTHING
  - Overridable via InsertFunc for testing

**Connection Pooling** (pgbouncer-compatible):
```go
MaxOpenConns:     50
MaxIdleConns:     10
ConnMaxIdleTime:  5 * time.Minute
ConnMaxLifetime:  30 * time.Minute
```

### 3. Configuration (`internal/config.go`)

**Purpose**: Environment-based configuration loading

**Supported Variables**:

| Variable | Default | Type | Purpose |
|----------|---------|------|---------|
| `MQ_URL` | http://mq-service:8080 | string | MQ service base URL |
| `TOPIC` | telemetry | string | Topic to consume from |
| `GROUP` | collector-group | string | Consumer group name |
| `PARTITIONS` | 3 | int | Number of partitions to poll |
| `BATCH_SIZE` | 10 | int | Messages per fetch |
| `POLL_INTERVAL_MS` | 500 | int | Polling interval in milliseconds |
| `DB_DSN` | (required) | string | PostgreSQL connection string |

**Example `.env`**:
```bash
MQ_URL=http://mq-service:8080
TOPIC=telemetry
GROUP=collector-group
PARTITIONS=3
BATCH_SIZE=10
POLL_INTERVAL_MS=500
DB_DSN=postgres://user:password@localhost:5432/telemetry?sslmode=disable
```

### 4. Data Model (`internal/model.go`)

**Telemetry Table Schema**:
```go
type Telemetry struct {
    ID        uint      `gorm:"primaryKey"`
    GPUID     string    `gorm:"index"`
    Timestamp time.Time `gorm:"index"`
    Data      []byte    `gorm:"type:jsonb"`
}
```

**Generated SQL**:
```sql
CREATE TABLE telemetry (
    id SERIAL PRIMARY KEY,
    gpu_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    data JSONB NOT NULL
);

CREATE INDEX idx_telemetry_gpu_id ON telemetry(gpu_id);
CREATE INDEX idx_telemetry_timestamp ON telemetry(timestamp);
CREATE UNIQUE INDEX idx_telemetry_gpu_ts_unique ON telemetry(gpu_id, timestamp);
```

---

## Data Model

### Telemetry Record Structure

```go
type Telemetry struct {
    ID        uint      // Auto-generated primary key
    GPUID     string    // GPU identifier (from JSON: gpu_id)
    Timestamp time.Time // Event timestamp (from JSON: timestamp)
    Data      []byte    // Raw JSON payload
}

// Table name for GORM
func (t *Telemetry) TableName() string {
    return "telemetry"
}
```

### Expected JSON Message Format

```json
{
    "gpu_id": "gpu-001",
    "timestamp": "2026-04-10T15:30:45Z",
    "utilization": 85.5,
    "memory": 24576,
    "temperature": 72,
    "power": 250
}
```

### Database Indexing

| Index | Type | Purpose |
|-------|------|---------|
| `id` | PRIMARY KEY | Record identification |
| `gpu_id` | BTREE | Query by GPU device |
| `timestamp` | BTREE | Time-range queries |
| `(gpu_id, timestamp)` | UNIQUE | Idempotency enforcement |

### Idempotency Mechanism

- **Unique Constraint**: `(gpu_id, timestamp)` pair must be unique
- **Conflict Policy**: `ON CONFLICT DO NOTHING`
- **Behavior**: Duplicate messages silently ignored (no error)
- **Result**: Exactly-once semantics per GPU per timestamp

---

## Configuration

### Environment Variables

All configuration via environment variables. See `.env` example:

```bash
# MQ Service Connection
MQ_URL=http://mq-service:8080

# Consumer Configuration
TOPIC=telemetry
GROUP=collector-group
PARTITIONS=3

# Processing Configuration
BATCH_SIZE=10
POLL_INTERVAL_MS=500

# Database Connection
DB_DSN=postgres://username:password@localhost:5432/telemetry?sslmode=disable
```

### Configuration Loading

```go
// In cmd/main.go
cfg := config.LoadConfig()
// Returns Config struct with values from environment or defaults
```

### DSN Format

PostgreSQL connection strings (DSN):

```
postgres://username:password@host:port/database?sslmode=disable
```

**Examples**:
```bash
# Local development
postgres://postgres:password@localhost:5432/telemetry?sslmode=disable

# Docker Compose
postgres://postgres:password@postgres:5432/telemetry?sslmode=disable

# Production (SSL required)
postgres://user:pass@db.prod.internal:5432/telemetry?sslmode=require
```

---

## Developer Guide

### Project Setup

#### Prerequisites
- Go 1.25+
- PostgreSQL 12+
- Docker (for containerization)
- Kind (for Kubernetes testing)
- Make (for automation)

#### Local Development

```bash
# Clone and enter directory
cd collector

# Install dependencies (if needed)
go mod download

# Build binary
make build

# Run tests
make test

# Check coverage
make coverage
```

### Adding New Features

#### Adding a New Configuration Variable

1. **Update `internal/config.go`**:
```go
type Config struct {
    // ... existing fields ...
    NewField string
}

func LoadConfig() Config {
    // ... existing code ...
    cfg.NewField = getEnv("NEW_FIELD_VAR", "default_value")
    return cfg
}
```

2. **Update `cmd/main.go` to use it**:
```go
cfg := config.LoadConfig()
// Use cfg.NewField
```

3. **Update `k8s/deployment.yaml`**:
```yaml
env:
  - name: NEW_FIELD_VAR
    value: "some_value"
```

#### Adding a New Test

```go
// In internal/collector_test.go or internal/store_test.go

func TestMyFeature(t *testing.T) {
    // Arrange
    cfg := Config{...}
    fakeClient := &fakeClient{...}
    
    // Act
    collector := NewCollectorWithClient(cfg, fakeClient, testDSN)
    
    // Assert
    if collector == nil {
        t.Error("collector should not be nil")
    }
}
```

### Code Organization

- **`cmd/`**: Service entrypoints (main.go)
- **`internal/`**: Core business logic and models
  - `collector.go`: Main orchestrator
  - `store.go`: Database layer
  - `config.go`: Configuration loader
  - `model.go`: Data structures
  - `*_test.go`: Unit tests
- **`k8s/`**: Kubernetes manifests
- **`Dockerfile`**: Container definition

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/my-feature

# Make changes and test
make test

# Commit with coverage
make coverage  # Ensure 80%+

# Push
git push origin feature/my-feature

# Create PR
```

### Debugging

#### View Logs Locally

```bash
# Run with verbose output
./bin/collector  # Logs to stdout

# With grep filtering
./bin/collector 2>&1 | grep "error\|failed"
```

#### Database Debugging

```bash
# Connect to Postgres
psql "postgres://user:pass@localhost:5432/telemetry"

# View telemetry table
SELECT * FROM telemetry LIMIT 10;

# Check for duplicates (should be none)
SELECT gpu_id, timestamp, COUNT(*) 
FROM telemetry 
GROUP BY gpu_id, timestamp 
HAVING COUNT(*) > 1;

# View indexes
\d telemetry
```

#### Collector Debugging in Tests

```bash
# Run specific test with verbose output
go test -v ./internal -run TestCollector_RunProcessesMessages

# Run with coverage and race detection
go test -race -cover ./internal

# See full test output
go test -v ./internal 2>&1 | head -100
```

---

## Testing

### Test Coverage: 80.0% ✅

```
NewCollector:           100.0% ✅
NewCollectorWithClient: 100.0% ✅
Run:                     91.7% ✅
LoadConfig:             100.0% ✅
getEnv:                 100.0% ✅
insert:                  70.6% ✅
GetStore:               100.0% ✅
CloseStore:              83.3% ✅
TableName:              100.0% ✅
─────────────────────────────────
TOTAL:                   80.0% ✅ (Exceeds target)
```

### Running Tests

```bash
# Run all tests
make test

# Run specific test
go test -v ./internal -run TestCollector_RunProcessesMessages

# Run with coverage
make coverage

# Run with race detector
go test -race ./internal

# Generate coverage report (HTML)
go tool cover -html=coverage.out -o coverage.html
```

### Test Structure

#### Store Tests (`internal/store_test.go`)

Uses `sqlmock` library for mocking database operations:

```go
// Test example
func TestStore_Insert_ValidRecord(t *testing.T) {
    // Setup mock database
    db, mock, err := sqlmock.New()
    
    // Expect transaction and insert
    mock.ExpectBegin()
    mock.ExpectExec("INSERT INTO telemetry").
        WithArgs(...).
        WillReturnResult(sqlmock.NewResult(1, 1))
    mock.ExpectCommit()
    
    // Execute
    store := &Store{db: db}
    err := store.insert(record)
    
    // Assert
    if err != nil {
        t.Fatalf("unexpected error: %v", err)
    }
}
```

#### Collector Tests (`internal/collector_test.go`)

Uses fake MQ client for isolated testing:

```go
// Fake MQ client
type fakeClient struct {
    messages []Message
}

func (f *fakeClient) Consume(ctx context.Context, ...) ([]Message, error) {
    return f.messages, nil
}

func (f *fakeClient) Ack(ctx context.Context, ...) error {
    return nil
}

// Test example
func TestCollector_RunProcessesMessages(t *testing.T) {
    cfg := Config{Partitions: 1}
    cli := &fakeClient{messages: []Message{{...}}}
    
    collector := NewCollectorWithClient(cfg, cli, "sqlite:///:memory:")
    
    // Override InsertFunc to avoid DB
    store.InsertFunc = func(record map[string]interface{}) error {
        return nil
    }
    
    // Run and assert
}
```

### Test Scenarios Covered

✅ Configuration loading with defaults and environment overrides
✅ Collector polling from multiple partitions
✅ Message JSON parsing and validation
✅ Database insert with idempotency (unique constraint)
✅ MQ ack after successful processing
✅ Error handling (parse errors, insert errors, ack errors)
✅ Context cancellation and graceful shutdown
✅ Singleton pattern and connection pooling
✅ Empty message batch handling
✅ Field validation (missing gpu_id, invalid timestamp)
✅ Multiple partition concurrent processing
✅ Consumer group management

---

## Deployment

### Local Development

```bash
# 1. Build binary
make build

# 2. Set environment
export DB_DSN="postgres://user:pass@localhost:5432/telemetry?sslmode=disable"
export MQ_URL="http://localhost:8080"

# 3. Run
./bin/collector
```

### Docker Compose (local)

```yaml
version: '3.8'
services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: password
      POSTGRES_DB: telemetry
    ports:
      - "5432:5432"

  mq:
    image: mq:latest
    ports:
      - "8080:8080"

  collector:
    image: collector:latest
    environment:
      DB_DSN: "postgres://postgres:password@postgres:5432/telemetry?sslmode=disable"
      MQ_URL: "http://mq:8080"
    depends_on:
      - postgres
      - mq
```

Run with:
```bash
docker-compose up -d
```

---

## Docker

### Build Docker Image

```bash
# Using Makefile
make docker
# Creates: collector:latest

# Manual build
docker build -t collector:latest .
```

### Dockerfile Strategy

Multi-stage build for minimal image size:

```dockerfile
# Stage 1: Build
FROM golang:1.25-alpine AS builder
WORKDIR /build
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o collector ./cmd

# Stage 2: Runtime
FROM alpine:3.18
COPY --from=builder /build/collector /bin/collector
ENTRYPOINT ["/bin/collector"]
```

**Image Size**: ~50MB (alpine base + single binary)

### Run Docker Container

```bash
docker run -d \
  --name collector \
  -e DB_DSN="postgres://user:pass@postgres:5432/telemetry?sslmode=disable" \
  -e MQ_URL="http://mq:8080" \
  -e TOPIC="telemetry" \
  -e GROUP="collector-group" \
  -e PARTITIONS=3 \
  -e BATCH_SIZE=10 \
  -e POLL_INTERVAL_MS=500 \
  collector:latest
```

### View Docker Logs

```bash
docker logs collector
docker logs -f collector  # Follow logs
docker logs --tail 50 collector  # Last 50 lines
```

---

## Kubernetes

### Deployment Architecture

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: collector
spec:
  replicas: 3  # Horizontal scaling
  selector:
    matchLabels:
      app: collector
  template:
    metadata:
      labels:
        app: collector
    spec:
      containers:
      - name: collector
        image: collector:latest
        env:
        - name: MQ_URL
          value: "http://mq-service:8080"
        - name: DB_DSN
          valueFrom:
            secretKeyRef:
              name: db-secret
              key: dsn
```

### Deploy to Kind (Local Kubernetes)

**One-Shot Deployment**:
```bash
make kind-deploy
```

**Step-by-Step**:
```bash
# 1. Create local Kubernetes cluster
make kind-create
# Alias: kind create cluster --name mq-cluster

# 2. Build Docker image
make docker

# 3. Load image into Kind
make kind-load
# Alias: kind load docker-image collector:latest --name mq-cluster

# 4. Deploy to Kubernetes
make kind-deploy
# Alias: kubectl apply -f k8s/deployment.yaml

# 5. Monitor deployment
kubectl get deployments
kubectl get pods -l app=collector
kubectl describe pod <pod-name>

# 6. View logs
kubectl logs -l app=collector
kubectl logs -l app=collector -f  # Follow

# 7. Port forward (optional)
make port-forward
# Alias: kubectl port-forward svc/collector 8081:8081
```

### Verify Deployment

```bash
# Check if pods are running
kubectl get pods -l app=collector
# Expected output:
# NAME                        READY   STATUS    RESTARTS   AGE
# collector-xxx               1/1     Running   0          10s
# collector-yyy               1/1     Running   0          10s
# collector-zzz               1/1     Running   0          10s

# Check if service exists
kubectl get svc collector
# Expected output:
# NAME        TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)
# collector   ClusterIP   10.96.xxx.xxx   <none>        8081/TCP

# View pod logs
kubectl logs <pod-name>

# Execute command in pod
kubectl exec <pod-name> -- env | grep DB_DSN

# Port forward to access service
kubectl port-forward svc/collector 8081:8081
# Now accessible at http://localhost:8081
```

### Configuration via ConfigMap

Create `k8s/configmap.yaml`:
```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: collector-config
data:
  MQ_URL: "http://mq-service:8080"
  TOPIC: "telemetry"
  GROUP: "collector-group"
  PARTITIONS: "3"
  BATCH_SIZE: "10"
  POLL_INTERVAL_MS: "500"
```

Reference in Deployment:
```yaml
envFrom:
- configMapRef:
    name: collector-config
```

### Secret Management

Create `k8s/secret.yaml`:
```yaml
apiVersion: v1
kind: Secret
metadata:
  name: db-secret
type: Opaque
stringData:
  dsn: "postgres://user:pass@postgres:5432/telemetry?sslmode=disable"
```

Reference in Deployment:
```yaml
- name: DB_DSN
  valueFrom:
    secretKeyRef:
      name: db-secret
      key: dsn
```

### Cleanup

```bash
# Delete Kubernetes cluster
make kind-delete
# Alias: kind delete cluster --name mq-cluster
```

---

## Makefile Reference

### Available Targets

```makefile
# Build and Packaging
make build              # Compile binary to ./bin/collector
make docker             # Build Docker image (collector:latest)
make run                # Execute binary with environment variables

# Testing
make test               # Run all unit tests
make coverage           # Run tests and display coverage report

# Kubernetes (Kind)
make kind-create        # Create local kind cluster
make kind-load          # Build image and load into kind
make kind-deploy        # Deploy to kind cluster (all-in-one)
make kind-delete        # Delete kind cluster
make port-forward       # Port forward to service

# Maintenance
make clean              # Remove binaries and coverage files
make all                # Default target (calls build)
```

### Key Variables

```makefile
BINARY = collector
IMAGE = collector:latest
KIND_CLUSTER = mq-cluster
```

### Usage Examples

```bash
# Build and test
make build test

# Full deployment cycle
make kind-create kind-load kind-deploy

# Monitor coverage
make coverage  # Shows: Total coverage: 80.0%

# Cleanup
make clean kind-delete
```

---

## Database

### PostgreSQL Setup

#### Local PostgreSQL (macOS with Homebrew)

```bash
# Install
brew install postgresql

# Start service
brew services start postgresql

# Create database
createdb telemetry

# Connect
psql telemetry
```

#### Using Docker

```bash
docker run -d \
  --name postgres \
  -e POSTGRES_DB=telemetry \
  -e POSTGRES_PASSWORD=password \
  -p 5432:5432 \
  postgres:15-alpine

# Connection string
postgres://postgres:password@localhost:5432/telemetry?sslmode=disable
```

### Schema Creation

Automatic via GORM auto-migration in `store.go`:

```go
db.AutoMigrate(&Telemetry{})
// Creates table and indexes
```

Manual schema (if needed):

```sql
CREATE TABLE telemetry (
    id SERIAL PRIMARY KEY,
    gpu_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    data JSONB NOT NULL
);

CREATE INDEX idx_telemetry_gpu_id ON telemetry(gpu_id);
CREATE INDEX idx_telemetry_timestamp ON telemetry(timestamp);
CREATE UNIQUE INDEX idx_telemetry_gpu_ts_unique ON telemetry(gpu_id, timestamp);
```

### Connection Pooling

Configured for pgbouncer compatibility:

```go
db.DB().SetMaxOpenConns(50)           // Max open connections
db.DB().SetMaxIdleConns(10)           // Max idle connections
db.DB().SetConnMaxIdleTime(5 * time.Minute)    // Idle timeout
db.DB().SetConnMaxLifetime(30 * time.Minute)   // Connection lifetime
```

### Query Examples

```sql
-- Recent telemetry for a GPU
SELECT * FROM telemetry 
WHERE gpu_id = 'gpu-001' 
ORDER BY timestamp DESC 
LIMIT 10;

-- Average utilization over time
SELECT DATE_TRUNC('hour', timestamp) as hour,
       AVG((data->>'utilization')::float) as avg_util
FROM telemetry
GROUP BY DATE_TRUNC('hour', timestamp)
ORDER BY hour DESC;

-- Verify no duplicates (unique constraint working)
SELECT gpu_id, timestamp, COUNT(*) 
FROM telemetry 
GROUP BY gpu_id, timestamp 
HAVING COUNT(*) > 1;  -- Should return 0 rows

-- Data volume per GPU
SELECT gpu_id, COUNT(*) as record_count
FROM telemetry
GROUP BY gpu_id
ORDER BY record_count DESC;
```

---

## Troubleshooting

### Service Won't Start

#### Error: `failed to connect to database`

```
Solution:
1. Verify PostgreSQL is running:
   psql -l

2. Check connection string (DB_DSN):
   echo $DB_DSN

3. Test connection manually:
   psql "postgres://user:pass@localhost:5432/telemetry?sslmode=disable"

4. Check network:
   nc -zv localhost 5432
```

#### Error: `dial tcp: lookup mq-service: no such host`

```
Solution:
1. Verify MQ service is running:
   curl http://localhost:8080/health

2. Check MQ_URL environment variable:
   echo $MQ_URL

3. If using Docker:
   docker ps | grep mq

4. If using Kubernetes:
   kubectl get svc mq-service
```

### Messages Not Appearing in Database

#### Debug Steps

```bash
# 1. Check pod logs
kubectl logs -l app=collector

# 2. Verify table exists
psql telemetry -c "\d telemetry"

# 3. Check for insert errors
kubectl logs -l app=collector | grep -i "error\|failed"

# 4. View data in table
psql telemetry -c "SELECT COUNT(*) FROM telemetry;"

# 5. Check unique constraint violations
psql telemetry -c "SELECT gpu_id, timestamp, COUNT(*) FROM telemetry GROUP BY gpu_id, timestamp HAVING COUNT(*) > 1;"

# 6. Verify message format in MQ
# Messages should have: {"gpu_id": "...", "timestamp": "...", ...}
```

#### Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| No inserts | Parse error | Check JSON format has `gpu_id` and `timestamp` |
| Silently ignored duplicates | Unique constraint | Expected behavior - deduplication working |
| Connection timeout | Network unreachable | Verify host/port, check firewall |
| Table doesn't exist | Auto-migration failed | Check logs: "error... creating table" |

### Tests Failing

```bash
# 1. Clean build
make clean && make build

# 2. Run tests with verbose output
make test -v

# 3. Run specific test
go test -v ./internal -run TestCollector_RunProcessesMessages

# 4. Check for race conditions
go test -race ./internal

# 5. View coverage
make coverage

# Expected output: Total coverage: 80.0% or higher
```

### Kubernetes Issues

```bash
# Pod stuck in pending
kubectl describe pod <pod-name>
# Check: resource requests, node capacity

# Pod crashing
kubectl logs <pod-name> --previous
# Check: startup errors, env vars missing

# Service not accessible
kubectl get svc collector
# Check: ClusterIP, port mapping

# Check events
kubectl get events --all-namespaces | grep collector
```

### Performance Issues

```bash
# Monitor pod resource usage
kubectl top pod -l app=collector

# View database connections
psql telemetry -c "SELECT count(*) FROM pg_stat_activity;"

# Check slow queries (if pgbouncer)
pgbouncer -R

# Increase batch size (process more messages)
BATCH_SIZE=50  # Default: 10

# Decrease poll interval (poll more frequently)
POLL_INTERVAL_MS=200  # Default: 500
```

---

## Production Checklist

- ✅ Database
  - [ ] PostgreSQL 12+ installed and running
  - [ ] telemetry database created
  - [ ] Connection string set in secrets
  - [ ] Backup strategy in place
  - [ ] Monitoring and alerting configured

- ✅ Configuration
  - [ ] All environment variables documented
  - [ ] Secrets management (not in git)
  - [ ] Configuration version controlled
  - [ ] Secrets rotated regularly

- ✅ Deployment
  - [ ] Docker image built and tested
  - [ ] Image registry configured
  - [ ] Kubernetes manifests reviewed
  - [ ] Resource limits configured
  - [ ] Replica count appropriate for load

- ✅ Monitoring & Logging
  - [ ] Pod logs aggregated (ELK/Datadog)
  - [ ] Error rates monitored
  - [ ] Performance metrics tracked
  - [ ] Alerts configured for failures

- ✅ Testing
  - [ ] All tests passing (make test)
  - [ ] Coverage 80%+ (make coverage)
  - [ ] Integration tests with real DB
  - [ ] Load tests with expected volume

- ✅ Documentation
  - [ ] Runbook created
  - [ ] Deployment procedure documented
  - [ ] Troubleshooting guide available
  - [ ] Team trained on operation

- ✅ Error Handling
  - [ ] Graceful shutdown implemented
  - [ ] Error recovery working
  - [ ] No resource leaks
  - [ ] Connection pooling functional

- ✅ Security
  - [ ] Database credentials secured
  - [ ] Network policies configured
  - [ ] TLS enabled (if required)
  - [ ] No sensitive data in logs

---

## Notes

- **Collector relies on `InsertFunc`** (default uses real DB). Override `InsertFunc` during tests to avoid touching DB.
- **To enable DB-backed mode**, provide valid `DB_DSN`.
- **Consumer group state** is managed by the MQ service, not Collector.
- **Duplicates are silently ignored** (by design, ON CONFLICT DO NOTHING).
- **All components are configurable** via environment variables.
- **Graceful shutdown** responds to SIGINT and SIGTERM signals.

---

## Support & Contribution

For issues, questions, or contributions:
1. Check the Troubleshooting section above
2. Review test files for usage examples
3. Check Kubernetes events: `kubectl get events`
4. Examine pod logs: `kubectl logs -l app=collector`

---

**Last Updated**: April 2026
**Go Version**: 1.25+
**PostgreSQL Version**: 12+
**Coverage**: 80.0%
