# Telemetry Streamer

## Overview

The Telemetry Streamer reads telemetry data from a CSV file and
continuously publishes it to the Message Queue (MQ).

It simulates real-time telemetry ingestion and is designed to scale
horizontally in Kubernetes.


## Configuration
```
  Variable             Description                   Default
  -------------------- ----------------------------- ---------------------
  CSV_FILE             Path to telemetry CSV file    /data/telemetry.csv
  MQ_URL               MQ service URL                http://mq-service
  TOPIC                MQ topic name                 telemetry
  STREAM_INTERVAL_MS   Delay between messages (ms)   5000
```

## Local Development

### Build

``` bash
make build
```

### Run

``` bash
make run
```

## Running with Docker

### Build Image

``` bash
make docker-build
```

### Run Container

``` bash
make docker-run
```


## Running on Kubernetes (Kind)

### Create ConfigMap from CSV

``` bash
make config
```

### Deploy Streamer

``` bash
make deploy
```

### Verify

``` bash
kubectl get pods
kubectl logs -l app=streamer
```

### View Logs

``` bash
make logs
```

## Data Injection Strategy
```
  Environment   Method
  ------------- ------------------
  Local         Direct file path
  Docker        Volume mount
  Kubernetes    ConfigMap
```

## Scalability

``` bash
kubectl scale deployment streamer --replicas=5
```

## Timestamp Handling

Each record's timestamp is overridden with the current processing time.

## Design Decisions

-   gpu_id used as partition key
-   ConfigMap for dynamic data injection
-   ENV-based config for flexibility

## Developer Workflow

``` bash
make run
make docker-build
make docker-run
make deploy
make logs
```
