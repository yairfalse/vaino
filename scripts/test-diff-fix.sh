#!/bin/bash

echo "Testing vaino diff with single snapshot scenario"
echo "=============================================="

# Create a temporary test directory
TEST_DIR="/tmp/vaino-diff-test-$$"
mkdir -p "$TEST_DIR/.vaino/history"

# Save current HOME and set test HOME
ORIGINAL_HOME=$HOME
export HOME=$TEST_DIR

echo "1. Testing with NO snapshots (should show helpful error):"
./vaino diff
echo -e "\nExit code: $?\n"

echo "2. Creating single snapshot in history:"
cat > "$TEST_DIR/.vaino/history/snapshot-001.json" << EOF
{
  "id": "test-snapshot-001",
  "timestamp": "2025-01-15T10:00:00Z",
  "provider": "test",
  "resources": []
}
EOF

echo "3. Testing with ONE snapshot (should show 'nothing to compare' error):"
./vaino diff
echo -e "\nExit code: $?\n"

echo "4. Creating second snapshot:"
cat > "$TEST_DIR/.vaino/history/snapshot-002.json" << EOF
{
  "id": "test-snapshot-002", 
  "timestamp": "2025-01-15T11:00:00Z",
  "provider": "test",
  "resources": []
}
EOF

echo "5. Testing with TWO snapshots (should work):"
./vaino diff --quiet
echo -e "Exit code: $?\n"

# Cleanup
export HOME=$ORIGINAL_HOME
rm -rf "$TEST_DIR"

echo "Test complete!"