#!/bin/bash
# Test service deployment correlation pattern

set -e

echo "Testing service deployment correlation pattern..."

# Create test snapshots
BEFORE=$(mktemp)
AFTER=$(mktemp)

# Before snapshot - no service
cat > "$BEFORE" << 'EOF'
{
  "id": "test-before",
  "timestamp": "2023-01-01T10:00:00Z", 
  "provider": "kubernetes",
  "resources": []
}
EOF

# After snapshot - service + related resources
cat > "$AFTER" << 'EOF'
{
  "id": "test-after",
  "timestamp": "2023-01-01T10:02:00Z",
  "provider": "kubernetes", 
  "resources": [
    {
      "id": "service/api-service",
      "type": "service",
      "name": "api-service",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "ports": [{"port": 8080}]
      }
    },
    {
      "id": "deployment/api",
      "type": "deployment",
      "name": "api",
      "provider": "kubernetes", 
      "namespace": "default",
      "configuration": {
        "replicas": 2,
        "image": "api:v1.0"
      }
    },
    {
      "id": "configmap/api-config",
      "type": "configmap",
      "name": "api-config",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "data": {"env": "prod"}
      }
    }
  ]
}
EOF

# Test correlation
OUTPUT=$(./wgo changes --from "$BEFORE" --to "$AFTER" --correlated 2>&1)

# Verify service deployment pattern
if ! echo "$OUTPUT" | grep -q "New Service: api-service"; then
    echo "ERROR: Service deployment correlation not detected"
    exit 1
fi

# Should group related resources
if ! echo "$OUTPUT" | grep -q "deployment/api"; then
    echo "ERROR: Related deployment not correlated"
    exit 1
fi

# Cleanup
rm -f "$BEFORE" "$AFTER"

echo "✅ Service deployment correlation test passed"