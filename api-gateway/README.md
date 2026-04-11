# 🔗 API Gateway - GPU Pipeline

> REST API gateway for querying GPU telemetry data from the central PostgreSQL database with 80%+ test coverage and clean layered architecture

## Table of Contents

- [Features](#features)
- [Architecture](#architecture)
- [Quick Start](#quick-start)
- [Project Structure](#project-structure)
- [API Endpoints](#api-endpoints)
- [Building & Testing](#building--testing)
- [Running](#running)
- [Configuration](#configuration)
- [Makefile Commands](#makefile-commands)
- [Testing the API](#testing-the-api)
- [Database Schema](#database-schema)
- [Troubleshooting](#troubleshooting)

## Features

- ✅ **List GPUs** - Get all unique GPU IDs with telemetry data
- ✅ **Query Telemetry** - Query GPU telemetry with time-range filtering
- ✅ **OpenAPI/Swagger** - Automatic API documentation
- ✅ **Health Checks** - Built-in health endpoint
- ✅ **80%+ Test Coverage** - Comprehensive unit and integration tests
- ✅ **Production Ready** - Error handling, validation, logging
- ✅ **Clean Architecture** - Layered design with interfaces, dependency injection, and singleton pattern
- ✅ **Testable Helpers** - Refactored cmd package with extracted helper functions

## Architecture

### Layered Design

The API Gateway follows a clean, layered architecture pattern for maintainability and testability:

```
┌─────────────────────────────────────────────┐
│         HTTP Handlers Layer                 │
│  (internal/handler.go)                      │
│  - ListGPUs, QueryTelemetry, GetGPUTelemetry│
│  - Health check, Error handling             │
└────────────────┬────────────────────────────┘
                 │ implements
                 │ TelemetryService interface
┌────────────────▼────────────────────────────┐
│        Service Layer                        │
│  (pkg/service/telemetry.go)                 │
│  - Business logic                           │
│  - Orchestrates repository calls            │
│  - Request validation                       │
└────────────────┬────────────────────────────┘
                 │ implements
                 │ TelemetryRepository interface
┌────────────────▼────────────────────────────┐
│      Repository Layer                       │
│  (pkg/repository/telemetry.go)              │
│  - GORM database queries                    │
│  - Data access abstraction                  │
└────────────────┬────────────────────────────┘
                 │ uses
                 │ Singleton DB connection
┌────────────────▼────────────────────────────┐
│      Database Layer                         │
│  (pkg/db/connection.go)                     │
│  - Singleton pattern with sync.Once         │
│  - Thread-safe connection management        │
│  - Connection lifecycle (Connect, Close)    │
└────────────────┬────────────────────────────┘
                 │
         Reads from
                 │
         ┌───────▼────────┐
         │  PostgreSQL    │
         │   Database     │
         │  (Telemetry)   │
         └────────────────┘
```

### Design Patterns

1. **Layered Architecture** - Separation of concerns across handler, service, and repository layers
2. **Singleton Pattern** - Thread-safe database connection management
3. **Repository Pattern** - Data access abstraction layer
4. **Dependency Injection** - Constructor-based DI for loose coupling
5. **Interface-Based Design** - Service and Repository interfaces for testability

## Project Structure

```
api-gateway/
├── cmd/
│   ├── main.go               # Entry point with helper functions
│   └── main_test.go          # Tests for cmd package
├── internal/
│   ├── handler.go            # HTTP request handlers
│   ├── handler_test.go       # Handler tests with mocks
│   ├── routes.go             # Route registration
│   └── routes_test.go        # Route tests
├── pkg/
│   ├── db/
│   │   ├── connection.go     # Singleton DB connection
│   │   └── connection_test.go# DB connection tests
│   ├── interfaces/
│   │   ├── repository.go     # TelemetryRepository interface
│   │   └── service.go        # TelemetryService interface
│   ├── models/
│   │   ├── telemetry.go      # Domain models and DTOs
│   │   └── telemetry_test.go # Model tests
│   ├── repository/
│   │   ├── telemetry.go      # Repository implementation
│   │   └── telemetry_test.go # Repository tests
│   └── service/
│       ├── telemetry.go      # Service implementation
│       └── telemetry_test.go # Service tests
├── docs/                      # Generated Swagger docs
├── Makefile                   # Build and test commands
├── Dockerfile                 # Docker image definition
├── go.mod                     # Go module definition
├── go.sum                     # Go module checksums
└── README.md                  # This file
```

## API Endpoints

All API endpoints are documented with OpenAPI 3.0 specification via Swagger UI.

### Access Documentation
- **Swagger UI**: `http://localhost:8000/swagger/index.html`
- **OpenAPI JSON Spec**: `http://localhost:8000/api/v1/docs`

### Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/health` | Health check |
| GET | `/api/v1/gpus` | List all GPU IDs |
| GET | `/api/v1/gpus/{id}/telemetry` | Get telemetry for specific GPU |

Refer to Swagger UI for complete endpoint documentation, request/response schemas, and interactive testing.

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
make coverage         # Generate HTML coverage report
make coverage-check   # Verify >= 80% coverage
```

### Generate Swagger Documentation
```bash
make swagger
```

## Running

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

### Kubernetes
```bash
# Create Kind cluster
make kind-create

# Deploy to cluster
make kind-deploy

# Port-forward service
make port-forward
```

## Configuration

### Environment Variables
- `DATABASE_URL` - PostgreSQL connection string
- `PORT` - Server port (default: 8000)

### Command Line Flags
```bash
./bin/api-gateway -dsn "postgresql://user:pass@localhost/db" -port 8000
```

## Makefile Commands

```bash
# Build
make build              # Build the binary
make all                # Clean and build everything
make lint               # Run code linter

# Test
make test               # Run all tests with -race flag
make coverage           # Generate coverage report (HTML)
make coverage-check     # Verify >= 80% coverage

# Documentation
make swagger            # Generate Swagger/OpenAPI docs

# Docker
make docker-build       # Build Docker image
make docker-push        # Push to registry

# Kubernetes
make kind-create        # Create Kind cluster
make kind-deploy        # Deploy to cluster
make kind-delete        # Delete cluster
make port-forward       # Port-forward service

# Development
make run                # Build and run locally
make clean              # Clean artifacts
make help               # Show all commands
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

### Get GPU Telemetry
```bash
curl -X GET "http://localhost:8000/api/v1/gpus/gpu-001/telemetry?start_time=2026-04-11T00:00:00Z&end_time=2026-04-12T00:00:00Z" \
  -H "Content-Type: application/json"
```

### Health Check
```bash
curl -X GET http://localhost:8000/api/v1/health \
  -H "Content-Type: application/json"
```

## Error Handling

The API returns standard HTTP status codes:
- `200 OK` - Successful query
- `400 Bad Request` - Invalid request parameters or malformed query
- `404 Not Found` - GPU or telemetry data not found
- `405 Method Not Allowed` - Wrong HTTP method
- `500 Internal Server Error` - Server error

Error responses include a descriptive message:
```json
HTTP/1.1 400 Bad Request
{
  "error": "gpu_id is required"
}
```

## Database Schema

The API expects the following database schema (provided by the Collector service):

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

## Test Coverage

Current test coverage: **80%+**

Coverage by package:
- `internal` (handlers/routes): 94.6%
- `pkg/service`: 100.0%
- `pkg/models`: 100.0%
- `pkg/repository`: 97.0%
- `pkg/db`: 79.2%
- `cmd`: Tested with helper functions

Tests cover:
- ✅ All HTTP endpoints (success and error cases)
- ✅ Path extraction and validation
- ✅ Error status code determination
- ✅ Time-range filtering
- ✅ Database queries and error handling
- ✅ Service layer orchestration
- ✅ Repository layer data access
- ✅ Dependency injection setup
- ✅ Environment variable handling

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

### Test Failures
```bash
# Clear cache and rebuild
make clean
make test
```

## License

Same as main GPU Pipeline project

## Status

✅ Phase 7 Complete - API Gateway Service
- Clean layered architecture ✅
- Singleton pattern DB connection ✅
- Repository pattern ✅
- Service layer ✅
- Dependency injection ✅
- List GPUs endpoint ✅
- Query telemetry endpoint ✅
- Get GPU telemetry endpoint ✅
- Time-range filtering ✅
- 80%+ test coverage ✅
- Swagger/OpenAPI documentation ✅
- Refactored cmd package with testable helpers ✅

