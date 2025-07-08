#!/bin/bash
# Build script for Debian/Ubuntu packages
# This is called by GoReleaser but can also be run manually

set -e

PACKAGE_NAME="wgo"
VERSION="${1:-dev}"
ARCH="${2:-amd64}"
BUILD_DIR="build/deb"

echo "Building DEB package for WGO ${VERSION} (${ARCH})..."

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
Homepage: https://github.com/yairfalse/wgo
Description: Git diff for infrastructure - simple drift detection
 WGO (What's Going On) is a comprehensive infrastructure drift detection tool
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
        echo "WGO installed successfully!"
        echo "Run 'wgo version' to verify installation"
        echo "Run 'wgo --help' to get started"
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

echo "DEB package structure created successfully"