# build_gobench.ps1
# 批量编译 testdata/gobench/nonblocking/ 下的所有 GoBench 测试二进制
# 使用项目自带的 "bins" 命令，确保正确的 .exe 处理
# 输出目录: testbins/nonblocking/<项目>/<编号>/<二进制文件>

$ErrorActionPreference = "Stop"

$ROOT_DIR = Split-Path -Parent $PSScriptRoot
Set-Location $ROOT_DIR

$SRC_BASE = "testdata\gobench\nonblocking"
$OUT_BASE = "testbins\nonblocking"

Write-Host "========================================================"
Write-Host "       GoBench Test Binary Builder                      "
Write-Host "========================================================"
Write-Host ""

# Step 1: build project tools
Write-Host "--- Step 1: Building project tools ---"
$sw = [System.Diagnostics.Stopwatch]::StartNew()
go build -o .\bin .\cmd\...
$sw.Stop()
Write-Host "OK ($([math]::Round($sw.Elapsed.TotalSeconds, 1))s)"
Write-Host ""

# Step 2: collect targets
Write-Host "--- Step 2: Collecting targets ---"
$tasks = @()
Get-ChildItem -Path $SRC_BASE -Directory | ForEach-Object {
    $proj = $_.Name
    Get-ChildItem -Path $_.FullName -Directory | ForEach-Object {
        $num = $_.Name
        $tasks += @{ Proj = $proj; Num = $num; SrcDir = $_.FullName }
    }
}
$TOTAL = $tasks.Count
Write-Host "  $TOTAL packages across $((Get-ChildItem -Path $SRC_BASE -Directory).Count) projects"
Write-Host ""

# Step 3: compile each package using bins
Write-Host "--- Step 3: Compiling ---"
$OK = 0
$FAIL = 0

$i = 0
foreach ($task in $tasks) {
    $i++
    $proj = $task.Proj
    $num  = $task.Num
    $src  = $task.SrcDir

    $outDir = "$OUT_BASE\$proj\$num"

    $captured = (.\bin\fuzz --task bins --path $src -o $outDir 2>&1)
    if ($LASTEXITCODE -eq 0) {
        $OK++
        Write-Host "  [$i/$TOTAL] + $proj/$num"
    } else {
        $FAIL++
        Write-Host "  [$i/$TOTAL] x $proj/$num  --  $captured"
    }
}

Write-Host ""

# Step 4: summary
Write-Host "--- Step 4: Summary ---"
Write-Host "  Total:   $TOTAL"
Write-Host "  Success: $OK"
Write-Host "  Failed:  $FAIL"
Write-Host ""

if ($FAIL -eq 0) {
    Write-Host "All $TOTAL packages compiled successfully!"
} else {
    Write-Host "WARNING: $FAIL package(s) failed"
    exit 1
}
