# App
APP_NAME=streamer
MAIN=cmd/main.go

# Image
IMAGE_NAME=streamer:latest
KIND_CLUSTER=mq-cluster

.PHONY: build run test docker-build docker-run kind-load deploy logs

## Build binary
build:
	go build -o bin/$(APP_NAME) $(MAIN)

## Run locally
run:
	CSV_FILE=./data/telemetry.csv \
	MQ_URL=http://localhost:8080 \
	TOPIC=telemetry \
	STREAM_INTERVAL_MS=5000 \
	./bin/$(APP_NAME)

## Run tests
test:
	go test ./... -cover

coverage:
	@echo "Running streamer tests with coverage..."
	@go test ./... -coverprofile=coverage.out || true
	@go tool cover -func=coverage.out | sed -n '1,200p'
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}'); echo "Total coverage: $$TOTAL"

## Build Docker image
docker-build:
	docker build -t $(IMAGE_NAME) .

## Run Docker (mount CSV)
docker-run:
	docker run -it --rm \
	-e CSV_FILE=/data/telemetry.csv \
	-e MQ_URL=http://host.docker.internal:8080 \
	-e TOPIC=telemetry \
	-e STREAM_INTERVAL_MS=5000 \
	-v $(PWD)/data:/data \
	$(IMAGE_NAME)

## Load image into Kind
kind-load:
	kind load docker-image $(IMAGE_NAME) --name $(KIND_CLUSTER)

## Create ConfigMap from CSV
config:
	kubectl create configmap telemetry-csv \
	--from-file=telemetry.csv=../data/telemetry.csv \
	--dry-run=client -o yaml | kubectl apply -f -

## Deploy to Kubernetes
deploy: docker-build kind-load config
	kubectl apply -f ../deployments/k8s/streamer.yaml

## View logs
logs:
	kubectl logs -l app=$(APP_NAME) -f