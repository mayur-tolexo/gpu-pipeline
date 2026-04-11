# App
BINARY= streamer
IMAGE= streamer:latest
CONFIG= config.json
KIND_CLUSTER= gpu-pipeline

.PHONY: all build docker test coverage lint helm run clean config kind-create kind-load kind-deploy kind-delete help

help:
	@echo "Streamer Service - Available Commands"
	@echo ""
	@echo "Build & Test:"
	@echo "  make build              - Build streamer binary"
	@echo "  make test               - Run tests"
	@echo "  make coverage           - Show test coverage"
	@echo "  make lint               - Format and lint code"
	@echo ""
	@echo "Docker:"
	@echo "  make docker             - Build Docker image"
	@echo ""
	@echo "Local Development:"
	@echo "  make run                - Run streamer locally (requires MQ running)"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make deploy             - Deploy to Kubernetes (expects deployment files)"
	@echo "  make logs               - View streamer pod logs"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean              - Clean build artifacts"

all: build

build:
	go build -o bin/$(BINARY) ./cmd

docker:
	docker build -t $(IMAGE) --build-arg ROOT=.. -f ./Dockerfile ..

test:
	go test ./internal -v

coverage:
	@echo "Running streamer tests with coverage..."
	@go test ./internal -coverprofile=coverage.out || true
	@go tool cover -func=coverage.out | sed -n '1,200p'
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}'); echo "Total coverage: $$TOTAL"

lint:
	go fmt ./...
	go vet ./...

run: build
	@echo "Starting streamer (requires MQ running at http://localhost:8080)"
	CSV_FILE=./data/telemetry.csv \
	MQ_URL=http://localhost:8080 \
	TOPIC=telemetry \
	STREAM_INTERVAL_MS=5000 \
	./bin/$(BINARY)

config:
	@echo '{"topic":"telemetry","interval_ms":5000}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG) coverage.out

# Kubernetes targets (use from root Makefile instead)
logs:
	@echo "Viewing streamer logs..."
	kubectl logs -l app=streamer -n gpu-pipeline -f
