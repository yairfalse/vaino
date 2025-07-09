#!/bin/bash
# Quick drift detection - compares current state with last saved scan

echo "Add this to your .bashrc or .zshrc:"
echo ""
echo "# WGO drift detection alias"
echo "alias wgo-drift='wgo scan --provider kubernetes && wgo diff --from ~/.wgo/last-scan-kubernetes.json --to ~/.wgo/last-scan-kubernetes.json'"
echo ""
echo "# Or as a function for any provider:"
echo 'wdrift() {'
echo '    local provider=${1:-kubernetes}'
echo '    shift'
echo '    local last_scan="$HOME/.wgo/last-scan-$provider.json"'
echo '    if [ ! -f "$last_scan" ]; then'
echo '        echo "No previous scan for $provider. Running initial scan..."'
echo '        wgo scan --provider "$provider" "$@"'
echo '    else'
echo '        # Save current as temp'
echo '        local temp_scan=$(mktemp)'
echo '        echo "🔍 Scanning $provider for changes..."'
echo '        if wgo scan --provider "$provider" "$@" --output-file "$temp_scan" > /dev/null 2>&1; then'
echo '            wgo diff --from "$last_scan" --to "$temp_scan"'
echo '            # Update last scan'
echo '            mv "$temp_scan" "$last_scan"'
echo '        else'
echo '            echo "Scan failed"'
echo '            rm -f "$temp_scan"'
echo '        fi'
echo '    fi'
echo '}'