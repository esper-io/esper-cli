# Install Esper CLI for the current Windows user. No administrator access needed.
# ESPERCLI_VERSION and ESPERCLI_INSTALL_DIR override the release and destination.
$ErrorActionPreference = 'Stop'

function Install-EsperCLI {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLowerInvariant()
    if ($arch -eq 'x64') { $arch = 'amd64' }
    if ($arch -notin @('amd64', 'arm64')) { throw "Unsupported architecture: $arch" }
    $version = $env:ESPERCLI_VERSION
    if (-not $version) {
        $release = Invoke-RestMethod -Uri 'https://api.github.com/repos/esper-io/esper-cli/releases/latest'
        $version = $release.tag_name
    }
    $version = $version -replace '^v', ''
    if ($version -notmatch '^2\.(0|[1-9][0-9]*)\.(0|[1-9][0-9]*)$') { throw 'Expected a stable version such as 2.0.1.' }
    $dir = $env:ESPERCLI_INSTALL_DIR
    if (-not $dir) { $dir = Join-Path $env:LOCALAPPDATA 'Programs\espercli' }
    $dir = [IO.Path]::GetFullPath($dir)
    $asset = "espercli_${version}_windows_${arch}.zip"
    $base = "https://github.com/esper-io/esper-cli/releases/download/v$version"
    $temp = Join-Path ([IO.Path]::GetTempPath()) ([Guid]::NewGuid().ToString())
    New-Item -ItemType Directory -Path $temp | Out-Null
    try {
        $archive = Join-Path $temp $asset
        $checksums = Join-Path $temp 'checksums.txt'
        Invoke-WebRequest -UseBasicParsing -Uri "$base/$asset" -OutFile $archive
        Invoke-WebRequest -UseBasicParsing -Uri "$base/checksums.txt" -OutFile $checksums
        $entries = @(Get-Content $checksums | Where-Object { $_ -match "^([a-fA-F0-9]{64})\s+$([regex]::Escape($asset))$" })
        if ($entries.Count -ne 1) { throw "No unique checksum found for $asset." }
        $expected = ($entries[0] -split '\s+')[0]
        if ((Get-FileHash $archive -Algorithm SHA256).Hash -ine $expected) { throw 'Checksum mismatch. Nothing was installed.' }
        Expand-Archive -LiteralPath $archive -DestinationPath $temp
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
        Copy-Item -LiteralPath (Join-Path $temp 'espercli.exe') -Destination (Join-Path $dir 'espercli.exe') -Force
        $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
        if ($dir -notin ($userPath -split ';')) {
            [Environment]::SetEnvironmentVariable('Path', "$dir;$userPath", 'User')
        }
        if ($dir -notin ($env:Path -split ';')) { $env:Path = "$dir;$env:Path" }
        Write-Output "Installed Esper CLI $version to $dir\espercli.exe"
    } finally {
        Remove-Item -LiteralPath $temp -Recurse -Force
    }
}

Install-EsperCLI
