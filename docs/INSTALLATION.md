# Installation Guide

VAINO can be installed on any platform with a single command. Choose the method that works best for your system.

## 🚀 Quick Install (Recommended)

The universal installer automatically detects your system and installs VAINO:

```bash
curl -sSL https://install.wgo.sh | bash
```

This script will:
- Detect your operating system and architecture
- Try to use your system's package manager first
- Fall back to direct binary installation if needed
- Install shell completions automatically

## 📦 Package Managers

### macOS (Homebrew)

```bash
brew install yairfalse/wgo/wgo
```

Or tap the repository first:

```bash
brew tap yairfalse/wgo
brew install wgo
```

### Linux

#### Ubuntu/Debian (APT)

```bash
# Add VAINO repository
curl -fsSL https://apt.wgo.sh/ubuntu/wgo.gpg | sudo apt-key add -
echo "deb https://apt.wgo.sh/ubuntu stable main" | sudo tee /etc/apt/sources.list.d/wgo.list

# Install VAINO
sudo apt-get update
sudo apt-get install wgo
```

#### RHEL/CentOS/Fedora (YUM/DNF)

```bash
# Add VAINO repository
sudo curl -fsSL https://yum.wgo.sh/rhel/wgo.repo -o /etc/yum.repos.d/wgo.repo

# Install VAINO (use dnf on Fedora/RHEL 8+)
sudo yum install wgo
# or
sudo dnf install wgo
```

#### Arch Linux (AUR)

```bash
yay -S wgo-bin
# or
paru -S wgo-bin
```

#### Snap (Universal Linux Package)

```bash
sudo snap install wgo
```

### Windows

#### Chocolatey

```powershell
choco install wgo
```

#### Scoop

```powershell
scoop bucket add wgo https://github.com/yairfalse/scoop-wgo
scoop install wgo
```

#### WSL (Windows Subsystem for Linux)

Use the Linux installation methods above within WSL.

## 🐳 Docker

Run VAINO in a container:

```bash
# Run a single command
docker run --rm -v $(pwd):/workspace yairfalse/wgo:latest scan

# Interactive mode
docker run --rm -it -v $(pwd):/workspace yairfalse/wgo:latest

# With AWS credentials
docker run --rm \
  -v $(pwd):/workspace \
  -v ~/.aws:/home/wgo/.aws:ro \
  yairfalse/wgo:latest scan --provider aws
```

## 📥 Direct Download

Download pre-built binaries from the [releases page](https://github.com/yairfalse/wgo/releases).

### Linux/macOS

```bash
# Download latest release (replace VERSION and PLATFORM)
curl -L https://github.com/yairfalse/wgo/releases/download/VERSION/wgo_PLATFORM.tar.gz | tar xz

# Move to PATH
sudo mv wgo /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/wgo
```

### Windows

1. Download the `.zip` file from the [releases page](https://github.com/yairfalse/wgo/releases)
2. Extract the archive
3. Add the directory to your PATH
4. Or move `wgo.exe` to a directory already in PATH

## 🔧 Build from Source

Requirements:
- Go 1.21 or later
- Git

```bash
# Clone the repository
git clone https://github.com/yairfalse/wgo.git
cd wgo

# Build and install
go install ./cmd/wgo

# Or use make
make install
```

## 🎯 Shell Completions

Shell completions are automatically installed with most package managers. For manual installation:

### Bash

```bash
# Generate completion script
wgo completion bash > wgo-completion.bash

# Install for current user
mkdir -p ~/.local/share/bash-completion/completions
mv wgo-completion.bash ~/.local/share/bash-completion/completions/wgo

# Or install system-wide
sudo mv wgo-completion.bash /etc/bash_completion.d/wgo
```

### Zsh

```bash
# Generate completion script
wgo completion zsh > _wgo

# Install
sudo mv _wgo /usr/local/share/zsh/site-functions/
```

### Fish

```bash
# Generate and install
wgo completion fish > ~/.config/fish/completions/wgo.fish
```

### PowerShell

```powershell
# Generate completion script
wgo completion powershell > wgo.ps1

# Install (add to profile)
echo ". $(pwd)/wgo.ps1" >> $PROFILE
```

## ✅ Verify Installation

After installation, verify VAINO is working:

```bash
# Check version
wgo version

# Get help
wgo --help

# Run a test scan
wgo scan
```

## 🆙 Updating

### Package Managers

Use your package manager's update command:

```bash
# Homebrew
brew upgrade wgo

# APT
sudo apt-get update && sudo apt-get upgrade wgo

# YUM/DNF
sudo yum update wgo
# or
sudo dnf update wgo

# Chocolatey
choco upgrade wgo

# Scoop
scoop update wgo
```

### Universal Installer

Re-run the install script:

```bash
curl -sSL https://install.wgo.sh | bash
```

## 🗑️ Uninstalling

### Package Managers

```bash
# Homebrew
brew uninstall wgo

# APT
sudo apt-get remove wgo

# YUM/DNF
sudo yum remove wgo
# or
sudo dnf remove wgo

# Chocolatey
choco uninstall wgo

# Scoop
scoop uninstall wgo
```

### Manual Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/wgo

# Remove completions
sudo rm /etc/bash_completion.d/wgo
sudo rm /usr/local/share/zsh/site-functions/_wgo
rm ~/.config/fish/completions/wgo.fish

# Remove configuration
rm -rf ~/.wgo
```

## 🤔 Troubleshooting

### Command not found

Make sure `/usr/local/bin` is in your PATH:

```bash
echo 'export PATH="/usr/local/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
```

### Permission denied

The installer needs write access to `/usr/local/bin`. Either:
- Run with `sudo`: `curl -sSL https://install.wgo.sh | sudo bash`
- Install to a user directory and add to PATH

### Certificate errors

If you get SSL/TLS errors, try:

```bash
# Update certificates
sudo apt-get install ca-certificates  # Debian/Ubuntu
sudo yum install ca-certificates      # RHEL/CentOS

# Or bypass (not recommended)
curl -sSLk https://install.wgo.sh | bash
```

### Package manager not detected

The universal installer will fall back to direct binary download if your package manager isn't detected.

## 📞 Support

- [GitHub Issues](https://github.com/yairfalse/wgo/issues)
- [Documentation](https://github.com/yairfalse/wgo/wiki)
- [Discord Community](https://discord.gg/wgo)