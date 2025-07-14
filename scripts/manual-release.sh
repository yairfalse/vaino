#!/bin/bash
# Manual release builder for VAINO - No CI needed!

set -e

VERSION=${1:-"0.1.0"}
echo "Building VAINO v$VERSION..."

# Create dist directory
rm -rf dist
mkdir -p dist

# Build for each platform
echo "Building binaries..."

# Linux AMD64
echo "  → linux/amd64"
GOOS=linux GOARCH=amd64 go build -o dist/vaino-linux-amd64 ./cmd/vaino

# Linux ARM64
echo "  → linux/arm64"
GOOS=linux GOARCH=arm64 go build -o dist/vaino-linux-arm64 ./cmd/vaino

# macOS AMD64
echo "  → darwin/amd64"
GOOS=darwin GOARCH=amd64 go build -o dist/vaino-darwin-amd64 ./cmd/vaino

# macOS ARM64 (M1/M2)
echo "  → darwin/arm64"
GOOS=darwin GOARCH=arm64 go build -o dist/vaino-darwin-arm64 ./cmd/vaino

# Windows AMD64
echo "  → windows/amd64"
GOOS=windows GOARCH=amd64 go build -o dist/vaino-windows-amd64.exe ./cmd/vaino

# Create checksums
echo "Creating checksums..."
cd dist
shasum -a 256 * > checksums.txt
cd ..

echo ""
echo "✅ Release artifacts created in dist/"
echo ""
echo "To publish:"
echo "1. Create a new release on GitHub: https://github.com/yairfalse/vaino/releases/new"
echo "2. Tag version: v$VERSION"
echo "3. Upload all files from dist/"
echo "4. Publish release"
echo ""
echo "Then users can install with:"
echo "curl -sSL https://raw.githubusercontent.com/yairfalse/vaino/main/scripts/simple-install.sh | bash"