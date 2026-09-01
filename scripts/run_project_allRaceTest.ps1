<#
.SYNOPSIS
    对 Go 项目中所有测试包运行 go test -race，收集 race detector 警告和 panic 信息。
.DESCRIPTION
    本脚本执行以下步骤:
      1. 通过 'go list ./...' 发现所有测试包
      2. 按顺序对每个包运行 'go test -race -json'
      3. 将每个包的 JSON 输出保存到带时间戳的结果目录
      4. 生成汇总报告，列出所有包含 DATA RACE 警告和 panic 的包
.PARAMETER ProjectPath
    目标 Go 项目根目录的绝对路径或相对路径。
.PARAMETER OutputDir
    存放结果的目录。默认值: 当前目录下的 ./race_results/<项目>_<时间戳>。
.PARAMETER Count
    每个测试函数执行的次数（-count 参数）。默认值: 1。
.PARAMETER TimeoutMinutes
    每个包的测试超时时间（分钟）。默认值: 5。
.EXAMPLE
    # 对 beego 运行 3 次迭代
    .\scripts\run_project_allRaceTest.ps1 -ProjectPath D:\gopath\src\beego -Count 3

.EXAMPLE
    # 对 gin 运行，自定义超时和输出目录
    .\scripts\run_project_allRaceTest.ps1 -ProjectPath D:\gopath\src\gin -TimeoutMinutes 10 -OutputDir D:\results\gin_race -Count 10
#>

param(
    [Parameter(Mandatory = $true, HelpMessage = "Path to the target Go project root")]
    [string]$ProjectPath,

    [string]$OutputDir = "",

    [int]$Count = 1,

    [int]$TimeoutMinutes = 5
)

$ErrorActionPreference = "Stop"

# ============================================================
# 0. Validate inputs
# ============================================================
try {
    $ProjectPath = Resolve-Path $ProjectPath -ErrorAction Stop
} catch {
    Write-Host "[ERROR] Path not found: $ProjectPath" -ForegroundColor Red
    Write-Host "   (resolved from current directory: $(Get-Location))" -ForegroundColor DarkGray
    Write-Host "   Tip: Use absolute path or '../' to go up from current directory." -ForegroundColor DarkGray
    exit 1
}
$ProjectName = Split-Path $ProjectPath -Leaf

if ($OutputDir -eq "") {
    $timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
    $OutputDir = Join-Path (Join-Path (Get-Location) "race_results") "$($ProjectName)_$timestamp"
}
New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Go Race Detector Batch Tester" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Project  : $ProjectPath" -ForegroundColor Cyan
Write-Host "  Output   : $OutputDir" -ForegroundColor Cyan
Write-Host "  Count    : $Count" -ForegroundColor Cyan
Write-Host "  Timeout  : ${TimeoutMinutes} min/package" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Start global wall-clock timer
$globalSW = [System.Diagnostics.Stopwatch]::StartNew()

# ============================================================
# 1. Discover test packages
# ============================================================
Write-Host "[1/4] Discovering test packages..." -ForegroundColor Yellow

Push-Location $ProjectPath
try {
    $allPkgsRaw = go list ./... 2>&1
} finally {
    Pop-Location
}

if ($LASTEXITCODE -ne 0 -and $allPkgsRaw -is [string] -and $allPkgsRaw -match "^(go: )") {
    Write-Error "[ERROR] go list failed: $allPkgsRaw"
    exit 1
}

$testPkgs = @()
foreach ($line in $allPkgsRaw) {
    $pkg = $line.Trim()
    if ($pkg -eq "") { continue }
    # Skip vendor packages (external dependencies)
    if ($pkg -match "(^|/)vendor/") { continue }

    Push-Location $ProjectPath
    try {
        $testFiles = go list -f "{{.TestGoFiles}}" $pkg 2>&1
    } finally {
        Pop-Location
    }

    # TestGoFiles is empty "[]" when no _test.go files exist
    if ($testFiles -is [array]) { $testFiles = $testFiles -join "" }
    if ($testFiles -and $testFiles.Trim() -notmatch "^\[\]$") {
        $testPkgs += $pkg
    }
}

$totalPkgs = $testPkgs.Count
if ($totalPkgs -eq 0) {
    Write-Warning "[WARN] No test packages found in $ProjectPath"
    exit 0
}
Write-Host "  Found $totalPkgs test package(s)." -ForegroundColor Green
Write-Host ""

# ============================================================
# 2. Run go test -race on each package
# ============================================================
Write-Host "[2/4] Running go test -race on each package..." -ForegroundColor Yellow
Write-Host ""

$results = [System.Collections.ArrayList]::new()
$raceWarningsAll = [System.Collections.ArrayList]::new()
$panicWarningsAll = [System.Collections.ArrayList]::new()
$currentIdx = 0

$timeoutFlag = "${TimeoutMinutes}m"

foreach ($pkg in $testPkgs) {
    $currentIdx++

    Write-Host "  [$currentIdx/$totalPkgs] $pkg " -NoNewline

    Push-Location $ProjectPath
    try {
        $sw = [System.Diagnostics.Stopwatch]::StartNew()
        $testOutput = go test -race -json -count $($Count) -timeout $timeoutFlag $pkg 2>&1
        $sw.Stop()
    } catch {
        $sw.Stop()
        $testOutput = @("ERROR: $_")
    } finally {
        Pop-Location
    }

    # Detect go test command failure (e.g. bad flags, binary not compiled)
    $cmdFailed = ($testOutput.Count -eq 1 -and $testOutput[0] -match "^ERROR:")

    # Parse result
    if ($cmdFailed) {
        $pkgResult = "CMDFAIL"
    } else {
        $pkgResult = "PASS"
    }
    $raceCount = 0
    $raceLines = @()
    $inRaceBlock = $false
    $panicCount = 0
    $panicLines = @()
    $inPanicBlock = $false

    foreach ($line in $testOutput) {
        # Try to parse as JSON
        try {
            $obj = $line | ConvertFrom-Json
        } catch {
            # Non-JSON line - raw stderr from race detector (tsan) or runtime panic.
            # These write directly to fd 2 via C syscalls and bypass go test -json formatting.

            # --- Race detection on raw stderr lines ---
            if ($line -match "WARNING: DATA RACE" -and -not $inRaceBlock) {
                $inRaceBlock = $true
                $raceCount++
                $raceLines += ($line + "`n")
            }
            elseif ($inRaceBlock) {
                $raceLines += ($line + "`n")
                if ($line -match "==================" -and $raceLines.Count -gt 1) {
                    $raceBlock = ($raceLines | ForEach-Object {
                        try { ($_ | ConvertFrom-Json).Output } catch { $_ }
                    }) -join ""
                    [void]$raceWarningsAll.Add(@{
                        Package = $pkg
                        Block   = $raceBlock
                    })
                    $raceLines = @()
                    $inRaceBlock = $false
                }
            }

            # --- Panic detection on raw stderr lines ---
            if ($line -match "^\s*panic:" -and -not $inRaceBlock -and -not $inPanicBlock) {
                $inPanicBlock = $true
                $panicCount++
                $panicLines += ($line + "`n")
            }
            elseif ($inPanicBlock) {
                $panicLines += ($line + "`n")
                # Raw panic block ends at a test-result / package-result boundary
                if ($line -match "^(FAIL|PASS|ok\s|---)") {
                    $panicBlock = ($panicLines | ForEach-Object {
                        try { ($_ | ConvertFrom-Json).Output } catch { $_ }
                    }) -join ""
                    [void]$panicWarningsAll.Add(@{
                        Package = $pkg
                        Block   = $panicBlock
                    })
                    $panicLines = @()
                    $inPanicBlock = $false
                }
            }

            continue
        }

        # Detect package-level result
        if ($obj.Action -eq "fail" -and (-not $obj.Test)) {
            $pkgResult = "FAIL"
        }

        # Detect panic in output (before race check - a panic line should not start a race block)
        if ($obj.Output -and $obj.Output -match "^\s*panic:" -and -not $inRaceBlock -and -not $inPanicBlock) {
            $inPanicBlock = $true
            $panicCount++
            $panicLines += $line
        }
        elseif ($inPanicBlock -and $obj.Output) {
            $panicLines += $line
        }
        elseif ($inPanicBlock -and $obj.Action -ne "output") {
            # End of panic block - save it
            $panicBlock = ($panicLines | ForEach-Object {
                try { ($_ | ConvertFrom-Json).Output } catch { $_ }
            }) -join ""
            [void]$panicWarningsAll.Add(@{
                Package = $pkg
                Block   = $panicBlock
            })
            $panicLines = @()
            $inPanicBlock = $false
        }

        # Detect race warning in output
        if ($obj.Output -and $obj.Output -match "WARNING: DATA RACE") {
            $inRaceBlock = $true
            $raceCount++
            $raceLines += $line
        }

        # Collect race block lines (stack traces follow the warning)
        elseif ($inRaceBlock -and $obj.Output) {
            $raceLines += $line
            if ($obj.Output -match "==================" -and $raceLines.Count -gt 1) {
                # End of a race block - save it
                $raceBlock = ($raceLines | ForEach-Object {
                    try { ($_ | ConvertFrom-Json).Output } catch { $_ }
                }) -join ""
                [void]$raceWarningsAll.Add(@{
                    Package = $pkg
                    Block   = $raceBlock
                })
                $raceLines = @()
                $inRaceBlock = $false
            }
        }
    }

    # If still in a race block at end of output, save it
    if ($inRaceBlock -and $raceLines.Count -gt 0) {
        $raceBlock = ($raceLines | ForEach-Object {
            try { ($_ | ConvertFrom-Json).Output } catch { $_ }
        }) -join ""
        [void]$raceWarningsAll.Add(@{
            Package = $pkg
            Block   = $raceBlock
        })
    }

    # If still in a panic block at end of output, save it
    if ($inPanicBlock -and $panicLines.Count -gt 0) {
        $panicBlock = ($panicLines | ForEach-Object {
            try { ($_ | ConvertFrom-Json).Output } catch { $_ }
        }) -join ""
        [void]$panicWarningsAll.Add(@{
            Package = $pkg
            Block   = $panicBlock
        })
    }

    $elapsed = $sw.Elapsed.ToString("mm\:ss")
    $elapsedSec = [math]::Round($sw.Elapsed.TotalSeconds, 2)
    $statusIcon = if ($pkgResult -eq "CMDFAIL") { "[CMDFAIL]" } elseif ($raceCount -gt 0) { "[RACE]" } elseif ($panicCount -gt 0) { "[PANIC]" } elseif ($pkgResult -eq "FAIL") { "[FAIL]" } else { "OK" }
    $statusColor = if ($pkgResult -eq "CMDFAIL") { "Magenta" } elseif ($raceCount -gt 0) { "Red" } elseif ($panicCount -gt 0) { "Red" } elseif ($pkgResult -eq "FAIL") { "Yellow" } else { "Green" }
    Write-Host " ${elapsed}  $statusIcon (races=$raceCount, panics=$panicCount)" -ForegroundColor $statusColor

    [void]$results.Add([PSCustomObject]@{
        Package        = $pkg
        Result         = $pkgResult
        RaceCount      = $raceCount
        PanicCount     = $panicCount
        Elapsed        = $elapsed
        ElapsedSeconds = $elapsedSec
    })
}

Write-Host ""

# ============================================================
# 3. Save race warnings
# ============================================================
Write-Host "[3/4] Saving race warnings..." -ForegroundColor Yellow

$raceFile = Join-Path $OutputDir "all_race_warnings.txt"
if ($raceWarningsAll.Count -gt 0) {
    $raceContent = @"
============================================
  Race Detector Warnings Summary
  Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
  Total race blocks: $($raceWarningsAll.Count)
============================================

"@
    foreach ($rw in $raceWarningsAll) {
        $raceContent += @"

----------------------------------------
  Package: $($rw.Package)
----------------------------------------
$($rw.Block)
"@
    }
    $raceContent | Out-File -FilePath $raceFile -Encoding UTF8
    Write-Host "  Saved $($raceWarningsAll.Count) race warning(s) -> $raceFile" -ForegroundColor Green
} else {
    "(No race warnings detected)" | Out-File -FilePath $raceFile -Encoding UTF8
    Write-Host "  No race warnings detected." -ForegroundColor Green
}

# ============================================================
# 3.5. Save panic warnings
# ============================================================
Write-Host "[3.5/4] Saving panic warnings..." -ForegroundColor Yellow

$panicFile = Join-Path $OutputDir "all_panics.txt"
if ($panicWarningsAll.Count -gt 0) {
    $panicContent = @"
============================================
  Panic Warnings Summary
  Generated: $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
  Total panic blocks: $($panicWarningsAll.Count)
============================================

"@
    foreach ($pw in $panicWarningsAll) {
        $panicContent += @"

----------------------------------------
  Package: $($pw.Package)
----------------------------------------
$($pw.Block)
"@
    }
    $panicContent | Out-File -FilePath $panicFile -Encoding UTF8
    Write-Host "  Saved $($panicWarningsAll.Count) panic warning(s) -> $panicFile" -ForegroundColor Green
} else {
    "(No panics detected)" | Out-File -FilePath $panicFile -Encoding UTF8
    Write-Host "  No panics detected." -ForegroundColor Green
}

# ============================================================
# 4. Generate summary
# ============================================================
$globalSW.Stop()
$wallTimeSec = [math]::Round($globalSW.Elapsed.TotalSeconds, 2)
$testTimeSec = [math]::Round(($results | Measure-Object -Property ElapsedSeconds -Sum).Sum, 2)

Write-Host "[4/4] Generating summary..." -ForegroundColor Yellow

$passCount = @($results | Where-Object { $_.Result -eq "PASS" -and $_.RaceCount -eq 0 -and $_.PanicCount -eq 0 }).Count
$failCount = @($results | Where-Object { $_.Result -eq "FAIL" }).Count
$cmdFailCount = @($results | Where-Object { $_.Result -eq "CMDFAIL" }).Count
$racePkgCount = @($results | Where-Object { $_.RaceCount -gt 0 }).Count
$panicPkgCount = @($results | Where-Object { $_.PanicCount -gt 0 }).Count
$totalRaces = ($results | Measure-Object -Property RaceCount -Sum).Sum
$totalPanics = ($results | Measure-Object -Property PanicCount -Sum).Sum

$summaryFile = Join-Path $OutputDir "summary.txt"
$summaryContent = @"
============================================
  Go Race Detector Batch Test - Summary
============================================
  Project        : $ProjectPath
  Date           : $(Get-Date -Format "yyyy-MM-dd HH:mm:ss")
  Count per test : $Count
  Timeout/pkg    : ${TimeoutMinutes} min
============================================

  Total packages  : $totalPkgs
  Pass (clean)    : $passCount
  Failed          : $failCount
  Cmd Failed      : $cmdFailCount
  Race detected   : $racePkgCount
  Panic detected  : $panicPkgCount
  Total race blocks: $totalRaces
  Total panic blocks: $totalPanics
  ----------------------------------------
  Wall time       : $($wallTimeSec.ToString('0.00')) s
  go test (sum)   : $($testTimeSec.ToString('0.00')) s

============================================
  Per-Package Results
============================================
$(
($results | Format-Table -Property @(
    @{Label="Package"; Expression={$_.Package}; Width=60},
    @{Label="Result"; Expression={$_.Result}; Width=8},
    @{Label="Races"; Expression={$_.RaceCount}; Width=8},
    @{Label="Panics"; Expression={$_.PanicCount}; Width=8},
    @{Label="Time"; Expression={$_.Elapsed}; Width=8}
) -AutoSize | Out-String)
)
============================================

Packages with Race Warnings:
$(
if ($racePkgCount -gt 0) {
    ($results | Where-Object { $_.RaceCount -gt 0 } | ForEach-Object {
        "  - $($_.Package)  ($($_.RaceCount) race(s))"
    }) -join "`n"
} else {
    "  (none)"
}
)

Packages with Panics:
$(
if ($panicPkgCount -gt 0) {
    ($results | Where-Object { $_.PanicCount -gt 0 } | ForEach-Object {
        "  - $($_.Package)  ($($_.PanicCount) panic(s))"
    }) -join "`n"
} else {
    "  (none)"
}
)
"@

$summaryContent | Out-File -FilePath $summaryFile -Encoding UTF8

# Print summary to console
Write-Host ""
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  SUMMARY" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host "  Total packages  : $totalPkgs" -ForegroundColor White
Write-Host "  Pass (clean)    : $passCount" -ForegroundColor Green
Write-Host "  Failed          : $failCount" -ForegroundColor Yellow
Write-Host "  Cmd Failed      : $cmdFailCount" -ForegroundColor Magenta
Write-Host "  Race detected   : $racePkgCount" -ForegroundColor Red
Write-Host "  Panic detected  : $panicPkgCount" -ForegroundColor Red
Write-Host "  Total race blocks: $totalRaces" -ForegroundColor Red
Write-Host "  Total panic blocks: $totalPanics" -ForegroundColor Red
Write-Host "  ----------------------------------------" -ForegroundColor DarkGray
Write-Host "  Wall time       : $($wallTimeSec.ToString('0.00')) s" -ForegroundColor White
Write-Host "  Test time (sum) : $($testTimeSec.ToString('0.00')) s" -ForegroundColor White
Write-Host ""
Write-Host "  Full report: $summaryFile" -ForegroundColor Cyan
Write-Host "  Race warnings: $raceFile" -ForegroundColor Cyan
Write-Host "  Panic warnings: $panicFile" -ForegroundColor Cyan
Write-Host "============================================" -ForegroundColor Cyan
Write-Host ""

# Exit code: non-zero if races or panics found (for CI integration)
if ($racePkgCount -gt 0 -or $panicPkgCount -gt 0) {
    exit 1
}
exit 0
