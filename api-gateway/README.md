# 🔗 API Gateway - GPU Pipeline

> REST API gateway for querying GPU telemetry data from the central PostgreSQL database

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [API Endpoints](#api-endpoints)
- [Time-Range Filtering](#time-range-filtering)
- [OpenAPI Documentation](#openapi-documentation)
- [Building & Testing](#building--testing)
- [Deployment](#deployment)
- [Configuration](#configuration)
- [Makefile Commands](#makefile-commands)
- [Troubleshooting](#troubleshooting)
- [Future Enhancements](#future-enhancements)

## Features

- ✅ **List GPUs** - Get all unique GPU IDs with telemetry data
- ✅ **Query Telemetry** - Query GPU telemetry with time-range filtering
- ✅ **OpenAPI/Swagger** - Automatic API documentation
- ✅ **Health Checks** - Built-in health endpoint
- ✅ **80%+ Test Coverage** - Comprehensive test suite
- ✅ **Production Ready** - Error handling, validation, logging

## Architecture

```
┌─────────────────────────────────────────┐
│      API Gateway (Port 8000)            │
├─────────────────────────────────────────┤
│  • List GPUs endpoint                   │
│  • Query Telemetry endpoint             │
│  • Time-range filtering                 │
│  • OpenAPI/Swagger documentation        │
└────────────────┬────────────────────────┘
                 │
          Reads from
                 │
         ┌───────▼────────┐
         │  PostgreSQL    │
         │   Database     │
         │  (Telemetry)   │
         └────────────────┘
```

## Endpoints

All API endpoints are documented with OpenAPI 3.0 specification via Swagger UI.

### Access Documentation
- **Swagger UI**: `http://localhost:8000/swagger/index.html` (when running)
- **OpenAPI JSON Spec**: `http://localhost:8000/api/v1/docs`

Refer to the Swagger UI for complete endpoint documentation, request/response schemas, and interactive testing.

## Quick Start

### Prerequisites
- Go 1.25+
- PostgreSQL 15+
- Docker (optional)

### Local Development
```bash
# Build
make build

# Run tests with coverage
make coverage

# Generate Swagger docs
make swagger

# Run locally
make run
```

### Docker
```bash
# Build Docker image
make docker-build

# Run in Docker
docker run -p 8000:8000 \
  -e DATABASE_URL="postgresql://user:pass@postgres:5432/telemetry" \
  gpu-pipeline/api-gateway:latest
```

## Building & Testing

### Local Build
```bash
make build
```

### Docker Build
```bash
make docker-build
```

### Run Tests
```bash
make test
```

### View Coverage
```bash
make coverage
make coverage-check  # Verify >= 80%
```

### Generate Swagger Documentation
```bash
make swagger
# Generates docs/ directory with OpenAPI specification
```

## Running

### Prerequisites
- PostgreSQL 15+ running with telemetry database
- Database connection configured

### Local Execution
```bash
# Set database URL
export DATABASE_URL="user=user password=pass host=localhost port=5432 dbname=telemetry sslmode=disable"

# Run API Gateway
./bin/api-gateway --port 8000
```

### Docker
```bash
docker run -p 8000:8000 \
  -e DATABASE_URL="postgresql://user:pass@postgres:5432/telemetry?sslmode=disable" \
  gpu-pipeline/api-gateway:latest
```

## Configuration

### Environment Variables
- `DATABASE_URL` - PostgreSQL connection string
- `PORT` - Server port (default: 8000)

### Command Line Flags
```bash
./bin/api-gateway -dsn "user=user password=pass host=localhost dbname=telemetry" -port 8000
```

## OpenAPI Documentation

The API is fully documented with OpenAPI 3.0 specification and supports Swagger UI.

### Access Swagger UI
1. Build and run the service:
   ```bash
   make run
   ```

2. Visit Swagger UI: `http://localhost:8000/swagger/index.html`

### Access OpenAPI JSON
- Raw OpenAPI spec: `http://localhost:8000/api/v1/docs`
- Swagger UI: `http://localhost:8000/swagger/`

### Generate Documentation
```bash
make swagger
# Generates documentation in docs/ directory
```

## Time-Range Filtering

Query telemetry data within a specific time range:

```bash
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu-001",
    "start_time": "2026-04-11T10:00:00Z",
    "end_time": "2026-04-11T15:00:00Z"
  }'
```

## Deployment

### Kubernetes Deployment
```bash
# Create Kind cluster and deploy
make kind-create kind-deploy

# Port-forward the service
make port-forward
```

### View Deployment Status
```bash
kubectl get deployment -n gpu-pipeline
kubectl logs -n gpu-pipeline deployment/api-gateway
```

## Makefile Commands

A comprehensive Makefile is provided for common operations:

### Build Commands
```bash
make all              # Clean and build everything
make build            # Build the binary
make lint             # Run code linter
```

### Test Commands
```bash
make test             # Run all tests with -race flag
make coverage         # Generate coverage report (HTML)
make coverage-check   # Check if coverage >= 80%
```

### Docker Commands
```bash
make docker-build     # Build Docker image
make docker-push      # Push image to registry
```

### Kubernetes Commands
```bash
make kind-create      # Create Kind cluster
make kind-deploy      # Deploy to Kind cluster
make kind-delete      # Delete Kind cluster
make docker-load      # Load Docker image into Kind
make port-forward     # Port-forward service (8000:8000)
```

### Documentation
```bash
make swagger          # Generate Swagger/OpenAPI docs
```

### Development
```bash
make run              # Build and run locally
make config           # Generate default config.json
make clean            # Clean build artifacts
make help             # Show all available commands
```

## Testing the API

### List GPUs
```bash
curl -X GET http://localhost:8000/api/v1/gpus \
  -H "Content-Type: application/json"
```

### Query Telemetry
```bash
curl -X POST http://localhost:8000/api/v1/telemetry/query \
  -H "Content-Type: application/json" \
  -d '{
    "gpu_id": "gpu-001",
    "start_time": "2026-04-11T00:00:00Z",
    "end_time": "2026-04-12T00:00:00Z"
  }'
```

### Health Check
```bash
curl -X GET http://localhost:8000/api/v1/health \
  -H "Content-Type: application/json"
```

## Error Handling

The API returns standard HTTP status codes:
- `200 OK` - Successful query
- `400 Bad Request` - Invalid request parameters
- `404 Not Found` - GPU or telemetry data not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

Error responses include a descriptive message:
```
HTTP/1.1 400 Bad Request
gpu_id is required
```

## Performance Considerations

### Queries
- List GPUs: Fast (distinct query on indexed column)
- Query Telemetry: Optimized with timestamp indexing
- Time-range filtering: Uses indexed queries

### Recommendations
- Use time-range filtering to limit result sets
- Query specific GPUs when possible
- Implement pagination for large datasets (future enhancement)

## Database Schema Requirements

The API expects the following database schema (provided by the Collector):

```sql
CREATE TABLE telemetry (
    id SERIAL PRIMARY KEY,
    gpu_id VARCHAR(255) NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    data JSONB NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (gpu_id, timestamp)
);

CREATE INDEX idx_telemetry_gpu_id ON telemetry(gpu_id);
CREATE INDEX idx_telemetry_timestamp ON telemetry(timestamp);
CREATE UNIQUE INDEX telemetry_gpu_ts_unique ON telemetry (gpu_id, timestamp);
```

## Integration with GPU Pipeline

The API Gateway integrates with the GPU Pipeline architecture:

1. **Collector Service** - Writes telemetry to PostgreSQL
2. **API Gateway** - Reads from PostgreSQL and exposes REST API
3. **Clients** - Query telemetry through REST API

## Test Coverage

Current test coverage: **80%+**

Tests cover:
- ✅ List GPUs endpoint
- ✅ Query Telemetry endpoint
- ✅ Time-range filtering
- ✅ Error handling
- ✅ Invalid requests
- ✅ Database errors
- ✅ HTTP method validation
- ✅ Health endpoint

## Future Enhancements

- [ ] Pagination for large result sets
- [ ] Aggregation endpoints (avg, max, min power)
- [ ] Export to CSV/JSON
- [ ] Real-time WebSocket streaming
- [ ] Caching layer (Redis)
- [ ] Authentication & authorization
- [ ] Rate limiting
- [ ] GraphQL API alternative

## Troubleshooting

### Database Connection Error
```
Error: failed to connect to database: connection refused
```
**Solution**: Ensure PostgreSQL is running and `DATABASE_URL` is correct

### Port Already in Use
```
listen tcp :8000: bind: address already in use
```
**Solution**: Change port with `-port` flag: `./bin/api-gateway -port 8082`

### No Data Returned
```
HTTP 404: no telemetry data found for gpu_id: gpu-001
```
**Solution**: Ensure the Collector service has written data for that GPU

## License

Same as main GPU Pipeline project

## Status

✅ Phase 7 Complete - API Gateway Service
- List GPUs endpoint ✅
- Query telemetry endpoint ✅
- Time-range filtering ✅
- 80%+ test coverage ✅
- Swagger/OpenAPI ready ✅
