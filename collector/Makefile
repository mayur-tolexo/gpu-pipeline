BINARY= collector
IMAGE= collector:latest
CONFIG= config.json
KIND_CLUSTER= mq-cluster
K8S_NS= telemetry
K8S_MANIFEST= k8s/postgres-deployment.yaml

.PHONY: all build docker test coverage run clean config kind-create kind-load kind-deploy kind-delete port-forward k8s-deploy k8s-verify k8s-logs k8s-cleanup k8s-db-connect k8s-query k8s-backup k8s-restore k8s-test-data

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

## Deploy to Kubernetes (Kind)
kind-deploy: kind-create kind-load k8s-deploy

## Delete cluster
kind-delete:
	kind delete cluster --name $(KIND_CLUSTER)

## Port forward the service
port-forward:
	kubectl port-forward svc/collector 8081:8081 -n $(K8S_NS)

# ===== Kubernetes Deployment Commands =====

## Deploy PostgreSQL and Collector to Kubernetes
k8s-deploy: docker
	@echo "Deploying PostgreSQL and Collector..."
	kubectl apply -f $(K8S_MANIFEST)
	@echo "Waiting for PostgreSQL to be ready..."
	kubectl wait --for=condition=ready pod -l app=postgres -n $(K8S_NS) --timeout=300s
	@echo "Waiting for Collector to be ready..."
	kubectl wait --for=condition=ready pod -l app=collector -n $(K8S_NS) --timeout=300s
	@echo "✓ Deployment complete!"

## Verify Kubernetes deployment
k8s-verify:
	@echo "=== Namespace ==="
	kubectl get namespace $(K8S_NS)
	@echo "\n=== Secrets ==="
	kubectl get secrets -n $(K8S_NS)
	@echo "\n=== PersistentVolumeClaims ==="
	kubectl get pvc -n $(K8S_NS)
	@echo "\n=== Services ==="
	kubectl get svc -n $(K8S_NS)
	@echo "\n=== StatefulSets ==="
	kubectl get statefulsets -n $(K8S_NS)
	@echo "\n=== Deployments ==="
	kubectl get deployments -n $(K8S_NS)
	@echo "\n=== Pods ==="
	kubectl get pods -n $(K8S_NS)
	@echo "\n=== HPA ==="
	kubectl get hpa -n $(K8S_NS)
	@echo "\n=== Pod Events ==="
	kubectl get events -n $(K8S_NS) --sort-by='.lastTimestamp'

## View Kubernetes logs
k8s-logs:
	@if [ -z "$(POD)" ]; then \
		echo "Usage: make k8s-logs POD=<pod_name>"; \
		echo "Available pods:"; \
		kubectl get pods -n $(K8S_NS) | awk 'NR>1 {print "  " $$1}'; \
	else \
		kubectl logs -f $(POD) -n $(K8S_NS); \
	fi

## Watch all logs in telemetry namespace
k8s-logs-all:
	kubectl logs -f -l app=collector -n $(K8S_NS) --all-containers

## Connect to PostgreSQL pod
k8s-db-connect:
	@POD=$$(kubectl get pod -l app=postgres -n $(K8S_NS) -o jsonpath='{.items[0].metadata.name}'); \
	echo "Connecting to PostgreSQL pod: $$POD"; \
	kubectl exec -it $$POD -n $(K8S_NS) -- psql -U postgres -d telemetry

## Execute query in PostgreSQL
k8s-query:
	@if [ -z "$(SQL)" ]; then \
		echo "Usage: make k8s-query SQL='SELECT COUNT(*) FROM telemetry;'"; \
	else \
		POD=$$(kubectl get pod -l app=postgres -n $(K8S_NS) -o jsonpath='{.items[0].metadata.name}'); \
		kubectl exec $$POD -n $(K8S_NS) -- psql -U postgres -d telemetry -c "$(SQL)"; \
	fi

## Insert test data into PostgreSQL
k8s-test-data:
	@echo "Inserting test data..."
	@POD=$$(kubectl get pod -l app=postgres -n $(K8S_NS) -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec $$POD -n $(K8S_NS) -- psql -U postgres -d telemetry -c \
	"INSERT INTO telemetry (gpu_id, timestamp, data) VALUES \
	('gpu-001', NOW(), '{\"usage\": 45.2}'), \
	('gpu-002', NOW(), '{\"usage\": 67.8}'), \
	('gpu-003', NOW(), '{\"usage\": 23.1}')" 2>/dev/null || \
	echo "Note: Table may not exist yet if using GORM migration"

## Backup PostgreSQL database
k8s-backup:
	@BACKUP_FILE=$${BACKUP_FILE:-telemetry-backup-$$(date +%s).sql}; \
	echo "Creating backup to $$BACKUP_FILE..."; \
	POD=$$(kubectl get pod -l app=postgres -n $(K8S_NS) -o jsonpath='{.items[0].metadata.name}'); \
	kubectl exec $$POD -n $(K8S_NS) -- pg_dump -U postgres telemetry > $$BACKUP_FILE; \
	echo "✓ Backup created: $$BACKUP_FILE"

## Restore PostgreSQL database
k8s-restore:
	@if [ -z "$(BACKUP)" ]; then \
		echo "Usage: make k8s-restore BACKUP=<backup_file>"; \
		echo "Example: make k8s-restore BACKUP=telemetry-backup-123456.sql"; \
	else \
		echo "Restoring from $(BACKUP)..."; \
		POD=$$(kubectl get pod -l app=postgres -n $(K8S_NS) -o jsonpath='{.items[0].metadata.name}'); \
		kubectl exec -i $$POD -n $(K8S_NS) -- psql -U postgres telemetry < $(BACKUP); \
		echo "✓ Restore complete"; \
	fi

## Scale Collector replicas
k8s-scale:
	@if [ -z "$(REPLICAS)" ]; then \
		echo "Usage: make k8s-scale REPLICAS=<number>"; \
		echo "Current replicas:"; \
		kubectl get deployment collector -n $(K8S_NS) -o jsonpath='{.spec.replicas}'; \
		echo ""; \
	else \
		echo "Scaling Collector to $(REPLICAS) replicas..."; \
		kubectl scale deployment collector --replicas=$(REPLICAS) -n $(K8S_NS); \
		kubectl get deployment collector -n $(K8S_NS); \
	fi

## View HPA status
k8s-hpa:
	kubectl describe hpa collector-hpa -n $(K8S_NS)

## View resource usage
k8s-resources:
	@echo "=== Pod Resource Usage ==="
	kubectl top pods -n $(K8S_NS) || echo "Metrics not available. Install metrics-server."
	@echo "\n=== Node Resource Usage ==="
	kubectl top nodes || echo "Metrics not available. Install metrics-server."

## Clean up Kubernetes deployment
k8s-cleanup:
	@echo "Deleting deployment..."
	kubectl delete namespace $(K8S_NS)
	@echo "✓ Cleanup complete"

## Watch pods until ready
k8s-watch:
	kubectl get pods -n $(K8S_NS) -w
