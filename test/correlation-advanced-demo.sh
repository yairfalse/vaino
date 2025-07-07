#!/bin/bash
# Advanced demo of correlated change detection

set -e

echo "🔗 WGO Advanced Correlation Demo"
echo "================================"
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
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$BASELINE" > /dev/null

# Scenario 1: Full application deployment
echo ""
echo "🚀 Scenario 1: Deploying new application (JarJar)..."
echo "  Creating pod, service, configmap..."

# Create a configmap for jarjar
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: jarjar-config
  namespace: test-workloads
data:
  mesa_says: "yousa in big doo doo dis time"
  planet: "naboo"
EOF

# Create the pod and service
kubectl apply -f ./test/jarjar.yaml
sleep 2

# Take snapshot
SNAPSHOT1=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT1" > /dev/null

echo ""
echo "New application deployment correlation:"
echo "--------------------------------------"
./wgo changes --from "$BASELINE" --to "$SNAPSHOT1" --correlated

# Scenario 2: Network configuration change
echo ""
echo "🌐 Scenario 2: Network configuration change..."
echo "  • Updating ingress rules"
echo "  • Modifying service ports"

kubectl patch ingress main-ingress -n test-workloads --type='json' -p='[
  {"op": "add", "path": "/spec/rules/0/http/paths/-", "value": {
    "path": "/jarjar",
    "pathType": "Prefix",
    "backend": {
      "service": {
        "name": "jarjar-service",
        "port": {
          "number": 9999
        }
      }
    }
  }}
]'

kubectl patch service frontend -n test-workloads -p '{"spec":{"ports":[{"port":80,"targetPort":80,"protocol":"TCP","nodePort":30524}]}}'
sleep 2

# Take snapshot
SNAPSHOT2=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT2" > /dev/null

echo ""
echo "Network changes correlation:"
echo "---------------------------"
./wgo changes --from "$SNAPSHOT1" --to "$SNAPSHOT2" --correlated

# Scenario 3: Security rotation
echo ""
echo "🔐 Scenario 3: Coordinated secret rotation..."
echo "  • Rotating multiple secrets"

kubectl patch secret api-keys -n test-workloads -p '{"data":{"new_key":"c2VjcmV0"}}'
kubectl patch secret db-secret -n test-workloads -p '{"data":{"rotated":"dHJ1ZQ=="}}'
kubectl patch secret rabbitmq-secret -n test-workloads -p '{"data":{"updated":"eWVz"}}'
sleep 2

# Take snapshot
SNAPSHOT3=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT3" > /dev/null

echo ""
echo "Security changes correlation:"
echo "----------------------------"
./wgo changes --from "$SNAPSHOT2" --to "$SNAPSHOT3" --correlated

# Scenario 4: Mixed changes
echo ""
echo "🔄 Scenario 4: Mixed infrastructure changes..."
echo "  • Scaling deployment"
echo "  • Updating config"
echo "  • Modifying unrelated resources"

kubectl scale deployment api-server --replicas=3 -n test-workloads
kubectl patch configmap app-config -n test-workloads --type merge -p '{"data":{"version":"2.0"}}'
kubectl scale deployment redis --replicas=2 -n test-workloads
sleep 2

# Take snapshot
SNAPSHOT4=$(mktemp)
./wgo scan --provider kubernetes --namespace test-workloads --output-file "$SNAPSHOT4" > /dev/null

echo ""
echo "Mixed changes with correlation:"
echo "------------------------------"
./wgo changes --from "$SNAPSHOT3" --to "$SNAPSHOT4" --correlated

# Show timeline view
echo ""
echo "📊 Complete timeline from baseline:"
echo "==================================="
./wgo changes --from "$BASELINE" --to "$SNAPSHOT4" --correlated

# Cleanup
kubectl delete configmap jarjar-config -n test-workloads 2>/dev/null || true
kubectl delete pod jarjar -n test-workloads 2>/dev/null || true
kubectl delete service jarjar-service -n test-workloads 2>/dev/null || true
kubectl scale deployment frontend --replicas=3 -n test-workloads
kubectl scale deployment api-server --replicas=2 -n test-workloads
kubectl scale deployment redis --replicas=1 -n test-workloads

# Remove ingress patch
kubectl patch ingress main-ingress -n test-workloads --type='json' -p='[
  {"op": "remove", "path": "/spec/rules/0/http/paths/2"}
]' 2>/dev/null || true

rm -f "$BASELINE" "$SNAPSHOT1" "$SNAPSHOT2" "$SNAPSHOT3" "$SNAPSHOT4"

echo ""
echo "✅ Advanced demo complete!"
echo ""
echo "Key takeaways:"
echo "• WGO groups related changes intelligently"
echo "• Detects patterns: scaling, deployments, network changes, security rotations"
echo "• Makes infrastructure changes easier to understand"
echo "• Like 'git log' but for your infrastructure"