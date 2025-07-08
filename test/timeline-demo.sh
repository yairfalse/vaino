#!/bin/bash
# Demo of timeline visualization

set -e

echo "📅 WGO Timeline Demo"
echo "==================="
echo ""

# Clean state
echo "🧹 Starting clean..."
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads 2>/dev/null || true
sleep 2

# Take baseline
echo "📸 Taking baseline..."
BASELINE=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$BASELINE" > /dev/null

echo "⏰ Step 1: Frontend scaling (00:00)"
kubectl scale deployment frontend --replicas=5 -n test-workloads
sleep 3

SNAPSHOT1=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT1" > /dev/null

echo "⏰ Step 2: Config update (00:05)"
kubectl patch configmap app-config -n test-workloads --type merge -p '{"data":{"deployment_time":"'$(date +%s)'"}}'
sleep 3

SNAPSHOT2=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT2" > /dev/null

echo "⏰ Step 3: New service deployment (00:10)"
kubectl apply -f ./test/jarjar.yaml
sleep 3

SNAPSHOT3=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT3" > /dev/null

echo "⏰ Step 4: API scaling (00:15)"
kubectl scale deployment api-server --replicas=3 -n test-workloads
sleep 3

SNAPSHOT4=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT4" > /dev/null

echo ""
echo "📊 Timeline View:"
echo "================="
./wgo changes --from "$BASELINE" --to "$SNAPSHOT4" --timeline

echo ""
echo "📊 Correlated View:"
echo "=================="
./wgo changes --from "$BASELINE" --to "$SNAPSHOT4" --correlated

echo ""
echo "📊 Regular View:"
echo "==============="
./wgo changes --from "$BASELINE" --to "$SNAPSHOT4"

# Cleanup
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads
kubectl scale deployment api-server --replicas=2 -n test-workloads
rm -f "$BASELINE" "$SNAPSHOT1" "$SNAPSHOT2" "$SNAPSHOT3" "$SNAPSHOT4"

echo ""
echo "✅ Timeline demo complete!"