# run_full.ps1
# 对指定目录下的每个二进制文件运行 fuzz --task full，每个二进制生成一个 txt 结果文件
#
# 参数说明:
#   -BinDir         : 测试二进制文件所在目录（必填）
#   -OutDir         : fuzz 结果输出目录（必填），每个二进制输出为 <二进制名>.txt
#   -Timeout        : 单次子进程执行超时时间（秒），默认 60。fuzz 会循环调用测试二进制数千次，
#                     该参数限制的是每一次调用的运行时长，并非整个流程的总超时
#   -RecoverTimeout : panic 后恢复等待超时（秒），默认 200。当前预留参数，运行时尚未接线生效
#
# 用法: .\scripts\run_full.ps1 -BinDir testbins\beego -OutDir zgortResult\beego -Timeout 60 -RecoverTimeout 300

param(
    [Parameter(Mandatory=$true)]
    [string]$BinDir,

    [Parameter(Mandatory=$true)]
    [string]$OutDir,

    [int]$Timeout = 60,

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
