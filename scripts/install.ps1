# sdd universal installer for Windows
# Usage: irm https://raw.githubusercontent.com/vikashkumar21/sdd/main/scripts/install.ps1 | iex

$ErrorActionPreference = 'Stop'

# Detect Architecture
$arch = if ([System.Environment]::Is64BitOperatingSystem) {
    if ($env:PROCESSOR_ARCHITECTURE -match "ARM") { "arm64" } else { "amd64" }
} else {
    throw "sdd requires a 64-bit operating system (x64 or ARM64)."
}

$repo = if ($env:SDD_REPO) { $env:SDD_REPO } else { "vikashkumar21/sdd" }
$asset = "sdd_windows_$arch.exe"
$sums = "SHA256SUMS"

$baseUrl = if ($env:SDD_BASE_URL) {
    $env:SDD_BASE_URL
} elseif ($env:SDD_VERSION) {
    "https://github.com/$repo/releases/download/$($env:SDD_VERSION)"
} else {
    "https://github.com/$repo/releases/latest/download"
}

# Create temporary directory
$tempDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.Guid]::NewGuid().ToString())
New-Item -ItemType Directory -Path $tempDir -Force | Out-Null

try {
    $binPath = Join-Path $tempDir $asset
    $downloadUrl = "$baseUrl/$asset"

    Write-Host "Downloading sdd (windows/$arch)..."
    Invoke-WebRequest -Uri $downloadUrl -OutFile $binPath -UseBasicParsing

    # Verify SHA256 checksum if available
    try {
        $sumsPath = Join-Path $tempDir $sums
        Invoke-WebRequest -Uri "$baseUrl/$sums" -OutFile $sumsPath -UseBasicParsing -ErrorAction SilentlyContinue

        if (Test-Path $sumsPath) {
            $sumsContent = Get-Content $sumsPath
            $expectedLine = $sumsContent | Where-Object { $_ -match "$asset$" }
            if ($expectedLine) {
                $expectedHash = ($expectedLine -split '\s+')[0].Trim().ToLower()
                $actualHash = (Get-FileHash -Path $binPath -Algorithm SHA256).Hash.ToLower()

                if ($expectedHash -and ($actualHash -ne $expectedHash)) {
                    throw "SHA256 checksum mismatch for $asset. Expected: $expectedHash, Actual: $actualHash"
                }
                Write-Host "✓ Verified SHA256 checksum."
            }
        }
    } catch {
        # Proceed if checksum file is absent or network fails, but throw if mismatch
        if ($_.Exception.Message -match "SHA256 checksum mismatch") {
            throw $_
        }
    }

    # Delegate installation to sdd install
    if ($env:SDD_DRY_RUN -eq "1" -or $env:SDD_DRY_RUN -eq "true") {
        & $binPath install --dry-run
    } else {
        & $binPath install
    }
} finally {
    if (Test-Path $tempDir) {
        Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}
