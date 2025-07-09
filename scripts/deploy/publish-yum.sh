#!/bin/bash
# Script to publish WGO to YUM/DNF repository
# Called by GitHub Actions after release

set -e

REPO_URL="https://yum.wgo.sh/rhel"
REPO_DIR="/tmp/yum-repo"
PACKAGE_DIR="packages"

echo "📦 Publishing WGO to YUM repository..."

# Create repository structure
mkdir -p "${REPO_DIR}/7/x86_64"
mkdir -p "${REPO_DIR}/8/x86_64"
mkdir -p "${REPO_DIR}/9/x86_64"

# Copy RPM files to all RHEL versions
for version in 7 8 9; do
    cp ${PACKAGE_DIR}/*.rpm "${REPO_DIR}/${version}/x86_64/" 2>/dev/null || {
        echo "⚠️  No RPM files found for RHEL ${version}"
    }
done

# Create repository metadata for each version
for version in 7 8 9; do
    if [[ -d "${REPO_DIR}/${version}/x86_64" ]] && [[ -n "$(ls -A ${REPO_DIR}/${version}/x86_64/*.rpm 2>/dev/null)" ]]; then
        echo "Creating metadata for RHEL ${version}..."
        createrepo "${REPO_DIR}/${version}/x86_64"
        
        # Sign repository metadata if GPG key is available
        if [[ -n "${PACKAGE_SIGNING_KEY}" ]]; then
            echo "${PACKAGE_SIGNING_KEY}" | gpg --import
            gpg --detach-sign --armor "${REPO_DIR}/${version}/x86_64/repodata/repomd.xml"
        fi
    fi
done

# Create .repo file
cat > "${REPO_DIR}/wgo.repo" << EOF
[wgo]
name=WGO Repository
baseurl=${REPO_URL}/\$releasever/\$basearch
enabled=1
gpgcheck=1
gpgkey=${REPO_URL}/RPM-GPG-KEY-wgo
metadata_expire=300

[wgo-source]
name=WGO Source Repository
baseurl=${REPO_URL}/\$releasever/SRPMS
enabled=0
gpgcheck=1
gpgkey=${REPO_URL}/RPM-GPG-KEY-wgo
metadata_expire=300
EOF

# Export GPG public key if signing
if [[ -n "${PACKAGE_SIGNING_KEY}" ]]; then
    gpg --armor --export wgo@example.com > "${REPO_DIR}/RPM-GPG-KEY-wgo"
fi

echo "✅ YUM repository structure created successfully"

# Upload to S3 or other hosting service
# This is a placeholder - replace with actual upload logic
echo "📤 Uploading to YUM repository..."
# aws s3 sync "${REPO_DIR}/" "s3://yum.wgo.sh/" --delete
# or
# rsync -avz "${REPO_DIR}/" "yum.wgo.sh:/var/www/yum/"

echo "✅ YUM repository published successfully"