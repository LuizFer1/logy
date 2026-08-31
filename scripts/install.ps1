# Install Logy from GitHub Releases into %LOCALAPPDATA%\Logy (or LOGY_INSTALL_DIR).
$ErrorActionPreference = 'Stop'

$Repo = if ($env:LOGY_GITHUB_REPO) { $env:LOGY_GITHUB_REPO.Trim() } else { 'LuizFer1/logy' }
$InstallDir = if ($env:LOGY_INSTALL_DIR) { $env:LOGY_INSTALL_DIR.Trim() } else { Join-Path $env:LOCALAPPDATA 'Logy' }

function Get-LogyArch {
    $arch = $env:PROCESSOR_ARCHITECTURE
    switch -Regex ($arch) {
        '^(AMD64|x86_64)$' { return 'amd64' }
        '^(ARM64|aarch64)$' { return 'arm64' }
        default {
            # PROCESSOR_ARCHITECTURE can be x86 under WOW64; prefer native.
            if ($env:PROCESSOR_ARCHITEW6432 -match '^(AMD64|x86_64)$') { return 'amd64' }
            if ($env:PROCESSOR_ARCHITEW6432 -match '^(ARM64|aarch64)$') { return 'arm64' }
            throw "Unsupported architecture: $arch (supported: amd64, arm64)"
        }
    }
}

function Normalize-Tag([string]$t) {
    $t = $t.Trim()
    if ($t -match '^[vV]') { return $t }
    return "v$t"
}

function Strip-V([string]$t) {
    return ($t -replace '^[vV]', '')
}

function Invoke-GitHubGet([string]$Url) {
    $headers = @{ Accept = 'application/vnd.github+json' }
    $token = $env:GH_TOKEN
    if (-not $token) { $token = $env:GITHUB_TOKEN }
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    return Invoke-RestMethod -Uri $Url -Headers $headers -Method Get
}

function Download-File([string]$Url, [string]$OutPath) {
    $headers = @{}
    $token = $env:GH_TOKEN
    if (-not $token) { $token = $env:GITHUB_TOKEN }
    if ($token) { $headers['Authorization'] = "Bearer $token" }
    Invoke-WebRequest -Uri $Url -OutFile $OutPath -Headers $headers -UseBasicParsing | Out-Null
}

function Ensure-UserPath([string]$Dir) {
    $userPath = [Environment]::GetEnvironmentVariable('Path', 'User')
    if (-not $userPath) { $userPath = '' }
    $parts = $userPath -split ';' | Where-Object { $_ -ne '' }
    $normalized = $Dir.TrimEnd('\')
    foreach ($p in $parts) {
        if ($p.TrimEnd('\') -ieq $normalized) {
            return $false
        }
    }
    $newPath = if ($userPath.Trim() -eq '') { $Dir } else { "$userPath;$Dir" }
    [Environment]::SetEnvironmentVariable('Path', $newPath, 'User')
    # Update current session PATH too.
    if (-not (($env:Path -split ';') | Where-Object { $_.TrimEnd('\') -ieq $normalized })) {
        $env:Path = "$Dir;$env:Path"
    }
    return $true
}

$Arch = Get-LogyArch
$Os = 'windows'

if ($env:LOGY_VERSION) {
    $Tag = Normalize-Tag $env:LOGY_VERSION
} else {
    Write-Host "Resolving latest release for $Repo..."
    try {
        $latest = Invoke-GitHubGet "https://api.github.com/repos/$Repo/releases/latest"
    } catch {
        throw "Failed to fetch latest release (is the repo public? set GH_TOKEN if rate-limited): $_"
    }
    if (-not $latest.tag_name) {
        throw 'Could not parse tag_name from GitHub API response'
    }
    $Tag = [string]$latest.tag_name
}

$Version = Strip-V $Tag
$Asset = "logy_${Version}_${Os}_${Arch}.zip"
$BaseUrl = "https://github.com/$Repo/releases/download/$Tag"
$AssetUrl = "$BaseUrl/$Asset"
$ChecksumsUrl = "$BaseUrl/checksums.txt"

$Tmp = Join-Path ([System.IO.Path]::GetTempPath()) ("logy-install-" + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $Tmp | Out-Null
try {
    $AssetPath = Join-Path $Tmp $Asset
    $ChecksumsPath = Join-Path $Tmp 'checksums.txt'

    Write-Host "Downloading $Asset..."
    try {
        Download-File $AssetUrl $AssetPath
    } catch {
        throw "Failed to download $AssetUrl : $_"
    }

    Write-Host 'Downloading checksums.txt...'
    try {
        Download-File $ChecksumsUrl $ChecksumsPath
    } catch {
        throw "Failed to download $ChecksumsUrl : $_"
    }

    $wantHash = $null
    foreach ($line in Get-Content -Path $ChecksumsPath) {
        $trim = $line.Trim()
        if ($trim -eq '') { continue }
        $fields = $trim -split '\s+'
        if ($fields.Count -lt 2) { continue }
        $name = $fields[-1] -replace '^\*', ''
        if ($name -eq $Asset) {
            $wantHash = $fields[0]
            break
        }
    }
    if (-not $wantHash) {
        throw "Checksum not found for $Asset in checksums.txt"
    }

    $gotHash = (Get-FileHash -Path $AssetPath -Algorithm SHA256).Hash
    if ($gotHash.ToLowerInvariant() -ne $wantHash.ToLowerInvariant()) {
        throw "sha256 mismatch: got $gotHash want $wantHash"
    }
    Write-Host 'Checksum OK'

    $ExtractDir = Join-Path $Tmp 'extract'
    New-Item -ItemType Directory -Path $ExtractDir | Out-Null
    Expand-Archive -Path $AssetPath -DestinationPath $ExtractDir -Force

    $Binary = Get-ChildItem -Path $ExtractDir -Recurse -File -Filter 'logy.exe' | Select-Object -First 1
    if (-not $Binary) {
        throw 'logy.exe not found in archive'
    }

    New-Item -ItemType Directory -Force -Path $InstallDir | Out-Null
    $Dest = Join-Path $InstallDir 'logy.exe'
    Copy-Item -Path $Binary.FullName -Destination $Dest -Force

    $pathAdded = Ensure-UserPath $InstallDir
    if ($pathAdded) {
        Write-Host ""
        Write-Host "Added $InstallDir to your user PATH."
        Write-Host 'Open a new terminal (or refresh PATH) if logy is not found yet.'
    }

    Write-Host ""
    Write-Host "Installed logy $Version to $Dest"
    & $Dest version
} finally {
    Remove-Item -Recurse -Force -Path $Tmp -ErrorAction SilentlyContinue
}
