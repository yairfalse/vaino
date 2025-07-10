#!/bin/bash
# Test drift detection by adding JarJar

set -e

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

print_step() {
    echo -e "\n${BLUE}[STEP]${NC} $1"
}

print_info() {
    echo -e "${YELLOW}[INFO]${NC} $1"
}

print_success() {
    echo -e "${GREEN}[✓]${NC} $1"
}

echo "🎪 JarJar Drift Detection Test"
echo "=============================="
echo "Mesa gonna test yousa drift detection!"
echo ""

# Check if test environment is running
if ! kubectl get namespace test-workloads &>/dev/null; then
    echo "❌ Test environment not running!"
    echo "   Run: ./test/test-env.sh start"
    exit 1
fi

# Step 1: Initial scan
print_step "1. Running initial Kubernetes scan"
vaino scan --provider kubernetes --namespace test-workloads

# Step 2: Create baseline
print_step "2. Creating baseline (before JarJar)"
vaino baseline create --name before-jarjar --description "State before JarJar arrives"

print_info "Current pods:"
kubectl get pods -n test-workloads --no-headers | wc -l | xargs echo "Pod count:"

# Step 3: Add JarJar
print_step "3. Adding JarJar to the cluster"
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
kubectl apply -f "$SCRIPT_DIR/jarjar.yaml"

print_info "Waiting for JarJar to start..."
kubectl wait --for=condition=ready pod/jarjar -n test-workloads --timeout=30s || true

print_success "JarJar has arrived! 'Mesa here!'"
echo ""
print_info "JarJar's wisdom:"
kubectl logs jarjar -n test-workloads --tail=3 || true

# Step 4: Scan again
print_step "4. Scanning again (with JarJar)"
vaino scan --provider kubernetes --namespace test-workloads

# Step 5: Check for drift
print_step "5. Checking for drift"
echo "Expected: Drift should be detected (new pod added)"
echo ""

if vaino check --baseline before-jarjar; then
    echo -e "${RED}❌ No drift detected - This is wrong! JarJar should cause drift!${NC}"
else
    echo -e "${GREEN}✅ Drift detected correctly! JarJar has disturbed the force!${NC}"
fi

# Step 6: Show the diff
print_step "6. Viewing the differences"
vaino diff || true

# Step 7: Let's make JarJar even more annoying by scaling him
print_step "7. Making JarJar multiply (creating more pods)"
cat <<EOF | kubectl apply -f -
apiVersion: apps/v1
kind: Deployment
metadata:
  name: jarjar-army
  namespace: test-workloads
spec:
  replicas: 3
  selector:
    matchLabels:
      app: jarjar-clone
  template:
    metadata:
      labels:
        app: jarjar-clone
    spec:
      containers:
      - name: jarjar-clone
        image: busybox
        command: ['sh', '-c', 'echo "Mesa $(hostname)! More JarJars!" && sleep 3600']
EOF

print_info "Waiting for JarJar army..."
sleep 5

# Step 8: Final scan
print_step "8. Final scan (with JarJar army)"
vaino scan --provider kubernetes --namespace test-workloads

# Step 9: Check drift again
print_step "9. Checking drift again"
vaino check --baseline before-jarjar || true

# Summary
echo ""
echo "📊 Test Summary"
echo "=============="
kubectl get pods -n test-workloads -l 'app in (jarjar, jarjar-clone)' 
echo ""
print_info "Total JarJars in cluster: $(kubectl get pods -n test-workloads -l 'app in (jarjar, jarjar-clone)' --no-headers | wc -l)"

# Cleanup option
echo ""
echo "🧹 Cleanup Commands:"
echo "  Remove JarJar:      kubectl delete -f $SCRIPT_DIR/jarjar.yaml"
echo "  Remove JarJar army: kubectl delete deployment jarjar-army -n test-workloads"
echo "  Remove baseline:    vaino baseline delete before-jarjar"