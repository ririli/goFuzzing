# run_full.ps1
# Run fuzz --task full on each binary in a directory, one txt per binary
# Usage: .\scripts\run_full.ps1 -BinDir testbins\beego -OutDir zgortResult\beego -Timeout 60 -RecoverTimeout 300

param(
    [Parameter(Mandatory=$true)]
    [string]$BinDir,

    [Parameter(Mandatory=$true)]
    [string]$OutDir,

    [int]$Timeout = 30,

    [int]$RecoverTimeout = 200
)

$ErrorActionPreference = "Stop"

$ROOT_DIR = Split-Path -Parent $PSScriptRoot
Set-Location $ROOT_DIR

if (-not (Test-Path $BinDir)) {
    Write-Host "ERROR: BinDir not found: $BinDir"
    exit 1
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null

$bins = Get-ChildItem -Path $BinDir -File
if ($bins.Count -eq 0) {
    Write-Host "No files found in $BinDir"
    exit 0
}

Write-Host "BinDir: $BinDir"
Write-Host "OutDir: $OutDir"
Write-Host "Timeout: ${Timeout}s, RecoverTimeout: ${RecoverTimeout}s"
Write-Host "Found $($bins.Count) binaries"
Write-Host ""

$OK = 0
$FAIL = 0

foreach ($bin in $bins) {
    $outFile = Join-Path $OutDir "$($bin.Name).txt"
    Write-Host "[$($OK + $FAIL + 1)/$($bins.Count)] $($bin.Name) -> $outFile"

    $captured = (& .\bin\fuzz --task full --path $bin.FullName /timeout:$Timeout /recovertimeout:$RecoverTimeout 2>&1)
    $rc = $LASTEXITCODE

    $captured | Out-File -FilePath $outFile -Encoding UTF8

    if ($rc -eq 0) {
        $OK++
    } else {
        $FAIL++
        Write-Host "  WARNING: exit code $rc"
    }
}

Write-Host ""
Write-Host "Done: $OK ok, $FAIL failed, $($bins.Count) total"
