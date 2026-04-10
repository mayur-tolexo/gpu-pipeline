# 📦 Custom Message Queue (MQ)

## Overview

This module implements a lightweight, partitioned message queue built
from scratch in Go.

It is designed as a generic, reusable infrastructure component,
independent of any domain (e.g., GPU telemetry), and supports scalable,
concurrent message processing.


## Goals

- Provide a simple and extensible message queue
- Support multiple producers and consumers
- Ensure ordering guarantees within partitions
- Enable horizontal scalability via partitioning
- Maintain at-least-once delivery semantics


## Core Concepts

### Message

``` go
type Message struct {
    ID        string
    Key       string
    Payload   []byte
    Timestamp time.Time
}
```


### Topic

A logical stream of messages divided into partitions.

### Partition

- Append-only log
- Guarantees ordering
- Independent unit of scalability

### Consumer Group

- Tracks offsets per partition
- Enables parallel consumption
- Supports replay


## Architecture

Producers → Topic → Partition → Consumers


## Delivery Semantics

At-least-once delivery: - Messages are acknowledged after processing -
Failures lead to re-delivery


## Limitations

- In-memory only (no persistence)
- No replication
- No rebalancing


## Future Improvements

The current implementation is focused on simplicity and correctness. The following enhancements can be added to make the system production-ready:

### Storage & Durability
- Write-Ahead Log (WAL) for persistence
- Disk-backed message storage

### Reliability & Fault Tolerance
- Replication across nodes
- Leader election for partitions

### Scalability
- Dynamic partition rebalancing
- Consumer group rebalancing

### Developer Experience
- Lightweight client SDK (Go) for producers and consumers
- Support for multiple languages (e.g., Python, Java)
- Built-in retry and batching support in SDK

### Observability
- Metrics (Prometheus)
- Distributed tracing
- Structured logging

### Performance
- Message batching
- Compression
- gRPC-based communication


# MQ module

This package provides a minimal in-memory message queue with partitions and consumer groups.

Components:
- internal.Partition: append-only log with per-consumer committed offsets
- Topic: collection of partitions and partitioning logic
- Producer: sends messages to a topic
- Consumer: simple pull consumer that commits offsets on success

Design goals:
- Simple, extensible, unit-testable
- In-memory for now; persistence can be added later
- At-least-once delivery via consumer commit

## Prerequisites
- Go 1.25+
- Docker (for building container)
- Helm (for deploying to Kubernetes)

## Build
- Build binary: `make build`
- Build Docker image: `make docker` (image: `mq:latest`)

## Run locally
- Generate example config: `make config` (writes `config.json`)
- Run server using config: `make run` (reads `config.json` by default)
- Or run directly: `./bin/mq-server -listen :8080 -partitions 3`
- Health: `curl http://localhost:8080/healthz`
- Produce: `curl "http://localhost:8080/produce?key=abc&payload=hello"`

## Deploy to Kubernetes (Helm)
- `make helm` will install the chart in the current kubecontext

## Testing
- Unit tests: `make test`
- Coverage: `go test ./... -coverprofile=coverage.out`

## Config file
- The server accepts a JSON config file with these keys:
  - listen: HTTP bind address
  - partitions: number of partitions
- Provide path with `-config` flag or set `CONFIG_PATH` env var.

## WAL and extensibility notes
- The partition implementation keeps messages in-memory. To add WAL:
  - Implement a Write-Ahead Log interface (append, read, sync) under `internal/wal`.
  - Make `internal.Partition` accept a WAL implementation and write messages to WAL before appending to in-memory slice.
  - On startup, replay WAL to restore in-memory state.

## Repo layout
- cmd/mq-server: simple HTTP server exposing produce endpoint
- internal: partition and message implementations
- mq: public API (topic, producer, consumer)
- chart/mq: Helm chart for deployment

Notes
- Currently in-memory only. Add WAL/persistence and replication for production use.
