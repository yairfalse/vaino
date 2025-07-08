#!/bin/bash

# Script to set up GitHub Pages branch for benchmark tracking
# Run this script once to enable benchmark history tracking

set -e

echo "🔧 Setting up GitHub Pages branch for benchmark tracking..."

# Check if we're in a git repository
if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "❌ Error: Not in a git repository"
    exit 1
fi

# Check if gh-pages branch already exists
if git show-ref --verify --quiet refs/heads/gh-pages; then
    echo "✅ gh-pages branch already exists locally"
elif git show-ref --verify --quiet refs/remotes/origin/gh-pages; then
    echo "✅ gh-pages branch exists remotely, checking out..."
    git checkout -b gh-pages origin/gh-pages
else
    echo "📄 Creating new gh-pages branch..."
    
    # Save current branch
    CURRENT_BRANCH=$(git branch --show-current)
    
    # Create orphan gh-pages branch
    git checkout --orphan gh-pages
    
    # Remove all files
    git rm -rf . 2>/dev/null || true
    
    # Create initial page
    cat > index.html << 'EOF'
<!DOCTYPE html>
<html>
<head>
    <meta charset="utf-8">
    <title>WGO Performance Benchmarks</title>
    <style>
        body { font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif; margin: 40px; }
        .header { text-align: center; margin-bottom: 40px; }
        .benchmark-container { max-width: 1200px; margin: 0 auto; }
    </style>
</head>
<body>
    <div class="header">
        <h1>🚀 WGO Performance Benchmarks</h1>
        <p>Infrastructure drift detection tool performance tracking</p>
    </div>
    <div class="benchmark-container">
        <div id="benchmark-chart"></div>
    </div>
    
    <script>
        // Benchmark data will be inserted here by GitHub Actions
        console.log('WGO Benchmark tracking initialized');
    </script>
</body>
</html>
EOF
    
    cat > README.md << 'EOF'
# WGO Benchmark Results

This branch contains performance benchmark results for the WGO infrastructure drift detection tool.

## Latest Performance Metrics

- **Terraform State Parsing**: Sub-millisecond performance for typical workloads
- **Parallel Processing**: Handles multiple state files concurrently  
- **Streaming Parser**: Efficient processing of large state files (100MB+)
- **Resource Normalization**: Fast conversion across multiple cloud providers

## Benchmark History

Benchmark results are automatically updated on every commit to the main branch.
View the interactive charts at: https://yourusername.github.io/wgo/

## Key Performance Targets

- Small state files (10 resources): < 100µs
- Medium state files (100 resources): < 500µs  
- Large state files (500+ resources): < 5ms
- Parallel processing: Linear scaling with worker count
- Memory usage: Constant for streaming, minimal for standard parsing
EOF
    
    # Add and commit
    git add .
    git commit -m "Initialize gh-pages branch for benchmark tracking"
    
    # Push to remote
    echo "📤 Pushing gh-pages branch to remote..."
    git push -u origin gh-pages
    
    # Return to original branch
    git checkout "$CURRENT_BRANCH"
    
    echo "✅ GitHub Pages branch created successfully!"
fi

echo ""
echo "🎯 Next steps:"
echo "1. Enable GitHub Pages in your repository settings"
echo "2. Set source to 'gh-pages' branch"  
echo "3. Uncomment the benchmark-action step in .github/workflows/benchmark.yml"
echo "4. Your benchmarks will be available at: https://yourusername.github.io/wgo/"
echo ""
echo "📊 Benchmark tracking is now ready!"