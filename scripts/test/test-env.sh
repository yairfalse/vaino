#!/bin/bash
# VAINO Test Environment Manager
# Works on macOS and Ubuntu

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
CLUSTER_NAME="vaino-test"
LOCALSTACK_CONTAINER="vaino-localstack"
TEST_NAMESPACE="test-workloads"

# Functions
print_status() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_error() {
    echo -e "${RED}[✗]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[i]${NC} $1"
}

check_dependencies() {
    local missing=()
    
    # Check Docker
    if ! command -v docker &> /dev/null; then
        missing+=("docker")
    fi
    
    # Check kind
    if ! command -v kind &> /dev/null; then
        missing+=("kind")
    fi
    
    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        missing+=("kubectl")
    fi
    
    if [ ${#missing[@]} -ne 0 ]; then
        print_error "Missing dependencies: ${missing[*]}"
        echo ""
        echo "Install instructions:"
        
        # Detect OS
        if [[ "$OSTYPE" == "darwin"* ]]; then
            echo "  # macOS (using Homebrew)"
            for dep in "${missing[@]}"; do
                case $dep in
                    docker)
                        echo "  brew install --cask docker"
                        ;;
                    kind)
                        echo "  brew install kind"
                        ;;
                    kubectl)
                        echo "  brew install kubectl"
                        ;;
                esac
            done
        else
            echo "  # Ubuntu/Debian"
            for dep in "${missing[@]}"; do
                case $dep in
                    docker)
                        echo "  sudo apt-get update && sudo apt-get install docker.io"
                        echo "  sudo usermod -aG docker $USER"
                        ;;
                    kind)
                        echo "  curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64"
                        echo "  chmod +x ./kind && sudo mv ./kind /usr/local/bin/kind"
                        ;;
                    kubectl)
                        echo "  sudo snap install kubectl --classic"
                        ;;
                esac
            done
        fi
        exit 1
    fi
    
    print_status "All dependencies installed"
}

start_k8s() {
    print_info "Starting Kubernetes cluster..."
    
    # Create kind cluster with extra port mappings
    cat <<EOF | kind create cluster --name "$CLUSTER_NAME" --config=-
kind: Cluster
apiVersion: kind.x-k8s.io/v1alpha4
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
EOF
    
    print_status "Kubernetes cluster started"
    
    # Wait for cluster to be ready
    print_info "Waiting for cluster to be ready..."
    kubectl wait --for=condition=ready node --all --timeout=60s
    
    # Deploy some test workloads
    deploy_test_workloads
}

deploy_test_workloads() {
    print_info "Deploying test workloads..."
    
    # Check if workloads yaml exists
    SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
    WORKLOADS_FILE="$SCRIPT_DIR/k8s-workloads.yaml"
    
    if [ -f "$WORKLOADS_FILE" ]; then
        kubectl apply -f "$WORKLOADS_FILE"
        
        # Wait for deployments to be ready
        print_info "Waiting for workloads to be ready..."
        kubectl wait --for=condition=available --timeout=120s deployment --all -n "$TEST_NAMESPACE" || true
        
        print_status "Test workloads deployed:"
        echo "  • Frontend (3 replicas) - nginx web server"
        echo "  • API Server (2 replicas) - httpbin API"
        echo "  • PostgreSQL database (StatefulSet)"
        echo "  • Redis cache"
        echo "  • RabbitMQ message queue"
        echo "  • Batch jobs and CronJobs"
        echo "  • ConfigMaps, Secrets, NetworkPolicies"
        echo "  • HPA and Ingress configured"
    else
        print_error "Workloads file not found, using simple deployment"
        # Fallback to simple deployments
        kubectl create namespace "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
        kubectl create deployment nginx --image=nginx:alpine -n "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
        kubectl expose deployment nginx --port=80 --type=NodePort -n "$TEST_NAMESPACE" --dry-run=client -o yaml | kubectl apply -f -
    fi
    
    print_status "Test workloads deployed"
}

start_localstack() {
    print_info "Starting LocalStack (AWS emulator)..."
    
    docker run -d \
        --name "$LOCALSTACK_CONTAINER" \
        -p 4566:4566 \
        -e SERVICES=s3,ec2,iam,lambda \
        -e DEFAULT_REGION=us-east-1 \
        -e DATA_DIR=/tmp/localstack/data \
        -v /tmp/localstack:/tmp/localstack \
        localstack/localstack:latest
    
    print_info "Waiting for LocalStack to be ready..."
    sleep 10
    
    # Create some test AWS resources
    create_aws_resources
    
    print_status "LocalStack started"
}

create_aws_resources() {
    print_info "Creating test AWS resources..."
    
    # Configure AWS CLI for LocalStack
    export AWS_ACCESS_KEY_ID=test
    export AWS_SECRET_ACCESS_KEY=test
    export AWS_DEFAULT_REGION=us-east-1
    export AWS_ENDPOINT_URL=http://localhost:4566
    
    # Create S3 buckets
    aws s3 mb s3://test-bucket-1 --endpoint-url=http://localhost:4566 || true
    aws s3 mb s3://test-bucket-2 --endpoint-url=http://localhost:4566 || true
    
    # Create EC2 instances (mock)
    aws ec2 run-instances \
        --image-id ami-12345678 \
        --count 2 \
        --instance-type t2.micro \
        --endpoint-url=http://localhost:4566 || true
    
    print_status "Test AWS resources created"
}

stop_k8s() {
    print_info "Stopping Kubernetes cluster..."
    kind delete cluster --name "$CLUSTER_NAME"
    print_status "Kubernetes cluster stopped"
}

stop_localstack() {
    print_info "Stopping LocalStack..."
    docker stop "$LOCALSTACK_CONTAINER" 2>/dev/null || true
    docker rm "$LOCALSTACK_CONTAINER" 2>/dev/null || true
    print_status "LocalStack stopped"
}

start_all() {
    check_dependencies
    
    echo "🚀 Starting VAINO Test Environment"
    echo "================================"
    
    start_k8s
    start_localstack
    
    echo ""
    echo "✅ Test environment is ready!"
    echo ""
    echo "📋 Quick test commands:"
    echo "  # Test Kubernetes scanning"
    echo "  vaino scan --provider kubernetes --namespace $TEST_NAMESPACE"
    echo ""
    echo "  # Test AWS scanning (LocalStack)"
    echo "  AWS_ENDPOINT_URL=http://localhost:4566 vaino scan --provider aws"
    echo ""
    echo "  # View Kubernetes resources"
    echo "  kubectl get all -n $TEST_NAMESPACE"
    echo ""
    echo "To stop: $0 stop"
}

stop_all() {
    echo "🛑 Stopping VAINO Test Environment"
    echo "================================"
    
    stop_k8s
    stop_localstack
    
    echo ""
    echo "✅ Test environment stopped"
}

status() {
    echo "📊 VAINO Test Environment Status"
    echo "=============================="
    echo ""
    
    # Check Kubernetes
    if kind get clusters 2>/dev/null | grep -q "$CLUSTER_NAME"; then
        print_status "Kubernetes cluster: Running"
        echo "    Workloads in $TEST_NAMESPACE:"
        kubectl get deployments -n "$TEST_NAMESPACE" 2>/dev/null | tail -n +2 | while read line; do
            echo "    - $line"
        done
    else
        print_error "Kubernetes cluster: Not running"
    fi
    
    echo ""
    
    # Check LocalStack
    if docker ps --format "table {{.Names}}" | grep -q "$LOCALSTACK_CONTAINER"; then
        print_status "LocalStack: Running"
        echo "    Endpoint: http://localhost:4566"
    else
        print_error "LocalStack: Not running"
    fi
}

# Main script
case "${1:-}" in
    start)
        start_all
        ;;
    stop)
        stop_all
        ;;
    restart)
        stop_all
        echo ""
        start_all
        ;;
    status)
        status
        ;;
    k8s-only)
        check_dependencies
        start_k8s
        ;;
    aws-only)
        check_dependencies
        start_localstack
        ;;
    add-jarjar)
        print_info "Adding JarJar to the cluster..."
        SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        kubectl apply -f "$SCRIPT_DIR/jarjar.yaml"
        print_success "JarJar deployed! Mesa gonna be muy muy helpful!"
        ;;
    remove-jarjar)
        print_info "Removing JarJar from the cluster..."
        SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
        kubectl delete -f "$SCRIPT_DIR/jarjar.yaml" 2>/dev/null || true
        kubectl delete deployment jarjar-army -n test-workloads 2>/dev/null || true
        print_success "JarJar removed. The senate will decide his fate."
        ;;
    *)
        echo "VAINO Test Environment Manager"
        echo ""
        echo "Usage: $0 {start|stop|restart|status|k8s-only|aws-only|add-jarjar|remove-jarjar}"
        echo ""
        echo "Commands:"
        echo "  start         - Start complete test environment (K8s + LocalStack)"
        echo "  stop          - Stop all test services"
        echo "  restart       - Restart test environment"
        echo "  status        - Show current status"
        echo "  k8s-only      - Start only Kubernetes"
        echo "  aws-only      - Start only LocalStack (AWS)"
        echo "  add-jarjar    - Deploy JarJar pod (for drift testing)"
        echo "  remove-jarjar - Remove JarJar from cluster"
        echo ""
        echo "Test drift detection:"
        echo "  ./test/test-drift.sh"
        echo ""
        echo "This script works on both macOS and Ubuntu!"
        exit 1
        ;;
esac