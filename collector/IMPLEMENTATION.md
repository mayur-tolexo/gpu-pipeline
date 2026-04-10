# Collector Microservice - Implementation Complete ✅

## Overview
The Collector microservice is a production-ready service that pulls telemetry messages from the MQ service and persists them into PostgreSQL with idempotency guarantees.

## Architecture

### Components

1. **Store (Singleton Pattern)**
   - GORM-based wrapper around PostgreSQL
   - pgbouncer-compatible connection pooling
   - Idempotent inserts using `ON CONFLICT DO NOTHING`
   - Unique constraint on (gpu_id, timestamp)

2. **Collector (Poll-Based)**
   - Polls each MQ partition independently
   - Processes messages in batches
   - Parses JSON payloads
   - Calls Store.Insert for persistence
   - Acks messages after successful processing
   - Graceful shutdown on context cancellation

3. **Configuration (Environment-Based)**
   - `MQ_URL`: Message queue service URL (default: http://mq-service:8080)
   - `TOPIC`: Topic to consume from (default: telemetry)
   - `GROUP`: Consumer group name (default: collector-group)
   - `PARTITIONS`: Number of partitions (default: 3)
   - `BATCH_SIZE`: Messages per fetch (default: 10)
   - `POLL_INTERVAL_MS`: Poll interval in ms (default: 500)
   - `DB_DSN`: PostgreSQL connection string

## Data Model

```go
type Telemetry struct {
    ID        uint      // Auto-generated primary key
    GPUID     string    // GPU identifier (indexed)
    Timestamp time.Time // Event timestamp (indexed)
    Data      []byte    // JSON payload
}
```

**Indexes:**
- Primary key on `id`
- Index on `gpu_id`
- Index on `timestamp`
- Unique index on `(gpu_id, timestamp)` for idempotency

## Test Coverage: 80.0% ✅

### Covered Functions
- `NewCollector`: 100%
- `NewCollectorWithClient`: 100%
- `Run`: 91.7%
- `LoadConfig`: 100%
- `getEnv`: 100%
- `insert`: 70.6%
- `GetStore`: 100%
- `CloseStore`: 83.3%

### Test Scenarios
1. **Config Loading** - Default values and environment overrides
2. **Collector Polling** - Multi-partition consumption with fake MQ client
3. **Message Processing** - JSON parsing and record insertion
4. **Ack Handling** - Verifying acks sent after processing
5. **Error Cases** - Missing fields, invalid timestamps, insert errors
6. **Idempotency** - Duplicate handling via unique constraint

## Running Locally

### Prerequisites
```bash
go 1.25+
PostgreSQL 12+
MQ service running (http://localhost:8080)
```

### Build
```bash
make build
# Binary: ./bin/collector
```

### Run with Test Database
```bash
# Set environment variables
export DB_DSN="postgres://user:pass@localhost:5432/telemetry?sslmode=disable"
export MQ_URL="http://localhost:8080"

# Run
./bin/collector
```

### Run Tests
```bash
# Run all tests
make test

# Run with coverage report
make coverage
```

### Expected Output
```
Running collector tests with coverage...
ok  	gpu-pipeline/collector/internal	1.577s	coverage: 80.0% of statements

gpu-pipeline/collector/internal/collector.go:30:	NewCollector		100.0%
gpu-pipeline/collector/internal/collector.go:37:	NewCollectorWithClient	100.0%
gpu-pipeline/collector/internal/collector.go:47:	Run			91.7%
...
total:							(statements)		80.0%
Total coverage: 80.0%
```

## Docker Deployment

### Build Image
```bash
docker build -t collector:latest .
```

### Run Container
```bash
docker run -e DB_DSN="postgres://user:pass@postgres:5432/telemetry?sslmode=disable" \
           -e MQ_URL="http://mq-service:8080" \
           -e PARTITIONS=3 \
           collector:latest
```

## Kubernetes Deployment

### Deploy to Kind Cluster
```bash
# Load image into kind
kind load docker-image collector:latest --name <cluster-name>

# Apply manifests
kubectl apply -f k8s/deployment.yaml

# Verify deployment
kubectl get pods -l app=collector
kubectl logs -l app=collector
```

### Deployment Configuration
- **Replicas**: 3 (configurable in k8s/deployment.yaml)
- **Image**: collector:latest
- **Resources**: Configure as needed
- **Environment Variables**: Passed via ConfigMap (optional)

## Database Schema

### Auto-Migration
The collector automatically creates the telemetry table on startup:

```sql
CREATE TABLE IF NOT EXISTS telemetry (
    id SERIAL PRIMARY KEY,
    gpu_id TEXT NOT NULL,
    timestamp TIMESTAMPTZ NOT NULL,
    data JSONB NOT NULL
);

CREATE UNIQUE INDEX IF NOT EXISTS telemetry_gpu_ts_unique 
    ON telemetry (gpu_id, timestamp);

CREATE INDEX IF NOT EXISTS telemetry_gpu_id_idx 
    ON telemetry (gpu_id);

CREATE INDEX IF NOT EXISTS telemetry_timestamp_idx 
    ON telemetry (timestamp);
```

## Connection Pooling

Configured for pgbouncer in transaction pooling mode:
- Max open connections: 50
- Max idle connections: 10
- Connection max idle time: 5 minutes
- Connection max lifetime: 30 minutes

## Idempotency Guarantee

Messages are deduplicated based on unique (gpu_id, timestamp) constraint:
- **First attempt**: Insert succeeds
- **Duplicate arrival**: Silently ignored (ON CONFLICT DO NOTHING)
- **Result**: Exactly-once semantics per GPU per timestamp

## Error Handling

The collector handles errors gracefully:
1. **Parse errors** - Logs and continues to next message
2. **Insert errors** - Logs and continues (may retry on next poll)
3. **Ack errors** - Logs but doesn't fail (message may be reprocessed)
4. **MQ errors** - Logs and retries on next poll interval

## Monitoring

### Health Indicators
- Process running without crashes
- Log output shows "collector: starting" and "collector: stopping"
- Database connectivity tested on startup

### Recommended Metrics
- Messages polled per second
- Insert success/failure rate
- Ack success/failure rate
- Processing latency

## Troubleshooting

### Database Connection Fails
```
Error: "failed to connect database"
Solution: Verify DB_DSN is correct and Postgres is running
```

### MQ Connection Fails
```
Error: "consume partition=0 error: connection refused"
Solution: Verify MQ_URL is correct and MQ service is running
```

### Messages Not Appearing in DB
```
Check:
1. Logs for "failed to insert record"
2. Database connectivity
3. Unique constraint violations (check telemetry_gpu_ts_unique index)
4. Ensure messages have valid gpu_id and timestamp
```

## Files

- `collector/cmd/main.go` - Service entrypoint
- `collector/internal/collector.go` - Main collector logic
- `collector/internal/store.go` - Database persistence layer
- `collector/internal/config.go` - Configuration loading
- `collector/internal/model.go` - Data models
- `collector/internal/store_test.go` - Store tests (sqlmock + integration)
- `collector/internal/collector_test.go` - Collector tests
- `collector/Makefile` - Build and test targets
- `collector/Dockerfile` - Multi-stage Docker build
- `collector/k8s/deployment.yaml` - Kubernetes deployment (3 replicas)
- `collector/README.md` - User documentation

## Next Steps

1. **Deploy MQ** - Ensure MQ service is running
2. **Create Postgres DB** - `createdb telemetry`
3. **Start Collector** - `make build && DB_DSN="..." ./bin/collector`
4. **Publish Messages** - Send telemetry to MQ topic
5. **Verify** - Check telemetry table for data

## License

See LICENSE file in repository root.
