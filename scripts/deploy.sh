#!/bin/bash

# TradSys Production Deployment Script
# This script deploys the complete TradSys HFT system to Kubernetes

set -e

# Configuration
NAMESPACE="tradsys"
IMAGE_TAG="${IMAGE_TAG:-v2.0}"
REGISTRY="${REGISTRY:-docker.io/tradsys}"
ENVIRONMENT="${ENVIRONMENT:-production}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."
    
    # Check if kubectl is installed
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    # Check if docker is installed
    if ! command -v docker &> /dev/null; then
        log_error "docker is not installed. Please install docker first."
        exit 1
    fi
    
    # Check if we can connect to Kubernetes cluster
    if ! kubectl cluster-info &> /dev/null; then
        log_error "Cannot connect to Kubernetes cluster. Please check your kubeconfig."
        exit 1
    fi
    
    log_success "Prerequisites check passed"
}

# Build Docker image
build_image() {
    log_info "Building TradSys Docker image..."
    
    docker build -t "${REGISTRY}/tradsys:${IMAGE_TAG}" .
    docker build -t "${REGISTRY}/tradsys:latest" .
    
    log_success "Docker image built successfully"
}

# Push Docker image
push_image() {
    log_info "Pushing Docker image to registry..."
    
    docker push "${REGISTRY}/tradsys:${IMAGE_TAG}"
    docker push "${REGISTRY}/tradsys:latest"
    
    log_success "Docker image pushed successfully"
}

# Create namespace and secrets
setup_namespace() {
    log_info "Setting up namespace and secrets..."
    
    # Apply namespace
    kubectl apply -f deployments/kubernetes/namespace.yaml
    
    # Create secrets if they don't exist
    if ! kubectl get secret tradsys-secrets -n ${NAMESPACE} &> /dev/null; then
        log_info "Creating secrets..."
        kubectl create secret generic tradsys-secrets \
            --from-literal=db-username=tradsys \
            --from-literal=db-password=$(openssl rand -base64 32) \
            --from-literal=redis-password=$(openssl rand -base64 32) \
            --from-literal=jwt-secret=$(openssl rand -base64 64) \
            --from-literal=grafana-password=$(openssl rand -base64 16) \
            -n ${NAMESPACE}
        log_success "Secrets created"
    else
        log_info "Secrets already exist, skipping creation"
    fi
}

# Deploy infrastructure components
deploy_infrastructure() {
    log_info "Deploying infrastructure components..."
    
    # Deploy PostgreSQL
    kubectl apply -f deployments/kubernetes/postgres.yaml
    
    # Wait for PostgreSQL to be ready
    log_info "Waiting for PostgreSQL to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=postgres -n ${NAMESPACE} --timeout=300s
    
    log_success "Infrastructure components deployed"
}

# Deploy monitoring stack
deploy_monitoring() {
    log_info "Deploying monitoring stack..."
    
    kubectl apply -f deployments/kubernetes/monitoring.yaml
    
    # Wait for monitoring components to be ready
    log_info "Waiting for monitoring components to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=prometheus -n ${NAMESPACE} --timeout=300s
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=grafana -n ${NAMESPACE} --timeout=300s
    
    log_success "Monitoring stack deployed"
}

# Deploy TradSys application
deploy_application() {
    log_info "Deploying TradSys application..."
    
    # Apply ConfigMaps
    kubectl apply -f deployments/kubernetes/configmap.yaml
    
    # Apply main deployment
    kubectl apply -f deployments/kubernetes/tradsys-deployment.yaml
    
    # Wait for application to be ready
    log_info "Waiting for TradSys application to be ready..."
    kubectl wait --for=condition=ready pod -l app.kubernetes.io/name=tradsys -n ${NAMESPACE} --timeout=600s
    
    log_success "TradSys application deployed"
}

# Verify deployment
verify_deployment() {
    log_info "Verifying deployment..."
    
    # Check pod status
    log_info "Pod status:"
    kubectl get pods -n ${NAMESPACE}
    
    # Check service status
    log_info "Service status:"
    kubectl get services -n ${NAMESPACE}
    
    # Check if TradSys is responding
    log_info "Checking TradSys health endpoint..."
    if kubectl exec -n ${NAMESPACE} deployment/tradsys-core -- wget -q --spider http://localhost:8080/health; then
        log_success "TradSys health check passed"
    else
        log_warning "TradSys health check failed, but deployment may still be starting"
    fi
    
    # Display access information
    log_info "Deployment verification complete!"
    echo ""
    echo "Access Information:"
    echo "==================="
    echo "TradSys API: kubectl port-forward -n ${NAMESPACE} svc/tradsys-core 8080:80"
    echo "Grafana: kubectl port-forward -n ${NAMESPACE} svc/grafana 3000:3000"
    echo "Prometheus: kubectl port-forward -n ${NAMESPACE} svc/prometheus 9090:9090"
    echo ""
    echo "To get Grafana admin password:"
    echo "kubectl get secret tradsys-secrets -n ${NAMESPACE} -o jsonpath='{.data.grafana-password}' | base64 -d"
}

# Cleanup function
cleanup() {
    log_info "Cleaning up deployment..."
    
    kubectl delete namespace ${NAMESPACE} --ignore-not-found=true
    
    log_success "Cleanup completed"
}

# Main deployment function
deploy() {
    log_info "Starting TradSys deployment to ${ENVIRONMENT} environment..."
    
    check_prerequisites
    build_image
    
    if [[ "${PUSH_IMAGE}" == "true" ]]; then
        push_image
    fi
    
    setup_namespace
    deploy_infrastructure
    deploy_monitoring
    deploy_application
    verify_deployment
    
    log_success "TradSys deployment completed successfully!"
}

# Parse command line arguments
case "${1:-deploy}" in
    "deploy")
        deploy
        ;;
    "cleanup")
        cleanup
        ;;
    "build")
        check_prerequisites
        build_image
        ;;
    "verify")
        verify_deployment
        ;;
    *)
        echo "Usage: $0 {deploy|cleanup|build|verify}"
        echo ""
        echo "Commands:"
        echo "  deploy  - Full deployment (default)"
        echo "  cleanup - Remove all deployed resources"
        echo "  build   - Build Docker image only"
        echo "  verify  - Verify existing deployment"
        echo ""
        echo "Environment Variables:"
        echo "  IMAGE_TAG     - Docker image tag (default: v2.0)"
        echo "  REGISTRY      - Docker registry (default: docker.io/tradsys)"
        echo "  ENVIRONMENT   - Deployment environment (default: production)"
        echo "  PUSH_IMAGE    - Push image to registry (default: false)"
        exit 1
        ;;
esac
