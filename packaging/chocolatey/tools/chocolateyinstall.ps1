$ErrorActionPreference = 'Stop'

$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'vaino'
$url64      = 'https://github.com/yairfalse/vaino/releases/download/v$version$/vaino_Windows_x86_64.zip'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = '$checksum64$'
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Create shim for vaino.exe
$exePath = Join-Path $toolsDir "vaino.exe"
Install-BinFile -Name 'vaino' -Path $exePath

Write-Host "$packageName has been installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Quick Start:" -ForegroundColor Yellow
Write-Host "  vaino scan                  # Auto-discover and scan infrastructure"
Write-Host "  vaino diff                  # Compare infrastructure states"
Write-Host "  vaino scan --provider aws   # Scan AWS resources"
Write-Host "  vaino --help               # Show all commands"
Write-Host ""
Write-Host "Documentation: https://github.com/yairfalse/vaino" -ForegroundColor Cyan