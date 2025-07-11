#!/bin/bash
# GPG Key Management Script for VAINO
# This script manages GPG keys for package signing and verification

set -e

# Configuration
GPG_KEY_ID="VAINO Package Signing Key"
GPG_KEY_EMAIL="packages@vaino.sh"
GPG_KEY_REAL_NAME="VAINO Package Signing"
GPG_KEY_COMMENT="Used for signing VAINO packages"
KEY_LENGTH=4096
EXPIRY_PERIOD="2y"

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
    
    if ! command -v gpg >/dev/null 2>&1; then
        log_error "GPG is not installed"
        log_info "Install it with:"
        echo "  macOS: brew install gnupg"
        echo "  Ubuntu/Debian: sudo apt-get install gnupg"
        echo "  RHEL/CentOS: sudo dnf install gnupg"
        exit 1
    fi
    
    log_success "GPG is available"
}

# Generate GPG key
generate_key() {
    log_info "Generating GPG key for package signing..."
    
    # Check if key already exists
    if gpg --list-secret-keys | grep -q "$GPG_KEY_EMAIL"; then
        log_warn "GPG key already exists for $GPG_KEY_EMAIL"
        return 0
    fi
    
    # Generate key configuration
    local key_config=$(mktemp)
    cat > "$key_config" << EOF
%echo Generating GPG key for VAINO package signing
Key-Type: RSA
Key-Length: $KEY_LENGTH
Subkey-Type: RSA
Subkey-Length: $KEY_LENGTH
Name-Real: $GPG_KEY_REAL_NAME
Name-Comment: $GPG_KEY_COMMENT
Name-Email: $GPG_KEY_EMAIL
Expire-Date: $EXPIRY_PERIOD
%no-protection
%commit
%echo done
EOF
    
    # Generate the key
    gpg --batch --generate-key "$key_config"
    rm "$key_config"
    
    log_success "GPG key generated successfully"
    
    # Display key information
    gpg --list-secret-keys "$GPG_KEY_EMAIL"
}

# Export public key
export_public_key() {
    log_info "Exporting public key..."
    
    # Export ASCII armored public key
    gpg --armor --export "$GPG_KEY_EMAIL" > vaino-signing-key.asc
    
    # Export binary public key for APT
    gpg --export "$GPG_KEY_EMAIL" > vaino-signing-key.gpg
    
    log_success "Public key exported to vaino-signing-key.asc and vaino-signing-key.gpg"
}

# Create GPG configuration for automated signing
setup_automated_signing() {
    log_info "Setting up automated signing configuration..."
    
    # Create GPG agent configuration
    local gpg_agent_conf="$HOME/.gnupg/gpg-agent.conf"
    
    if [[ ! -f "$gpg_agent_conf" ]]; then
        mkdir -p "$HOME/.gnupg"
        chmod 700 "$HOME/.gnupg"
        
        cat > "$gpg_agent_conf" << EOF
# GPG Agent configuration for automated signing
default-cache-ttl 28800
max-cache-ttl 86400
allow-loopback-pinentry
EOF
        
        chmod 600 "$gpg_agent_conf"
        log_success "GPG agent configuration created"
    else
        log_info "GPG agent configuration already exists"
    fi
    
    # Create signing script
    cat > "sign-package.sh" << EOF
#!/bin/bash
# Package signing script
# Usage: ./sign-package.sh <package-file>

set -e

PACKAGE_FILE="\$1"
GPG_KEY_EMAIL="$GPG_KEY_EMAIL"

if [[ -z "\$PACKAGE_FILE" ]]; then
    echo "Usage: \$0 <package-file>"
    exit 1
fi

if [[ ! -f "\$PACKAGE_FILE" ]]; then
    echo "Package file not found: \$PACKAGE_FILE"
    exit 1
fi

echo "Signing package: \$PACKAGE_FILE"

# Detect package type and sign accordingly
case "\$PACKAGE_FILE" in
    *.deb)
        # Sign Debian package
        dpkg-sig --sign builder "\$PACKAGE_FILE"
        echo "Debian package signed successfully"
        ;;
    *.rpm)
        # Sign RPM package
        rpm --addsign "\$PACKAGE_FILE"
        echo "RPM package signed successfully"
        ;;
    *)
        # Generic file signing
        gpg --detach-sign --armor "\$PACKAGE_FILE"
        echo "File signed successfully: \$PACKAGE_FILE.asc"
        ;;
esac
EOF
    
    chmod +x "sign-package.sh"
    log_success "Package signing script created"
}

# Configure RPM signing
setup_rpm_signing() {
    log_info "Setting up RPM signing configuration..."
    
    # Create RPM macros file
    cat > "$HOME/.rpmmacros" << EOF
# RPM signing configuration
%_gpg_name $GPG_KEY_EMAIL
%_signature gpg
%_gpg_path $HOME/.gnupg
%_gpgbin $(which gpg)
%__gpg_sign_cmd %{__gpg} gpg --force-v3-sigs --batch --verbose --no-armor --passphrase-fd 3 --no-secmem-warning -u "%{_gpg_name}" -sbo %{__signature_filename} --digest-algo sha256 %{__plaintext_filename}
EOF
    
    log_success "RPM signing configuration created"
}

# Create GitHub Actions secrets template
create_secrets_template() {
    log_info "Creating GitHub Actions secrets template..."
    
    cat > "github-secrets.md" << EOF
# GitHub Actions Secrets for VAINO Release Pipeline

To enable automated package signing in the release pipeline, add these secrets to your GitHub repository:

## Required Secrets

1. **GPG_PRIVATE_KEY**
   - Description: Private GPG key for package signing
   - Value: Run \`gpg --armor --export-secret-keys $GPG_KEY_EMAIL\` and copy the output
   - Usage: Package signing in release workflow

2. **GPG_PASSPHRASE**
   - Description: Passphrase for the GPG private key
   - Value: The passphrase you set when creating the GPG key
   - Usage: Unlock GPG key for signing

3. **HOMEBREW_TAP_GITHUB_TOKEN**
   - Description: GitHub token for updating Homebrew tap
   - Value: GitHub personal access token with repo permissions
   - Usage: Update Homebrew formula automatically

4. **SCOOP_GITHUB_TOKEN**
   - Description: GitHub token for updating Scoop bucket
   - Value: GitHub personal access token with repo permissions
   - Usage: Update Scoop manifest automatically

5. **DOCKER_USERNAME**
   - Description: Docker Hub username
   - Value: Your Docker Hub username
   - Usage: Push Docker images

6. **DOCKER_PASSWORD**
   - Description: Docker Hub password/token
   - Value: Your Docker Hub password or access token
   - Usage: Authenticate with Docker Hub

## How to Add Secrets

1. Go to your GitHub repository
2. Click on "Settings" tab
3. In the left sidebar, click "Secrets and variables" → "Actions"
4. Click "New repository secret"
5. Enter the secret name and value
6. Click "Add secret"

## Testing

After adding the secrets, test the release pipeline by:
1. Creating a new git tag: \`git tag v1.0.0-test\`
2. Pushing the tag: \`git push origin v1.0.0-test\`
3. Check the GitHub Actions workflow for any errors

## Security Notes

- Never commit private keys or passphrases to version control
- Use repository secrets, not environment variables in workflows
- Regularly rotate your GPG keys and access tokens
- Monitor secret usage in the Actions tab
EOF
    
    log_success "GitHub Actions secrets template created: github-secrets.md"
}

# Verify GPG setup
verify_setup() {
    log_info "Verifying GPG setup..."
    
    # Check if key exists
    if ! gpg --list-secret-keys "$GPG_KEY_EMAIL" >/dev/null 2>&1; then
        log_error "GPG key not found for $GPG_KEY_EMAIL"
        return 1
    fi
    
    # Test signing
    echo "test" | gpg --clearsign -u "$GPG_KEY_EMAIL" >/dev/null 2>&1
    if [[ $? -eq 0 ]]; then
        log_success "GPG signing test passed"
    else
        log_error "GPG signing test failed"
        return 1
    fi
    
    # Check exported keys
    if [[ -f "vaino-signing-key.asc" ]]; then
        log_success "Public key exported successfully"
    else
        log_error "Public key export failed"
        return 1
    fi
    
    # Verify key information
    log_info "Key information:"
    gpg --list-keys "$GPG_KEY_EMAIL"
    
    log_success "GPG setup verification completed"
}

# Display usage information
display_usage() {
    cat << EOF
VAINO GPG Key Management Script

Usage: $0 [command]

Commands:
  generate    Generate a new GPG key for package signing
  export      Export the public key
  setup       Set up automated signing configuration
  verify      Verify the GPG setup
  secrets     Create GitHub Actions secrets template
  all         Run all setup steps (default)

Examples:
  $0 generate     # Generate new GPG key
  $0 export       # Export public key
  $0 setup        # Set up signing configuration
  $0 verify       # Verify everything is working
  $0 secrets      # Create secrets template
  $0 all          # Complete setup (default)

Files created:
  vaino-signing-key.asc     # ASCII armored public key
  vaino-signing-key.gpg     # Binary public key
  sign-package.sh         # Package signing script
  github-secrets.md       # GitHub Actions secrets guide
EOF
}

# Main execution
main() {
    local command="${1:-all}"
    
    echo "🔐 VAINO GPG Key Management"
    echo "========================"
    echo ""
    
    case "$command" in
        generate)
            check_dependencies
            generate_key
            ;;
        export)
            check_dependencies
            export_public_key
            ;;
        setup)
            check_dependencies
            setup_automated_signing
            setup_rpm_signing
            ;;
        verify)
            check_dependencies
            verify_setup
            ;;
        secrets)
            create_secrets_template
            ;;
        all)
            check_dependencies
            generate_key
            export_public_key
            setup_automated_signing
            setup_rpm_signing
            create_secrets_template
            verify_setup
            ;;
        help|--help|-h)
            display_usage
            exit 0
            ;;
        *)
            log_error "Unknown command: $command"
            display_usage
            exit 1
            ;;
    esac
    
    echo ""
    log_success "GPG key management completed!"
    echo ""
    echo "Next steps:"
    echo "1. Add the secrets to GitHub Actions (see github-secrets.md)"
    echo "2. Upload public keys to your package repositories"
    echo "3. Test package signing with ./sign-package.sh"
    echo "4. Create a test release to verify the pipeline"
}

# Run main function
main "$@"