NAMESPACE=gpu-pipeline
KIND_CLUSTER=gpu-pipeline
REGISTRY=localhost:5000

# ===== Development/Local Commands =====

.PHONY: help kind-create kind-delete docker-build-all docker-load-all build-all test coverage \
	deploy deploy-all verify logs logs-all cleanup watch kind-full helm-install helm-uninstall \
	api-gateway-port-forward swagger-ui

help:
	@echo "GPU Pipeline - Available Commands"
	@echo ""
	@echo "Local Development:"
	@echo "  make build-all          - Build all services"
	@echo "  make test               - Run all tests"
	@echo "  make coverage           - Show test coverage"
	@echo ""
	@echo "Kind Cluster:"
	@echo "  make kind-create        - Create Kind cluster"
	@echo "  make kind-delete        - Delete Kind cluster"
	@echo ""
	@echo "Docker:"
	@echo "  make docker-build-all   - Build all Docker images"
	@echo "  make docker-load-all    - Load images into Kind"
	@echo ""
	@echo "Kubernetes:"
	@echo "  make deploy             - Deploy to Kubernetes (creates Kind cluster if needed)"
	@echo "  make deploy-all         - Deploy all services"
	@echo "  make verify             - Verify all deployments"
	@echo "  make logs               - Show collector logs"
	@echo "  make logs-all           - Show all service logs"
	@echo "  make watch              - Watch all pods"
	@echo ""
	@echo "API Gateway:"
	@echo "  make api-gateway-port-forward - Port-forward API Gateway (8000 -> 8000)"
	@echo "  make swagger-ui          - Open Swagger UI in browser (after port-forward)"
	@echo ""
	@echo "Helm:"
	@echo "  make helm-install       - Install via Helm"
	@echo "  make helm-uninstall     - Uninstall Helm release"
	@echo ""
	@echo "Cleanup:"
	@echo "  make cleanup            - Delete namespace (keep Kind cluster)"
	@echo "  make kind-full          - Full reset (delete Kind cluster and start fresh)"

# ===== Build Commands =====

build-all:
	@echo "Building all services..."
	cd mq && make build
	cd streamer && make build
	cd collector && make build
	cd api-gateway && make build
	@echo "✓ All services built"

test:
	@echo "Running tests..."
	cd mq && make test
	cd streamer && make test
	cd collector && make test
	cd api-gateway && make test

coverage:
	@echo "Test coverage by service:"
	@echo "\n=== MQ ==="
	cd mq && make coverage || true
	@echo "\n=== Streamer ==="
	cd streamer && make coverage || true
	@echo "\n=== Collector ==="
	cd collector && make coverage || true
	@echo "\n=== API Gateway ==="
	cd api-gateway && make coverage || true

# ===== Kind Cluster Commands =====

kind-create:
	@if kind get clusters | grep -q $(KIND_CLUSTER); then \
		echo "✓ Kind cluster '$(KIND_CLUSTER)' already exists"; \
	else \
		echo "Creating Kind cluster '$(KIND_CLUSTER)'..."; \
		kind create cluster --name $(KIND_CLUSTER) --wait 5m; \
		echo "✓ Kind cluster created"; \
	fi

kind-delete:
	@echo "Deleting Kind cluster '$(KIND_CLUSTER)'..."
	kind delete cluster --name $(KIND_CLUSTER)
	@echo "✓ Kind cluster deleted"

# ===== Docker Build & Load =====

docker-build-all: build-all
	@echo "Building Docker images..."
	cd mq && make docker
	cd streamer && make docker
	cd collector && make docker
	cd api-gateway && make docker-build
	@echo "✓ All Docker images built"

docker-load-all: docker-build-all
	@echo "Loading images into Kind cluster..."
	kind load docker-image mq-server:latest --name $(KIND_CLUSTER)
	kind load docker-image streamer:latest --name $(KIND_CLUSTER)
	kind load docker-image collector:latest --name $(KIND_CLUSTER)
	kind load docker-image gpu-pipeline/api-gateway:latest --name $(KIND_CLUSTER)
	@echo "✓ All images loaded into Kind cluster"

# ===== Kubernetes Deployment =====

deploy: kind-create docker-load-all deploy-all verify

deploy-all:
	@echo "Deploying all services to namespace $(NAMESPACE)..."
	kubectl apply -f deployment/k8s/namespace.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/postgres.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/mq.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/mq-service.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/job-topic.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/collector.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/streamer.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/api-gateway.yaml
	kubectl apply -n $(NAMESPACE) -f deployment/k8s/api-gateway-service.yaml
	@echo "✓ All services deployed"
	@echo "Waiting for deployments to be ready..."
	kubectl wait --for=condition=available --timeout=300s deployment/mq -n $(NAMESPACE) || true
	kubectl wait --for=condition=available --timeout=300s deployment/collector -n $(NAMESPACE) || true
	kubectl wait --for=condition=available --timeout=300s deployment/streamer -n $(NAMESPACE) || true
	kubectl wait --for=condition=available --timeout=300s deployment/api-gateway -n $(NAMESPACE) || true
	@echo "✓ Deployments ready"

verify:
	@echo "=== Namespace ==="
	kubectl get namespace $(NAMESPACE)
	@echo "\n=== Pods ==="
	kubectl get pods -n $(NAMESPACE)
	@echo "\n=== Services ==="
	kubectl get svc -n $(NAMESPACE)
	@echo "\n=== Deployments ==="
	kubectl get deployments -n $(NAMESPACE)
	@echo "\n=== Pod Status Details ==="
	kubectl describe pods -n $(NAMESPACE) | grep -A 5 "Status:"

logs:
	kubectl logs -n $(NAMESPACE) -l app=collector -f

logs-all:
	@echo "Logs from all services:"
	@echo "=== MQ ==="
	kubectl logs -n $(NAMESPACE) -l app=mq --tail=20 || echo "No MQ logs"
	@echo "\n=== Collector ==="
	kubectl logs -n $(NAMESPACE) -l app=collector --tail=20 || echo "No Collector logs"
	@echo "\n=== Streamer ==="
	kubectl logs -n $(NAMESPACE) -l app=streamer --tail=20 || echo "No Streamer logs"
	@echo "\n=== API Gateway ==="
	kubectl logs -n $(NAMESPACE) -l app=api-gateway --tail=20 || echo "No API Gateway logs"

watch:
	kubectl get pods -n $(NAMESPACE) -w

# ===== API Gateway =====

api-gateway-port-forward:
	@echo "Port-forwarding API Gateway..."
	@echo "Swagger UI will be available at: http://localhost:8000/swagger/"
	@echo "API health check: http://localhost:8000/api/v1/health"
	kubectl port-forward -n $(NAMESPACE) svc/api-gateway-service 8000:8000

swagger-ui:
	@echo "Opening Swagger UI in browser..."
	@open http://localhost:8000/swagger/ || xdg-open http://localhost:8000/swagger/ || echo "Please open http://localhost:8000/swagger/ in your browser"

# ===== Helm =====

helm-install:
	@echo "Installing with Helm..."
	helm install gpu-pipeline ./helm/gpu-pipeline -n $(NAMESPACE) --create-namespace
	@echo "✓ Helm installation complete"

helm-uninstall:
	@echo "Uninstalling Helm release..."
	helm uninstall gpu-pipeline -n $(NAMESPACE)
	@echo "✓ Helm release uninstalled"

# ===== Cleanup =====

cleanup:
	@echo "Cleaning up namespace $(NAMESPACE)..."
	kubectl delete namespace $(NAMESPACE) --ignore-not-found
	@echo "✓ Namespace deleted"

kind-full: kind-delete kind-create
	@echo "✓ Kind cluster reset complete"
	@$(MAKE) deploy