#!/bin/bash
# Build script for RPM packages
# This script builds RPM packages for different architectures and distributions

set -e

PACKAGE_NAME="vaino"
VERSION="${1:-dev}"
ARCH="${2:-x86_64}"
DIST="${3:-el9}"
BUILD_DIR="/tmp/rpmbuild-${PACKAGE_NAME}"

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
    
    for dep in rpmbuild rpm-build; do
        if ! command -v "$dep" >/dev/null 2>&1; then
            missing_deps+=("$dep")
        fi
    done
    
    if [[ ${#missing_deps[@]} -gt 0 ]]; then
        log_error "Missing dependencies: ${missing_deps[*]}"
        log_info "Install them with:"
        echo "  RHEL/CentOS: sudo dnf install rpm-build rpm-devel rpmdevtools"
        echo "  Fedora: sudo dnf install rpm-build rpm-devel rpmdevtools"
        echo "  Ubuntu/Debian: sudo apt-get install rpm"
        exit 1
    fi
    
    log_success "All dependencies available"
}

# Create build environment
create_build_env() {
    log_info "Creating build environment..."
    
    # Clean and recreate build directory
    rm -rf "$BUILD_DIR"
    mkdir -p "$BUILD_DIR"/{BUILD,BUILDROOT,RPMS,SOURCES,SPECS,SRPMS}
    
    # Copy spec file
    cp packaging/yum/vaino.spec "$BUILD_DIR/SPECS/"
    
    log_success "Build environment created"
}

# Prepare source archive
prepare_source() {
    log_info "Preparing source archive..."
    
    # Create source directory
    local src_dir="${BUILD_DIR}/SOURCES/vaino-${VERSION}"
    mkdir -p "$src_dir"
    
    # Copy binary
    if [[ -f "vaino" ]]; then
        cp vaino "$src_dir/"
        chmod +x "$src_dir/vaino"
    else
        log_error "vaino binary not found. Please build it first with: go build -o vaino ./cmd/vaino"
        exit 1
    fi
    
    # Copy documentation
    if [[ -f "README.md" ]]; then
        cp README.md "$src_dir/"
    fi
    
    if [[ -f "LICENSE" ]]; then
        cp LICENSE "$src_dir/"
    fi
    
    # Generate completions
    log_info "Generating shell completions..."
    mkdir -p "$src_dir/completions"
    ./vaino completion bash > "$src_dir/completions/vaino.bash" 2>/dev/null || log_warn "Failed to generate bash completion"
    ./vaino completion zsh > "$src_dir/completions/vaino.zsh" 2>/dev/null || log_warn "Failed to generate zsh completion"
    ./vaino completion fish > "$src_dir/completions/vaino.fish" 2>/dev/null || log_warn "Failed to generate fish completion"
    
    # Create source tarball
    cd "${BUILD_DIR}/SOURCES"
    tar -czf "vaino-${VERSION}.tar.gz" "vaino-${VERSION}/"
    rm -rf "vaino-${VERSION}/"
    cd - > /dev/null
    
    log_success "Source archive prepared"
}

# Build RPM package
build_rpm() {
    log_info "Building RPM package for ${PACKAGE_NAME} ${VERSION} (${ARCH}, ${DIST})..."
    
    # Build the RPM
    rpmbuild --define="_topdir $BUILD_DIR" \
             --define="version $VERSION" \
             --define="dist .${DIST}" \
             --target="$ARCH" \
             -ba "$BUILD_DIR/SPECS/vaino.spec"
    
    # Find the built RPM
    local rpm_file
    rpm_file=$(find "$BUILD_DIR/RPMS" -name "*.rpm" -type f | head -1)
    
    if [[ -f "$rpm_file" ]]; then
        # Copy to current directory
        cp "$rpm_file" .
        log_success "RPM built successfully: $(basename "$rpm_file")"
        
        # Display package information
        rpm -qip "$(basename "$rpm_file")"
    else
        log_error "RPM build failed - no package file found"
        exit 1
    fi
    
    # Also copy SRPM if it exists
    local srpm_file
    srpm_file=$(find "$BUILD_DIR/SRPMS" -name "*.src.rpm" -type f | head -1)
    if [[ -f "$srpm_file" ]]; then
        cp "$srpm_file" .
        log_success "SRPM built successfully: $(basename "$srpm_file")"
    fi
}

# Test the built package
test_package() {
    local rpm_file
    rpm_file=$(find . -name "vaino-*.rpm" -not -name "*.src.rpm" -type f | head -1)
    
    if [[ ! -f "$rpm_file" ]]; then
        log_error "No RPM file found for testing"
        return 1
    fi
    
    log_info "Testing package: $rpm_file"
    
    # Check package contents
    log_info "Package contents:"
    rpm -qlp "$rpm_file"
    
    # Verify dependencies
    log_info "Package dependencies:"
    rpm -qRp "$rpm_file"
    
    # Check for common issues
    log_info "Running package checks..."
    
    # Check if binary is executable
    if rpm -qlp "$rpm_file" | grep -q "/usr/bin/vaino"; then
        log_success "Binary is included in package"
    else
        log_error "Binary is missing from package"
        return 1
    fi
    
    # Check if completions are included
    if rpm -qlp "$rpm_file" | grep -q "completion"; then
        log_success "Shell completions are included"
    else
        log_warn "Shell completions might be missing"
    fi
    
    log_success "Package testing completed"
}

# Clean up build environment
cleanup() {
    if [[ -d "$BUILD_DIR" ]]; then
        log_info "Cleaning up build environment..."
        rm -rf "$BUILD_DIR"
    fi
}

# Main execution
main() {
    echo "🏗️  VAINO RPM Package Builder"
    echo "=========================="
    echo ""
    echo "Building: ${PACKAGE_NAME} ${VERSION} for ${ARCH} (${DIST})"
    echo ""
    
    check_dependencies
    create_build_env
    prepare_source
    build_rpm
    test_package
    
    echo ""
    log_success "RPM package build completed!"
    echo ""
    echo "Files created:"
    ls -la *.rpm 2>/dev/null || echo "  No RPM files found"
    echo ""
    echo "Next steps:"
    echo "1. Sign the package: rpm --addsign *.rpm"
    echo "2. Add to repository: ./packaging/yum/setup-repo.sh"
    echo "3. Test installation: sudo dnf install ./vaino-*.rpm"
}

# Set up cleanup trap
trap cleanup EXIT

# Run main function
main "$@"