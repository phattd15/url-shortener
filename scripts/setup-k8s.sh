#!/bin/bash

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_NAME="url-shortener-cluster"
REGISTRY_NAME="kind-registry"
REGISTRY_PORT="5001"

print_step() {
    echo -e "${BLUE}==>${NC} ${1}"
}

print_success() {
    echo -e "${GREEN}✓${NC} ${1}"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} ${1}"
}

print_error() {
    echo -e "${RED}✗${NC} ${1}"
}

check_prerequisites() {
    print_step "Checking prerequisites..."
    
    if ! command -v docker &> /dev/null; then
        print_error "Docker is not installed. Please install Docker first."
        exit 1
    fi
    
    if ! command -v kubectl &> /dev/null; then
        print_error "kubectl is not installed. Please install kubectl first."
        exit 1
    fi
    
    if ! command -v kind &> /dev/null; then
        print_warning "kind is not installed. Installing kind..."
        
        # Install kind
        if [[ "$OSTYPE" == "darwin"* ]]; then
            # macOS
            if command -v brew &> /dev/null; then
                brew install kind
            else
                curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-darwin-amd64
                chmod +x ./kind
                sudo mv ./kind /usr/local/bin/kind
            fi
        elif [[ "$OSTYPE" == "linux-gnu"* ]]; then
            # Linux
            curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
            chmod +x ./kind
            sudo mv ./kind /usr/local/bin/kind
        else
            print_error "Unsupported OS. Please install kind manually."
            exit 1
        fi
        
        print_success "kind installed successfully"
    fi
    
    print_success "All prerequisites are met"
}

setup_local_registry() {
    print_step "Setting up local Docker registry..."
    
    # Check if registry already exists
    if docker ps --format "table {{.Names}}" | grep -q "^${REGISTRY_NAME}$"; then
        print_warning "Registry ${REGISTRY_NAME} already exists"
    else
        docker run -d --restart=always -p "127.0.0.1:${REGISTRY_PORT}:5000" --name "${REGISTRY_NAME}" registry:2
        print_success "Local registry created at localhost:${REGISTRY_PORT}"
    fi
}

create_kind_cluster() {
    print_step "Creating kind cluster..."
    
    # Check if cluster already exists
    if kind get clusters | grep -q "^${CLUSTER_NAME}$"; then
        print_warning "Cluster ${CLUSTER_NAME} already exists"
        return
    fi
    
    # Create kind cluster config
    cat <<EOF > /tmp/kind-config.yaml
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
containerdConfigPatches:
- |-
  [plugins."io.containerd.grpc.v1.cri".registry.mirrors."localhost:${REGISTRY_PORT}"]
    endpoint = ["http://${REGISTRY_NAME}:5000"]
nodes:
- role: control-plane
  kubeadmConfigPatches:
  - |
    kind: InitConfiguration
    nodeRegistration:
      kubeletExtraArgs:
        node-labels: "ingress-ready=true"
  extraPortMappings:
  - containerPort: 80
    hostPort: 80
    protocol: TCP
  - containerPort: 443
    hostPort: 443
    protocol: TCP
- role: worker
- role: worker
EOF
    
    kind create cluster --name="${CLUSTER_NAME}" --config=/tmp/kind-config.yaml
    
    # Connect registry to cluster network
    docker network connect "kind" "${REGISTRY_NAME}" || true
    
    # Configure kubectl context
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
    print_success "Kind cluster created successfully"
}

install_ingress_controller() {
    print_step "Installing NGINX Ingress Controller..."
    
    kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/main/deploy/static/provider/kind/deploy.yaml
    
    print_step "Waiting for ingress controller to be ready..."
    kubectl wait --namespace ingress-nginx \
        --for=condition=ready pod \
        --selector=app.kubernetes.io/component=controller \
        --timeout=90s
    
    print_success "NGINX Ingress Controller installed"
}

build_and_push_image() {
    print_step "Building and pushing application image..."
    
    # Build the application image
    docker build -t "localhost:${REGISTRY_PORT}/url-shortener:latest" .
    
    # Push to local registry
    docker push "localhost:${REGISTRY_PORT}/url-shortener:latest"
    
    print_success "Application image built and pushed"
}

deploy_applications() {
    print_step "Deploying applications to Kubernetes..."
    
    # Apply namespace first
    kubectl apply -f manifests/namespace.yaml
    
    # Apply PostgreSQL
    print_step "Deploying PostgreSQL..."
    kubectl apply -f manifests/postgres/
    
    # Wait for PostgreSQL master to be ready
    print_step "Waiting for PostgreSQL master to be ready..."
    kubectl wait --namespace url-shortener \
        --for=condition=ready pod \
        --selector=app=postgres-master \
        --timeout=300s
    
    # Apply Redis
    print_step "Deploying Redis..."
    kubectl apply -f manifests/redis/
    
    # Wait for Redis master to be ready
    print_step "Waiting for Redis master to be ready..."
    kubectl wait --namespace url-shortener \
        --for=condition=ready pod \
        --selector=app=redis-master \
        --timeout=300s
    
    # Update app image to use local registry
    sed -i.bak "s|image: url-shortener:latest|image: localhost:${REGISTRY_PORT}/url-shortener:latest|g" manifests/app/url-shortener.yaml
    
    # Apply application
    print_step "Deploying URL Shortener application..."
    kubectl apply -f manifests/app/
    
    # Restore original app manifest
    mv manifests/app/url-shortener.yaml.bak manifests/app/url-shortener.yaml
    
    # Wait for application to be ready
    print_step "Waiting for application to be ready..."
    kubectl wait --namespace url-shortener \
        --for=condition=ready pod \
        --selector=app=url-shortener \
        --timeout=300s
    
    print_success "All applications deployed successfully"
}

show_status() {
    print_step "Checking deployment status..."
    
    echo ""
    echo "=== Cluster Information ==="
    kubectl cluster-info --context "kind-${CLUSTER_NAME}"
    
    echo ""
    echo "=== Pods Status ==="
    kubectl get pods -n url-shortener -o wide
    
    echo ""
    echo "=== Services ==="
    kubectl get services -n url-shortener
    
    echo ""
    echo "=== Ingress ==="
    kubectl get ingress -n url-shortener
    
    echo ""
    echo "=== Access Information ==="
    echo "Application URL: http://url-shortener.local"
    echo "Swagger Docs: http://url-shortener.local/swagger/index.html"
    echo ""
    echo "Add to /etc/hosts:"
    echo "127.0.0.1 url-shortener.local"
    echo ""
    echo "Or port-forward for direct access:"
    echo "kubectl port-forward -n url-shortener service/url-shortener-service 8080:80"
}

cleanup() {
    print_step "Cleaning up..."
    
    # Delete kind cluster
    kind delete cluster --name="${CLUSTER_NAME}"
    
    # Stop and remove local registry
    docker stop "${REGISTRY_NAME}" || true
    docker rm "${REGISTRY_NAME}" || true
    
    print_success "Cleanup completed"
}

main() {
    case "${1:-setup}" in
        "setup")
            check_prerequisites
            setup_local_registry
            create_kind_cluster
            install_ingress_controller
            build_and_push_image
            deploy_applications
            show_status
            ;;
        "cleanup")
            cleanup
            ;;
        "status")
            show_status
            ;;
        "rebuild")
            build_and_push_image
            kubectl rollout restart deployment/url-shortener -n url-shortener
            kubectl rollout status deployment/url-shortener -n url-shortener
            print_success "Application rebuilt and redeployed"
            ;;
        *)
            echo "Usage: $0 {setup|cleanup|status|rebuild}"
            echo ""
            echo "Commands:"
            echo "  setup   - Set up the entire stack (default)"
            echo "  cleanup - Delete the cluster and registry"
            echo "  status  - Show current status"
            echo "  rebuild - Rebuild and redeploy the application"
            exit 1
            ;;
    esac
}

main "$@" 