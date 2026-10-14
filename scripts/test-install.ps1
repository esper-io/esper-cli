# Offline checks for the native Windows installer.
$ErrorActionPreference = 'Stop'
$root = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
$oldPath = $env:Path
$oldUserPath = [Environment]::GetEnvironmentVariable('Path', 'User')
$oldLocal = $env:LOCALAPPDATA
$oldVersion = $env:ESPERCLI_VERSION
$oldDir = $env:ESPERCLI_INSTALL_DIR
$oldArch = $env:PROCESSOR_ARCHITECTURE
$oldNativeArch = $env:PROCESSOR_ARCHITEW6432
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
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
    $env:PROCESSOR_ARCHITEW6432 = ''
    Invoke-Expression (Get-Content -Raw "$PSScriptRoot/../install.ps1")
    if (-not (Test-Path "$root/Programs/espercli/espercli.exe") -or $script:lookups -ne 1) { throw 'Latest/default install failed' }
    if ($script:asset -ne 'espercli_2.0.1_windows_x86.zip') { throw 'Incorrect release architecture label' }
    foreach ($case in @(
        @{ Process = 'ARM64'; Native = ''; Expected = 'arm64' },
        @{ Process = 'x86'; Native = 'AMD64'; Expected = 'x86' },
        @{ Process = 'AMD64'; Native = 'ARM64'; Expected = 'arm64' }
    )) {
        $env:PROCESSOR_ARCHITECTURE = $case.Process
        $env:PROCESSOR_ARCHITEW6432 = $case.Native
        Invoke-Expression (Get-Content -Raw "$PSScriptRoot/../install.ps1")
        if ($script:asset -ne "espercli_2.0.1_windows_$($case.Expected).zip") { throw 'Incorrect native OS architecture' }
    }
    $lookups = $script:lookups
    $env:ESPERCLI_VERSION = 'v2.0.2'
    $env:ESPERCLI_INSTALL_DIR = "$root/custom bin"
    . "$PSScriptRoot/../install.ps1"
    if (-not (Test-Path "$root/custom bin/espercli.exe") -or $script:lookups -ne $lookups -or $script:asset -notlike 'espercli_2.0.2_*') { throw 'Explicit overrides were ignored' }
    $env:ESPERCLI_VERSION = 'v3.0.0'
    . "$PSScriptRoot/../install.ps1"
    if ($script:asset -notlike 'espercli_3.0.0_*') { throw 'Future major release was rejected' }
    $env:PROCESSOR_ARCHITEW6432 = ''
    foreach ($unsupported in @('x86', '')) {
        $env:PROCESSOR_ARCHITECTURE = $unsupported
        try { . "$PSScriptRoot/../install.ps1"; throw 'Expected architecture rejection' }
        catch { if ($_.Exception.Message -notlike '*Unsupported architecture*') { throw } }
    }
    $env:PROCESSOR_ARCHITECTURE = 'AMD64'
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
    $env:PROCESSOR_ARCHITECTURE = $oldArch
    $env:PROCESSOR_ARCHITEW6432 = $oldNativeArch
    Remove-Item $root -Recurse -Force
}
