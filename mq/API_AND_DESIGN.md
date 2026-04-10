# MQ - Production-Grade In-Memory Message Queue

## Overview

A clean, extensible message queue system built in Go with:
- **Partition-based parallelism** for horizontal scalability
- **Consumer groups** with per-partition offset tracking
- **At-least-once delivery** semantics via offset commits
- **RESTful HTTP API** for producers and consumers
- **Thread-safe concurrent access** without global lock contention
- **Extensible design** for WAL and persistence

## Project Structure

```
mq/
├── internal/                 # Core MQ logic (no HTTP/transport dependencies)
│   ├── message.go           # Message struct
│   ├── partition.go         # Append-only partition log with RWMutex
│   ├── topic.go             # Topic managing multiple partitions
│   ├── queue.go             # Queue managing topics (public API)
│   ├── consumer_group.go    # Consumer group for coordinated consumption
│   ├── partition_test.go    # Partition unit tests
│   ├── queue_test.go        # Queue unit tests
│   └── consumer_group_test.go # (optional)
│
├── api/                      # HTTP transport layer
│   ├── handler.go           # HTTP request handlers
│   ├── handler_test.go      # Handler unit tests
│   └── routes.go            # Route registration
│
├── cmd/
│   └── server/
│       └── main.go          # Server entrypoint
│
├── chart/                    # Helm chart for Kubernetes
├── Dockerfile
├── Makefile
├── config.example.json
└── README.md
```

## Core Design

### Partition-Level Locking (No Global Contention)

Each partition has its own `sync.RWMutex`. This means:
- Multiple goroutines can read/write to different partitions in parallel
- Only writes within a partition are serialized
- With N partitions and up to ~10 producers/consumers, you get near-linear scalability

```
Partition 0: [Lock A] - Producer 1, Consumer 1 can operate in parallel
Partition 1: [Lock B] - Producer 2, Consumer 2 can operate in parallel
Partition 2: [Lock C] - Producer 3, Consumer 3 can operate in parallel
```

### Per-Consumer Offset Tracking (No Lock Contention)

Consumer offsets stored in `sync.Map` + `atomic.Int64`:
- Each consumer group independently tracks per-partition offsets
- Atomic updates avoid holding partition locks during offset commits
- Different groups committing concurrently don't block each other

### Partitioning Strategy

Messages are routed to partitions using FNV-1a hash modulo:
```
partition = fnv32(message.key) % numPartitions
```

**At-least-once delivery**: Consumers must explicitly acknowledge (commit offset) after processing. If consumer fails before ack, message can be redelivered.

## API Endpoints

### 1. Create Topic
```bash
POST /topics
Content-Type: application/json

{
  "name": "events",
  "partitions": 3,
  "partition_capacity": 1000
}
```

Response: `201 Created`

### 2. Publish Message
```bash
POST /topics/{topic}/publish
Content-Type: application/json

{
  "key": "user-456",
  "payload": "hello world"
}
```

Response: `200 OK`
```json
{
  "partition": 0,
  "offset": 5
}
```

### 3. Consume Messages
```bash
GET /topics/{topic}/consume?group=consumer-group-1&partition=0&batch=10
```

Response: `200 OK`
```json
{
  "messages": [
    {
      "offset": 0,
      "key": "key-1",
      "payload": "data"
    }
  ],
  "next_offset": 1
}
```

### 4. Acknowledge (Commit Offset)
```bash
POST /topics/{topic}/ack
Content-Type: application/json

{
  "group": "consumer-group-1",
  "partition": 0,
  "offset": 5
}
```

Response: `200 OK`

### 5. Health Check
```bash
GET /healthz
```

Response: `200 OK` with body `ok`

## Build & Run

### Prerequisites
- Go 1.25+
- Docker (optional)
- Helm (optional, for Kubernetes)

### Build Binary
```bash
make build
# binary: ./bin/mq-server
```

### Run Locally
```bash
make config  # generates config.json
make run     # runs server with config
```

Or directly:
```bash
./bin/mq-server -listen :8080 -partitions 3
```

### Docker
```bash
make docker
# image: mq:latest
docker run -p 8080:8080 mq:latest
```

### Kubernetes (Helm)
```bash
make helm
# installs mq chart in current kubecontext
```

## Testing & Coverage

```bash
# Run all tests
make test

# Run with coverage
go test ./... -coverprofile=coverage.out
go tool cover -html=coverage.out

# Current coverage target: >=80%
```

## Extensibility: Adding WAL

To add a Write-Ahead Log (for durability):

1. **Define WAL interface** in `internal/wal/`:
   ```go
   type WAL interface {
       Append(msg Message) (offset int64, err error)
       ReadFrom(offset int64, max int) ([]Message, error)
       Close() error
   }
   ```

2. **Modify Partition** to accept WAL:
   ```go
   type Partition struct {
       // ...existing fields...
       wal WAL // optional
   }
   ```

3. **Update Partition.Append**:
   ```go
   func (p *Partition) Append(msg Message) (int64, error) {
       if p.wal != nil {
           offset, err := p.wal.Append(msg)  // write first
           if err != nil { return -1, err }
       }
       p.mu.Lock()
       offset := int64(len(p.messages))
       p.messages = append(p.messages, msg)
       p.mu.Unlock()
       return offset, nil
   }
   ```

4. **Replay on startup** (in Queue.CreateTopic or elsewhere):
   ```go
   if walImpl != nil {
       messages, _ := walImpl.ReadFrom(0, 0) // read all
       for _, m := range messages {
           partition.Append(m)
       }
   }
   ```

The API surface remains unchanged; callers don't need to know about WAL.

## Concurrency & Scalability

### Why Partition-Level Locking Works

With N partitions and K producers/consumers:
- Each producer hashes message key → picks partition
- Each partition is independent with its own lock
- **Expected parallelism**: min(K, N)
- For K=10 producers and N=3 partitions: ~3 concurrent appends

### Why sync.Map for Offsets

- `sync.Map` is optimized for reads (most operations)
- `atomic.Int64` for offset updates (no contention with reads)
- Different consumer groups committing concurrently don't block

## Limitations & Future Work

**Current (In-Memory)**
- Messages stored in RAM only
- No persistence across restarts
- No replication or failover
- Single-instance deployment

**Future Enhancements**
- WAL for durability
- Replication across nodes
- Dynamic partition rebalancing
- Consumer group rebalancing protocol
- gRPC transport
- Metrics (Prometheus)
- Distributed tracing

## Example Workflow

```bash
# 1. Create topic
curl -X POST -H "Content-Type: application/json" \
  -d '{"name":"logs","partitions":3}' \
  http://localhost:8080/topics

# 2. Publish messages
curl -X POST -H "Content-Type: application/json" \
  -d '{"id":"1","key":"app1","payload":"error occurred"}' \
  http://localhost:8080/topics/logs/publish

# 3. Consume as group
curl "http://localhost:8080/topics/logs/consume?group=alerting&partition=0&batch=10"

# 4. Commit offset
curl -X POST -H "Content-Type: application/json" \
  -d '{"group":"alerting","partition":0,"offset":1}' \
  http://localhost:8080/topics/logs/ack
```

## Code Quality

- **Idiomatic Go**: Small functions, clear variable names
- **Comments**: Explain partitioning, offset semantics, concurrency
- **Error handling**: Structured errors instead of silent failures
- **Input validation**: Key must be non-empty, batch > 0, partition in range
- **Tests**: Unit tests for partition, queue, consumer groups, and HTTP handlers
- **Concurrency-safe**: No race conditions (verified with `-race` flag)

## References

- Kafka (partitioning, consumer groups, offset management)
- RabbitMQ (simple queue semantics)
- etcd (concurrency patterns in Go)
