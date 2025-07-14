#!/bin/bash
# Demo of correlated change detection

set -e

echo "🔗 VAINO Correlated Changes Demo"
echo "=============================="
echo ""

# Clean state
echo "🧹 Starting clean..."
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads 2>/dev/null || true
sleep 2

# Take baseline
echo "📸 Taking baseline snapshot..."
BASELINE=$(mktemp)
./vaino scan --provider kubernetes --namespace test-workloads --output-file "$BASELINE" > /dev/null

# Scenario 1: Scaling with HPA
echo ""
echo "📈 Scenario 1: Simulating traffic spike..."
echo "  • Scaling frontend from 3 to 5 replicas"
kubectl scale deployment frontend --replicas=5 -n test-workloads
sleep 1

# Take snapshot
SNAPSHOT1=$(mktemp)
./vaino scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT1" > /dev/null

echo ""
echo "Regular changes view:"
echo "--------------------"
./vaino changes --from "$BASELINE" --to "$SNAPSHOT1"

echo ""
echo "Correlated changes view:"
echo "-----------------------"
./vaino changes --from "$BASELINE" --to "$SNAPSHOT1" --correlated

# Scenario 2: Config update and restart
echo ""
echo "🔧 Scenario 2: Configuration update..."
echo "  • Updating ConfigMap"
kubectl patch configmap app-config -n test-workloads --type merge -p '{"data":{"new_feature":"enabled"}}'
echo "  • This might trigger pod restarts..."
sleep 1

# Update deployment to force restart (in real scenario, this would be automatic)
kubectl rollout restart deployment api-server -n test-workloads 2>/dev/null || true
sleep 2

# Take snapshot
SNAPSHOT2=$(mktemp)
./vaino scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT2" > /dev/null

echo ""
echo "Config change correlation:"
echo "-------------------------"
./vaino changes --from "$SNAPSHOT1" --to "$SNAPSHOT2" --correlated

# Scenario 3: New service deployment
echo ""
echo "🚀 Scenario 3: Deploying new service (JarJar)..."
kubectl apply -f ./test/jarjar.yaml
sleep 2

# Take snapshot
SNAPSHOT3=$(mktemp)
./vaino scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT3" > /dev/null

echo ""
echo "New service correlation:"
echo "-----------------------"
./vaino changes --from "$SNAPSHOT2" --to "$SNAPSHOT3" --correlated

# Show all changes from baseline
echo ""
echo "📊 All changes from baseline (correlated):"
echo "========================================="
./vaino changes --from "$BASELINE" --to "$SNAPSHOT3" --correlated

# Cleanup
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads
rm -f "$BASELINE" "$SNAPSHOT1" "$SNAPSHOT2" "$SNAPSHOT3"

echo ""
echo "✅ Demo complete!"