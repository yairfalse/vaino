#!/bin/bash
set -e

# Install Task runner if not already installed
if command -v task >/dev/null 2>&1; then
    echo "✅ Task is already installed: $(task --version)"
    exit 0
fi

echo "📦 Installing Task runner..."

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    armv7*) ARCH="arm" ;;
    *) echo "❌ Unsupported architecture: $ARCH"; exit 1 ;;
esac

case $OS in
    linux|darwin) ;;
    *) echo "❌ Unsupported OS: $OS"; exit 1 ;;
esac

# Get the latest version
LATEST_VERSION=$(curl -s https://api.github.com/repos/go-task/task/releases/latest | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "❌ Failed to get latest Task version"
    exit 1
fi

# Download and install
DOWNLOAD_URL="https://github.com/go-task/task/releases/download/${LATEST_VERSION}/task_${OS}_${ARCH}.tar.gz"
TEMP_DIR=$(mktemp -d)
TEMP_FILE="$TEMP_DIR/task.tar.gz"

echo "📥 Downloading Task ${LATEST_VERSION} for ${OS}/${ARCH}..."
curl -sL "$DOWNLOAD_URL" -o "$TEMP_FILE"

echo "📁 Extracting..."
tar -xzf "$TEMP_FILE" -C "$TEMP_DIR"

echo "🔧 Installing to /usr/local/bin..."
if [ -w "/usr/local/bin" ]; then
    mv "$TEMP_DIR/task" /usr/local/bin/
else
    sudo mv "$TEMP_DIR/task" /usr/local/bin/
fi

# Cleanup
rm -rf "$TEMP_DIR"

echo "✅ Task installed successfully!"
echo "🚀 Run 'task --version' to verify installation"
echo "📖 Run 'task' to see available tasks for this project"