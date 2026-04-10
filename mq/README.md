# 📦 Custom Message Queue (MQ)

A lightweight, production-quality in-memory message queue built from scratch in Go with clean layered architecture.

## 📋 Overview

This module implements a partitioned message queue designed for:
- **Scalability**: Partition-level concurrency with zero global lock contention in hot paths
- **Reliability**: At-least-once delivery semantics via consumer group offset management
- **Extensibility**: Clean internal package design ready for WAL/persistence with minimal changes
- **Clean Architecture**: Separation of concerns (core logic in `internal/`, HTTP API in `api/`, server in `cmd/`)

## 🎯 Key Features

### ✅ Implemented

- **Partitioned Topics**: Messages distributed across partitions via consistent hashing
- **Consistent Hashing**: Configurable replicas (default 3) on hash ring for deterministic routing
- **Consumer Groups**: Per-partition offset tracking with atomic operations (no bottlenecks)
- **Thread-Safe Operations**: RWMutex per partition, atomic offsets, zero global contention
- **Capacity Limits**: Configurable partition capacity with proper error handling
- **HTTP API**: REST endpoints for topic management and message produce/consume/ack
- **Health Checks**: Liveness and readiness probes for Kubernetes deployment
- **Configuration**: JSON config file with environment variable overrides
- **Comprehensive Tests**: Unit tests for partition (100% coverage), queue, and HTTP handlers
- **Docker & Kubernetes**: Multi-stage Dockerfile, Helm charts with probes and resource limits
- **CI/CD**: GitHub Actions workflow for automated testing

### 🔮 Extensibility Points

The following can be added with **minimal changes to public APIs**:

- **Write-Ahead Log (WAL)**: Extension hooks documented in `internal/partition.go`
- **Persistence**: Replace in-memory storage with disk-backed implementation
- **Replication**: Extend Topic to manage replicas across nodes
- **Metrics**: Add Prometheus instrumentation
- **Authentication**: Add middleware to HTTP API

## 🏗️ Architecture

### Layered Design

```
┌─────────────────────────────────────────┐
│ cmd/mq-server (HTTP server entrypoint)  │
├─────────────────────────────────────────┤
│ api/ (HTTP transport layer)             │
│ - handler.go: REST endpoint handlers    │
│ - routes.go: route registration         │
├─────────────────────────────────────────┤
│ internal/ (core MQ logic, no HTTP)      │
│ - partition.go: append-only log         │
│ - topic.go: partitioning & hashing      │
│ - queue.go: public Queue API            │
│ - consumer_group.go: offset tracking    │
│ - message.go: message type definition   │
└─────────────────────────────────────────┘
```

### Message Flow

```
Producer 
  ↓ (message with key)
Queue.Publish()
  ↓ (validates, routes to topic)
Topic.Produce() 
  ↓ (consistent hash key → partition)
Partition.Append() 
  ↓ (RWMutex protected, atomic offset)
Consumer Group
  ↓ (tracks per-partition offset)
Queue.Consume() 
  ↓ (returns batch from partition)
Consumer
  ↓ (processes message)
Queue.Ack() 
  ↓ (commits offset atomically)
```

### Consistent Hashing

```
Partition ring with replicas:
  - Each partition appears N times on the hash ring (default N=3)
  - Same key always maps to same partition (deterministic)
  - Reduces impact of partition changes on existing messages
  - Binary search for O(log M) lookup time (M = #partitions × replicas)
```

### Concurrency Model

```
Per-Partition Concurrency:
  - RWMutex guards append-only message slice
  - Multiple readers (consumers) can read concurrently
  - Single writer for appends (producer)

Per-Consumer Offset Tracking:
  - sync.Map + atomic.Int64 per partition per consumer
  - Zero global lock contention during consume/ack
  - Lock-free operations in hot path
```

## 🚀 Getting Started

### Prerequisites
```bash
Go 1.25+
Docker (optional, for containers)
Helm 3+ (optional, for Kubernetes deployment)
```

### Build

```bash
# Build binary
make build
# Binary created at: bin/mq-server

# Build Docker image
make docker
# Image: mq:latest

# Run tests
make test
```

### Run Locally

```bash
# Generate config file
make config
# Creates config.json with default values

# Run server
make run
# Server listens on :8080 by default

# Or run directly
./bin/mq-server -listen :8080 -partitions 3

# Check health
curl http://localhost:8080/healthz
```

## 📡 HTTP API

### Swagger
```bash
# Generage swagger doc
make swagger

# Build and run
make build run

# Open swagger UI
http://localhost:8080/swagger/index.html
```


## 📊 Testing & Coverage

### Run Tests
```bash
# Run all internal & API tests
make test

# Run with verbose output
go test ./internal ./api -v

# Run with coverage
go test ./internal ./api -coverprofile=coverage.out
go tool cover -html=coverage.out

# Check for race conditions
go test ./internal ./api -race
```

## 🐳 Docker & Kubernetes

### Docker

```bash
# Build image
make docker

# Run container
docker run -p 8080:8080 mq:latest -listen :8080 -partitions 3

# Check logs
docker logs <container-id>
```

### Kubernetes (Helm)

```bash
# Deploy to current cluster
make helm

# Or manual deployment
helm install mq chart/mq -f chart/mq/values.yaml

# Verify deployment
kubectl get pods -l app=mq
kubectl logs -l app=mq

# Check service
kubectl get svc mq

# Port forward to test locally
kubectl port-forward svc/mq 8080:80
curl http://localhost:8080/healthz
```

### Helm Configuration

Configure in `chart/mq/values.yaml`:
- `replicaCount`: Number of pod replicas
- `image.repository`, `image.tag`: Docker image details
- `resources.limits/requests`: CPU and memory limits
- `liveness.*, readiness.*`: Probe settings

## 📝 Configuration

### Config File Format

```json
{
  "listen": ":8080",
  "partitions": 3,
  "defaultCapacity": 0
}
```

### Environment Variables

- `CONFIG_PATH`: Path to config file (default: `config.json`)
- Override via command-line flags (see `cmd/mq-server/main.go`)

## 🔒 Concurrency & Performance

### Design Decisions

1. **No Global Lock**: Each partition has its own RWMutex
   - Supports ~10+ concurrent producers/consumers
   - Zero contention in typical scenarios

2. **Atomic Offsets**: Consumer offsets use sync.Map + atomic.Int64
   - Consume and Ack operations don't block each other
   - Lock-free in the common case

3. **Consistent Hashing**: Keys deterministically map to partitions
   - Same key always routes to same partition (ordering guarantee)
   - Configurable replicas improve distribution

### Scalability

- **Horizontal**: Add more partitions to the topic
- **Vertical**: Increase partition capacity
- **Future**: Extend to multiple nodes with leader-based replication

## 🛠️ Extensibility

### Adding Write-Ahead Log (WAL)

The codebase is designed to support WAL with minimal changes:

1. Create `internal/wal/interface.go`:
```go
type WriteAheadLog interface {
    Append(msg Message) error
    Read(offset int64, max int) ([]Message, error)
    Sync() error
    Close() error
}
```

2. Modify `internal/partition.go` Append method (see TODO comment):
```go
func (p *Partition) Append(msg Message) (int64, error) {
    // 1. Write to WAL first (NEW)
    if p.wal != nil {
        if err := p.wal.Append(msg); err != nil {
            return -1, err
        }
    }
    // 2. Then to in-memory slice
    p.mu.Lock()
    // ... existing code ...
    p.mu.Unlock()
    return off, nil
}
```

3. Initialize WAL in `internal/queue.go`:
```go
func (q *Queue) CreateTopic(...) error {
    // Create topic with WAL implementation
    topic := internal.NewTopic(...)
    // topic.SetWAL(walImpl) // NEW
    return nil
}
```

See comments in source code for exact integration points.

## 📚 Project Structure

```
mq/
├── cmd/
│   └── mq-server/
│       └── main.go              # HTTP server entrypoint
├── api/
│   ├── handler.go               # HTTP endpoint handlers (no business logic)
│   ├── handler_test.go          # HTTP handler tests
│   └── routes.go                # Route registration
├── internal/
│   ├── message.go               # Message type
│   ├── partition.go             # Append-only log (100% coverage)
│   ├── partition_test.go        # Partition tests
│   ├── topic.go                 # Topic with consistent hashing
│   ├── queue.go                 # Public Queue API
│   ├── queue_test.go            # Queue API tests
│   ├── consumer_group.go        # Consumer group offset tracking
│   └── go.mod                   # Go module definition
├── chart/
│   └── mq/                      # Helm chart for Kubernetes
│       ├── templates/
│       │   ├── deployment.yaml
│       │   ├── service.yaml
│       │   └── _helpers.tpl
│       ├── values.yaml
│       └── Chart.yaml
├── Dockerfile                   # Multi-stage Docker build
├── Makefile                     # Build and deployment targets
├── go.mod                       # Go module file
└── README.md                    # This file
```

## 📖 Design Principles

1. **Simplicity**: Clear, idiomatic Go code without unnecessary abstractions
2. **Testability**: Unit-testable partition and queue logic, mocked in API layer
3. **Extensibility**: Internal package designed for WAL/persistence/replication
4. **Performance**: Lock-free designs where possible, per-partition locking otherwise
5. **Reliability**: At-least-once delivery via explicit consumer offset commits
6. **Observability**: Health checks, structured errors, logging-ready

## 🚦 Limitations

Current limitations (can be addressed):

- **In-memory only**: Messages lost on restart (add WAL for persistence)
- **Single node**: No replication or failover (extend Topic for multi-node)
- **No consumer rebalancing**: Fixed partition assignment (add rebalancing API)
- **No message expiry**: Messages kept indefinitely (add TTL logic)

## 🎓 Learning Resources

### Message Queue Concepts

- **Partitioning**: Enables horizontal scaling and parallel consumption
- **Consumer Groups**: Coordinate multiple consumers reading same topic
- **Offsets**: Position in partition log, enables replay and exactly-once semantics
- **Consistent Hashing**: Distributes keys across partitions with minimal redistribution on changes

### Production Considerations

- Add metrics (Prometheus format)
- Add distributed tracing (OpenTelemetry)
- Add authentication/authorization
- Monitor partition sizes and consumer lag
- Set up alerting for missing heartbeats

## 📄 License

See LICENSE file.

## 💬 Questions?

Refer to:
- `API_AND_DESIGN.md`: Comprehensive design document and examples
- `REFACTORING_SUMMARY.md`: Summary of architectural decisions
- Inline code comments in `internal/*.go` for implementation details
