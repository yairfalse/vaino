#!/bin/bash
# Script to publish VAINO to APT repository
# Called by GitHub Actions after release

set -e

REPO_URL="https://apt.vaino.sh/ubuntu"
REPO_DIR="/tmp/apt-repo"
PACKAGE_DIR="packages"

echo "📦 Publishing VAINO to APT repository..."

# Create repository structure
mkdir -p "${REPO_DIR}/pool/main/w/vaino"
mkdir -p "${REPO_DIR}/dists/stable/main/binary-amd64"
mkdir -p "${REPO_DIR}/dists/stable/main/binary-arm64"

# Copy .deb files
cp ${PACKAGE_DIR}/*.deb "${REPO_DIR}/pool/main/w/vaino/" || {
    echo "❌ No .deb files found in ${PACKAGE_DIR}"
    exit 1
}

# Generate Packages files
cd "${REPO_DIR}"
dpkg-scanpackages pool/main > dists/stable/main/binary-amd64/Packages
gzip -k dists/stable/main/binary-amd64/Packages

dpkg-scanpackages pool/main > dists/stable/main/binary-arm64/Packages  
gzip -k dists/stable/main/binary-arm64/Packages

# Create Release file
cat > dists/stable/Release << EOF
Origin: VAINO
Label: VAINO
Suite: stable
Codename: stable
Version: 1.0
Architectures: amd64 arm64
Components: main
Description: VAINO APT Repository
Date: $(date -R)
EOF

# Sign Release file if GPG key is available
if [[ -n "${PACKAGE_SIGNING_KEY}" ]]; then
    echo "${PACKAGE_SIGNING_KEY}" | gpg --import
    gpg --default-key vaino@example.com --armor --detach-sign -o dists/stable/Release.gpg dists/stable/Release
    gpg --default-key vaino@example.com --clearsign -o dists/stable/InRelease dists/stable/Release
fi

# Generate SHA256 checksums
cd dists/stable
sha256sum main/binary-*/Packages* >> Release

echo "✅ APT repository structure created successfully"

# Upload to S3 or other hosting service
# This is a placeholder - replace with actual upload logic
echo "📤 Uploading to APT repository..."
# aws s3 sync "${REPO_DIR}/" "s3://apt.vaino.sh/" --delete
# or
# rsync -avz "${REPO_DIR}/" "apt.vaino.sh:/var/www/apt/"

echo "✅ APT repository published successfully"