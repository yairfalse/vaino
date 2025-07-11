#!/bin/bash
# YUM/RPM Repository Setup Script for VAINO
# This script sets up a complete YUM repository for distributing VAINO packages

set -e

# Configuration
REPO_NAME="vaino"
REPO_OWNER="yairfalse"
REPO_ROOT="/tmp/yum-repo"
DISTRIBUTIONS=("el8" "el9" "fedora37" "fedora38" "fedora39")
ARCHITECTURES=("x86_64" "aarch64" "armv7hl")
GPG_KEY_ID="VAINO Package Signing Key"
GPG_KEY_EMAIL="packages@vaino.sh"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

# Check dependencies
check_dependencies() {
    log_info "Checking dependencies..."
    
    local missing_deps=()
    
    for dep in createrepo rpm-sign gpg; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Install them with:"
        echo "  RHEL/CentOS: sudo dnf install createrepo rpm-sign gnupg"
        echo "  Fedora: sudo dnf install createrepo_c rpm-sign gnupg"
        echo "  Ubuntu/Debian: sudo apt-get install createrepo-c rpm gnupg"
        exit 1
    fi
    
    log_success "All dependencies available"
}

# Create GPG key for package signing
create_gpg_key() {
    log_info "Setting up GPG key for package signing..."
    
    # Check if key already exists
    if gpg --list-secret-keys | grep -q "$GPG_KEY_EMAIL"; then
        log_info "GPG key already exists"
        return 0
    fi
    
    # Generate key configuration
    cat > /tmp/gpg-key-config << EOF
%echo Generating GPG key for VAINO package signing
Key-Type: RSA
Key-Length: 4096
Subkey-Type: RSA
Subkey-Length: 4096
Name-Real: ${GPG_KEY_ID}
Name-Email: ${GPG_KEY_EMAIL}
Expire-Date: 2y
%no-protection
%commit
%echo done
EOF
    
    # Generate the key
    gpg --batch --generate-key /tmp/gpg-key-config
    rm /tmp/gpg-key-config
    
    # Export public key
    gpg --armor --export "$GPG_KEY_EMAIL" > "${REPO_ROOT}/vaino.gpg"
    
    # Configure RPM signing
    cat > ~/.rpmmacros << EOF
%_gpg_name ${GPG_KEY_EMAIL}
%_signature gpg
%_gpg_path ~/.gnupg
%_gpgbin /usr/bin/gpg
%__gpg_sign_cmd %{__gpg} gpg --force-v3-sigs --batch --verbose --no-armor --passphrase-fd 3 --no-secmem-warning -u "%{_gpg_name}" -sbo %{__signature_filename} --digest-algo sha256 %{__plaintext_filename}
EOF
    
    log_success "GPG key created and configured for RPM signing"
}

# Create repository structure
create_repo_structure() {
    log_info "Creating repository structure..."
    
    # Clean and recreate repository root
    rm -rf "$REPO_ROOT"
    mkdir -p "$REPO_ROOT"
    
    # Create directory structure for different distributions
    for dist in "${DISTRIBUTIONS[@]}"; do
        for arch in "${ARCHITECTURES[@]}"; do
            mkdir -p "${REPO_ROOT}/rhel/${dist}/${arch}"
            mkdir -p "${REPO_ROOT}/rhel/${dist}/${arch}/repodata"
        done
    done
    
    # Create source RPM directory
    mkdir -p "${REPO_ROOT}/rhel/SRPMS"
    
    log_success "Repository structure created"
}

# Build RPM from spec file
build_rpm() {
    local version=$1
    local arch=$2
    
    log_info "Building RPM for version $version, architecture $arch"
    
    # Create build environment
    local build_dir="/tmp/rpmbuild"
    mkdir -p "$build_dir"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
    
    # Copy spec file
    cp packaging/yum/vaino.spec "$build_dir/SPECS/"
    
    # Build the RPM
    rpmbuild --define="_topdir $build_dir" \
             --define="version $version" \
             --target="$arch" \
             -ba "$build_dir/SPECS/vaino.spec"
    
    # Sign the RPM
    rpm-sign --addsign "$build_dir/RPMS/$arch/vaino-$version-1.*.rpm"
    
    log_success "RPM built successfully"
    echo "$build_dir/RPMS/$arch/vaino-$version-1.*.rpm"
}

# Add package to repository
add_package() {
    local package_file=$1
    local distribution=$2
    local architecture=$3
    
    if [[ ! -f "$package_file" ]]; then
        log_error "Package file not found: $package_file"
        return 1
    fi
    
    log_info "Adding package to repository: $(basename "$package_file")"
    
    # Copy package to repository
    cp "$package_file" "${REPO_ROOT}/rhel/${distribution}/${architecture}/"
    
    log_success "Package added to repository"
}

# Generate repository metadata
generate_metadata() {
    log_info "Generating repository metadata..."
    
    for dist in "${DISTRIBUTIONS[@]}"; do
        for arch in "${ARCHITECTURES[@]}"; do
            repo_path="${REPO_ROOT}/rhel/${dist}/${arch}"
            
            if ls "$repo_path"/*.rpm >/dev/null 2>&1; then
                log_info "Creating repository metadata for $dist/$arch"
                
                # Create repository metadata
                createrepo --database "$repo_path"
                
                # Sign repository metadata
                gpg --detach-sign --armor "$repo_path/repodata/repomd.xml"
            else
                log_warn "No RPM packages found for $dist/$arch"
            fi
        done
    done
    
    log_success "Repository metadata generated"
}

# Create repository configuration files
create_repo_config() {
    log_info "Creating repository configuration files..."
    
    # Create main repository configuration
    cat > "${REPO_ROOT}/vaino.repo" << EOF
[vaino]
name=VAINO Repository
baseurl=https://yum.vaino.sh/rhel/\$releasever/\$basearch/
enabled=1
gpgcheck=1
gpgkey=https://yum.vaino.sh/rhel/vaino.gpg
EOF
    
    # Create distribution-specific configurations
    for dist in "${DISTRIBUTIONS[@]}"; do
        cat > "${REPO_ROOT}/vaino-${dist}.repo" << EOF
[vaino-${dist}]
name=VAINO Repository for ${dist}
baseurl=https://yum.vaino.sh/rhel/${dist}/\$basearch/
enabled=1
gpgcheck=1
gpgkey=https://yum.vaino.sh/rhel/vaino.gpg
EOF
    done
    
    # Create installation script
    cat > "${REPO_ROOT}/install-repo.sh" << 'EOF'
#!/bin/bash
# VAINO YUM Repository Installation Script

set -e

# Detect distribution
if [[ -f /etc/redhat-release ]]; then
    if grep -q "CentOS\|Red Hat" /etc/redhat-release; then
        if grep -q "release 8" /etc/redhat-release; then
            DIST="el8"
        elif grep -q "release 9" /etc/redhat-release; then
            DIST="el9"
        else
            DIST="el9"  # Default to latest
        fi
    elif grep -q "Fedora" /etc/redhat-release; then
        FEDORA_VERSION=$(grep -o "release [0-9]*" /etc/redhat-release | awk '{print $2}')
        DIST="fedora${FEDORA_VERSION}"
    else
        DIST="el9"  # Default fallback
    fi
else
    echo "This script is designed for RHEL/CentOS/Fedora systems"
    exit 1
fi

# Add GPG key
echo "Adding VAINO GPG key..."
rpm --import https://yum.vaino.sh/rhel/vaino.gpg

# Add repository
echo "Adding VAINO repository..."
curl -fsSL https://yum.vaino.sh/rhel/vaino-${DIST}.repo -o /etc/yum.repos.d/vaino.repo

# Update metadata
if command -v dnf >/dev/null 2>&1; then
    dnf makecache
else
    yum makecache
fi

echo "VAINO YUM repository added successfully!"
echo "You can now install VAINO with:"
if command -v dnf >/dev/null 2>&1; then
    echo "  sudo dnf install vaino"
else
    echo "  sudo yum install vaino"
fi
EOF
    chmod +x "${REPO_ROOT}/install-repo.sh"
    
    log_success "Repository configuration files created"
}

# Test repository
test_repository() {
    log_info "Testing repository setup..."
    
    # Create temporary repository configuration
    local temp_repo="/tmp/test-vaino.repo"
    cat > "$temp_repo" << EOF
[vaino-test]
name=VAINO Test Repository
baseurl=file://${REPO_ROOT}/rhel/el9/x86_64/
enabled=1
gpgcheck=0
EOF
    
    # Test repository access
    if command -v dnf >/dev/null 2>&1; then
        if dnf --repo=vaino-test --disablerepo=* list available 2>/dev/null; then
            log_success "Repository structure is valid"
        else
            log_error "Repository test failed"
            return 1
        fi
    elif command -v yum >/dev/null 2>&1; then
        if yum --disablerepo=* --enablerepo=vaino-test list available 2>/dev/null; then
            log_success "Repository structure is valid"
        else
            log_error "Repository test failed"
            return 1
        fi
    else
        log_warn "Neither dnf nor yum available for testing"
    fi
    
    # Clean up
    rm -f "$temp_repo"
}

# Generate GitHub Pages compatible structure
generate_github_pages() {
    log_info "Generating GitHub Pages compatible structure..."
    
    # Create index.html
    cat > "${REPO_ROOT}/index.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>VAINO YUM Repository</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .code { background: #f4f4f4; padding: 10px; border-radius: 5px; font-family: monospace; }
        .header { text-align: center; margin-bottom: 40px; }
        .distro { margin: 20px 0; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>VAINO YUM Repository</h1>
            <p>Official YUM repository for VAINO (What's Going On)</p>
        </div>
        
        <h2>Quick Installation</h2>
        <div class="code">
            curl -fsSL https://yum.vaino.sh/rhel/install-repo.sh | sudo bash<br>
            sudo dnf install vaino
        </div>
        
        <h2>Manual Installation</h2>
        <ol>
            <li>Add the GPG key:
                <div class="code">sudo rpm --import https://yum.vaino.sh/rhel/vaino.gpg</div>
            </li>
            <li>Add the repository:
                <div class="code">sudo curl -fsSL https://yum.vaino.sh/rhel/vaino.repo -o /etc/yum.repos.d/vaino.repo</div>
            </li>
            <li>Install VAINO:
                <div class="code">sudo dnf install vaino</div>
            </li>
        </ol>
        
        <h2>Supported Distributions</h2>
EOF
    
    for dist in "${DISTRIBUTIONS[@]}"; do
        cat >> "${REPO_ROOT}/index.html" << EOF
        <div class="distro">
            <h3>${dist}</h3>
            <div class="code">
                sudo curl -fsSL https://yum.vaino.sh/rhel/vaino-${dist}.repo -o /etc/yum.repos.d/vaino.repo
            </div>
        </div>
EOF
    done
    
    cat >> "${REPO_ROOT}/index.html" << EOF
        
        <h2>Supported Architectures</h2>
        <ul>
            <li>x86_64 (AMD64)</li>
            <li>aarch64 (ARM64)</li>
            <li>armv7hl (ARM 32-bit)</li>
        </ul>
        
        <p><a href="https://github.com/${REPO_OWNER}/${REPO_NAME}">View on GitHub</a></p>
    </div>
</body>
</html>
EOF
    
    log_success "GitHub Pages structure generated"
}

# Main execution
main() {
    echo "🚀 VAINO YUM Repository Setup"
    echo "=========================="
    echo ""
    
    check_dependencies
    create_gpg_key
    create_repo_structure
    
    # Look for existing packages to add
    if ls *.rpm >/dev/null 2>&1; then
        log_info "Found existing .rpm packages"
        for pkg in *.rpm; do
            # Extract distribution and architecture from filename
            # This is a simplified approach - in practice you'd parse the RPM headers
            add_package "$pkg" "el9" "x86_64"
        done
    else
        log_warn "No .rpm packages found in current directory"
        log_info "Build packages first or use the build_rpm function"
    fi
    
    generate_metadata
    create_repo_config
    generate_github_pages
    test_repository
    
    echo ""
    log_success "YUM repository setup completed!"
    echo ""
    echo "Repository location: $REPO_ROOT"
    echo "GPG public key: ${REPO_ROOT}/vaino.gpg"
    echo "Installation script: ${REPO_ROOT}/install-repo.sh"
    echo ""
    echo "Next steps:"
    echo "1. Upload the contents of ${REPO_ROOT}/rhel to your web server"
    echo "2. Configure your web server to serve the repository"
    echo "3. Set up HTTPS with proper certificates"
    echo "4. Test the repository with: curl -fsSL https://yum.vaino.sh/rhel/install-repo.sh | sudo bash"
}

# Run main function
main "$@"