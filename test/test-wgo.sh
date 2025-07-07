#!/bin/bash
# WGO Test Runner - Tests WGO against the test environment

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m'

print_test() {
    echo -e "\n${YELLOW}[TEST]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

print_fail() {
    echo -e "${RED}[✗]${NC} $1"
}

# Check if test environment is running
check_env() {
    if ! kubectl get namespace test-workloads &>/dev/null; then
        echo "❌ Test environment not running!"
        echo "   Run: ./test/test-env.sh start"
        exit 1
    fi
}

run_tests() {
    echo "🧪 WGO Integration Tests"
    echo "======================="
    
    check_env
    
    # Test 1: Kubernetes Scan
    print_test "Testing Kubernetes scan..."
    if wgo scan --provider kubernetes --namespace test-workloads; then
        print_success "Kubernetes scan completed"
    else
        print_fail "Kubernetes scan failed"
    fi
    
    # Test 2: Create baseline
    print_test "Creating baseline..."
    if wgo baseline create --name k8s-test-baseline --description "Test K8s baseline"; then
        print_success "Baseline created"
    else
        print_fail "Baseline creation failed"
    fi
    
    # Test 3: Make a change
    print_test "Making infrastructure change..."
    kubectl scale deployment frontend --replicas=5 -n test-workloads
    sleep 5
    
    # Test 4: Check for drift
    print_test "Checking for drift..."
    if wgo check; then
        print_fail "No drift detected (expected drift!)"
    else
        print_success "Drift detected as expected"
    fi
    
    # Test 5: View diff
    print_test "Viewing differences..."
    if wgo diff; then
        print_success "Diff command completed"
    else
        print_fail "Diff command failed"
    fi
    
    # Test 6: AWS scan (if LocalStack running)
    if docker ps | grep -q wgo-localstack; then
        print_test "Testing AWS scan (LocalStack)..."
        export AWS_ACCESS_KEY_ID=test
        export AWS_SECRET_ACCESS_KEY=test
        export AWS_ENDPOINT_URL=http://localhost:4566
        
        if wgo scan --provider aws --region us-east-1; then
            print_success "AWS scan completed"
        else
            print_fail "AWS scan failed"
        fi
    fi
    
    # Test 7: Authentication commands
    print_test "Testing auth status..."
    if wgo auth status; then
        print_success "Auth status command works"
    else
        print_fail "Auth status failed"
    fi
    
    # Cleanup - restore original state
    print_test "Cleaning up..."
    kubectl scale deployment frontend --replicas=3 -n test-workloads
    
    echo ""
    echo "✅ Tests completed!"
}

# Quick scan test
quick_scan() {
    check_env
    echo "🔍 Quick Kubernetes Scan Test"
    echo "============================"
    
    wgo scan --provider kubernetes --namespace test-workloads
    
    echo ""
    echo "📊 Resources found in test environment"
}

# Main
case "${1:-}" in
    full)
        run_tests
        ;;
    scan)
        quick_scan
        ;;
    *)
        echo "WGO Test Runner"
        echo ""
        echo "Usage: $0 {full|scan}"
        echo ""
        echo "Commands:"
        echo "  full  - Run full test suite"
        echo "  scan  - Quick scan test only"
        echo ""
        echo "Make sure test environment is running:"
        echo "  ./test/test-env.sh start"
        ;;
esac