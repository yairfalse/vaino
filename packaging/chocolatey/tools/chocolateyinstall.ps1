$ErrorActionPreference = 'Stop'

$toolsDir   = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"
$packageName = 'wgo'
$url64      = 'https://github.com/yairfalse/wgo/releases/download/v$version$/wgo_Windows_x86_64.zip'

$packageArgs = @{
  packageName   = $packageName
  unzipLocation = $toolsDir
  url64bit      = $url64
  checksum64    = '$checksum64$'
  checksumType64= 'sha256'
}

Install-ChocolateyZipPackage @packageArgs

# Create shim for wgo.exe
$exePath = Join-Path $toolsDir "wgo.exe"
Install-BinFile -Name 'wgo' -Path $exePath

Write-Host "$packageName has been installed successfully!" -ForegroundColor Green
Write-Host ""
Write-Host "Quick Start:" -ForegroundColor Yellow
Write-Host "  wgo scan                  # Auto-discover and scan infrastructure"
Write-Host "  wgo diff                  # Compare infrastructure states"
Write-Host "  wgo scan --provider aws   # Scan AWS resources"
Write-Host "  wgo --help               # Show all commands"
Write-Host ""
Write-Host "Documentation: https://github.com/yairfalse/wgo" -ForegroundColor Cyan