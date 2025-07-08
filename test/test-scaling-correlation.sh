#!/bin/bash
# Test scaling correlation pattern specifically

set -e

echo "Testing scaling correlation pattern..."

# Create test snapshots
BASELINE=$(mktemp)
SCALED=$(mktemp)

# Baseline snapshot with 3 replicas
cat > "$BASELINE" << 'EOF'
{
  "id": "test-baseline",
  "timestamp": "2023-01-01T10:00:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "deployment/test-app",
      "type": "deployment",
      "name": "test-app",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "replicas": 3,
        "image": "test:v1.0"
      },
      "metadata": {
        "version": "100"
      }
    }
  ]
}
EOF

# Scaled snapshot with 5 replicas
cat > "$SCALED" << 'EOF'
{
  "id": "test-scaled",
  "timestamp": "2023-01-01T10:02:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "deployment/test-app",
      "type": "deployment",
      "name": "test-app",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "replicas": 5,
        "image": "test:v1.0"
      },
      "metadata": {
        "version": "101"
      }
    }
  ]
}
EOF

# Test correlation
OUTPUT=$(./wgo changes --from "$BASELINE" --to "$SCALED" --correlated 2>&1)

# Verify scaling was detected
if ! echo "$OUTPUT" | grep -q "test-app Scaling"; then
    echo "ERROR: Scaling correlation not detected"
    exit 1
fi

if ! echo "$OUTPUT" | grep -q "Scaled from 3 to 5 replicas"; then
    echo "ERROR: Scaling description incorrect"
    exit 1
fi

if ! echo "$OUTPUT" | grep -q "● 🔗"; then
    echo "ERROR: High confidence indicator missing"
    exit 1
fi

# Cleanup
rm -f "$BASELINE" "$SCALED"

echo "✅ Scaling correlation test passed"