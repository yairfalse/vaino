#!/bin/bash
# Simple drift detection demo

set -e

echo "🎯 WGO Drift Detection Demo"
echo "=========================="
echo ""

# Step 1: Clean state
echo "1️⃣ Removing JarJar (if exists)..."
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
sleep 2

# Step 2: Create baseline
echo ""
echo "2️⃣ Creating baseline (clean state)..."
./wgo scan --provider kubernetes --namespace test-workloads --output-file baseline.json
echo "✅ Baseline saved to baseline.json"

# Step 3: Add JarJar
echo ""
echo "3️⃣ Adding JarJar to cause drift..."
kubectl apply -f ./test/jarjar.yaml
sleep 3

# Step 4: Scan current state
echo ""
echo "4️⃣ Scanning current state..."
./wgo scan --provider kubernetes --namespace test-workloads --output-file current.json
echo "✅ Current state saved to current.json"

# Step 5: Compare
echo ""
echo "5️⃣ Detecting drift..."
echo ""
./wgo diff --from baseline.json --to current.json

echo ""
echo "🎉 Demo complete! JarJar was detected as drift."
echo ""
echo "To see more drift, try:"
echo "  kubectl scale deployment frontend --replicas=5 -n test-workloads"
echo "  ./wgo scan --provider kubernetes --namespace test-workloads --output-file more-drift.json"
echo "  ./wgo diff --from baseline.json --to more-drift.json"