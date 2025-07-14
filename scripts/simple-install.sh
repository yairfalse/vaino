#!/bin/bash
# Simple installer for VAINO - Just Works™

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

echo -e "${GREEN}Installing VAINO...${NC}"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64) ARCH="amd64" ;;
    aarch64|arm64) ARCH="arm64" ;;
    *) echo -e "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
esac

# Try to download from GitHub releases first
REPO="yairfalse/vaino"
LATEST_URL="https://api.github.com/repos/$REPO/releases/latest"

echo "Checking for latest release..."
if command -v curl >/dev/null 2>&1 && curl -s "$LATEST_URL" | grep -q "browser_download_url"; then
    VERSION=$(curl -s "$LATEST_URL" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    BINARY_URL="https://github.com/$REPO/releases/download/$VERSION/vaino-$OS-$ARCH"
    
    echo -e "${GREEN}Downloading VAINO $VERSION...${NC}"
    curl -L -o /tmp/vaino "$BINARY_URL"
    chmod +x /tmp/vaino
    
    # Install binary
    if [ -w /usr/local/bin ]; then
        mv /tmp/vaino /usr/local/bin/vaino
    else
        echo -e "${YELLOW}Need sudo to install to /usr/local/bin${NC}"
        sudo mv /tmp/vaino /usr/local/bin/vaino
    fi
    
    echo -e "${GREEN}✓ VAINO installed successfully!${NC}"
else
    # Fallback: Build from source
    echo -e "${YELLOW}No releases found. Building from source...${NC}"
    
    # Check for Go
    if ! command -v go >/dev/null 2>&1; then
        echo -e "${RED}Go is required but not installed.${NC}"
        echo "Install Go from https://golang.org/dl/"
        exit 1
    fi
    
    # Clone and build
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"
    
    echo "Cloning repository..."
    git clone --depth 1 https://github.com/$REPO.git
    cd vaino
    
    echo "Building VAINO..."
    go build -o vaino ./cmd/vaino
    
    # Install binary
    if [ -w /usr/local/bin ]; then
        mv vaino /usr/local/bin/vaino
    else
        echo -e "${YELLOW}Need sudo to install to /usr/local/bin${NC}"
        sudo mv vaino /usr/local/bin/vaino
    fi
    
    # Cleanup
    cd /
    rm -rf "$TMP_DIR"
    
    echo -e "${GREEN}✓ VAINO built and installed successfully!${NC}"
fi

# Verify installation
if command -v vaino >/dev/null 2>&1; then
    echo -e "${GREEN}VAINO is ready to use!${NC}"
    echo "Run 'vaino --help' to get started"
else
    echo -e "${RED}Installation failed${NC}"
    exit 1
fi