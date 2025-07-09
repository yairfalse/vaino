#!/bin/bash
# WGO Universal Installation Script
# Supports: Linux (x64, arm64), macOS (x64, arm64), Windows (via WSL)
# Auto-detects package managers and falls back to direct binary download

set -e

# Configuration
REPO_OWNER="yairfalse"
REPO_NAME="wgo"
INSTALL_DIR="/usr/local/bin"
TEMP_DIR=$(mktemp -d)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
    echo -e "${RED}[ERROR]${NC} $1"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Detect OS and Architecture
detect_os() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    case "$OS" in
        linux*)  OS="linux" ;;
        darwin*) OS="darwin" ;;
        msys*|mingw*|cygwin*) OS="windows" ;;
        *)       log_error "Unsupported OS: $OS"; exit 1 ;;
    esac
    echo "$OS"
}

detect_arch() {
    ARCH=$(uname -m)
    case "$ARCH" in
        x86_64|amd64) ARCH="x86_64" ;;
        aarch64|arm64) ARCH="arm64" ;;
        armv7l|armv7) ARCH="armv7" ;;
        i386|i686) ARCH="i386" ;;
        *) log_error "Unsupported architecture: $ARCH"; exit 1 ;;
    esac
    echo "$ARCH"
}

# Check if command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Get latest release version from GitHub
get_latest_version() {
    log_info "Fetching latest version..."
    VERSION=$(curl -s "https://api.github.com/repos/$REPO_OWNER/$REPO_NAME/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')
    if [[ -z "$VERSION" ]]; then
        log_error "Failed to fetch latest version"
        exit 1
    fi
    echo "$VERSION"
}

# Try package manager installation first
try_package_manager_install() {
    local os=$1
    
    # macOS - Homebrew
    if [[ "$os" == "darwin" ]] && command_exists brew; then
        log_info "Installing via Homebrew..."
        brew tap "$REPO_OWNER/wgo" 2>/dev/null || true
        if brew install wgo; then
            log_success "Installed via Homebrew"
            return 0
        fi
    fi
    
    # Linux - Various package managers
    if [[ "$os" == "linux" ]]; then
        # Try snap first (works on many distros)
        if command_exists snap; then
            log_info "Installing via Snap..."
            if sudo snap install wgo; then
                log_success "Installed via Snap"
                return 0
            fi
        fi
        
        # Debian/Ubuntu - APT
        if command_exists apt-get; then
            log_info "Detected Debian/Ubuntu system"
            # Check if our APT repo is available
            if curl -s https://apt.wgo.sh/ubuntu/dists/stable/Release >/dev/null 2>&1; then
                log_info "Adding WGO APT repository..."
                curl -fsSL https://apt.wgo.sh/ubuntu/wgo.gpg | sudo apt-key add -
                echo "deb https://apt.wgo.sh/ubuntu stable main" | sudo tee /etc/apt/sources.list.d/wgo.list
                sudo apt-get update
                if sudo apt-get install -y wgo; then
                    log_success "Installed via APT"
                    return 0
                fi
            fi
        fi
        
        # RHEL/CentOS/Fedora - YUM/DNF
        if command_exists yum || command_exists dnf; then
            log_info "Detected RHEL/CentOS/Fedora system"
            # Check if our YUM repo is available
            if curl -s https://yum.wgo.sh/rhel/wgo.repo >/dev/null 2>&1; then
                log_info "Adding WGO YUM repository..."
                sudo curl -fsSL https://yum.wgo.sh/rhel/wgo.repo -o /etc/yum.repos.d/wgo.repo
                if command_exists dnf; then
                    if sudo dnf install -y wgo; then
                        log_success "Installed via DNF"
                        return 0
                    fi
                else
                    if sudo yum install -y wgo; then
                        log_success "Installed via YUM"
                        return 0
                    fi
                fi
            fi
        fi
        
        # Arch Linux - AUR
        if command_exists pacman && command_exists yay; then
            log_info "Installing via AUR..."
            if yay -S wgo-bin --noconfirm; then
                log_success "Installed via AUR"
                return 0
            fi
        fi
    fi
    
    # Windows - Chocolatey/Scoop
    if [[ "$os" == "windows" ]]; then
        if command_exists choco; then
            log_info "Installing via Chocolatey..."
            if choco install wgo -y; then
                log_success "Installed via Chocolatey"
                return 0
            fi
        fi
        
        if command_exists scoop; then
            log_info "Installing via Scoop..."
            scoop bucket add wgo https://github.com/$REPO_OWNER/scoop-wgo
            if scoop install wgo; then
                log_success "Installed via Scoop"
                return 0
            fi
        fi
    fi
    
    return 1
}

# Verify file checksum
verify_checksum() {
    local file=$1
    local checksum_file=$2
    
    log_info "Verifying file integrity..."
    
    # Check if sha256sum is available
    if command_exists sha256sum; then
        local file_hash=$(sha256sum "$file" | cut -d' ' -f1)
        local expected_hash=$(grep "$(basename "$file")" "$checksum_file" | cut -d' ' -f1)
        
        if [[ -z "$expected_hash" ]]; then
            log_warn "Checksum not found for $(basename "$file"), skipping verification"
            return 0
        fi
        
        if [[ "$file_hash" == "$expected_hash" ]]; then
            log_success "File integrity verified ✓"
        else
            log_error "Checksum verification failed!"
            log_error "Expected: $expected_hash"
            log_error "Got:      $file_hash"
            exit 1
        fi
    elif command_exists shasum; then
        # macOS alternative
        local file_hash=$(shasum -a 256 "$file" | cut -d' ' -f1)
        local expected_hash=$(grep "$(basename "$file")" "$checksum_file" | cut -d' ' -f1)
        
        if [[ -z "$expected_hash" ]]; then
            log_warn "Checksum not found for $(basename "$file"), skipping verification"
            return 0
        fi
        
        if [[ "$file_hash" == "$expected_hash" ]]; then
            log_success "File integrity verified ✓"
        else
            log_error "Checksum verification failed!"
            log_error "Expected: $expected_hash"
            log_error "Got:      $file_hash"
            exit 1
        fi
    else
        log_warn "sha256sum/shasum not available, skipping checksum verification"
    fi
}

# Download and install binary directly
install_binary() {
    local os=$1
    local arch=$2
    local version=$3
    
    log_info "Installing WGO $version via direct download..."
    
    # Construct download URL
    local ext=""
    [[ "$os" == "windows" ]] && ext=".zip" || ext=".tar.gz"
    
    # Map OS names to match GoReleaser format (capitalized)
    local download_os=""
    case "$os" in
        linux)   download_os="Linux" ;;
        darwin)  download_os="Darwin" ;;
        windows) download_os="Windows" ;;
    esac
    
    # Map architecture names for download
    local download_arch="$arch"
    
    local url="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$version/${REPO_NAME}_${download_os}_${download_arch}${ext}"
    local download_file="$TEMP_DIR/wgo${ext}"
    local checksum_url="https://github.com/$REPO_OWNER/$REPO_NAME/releases/download/$version/checksums.txt"
    local checksum_file="$TEMP_DIR/checksums.txt"
    
    log_info "Downloading from: $url"
    
    # Download the file
    if ! curl -fsSL "$url" -o "$download_file"; then
        log_error "Failed to download WGO"
        exit 1
    fi
    
    # Download checksums for verification
    log_info "Downloading checksums for verification..."
    if ! curl -fsSL "$checksum_url" -o "$checksum_file"; then
        log_warn "Failed to download checksums, skipping verification"
    else
        verify_checksum "$download_file" "$checksum_file"
    fi
    
    # Extract based on file type
    cd "$TEMP_DIR"
    if [[ "$ext" == ".zip" ]]; then
        unzip -q "$download_file"
    else
        tar -xzf "$download_file"
    fi
    
    # Find the binary
    local binary_name="wgo"
    [[ "$os" == "windows" ]] && binary_name="wgo.exe"
    
    if [[ ! -f "$binary_name" ]]; then
        log_error "Binary not found in archive"
        exit 1
    fi
    
    # Install the binary
    chmod +x "$binary_name"
    
    # Check if we need sudo for installation
    if [[ -w "$INSTALL_DIR" ]]; then
        mv "$binary_name" "$INSTALL_DIR/"
    else
        log_info "Installing to $INSTALL_DIR (requires sudo)..."
        sudo mv "$binary_name" "$INSTALL_DIR/"
    fi
    
    log_success "WGO installed successfully to $INSTALL_DIR/wgo"
}

# Install shell completions
install_completions() {
    log_info "Installing shell completions..."
    
    # Bash completion
    if [[ -d /etc/bash_completion.d ]] || [[ -d /usr/local/etc/bash_completion.d ]]; then
        wgo completion bash > "$TEMP_DIR/wgo.bash" 2>/dev/null || true
        if [[ -f "$TEMP_DIR/wgo.bash" ]]; then
            if [[ -d /usr/local/etc/bash_completion.d ]]; then
                sudo cp "$TEMP_DIR/wgo.bash" /usr/local/etc/bash_completion.d/wgo
            else
                sudo cp "$TEMP_DIR/wgo.bash" /etc/bash_completion.d/wgo
            fi
            log_success "Bash completions installed"
        fi
    fi
    
    # Zsh completion
    if [[ -d /usr/local/share/zsh/site-functions ]] || [[ -d /usr/share/zsh/site-functions ]]; then
        wgo completion zsh > "$TEMP_DIR/_wgo" 2>/dev/null || true
        if [[ -f "$TEMP_DIR/_wgo" ]]; then
            if [[ -d /usr/local/share/zsh/site-functions ]]; then
                sudo cp "$TEMP_DIR/_wgo" /usr/local/share/zsh/site-functions/_wgo
            else
                sudo cp "$TEMP_DIR/_wgo" /usr/share/zsh/site-functions/_wgo
            fi
            log_success "Zsh completions installed"
        fi
    fi
    
    # Fish completion
    if command_exists fish && [[ -d ~/.config/fish/completions ]]; then
        wgo completion fish > ~/.config/fish/completions/wgo.fish 2>/dev/null || true
        log_success "Fish completions installed"
    fi
}

# Verify installation
verify_installation() {
    if command_exists wgo; then
        log_success "WGO installation verified!"
        wgo version
        echo ""
        log_info "Quick start:"
        echo "  wgo scan                  # Auto-discover and scan infrastructure"
        echo "  wgo diff                  # Compare infrastructure states"
        echo "  wgo scan --provider aws   # Scan AWS resources"
        echo "  wgo --help               # Show all commands"
        echo ""
        log_info "Documentation: https://github.com/$REPO_OWNER/$REPO_NAME"
    else
        log_error "WGO installation failed - 'wgo' command not found"
        log_info "You may need to add $INSTALL_DIR to your PATH"
        exit 1
    fi
}

# Cleanup
cleanup() {
    rm -rf "$TEMP_DIR"
}

# Main installation flow
main() {
    echo "🚀 WGO Installer"
    echo "================"
    echo ""
    
    # Detect system
    OS=$(detect_os)
    ARCH=$(detect_arch)
    log_info "Detected: $OS/$ARCH"
    
    # Try package manager first
    if try_package_manager_install "$OS"; then
        install_completions
        verify_installation
        cleanup
        return 0
    fi
    
    # Fall back to binary installation
    log_warn "Package manager installation not available, falling back to direct download"
    VERSION=$(get_latest_version)
    log_info "Latest version: $VERSION"
    
    install_binary "$OS" "$ARCH" "$VERSION"
    install_completions
    verify_installation
    cleanup
}

# Run main function
trap cleanup EXIT
main "$@"