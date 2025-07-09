#!/bin/bash
# APT Repository Setup Script for WGO
# This script sets up a complete APT repository for distributing WGO packages

set -e

# Configuration
REPO_NAME="wgo"
REPO_OWNER="yairfalse"
REPO_ROOT="/tmp/apt-repo"
DISTRIBUTIONS=("stable" "testing")
COMPONENTS=("main")
ARCHITECTURES=("amd64" "arm64" "armhf")
GPG_KEY_ID="WGO Package Signing Key"
GPG_KEY_EMAIL="packages@wgo.sh"

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
    
    for dep in dpkg-scanpackages dpkg-sig apt-ftparchive gpg; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Install them with: sudo apt-get install dpkg-dev dpkg-sig apt-utils gnupg"
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
%echo Generating GPG key for WGO package signing
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
    gpg --armor --export "$GPG_KEY_EMAIL" > "${REPO_ROOT}/wgo.gpg"
    
    log_success "GPG key created and exported"
}

# Create repository structure
create_repo_structure() {
    log_info "Creating repository structure..."
    
    # Clean and recreate repository root
    rm -rf "$REPO_ROOT"
    mkdir -p "$REPO_ROOT"
    
    # Create directory structure
    for dist in "${DISTRIBUTIONS[@]}"; do
        for comp in "${COMPONENTS[@]}"; do
            for arch in "${ARCHITECTURES[@]}"; do
                mkdir -p "${REPO_ROOT}/ubuntu/dists/${dist}/${comp}/binary-${arch}"
            done
        done
    done
    
    # Create pool directory
    mkdir -p "${REPO_ROOT}/ubuntu/pool/main"
    
    log_success "Repository structure created"
}

# Add package to repository
add_package() {
    local package_file=$1
    local distribution=$2
    
    if [[ ! -f "$package_file" ]]; then
        log_error "Package file not found: $package_file"
        return 1
    fi
    
    log_info "Adding package to repository: $(basename "$package_file")"
    
    # Copy package to pool
    cp "$package_file" "${REPO_ROOT}/ubuntu/pool/main/"
    
    # Sign the package
    dpkg-sig --sign builder "$package_file"
    
    log_success "Package added to repository"
}

# Generate repository metadata
generate_metadata() {
    log_info "Generating repository metadata..."
    
    cd "$REPO_ROOT"
    
    for dist in "${DISTRIBUTIONS[@]}"; do
        for comp in "${COMPONENTS[@]}"; do
            for arch in "${ARCHITECTURES[@]}"; do
                # Create Packages file
                log_info "Generating Packages file for $dist/$comp/$arch"
                
                packages_dir="ubuntu/dists/${dist}/${comp}/binary-${arch}"
                
                # Find packages for this architecture
                find ubuntu/pool/main -name "*.deb" | \
                    dpkg-scanpackages /dev/stdin /dev/null > \
                    "${packages_dir}/Packages"
                
                # Compress Packages file
                gzip -9c "${packages_dir}/Packages" > "${packages_dir}/Packages.gz"
                
                # Generate file size and checksums
                cd "${packages_dir}"
                apt-ftparchive release . > Release
                cd - > /dev/null
            done
        done
        
        # Generate Release file for distribution
        cd "ubuntu/dists/${dist}"
        apt-ftparchive release . > Release
        
        # Sign Release file
        gpg --clearsign -o InRelease Release
        gpg --armor --detach-sign -o Release.gpg Release
        
        cd - > /dev/null
    done
    
    log_success "Repository metadata generated"
}

# Create repository configuration files
create_repo_config() {
    log_info "Creating repository configuration files..."
    
    # Create sources.list example
    cat > "${REPO_ROOT}/sources.list.example" << EOF
# WGO APT Repository
# Add this line to your /etc/apt/sources.list or create a new file in /etc/apt/sources.list.d/
deb https://apt.wgo.sh/ubuntu stable main
EOF
    
    # Create installation script
    cat > "${REPO_ROOT}/install-repo.sh" << 'EOF'
#!/bin/bash
# WGO APT Repository Installation Script

set -e

# Add GPG key
curl -fsSL https://apt.wgo.sh/ubuntu/wgo.gpg | sudo apt-key add -

# Add repository
echo "deb https://apt.wgo.sh/ubuntu stable main" | sudo tee /etc/apt/sources.list.d/wgo.list

# Update package list
sudo apt-get update

echo "WGO APT repository added successfully!"
echo "You can now install WGO with: sudo apt-get install wgo"
EOF
    chmod +x "${REPO_ROOT}/install-repo.sh"
    
    # Create nginx configuration example
    cat > "${REPO_ROOT}/nginx.conf.example" << EOF
# Nginx configuration for WGO APT repository
server {
    listen 80;
    server_name apt.wgo.sh;
    
    # Redirect to HTTPS
    return 301 https://\$server_name\$request_uri;
}

server {
    listen 443 ssl http2;
    server_name apt.wgo.sh;
    
    # SSL configuration (replace with your certificates)
    ssl_certificate /path/to/your/cert.pem;
    ssl_certificate_key /path/to/your/key.pem;
    
    root ${REPO_ROOT}/ubuntu;
    index index.html;
    
    location / {
        try_files \$uri \$uri/ =404;
        autoindex on;
    }
    
    # Security headers
    add_header X-Frame-Options DENY;
    add_header X-Content-Type-Options nosniff;
    add_header X-XSS-Protection "1; mode=block";
    
    # Cache control
    location ~* \.(deb|gz|bz2|xz)$ {
        expires 1d;
        add_header Cache-Control "public, immutable";
    }
}
EOF
    
    log_success "Repository configuration files created"
}

# Test repository
test_repository() {
    log_info "Testing repository setup..."
    
    # Create temporary sources.list
    local temp_sources="/tmp/test-sources.list"
    echo "deb [trusted=yes] file://${REPO_ROOT}/ubuntu stable main" > "$temp_sources"
    
    # Test apt-get update
    if sudo apt-get update -o Dir::Etc::sourcelist="$temp_sources" -o Dir::Etc::sourceparts="-" -o APT::Get::List-Cleanup="0"; then
        log_success "Repository structure is valid"
    else
        log_error "Repository test failed"
        return 1
    fi
    
    # Clean up
    rm -f "$temp_sources"
}

# Generate GitHub Pages compatible structure
generate_github_pages() {
    log_info "Generating GitHub Pages compatible structure..."
    
    # Create index.html
    cat > "${REPO_ROOT}/index.html" << EOF
<!DOCTYPE html>
<html>
<head>
    <title>WGO APT Repository</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        .container { max-width: 800px; margin: 0 auto; }
        .code { background: #f4f4f4; padding: 10px; border-radius: 5px; font-family: monospace; }
        .header { text-align: center; margin-bottom: 40px; }
    </style>
</head>
<body>
    <div class="container">
        <div class="header">
            <h1>WGO APT Repository</h1>
            <p>Official APT repository for WGO (What's Going On)</p>
        </div>
        
        <h2>Quick Installation</h2>
        <div class="code">
            curl -fsSL https://apt.wgo.sh/ubuntu/install-repo.sh | sudo bash<br>
            sudo apt-get install wgo
        </div>
        
        <h2>Manual Installation</h2>
        <ol>
            <li>Add the GPG key:
                <div class="code">curl -fsSL https://apt.wgo.sh/ubuntu/wgo.gpg | sudo apt-key add -</div>
            </li>
            <li>Add the repository:
                <div class="code">echo "deb https://apt.wgo.sh/ubuntu stable main" | sudo tee /etc/apt/sources.list.d/wgo.list</div>
            </li>
            <li>Update and install:
                <div class="code">sudo apt-get update<br>sudo apt-get install wgo</div>
            </li>
        </ol>
        
        <h2>Available Distributions</h2>
        <ul>
            <li><strong>stable</strong> - Stable releases</li>
            <li><strong>testing</strong> - Beta releases</li>
        </ul>
        
        <h2>Supported Architectures</h2>
        <ul>
            <li>amd64 (x86_64)</li>
            <li>arm64 (aarch64)</li>
            <li>armhf (ARM 32-bit)</li>
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
    echo "🚀 WGO APT Repository Setup"
    echo "=========================="
    echo ""
    
    check_dependencies
    create_gpg_key
    create_repo_structure
    
    # Look for existing packages to add
    if ls *.deb >/dev/null 2>&1; then
        log_info "Found existing .deb packages"
        for pkg in *.deb; do
            add_package "$pkg" "stable"
        done
    else
        log_warn "No .deb packages found in current directory"
        log_info "Build packages first with: ./packaging/apt/build-deb.sh"
    fi
    
    generate_metadata
    create_repo_config
    generate_github_pages
    test_repository
    
    echo ""
    log_success "APT repository setup completed!"
    echo ""
    echo "Repository location: $REPO_ROOT"
    echo "GPG public key: ${REPO_ROOT}/wgo.gpg"
    echo "Installation script: ${REPO_ROOT}/install-repo.sh"
    echo ""
    echo "Next steps:"
    echo "1. Upload the contents of ${REPO_ROOT}/ubuntu to your web server"
    echo "2. Configure your web server (see nginx.conf.example)"
    echo "3. Set up HTTPS with proper certificates"
    echo "4. Test the repository with: curl -fsSL https://apt.wgo.sh/ubuntu/install-repo.sh | sudo bash"
}

# Run main function
main "$@"