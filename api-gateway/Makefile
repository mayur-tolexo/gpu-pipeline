
.PHONY: all build test coverage clean docker lint help run kind-create kind-delete docker-build docker-load kind-deploy swagger

BINARY_NAME=api-gateway
BINARY_PATH=bin/$(BINARY_NAME)
IMAGE_NAME=gpu-pipeline/api-gateway:latest
KIND_CLUSTER=gpu-pipeline
CONFIG_FILE=config.json

# ============ Help ============
help:
	@echo "================================"
	@echo "API Gateway Makefile Commands"
	@echo "================================"
	@echo ""
	@echo "Build & Test:"
	@echo "  make build              - Build the API Gateway binary"
	@echo "  make test               - Run all tests"
	@echo "  make coverage           - Show test coverage with HTML report"
	@echo "  make coverage-check     - Check if coverage is >= 80%"
	@echo "  make lint               - Run linter (fmt, vet)"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build       - Build Docker image"
	@echo "  make docker-push        - Push Docker image to registry"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make kind-create        - Create Kind cluster"
	@echo "  make docker-load        - Load Docker image into Kind"
	@echo "  make kind-deploy        - Deploy to Kind cluster"
	@echo "  make kind-delete        - Delete Kind cluster"
	@echo "  make port-forward       - Port-forward API Gateway (8000:8000)"
	@echo ""
	@echo "Documentation:"
	@echo "  make swagger            - Generate Swagger/OpenAPI documentation"
	@echo ""
	@echo "Development:"
	@echo "  make run                - Build and run locally"
	@echo "  make config             - Generate default config.json"
	@echo "  make clean              - Clean build artifacts"
	@echo "  make all                - Build everything"

# ============ Build ============
all: clean build

build:
	@echo "Building $(BINARY_NAME)..."
	@mkdir -p bin
	@go build -o $(BINARY_PATH) ./cmd/main.go

# ============ Test & Coverage ============
test:
	@echo "Running tests..."
	@go test -v -race ./...

coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out -covermode=atomic ./... || true
	@go tool cover -func=coverage.out | tail -20
	@go tool cover -html=coverage.out -o coverage.html
	@echo "✓ Coverage report generated: coverage.html"

coverage-check:
	@echo "Checking if coverage >= 80%..."
	@go test -coverprofile=coverage.out ./... || true
	@TOTAL=$$(go tool cover -func=coverage.out | awk '/total:/ {print $$3}' | sed 's/%//'); \
	if (( $$(echo "$$TOTAL >= 80" | bc -l) )); then \
		echo "✓ Coverage: $$TOTAL% (>= 80%)"; \
	else \
		echo "✗ Coverage: $$TOTAL% (< 80%)"; \
		exit 1; \
	fi

# ============ Linting ============
lint:
	@echo "Running linter..."
	@go fmt ./...
	@go vet ./...
	@echo "✓ Linting complete"

# ============ Documentation ============
swagger:
	@echo "Generating Swagger/OpenAPI documentation..."
	@swag init -g ./cmd/main.go
	@echo "✓ Swagger documentation generated in docs/"

# ============ Docker ============
docker-build:
	@echo "Building Docker image: $(IMAGE_NAME)"
	@docker build -t $(IMAGE_NAME) --build-arg ROOT=. -f ./Dockerfile ..
	@echo "✓ Docker image built successfully"

docker-push:
	@echo "Pushing Docker image to registry..."
	@docker push $(IMAGE_NAME)
	@echo "✓ Docker image pushed"

# ============ Kubernetes ============
kind-create:
	@echo "Creating Kind cluster: $(KIND_CLUSTER)"
	@kind get clusters | grep -q $(KIND_CLUSTER) || kind create cluster --name $(KIND_CLUSTER)
	@echo "✓ Kind cluster ready"

kind-delete:
	@echo "Deleting Kind cluster: $(KIND_CLUSTER)"
	@kind delete cluster --name $(KIND_CLUSTER)
	@echo "✓ Kind cluster deleted"

docker-load: docker-build
	@echo "Loading Docker image into Kind cluster..."
	@kind load docker-image $(IMAGE_NAME) --name $(KIND_CLUSTER)
	@echo "✓ Docker image loaded into Kind"

kind-deploy: kind-create docker-load
	@echo "Deploying API Gateway to Kind cluster..."
	@kubectl apply -f deployment/k8s/namespace.yaml
	@kubectl apply -f deployment/k8s/api-gateway.yaml
	@kubectl apply -f deployment/k8s/api-gateway-service.yaml
	@echo "✓ API Gateway deployed"
	@echo "Waiting for deployment to be ready..."
	@kubectl wait --for=condition=available --timeout=300s deployment/api-gateway -n gpu-pipeline
	@echo "✓ Deployment is ready"

port-forward:
	@echo "Port-forwarding API Gateway (8000:8000)..."
	@kubectl port-forward svc/api-gateway-service 8000:8000 -n gpu-pipeline

# ============ Configuration ============
config:
	@echo "Generating default config.json..."
	@echo '{"listen":":8000","mq_url":"http://mq-service:8080","db_url":"postgres://user:pass@postgres:5432/telemetry"}' > $(CONFIG_FILE)
	@echo "✓ Config file created: $(CONFIG_FILE)"

# ============ Development ============
run: build config
	@echo "Running $(BINARY_NAME)..."
	@./$(BINARY_PATH)

# ============ Cleanup ============
clean:
	@echo "Cleaning up..."
	@rm -rf bin coverage.out coverage.html
	@go clean
	@echo "✓ Cleanup complete"

