#!/bin/bash
# Simple changes detection demo

set -e

echo "📊 WGO Simple Change Detection Demo"
echo "==================================="
echo ""

# Clean state
echo "🧹 Starting with clean state..."
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl delete deployment jarjar-army -n test-workloads 2>/dev/null || true
sleep 2

# Initial scan
echo ""
echo "📸 Taking snapshot T1..."
SNAPSHOT1=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT1" > /dev/null
echo "✅ Snapshot saved"

# Make changes
echo ""
echo "🔧 Making changes..."
echo "  • Adding JarJar pod"
kubectl apply -f ./test/jarjar.yaml > /dev/null
echo "  • Scaling frontend to 5 replicas"
kubectl scale deployment frontend --replicas=5 -n test-workloads > /dev/null
sleep 3

# Second scan
echo ""
echo "📸 Taking snapshot T2..."
SNAPSHOT2=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT2" > /dev/null
echo "✅ Snapshot saved"

# Show changes
echo ""
echo "📊 What changed between T1 and T2?"
echo ""
./wgo changes --from "$SNAPSHOT1" --to "$SNAPSHOT2"

# More changes
echo ""
echo "🔧 Making more changes..."
echo "  • Removing redis deployment"
kubectl delete deployment redis -n test-workloads > /dev/null
echo "  • Creating JarJar army (3 replicas)"
kubectl create deployment jarjar-army --image=busybox --replicas=3 -n test-workloads -- sh -c 'sleep 3600' > /dev/null
sleep 3

# Third scan
echo ""
echo "📸 Taking snapshot T3..."
SNAPSHOT3=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT3" > /dev/null
echo "✅ Snapshot saved"

# Show all changes
echo ""
echo "📊 What changed between T1 and T3?"
echo ""
./wgo changes --from "$SNAPSHOT1" --to "$SNAPSHOT3"

# Time-based query
echo ""
echo "📊 Changes in the last 30 seconds:"
echo ""
./wgo changes --provider kubernetes --namespace test-workloads --since 30s

# Cleanup
echo ""
echo "🧹 Cleaning up..."
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl delete deployment jarjar-army -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads > /dev/null
kubectl apply -f ./test/k8s-workloads.yaml > /dev/null

rm -f "$SNAPSHOT1" "$SNAPSHOT2" "$SNAPSHOT3"

echo ""
echo "✅ Demo complete!"