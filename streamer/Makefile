# App
BINARY= streamer
IMAGE= streamer:latest
CONFIG= config.json
KIND_CLUSTER= mq-cluster

.PHONY: all build docker test coverage lint helm run clean config kind-create kind-load kind-deploy kind-delete

all: build

build:
	go build -o bin/$(BINARY) ./cmd

docker:
	docker build -t $(IMAGE) --build-arg ROOT=. -f ./Dockerfile ..

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

helm:
	helm upgrade --install streamer chart/streamer -f chart/streamer/values.yaml

run: build
	CSV_FILE=./data/telemetry.csv \
	MQ_URL=http://localhost:8080 \
	TOPIC=telemetry \
	STREAM_INTERVAL_MS=5000 \
	./bin/$(BINARY)

config:
	@echo '{"topic":"telemetry","interval_ms":5000}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG) coverage.out docs/

## Create Kind cluster
kind-create:
	kind get clusters | grep -q $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER)

## Load image into Kind
kind-load:
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## Deploy to Kubernetes
kind-deploy: kind-create docker kind-load
	kubectl apply -f ../deployment/k8s/## Create ConfigMap from CSV
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