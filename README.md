# Overview
This project implements an elastic, scalable telemetry pipeline for GPU clusters using a custom-built message queue (without Kafka/RabbitMQ).

The system is designed with a strong focus on:
- Decoupled architecture
- Reusability of infrastructure (MQ)
- Scalability and fault tolerance

# Architecture
The system is composed of independently deployable services:

## Telemetry Streamer
- Generates and streams telemetry data from CSV in a continuous loop.
- Custom Message Queue (MQ)
- A partitioned, pull-based messaging system supporting:
  - At-least-once delivery
  - Consumer groups
  - Offset tracking
  - Horizontal scalability
- Telemetry Collector
  - Consumes messages from MQ, processes telemetry, and persists data.
- API Gateway
  - Exposes telemetry data via REST APIs with auto-generated OpenAPI specs.

# Roadmap (Implementation Plan)
We follow a bottom-up approach, building the system layer by layer:

## Phase 1: Core Message Queue (Foundation)
- [ ] Message abstraction (generic payload)  
- [ ] Topic & partition model. 
- [ ] Partitioning strategy (hash-based)
- [ ] Thread-safe append/read
- [ ] Consumer groups
- [ ] Offset tracking
- [ ] At-least-once delivery semantics

## Phase 2: MQ Service Layer
- [ ] HTTP server for MQ
- [ ] Publish API
- [ ] Consume API
- [ ] Ack API
- [ ] Error handling & validation
- [ ] Configurable partitions

## Phase 3: MQ Client SDK
- [ ] Producer client (for streamer)
- [ ] Consumer client (for collector)
- [ ] Retry logic
- [ ] Configurable batching

## Phase 4: Telemetry Streamer
- [ ] CSV reader
- [ ] Continuous streaming loop
- [ ] Configurable rate
- [ ] Horizontal scalability

## Phase 5: Telemetry Collector
- [ ] Consumer group integration
- [ ] Message parsing
- [ ] Idempotent processing
- [ ] Batch processing

## Phase 6: Storage Layer
- [ ] Schema design (GPU telemetry)
- [ ] Indexing (gpu_id + timestamp)
- [ ] Efficient time-range queries

## Phase 7: API Gateway
- [ ] List GPUs endpoint
- [ ] Query telemetry endpoint
- [ ] Time-range filtering
- [ ] OpenAPI auto-generation

## Phase 8: Deployment
- [ ] Dockerfiles for all services
- [ ] Helm charts
- [ ] Kubernetes deployment configs
- [ ] Scaling configs

## Phase 9: Testing & Observability
- [ ] Unit tests (MQ core)
- [ ] Integration tests (pipeline)
- [ ] Logging
- [ ] Metrics (optional bonus)

# Future Improvements
- Disk-backed persistence (WAL)
- Replication & leader election
- Consumer group rebalancing
- gRPC instead of HTTP
- Compression & batching
- Observability (Prometheus + Grafana)

# Tech Stack
- Language: Go
- Messaging: Custom-built MQ
- Deployment: Docker + Kubernetes
- Packaging: Helm
- API Docs: OpenAPI (Swagger)

# Key Features
- Fully decoupled microservices architecture
- Generic, reusable message queue implementation
- Partition-based scalability and ordering guarantees
- Consumer group support with offset management
- Kubernetes-ready with Helm charts
- Clean, idiomatic Go code with unit tests

# Design Philosophy
- Separation of concerns: MQ is infrastructure, not domain-specific
- Extensibility: Each component is independently scalable and reusable
- Simplicity over over-engineering: Focus on correctness and clarity
- Production mindset: Observability, failure handling, and clean APIs