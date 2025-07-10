#!/bin/bash
# Build script for Debian/Ubuntu packages
# This is called by GoReleaser but can also be run manually

set -e

PACKAGE_NAME="vaino"
VERSION="${1:-dev}"
ARCH="${2:-amd64}"
BUILD_DIR="build/deb"

echo "Building DEB package for VAINO ${VERSION} (${ARCH})..."

# Create package structure
mkdir -p "${BUILD_DIR}/DEBIAN"
mkdir -p "${BUILD_DIR}/usr/bin"
mkdir -p "${BUILD_DIR}/usr/share/doc/${PACKAGE_NAME}"
mkdir -p "${BUILD_DIR}/usr/share/bash-completion/completions"
mkdir -p "${BUILD_DIR}/usr/share/zsh/site-functions"
mkdir -p "${BUILD_DIR}/usr/share/fish/vendor_completions.d"

# Create control file
cat > "${BUILD_DIR}/DEBIAN/control" << EOF
Package: ${PACKAGE_NAME}
Version: ${VERSION}
Architecture: ${ARCH}
Maintainer: Yair <yair@example.com>
Depends: git
Recommends: terraform, awscli
Section: utils
Priority: optional
Homepage: https://github.com/yairfalse/vaino
Description: Git diff for infrastructure - simple drift detection
 VAINO (What's Going On) is a comprehensive infrastructure drift detection tool
 that helps you track changes in your infrastructure over time.
 .
 Features:
  - Multi-provider support (Terraform, AWS, GCP, Kubernetes)
  - Smart auto-discovery
  - Time-based comparisons
  - Unix-style output
  - Zero configuration
EOF

# Create postinst script
cat > "${BUILD_DIR}/DEBIAN/postinst" << 'EOF'
#!/bin/bash
set -e

case "$1" in
    configure)
        echo "VAINO installed successfully!"
        echo "Run 'vaino version' to verify installation"
        echo "Run 'vaino --help' to get started"
        ;;
    abort-upgrade|abort-remove|abort-deconfigure)
        ;;
    *)
        echo "postinst called with unknown argument: $1" >&2
        exit 1
        ;;
esac

exit 0
EOF
chmod 755 "${BUILD_DIR}/DEBIAN/postinst"

# Create prerm script
cat > "${BUILD_DIR}/DEBIAN/prerm" << 'EOF'
#!/bin/bash
set -e

case "$1" in
    remove|upgrade|deconfigure)
        # Nothing to do
        ;;
    failed-upgrade)
        ;;
    *)
        echo "prerm called with unknown argument: $1" >&2
        exit 1
        ;;
esac

exit 0
EOF
chmod 755 "${BUILD_DIR}/DEBIAN/prerm"

# Copy binary
if [[ -f "vaino" ]]; then
    cp vaino "${BUILD_DIR}/usr/bin/"
    chmod 755 "${BUILD_DIR}/usr/bin/vaino"
else
    echo "Error: vaino binary not found. Please build it first."
    exit 1
fi

# Copy documentation
if [[ -f "README.md" ]]; then
    cp README.md "${BUILD_DIR}/usr/share/doc/${PACKAGE_NAME}/"
fi

if [[ -f "LICENSE" ]]; then
    cp LICENSE "${BUILD_DIR}/usr/share/doc/${PACKAGE_NAME}/"
fi

# Generate and copy completions
if [[ -f "vaino" ]]; then
    ./vaino completion bash > "${BUILD_DIR}/usr/share/bash-completion/completions/vaino" 2>/dev/null || echo "Warning: Failed to generate bash completion"
    ./vaino completion zsh > "${BUILD_DIR}/usr/share/zsh/site-functions/_vaino" 2>/dev/null || echo "Warning: Failed to generate zsh completion"
    ./vaino completion fish > "${BUILD_DIR}/usr/share/fish/vendor_completions.d/vaino.fish" 2>/dev/null || echo "Warning: Failed to generate fish completion"
fi

# Build the package
echo "Building DEB package..."
dpkg-deb --build "${BUILD_DIR}" "${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"

echo "DEB package built successfully: ${PACKAGE_NAME}_${VERSION}_${ARCH}.deb"