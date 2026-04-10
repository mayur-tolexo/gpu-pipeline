# 📦 Custom Message Queue (MQ)

## Overview

This module implements a lightweight, partitioned message queue built
from scratch in Go.

It is designed as a generic, reusable infrastructure component,
independent of any domain (e.g., GPU telemetry), and supports scalable,
concurrent message processing.

------------------------------------------------------------------------

## Goals

- Provide a simple and extensible message queue
- Support multiple producers and consumers
- Ensure ordering guarantees within partitions
- Enable horizontal scalability via partitioning
- Maintain at-least-once delivery semantics

------------------------------------------------------------------------

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

------------------------------------------------------------------------

### Topic

A logical stream of messages divided into partitions.

------------------------------------------------------------------------

### Partition

- Append-only log
- Guarantees ordering
- Independent unit of scalability

------------------------------------------------------------------------

### Consumer Group

- Tracks offsets per partition
- Enables parallel consumption
- Supports replay

------------------------------------------------------------------------

## Architecture

Producers → Topic → Partition → Consumers

------------------------------------------------------------------------

## Delivery Semantics

At-least-once delivery: - Messages are acknowledged after processing -
Failures lead to re-delivery

------------------------------------------------------------------------

## Limitations

- In-memory only (no persistence)
- No replication
- No rebalancing

------------------------------------------------------------------------

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
