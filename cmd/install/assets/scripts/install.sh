#!/usr/bin/env bash
# Tapio Universal Installation Script
# This script is embedded in the installer binary

set -euo pipefail

# Configuration
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"
TAPIO_VERSION="${TAPIO_VERSION:-latest}"
TAPIO_MIRROR="${TAPIO_MIRROR:-https://releases.tapio.io}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Helper functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

# Detect OS and architecture
detect_platform() {
    local os=$(uname -s | tr '[:upper:]' '[:lower:]')
    local arch=$(uname -m)

    case $arch in
        x86_64)
            arch="amd64"
            ;;
        aarch64|arm64)
            arch="arm64"
            ;;
        armv7l)
            arch="arm"
            ;;
    esac

    echo "${os}-${arch}"
}

# Check if running as root
check_root() {
    if [[ $EUID -eq 0 ]]; then
        return 0
    else
        return 1
    fi
}

# Check write permission to install directory
check_permissions() {
    local dir="$1"
    
    if [[ -w "$dir" ]]; then
        return 0
    else
        return 1
    fi
}

# Download file with progress
download_file() {
    local url="$1"
    local output="$2"
    
    if command -v curl &> /dev/null; then
        curl -L --progress-bar -o "$output" "$url"
    elif command -v wget &> /dev/null; then
        wget --show-progress -O "$output" "$url"
    else
        log_error "Neither curl nor wget found. Please install one of them."
        return 1
    fi
}

# Verify checksum
verify_checksum() {
    local file="$1"
    local expected_checksum="$2"
    
    if command -v sha256sum &> /dev/null; then
        local actual_checksum=$(sha256sum "$file" | awk '{print $1}')
    elif command -v shasum &> /dev/null; then
        local actual_checksum=$(shasum -a 256 "$file" | awk '{print $1}')
    else
        log_warning "No checksum utility found. Skipping verification."
        return 0
    fi
    
    if [[ "$actual_checksum" == "$expected_checksum" ]]; then
        return 0
    else
        log_error "Checksum mismatch!"
        log_error "Expected: $expected_checksum"
        log_error "Actual: $actual_checksum"
        return 1
    fi
}

# Main installation function
install_tapio() {
    log_info "Installing Tapio..."
    
    # Detect platform
    local platform=$(detect_platform)
    log_info "Detected platform: $platform"
    
    # Check permissions
    if ! check_permissions "$INSTALL_DIR"; then
        if check_root; then
            log_info "Running as root, proceeding with installation"
        else
            log_error "No write permission to $INSTALL_DIR"
            log_error "Please run with sudo or choose a different directory"
            return 1
        fi
    fi
    
    # Create temporary directory
    local temp_dir=$(mktemp -d -t tapio-install-XXXXXX)
    trap "rm -rf $temp_dir" EXIT
    
    # Construct download URL
    local binary_url="${TAPIO_MIRROR}/${TAPIO_VERSION}/tapio-${platform}"
    local checksum_url="${binary_url}.sha256"
    
    log_info "Downloading Tapio from $binary_url"
    
    # Download binary
    if ! download_file "$binary_url" "$temp_dir/tapio"; then
        log_error "Failed to download Tapio"
        return 1
    fi
    
    # Download and verify checksum
    log_info "Downloading checksum..."
    if download_file "$checksum_url" "$temp_dir/tapio.sha256" 2>/dev/null; then
        local expected_checksum=$(cat "$temp_dir/tapio.sha256" | awk '{print $1}')
        log_info "Verifying checksum..."
        if ! verify_checksum "$temp_dir/tapio" "$expected_checksum"; then
            log_error "Checksum verification failed"
            return 1
        fi
        log_success "Checksum verified"
    else
        log_warning "Checksum file not found, skipping verification"
    fi
    
    # Make binary executable
    chmod +x "$temp_dir/tapio"
    
    # Move to install directory
    log_info "Installing to $INSTALL_DIR/tapio"
    if [[ -f "$INSTALL_DIR/tapio" ]]; then
        log_info "Backing up existing installation..."
        mv "$INSTALL_DIR/tapio" "$INSTALL_DIR/tapio.backup"
    fi
    
    mv "$temp_dir/tapio" "$INSTALL_DIR/tapio"
    
    # Verify installation
    if [[ -x "$INSTALL_DIR/tapio" ]]; then
        log_success "Tapio installed successfully!"
        
        # Check if in PATH
        if command -v tapio &> /dev/null; then
            log_success "Tapio is available in your PATH"
            tapio --version
        else
            log_warning "Tapio is not in your PATH"
            log_info "Add $INSTALL_DIR to your PATH to use 'tapio' command"
            log_info "You can run it directly: $INSTALL_DIR/tapio"
        fi
    else
        log_error "Installation failed"
        return 1
    fi
}

# Run installation
install_tapio