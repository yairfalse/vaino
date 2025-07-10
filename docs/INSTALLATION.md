# Installation Guide

VAINO can be installed on any platform with a single command. Choose the method that works best for your system.

## 🚀 Quick Install (Recommended)

The universal installer automatically detects your system and installs VAINO:

```bash
curl -sSL https://install.vaino.sh | bash
```

This script will:
- Detect your operating system and architecture
- Try to use your system's package manager first
- Fall back to direct binary installation if needed
- Install shell completions automatically

## 📦 Package Managers

### macOS (Homebrew)

```bash
brew install yairfalse/vaino/vaino
```

Or tap the repository first:

```bash
brew tap yairfalse/vaino
brew install vaino
```

### Linux

#### Ubuntu/Debian (APT)

```bash
# Add VAINO repository
curl -fsSL https://apt.vaino.sh/ubuntu/vaino.gpg | sudo apt-key add -
echo "deb https://apt.vaino.sh/ubuntu stable main" | sudo tee /etc/apt/sources.list.d/vaino.list

# Install VAINO
sudo apt-get update
sudo apt-get install vaino
```

#### RHEL/CentOS/Fedora (YUM/DNF)

```bash
# Add VAINO repository
sudo curl -fsSL https://yum.vaino.sh/rhel/vaino.repo -o /etc/yum.repos.d/vaino.repo

# Install VAINO (use dnf on Fedora/RHEL 8+)
sudo yum install vaino
# or
sudo dnf install vaino
```

#### Arch Linux (AUR)

```bash
yay -S vaino-bin
# or
paru -S vaino-bin
```

#### Snap (Universal Linux Package)

```bash
sudo snap install vaino
```

### Windows

#### Chocolatey

```powershell
choco install vaino
```

#### Scoop

```powershell
scoop bucket add vaino https://github.com/yairfalse/scoop-vaino
scoop install vaino
```

#### WSL (Windows Subsystem for Linux)

Use the Linux installation methods above within WSL.

## 🐳 Docker

Run VAINO in a container:

```bash
# Run a single command
docker run --rm -v $(pwd):/workspace yairfalse/vaino:latest scan

# Interactive mode
docker run --rm -it -v $(pwd):/workspace yairfalse/vaino:latest

# With AWS credentials
docker run --rm \
  -v $(pwd):/workspace \
  -v ~/.aws:/home/vaino/.aws:ro \
  yairfalse/vaino:latest scan --provider aws
```

## 📥 Direct Download

Download pre-built binaries from the [releases page](https://github.com/yairfalse/vaino/releases).

### Linux/macOS

```bash
# Download latest release (replace VERSION and PLATFORM)
curl -L https://github.com/yairfalse/vaino/releases/download/VERSION/vaino_PLATFORM.tar.gz | tar xz

# Move to PATH
sudo mv vaino /usr/local/bin/

# Make executable
sudo chmod +x /usr/local/bin/vaino
```

### Windows

1. Download the `.zip` file from the [releases page](https://github.com/yairfalse/vaino/releases)
2. Extract the archive
3. Add the directory to your PATH
4. Or move `vaino.exe` to a directory already in PATH

## 🔧 Build from Source

Requirements:
- Go 1.21 or later
- Git

```bash
# Clone the repository
git clone https://github.com/yairfalse/vaino.git
cd vaino

# Build and install
go install ./cmd/vaino

# Or use make
make install
```

## 🎯 Shell Completions

Shell completions are automatically installed with most package managers. For manual installation:

### Bash

```bash
# Generate completion script
vaino completion bash > vaino-completion.bash

# Install for current user
mkdir -p ~/.local/share/bash-completion/completions
mv vaino-completion.bash ~/.local/share/bash-completion/completions/vaino

# Or install system-wide
sudo mv vaino-completion.bash /etc/bash_completion.d/vaino
```

### Zsh

```bash
# Generate completion script
vaino completion zsh > _vaino

# Install
sudo mv _vaino /usr/local/share/zsh/site-functions/
```

### Fish

```bash
# Generate and install
vaino completion fish > ~/.config/fish/completions/vaino.fish
```

### PowerShell

```powershell
# Generate completion script
vaino completion powershell > vaino.ps1

# Install (add to profile)
echo ". $(pwd)/vaino.ps1" >> $PROFILE
```

## ✅ Verify Installation

After installation, verify VAINO is working:

```bash
# Check version
vaino version

# Get help
vaino --help

# Run a test scan
vaino scan
```

## 🆙 Updating

### Package Managers

Use your package manager's update command:

```bash
# Homebrew
brew upgrade vaino

# APT
sudo apt-get update && sudo apt-get upgrade vaino

# YUM/DNF
sudo yum update vaino
# or
sudo dnf update vaino

# Chocolatey
choco upgrade vaino

# Scoop
scoop update vaino
```

### Universal Installer

Re-run the install script:

```bash
curl -sSL https://install.vaino.sh | bash
```

## 🗑️ Uninstalling

### Package Managers

```bash
# Homebrew
brew uninstall vaino

# APT
sudo apt-get remove vaino

# YUM/DNF
sudo yum remove vaino
# or
sudo dnf remove vaino

# Chocolatey
choco uninstall vaino

# Scoop
scoop uninstall vaino
```

### Manual Uninstall

```bash
# Remove binary
sudo rm /usr/local/bin/vaino

# Remove completions
sudo rm /etc/bash_completion.d/vaino
sudo rm /usr/local/share/zsh/site-functions/_vaino
rm ~/.config/fish/completions/vaino.fish

# Remove configuration
rm -rf ~/.vaino
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
- Run with `sudo`: `curl -sSL https://install.vaino.sh | sudo bash`
- Install to a user directory and add to PATH

### Certificate errors

If you get SSL/TLS errors, try:

```bash
# Update certificates
sudo apt-get install ca-certificates  # Debian/Ubuntu
sudo yum install ca-certificates      # RHEL/CentOS

# Or bypass (not recommended)
curl -sSLk https://install.vaino.sh | bash
```

### Package manager not detected

The universal installer will fall back to direct binary download if your package manager isn't detected.

## 📞 Support

- [GitHub Issues](https://github.com/yairfalse/vaino/issues)
- [Documentation](https://github.com/yairfalse/vaino/wiki)
- [Discord Community](https://discord.gg/vaino)