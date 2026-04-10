#!/bin/bash

# Kubernetes PostgreSQL + Collector Deployment Script
# Usage: ./deploy.sh [command] [options]
# Commands:
#   deploy         - Deploy full stack (PostgreSQL + Collector)
#   verify         - Verify deployment status
#   logs           - Show logs
#   cleanup        - Delete all resources
#   backup         - Backup PostgreSQL data
#   restore <file> - Restore PostgreSQL from backup

set -e

NAMESPACE="telemetry"
COLLECTOR_IMAGE="collector:latest"
POSTGRES_IMAGE="postgres:15-alpine"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl not found. Please install kubectl."
        exit 1
    fi
    
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Create namespace
create_namespace() {
    if kubectl get namespace $NAMESPACE &> /dev/null; then
        log_warning "Namespace $NAMESPACE already exists"
    else
        log_info "Creating namespace $NAMESPACE..."
        kubectl create namespace $NAMESPACE
        log_success "Namespace created"
    fi
}

# Deploy full stack
deploy() {
    log_info "Starting deployment..."
    
    check_prerequisites
    create_namespace
    
    # Apply deployment
    log_info "Applying postgres-deployment.yaml..."
    kubectl apply -f postgres-deployment.yaml -n $NAMESPACE
    
    log_success "Deployment applied"
    
    # Wait for PostgreSQL
    log_info "Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=Ready pod -l app=postgres -n $NAMESPACE --timeout=300s
    log_success "PostgreSQL is ready"
    
    # Wait for Collector
    log_info "Waiting for Collector to be ready..."
    kubectl wait --for=condition=Ready pod -l app=collector -n $NAMESPACE --timeout=300s
    log_success "Collector is ready"
    
    log_success "Full deployment completed successfully!"
    
    # Show summary
    show_summary
}

# Verify deployment
verify() {
    log_info "Verifying deployment..."
    
    echo ""
    echo "=== Namespace ==="
    kubectl get namespace $NAMESPACE || true
    
    echo ""
    echo "=== Secrets ==="
    kubectl get secrets -n $NAMESPACE || true
    
    echo ""
    echo "=== PersistentVolumeClaims ==="
    kubectl get pvc -n $NAMESPACE || true
    
    echo ""
    echo "=== Services ==="
    kubectl get svc -n $NAMESPACE || true
    
    echo ""
    echo "=== Pods ==="
    kubectl get pods -n $NAMESPACE -o wide || true
    
    echo ""
    echo "=== StatefulSet ==="
    kubectl get statefulset -n $NAMESPACE || true
    
    echo ""
    echo "=== Deployment ==="
    kubectl get deployment -n $NAMESPACE || true
    
    echo ""
    echo "=== HorizontalPodAutoscaler ==="
    kubectl get hpa -n $NAMESPACE || true
    
    echo ""
    echo "=== Events ==="
    kubectl get events -n $NAMESPACE --sort-by='.lastTimestamp' | tail -20 || true
}

# Show logs
show_logs() {
    local component="${1:-all}"
    
    case $component in
        postgres)
            log_info "PostgreSQL logs:"
            kubectl logs postgres-0 -n $NAMESPACE -f
            ;;
        collector)
            log_info "Collector logs:"
            kubectl logs -l app=collector -n $NAMESPACE -f
            ;;
        *)
            log_info "Collector logs:"
            kubectl logs -l app=collector -n $NAMESPACE --tail=50
            echo ""
            log_info "PostgreSQL logs:"
            kubectl logs postgres-0 -n $NAMESPACE --tail=50
            ;;
    esac
}

# Show summary
show_summary() {
    echo ""
    echo "=========================================="
    echo "    Deployment Summary"
    echo "=========================================="
    echo ""
    echo "Namespace: $NAMESPACE"
    echo ""
    
    # Get PostgreSQL info
    echo "PostgreSQL:"
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}' 2>/dev/null || echo "N/A")
    echo "  Pod: $POSTGRES_POD"
    POSTGRES_IP=$(kubectl get svc postgres -n $NAMESPACE -o jsonpath='{.spec.clusterIP}' 2>/dev/null || echo "N/A")
    echo "  Service IP: postgres.telemetry.svc.cluster.local:5432"
    echo "  Username: postgres"
    echo "  Database: telemetry"
    echo ""
    
    # Get Collector info
    echo "Collector:"
    COLLECTOR_REPLICAS=$(kubectl get deployment collector -n $NAMESPACE -o jsonpath='{.status.replicas}' 2>/dev/null || echo "0")
    echo "  Replicas: $COLLECTOR_REPLICAS/3"
    echo "  Service: http://collector.telemetry.svc.cluster.local:8081"
    echo ""
    
    echo "Useful Commands:"
    echo "  View Collector logs:"
    echo "    kubectl logs -l app=collector -n $NAMESPACE -f"
    echo ""
    echo "  View PostgreSQL logs:"
    echo "    kubectl logs postgres-0 -n $NAMESPACE -f"
    echo ""
    echo "  Connect to PostgreSQL:"
    echo "    kubectl exec -it postgres-0 -n $NAMESPACE -- psql -U postgres -d telemetry"
    echo ""
    echo "  Port-forward Collector:"
    echo "    kubectl port-forward svc/collector 8081:8081 -n $NAMESPACE"
    echo ""
    echo "  Scale Collector replicas:"
    echo "    kubectl scale deployment collector --replicas=5 -n $NAMESPACE"
    echo ""
    echo "=========================================="
    echo ""
}

# Cleanup deployment
cleanup() {
    log_warning "This will delete all resources in namespace $NAMESPACE"
    read -p "Are you sure? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        log_info "Cleanup cancelled"
        return
    fi
    
    log_info "Deleting namespace $NAMESPACE..."
    kubectl delete namespace $NAMESPACE --ignore-not-found=true
    
    log_success "Cleanup completed"
}

# Backup PostgreSQL
backup() {
    local backup_file="${1:-telemetry-backup-$(date +%Y%m%d-%H%M%S).sql}"
    
    log_info "Creating backup to $backup_file..."
    
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    
    kubectl exec -i $POSTGRES_POD -n $NAMESPACE -- pg_dump -U postgres telemetry > "$backup_file"
    
    log_success "Backup created: $backup_file"
    echo "File size: $(du -h "$backup_file" | cut -f1)"
}

# Restore PostgreSQL
restore() {
    local backup_file="$1"
    
    if [ -z "$backup_file" ]; then
        log_error "Usage: ./deploy.sh restore <backup-file>"
        exit 1
    fi
    
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: $backup_file"
        exit 1
    fi
    
    log_warning "This will overwrite existing data in PostgreSQL"
    read -p "Are you sure? (yes/no): " confirm
    
    if [ "$confirm" != "yes" ]; then
        log_info "Restore cancelled"
        return
    fi
    
    log_info "Restoring from $backup_file..."
    
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    
    kubectl exec -i $POSTGRES_POD -n $NAMESPACE -- psql -U postgres telemetry < "$backup_file"
    
    log_success "Restore completed"
}

# Connect to PostgreSQL
connect_postgres() {
    log_info "Connecting to PostgreSQL..."
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    kubectl exec -it $POSTGRES_POD -n $NAMESPACE -- psql -U postgres -d telemetry
}

# Connect to Collector pod (exec shell)
connect_collector() {
    log_info "Connecting to Collector pod..."
    COLLECTOR_POD=$(kubectl get pod -l app=collector -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    kubectl exec -it $COLLECTOR_POD -n $NAMESPACE -- sh
}

# Test data insertion
test_data() {
    log_info "Testing data insertion..."
    
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    
    # Insert test data
    kubectl exec -i $POSTGRES_POD -n $NAMESPACE -- psql -U postgres -d telemetry << EOF
INSERT INTO telemetry (gpu_id, timestamp, data) VALUES
('gpu-001', NOW(), '{"utilization": 85.5, "memory": 24576}'),
('gpu-002', NOW(), '{"utilization": 72.3, "memory": 16384}'),
('gpu-003', NOW(), '{"utilization": 91.2, "memory": 28672}');
EOF
    
    log_success "Test data inserted"
    
    # Verify
    log_info "Verifying data..."
    kubectl exec -i $POSTGRES_POD -n $NAMESPACE -- psql -U postgres -d telemetry << EOF
SELECT COUNT(*) as record_count FROM telemetry;
SELECT * FROM telemetry ORDER BY created_at DESC LIMIT 3;
EOF
}

# Query data
query_data() {
    local query="${1:-SELECT COUNT(*) FROM telemetry;}"
    
    POSTGRES_POD=$(kubectl get pod -l app=postgres -n $NAMESPACE -o jsonpath='{.items[0].metadata.name}')
    
    kubectl exec -i $POSTGRES_POD -n $NAMESPACE -- psql -U postgres -d telemetry -c "$query"
}

# Main entry point
main() {
    local command="${1:-help}"
    
    case $command in
        deploy)
            deploy
            ;;
        verify)
            verify
            ;;
        logs)
            show_logs "${2:-all}"
            ;;
        cleanup)
            cleanup
            ;;
        backup)
            backup "${2}"
            ;;
        restore)
            restore "${2}"
            ;;
        connect-postgres)
            connect_postgres
            ;;
        connect-collector)
            connect_collector
            ;;
        test-data)
            test_data
            ;;
        query)
            query_data "${2}"
            ;;
        summary)
            show_summary
            ;;
        help|*)
            cat << EOF
PostgreSQL + Collector Deployment Script

Usage: $0 [command] [options]

Commands:
  deploy                    - Deploy full stack (PostgreSQL + Collector)
  verify                    - Verify deployment status
  logs [postgres|collector] - Show logs (default: all)
  cleanup                   - Delete all resources
  backup [filename]         - Backup PostgreSQL data
  restore <filename>        - Restore PostgreSQL from backup
  connect-postgres          - Connect to PostgreSQL shell
  connect-collector         - Connect to Collector pod shell
  test-data                 - Insert test data
  query [sql]              - Execute SQL query
  summary                   - Show deployment summary
  help                      - Show this help

Examples:
  # Deploy everything
  $0 deploy

  # View Collector logs
  $0 logs collector

  # Backup database
  $0 backup

  # Connect to PostgreSQL
  $0 connect-postgres

  # Query data
  $0 query "SELECT COUNT(*) FROM telemetry;"

EOF
            ;;
    esac
}

# Run main
main "$@"
