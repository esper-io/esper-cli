# Offline checks for the native Windows installer.
$ErrorActionPreference = 'Stop'
$root = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
$oldPath = $env:Path
$oldUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$oldLocal = $env:LOCALAPPDATA
$oldVersion = $env:ESPERCLI_VERSION
$oldDir = $env:ESPERCLI_INSTALL_DIR
function Invoke-RestMethod { param($Uri) $script:lookups++; return @{ tag_name = 'v2.0.1' } }
function Invoke-WebRequest {
    param($Uri, $OutFile, [switch]$UseBasicParsing)
    if ($Uri.EndsWith('/checksums.txt')) {
        $name = $script:asset
        "$script:hash  $name" | Set-Content -Encoding ASCII $OutFile
    } else {
        $script:asset = $Uri.Split('/')[-1]
        Copy-Item (Join-Path $root 'fixture.zip') $OutFile
    }
}
try {
    New-Item -ItemType Directory -Path $root | Out-Null
    'fixture binary' | Set-Content (Join-Path $root 'espercli.exe')
    Compress-Archive (Join-Path $root 'espercli.exe') (Join-Path $root 'fixture.zip')
    $script:hash = (Get-FileHash (Join-Path $root 'fixture.zip')).Hash
    $script:lookups = 0
    $env:LOCALAPPDATA = $root
    $env:ESPERCLI_VERSION = ''
    $env:ESPERCLI_INSTALL_DIR = ''
    . "$PSScriptRoot/../install.ps1"
    if (-not (Test-Path "$root/Programs/espercli/espercli.exe") -or $script:lookups -ne 1) { throw 'Latest/default install failed' }
    $env:ESPERCLI_VERSION = 'v2.0.2'
    $env:ESPERCLI_INSTALL_DIR = "$root/custom bin"
    . "$PSScriptRoot/../install.ps1"
    if (-not (Test-Path "$root/custom bin/espercli.exe") -or $script:lookups -ne 1 -or $script:asset -notlike 'espercli_2.0.2_*') { throw 'Explicit overrides were ignored' }
    $script:hash = '0' * 64
    $env:ESPERCLI_INSTALL_DIR = "$root/rejected"
    try { . "$PSScriptRoot/../install.ps1"; throw 'Expected checksum rejection' }
    catch { if ($_.Exception.Message -notlike '*Checksum mismatch*') { throw } }
    if (Test-Path "$root/rejected/espercli.exe") { throw 'Installed an unverified archive' }
    Write-Output 'Installer checks passed.'
} finally {
    [Environment]::SetEnvironmentVariable('Path', $oldUserPath, 'User')
    $env:Path = $oldPath
    $env:LOCALAPPDATA = $oldLocal
    $env:ESPERCLI_VERSION = $oldVersion
    $env:ESPERCLI_INSTALL_DIR = $oldDir
    Remove-Item $root -Recurse -Force
}
