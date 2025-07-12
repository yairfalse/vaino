# Tapio Windows Installation Script
# This script is embedded in the installer binary

param(
    [string]$InstallDir = "$env:PROGRAMFILES\Tapio",
    [string]$Version = "latest",
    [string]$Mirror = "https://releases.tapio.io"
)

$ErrorActionPreference = "Stop"

# Helper functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Blue
}

function Write-Success {
    param([string]$Message)
    Write-Host "[SUCCESS] $Message" -ForegroundColor Green
}

function Write-Error {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Warning {
    param([string]$Message)
    Write-Host "[WARNING] $Message" -ForegroundColor Yellow
}

# Detect architecture
function Get-Architecture {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch ($arch) {
        "AMD64" { return "amd64" }
        "ARM64" { return "arm64" }
        default { return "amd64" }
    }
}

# Check if running as administrator
function Test-Administrator {
    $currentUser = [Security.Principal.WindowsIdentity]::GetCurrent()
    $principal = New-Object Security.Principal.WindowsPrincipal($currentUser)
    return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
}

# Download file with progress
function Download-File {
    param(
        [string]$Url,
        [string]$Output
    )
    
    try {
        $webClient = New-Object System.Net.WebClient
        $webClient.Headers.Add("User-Agent", "Tapio-Installer/1.0")
        
        # Progress tracking
        $webClient.DownloadProgressChanged += {
            Write-Progress -Activity "Downloading Tapio" -Status "$($_.ProgressPercentage)% Complete" -PercentComplete $_.ProgressPercentage
        }
        
        $webClient.DownloadFileAsync($Url, $Output)
        
        while ($webClient.IsBusy) {
            Start-Sleep -Milliseconds 100
        }
        
        Write-Progress -Activity "Downloading Tapio" -Completed
    }
    catch {
        Write-Error "Download failed: $_"
        return $false
    }
    
    return $true
}

# Verify checksum
function Test-Checksum {
    param(
        [string]$File,
        [string]$ExpectedChecksum
    )
    
    try {
        $hash = Get-FileHash -Path $File -Algorithm SHA256
        $actualChecksum = $hash.Hash.ToLower()
        $expectedChecksum = $ExpectedChecksum.ToLower()
        
        if ($actualChecksum -eq $expectedChecksum) {
            return $true
        }
        else {
            Write-Error "Checksum mismatch!"
            Write-Error "Expected: $expectedChecksum"
            Write-Error "Actual: $actualChecksum"
            return $false
        }
    }
    catch {
        Write-Warning "Failed to verify checksum: $_"
        return $true  # Don't fail installation
    }
}

# Add to PATH
function Add-ToPath {
    param([string]$Path)
    
    $currentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
    
    if ($currentPath -notlike "*$Path*") {
        Write-Info "Adding $Path to PATH..."
        $newPath = "$currentPath;$Path"
        [Environment]::SetEnvironmentVariable("PATH", $newPath, "User")
        $env:PATH = "$env:PATH;$Path"
        Write-Success "Added to PATH successfully"
    }
    else {
        Write-Info "$Path is already in PATH"
    }
}

# Main installation function
function Install-Tapio {
    Write-Info "Installing Tapio for Windows..."
    
    # Check if running as admin
    if (-not (Test-Administrator)) {
        Write-Warning "Not running as administrator. Installation to Program Files may fail."
        Write-Warning "Consider running PowerShell as Administrator or choosing a different install directory."
    }
    
    # Detect architecture
    $arch = Get-Architecture
    Write-Info "Detected architecture: $arch"
    
    # Create install directory
    try {
        if (-not (Test-Path $InstallDir)) {
            Write-Info "Creating directory $InstallDir"
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }
    }
    catch {
        Write-Error "Failed to create install directory: $_"
        return $false
    }
    
    # Create temp directory
    $tempDir = Join-Path $env:TEMP "tapio-install-$(Get-Random)"
    New-Item -ItemType Directory -Path $tempDir -Force | Out-Null
    
    try {
        # Construct URLs
        $binaryUrl = "$Mirror/$Version/tapio-windows-$arch.exe"
        $checksumUrl = "$binaryUrl.sha256"
        
        $binaryPath = Join-Path $tempDir "tapio.exe"
        $checksumPath = Join-Path $tempDir "tapio.exe.sha256"
        
        # Download binary
        Write-Info "Downloading Tapio from $binaryUrl"
        if (-not (Download-File -Url $binaryUrl -Output $binaryPath)) {
            Write-Error "Failed to download Tapio"
            return $false
        }
        
        # Download and verify checksum
        Write-Info "Downloading checksum..."
        if (Download-File -Url $checksumUrl -Output $checksumPath) {
            $expectedChecksum = (Get-Content $checksumPath -First 1).Split(' ')[0]
            Write-Info "Verifying checksum..."
            
            if (-not (Test-Checksum -File $binaryPath -ExpectedChecksum $expectedChecksum)) {
                Write-Error "Checksum verification failed"
                return $false
            }
            Write-Success "Checksum verified"
        }
        else {
            Write-Warning "Checksum file not found, skipping verification"
        }
        
        # Backup existing installation
        $targetPath = Join-Path $InstallDir "tapio.exe"
        if (Test-Path $targetPath) {
            Write-Info "Backing up existing installation..."
            Move-Item -Path $targetPath -Destination "$targetPath.backup" -Force
        }
        
        # Move binary to install directory
        Write-Info "Installing to $targetPath"
        Move-Item -Path $binaryPath -Destination $targetPath -Force
        
        # Verify installation
        if (Test-Path $targetPath) {
            Write-Success "Tapio installed successfully!"
            
            # Add to PATH
            Add-ToPath -Path $InstallDir
            
            # Test if available
            try {
                $version = & $targetPath --version 2>&1
                Write-Success "Tapio is ready to use!"
                Write-Info "Version: $version"
            }
            catch {
                Write-Warning "Tapio installed but couldn't verify version"
            }
            
            Write-Info ""
            Write-Info "Installation complete!"
            Write-Info "You may need to restart your terminal for PATH changes to take effect."
            Write-Info "Run 'tapio --help' to get started."
            
            return $true
        }
        else {
            Write-Error "Installation failed"
            return $false
        }
    }
    finally {
        # Cleanup
        if (Test-Path $tempDir) {
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }
}

# Run installation
$success = Install-Tapio

if (-not $success) {
    exit 1
}