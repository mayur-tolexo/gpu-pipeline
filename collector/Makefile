BINARY= collector
IMAGE= collector:latest
CONFIG= config.json
KIND_CLUSTER= mq-cluster

.PHONY: all build docker test coverage run clean config kind-create kind-load kind-deploy kind-delete port-forward

all: build

build:
	go build -o bin/$(BINARY) ./cmd

docker: build
	docker build -t $(IMAGE) .

test:
	go test ./internal -v

# Run unit tests for collector and show coverage summary
coverage:
	@echo "Running collector tests with coverage..."
	@go test ./internal -coverprofile=coverage.out || true
	@go tool cover -func=coverage.out | sed -n '1,200p'
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}'); echo "Total coverage: $$TOTAL"

run: build
	./bin/$(BINARY)

config:
	@echo '{"listen":":8081","topic":"telemetry","group":"collector-group"}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG) coverage.out

## Create Kind cluster
kind-create:
	kind get clusters | grep -q $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER)

## Load image into Kind
kind-load: docker
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## Deploy to Kubernetes
kind-deploy: kind-create kind-load
	kubectl apply -f k8s/deployment.yaml

## Delete cluster
kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)

## Port forward the service
port-forward:
	kubectl port-forward svc/collector 8081:8081
