# run_full.ps1
# 对指定目录下的每个测试二进制运行 fuzz --task full，每个二进制生成一个 txt 结果文件
#
# 参数说明:
#   -BinDir         : 测试二进制文件所在目录，默认 testbins\nonblocking（build_gobench.ps1 的输出目录），
#                     递归查找所有 .exe
#   -OutDir         : fuzz 结果输出目录（必填），每个二进制输出为 <二进制名>.txt；
#                     整轮结束后聚合报告为 OutDir\allpanic.txt 与 OutDir\alldatarace.txt，
#                     并在 alldatarace.txt 最上方写入整个项目的总耗时
#   -Granularity    : 调度颗粒度 [goroutine, function]，默认 goroutine，须与插桩时保持一致
#   -Timeout        : 单次子进程执行超时时间（秒），默认 60。fuzz 会循环调用测试二进制数千次，
#                     该参数限制的是每一次调用的运行时长，并非整个流程的总超时
#   -FuzzTime       : 单个测试函数的整个 fuzzing 会话总时长上限（秒），默认 0 = 不限时。
#                     与 -Timeout（单次执行超时）正交
#   -RecoverTimeout : panic 后恢复等待超时（秒），默认 200。当前预留参数，运行时尚未接线生效
#
# 用法: .\scripts\run_full.ps1 -BinDir  -OutDir zgortResult\nonblocking -Granularity function -Timeout 60 -RecoverTimeout 300

param(
    [string]$BinDir = "testbins\nonblocking",

    [Parameter(Mandatory=$true)]
    [string]$OutDir,

    [string]$Granularity = "goroutine",

    [int]$Timeout = 60,

    [int]$FuzzTime = 0,

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

# 聚合报告采用追加模式（fuzz 按二进制逐个调用），整轮实验开始前清理旧文件
Remove-Item (Join-Path $OutDir 'allpanic.txt') -ErrorAction SilentlyContinue
Remove-Item (Join-Path $OutDir 'alldatarace.txt') -ErrorAction SilentlyContinue

# 递归收集可执行文件（build_gobench 输出位于 testbins\nonblocking\<项目>\<编号>\ 嵌套目录）
$bins = Get-ChildItem -Path $BinDir -File -Recurse -Filter *.exe
if ($bins.Count -eq 0) {
    Write-Host "No .exe files found in $BinDir"
    exit 0
}

Write-Host "BinDir: $BinDir"
Write-Host "OutDir: $OutDir"
Write-Host "Granularity: $Granularity, Timeout: ${Timeout}s, FuzzTime: ${FuzzTime}s, RecoverTimeout: ${RecoverTimeout}s"
Write-Host "Found $($bins.Count) binaries"
Write-Host ""

# 整个项目（全部二进制）的总耗时起点
$projectStart = Get-Date

$OK = 0
$FAIL = 0

foreach ($bin in $bins) {
    $outFile = Join-Path $OutDir "$($bin.Name).txt"
    Write-Host "[$($OK + $FAIL + 1)/$($bins.Count)] $($bin.Name) -> $outFile"

    $captured = (& .\bin\fuzz --task full --path $bin.FullName /granularity:$Granularity /timeout:$Timeout /fuzztime:$FuzzTime /recovertimeout:$RecoverTimeout /outdir:$OutDir 2>&1)
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
Write-Host "Aggregated reports: $(Join-Path $OutDir 'allpanic.txt') / $(Join-Path $OutDir 'alldatarace.txt')"

# 将项目总耗时写入 alldatarace.txt 最上方（fuzz 逐二进制追加，只能整轮结束后回填头部）
$projectElapsed = (Get-Date) - $projectStart
$header = "==== Project total elapsed: {0} | BinDir: {1} | binaries: {2} (ok={3}, fail={4}) | granularity: {5} | finished: {6} ====" -f `
    $projectElapsed.ToString('d\.hh\:mm\:ss'), $BinDir, $bins.Count, $OK, $FAIL, $Granularity, (Get-Date -Format 'yyyy-MM-dd HH:mm:ss')
$raceFile = Join-Path $OutDir 'alldatarace.txt'
$existing = ""
if (Test-Path $raceFile) {
    $existing = [System.IO.File]::ReadAllText($raceFile)
}
[System.IO.File]::WriteAllText($raceFile, $header + "`r`n" + $existing)
Write-Host "Project total elapsed: $($projectElapsed.ToString('d\.hh\:mm\:ss')) (written to $raceFile)"
