# =============================================================
#  launch.ps1  -  Start all Mainframe services via launcher
#  Run from:   GoLang New\
#
#  Usage:
#    .\launch.ps1              - run pre-built bin\launcher.exe
#    .\launch.ps1 -Build       - build everything first, then launch
#    .\launch.ps1 -Run         - go run from source (no build step)
#    .\launch.ps1 -Config path - use a custom launcher config file
# =============================================================

param(
    [switch]$Build,
    [switch]$Run,
    [string]$Config = ""
)

$Root          = $PSScriptRoot
$LauncherExe   = Join-Path $Root "bin\launcher.exe"
$DefaultConfig = Join-Path $Root "launcher\config.yaml"

if ($Config -eq "") { $Config = $DefaultConfig }

function Write-Header {
    Write-Host ""
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host "  Mainframe Launcher" -ForegroundColor Cyan
    Write-Host "============================================================" -ForegroundColor Cyan
    Write-Host ""
}

# -- -Build: build all, then run exe --
if ($Build) {
    Write-Header
    Write-Host "  [1/2] Building all services..." -ForegroundColor Yellow
    Write-Host ""

    & "$Root\build.ps1"
    if ($LASTEXITCODE -ne 0) {
        Write-Host ""
        Write-Host "  [ERROR] Build failed. Aborting." -ForegroundColor Red
        exit 1
    }

    Write-Host ""
    Write-Host "  [2/2] Starting launcher..." -ForegroundColor Yellow
    Write-Host ""
    & $LauncherExe --config $Config
    exit $LASTEXITCODE
}

# -- -Run: go run from source --
if ($Run) {
    Write-Header
    Write-Host "  Mode   : source (go run)" -ForegroundColor Yellow
    Write-Host "  Config : $Config" -ForegroundColor DarkGray
    Write-Host "  Stop   : Ctrl+C" -ForegroundColor DarkGray
    Write-Host ""

    Push-Location (Join-Path $Root "launcher")
    try {
        & go run ./cmd --config $Config
    } finally {
        Pop-Location
    }
    exit $LASTEXITCODE
}

# -- default: run pre-built exe --
Write-Header

if (-not (Test-Path $LauncherExe)) {
    Write-Host "  [ERROR] launcher.exe not found at:" -ForegroundColor Red
    Write-Host "          $LauncherExe" -ForegroundColor Red
    Write-Host ""
    Write-Host "  Build first with one of:" -ForegroundColor Yellow
    Write-Host "    .\launch.ps1 -Build    build all services + launch" -ForegroundColor Yellow
    Write-Host "    .\launch.ps1 -Run      run from source without building" -ForegroundColor Yellow
    exit 1
}

Write-Host "  Exe    : $LauncherExe" -ForegroundColor DarkGray
Write-Host "  Config : $Config" -ForegroundColor DarkGray
Write-Host "  Stop   : Ctrl+C" -ForegroundColor DarkGray
Write-Host ""

& $LauncherExe --config $Config
exit $LASTEXITCODE
