$ErrorActionPreference = 'Stop'

$packageName = 'wgo'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove the binary
$exePath = Join-Path $toolsDir "wgo.exe"
if (Test-Path $exePath) {
    Uninstall-BinFile -Name 'wgo' -Path $exePath
    Remove-Item $exePath -Force
}

Write-Host "$packageName has been uninstalled." -ForegroundColor Green