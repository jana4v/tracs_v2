#!/usr/bin/env pwsh
# seed-mnemonics.ps1
# Seeds TM and TC mnemonic data into SQLite via the running gateway API.
# Usage: ./seed-mnemonics.ps1 [-GatewayURL http://localhost:21000] [-DataDir ./sample-data]

param(
    [string]$GatewayURL = "http://localhost:21000",
    [string]$DataDir    = ".\sample-data"
)

$tmFile = Join-Path $DataDir "tm.out"
$tcFile = Join-Path $DataDir "tc.out"

function Upload-File {
    param(
        [string]$FilePath,
        [string]$EndpointURL,
        [string]$Label
    )
    if (-not (Test-Path $FilePath)) {
        Write-Error "File not found: $FilePath"
        return $false
    }
    $bytes   = [System.IO.File]::ReadAllBytes($FilePath)
    $b64     = [System.Convert]::ToBase64String($bytes)
    $filename = Split-Path -Leaf $FilePath
    $body    = @{ filename = $filename; data = $b64 } | ConvertTo-Json -Compress

    Write-Host "Uploading $Label ($filename)..." -ForegroundColor Cyan
    try {
        $resp = Invoke-RestMethod -Uri $EndpointURL -Method POST `
            -Body $body -ContentType "application/json" -ErrorAction Stop
        Write-Host "[OK] $Label uploaded." -ForegroundColor Green
        Write-Host "     Total=$($resp.stats.total)  Inserted=$($resp.stats.inserted)  Updated=$($resp.stats.updated)  Skipped=$($resp.stats.skipped)"
        if ($resp.stats.errors -and $resp.stats.errors.Count -gt 0) {
            Write-Warning "     Errors:"
            $resp.stats.errors | ForEach-Object { Write-Warning "       - $_" }
        }
        return $true
    } catch {
        Write-Error "[FAIL] $Label upload failed: $_"
        return $false
    }
}

Write-Host ""
Write-Host "=== ASTRA SQLite Mnemonic Seeder ===" -ForegroundColor Yellow
Write-Host "Gateway: $GatewayURL"
Write-Host ""

# Upload TM mnemonics
$tmOk = Upload-File -FilePath $tmFile `
    -EndpointURL "$GatewayURL/api/go/v1/telemetry/upload" `
    -Label "TM Mnemonics"

Write-Host ""

# Upload TC mnemonics
$tcOk = Upload-File -FilePath $tcFile `
    -EndpointURL "$GatewayURL/api/go/v1/telecommand/upload" `
    -Label "TC Mnemonics"

Write-Host ""
Write-Host "=== Done ===" -ForegroundColor Yellow

# Verify row counts via sqlite3 if available on PATH
$sqlite = Get-Command sqlite3 -ErrorAction SilentlyContinue
if ($sqlite) {
    $dbPath = Join-Path $PSScriptRoot "..\launcher\astra.db"
    if (Test-Path $dbPath) {
        Write-Host ""
        Write-Host "Row counts in SQLite ($dbPath):" -ForegroundColor Cyan
        & sqlite3 $dbPath "SELECT 'tm_mnemonics', COUNT(*) FROM tm_mnemonics UNION ALL SELECT 'tc_mnemonics', COUNT(*) FROM tc_mnemonics;"
    }
}
