#!/bin/bash
# Test timeline accuracy

set -e

echo "Testing timeline accuracy..."

# Create test snapshots with specific timestamps
SNAPSHOT1=$(mktemp)
SNAPSHOT2=$(mktemp) 
SNAPSHOT3=$(mktemp)

# Snapshot 1 - 10:00:00
cat > "$SNAPSHOT1" << 'EOF'
{
  "id": "timeline-1",
  "timestamp": "2023-01-01T10:00:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "deployment/app",
      "type": "deployment",
      "name": "app",
      "configuration": {"replicas": 1}
    }
  ]
}
EOF

# Snapshot 2 - 10:02:00  
cat > "$SNAPSHOT2" << 'EOF'
{
  "id": "timeline-2", 
  "timestamp": "2023-01-01T10:02:00Z",
  "provider": "kubernetes",
  "resources": [
    {
      "id": "deployment/app",
      "type": "deployment",
      "name": "app", 
      "configuration": {"replicas": 3}
    }
  ]
}
EOF

# Snapshot 3 - 10:05:00
cat > "$SNAPSHOT3" << 'EOF'
{
  "id": "timeline-3",
  "timestamp": "2023-01-01T10:05:00Z", 
  "provider": "kubernetes",
  "resources": [
    {
      "id": "deployment/app",
      "type": "deployment",
      "name": "app",
      "configuration": {"replicas": 3}
    },
    {
      "id": "service/app-service",
      "type": "service", 
      "name": "app-service",
      "configuration": {"port": 8080}
    }
  ]
}
EOF

# Test timeline
OUTPUT=$(./wgo changes --from "$SNAPSHOT1" --to "$SNAPSHOT3" --timeline 2>&1)

# Verify timeline output
if ! echo "$OUTPUT" | grep -q "📅 Change Timeline"; then
    echo "ERROR: Timeline header missing"
    exit 1
fi

if ! echo "$OUTPUT" | grep -q "━"; then
    echo "ERROR: Timeline bar missing"
    exit 1
fi

if ! echo "$OUTPUT" | grep -q "▲"; then
    echo "ERROR: Timeline markers missing" 
    exit 1
fi

# Should show time range
if ! echo "$OUTPUT" | grep -q "10:00.*10:05"; then
    echo "ERROR: Time range incorrect"
    exit 1
fi

# Should show change groups
if ! echo "$OUTPUT" | grep -q "changes)"; then
    echo "ERROR: Change count missing"
    exit 1
fi

# Cleanup
rm -f "$SNAPSHOT1" "$SNAPSHOT2" "$SNAPSHOT3"

echo "✅ Timeline accuracy test passed"