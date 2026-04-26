# =============================================================
#  build.ps1  -  Build all Mainframe Go services
#  Run from:  GoLang New\
#  Output:    GoLang New\bin\*.exe
# =============================================================

param(
    [string[]]$Only = @()   # e.g.  -Only iam,gateway  to build a subset
)

$Root    = $PSScriptRoot
$BinDir  = Join-Path $Root "bin"
$Failed  = @()

$Services = @(
    @{ Name = "iam";               Dir = "iam"               },
    @{ Name = "gateway";           Dir = "gateway"           },
    @{ Name = "ingest";            Dir = "ingest"            },
    @{ Name = "storage";           Dir = "storage"           },
    @{ Name = "limiter";           Dir = "limiter"           },
    @{ Name = "simulator";         Dir = "simulator"         },
    @{ Name = "comparator";        Dir = "comparator"        },
    @{ Name = "chainmon";          Dir = "chainmon"          },
    @{ Name = "umacs-tc";          Dir = "umacs-tc"          },
    @{ Name = "umacs-tc-emulator"; Dir = "umacs-tc-emulator" },
    @{ Name = "launcher";          Dir = "launcher"          }
)

if ($Only.Count -gt 0) {
    $Services = $Services | Where-Object { $Only -contains $_.Name }
}

Write-Host ""
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host "  Mainframe - Build All Services" -ForegroundColor Cyan
Write-Host "  Output: $BinDir" -ForegroundColor Cyan
Write-Host "============================================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $BinDir)) {
    New-Item -ItemType Directory -Path $BinDir | Out-Null
}

# Run from workspace root so go.work is picked up
Set-Location $Root

foreach ($svc in $Services) {
    $cmdDir = Join-Path $Root "$($svc.Dir)\cmd"
    $outExe = Join-Path $BinDir "$($svc.Name).exe"
    $pkg    = "./$($svc.Dir)/cmd"

    if (-not (Test-Path $cmdDir)) {
        Write-Host ("  [SKIP]  {0,-25} (no cmd\ directory)" -f $svc.Name) -ForegroundColor DarkGray
        continue
    }

    Write-Host ("  [BUILD] {0,-25} ..." -f $svc.Name) -NoNewline -ForegroundColor Yellow

    $output = & go build -o $outExe $pkg 2>&1
    if ($LASTEXITCODE -eq 0) {
        Write-Host ("`r  [OK]    {0,-25} -> bin\$($svc.Name).exe" -f $svc.Name) -ForegroundColor Green
    } else {
        Write-Host ("`r  [FAIL]  {0,-25}" -f $svc.Name) -ForegroundColor Red
        Write-Host $output -ForegroundColor Red
        $Failed += $svc.Name
    }
}

Write-Host ""
if ($Failed.Count -eq 0) {
    Write-Host "  All services built successfully." -ForegroundColor Green
} else {
    $failedList = $Failed -join ", "
    Write-Host "  Failed: $failedList" -ForegroundColor Red
    exit 1
}
