#!/bin/bash
# Test config update correlation pattern

set -e

echo "Testing config update correlation pattern..."

# Create test snapshots
BEFORE=$(mktemp)
AFTER=$(mktemp)

# Before snapshot
cat > "$BEFORE" << 'EOF'
{
  "id": "test-before",
  "timestamp": "2023-01-01T10:00:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "configmap/app-config", 
      "type": "configmap",
      "name": "app-config",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "data": {"version": "1.0"}
      },
      "metadata": {
        "version": "100"
      }
    },
    {
      "id": "deployment/app",
      "type": "deployment", 
      "name": "app",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "replicas": 2,
        "image": "app:v1.0"
      },
      "metadata": {
        "version": "200"
      }
    }
  ]
}
EOF

# After snapshot - config updated + deployment restarted
cat > "$AFTER" << 'EOF'
{
  "id": "test-after",
  "timestamp": "2023-01-01T10:02:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "configmap/app-config",
      "type": "configmap", 
      "name": "app-config",
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "data": {"version": "1.1"}
      },
      "metadata": {
        "version": "101"
      }
    },
    {
      "id": "deployment/app",
      "type": "deployment",
      "name": "app", 
      "provider": "kubernetes",
      "namespace": "default",
      "configuration": {
        "replicas": 2,
        "image": "app:v1.0"
      },
      "metadata": {
        "version": "201"
      }
    }
  ]
}
EOF

# Test correlation
OUTPUT=$(./vaino changes --from "$BEFORE" --to "$AFTER" --correlated 2>&1)

# Verify config update pattern
if ! echo "$OUTPUT" | grep -q "app-config Update"; then
    echo "ERROR: Config update correlation not detected"
    exit 1
fi

# Should correlate with deployment change
CONFIG_SECTION=$(echo "$OUTPUT" | sed -n '/app-config Update/,/🔗/p')
if ! echo "$CONFIG_SECTION" | grep -q "deployment/app"; then
    echo "ERROR: Deployment restart not correlated with config change"
    exit 1
fi

# Cleanup
rm -f "$BEFORE" "$AFTER"

echo "✅ Config update correlation test passed"