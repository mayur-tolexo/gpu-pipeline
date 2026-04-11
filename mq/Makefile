BINARY= mq-server
IMAGE= mq-server:latest
CONFIG= config.json
KIND_CLUSTER= mq-cluster

.PHONY: all build docker test lint helm run clean config kind-create kind-load kind-deploy kind-delete

all: build

build:
	go build -o bin/$(BINARY) ./cmd/mq-server

docker:
	docker build -t $(IMAGE) --build-arg ROOT=. -f ./Dockerfile ..

test:
	go test ./client ./internal ./api -v

coverage:
	@echo "Running md tests with coverage..."
	@go test ./client ./internal ./api -coverprofile=coverage.out || true
	@go tool cover -func=coverage.out | sed -n '1,200p'
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}'); echo "Total coverage: $$TOTAL"

swagger:
	swag init -g ./cmd/mq-server/main.go

helm:
	helm upgrade --install mq chart/mq -f chart/mq/values.yaml

run: build
	./bin/$(BINARY) -config $(CONFIG)

config:
	@echo '{"listen":":8080","partitions":3}' > $(CONFIG) && echo "wrote $(CONFIG)"

clean:
	rm -f bin/$(BINARY) $(CONFIG) docs/

## Create Kind cluster
kind-create:
	kind get clusters | grep -q $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER)

## Load image into Kind
kind-load:
	kind load docker-image $(IMAGE) --name $(KIND_CLUSTER)

## Deploy to Kubernetes
kind-deploy: kind-create docker-build kind-load
	kubectl apply -f deployments/k8s/

## Delete cluster
kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)

## Port fording the service
port-forward:
	kubectl port-forward svc/mq-service 8080:80

## Create topic for telemetry data
create-topic:
	curl -X POST http://localhost:8080/topics \
	-H "Content-Type: application/json" \
	-d '{"name": "telemetry", "partitions": 10}'
