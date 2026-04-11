BINARY= collector
IMAGE= collector:latest
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
	@echo "Running collector tests with coverage..."
	@go test ./internal -coverprofile=coverage.out || true
	@go tool cover -func=coverage.out | sed -n '1,200p'
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}'); echo "Total coverage: $$TOTAL"

lint:
	go fmt ./...
	go vet ./...

helm:
	helm upgrade --install collector chart/collector -f chart/collector/values.yaml

run: build
	./bin/$(BINARY)

config:
	@echo '{"topic":"telemetry","group":"collector-group"}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG) coverage.out docs/

## Create Kind cluster
kind-create:
	kind get clusters | grep -q $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER)

## Load image into Kind
kind-load:
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## Deploy to Kubernetes
kind-deploy: kind-create docker-build kind-load
	kubectl apply -f ../deployment/k8s/