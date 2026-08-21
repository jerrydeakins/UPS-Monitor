param(
    [string]$Version = ""
)

$ErrorActionPreference = "Stop"

$ProjectRoot = Split-Path -Parent $MyInvocation.MyCommand.Path
Set-Location $ProjectRoot

$VersionFile = Join-Path $ProjectRoot "internal\version\version.go"

if (-not (Test-Path $VersionFile)) {
    throw "Version file not found: $VersionFile"
}

$versionSource = Get-Content $VersionFile -Raw
if ($versionSource -notmatch 'AppVersion\s*=\s*"([^"]+)"') {
    throw "Could not read AppVersion from $VersionFile"
}

$sourceVersion = $Matches[1]

if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = $sourceVersion
}

if ($Version -ne $sourceVersion) {
    throw "Version mismatch: version.go contains $sourceVersion, requested build version is $Version"
}

Write-Host "Building UPS Monitor $Version..."

$exePath = Join-Path $ProjectRoot "UPS-Monitor.exe"

go build `
    -tags migrated_fynedo `
    -ldflags="-H=windowsgui" `
    -o $exePath `
    .\cmd\app

if ($LASTEXITCODE -ne 0) {
    throw "Go build failed."
}

Write-Host "Built: $exePath"

$issPath = Join-Path $ProjectRoot "installer\ups-monitor.iss"

if (-not (Test-Path $issPath)) {
    throw "Inno Setup script not found: $issPath"
}

$iscc = $null

$command = Get-Command "ISCC.exe" -ErrorAction SilentlyContinue
if ($command) {
    $iscc = $command.Source
}

if (-not $iscc) {
    $candidates = @(
        "${env:ProgramFiles(x86)}\Inno Setup 7\ISCC.exe",
        "${env:ProgramFiles}\Inno Setup 7\ISCC.exe"
    )

    foreach ($candidate in $candidates) {
        if ($candidate -and (Test-Path $candidate)) {
            $iscc = $candidate
            break
        }
    }
}

if (-not $iscc) {
    throw "ISCC.exe not found. Install Inno Setup 7 or add ISCC.exe to PATH."
}

Write-Host "Building installer with: $iscc"

& $iscc "/DMyAppVersion=$Version" $issPath

if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup build failed."
}

$installerPath = Join-Path $ProjectRoot "installer\UPS-Monitor-$Version-Setup.exe"

if (-not (Test-Path $installerPath)) {
    throw "Installer was not created: $installerPath"
}

Write-Host ""
Write-Host "Release build complete:"
Write-Host "  EXE:       $exePath"
Write-Host "  Installer: $installerPath"