#\!/bin/bash

echo "=== Checking Architectural Level Violations ==="
echo

# Define architectural levels
declare -A levels=(
    ["cmd"]=1
    ["internal/watchers"]=2
    ["internal/workers"]=2  
    ["internal/scanner"]=2
    ["internal/collectors"]=3
    ["internal/storage"]=3
    ["internal/analysis"]=3
    ["internal/errors"]=4
    ["internal/utils"]=4
    ["pkg/types"]=5
)

# Check for violations
echo "Level 5 (pkg/types) importing from internal:"
grep -r "import.*internal/" pkg/types/ --include="*.go" 2>/dev/null | head -5

echo
echo "Level 4 (errors/utils) importing from higher levels:"
grep -r "import.*internal/\(collectors\|storage\|analysis\|watchers\|workers\|scanner\)" internal/errors/ internal/utils/ --include="*.go" 2>/dev/null | head -5

echo
echo "Level 3 (collectors/storage/analysis) importing from higher levels:"
grep -r "import.*internal/\(watchers\|workers\|scanner\)" internal/collectors/ internal/storage/ internal/analysis/ --include="*.go" 2>/dev/null | head -5

echo
echo "Any internal importing from cmd:"
grep -r "import.*cmd/" internal/ pkg/ --include="*.go" 2>/dev/null | head -5

echo
echo "=== Import Summary ==="
echo "Total imports from internal to cmd: $(grep -r "import.*cmd/" internal/ pkg/ --include="*.go" 2>/dev/null | wc -l)"
echo "Total imports from pkg to internal: $(grep -r "import.*internal/" pkg/ --include="*.go" 2>/dev/null | wc -l)"
