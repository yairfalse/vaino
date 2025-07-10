$ErrorActionPreference = 'Stop'

$packageName = 'vaino'
$toolsDir = "$(Split-Path -parent $MyInvocation.MyCommand.Definition)"

# Remove the binary
$exePath = Join-Path $toolsDir "vaino.exe"
if (Test-Path $exePath) {
    Uninstall-BinFile -Name 'vaino' -Path $exePath
    Remove-Item $exePath -Force
}

Write-Host "$packageName has been uninstalled." -ForegroundColor Green