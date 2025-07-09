#!/bin/bash
# Simple drift detection wrapper for WGO

# Get provider from args or detect from last scan
PROVIDER=${1:-}
WGO_DIR="$HOME/.wgo"

# If no provider specified, try to detect from available last scans
if [ -z "$PROVIDER" ]; then
    LAST_SCANS=$(ls "$WGO_DIR"/last-scan-*.json 2>/dev/null | head -1)
    if [ -n "$LAST_SCANS" ]; then
        PROVIDER=$(basename "$LAST_SCANS" | sed 's/last-scan-//;s/.json//')
        echo "🔍 Auto-detected provider: $PROVIDER"
    else
        echo "❌ No previous scans found. Run 'wgo scan' first."
        exit 1
    fi
fi

# Check if we have a previous scan for this provider
LAST_SCAN="$WGO_DIR/last-scan-$PROVIDER.json"
if [ ! -f "$LAST_SCAN" ]; then
    echo "❌ No previous scan found for provider: $PROVIDER"
    echo "   Run: wgo scan --provider $PROVIDER"
    exit 1
fi

# Create temporary file for new scan
TEMP_SCAN=$(mktemp)
trap "rm -f $TEMP_SCAN" EXIT

echo "📊 Checking for drift in $PROVIDER infrastructure..."

# Run new scan and save to temp file
if ! ./wgo scan --provider "$PROVIDER" --output-file "$TEMP_SCAN" ${@:2}; then
    echo "❌ Scan failed"
    exit 1
fi

# Compare with last scan
echo ""
echo "🔍 Comparing with previous scan..."
./wgo diff --from "$LAST_SCAN" --to "$TEMP_SCAN"

# Update last scan if successful
cp "$TEMP_SCAN" "$LAST_SCAN"